package selector

import (
	"testing"

	"github.com/JSLEEKR/scrapling-go/pkg/parser"
)

const testHTML = `<html>
<head><title>Test Page</title></head>
<body>
  <div id="main" class="container">
    <h1>Hello World</h1>
    <p class="intro">Welcome to <b>Scrapling</b></p>
    <ul id="list">
      <li class="item">Item 1</li>
      <li class="item">Item 2</li>
      <li class="item special">Item 3</li>
    </ul>
    <a href="https://example.com" title="Example">Link</a>
    <a href="https://test.com" title="Test">Test Link</a>
  </div>
  <footer>
    <p class="footer-text">Footer</p>
  </footer>
</body>
</html>`

func mustParseRoot(t *testing.T) *parser.Adaptable {
	t.Helper()
	root, err := parser.Parse(testHTML)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return root
}

func TestCSSBasicTag(t *testing.T) {
	root := mustParseRoot(t)
	results, err := CSS(root, "h1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 h1, got %d", len(results))
	}
	if results[0].Text() != "Hello World" {
		t.Errorf("expected 'Hello World', got '%s'", results[0].Text())
	}
}

func TestCSSByID(t *testing.T) {
	root := mustParseRoot(t)
	results, err := CSS(root, "#main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 #main, got %d", len(results))
	}
	if results[0].Attr("id") != "main" {
		t.Error("expected id=main")
	}
}

func TestCSSByClass(t *testing.T) {
	root := mustParseRoot(t)
	results, err := CSS(root, ".item")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 .item, got %d", len(results))
	}
}

func TestCSSDescendant(t *testing.T) {
	root := mustParseRoot(t)
	results, err := CSS(root, "div p")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 div p, got %d", len(results))
	}
}

func TestCSSAttrSelector(t *testing.T) {
	root := mustParseRoot(t)
	results, err := CSS(root, `a[href="https://example.com"]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1, got %d", len(results))
	}
}

func TestCSSFirst(t *testing.T) {
	root := mustParseRoot(t)
	first, err := CSSFirst(root, "li")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first == nil {
		t.Fatal("expected non-nil result")
	}
	if first.Text() != "Item 1" {
		t.Errorf("expected 'Item 1', got '%s'", first.Text())
	}
}

func TestCSSFirstNotFound(t *testing.T) {
	root := mustParseRoot(t)
	result, err := CSSFirst(root, "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil for non-existent element")
	}
}

func TestCSSInvalidSelector(t *testing.T) {
	root := mustParseRoot(t)
	_, err := CSS(root, "[[[invalid")
	if err == nil {
		t.Error("expected error for invalid selector")
	}
}

func TestCSSPseudoText(t *testing.T) {
	root := mustParseRoot(t)
	results, err := CSS(root, "h1::text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1, got %d", len(results))
	}
}

func TestCSSPseudoAttr(t *testing.T) {
	root := mustParseRoot(t)
	results, err := CSS(root, "a::attr(href)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 links, got %d", len(results))
	}
}

func TestCSSText(t *testing.T) {
	root := mustParseRoot(t)
	texts, err := CSSText(root, "li")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(texts) != 3 {
		t.Fatalf("expected 3 texts, got %d", len(texts))
	}
	if texts[0] != "Item 1" {
		t.Errorf("expected 'Item 1', got '%s'", texts[0])
	}
}

func TestCSSAttrExtract(t *testing.T) {
	root := mustParseRoot(t)
	hrefs, err := CSSAttr(root, "a", "href")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hrefs) != 2 {
		t.Fatalf("expected 2 hrefs, got %d", len(hrefs))
	}
	if hrefs[0] != "https://example.com" {
		t.Errorf("expected https://example.com, got %s", hrefs[0])
	}
}

func TestXPathAllElements(t *testing.T) {
	root := mustParseRoot(t)
	results, err := XPath(root, ".//*")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) < 10 {
		t.Errorf("expected at least 10 elements, got %d", len(results))
	}
}

func TestXPathByTag(t *testing.T) {
	root := mustParseRoot(t)
	results, err := XPath(root, "//li")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 li, got %d", len(results))
	}
}

func TestXPathByAttrValue(t *testing.T) {
	root := mustParseRoot(t)
	results, err := XPath(root, `//div[@id='main']`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1, got %d", len(results))
	}
}

func TestXPathByHasAttr(t *testing.T) {
	root := mustParseRoot(t)
	results, err := XPath(root, "//a[@href]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2, got %d", len(results))
	}
}

func TestXPathFirst(t *testing.T) {
	root := mustParseRoot(t)
	first, err := XPathFirst(root, "//p")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first == nil {
		t.Fatal("expected non-nil")
	}
}

func TestXPathFirstNotFound(t *testing.T) {
	root := mustParseRoot(t)
	result, err := XPathFirst(root, "//table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil")
	}
}

func TestXPathEmptyExpr(t *testing.T) {
	root := mustParseRoot(t)
	_, err := XPath(root, "")
	if err == nil {
		t.Error("expected error for empty xpath")
	}
}

func TestXPathUnsupported(t *testing.T) {
	root := mustParseRoot(t)
	_, err := XPath(root, "count(//li)")
	if err == nil {
		t.Error("expected error for unsupported xpath")
	}
}

func TestXPathStar(t *testing.T) {
	root := mustParseRoot(t)
	results, err := XPath(root, "//*")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) < 10 {
		t.Errorf("expected many elements, got %d", len(results))
	}
}

func TestXPathDotSlashDirectChildren(t *testing.T) {
	root := mustParseRoot(t)
	body := parser.Body(root)
	if body == nil {
		t.Fatal("no body")
	}
	results, err := XPath(body, "./div")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 direct div child, got %d", len(results))
	}
}

func TestCSSSelectorCaching(t *testing.T) {
	root := mustParseRoot(t)
	// Call twice to exercise cache
	_, err1 := CSS(root, "div.container")
	_, err2 := CSS(root, "div.container")
	if err1 != nil || err2 != nil {
		t.Error("cache should not cause errors")
	}
}
