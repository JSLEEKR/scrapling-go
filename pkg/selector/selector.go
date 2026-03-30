// Package selector provides CSS and XPath selector engines for HTML documents.
// It supports standard CSS selectors via cascadia, XPath via antchfx/htmlquery,
// and custom pseudo-elements (::text, ::attr()) matching Scrapy/Parsel syntax.
package selector

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"

	"github.com/JSLEEKR/scrapling-go/pkg/parser"
)

var (
	// cssCache caches compiled CSS selectors.
	cssCache   sync.Map
	pseudoText = regexp.MustCompile(`::text\s*$`)
	pseudoAttr = regexp.MustCompile(`::attr\(([^)]+)\)\s*$`)
)

// PseudoResult holds the result of a pseudo-element extraction.
type PseudoResult struct {
	Nodes []*parser.Adaptable
	Texts []string // populated for ::text or ::attr()
}

// CSS selects elements matching a CSS selector from the given root.
func CSS(root *parser.Adaptable, sel string) ([]*parser.Adaptable, error) {
	// Check for pseudo-elements
	pr, err := cssPseudo(root, sel)
	if err != nil {
		return nil, err
	}
	if pr != nil {
		return pr.Nodes, nil
	}

	compiled, err := compileCSS(sel)
	if err != nil {
		return nil, fmt.Errorf("compile css %q: %w", sel, err)
	}

	matches := compiled.MatchAll(root.Node())
	result := make([]*parser.Adaptable, len(matches))
	for i, m := range matches {
		result[i] = parser.NewAdaptable(m)
	}
	return result, nil
}

// CSSFirst selects the first element matching a CSS selector.
func CSSFirst(root *parser.Adaptable, sel string) (*parser.Adaptable, error) {
	// Check for pseudo-elements
	pr, err := cssPseudo(root, sel)
	if err != nil {
		return nil, err
	}
	if pr != nil {
		if len(pr.Nodes) > 0 {
			return pr.Nodes[0], nil
		}
		return nil, nil
	}

	compiled, err := compileCSS(sel)
	if err != nil {
		return nil, fmt.Errorf("compile css %q: %w", sel, err)
	}

	match := compiled.MatchFirst(root.Node())
	if match == nil {
		return nil, nil
	}
	return parser.NewAdaptable(match), nil
}

// CSSText extracts text from elements matching a CSS selector with ::text pseudo-element.
func CSSText(root *parser.Adaptable, sel string) ([]string, error) {
	pr, err := cssPseudo(root, sel+"::text")
	if err != nil {
		return nil, err
	}
	if pr != nil {
		return pr.Texts, nil
	}
	return nil, nil
}

// CSSAttr extracts attribute values from elements matching a CSS selector
// with ::attr(name) pseudo-element.
func CSSAttr(root *parser.Adaptable, sel, attr string) ([]string, error) {
	pr, err := cssPseudo(root, sel+"::attr("+attr+")")
	if err != nil {
		return nil, err
	}
	if pr != nil {
		return pr.Texts, nil
	}
	return nil, nil
}

// cssPseudo handles CSS selectors with ::text or ::attr() pseudo-elements.
func cssPseudo(root *parser.Adaptable, sel string) (*PseudoResult, error) {
	if m := pseudoText.FindStringIndex(sel); m != nil {
		baseSel := strings.TrimSpace(sel[:m[0]])
		if baseSel == "" {
			return nil, fmt.Errorf("empty base selector for ::text")
		}

		compiled, err := compileCSS(baseSel)
		if err != nil {
			return nil, fmt.Errorf("compile css %q: %w", baseSel, err)
		}

		matches := compiled.MatchAll(root.Node())
		var texts []string
		nodes := make([]*parser.Adaptable, 0, len(matches))
		for _, m := range matches {
			ad := parser.NewAdaptable(m)
			nodes = append(nodes, ad)
			texts = append(texts, ad.AllText())
		}
		return &PseudoResult{Nodes: nodes, Texts: texts}, nil
	}

	if m := pseudoAttr.FindStringSubmatch(sel); m != nil {
		attrName := strings.TrimSpace(m[1])
		idx := pseudoAttr.FindStringIndex(sel)
		baseSel := strings.TrimSpace(sel[:idx[0]])
		if baseSel == "" {
			return nil, fmt.Errorf("empty base selector for ::attr()")
		}

		compiled, err := compileCSS(baseSel)
		if err != nil {
			return nil, fmt.Errorf("compile css %q: %w", baseSel, err)
		}

		matches := compiled.MatchAll(root.Node())
		var texts []string
		nodes := make([]*parser.Adaptable, 0, len(matches))
		for _, m := range matches {
			ad := parser.NewAdaptable(m)
			nodes = append(nodes, ad)
			texts = append(texts, ad.Attr(attrName))
		}
		return &PseudoResult{Nodes: nodes, Texts: texts}, nil
	}

	return nil, nil
}

