// Package parser provides HTML parsing with rich node traversal.
// It wraps golang.org/x/net/html nodes into Adaptable structs with
// parent, children, sibling navigation, text extraction, and attribute access.
package parser

import (
	"strings"

	"golang.org/x/net/html"
)

// Adaptable wraps an html.Node with rich traversal and extraction methods.
type Adaptable struct {
	node *html.Node
}

// NewAdaptable wraps an html.Node into an Adaptable.
func NewAdaptable(n *html.Node) *Adaptable {
	if n == nil {
		return nil
	}
	return &Adaptable{node: n}
}

// Node returns the underlying html.Node.
func (a *Adaptable) Node() *html.Node {
	return a.node
}

// Tag returns the element's tag name. Returns empty string for non-element nodes.
func (a *Adaptable) Tag() string {
	if a.node.Type == html.ElementNode {
		return a.node.Data
	}
	return ""
}

// Text returns the direct text content of this element (not recursive).
func (a *Adaptable) Text() string {
	var sb strings.Builder
	for c := a.node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			sb.WriteString(c.Data)
		}
	}
	return strings.TrimSpace(sb.String())
}

// AllText returns all text content recursively from this element and descendants.
func (a *Adaptable) AllText() string {
	var sb strings.Builder
	collectText(a.node, &sb)
	return strings.TrimSpace(sb.String())
}

func collectText(n *html.Node, sb *strings.Builder) {
	if n.Type == html.TextNode {
		sb.WriteString(n.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectText(c, sb)
	}
}

// Attr returns the value of the named attribute, or empty string if not found.
func (a *Adaptable) Attr(name string) string {
	for _, attr := range a.node.Attr {
		if attr.Key == name {
			return attr.Val
		}
	}
	return ""
}

// HasAttr checks if the element has the named attribute.
func (a *Adaptable) HasAttr(name string) bool {
	for _, attr := range a.node.Attr {
		if attr.Key == name {
			return true
		}
	}
	return false
}

// Attrs returns all attributes as a map.
func (a *Adaptable) Attrs() map[string]string {
	m := make(map[string]string, len(a.node.Attr))
	for _, attr := range a.node.Attr {
		m[attr.Key] = attr.Val
	}
	return m
}

// AttrKeys returns all attribute key names.
func (a *Adaptable) AttrKeys() []string {
	keys := make([]string, 0, len(a.node.Attr))
	for _, attr := range a.node.Attr {
		keys = append(keys, attr.Key)
	}
	return keys
}

// AttrValues returns all attribute values.
func (a *Adaptable) AttrValues() []string {
	vals := make([]string, 0, len(a.node.Attr))
	for _, attr := range a.node.Attr {
		vals = append(vals, attr.Val)
	}
	return vals
}

// Parent returns the parent element, or nil.
func (a *Adaptable) Parent() *Adaptable {
	if a.node.Parent == nil {
		return nil
	}
	return NewAdaptable(a.node.Parent)
}

// Children returns all direct child elements (skipping text/comment nodes).
func (a *Adaptable) Children() []*Adaptable {
	var children []*Adaptable
	for c := a.node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			children = append(children, NewAdaptable(c))
		}
	}
	return children
}

// AllChildren returns all direct child nodes including text nodes.
func (a *Adaptable) AllChildren() []*Adaptable {
	var children []*Adaptable
	for c := a.node.FirstChild; c != nil; c = c.NextSibling {
		children = append(children, NewAdaptable(c))
	}
	return children
}

// NextSibling returns the next sibling element, or nil.
func (a *Adaptable) NextSibling() *Adaptable {
	for n := a.node.NextSibling; n != nil; n = n.NextSibling {
		if n.Type == html.ElementNode {
			return NewAdaptable(n)
		}
	}
	return nil
}

// PrevSibling returns the previous sibling element, or nil.
func (a *Adaptable) PrevSibling() *Adaptable {
	for n := a.node.PrevSibling; n != nil; n = n.PrevSibling {
		if n.Type == html.ElementNode {
			return NewAdaptable(n)
		}
	}
	return nil
}

// Siblings returns all sibling elements (excluding self).
func (a *Adaptable) Siblings() []*Adaptable {
	if a.node.Parent == nil {
		return nil
	}
	var siblings []*Adaptable
	for c := a.node.Parent.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c != a.node {
			siblings = append(siblings, NewAdaptable(c))
		}
	}
	return siblings
}

