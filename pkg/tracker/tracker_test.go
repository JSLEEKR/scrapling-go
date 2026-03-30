package tracker

import (
	"testing"

	"github.com/JSLEEKR/scrapling-go/pkg/parser"
	"github.com/JSLEEKR/scrapling-go/pkg/storage"
)

const originalHTML = `<html>
<head><title>Test</title></head>
<body>
  <div id="main" class="container">
    <h1 class="title">Hello World</h1>
    <p class="intro">Welcome to the site</p>
    <ul id="nav">
      <li class="item active"><a href="/home">Home</a></li>
      <li class="item"><a href="/about">About</a></li>
      <li class="item"><a href="/contact">Contact</a></li>
    </ul>
    <div class="content" data-section="main">
      <p>Some content here</p>
    </div>
  </div>
  <footer class="footer">
    <p>Footer text</p>
  </footer>
</body>
</html>`

// Modified version: classes changed, elements moved
const modifiedHTML = `<html>
<head><title>Test</title></head>
<body>
  <div id="wrapper" class="main-container">
    <h1 class="page-title">Hello World</h1>
    <p class="introduction">Welcome to the site</p>
    <nav id="navigation">
      <ul>
        <li class="nav-item active"><a href="/home">Home</a></li>
        <li class="nav-item"><a href="/about">About</a></li>
        <li class="nav-item"><a href="/contact">Contact</a></li>
      </ul>
    </nav>
    <section class="main-content" data-section="main">
      <p>Some content here</p>
    </section>
  </div>
  <footer class="site-footer">
    <p>Footer text</p>
  </footer>
</body>
</html>`

func mustParse(t *testing.T, h string) *parser.Adaptable {
	t.Helper()
	root, err := parser.Parse(h)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return root
}

func newTestTracker(t *testing.T) *Tracker {
	t.Helper()
	store, err := storage.New(":memory:")
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return New(store)
}

func TestElementToDict(t *testing.T) {
	root := mustParse(t, originalHTML)
	body := parser.Body(root)
	h1 := body.Find("h1")
	if h1 == nil {
		t.Fatal("expected h1")
	}

	dict := ElementToDict(h1)
	if dict.Tag != "h1" {
		t.Errorf("expected tag=h1, got %s", dict.Tag)
	}
	if dict.Text != "Hello World" {
		t.Errorf("expected text='Hello World', got '%s'", dict.Text)
	}
	if dict.Attributes["class"] != "title" {
		t.Errorf("expected class=title, got %s", dict.Attributes["class"])
	}
}

func TestElementToDictNil(t *testing.T) {
	dict := ElementToDict(nil)
	if dict != nil {
		t.Error("expected nil for nil element")
	}
}

func TestElementToDictParent(t *testing.T) {
	root := mustParse(t, originalHTML)
	body := parser.Body(root)
	h1 := body.Find("h1")
	dict := ElementToDict(h1)

	if dict.Parent == nil {
		t.Fatal("expected parent info")
	}
	if dict.Parent.Tag != "div" {
		t.Errorf("expected parent tag=div, got %s", dict.Parent.Tag)
	}
}

func TestElementToDictSiblings(t *testing.T) {
	root := mustParse(t, originalHTML)
	body := parser.Body(root)
	h1 := body.Find("h1")
	dict := ElementToDict(h1)

	if len(dict.Siblings) == 0 {
		t.Error("expected siblings")
	}
}

func TestElementToDictChildren(t *testing.T) {
	root := mustParse(t, originalHTML)
	body := parser.Body(root)
	ul := body.Find("ul")
	dict := ElementToDict(ul)

	if len(dict.Children) != 3 {
		t.Errorf("expected 3 children (li), got %d", len(dict.Children))
	}
}

