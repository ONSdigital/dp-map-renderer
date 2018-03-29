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
	"math"
)

// Placeholders are inserted into the html to be replaced with the svg map, legends, css and javascript
const (
	svgReplacementText           = "[SVG Here]"
	verticalKeyReplacementText   = "[Vertical key Here]"
	horizontalKeyReplacementText = "[Horizontal key Here]"
	cssReplacementText           = "[CSS Here]"
)

var (
	newLine      = regexp.MustCompile(`\n`)
	footnoteLink = regexp.MustCompile(`\[[0-9]+]`)

	widthPattern  = regexp.MustCompile(`width="[^"]*"`)
	heightPattern = regexp.MustCompile(`height="[^"]+"`)

	// text that will need internationalising at some point:
	sourceText         = "Source: "
	notesText          = "Notes"
	footnoteHiddenText = "Footnote "
)

// RenderHTMLWithSVG returns an HTML figure element with caption and footer, and an SVG version of the map and (optional) legend
func RenderHTMLWithSVG(request *models.RenderRequest) ([]byte, error) {
	s := renderHTML(request)
	result := renderSVGs(request, s)
	return []byte(result), nil
}

// RenderHTMLWithPNG returns an HTML figure element with caption and footer, and a PNG version of the map and (optional) legend
func RenderHTMLWithPNG(request *models.RenderRequest) ([]byte, error) {
	request.IncludeFallbackPng = false
	s := renderHTML(request)
	result := renderPNGs(request, s)
	return []byte(result), nil
}

// renderHTML returns an HTML figure element with caption and footer, and divs with placeholder text for the map and legend
func renderHTML(request *models.RenderRequest) string {
	figure := createFigure(request)
	svgContainer := h.CreateNode("div", atom.Div, h.Attr("class", "map_container"))
	figure.AppendChild(svgContainer)
	addCssPlaceholder(request, svgContainer)
	addSVGDivs(request, svgContainer)
	addFooter(request, figure)
	var buf bytes.Buffer
	html.Render(&buf, figure)
	buf.WriteString("\n")
	return buf.String()
}

// createFigure creates a figure element and adds a caption with the title and subtitle
func createFigure(request *models.RenderRequest) *html.Node {
	figure := h.CreateNode("figure", atom.Figure,
		h.Attr("class", "figure"),
		h.Attr("id", idPrefix(request) + "-figure"),
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

// idPrefix returns the prefix that should be used for all ids
func idPrefix(request *models.RenderRequest) string {
	return "map-" + request.Filename
}

// mapID returns the id for the map, as used in links etc
func mapID(request *models.RenderRequest) string {
	return idPrefix(request) + "-map"
}

// addSVGDivs adds divs with marker text for each of the horizontal & vertical legends, and the map
func addSVGDivs(request *models.RenderRequest, parent *html.Node) {
	if request.Choropleth == nil {
		return
	}

	prefix := idPrefix(request)

	if request.Choropleth.HorizontalLegendPosition == models.LegendPositionBefore {
		parent.AppendChild(h.CreateNode("div", atom.Div,
			h.Attr("id", prefix+"-legend-horizontal"),
			h.Attr("class", "map_key map_key__horizontal"),
			horizontalKeyReplacementText))
	}
	if request.Choropleth.VerticalLegendPosition == models.LegendPositionBefore {
		parent.AppendChild(h.CreateNode("div", atom.Div,
			h.Attr("id", prefix+"-legend-vertical"),
			h.Attr("class", "map_key map_key__vertical"),
			verticalKeyReplacementText))
	}

	parent.AppendChild(h.CreateNode("div", atom.Div,
		h.Attr("id", mapID(request)),
		h.Attr("class", "map"),
		svgReplacementText))

	if request.Choropleth.VerticalLegendPosition == models.LegendPositionAfter {
		parent.AppendChild(h.CreateNode("div", atom.Div,
			h.Attr("id", prefix+"-legend-vertical"),
			h.Attr("class", "map_key map_key__vertical"),
			verticalKeyReplacementText))
	}
	if request.Choropleth.HorizontalLegendPosition == models.LegendPositionAfter {
		parent.AppendChild(h.CreateNode("div", atom.Div,
			h.Attr("id", prefix+"-legend-horizontal"),
			h.Attr("class", "map_key map_key__horizontal"),
			horizontalKeyReplacementText))
	}

}

// addFooter adds a footer to the given element, containing the source and footnotes
func addFooter(request *models.RenderRequest, parent *html.Node) {
	footer := h.CreateNode("footer", atom.Footer,
		h.Attr("class", "figure__footer"),
		"\n")
	if len(request.Licence) > 0 {
		footer.AppendChild(h.CreateNode("p", atom.P,
			h.Attr("class", "figure__licence"),
			request.Licence))
		footer.AppendChild(h.Text("\n"))
	}
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
			h.Attr("id", fmt.Sprintf("%s-note-%d", idPrefix(request), i+1)),
			h.Attr("class", "figure__footnote-item"),
			parseValue(request, note))
		ol.AppendChild(li)
		ol.AppendChild(h.Text("\n"))
	}
}

