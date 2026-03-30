// Package tracker provides adaptive element tracking with multi-factor
// similarity scoring. When a website changes its DOM structure, the tracker
// can relocate previously-tracked elements by comparing stored fingerprints
// against current page elements.
package tracker

import (
	"unicode"

	"github.com/JSLEEKR/scrapling-go/pkg/parser"
	"github.com/JSLEEKR/scrapling-go/pkg/storage"
)

// ElementToDict converts an Adaptable element into an ElementDict fingerprint
// for storage. This captures tag, text, attributes, DOM path, parent info,
// sibling tags, and child tags.
func ElementToDict(a *parser.Adaptable) *storage.ElementDict {
	if a == nil {
		return nil
	}

	dict := &storage.ElementDict{
		Tag:        a.Tag(),
		Text:       cleanText(a.Text()),
		Attributes: a.Attrs(),
		Path:       a.Path(),
		Siblings:   a.SiblingTags(),
		Children:   a.ChildTags(),
	}

	// Capture parent metadata
	parent := a.Parent()
	if parent != nil && parent.Tag() != "" {
		dict.Parent = &storage.ParentDict{
			Tag:        parent.Tag(),
			Attributes: parent.Attrs(),
			Text:       cleanText(parent.Text()),
		}
	}

	return dict
}

// cleanText normalizes whitespace in text content.
// Uses unicode.IsSpace to handle all Unicode whitespace characters including
// \u2000-\u200A, \u202F, \u205F, \u3000, \uFEFF, etc.
func cleanText(s string) string {
	result := make([]rune, 0, len(s))
	inSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) || r == '\u00A0' || r == '\uFEFF' {
			if !inSpace && len(result) > 0 {
				result = append(result, ' ')
			}
			inSpace = true
		} else {
			result = append(result, r)
			inSpace = false
		}
	}
	// Trim trailing space
	if len(result) > 0 && result[len(result)-1] == ' ' {
		result = result[:len(result)-1]
	}
	return string(result)
}