// compileCSS compiles a CSS selector with caching.
func compileCSS(sel string) (cascadia.Selector, error) {
	if cached, ok := cssCache.Load(sel); ok {
		return cached.(cascadia.Selector), nil
	}

	compiled, err := cascadia.Compile(sel)
	if err != nil {
		return nil, err
	}
	cssCache.Store(sel, compiled)
	return compiled, nil
}

// XPath selects elements matching an XPath expression from the given root.
// This is a simplified XPath implementation supporting common patterns.
func XPath(root *parser.Adaptable, expr string) ([]*parser.Adaptable, error) {
	results, err := evaluateXPath(root.Node(), expr)
	if err != nil {
		return nil, fmt.Errorf("xpath %q: %w", expr, err)
	}
	adaptables := make([]*parser.Adaptable, len(results))
	for i, n := range results {
		adaptables[i] = parser.NewAdaptable(n)
	}
	return adaptables, nil
}

// XPathFirst selects the first element matching an XPath expression.
func XPathFirst(root *parser.Adaptable, expr string) (*parser.Adaptable, error) {
	results, err := XPath(root, expr)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	return results[0], nil
}

// evaluateXPath evaluates common XPath expressions against an HTML node tree.
// Supports: //tag, //tag[@attr], //tag[@attr='value'], ./tag, .//tag, //*
func evaluateXPath(root *html.Node, expr string) ([]*html.Node, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, fmt.Errorf("empty xpath expression")
	}

	// .//* or //* — all descendant elements
	if expr == ".//*" || expr == "//*" {
		var results []*html.Node
		collectAllElements(root, &results)
		return results, nil
	}

	// //tag[@attr='value']
	if strings.HasPrefix(expr, "//") {
		return parseDoubleSlash(root, expr[2:])
	}

	// .//tag
	if strings.HasPrefix(expr, ".//") {
		return parseDoubleSlash(root, expr[3:])
	}

	// ./tag — direct children only
	if strings.HasPrefix(expr, "./") {
		tag := expr[2:]
		var results []*html.Node
		for c := root.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && c.Data == tag {
				results = append(results, c)
			}
		}
		return results, nil
	}

	return nil, fmt.Errorf("unsupported xpath expression: %s", expr)
}

// parseDoubleSlash handles //tag, //tag[@attr], //tag[@attr='value']
func parseDoubleSlash(root *html.Node, rest string) ([]*html.Node, error) {
	// //tag[@attr='value'] or //tag[@attr="value"]
	if bracketIdx := strings.Index(rest, "["); bracketIdx >= 0 {
		tag := rest[:bracketIdx]
		predicate := rest[bracketIdx:]
		if !strings.HasPrefix(predicate, "[@") || !strings.HasSuffix(predicate, "]") {
			return nil, fmt.Errorf("unsupported predicate: %s", predicate)
		}
		inner := predicate[2 : len(predicate)-1]

		// [@attr='value'] or [@attr="value"]
		if eqIdx := strings.Index(inner, "="); eqIdx >= 0 {
			attr := inner[:eqIdx]
			val := strings.Trim(inner[eqIdx+1:], "\"' ")
			var results []*html.Node
			collectByTagAttrVal(root, tag, attr, val, &results)
			return results, nil
		}

		// [@attr] — has attribute
		attr := inner
		var results []*html.Node
		collectByTagHasAttr(root, tag, attr, &results)
		return results, nil
	}

	// //tag or //*
	tag := rest
	if tag == "*" {
		var results []*html.Node
		collectAllElements(root, &results)
		return results, nil
	}

	var results []*html.Node
	collectByTag(root, tag, &results)
	return results, nil
}

func collectAllElements(n *html.Node, results *[]*html.Node) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			*results = append(*results, c)
		}
		collectAllElements(c, results)
	}
}

func collectByTag(n *html.Node, tag string, results *[]*html.Node) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			*results = append(*results, c)
		}
		collectByTag(c, tag, results)
	}
}

func collectByTagAttrVal(n *html.Node, tag, attr, val string, results *[]*html.Node) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && (tag == "*" || c.Data == tag) {
			for _, a := range c.Attr {
				if a.Key == attr && a.Val == val {
					*results = append(*results, c)
					break
				}
			}
		}
		collectByTagAttrVal(c, tag, attr, val, results)
	}
}

func collectByTagHasAttr(n *html.Node, tag, attr string, results *[]*html.Node) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && (tag == "*" || c.Data == tag) {
			for _, a := range c.Attr {
				if a.Key == attr {
					*results = append(*results, c)
					break
				}
			}
		}
		collectByTagHasAttr(c, tag, attr, results)
	}
}