// SiblingTags returns the tag names of all sibling elements.
func (a *Adaptable) SiblingTags() []string {
	sibs := a.Siblings()
	tags := make([]string, len(sibs))
	for i, s := range sibs {
		tags[i] = s.Tag()
	}
	return tags
}

// ChildTags returns tag names of all direct child elements.
func (a *Adaptable) ChildTags() []string {
	children := a.Children()
	tags := make([]string, len(children))
	for i, c := range children {
		tags[i] = c.Tag()
	}
	return tags
}

// Ancestors returns all ancestor elements from parent to root.
func (a *Adaptable) Ancestors() []*Adaptable {
	var ancestors []*Adaptable
	for n := a.node.Parent; n != nil; n = n.Parent {
		if n.Type == html.ElementNode {
			ancestors = append(ancestors, NewAdaptable(n))
		}
	}
	return ancestors
}

// Path returns the DOM path as a slice of tag names from root to this element.
func (a *Adaptable) Path() []string {
	ancestors := a.Ancestors()
	path := make([]string, len(ancestors))
	for i, anc := range ancestors {
		path[len(ancestors)-1-i] = anc.Tag()
	}
	return path
}

// PathString returns the DOM path as a slash-separated string.
func (a *Adaptable) PathString() string {
	return "/" + strings.Join(append(a.Path(), a.Tag()), "/")
}

// Depth returns the nesting depth of this element (root = 0).
func (a *Adaptable) Depth() int {
	depth := 0
	for n := a.node.Parent; n != nil; n = n.Parent {
		if n.Type == html.ElementNode {
			depth++
		}
	}
	return depth
}

// HTML returns the outer HTML of this element.
func (a *Adaptable) HTML() string {
	var sb strings.Builder
	if err := html.Render(&sb, a.node); err != nil {
		return ""
	}
	return sb.String()
}

// InnerHTML returns the inner HTML of this element.
func (a *Adaptable) InnerHTML() string {
	var sb strings.Builder
	for c := a.node.FirstChild; c != nil; c = c.NextSibling {
		if err := html.Render(&sb, c); err != nil {
			return ""
		}
	}
	return sb.String()
}

// FindAll finds all descendant elements matching the given tag name.
func (a *Adaptable) FindAll(tag string) []*Adaptable {
	var result []*Adaptable
	findAllByTag(a.node, tag, &result)
	return result
}

func findAllByTag(n *html.Node, tag string, result *[]*Adaptable) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			*result = append(*result, NewAdaptable(c))
		}
		findAllByTag(c, tag, result)
	}
}

// Find finds the first descendant element matching the given tag name.
func (a *Adaptable) Find(tag string) *Adaptable {
	return findFirstByTag(a.node, tag)
}

func findFirstByTag(n *html.Node, tag string) *Adaptable {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			return NewAdaptable(c)
		}
		if found := findFirstByTag(c, tag); found != nil {
			return found
		}
	}
	return nil
}

// FindByAttr finds all descendant elements with the given attribute key=value.
func (a *Adaptable) FindByAttr(key, value string) []*Adaptable {
	var result []*Adaptable
	findByAttr(a.node, key, value, &result)
	return result
}

func findByAttr(n *html.Node, key, value string, result *[]*Adaptable) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			for _, attr := range c.Attr {
				if attr.Key == key && attr.Val == value {
					*result = append(*result, NewAdaptable(c))
					break
				}
			}
		}
		findByAttr(c, key, value, result)
	}
}

// FindByText finds all descendant elements containing the given text.
func (a *Adaptable) FindByText(text string) []*Adaptable {
	var result []*Adaptable
	findByText(a.node, text, &result)
	return result
}

func findByText(n *html.Node, text string, result *[]*Adaptable) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			ad := NewAdaptable(c)
			if strings.Contains(ad.AllText(), text) {
				*result = append(*result, ad)
			}
		}
		findByText(c, text, result)
	}
}

// AllElements returns all descendant elements in document order.
func (a *Adaptable) AllElements() []*Adaptable {
	var result []*Adaptable
	collectElements(a.node, &result)
	return result
}

func collectElements(n *html.Node, result *[]*Adaptable) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			*result = append(*result, NewAdaptable(c))
		}
		collectElements(c, result)
	}
}
