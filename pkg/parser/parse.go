package parser

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Parse parses an HTML string and returns the root Adaptable node.
func Parse(htmlStr string) (*Adaptable, error) {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}
	return NewAdaptable(doc), nil
}

// ParseReader parses HTML from a reader and returns the root Adaptable node.
func ParseReader(r io.Reader) (*Adaptable, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parse html from reader: %w", err)
	}
	return NewAdaptable(doc), nil
}

// ParseFragment parses an HTML fragment within a body context.
func ParseFragment(htmlStr string) ([]*Adaptable, error) {
	ctx := &html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.Body,
		Data:     "body",
	}
	nodes, err := html.ParseFragment(strings.NewReader(htmlStr), ctx)
	if err != nil {
		return nil, fmt.Errorf("parse html fragment: %w", err)
	}
	result := make([]*Adaptable, len(nodes))
	for i, n := range nodes {
		result[i] = NewAdaptable(n)
	}
	return result, nil
}

// Body returns the <body> element from a parsed document, or nil.
func Body(root *Adaptable) *Adaptable {
	return findElement(root.node, "body")
}

// Head returns the <head> element from a parsed document, or nil.
func Head(root *Adaptable) *Adaptable {
	return findElement(root.node, "head")
}

func findElement(n *html.Node, tag string) *Adaptable {
	if n.Type == html.ElementNode && n.Data == tag {
		return NewAdaptable(n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findElement(c, tag); found != nil {
			return found
		}
	}
	return nil
}
