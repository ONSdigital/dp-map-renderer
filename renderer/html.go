package renderer

import (
	"bytes"
	"fmt"

	"regexp"

	"strings"

	h "github.com/ONSdigital/dp-map-renderer/htmlutil"
	"github.com/ONSdigital/dp-map-renderer/models"
	"github.com/ONSdigital/go-ns/log"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const SVG_REPLACEMENT_TEXT = "[SVG Here]"
const VERTICAL_KEY_REPLACEMENT_TEXT = "[Vertical key Here]"
const HORIZONTAL_KEY_REPLACEMENT_TEXT = "[Horizontal key Here]"

var (
	newLine      = regexp.MustCompile(`\n`)
	footnoteLink = regexp.MustCompile(`\[[0-9]+]`)

	// text that will need internationalising at some point:
	sourceText         = "Source: "
	notesText          = "Notes"
	footnoteHiddenText = "Footnote "
)

// TODO consider including a fallback base64-encoded png for the few browsers that don't support inline svg (IE8, Opera Mini, Android 2.3)
// best option: http://davidensinger.com/2013/04/inline-svg-with-png-fallback/
// other possibilities: https://css-tricks.com/svg-fallbacks/
//						https://stackoverflow.com/a/28239372

// RenderHTML returns an HTML figure element with caption and footer, and a div with text that should be replaced by the SVG map
func RenderHTML(request *models.RenderRequest) ([]byte, error) {

	figure := createFigure(request)

	svgContainer := h.CreateNode("div", atom.Div, h.Attr("class", "map_container"))
	figure.AppendChild(svgContainer)
	svgContainer.AppendChild(h.CreateNode("div", atom.Div, h.Attr("class", "map"), SVG_REPLACEMENT_TEXT))
	svgContainer.AppendChild(h.CreateNode("div", atom.Div, h.Attr("class", "map_key__vertical"), VERTICAL_KEY_REPLACEMENT_TEXT))
	svgContainer.AppendChild(h.CreateNode("div", atom.Div, h.Attr("class", "map_key__horizontal"), HORIZONTAL_KEY_REPLACEMENT_TEXT))

	addFooter(request, figure)

	var buf bytes.Buffer
	html.Render(&buf, figure)
	buf.WriteString("\n")

	result := strings.Replace(buf.String(), SVG_REPLACEMENT_TEXT, RenderSVG(request), 1)
	result = strings.Replace(result, VERTICAL_KEY_REPLACEMENT_TEXT, RenderVerticalKey(request), 1)
	result = strings.Replace(result, HORIZONTAL_KEY_REPLACEMENT_TEXT, RenderHorizontalKey(request), 1)

	return []byte(result), nil
}

// createFigure creates a figure element and adds a caption with the title and subtitle
func createFigure(request *models.RenderRequest) *html.Node {
	figure := h.CreateNode("figure", atom.Figure,
		h.Attr("class", "figure"),
		h.Attr("id", mapID(request)),
		"\n")
	// add title and subtitle as a caption
	if len(request.Title) > 0 || len(request.Subtitle) > 0 {
		caption := h.CreateNode("figcaption", atom.Figcaption,
			h.Attr("class", "map__caption"),
			parseValue(request, request.Title))
		if len(request.Subtitle) > 0 {
			subtitle := h.CreateNode("span", atom.Span,
				h.Attr("class", "map__subtitle"),
				parseValue(request, request.Subtitle))

			caption.AppendChild(h.CreateNode("br", atom.Br))
			caption.AppendChild(subtitle)

		}
		figure.AppendChild(caption)
		figure.AppendChild(h.Text("\n"))
	}
	return figure
}

// mapID returns the id for the map, as used in links etc
func mapID(request *models.RenderRequest) string {
	return "map-" + request.Filename
}

// addFooter adds a footer to the given element, containing the source and footnotes
func addFooter(request *models.RenderRequest, parent *html.Node) {
	footer := h.CreateNode("footer", atom.Footer,
		h.Attr("class", "figure__footer"),
		"\n")
	if len(request.Source) > 0 {
		var source interface{} = request.Source
		if len(request.SourceLink) > 0 {
			source = h.CreateNode("a", atom.A,
				h.Attr("href", request.SourceLink),
				request.Source)
		}

		footer.AppendChild(h.CreateNode("p", atom.P,
			h.Attr("class", "figure__source"),
			sourceText,
			source))
		footer.AppendChild(h.Text("\n"))
	}
	if len(request.Footnotes) > 0 {
		footer.AppendChild(h.CreateNode("p", atom.P,
			h.Attr("class", "figure__notes"),
			notesText))
		footer.AppendChild(h.Text("\n"))

		ol := h.CreateNode("ol", atom.Ol,
			h.Attr("class", "figure__footnotes"),
			"\n")
		addFooterItemsToList(request, ol)
		footer.AppendChild(ol)
		footer.AppendChild(h.Text("\n"))
	}
	parent.AppendChild(footer)
	parent.AppendChild(h.Text("\n"))
}

// addFooterItemsToList adds one li node for each footnote to the given list node
func addFooterItemsToList(request *models.RenderRequest, ol *html.Node) {
	for i, note := range request.Footnotes {
		li := h.CreateNode("li", atom.Li,
			h.Attr("id", fmt.Sprintf("map-%s-note-%d", request.Filename, i+1)),
			h.Attr("class", "figure__footnote-item"),
			parseValue(request, note))
		ol.AppendChild(li)
		ol.AppendChild(h.Text("\n"))
	}
}

// Parses the string to replace \n with <br /> and wrap [1] with a link to the footnote
func parseValue(request *models.RenderRequest, value string) []*html.Node {
	hasBr := newLine.MatchString(value)
	hasFootnote := len(request.Footnotes) > 0 && footnoteLink.MatchString(value)
	if hasBr || hasFootnote {
		return replaceValues(request, value, hasBr, hasFootnote)
	}
	return []*html.Node{{Type: html.TextNode, Data: value}}
}

// replaceValues uses regexp to replace new lines and footnotes with <br/> and <a>.../<a> tags, then parses the result into an array of nodes
func replaceValues(request *models.RenderRequest, value string, hasBr bool, hasFootnote bool) []*html.Node {
	original := value
	if hasBr {
		value = newLine.ReplaceAllLiteralString(value, "<br />")
	}
	if hasFootnote {
		for i := range request.Footnotes {
			n := i + 1
			linkText := fmt.Sprintf("<a href=\"#map-%s-note-%d\" class=\"footnote__link\"><span class=\"visuallyhidden\">%s</span>%d</a>", request.Filename, n, footnoteHiddenText, n)
			value = strings.Replace(value, fmt.Sprintf("[%d]", n), linkText, -1)
		}
	}
	nodes, err := html.ParseFragment(strings.NewReader(value), &html.Node{
		Type:     html.ElementNode,
		Data:     "body",
		DataAtom: atom.Body,
	})
	if err != nil {
		log.ErrorC(request.Filename, err, log.Data{"replaceValues": "Unable to parse value!", "value": original})
		return []*html.Node{{Type: html.TextNode, Data: original}}
	}
	return nodes
}