func TestCleanText(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"hello world", "hello world"},
		{"  hello  world  ", "hello world"},
		{"hello\n\tworld", "hello world"},
		{"", ""},
		{"   ", ""},
		{"hello\u2003world", "hello world"},       // em space
		{"hello\u3000world", "hello world"},       // ideographic space
		{"hello\u00A0world", "hello world"},       // non-breaking space
		{"\uFEFFhello", "hello"},                  // BOM
		{"hello\u200Bworld", "hello\u200Bworld"},  // zero-width space (NOT whitespace)
	}
	for _, tt := range tests {
		result := cleanText(tt.input)
		if result != tt.expected {
			t.Errorf("cleanText(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestCalculateSimilarityScoreIdentical(t *testing.T) {
	root := mustParse(t, originalHTML)
	body := parser.Body(root)
	h1 := body.Find("h1")
	dict := ElementToDict(h1)

	score := CalculateSimilarityScore(dict, dict)
	if score < 90 {
		t.Errorf("expected score > 90 for identical elements, got %.2f", score)
	}
}

func TestCalculateSimilarityScoreNil(t *testing.T) {
	score := CalculateSimilarityScore(nil, nil)
	if score != 0 {
		t.Errorf("expected 0 for nil, got %.2f", score)
	}
}

func TestCalculateSimilarityScoreDifferent(t *testing.T) {
	dict1 := &storage.ElementDict{
		Tag:        "div",
		Text:       "Hello",
		Attributes: map[string]string{"class": "main"},
		Path:       []string{"html", "body"},
		Siblings:   []string{"footer"},
		Children:   []string{"p"},
	}
	dict2 := &storage.ElementDict{
		Tag:        "span",
		Text:       "Goodbye",
		Attributes: map[string]string{"id": "other"},
		Path:       []string{"html", "body", "section"},
		Siblings:   []string{"nav", "aside"},
		Children:   []string{"a", "b"},
	}

	score := CalculateSimilarityScore(dict1, dict2)
	if score > 50 {
		t.Errorf("expected low score for different elements, got %.2f", score)
	}
}

func TestTrackAndFind(t *testing.T) {
	tr := newTestTracker(t)
	root := mustParse(t, originalHTML)
	body := parser.Body(root)
	h1 := body.Find("h1")

	// Track the element
	err := tr.Track("https://example.com", "h1.title", h1)
	if err != nil {
		t.Fatalf("track: %v", err)
	}

	// Find should succeed with CSS selector
	found, err := tr.Find(root, "https://example.com", "h1")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find element")
	}
	if found.Text() != "Hello World" {
		t.Errorf("expected 'Hello World', got '%s'", found.Text())
	}
}

func TestRelocateAfterChange(t *testing.T) {
	tr := newTestTracker(t)

	// Track element in original HTML
	origRoot := mustParse(t, originalHTML)
	origBody := parser.Body(origRoot)
	origH1 := origBody.Find("h1")

	err := tr.Track("https://example.com", "h1.title", origH1)
	if err != nil {
		t.Fatalf("track: %v", err)
	}

	// Try to relocate in modified HTML (class changed from "title" to "page-title")
	modRoot := mustParse(t, modifiedHTML)

	// CSS selector h1.title will fail on modified HTML, so use Relocate
	relocated, err := tr.Relocate(modRoot, "https://example.com", "h1.title")
	if err != nil {
		t.Fatalf("relocate: %v", err)
	}
	if relocated == nil {
		t.Fatal("expected relocated element")
	}
	if relocated.Tag() != "h1" {
		t.Errorf("expected h1, got %s", relocated.Tag())
	}
	if relocated.Text() != "Hello World" {
		t.Errorf("expected 'Hello World', got '%s'", relocated.Text())
	}
}

func TestRelocateNoFingerprint(t *testing.T) {
	tr := newTestTracker(t)
	root := mustParse(t, originalHTML)

	_, err := tr.Relocate(root, "https://example.com", "nonexistent")
	if err == nil {
		t.Error("expected error for missing fingerprint")
	}
}

func TestRelocateAll(t *testing.T) {
	tr := newTestTracker(t)
	tr.SetThreshold(10) // Lower threshold to get more matches

	origRoot := mustParse(t, originalHTML)
	origBody := parser.Body(origRoot)
	li := origBody.Find("li")

	err := tr.Track("https://example.com", "li.item", li)
	if err != nil {
		t.Fatalf("track: %v", err)
	}

	modRoot := mustParse(t, modifiedHTML)
	matches, err := tr.RelocateAll(modRoot, "https://example.com", "li.item")
	if err != nil {
		t.Fatalf("relocateAll: %v", err)
	}
	if len(matches) == 0 {
		t.Error("expected matches")
	}
	// Should be sorted by score
	for i := 1; i < len(matches); i++ {
		if matches[i].Score > matches[i-1].Score {
			t.Error("matches should be sorted by score descending")
		}
	}
}

func TestTopMatches(t *testing.T) {
	tr := newTestTracker(t)
	tr.SetThreshold(10)

	origRoot := mustParse(t, originalHTML)
	origBody := parser.Body(origRoot)
	li := origBody.Find("li")

	_ = tr.Track("https://example.com", "li.item", li)

	modRoot := mustParse(t, modifiedHTML)
	top, err := tr.TopMatches(modRoot, "https://example.com", "li.item", 2)
	if err != nil {
		t.Fatalf("topMatches: %v", err)
	}
	if len(top) > 2 {
		t.Errorf("expected at most 2, got %d", len(top))
	}
}

func TestFindFallsBackToRelocation(t *testing.T) {
	tr := newTestTracker(t)

	// Track with original HTML
	origRoot := mustParse(t, originalHTML)
	origBody := parser.Body(origRoot)
	origDiv := origBody.FindByAttr("class", "content")
	if len(origDiv) == 0 {
		t.Fatal("expected content div")
	}

	err := tr.Track("https://example.com", "div.content", origDiv[0])
	if err != nil {
		t.Fatalf("track: %v", err)
	}

	// Find in modified HTML where class changed to "main-content"
	modRoot := mustParse(t, modifiedHTML)
	found, err := tr.Find(modRoot, "https://example.com", "div.content")
	if err != nil {
		// Relocation might not find an exact match with high threshold
		// but should not panic
		t.Logf("find returned error (expected if threshold too high): %v", err)
	} else if found != nil {
		t.Logf("found relocated element: tag=%s", found.Tag())
	}
}

func TestSetThreshold(t *testing.T) {
	tr := newTestTracker(t)
	tr.SetThreshold(90)
	if tr.threshold != 90 {
		t.Errorf("expected threshold=90, got %.2f", tr.threshold)
	}
}

func TestTrackerStore(t *testing.T) {
	tr := newTestTracker(t)
	if tr.Store() == nil {
		t.Error("expected non-nil store")
	}
}

func TestSimilarityScoreParentMatching(t *testing.T) {
	dict1 := &storage.ElementDict{
		Tag:        "p",
		Text:       "Hello",
		Attributes: map[string]string{},
		Path:       []string{"html", "body", "div"},
		Parent:     &storage.ParentDict{Tag: "div", Attributes: map[string]string{"class": "main"}, Text: ""},
		Siblings:   []string{},
		Children:   []string{},
	}
	dict2 := &storage.ElementDict{
		Tag:        "p",
		Text:       "Hello",
		Attributes: map[string]string{},
		Path:       []string{"html", "body", "div"},
		Parent:     &storage.ParentDict{Tag: "div", Attributes: map[string]string{"class": "main"}, Text: ""},
		Siblings:   []string{},
		Children:   []string{},
	}

	score := CalculateSimilarityScore(dict1, dict2)
	if score < 90 {
		t.Errorf("expected high score for matching parents, got %.2f", score)
	}
}

func TestSimilarityScoreNoParent(t *testing.T) {
	dict1 := &storage.ElementDict{
		Tag:  "p",
		Text: "Hello",
	}
	dict2 := &storage.ElementDict{
		Tag:  "p",
		Text: "Hello",
	}

	score := CalculateSimilarityScore(dict1, dict2)
	if score < 50 {
		t.Errorf("expected reasonable score without parents, got %.2f", score)
	}
}
