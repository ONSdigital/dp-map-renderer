package htmlutil_test

import (
	"testing"

	. "github.com/ONSdigital/dp-map-renderer/htmlutil"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func TestCreateNode(t *testing.T) {
	Convey("CreateNode should return a Node with the appropriate data type", t, func() {

		result := CreateNode("div", atom.Div)

		So(result, ShouldNotBeNil)
		So(result.Type, ShouldEqual, html.ElementNode)
		So(result.DataAtom, ShouldEqual, atom.Div)
		So(result.Data, ShouldEqual, "div")
		So(result.FirstChild, ShouldBeNil)
		So(result.Attr, ShouldBeEmpty)
	})

	Convey("CreateNode should return a Node with the correct attributes and child nodes", t, func() {

		result := CreateNode("div", atom.Div,
			Attr("foo", "bar"),
			Attr("bar", "baz"),
			"\n",
			CreateNode("p", atom.P))

		So(result, ShouldNotBeNil)

		attributes := result.Attr
		So(len(attributes), ShouldEqual, 2)
		So(attributes[0].Key, ShouldEqual, "foo")
		So(attributes[0].Val, ShouldEqual, "bar")
		So(attributes[1].Key, ShouldEqual, "bar")
		So(attributes[1].Val, ShouldEqual, "baz")

		child := result.FirstChild
		So(child, ShouldNotBeNil)
		So(child.Type, ShouldEqual, html.TextNode)

		child = result.LastChild
		So(child, ShouldNotBeNil)
		So(child.Type, ShouldEqual, html.ElementNode)
		So(child.DataAtom, ShouldEqual, atom.P)
	})

}

func TestAddAttribute(t *testing.T) {
	Convey("AddAttribute should add an attribute", t, func() {

		node := CreateNode("div", atom.Div)

		AddAttribute(node, "foo", "bar")
		So(len(node.Attr), ShouldEqual, 1)
		So(node.Attr[0].Key, ShouldEqual, "foo")
		So(node.Attr[0].Val, ShouldEqual, "bar")
	})
}

func TestReplaceAttribute(t *testing.T) {
	Convey("ReplaceAttribute should replace an existing attribute", t, func() {

		node := CreateNode("div", atom.Div, Attr("foo", "bar"))

		ReplaceAttribute(node, "foo", "baz")
		So(len(node.Attr), ShouldEqual, 1)
		So(node.Attr[0].Key, ShouldEqual, "foo")
		So(node.Attr[0].Val, ShouldEqual, "baz")
	})

	Convey("ReplaceAttribute should add an attribute if it doesn't already exist", t, func() {

		node := CreateNode("div", atom.Div, Attr("foo", "bar"))

		ReplaceAttribute(node, "bar", "baz")
		So(len(node.Attr), ShouldEqual, 2)
		So(node.Attr[0].Key, ShouldEqual, "foo")
		So(node.Attr[0].Val, ShouldEqual, "bar")
		So(node.Attr[1].Key, ShouldEqual, "bar")
		So(node.Attr[1].Val, ShouldEqual, "baz")
	})

	Convey("ReplaceAttribute should replace an existing attribute, leaving other attributes in place", t, func() {

		node := CreateNode("div", atom.Div, Attr("foo", "bar"), Attr("bar", "baz"))

		ReplaceAttribute(node, "bar", "foo")
		So(len(node.Attr), ShouldEqual, 2)
		So(node.Attr[0].Key, ShouldEqual, "foo")
		So(node.Attr[0].Val, ShouldEqual, "bar")
		So(node.Attr[1].Key, ShouldEqual, "bar")
		So(node.Attr[1].Val, ShouldEqual, "foo")
	})
}

func TestAppendAttribute(t *testing.T) {
	Convey("AppendAttribute should append to an existing attribute", t, func() {

		node := CreateNode("div", atom.Div, Attr("foo", "bar"))

		AppendAttribute(node, "foo", "baz")
		So(len(node.Attr), ShouldEqual, 1)
		So(node.Attr[0].Key, ShouldEqual, "foo")
		So(node.Attr[0].Val, ShouldEqual, "bar baz")
	})

	Convey("AppendAttribute should add an attribute if it doesn't already exist", t, func() {

		node := CreateNode("div", atom.Div)

		AppendAttribute(node, "foo", "bar")
		So(len(node.Attr), ShouldEqual, 1)
		So(node.Attr[0].Val, ShouldEqual, "bar")
	})

	Convey("AppendAttribute should leave other attributes in place", t, func() {

		node := CreateNode("div", atom.Div, Attr("foo", "bar"), Attr("bar", "baz"))

		AppendAttribute(node, "bar", "foo")
		So(len(node.Attr), ShouldEqual, 2)
		So(node.Attr[0].Key, ShouldEqual, "foo")
		So(node.Attr[0].Val, ShouldEqual, "bar")
		So(node.Attr[1].Key, ShouldEqual, "bar")
		So(node.Attr[1].Val, ShouldEqual, "baz foo")
	})
}

func TestAttr(t *testing.T) {
	Convey("Attr should create an attribute", t, func() {

		result := Attr("foo", "bar")

		So(result, ShouldNotBeNil)
		So(result.Key, ShouldEqual, "foo")
		So(result.Val, ShouldEqual, "bar")
	})
}

func TestText(t *testing.T) {
	Convey("Text should create a text node", t, func() {

		result := Text("foo")

		So(result, ShouldNotBeNil)
		So(result.Type, ShouldEqual, html.TextNode)
		So(result.Data, ShouldEqual, "foo")

	})
}

func TestGetAttribute(t *testing.T) {
	Convey("GetAttribute should return the value of an existing attribute", t, func() {

		node := CreateNode("div", atom.Div, Attr("foo", "bar"), Attr("bar", "baz"))

		result := GetAttribute(node, "foo")
		So(result, ShouldEqual, "bar")

		result = GetAttribute(node, "bar")
		So(result, ShouldEqual, "baz")
	})

	Convey("GetAttribute should return an empty string when the attribute does not exist", t, func() {

		node := CreateNode("div", atom.Div, Attr("foo", "bar"))

		result := GetAttribute(node, "bar")
		So(result, ShouldEqual, "")
	})
}

func TestFindNode(t *testing.T) {
	Convey("FindNode should return the first node of the requested type (depth-first)", t, func() {

		node := CreateNode("div", atom.Div,
			CreateNode("p", atom.P, CreateNode("span", atom.Span, Attr("position", "first"))),
			CreateNode("span", atom.Span, Attr("position", "second")))

		result := FindNode(node, atom.Span)
		So(result, ShouldNotBeNil)
		So(GetAttribute(result, "position"), ShouldEqual, "first")
	})

	Convey("FindNode should return nil if there is no child node of the requested type", t, func() {

		node := CreateNode("div", atom.Div,
			CreateNode("p", atom.P))

		result := FindNode(node, atom.Span)
		So(result, ShouldBeNil)
	})
}

func TestFindNodeWithAttributes(t *testing.T) {
	Convey("FindNode should return the node with the requested attributes", t, func() {

		node := CreateNode("div", atom.Div,
			CreateNode("p", atom.P, CreateNode("span", atom.Span, Attr("position", "first"))),
			CreateNode("span", atom.Span, Attr("position", "second")))

		result := FindNodeWithAttributes(node, atom.Span, map[string]string{"position": "second"})
		So(result, ShouldNotBeNil)
		So(result.DataAtom, ShouldEqual, atom.Span)
		So(GetAttribute(result, "position"), ShouldEqual, "second")
	})

	Convey("FindNode should return nil if there is no child node of the requested type", t, func() {

		node := CreateNode("div", atom.Div,
			CreateNode("p", atom.P, CreateNode("span", atom.Span, Attr("position", "first"))),
			CreateNode("span", atom.Span, Attr("position", "second")))

		result := FindNodeWithAttributes(node, atom.Span, map[string]string{"position": "third"})
		So(result, ShouldBeNil)
	})

}

func TestHasAttributes(t *testing.T) {
	Convey("HasAttributes should return true if the node has the attributes", t, func() {

		node := CreateNode("div", atom.Div, Attr("foo", "bar"), Attr("bar", "baz"))

		So(HasAttributes(node, map[string]string{"foo": "bar"}), ShouldBeTrue)
		So(HasAttributes(node, map[string]string{"foo": "bar", "bar": "baz"}), ShouldBeTrue)
		So(HasAttributes(node, map[string]string{"foo": "false"}), ShouldBeFalse)
		So(HasAttributes(node, map[string]string{"foo": "false", "bar": "baz"}), ShouldBeFalse)

	})
}

func TestFindNodes(t *testing.T) {
	Convey("FindNodes should return the all nodes of the requested type", t, func() {

		node := CreateNode("div", atom.Div,
			CreateNode("p", atom.P, CreateNode("span", atom.Span, Attr("position", "first"))),
			CreateNode("span", atom.Span, Attr("position", "second")))

		result := FindNodes(node, atom.Span)
		So(len(result), ShouldEqual, 2)
		So(GetAttribute(result[0], "position"), ShouldEqual, "first")
		So(GetAttribute(result[1], "position"), ShouldEqual, "second")
	})

	Convey("FindNodes should return nil if there is no child node of the requested type", t, func() {

		node := CreateNode("div", atom.Div,
			CreateNode("p", atom.P))

		result := FindNodes(node, atom.Span)
		So(result, ShouldBeNil)
	})
}

func TestFindNodesWithAttributes(t *testing.T) {
	Convey("FindNodesWithAttributes should return the all nodes with the requested attributes", t, func() {

		node := CreateNode("div", atom.Div,
			CreateNode("p", atom.P, CreateNode("span", atom.Span, Attr("match", "true"))),
			CreateNode("span", atom.Span, Attr("match", "false")),
			CreateNode("span", atom.Span, Attr("match", "true")))

		result := FindNodesWithAttributes(node, atom.Span, map[string]string{"match": "true"})
		So(len(result), ShouldEqual, 2)
		for _, child := range result {
			So(child.DataAtom, ShouldEqual, atom.Span)
			So(HasAttributes(child, map[string]string{"match": "true"}), ShouldBeTrue)
		}
	})

}

func TestFindAllNodes(t *testing.T) {
	Convey("FindAllNodes should return the all nodes of all requested types", t, func() {

		node := CreateNode("div", atom.Div,
			CreateNode("p", atom.P),
			CreateNode("div", atom.Div, CreateNode("p", atom.P)),
			CreateNode("span", atom.Span))

		result := FindAllNodes(node, atom.Span, atom.P)
		So(len(result), ShouldEqual, 3)
	})

	Convey("FindAllNodes should return nil if there is no child node of the requested type", t, func() {

		node := CreateNode("div", atom.Div,
			CreateNode("p", atom.P))

		result := FindAllNodes(node, atom.Span, atom.Div)
		So(result, ShouldBeNil)
	})
}

func TestGetText(t *testing.T) {
	Convey("GetText should return the text content of the node", t, func() {

		node := CreateNode("div", atom.Div,
			"\n",
			CreateNode("p", atom.P, "hello "),
			CreateNode("div", atom.Div, CreateNode("p", atom.P, "world")),
			CreateNode("span", atom.Span, "!"))

		result := GetText(node)
		So(result, ShouldEqual, "hello world!")
	})
}