// addCssPlaceholder adds a text node that should be replaced with style enabling the map to responsively adjust its size.
func addCssPlaceholder(request *models.RenderRequest, parent *html.Node) {
	parent.AppendChild(h.Text(cssReplacementText))
}

// renderSVGs replaces the SVG marker text with the actual SVG(s)
func renderSVGs(request *models.RenderRequest, original string) string {
	svgRequest := PrepareSVGRequest(request)
	result := strings.Replace(original, svgReplacementText, "\n" + RenderSVG(svgRequest) + "\n", 1)
	if strings.Contains(result, verticalKeyReplacementText) {
		result = strings.Replace(result, verticalKeyReplacementText, "\n" + RenderVerticalKey(svgRequest) + "\n", 1)
	}
	if strings.Contains(result, horizontalKeyReplacementText) {
		result = strings.Replace(result, horizontalKeyReplacementText, "\n" + RenderHorizontalKey(svgRequest) + "\n", 1)
	}
	result = strings.Replace(result, cssReplacementText, renderCss(svgRequest), 1)
	return result
}

// renderCss creates a <script> block that has styles specific to this svg that allow it to be responsive and
// switch between the horizontal and vertical legends according to window width
func renderCss(svgRequest *SVGRequest) string {
	id := idPrefix(svgRequest.request)
	css := bytes.NewBufferString("\n<style type=\"text/css\">")
	if svgRequest.responsiveSize {
		// min/max width for svg
		fmt.Fprintf(css, "\n\t#%s-map, #%s-legend-horizontal {", id, id)
		fmt.Fprintf(css, "\n\t\tmin-width: %.0fpx;", svgRequest.request.MinWidth)
		fmt.Fprintf(css, "\n\t\tmax-width: %.0fpx;", svgRequest.request.MaxWidth)
		fmt.Fprintf(css, "\n\t}")
	} else {
		// fixed width for svg
		fmt.Fprintf(css, "\n\t#%s-map, #%s-legend-horizontal {", id, id)
		fmt.Fprintf(css, "\n\t\twidth: %.0fpx;", svgRequest.ViewBoxWidth)
		fmt.Fprintf(css, "\n\t}")
	}

	if hasVerticalLegend(svgRequest.request) {
		// relative width of svg and vertical legend
		svgWidthPercent := math.Floor(svgRequest.ViewBoxWidth / (svgRequest.ViewBoxWidth + svgRequest.VerticalLegendWidth) * 100.0)
		vlWidthPercent := 100.0 - svgWidthPercent - 1
		vlMaxWidth := (math.Max(svgRequest.request.MaxWidth, svgRequest.ViewBoxWidth) / svgWidthPercent) * vlWidthPercent

		if hasHorizontalLegend(svgRequest.request) && svgRequest.responsiveSize {
			// switch between both legends
			switchPoint := svgRequest.ViewBoxWidth + svgRequest.VerticalLegendWidth

			fmt.Fprintf(css, "\n\t@media (min-width: %.0fpx) {", switchPoint + 1.0)
			fmt.Fprintf(css, "\n\t\t#%s-legend-horizontal { display: none;}", id)
			fmt.Fprintf(css, "\n\t\t#%s-map { display: inline-block; width: %.0f%%;}", id, svgWidthPercent)
			fmt.Fprintf(css, "\n\t\t#%s-legend-vertical { display: inline-block; width: %.0f%%; max-width: %.0fpx;}", id, vlWidthPercent, vlMaxWidth)
			fmt.Fprintf(css, "\n\t}")

			fmt.Fprintf(css, "\n\t@media (max-width: %.0fpx) {", switchPoint)
			fmt.Fprintf(css, "\n\t\t#%s-legend-vertical { display: none;}", id)
			fmt.Fprintf(css, "\n\t\t#%s-map { width: 100%%;}", id)
			fmt.Fprintf(css, "\n\t}")

		} else {
			// vertical legend only
			fmt.Fprintf(css, "\n\t#%s-map { display: inline-block; width: %.0f%%;}", id, svgWidthPercent)
			fmt.Fprintf(css, "\n\t#%s-legend-vertical { display: inline-block; width: %.0f%%; max-width: %.0fpx;}", id, vlWidthPercent, vlMaxWidth)
		}
	}

	fmt.Fprintf(css, "\n</style>\n")
	return css.String()
}

