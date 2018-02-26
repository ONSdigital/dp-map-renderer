package htmlutil

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// CreateNode creates an html Node and sets attributes or adds child nodes according to the type of each value
func CreateNode(data string, dataAtom atom.Atom, values ...interface{}) *html.Node {
	node := &html.Node{
		Type:     html.ElementNode,
		Data:     data,
		DataAtom: dataAtom,
	}
	for _, value := range values {
		switch v := value.(type) {
		case html.Attribute:
			node.Attr = append(node.Attr, v)
		case *html.Node:
			node.AppendChild(v)
		case []*html.Node:
			for _, c := range v {
				node.AppendChild(c)
			}
		case string:
			node.AppendChild(&html.Node{Type: html.TextNode, Data: v})
		}
	}
	return node
}

// AddAttribute adds an attribute to the node
func AddAttribute(node *html.Node, key string, val string) {
	node.Attr = append(node.Attr, html.Attribute{Key: key, Val: val})
}

// ReplaceAttribute adds an attribute to the node, replacing any existing attribute with the same name
func ReplaceAttribute(node *html.Node, key string, val string) {
	var attr []html.Attribute
	for _, a := range node.Attr {
		if a.Key != key {
			attr = append(attr, a)
		}
	}
	node.Attr = append(attr, html.Attribute{Key: key, Val: val})
}

// AppendAttribute appends the new value to any existing attribute with the same name, separating them with a space
func AppendAttribute(node *html.Node, key string, val string) {
	current := ""
	var attr []html.Attribute
	for _, a := range node.Attr {
		if a.Key == key {
			current += " " + a.Val
		} else {
			attr = append(attr, a)
		}
	}
	newValue := strings.Trim(current+" "+val, " ")
	node.Attr = append(attr, html.Attribute{Key: key, Val: newValue})
}

// Attr creates a new Attribute
func Attr(key string, val string) html.Attribute {
	return html.Attribute{Key: key, Val: val}
}

// Text creates a new Text node
func Text(text string) *html.Node {
	return &html.Node{Type: html.TextNode, Data: text}
}

// GetAttribute finds an attribute for the node - returns empty string if not found
func GetAttribute(node *html.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

// FindNode is a depth-first search for the first node of the given type
func FindNode(n *html.Node, a atom.Atom) *html.Node {
	return FindNodeWithAttributes(n, a, nil)
}

// FindNodeWithAttributes is a depth-first search for the first node of the given type with the given attributes
func FindNodeWithAttributes(n *html.Node, a atom.Atom, attr map[string]string) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.DataAtom == a && HasAttributes(c, attr) {
			return c
		}
		gc := FindNodeWithAttributes(c, a, attr)
		if gc != nil {
			return gc
		}
	}
	return nil
}

// HasAttributes returns true if the given node has all the attribute values
func HasAttributes(n *html.Node, attr map[string]string) bool {
	for key, value := range attr {
		if GetAttribute(n, key) != value {
			return false
		}
	}
	return true
}

// FindNodes returns all child nodes of the given type
func FindNodes(n *html.Node, a atom.Atom) []*html.Node {
	return FindNodesWithAttributes(n, a, nil)
}

// FindNodesWithAttributes returns all child nodes of the given type with the given attributes
func FindNodesWithAttributes(n *html.Node, a atom.Atom, attr map[string]string) []*html.Node {
	var result []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.DataAtom == a && HasAttributes(c, attr) {
			result = append(result, c)
		}
		result = append(result, FindNodesWithAttributes(c, a, attr)...)
	}
	return result
}

// FindAllNodes returns all child nodes of any of the given types, in the order in which they are found (a depth-first search)
func FindAllNodes(n *html.Node, all ...atom.Atom) []*html.Node {
	var result []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		for _, a := range all {
			if c.DataAtom == a {
				result = append(result, c)
				break
			}
		}
		result = append(result, FindAllNodes(c, all...)...)
	}
	return result
}

// GetText returns the text content of the given node, including the text content of all child nodes. Extraneous newline characters are removed.
func GetText(n *html.Node) string {
	var buffer bytes.Buffer
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			buffer.WriteString(c.Data)
		} else {
			buffer.WriteString(GetText(c))
		}
	}
	return strings.Trim(buffer.String(), "\n")
}
