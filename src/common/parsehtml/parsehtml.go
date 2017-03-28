// Package parsehtml provides utilities for parsing HTML-format library documentation, which is
// referenced for type information by the Python "compile" check.
package parsehtml

import (
	"strings"

	"golang.org/x/net/html"
)

// Node wraps html.Node so we can define additional convenience methods
type Node struct {
	*html.Node
}

// Attribute wraps html.Attribute so we can define additional convenience methods
type Attribute struct {
	*html.Attribute
}

// NodeP represents a boolean predicate function on HTML nodes.
type NodeP func(Node) bool

// Text returns the concatenation of the contents of all HTML text child nodes
// of the given node.
func (node Node) Text() string {
	data := ""
	node.OnEachChildNode(IsText, func(n Node) error {
		data = data + n.Data
		return nil
	})
	return data
}

// IsText returns true if the given HTML node is a text node.
func IsText(node Node) bool {
	return node.Type == html.TextNode
}

// HasClass returns a NodeP function that returns true if the HTML node it is passed has a class
// attribute with the value of the given name, otherwise false.
func HasClass(name string) NodeP {
	return Attribute{&html.Attribute{"", "class", name}}.IsAttribute
}

// AttributeIsMember returns true if the given slice of HTML attributes contains the given
// attribute, otherwise false.
func (attribute Attribute) AttributeIsMember(attributes []html.Attribute) bool {
	for _, a := range attributes {
		if a == *attribute.Attribute {
			return true
		}
	}
	return false
}

// IsAttribute returns true if the given HTML node has the given attribute, otherwise false.
func (attribute Attribute) IsAttribute(node Node) bool {
	return attribute.AttributeIsMember(node.Attr)
}

// HasElementName returns a NodeP function that returns true if the HTML node it is passed is an
// element node with the given name, otherwise false.
func HasElementName(name string) NodeP {
	return func(node Node) bool {
		return node.Type == html.ElementNode && node.Data == name
	}
}

// OnEachChildNode executes the given process function on each HTML child node of the given node
// satisfying the given predicate function, stopping if an error occurs.
func (node Node) OnEachChildNode(predicate NodeP, process func(Node) error) error {
	for c := node.FindChildNode(predicate); c.Node != nil; c = c.FindNextNode(predicate) {
		if err := process(c); err != nil {
			return err
		}
	}
	return nil
}

// FindChildNode returns the first HTML child node of the given node that satisfies the given
// predicate function if one exists, otherwise nil.
func (node Node) FindChildNode(predicate NodeP) Node {
	return Node{node.FirstChild}.FindNode(predicate)
}

// FindNextNode returns the next HTML sibling node of the given node that satisfies the given
// predicate function if one exists, otherwise nil.
func (node Node) FindNextNode(predicate NodeP) Node {
	return Node{node.NextSibling}.FindNode(predicate)
}

// FindNode returns the next HTML sibling node, beginning with `node` itself, that satisfies the
// given predicate function if one exists, otherwise nil.
func (node Node) FindNode(predicate NodeP) Node {
	for c := node; c.Node != nil; c = (Node{c.NextSibling}) {
		if predicate(c) {
			return c
		}
	}
	return Node{nil}
}

// NodeIsAll returns the logical conjunction ("and") of the given predicate functions of HTML nodes.
func NodeIsAll(predicates ...NodeP) func(Node) bool {
	return func(node Node) bool {
		for _, p := range predicates {
			if !p(node) {
				return false
			}
		}
		return true
	}
}

// InBetween returns the portion of string `s` appearing after the first occurrence of substring
// `before` (exclusive), assuming it exists, and before the first occurrence of substring `after`
// (exclusive), if it exists.
func InBetween(s, before, after string) string {
	return TrimSince(TrimPast(s, before), after)
}

// TrimPast returns the portion of string `s` appearing after the first occurrence of substring
// `before` (exclusive), assuming it exists.
func TrimPast(s, before string) string {
	return strings.SplitAfterN(s, before, 2)[1]
}

// TrimSince returns the portion of string `s` appearing before the first occurrence of substring
// `after` (exclusive), if it exists, otherwise the entire string.
func TrimSince(s, after string) string {
	return strings.SplitN(s, after, 2)[0]
}
