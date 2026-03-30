package storage

import (
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func testElem() *ElementDict {
	return &ElementDict{
		Tag:        "div",
		Text:       "Hello World",
		Attributes: map[string]string{"class": "main", "id": "content"},
		Path:       []string{"html", "body"},
		Parent:     &ParentDict{Tag: "body", Attributes: map[string]string{}, Text: ""},
		Siblings:   []string{"footer", "nav"},
		Children:   []string{"h1", "p", "a"},
	}
}

func TestNewStore(t *testing.T) {
	s := newTestStore(t)
	if s == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestSaveAndLoad(t *testing.T) {
	s := newTestStore(t)
	elem := testElem()

	err := s.Save("https://example.com/page", "div#content", elem)
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := s.Load("https://example.com/page", "div#content")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected non-nil loaded element")
	}
	if loaded.Tag != "div" {
		t.Errorf("expected tag=div, got %s", loaded.Tag)
	}
	if loaded.Text != "Hello World" {
		t.Errorf("expected text='Hello World', got '%s'", loaded.Text)
	}
	if loaded.Attributes["id"] != "content" {
		t.Errorf("expected id=content, got %s", loaded.Attributes["id"])
	}
}

func TestLoadNotFound(t *testing.T) {
	s := newTestStore(t)
	loaded, err := s.Load("https://example.com", "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded != nil {
		t.Error("expected nil for not found")
	}
}

func TestSaveOverwrite(t *testing.T) {
	s := newTestStore(t)
	elem1 := testElem()
	elem2 := &ElementDict{Tag: "span", Text: "Updated"}

	_ = s.Save("https://example.com", "sel", elem1)
	_ = s.Save("https://example.com", "sel", elem2)

	loaded, _ := s.Load("https://example.com", "sel")
	if loaded.Tag != "span" {
		t.Errorf("expected overwritten tag=span, got %s", loaded.Tag)
	}
}

func TestDelete(t *testing.T) {
	s := newTestStore(t)
	_ = s.Save("https://example.com", "sel", testElem())

	err := s.Delete("https://example.com", "sel")
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	loaded, _ := s.Load("https://example.com", "sel")
	if loaded != nil {
		t.Error("expected nil after delete")
	}
}

func TestList(t *testing.T) {
	s := newTestStore(t)
	_ = s.Save("https://example.com/p1", "sel1", testElem())
	_ = s.Save("https://example.com/p1", "sel2", testElem())
	_ = s.Save("https://other.com", "sel3", testElem())

	ids, err := s.List("https://example.com/p1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("expected 2 identifiers, got %d", len(ids))
	}
}

func TestCount(t *testing.T) {
	s := newTestStore(t)
	_ = s.Save("https://example.com", "sel1", testElem())
	_ = s.Save("https://example.com", "sel2", testElem())

	count, err := s.Count()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestClear(t *testing.T) {
	s := newTestStore(t)
	_ = s.Save("https://example.com", "sel1", testElem())
	_ = s.Save("https://example.com", "sel2", testElem())

	err := s.Clear()
	if err != nil {
		t.Fatalf("clear: %v", err)
	}

	count, _ := s.Count()
	if count != 0 {
		t.Errorf("expected 0 after clear, got %d", count)
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://www.example.com/page", "example.com/page"},
		{"https://example.com/page", "example.com/page"},
		{"http://example.com", "example.com"},
		{"invalid-url", "invalid-url"},
	}
	for _, tt := range tests {
		result := normalizeURL(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeURL(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSaveWithParentNil(t *testing.T) {
	s := newTestStore(t)
	elem := &ElementDict{
		Tag:        "p",
		Text:       "Test",
		Attributes: map[string]string{},
		Path:       []string{"html", "body"},
		Parent:     nil,
		Siblings:   nil,
		Children:   nil,
	}
	err := s.Save("https://example.com", "p", elem)
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := s.Load("https://example.com", "p")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Parent != nil {
		t.Error("expected nil parent")
	}
}

func TestMultipleURLs(t *testing.T) {
	s := newTestStore(t)
	_ = s.Save("https://a.com", "sel", testElem())
	_ = s.Save("https://b.com", "sel", &ElementDict{Tag: "span"})

	loadedA, _ := s.Load("https://a.com", "sel")
	loadedB, _ := s.Load("https://b.com", "sel")

	if loadedA.Tag != "div" {
		t.Errorf("expected div for a.com, got %s", loadedA.Tag)
	}
	if loadedB.Tag != "span" {
		t.Errorf("expected span for b.com, got %s", loadedB.Tag)
	}
}
