package tracker

import (
	"fmt"
	"sort"

	"github.com/JSLEEKR/scrapling-go/pkg/parser"
	"github.com/JSLEEKR/scrapling-go/pkg/selector"
	"github.com/JSLEEKR/scrapling-go/pkg/storage"
)

// DefaultThreshold is the minimum similarity score for relocation to succeed.
const DefaultThreshold = 50.0

// Tracker manages element fingerprinting and adaptive relocation.
type Tracker struct {
	store     *storage.Store
	threshold float64
}

// New creates a new Tracker with the given storage backend.
func New(store *storage.Store) *Tracker {
	return &Tracker{
		store:     store,
		threshold: DefaultThreshold,
	}
}

// SetThreshold sets the minimum similarity score for relocation.
func (tr *Tracker) SetThreshold(threshold float64) {
	tr.threshold = threshold
}

// Track saves an element's fingerprint for later relocation.
func (tr *Tracker) Track(pageURL, selectorStr string, elem *parser.Adaptable) error {
	dict := ElementToDict(elem)
	if dict == nil {
		return fmt.Errorf("cannot fingerprint nil element")
	}
	return tr.store.Save(pageURL, selectorStr, dict)
}

// Find attempts to find an element using CSS selector first, then falls back
// to fingerprint-based relocation if the selector fails.
func (tr *Tracker) Find(root *parser.Adaptable, pageURL, cssSelector string) (*parser.Adaptable, error) {
	// Try CSS selector first
	found, err := selector.CSSFirst(root, cssSelector)
	if err == nil && found != nil {
		// Update fingerprint with current state
		_ = tr.Track(pageURL, cssSelector, found)
		return found, nil
	}

	// CSS failed — attempt relocation via fingerprint
	return tr.Relocate(root, pageURL, cssSelector)
}

// Relocate attempts to find the best-matching element on the page using
// stored fingerprints and multi-factor similarity scoring.
func (tr *Tracker) Relocate(root *parser.Adaptable, pageURL, identifier string) (*parser.Adaptable, error) {
	stored, err := tr.store.Load(pageURL, identifier)
	if err != nil {
		return nil, fmt.Errorf("load fingerprint: %w", err)
	}
	if stored == nil {
		return nil, fmt.Errorf("no stored fingerprint for %q on %q", identifier, pageURL)
	}

	// Get all elements on page
	allElements := root.AllElements()
	if len(allElements) == 0 {
		return nil, fmt.Errorf("no elements on page")
	}

	// Score each candidate
	type scored struct {
		elem  *parser.Adaptable
		score float64
	}
	var candidates []scored

	for _, elem := range allElements {
		candidateDict := ElementToDict(elem)
		score := CalculateSimilarityScore(stored, candidateDict)
		if score >= tr.threshold {
			candidates = append(candidates, scored{elem: elem, score: score})
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no element above threshold %.1f", tr.threshold)
	}

	// Sort by score descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	best := candidates[0]

	// Update the stored fingerprint with the relocated element
	_ = tr.Track(pageURL, identifier, best.elem)

	return best.elem, nil
}

// RelocateAll returns all elements above threshold, sorted by score descending.
func (tr *Tracker) RelocateAll(root *parser.Adaptable, pageURL, identifier string) ([]*ScoredMatch, error) {
	stored, err := tr.store.Load(pageURL, identifier)
	if err != nil {
		return nil, fmt.Errorf("load fingerprint: %w", err)
	}
	if stored == nil {
		return nil, fmt.Errorf("no stored fingerprint for %q on %q", identifier, pageURL)
	}

	allElements := root.AllElements()
	var matches []*ScoredMatch

	for _, elem := range allElements {
		candidateDict := ElementToDict(elem)
		score := CalculateSimilarityScore(stored, candidateDict)
		if score >= tr.threshold {
			matches = append(matches, &ScoredMatch{
				Element: elem,
				Score:   score,
			})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	return matches, nil
}

// ScoredMatch pairs an element with its similarity score.
type ScoredMatch struct {
	Element *parser.Adaptable
	Score   float64
}

// TopMatches returns the top N matches for relocation.
func (tr *Tracker) TopMatches(root *parser.Adaptable, pageURL, identifier string, n int) ([]*ScoredMatch, error) {
	all, err := tr.RelocateAll(root, pageURL, identifier)
	if err != nil {
		return nil, err
	}
	if len(all) > n {
		all = all[:n]
	}
	return all, nil
}

// Store returns the underlying storage backend.
func (tr *Tracker) Store() *storage.Store {
	return tr.store
}
