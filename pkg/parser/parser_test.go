package parser

import (
	"strings"
	"testing"
)

const testHTML = `<html>
<head><title>Test Page</title></head>
<body>
  <div id="main" class="container">
    <h1>Hello World</h1>
    <p class="intro">Welcome to <b>Scrapling</b></p>
    <ul>
      <li class="item">Item 1</li>
      <li class="item">Item 2</li>
      <li class="item special">Item 3</li>
    </ul>
    <a href="https://example.com" title="Example">Link</a>
  </div>
  <footer>
    <p>Footer text</p>
  </footer>
</body>
</html>`

func mustParse(t *testing.T, h string) *Adaptable {
	t.Helper()
	a, err := Parse(h)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return a
}

func TestParse(t *testing.T) {
	root := mustParse(t, testHTML)
	if root == nil {
		t.Fatal("expected non-nil root")
	}
}

func TestParseReader(t *testing.T) {
	root, err := ParseReader(strings.NewReader(testHTML))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root == nil {
		t.Fatal("expected non-nil root")
	}
}

func TestParseFragment(t *testing.T) {
	nodes, err := ParseFragment("<p>hello</p><p>world</p>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
}

func TestBody(t *testing.T) {
	root := mustParse(t, testHTML)
	body := Body(root)
	if body == nil {
		t.Fatal("expected body")
	}
	if body.Tag() != "body" {
		t.Errorf("expected body tag, got %s", body.Tag())
	}
}

func TestHead(t *testing.T) {
	root := mustParse(t, testHTML)
	head := Head(root)
	if head == nil {
		t.Fatal("expected head")
	}
	if head.Tag() != "head" {
		t.Errorf("expected head tag, got %s", head.Tag())
	}
}

func TestTag(t *testing.T) {
	root := mustParse(t, testHTML)
	body := Body(root)
	div := body.Find("div")
	if div == nil {
		t.Fatal("expected div")
	}
	if div.Tag() != "div" {
		t.Errorf("expected div, got %s", div.Tag())
	}
}

func TestText(t *testing.T) {
	root := mustParse(t, testHTML)
	body := Body(root)
	h1 := body.Find("h1")
	if h1 == nil {
		t.Fatal("expected h1")
	}
	if h1.Text() != "Hello World" {
		t.Errorf("expected 'Hello World', got '%s'", h1.Text())
	}
}

func TestAllText(t *testing.T) {
	root := mustParse(t, testHTML)
	body := Body(root)
	p := body.Find("p")
	if p == nil {
		t.Fatal("expected p")
	}
	text := p.AllText()
	if !strings.Contains(text, "Welcome") || !strings.Contains(text, "Scrapling") {
		t.Errorf("expected text with Welcome and Scrapling, got '%s'", text)
	}
}

func TestAttr(t *testing.T) {
	root := mustParse(t, testHTML)
	body := Body(root)
	div := body.Find("div")
	if div.Attr("id") != "main" {
		t.Errorf("expected id=main, got %s", div.Attr("id"))
	}
	if div.Attr("class") != "container" {
		t.Errorf("expected class=container, got %s", div.Attr("class"))
	}
}

func TestHasAttr(t *testing.T) {
	root := mustParse(t, testHTML)
	body := Body(root)
	div := body.Find("div")
	if !div.HasAttr("id") {
		t.Error("expected HasAttr(id) to be true")
	}
	if div.HasAttr("nonexistent") {
		t.Error("expected HasAttr(nonexistent) to be false")
	}
}

func TestAttrs(t *testing.T) {
	root := mustParse(t, testHTML)
	body := Body(root)
	div := body.Find("div")
	attrs := div.Attrs()
	if attrs["id"] != "main" {
		t.Error("expected id=main in attrs")
	}
}

func TestAttrKeys(t *testing.T) {
	root := mustParse(t, testHTML)
	body := Body(root)
	div := body.Find("div")
	keys := div.AttrKeys()
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}

func TestParent(t *testing.T) {
	root := mustParse(t, testHTML)
	body := Body(root)
	h1 := body.Find("h1")
	parent := h1.Parent()
	if parent == nil {
		t.Fatal("expected parent")
	}
	if parent.Tag() != "div" {
		t.Errorf("expected div parent, got %s", parent.Tag())
	}
}

func TestChildren(t *testing.T) {
	root := mustParse(t, testHTML)
	body := Body(root)
	ul := body.Find("ul")
	if ul == nil {
		t.Fatal("expected ul")
	}
	children := ul.Children()
	if len(children) != 3 {
		t.Errorf("expected 3 children (li), got %d", len(children))
	}
}

func TestNextSibling(t *testing.T) {
	root := mustParse(t, testHTML)
	body := Body(root)
	div := body.Find("div")
	next := div.NextSibling()
	if next == nil {
		t.Fatal("expected next sibling")
	}
	if next.Tag() != "footer" {
		t.Errorf("expected footer, got %s", next.Tag())
	}
}

func TestPrevSibling(t *testing.T) {
	root := mustParse(t, testHTML)
	body := Body(root)
	footer := body.Find("footer")
	prev := footer.PrevSibling()
	if prev == nil {
		t.Fatal("expected prev sibling")
	}
	if prev.Tag() != "div" {
		t.Errorf("expected div, got %s", prev.Tag())
	}
}

func TestSiblings(t *testing.T) {
	root := mustParse(t, testHTML)
	body := Body(root)
	div := body.Find("div")
	sibs := div.Siblings()
	if len(sibs) != 1 {
		t.Errorf("expected 1 sibling (footer), got %d", len(sibs))
	}
}

func TestSiblingTags(t *testing.T) {
	root := mustParse(t, testHTML)
	body := Body(root)
	ul := body.Find("ul")
	lis := ul.Children()
	if len(lis) < 1 {
		t.Fatal("expected li children")
	}
	tags := lis[0].SiblingTags()
	if len(tags) != 2 {
		t.Errorf("expected 2 sibling tags, got %d", len(tags))
	}
}

func TestChildTags(t *testing.T) {
	root := mustParse(t, testHTML)
	body := Body(root)
	div := body.Find("div")
	tags := div.ChildTags()
	expected := []string{"h1", "p", "ul", "a"}
	if len(tags) != len(expected) {
		t.Errorf("expected %d child tags, got %d: %v", len(expected), len(tags), tags)
	}
}

func TestAncestors(t *testing.T) {
	root := mustParse(t, testHTML)
	body := Body(root)
	h1 := body.Find("h1")
	ancestors := h1.Ancestors()
	if len(ancestors) < 2 {
		t.Errorf("expected at least 2 ancestors, got %d", len(ancestors))
	}
}

func TestPath(t *testing.T) {
	root := mustParse(t, testHTML)
	body := Body(root)
	h1 := body.Find("h1")
	path := h1.Path()
	if len(path) < 2 {
		t.Errorf("expected at least 2 elements in path, got %d", len(path))
	}
}

func TestPathString(t *testing.T) {
	root := mustParse(t, testHTML)
	body := Body(root)
	h1 := body.Find("h1")
	ps := h1.PathString()
	if !strings.Contains(ps, "body") || !strings.Contains(ps, "h1") {
		t.Errorf("expected path with body and h1, got %s", ps)
	}
}

func TestDepth(t *testing.T) {
	root := mustParse(t, testHTML)
	body := Body(root)
	h1 := body.Find("h1")
	d := h1.Depth()
	if d < 2 {
		t.Errorf("expected depth >= 2, got %d", d)
	}
}

func TestHTML(t *testing.T) {
	root := mustParse(t, testHTML)
	body := Body(root)
	h1 := body.Find("h1")
	h := h1.HTML()
	if !strings.Contains(h, "<h1>") {
		t.Errorf("expected <h1> in HTML, got %s", h)
	}
}

func TestInnerHTML(t *testing.T) {
	root := mustParse(t, testHTML)
	body := Body(root)
	h1 := body.Find("h1")
	inner := h1.InnerHTML()
	if inner != "Hello World" {
		t.Errorf("expected 'Hello World', got '%s'", inner)
	}
}

func TestFindAll(t *testing.T) {
	root := mustParse(t, testHTML)
	body := Body(root)
	lis := body.FindAll("li")
	if len(lis) != 3 {
		t.Errorf("expected 3 li elements, got %d", len(lis))
	}
}

func TestFindByAttr(t *testing.T) {
	root := mustParse(t, testHTML)
	body := Body(root)
	items := body.FindByAttr("class", "item")
	if len(items) != 2 {
		t.Errorf("expected 2 items with class=item, got %d", len(items))
	}
}

func TestFindByText(t *testing.T) {
	root := mustParse(t, testHTML)
	body := Body(root)
	results := body.FindByText("Item 2")
	if len(results) == 0 {
		t.Error("expected to find elements with text 'Item 2'")
	}
}

func TestAllElements(t *testing.T) {
	root := mustParse(t, testHTML)
	body := Body(root)
	all := body.AllElements()
	if len(all) < 8 {
		t.Errorf("expected at least 8 elements, got %d", len(all))
	}
}

func TestNewAdaptableNil(t *testing.T) {
	a := NewAdaptable(nil)
	if a != nil {
		t.Error("expected nil for nil node")
	}
}

func TestAttrValues(t *testing.T) {
	root := mustParse(t, testHTML)
	body := Body(root)
	a := body.Find("a")
	if a == nil {
		t.Fatal("expected <a> element")
	}
	vals := a.AttrValues()
	if len(vals) != 2 {
		t.Errorf("expected 2 attr values, got %d", len(vals))
	}
}
