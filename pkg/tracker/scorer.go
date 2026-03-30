package tracker

import (
	"strings"

	"github.com/JSLEEKR/scrapling-go/pkg/similarity"
	"github.com/JSLEEKR/scrapling-go/pkg/storage"
)

// ScoredElement pairs an element fingerprint with its similarity score.
type ScoredElement struct {
	Index int
	Score float64
}

// CalculateSimilarityScore computes a weighted similarity score between
// a stored element fingerprint and a candidate element fingerprint.
// Returns a score from 0 to 100.
func CalculateSimilarityScore(stored, candidate *storage.ElementDict) float64 {
	if stored == nil || candidate == nil {
		return 0
	}

	totalScore := 0.0
	checks := 0

	// 1. Tag match (binary)
	checks++
	if stored.Tag == candidate.Tag {
		totalScore += 1.0
	}

	// 2. Text similarity
	checks++
	totalScore += similarity.StringRatio(stored.Text, candidate.Text)

	// 3. Attribute keys match (set intersection/union)
	storedKeys := mapKeys(stored.Attributes)
	candidateKeys := mapKeys(candidate.Attributes)
	checks++
	totalScore += similarity.SetRatio(storedKeys, candidateKeys)

	// 4. Attribute values match (set intersection/union)
	storedVals := mapValues(stored.Attributes)
	candidateVals := mapValues(candidate.Attributes)
	checks++
	totalScore += similarity.SetRatio(storedVals, candidateVals)

	// 5. Class attribute similarity
	checks++
	totalScore += similarity.StringRatio(
		stored.Attributes["class"],
		candidate.Attributes["class"],
	)

	// 6. ID attribute similarity
	checks++
	totalScore += similarity.StringRatio(
		stored.Attributes["id"],
		candidate.Attributes["id"],
	)

	// 7. href attribute similarity
	checks++
	totalScore += similarity.StringRatio(
		stored.Attributes["href"],
		candidate.Attributes["href"],
	)

	// 8. src attribute similarity
	checks++
	totalScore += similarity.StringRatio(
		stored.Attributes["src"],
		candidate.Attributes["src"],
	)

	// 9. XPath/path similarity
	checks++
	storedPath := strings.Join(stored.Path, "/")
	candidatePath := strings.Join(candidate.Path, "/")
	totalScore += similarity.StringRatio(storedPath, candidatePath)

	// 10. Parent tag match (binary)
	if stored.Parent != nil && candidate.Parent != nil {
		checks++
		if stored.Parent.Tag == candidate.Parent.Tag {
			totalScore += 1.0
		}

		// 11. Parent attributes similarity
		checks++
		parentStoredKeys := mapKeys(stored.Parent.Attributes)
		parentCandidateKeys := mapKeys(candidate.Parent.Attributes)
		totalScore += similarity.SetRatio(parentStoredKeys, parentCandidateKeys)

		// 12. Parent text similarity
		checks++
		totalScore += similarity.StringRatio(stored.Parent.Text, candidate.Parent.Text)
	}

	// 13. Sibling count similarity
	checks++
	storedSibCount := len(stored.Siblings)
	candidateSibCount := len(candidate.Siblings)
	if storedSibCount == 0 && candidateSibCount == 0 {
		totalScore += 1.0
	} else if storedSibCount > 0 && candidateSibCount > 0 {
		minSib := storedSibCount
		maxSib := candidateSibCount
		if minSib > maxSib {
			minSib, maxSib = maxSib, minSib
		}
		totalScore += float64(minSib) / float64(maxSib)
	}

	if checks == 0 {
		return 0
	}
	score := (totalScore / float64(checks)) * 100
	// Round to 2 decimal places
	return float64(int(score*100+0.5)) / 100
}

func mapKeys(m map[string]string) []string {
	if m == nil {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func mapValues(m map[string]string) []string {
	if m == nil {
		return nil
	}
	vals := make([]string, 0, len(m))
	for _, v := range m {
		vals = append(vals, v)
	}
	return vals
}