// renderPNGs replaces the SVG marker text with png images. It will not return a responsive design, and will ensure that only one of the legends is included.
func renderPNGs(request *models.RenderRequest, original string) string {
	svgRequest := PrepareSVGRequest(request)
	svgRequest.responsiveSize = false

	svg := RenderSVG(svgRequest)
	result := strings.Replace(original, svgReplacementText, renderPNG(svg), 1)
	if strings.Contains(result, verticalKeyReplacementText) {
		key := RenderVerticalKey(svgRequest)
		result = strings.Replace(result, verticalKeyReplacementText, renderPNG(key), 1)
	}
	if strings.Contains(result, horizontalKeyReplacementText) {
		// only render horizontal if we won't have vertical
		if hasVerticalLegend(request) {
			result = strings.Replace(result, horizontalKeyReplacementText, "", 1)
		} else {
			key := RenderHorizontalKey(svgRequest)
			result = strings.Replace(result, horizontalKeyReplacementText, renderPNG(key), 1)
		}
	}
	result = strings.Replace(result, cssReplacementText, "", 1)
	return result
}

// renderPNG converts the given svg to a png, retaining the width and height attributes
func renderPNG(svg string) string {
	if pngConverter == nil {
		log.Error(fmt.Errorf("pngConverter is nil - cannot convert svg to png"), nil)
		return svg
	}
	png := svg
	b64, err := pngConverter.Convert([]byte(svg))
	if err == nil {
		width := widthPattern.FindString(svg)
		height := heightPattern.FindString(svg)
		png = fmt.Sprintf(`<img %s %s src="data:image/png;base64,%s" />`, width, height, string(b64))
	} else {
		log.Error(err, log.Data{"_message": "Unable to convert svg to png"})
	}
	return png
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
			linkText := fmt.Sprintf("<a href=\"#%s-note-%d\" class=\"footnote__link\"><span class=\"visuallyhidden\">%s</span>%d</a>", idPrefix(request), n, footnoteHiddenText, n)
			value = strings.Replace(value, fmt.Sprintf("[%d]", n), linkText, -1)
		}
	}
	nodes, err := html.ParseFragment(strings.NewReader(value), &html.Node{
		Type:     html.ElementNode,
		Data:     "body",
		DataAtom: atom.Body,
	})
	if err != nil {
		log.Error(err, log.Data{"replaceValues": "Unable to parse value!", "value": original})
		return []*html.Node{{Type: html.TextNode, Data: original}}
	}
	return nodes
}
