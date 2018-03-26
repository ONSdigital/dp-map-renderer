package renderer

import (
	"bytes"
	"fmt"
	"math"
	"sort"

	"strings"

	g2s "github.com/ONSdigital/dp-map-renderer/geojson2svg"
	"github.com/ONSdigital/dp-map-renderer/htmlutil"
	"github.com/ONSdigital/dp-map-renderer/models"
	"github.com/paulmach/go.geojson"
)

// RegionClassName is the name of the class assigned to all map regions (denoted by features in the input topology)
const RegionClassName = "mapRegion"

// MissingDataText is the text appended to the title of a region that has missing data
const MissingDataText = "data unavailable"

// MissingDataPattern is the fmt template used to generate the pattern used for regions with missing data
const MissingDataPattern = `<pattern id="%s-nodata" width="20" height="20" patternUnits="userSpaceOnUse">
<g fill="#6D6E72">
<polygon points="00 00 02 00 00 02 00 00"></polygon>
<polygon points="04 00 06 00 00 06 00 04"></polygon>
<polygon points="08 00 10 00 00 10 00 08"></polygon>
<polygon points="12 00 14 00 00 14 00 12"></polygon>
<polygon points="16 00 18 00 00 18 00 16"></polygon>
<polygon points="20 00 20 02 02 20 00 20"></polygon>
<polygon points="20 04 20 06 06 20 04 20"></polygon>
<polygon points="20 08 20 10 10 20 08 20"></polygon>
<polygon points="20 12 20 14 14 20 12 20"></polygon>
<polygon points="20 16 20 18 18 20 16 20"></polygon>
</g>
</pattern>`

var pngConverter g2s.PNGConverter

// UsePNGConverter assigns a PNGConverter that will be used to generate fallback png images for svgs.
func UsePNGConverter(p g2s.PNGConverter) {
	pngConverter = p
}

// valueAndColour represents a choropleth data point, which has both a numeric value and an associated colour
type valueAndColour struct {
	value  float64
	colour string
}

// SVGRequest wraps a models.RenderRequest and allows caching of expensive calculations (such as converting topojson to geojson)
type SVGRequest struct {
	request                     *models.RenderRequest
	geoJSON                     *geojson.FeatureCollection
	svg                         *g2s.SVG
	ViewBoxWidth, ViewBoxHeight float64
	breaks                      []*breakInfo // sorted breaks
	referencePos                float64      // the relative position of the reference tick
	VerticalLegendWidth         float64      // the view box width of the vertical legend
	verticalKeyOffset           float64      // offset for the position of the key. // I.e. the middle of the key should be positioned in the middle of the legend, plus the offset.
}

// PrepareSVGRequest wraps the request in an SVGRequest, caching expensive calculations up front
func PrepareSVGRequest(request *models.RenderRequest) *SVGRequest {
	geoJSON := getGeoJSON(request)

	svg := g2s.New()
	svg.AppendFeatureCollection(geoJSON)

	width, height := 0.0, 0.0
	if geoJSON != nil {
		width, height = getViewBoxDimensions(svg)
	}

	svgRequest := &SVGRequest{
		request:       request,
		geoJSON:       geoJSON,
		svg:           svg,
		ViewBoxWidth:  width,
		ViewBoxHeight: height,
	}

	if request.Choropleth != nil && len(request.Choropleth.Breaks) > 0 {
		svgRequest.breaks, svgRequest.referencePos = getSortedBreakInfo(request)

		svgRequest.VerticalLegendWidth, svgRequest.verticalKeyOffset = getVerticalLegendWidth(request, svgRequest.breaks)
	}

	return svgRequest
}

// RenderSVG generates an SVG map for the given request
func RenderSVG(svgRequest *SVGRequest) string {

	geoJSON := svgRequest.geoJSON
	if geoJSON == nil {
		return ""
	}
	request := svgRequest.request
	vbWidth := svgRequest.ViewBoxWidth
	vbHeight := svgRequest.ViewBoxHeight

	idPrefix := request.Filename + "-"
	setFeatureIDs(geoJSON.Features, request.Geography.IDProperty, idPrefix)
	setClassProperty(geoJSON.Features, RegionClassName)
	setChoroplethColoursAndTitles(geoJSON.Features, request, idPrefix)

	converter := pngConverter
	if !request.IncludeFallbackPng {
		converter = nil
	}

	missingDataPattern := strings.Replace(fmt.Sprintf(MissingDataPattern, request.Filename), "\n", "", -1)

	return svgRequest.svg.DrawWithProjection(vbWidth, vbHeight, g2s.MercatorProjection,
		g2s.UseProperties([]string{"style", "class"}),
		g2s.WithTitles(request.Geography.NameProperty),
		g2s.WithAttribute("id", mapID(request)+"-svg"),
		g2s.WithAttribute("style", "width=100%;"), // an explicit width is necessary for the pan-and-zoom js to work
		g2s.WithAttribute("viewBox", fmt.Sprintf("0 0 %.f %.f", vbWidth, vbHeight)),
		g2s.WithPNGFallback(converter),
		g2s.WithPattern(missingDataPattern))
}

// getGeoJSON performs a sanity check for missing properties, then converts the topojson to geojson
func getGeoJSON(request *models.RenderRequest) *geojson.FeatureCollection {
	// sanity check
	if request.Geography == nil ||
		request.Geography.Topojson == nil ||
		len(request.Geography.Topojson.Arcs) == 0 ||
		len(request.Geography.Topojson.Objects) == 0 {
		return nil
	}

	return request.Geography.Topojson.ToGeoJSON()
}

// getViewBoxDimensions assigns the viewbox a fixed width (400) and calculates the height relative to this,
// returning (width, height)
func getViewBoxDimensions(svg *g2s.SVG) (float64, float64) {
	width := 400.0
	height := svg.GetHeightForWidth(width, g2s.MercatorProjection)
	return width, height
}

// setFeatureIDs looks in each Feature for a property with the given idProperty, using it as the feature id.
func setFeatureIDs(features []*geojson.Feature, idProperty string, idPrefix string) {
	for _, feature := range features {
		id, isString := feature.Properties[idProperty].(string)
		if isString && len(id) > 0 {
			feature.ID = idPrefix + id
		} else {
			id, isString := feature.ID.(string)
			if isString && len(id) > 0 {
				feature.ID = idPrefix + id
			}
		}
	}
}

// setClassProperty populates a class property in each feature with the given class name, appending any existing class property.
func setClassProperty(features []*geojson.Feature, className string) {
	for _, feature := range features {
		appendProperty(feature, "class", className)
	}
}

// appendProperty sets a property by the given name, appending any existing value
// (appending existing value rather than the new value so that, in the case of style, we can ensure there's a semi-colon between values)
func appendProperty(feature *geojson.Feature, propertyName string, value string) {
	s := value
	if original, exists := feature.Properties[propertyName]; exists {
		s = fmt.Sprintf("%s %v", value, original)
	}
	feature.Properties[propertyName] = s
}

// setChoroplethColoursAndTitles creates a mapping from the id of a data row to its value and colour,
// then iterates through the features assigning a title and style for the colour.
func setChoroplethColoursAndTitles(features []*geojson.Feature, request *models.RenderRequest, idPrefix string) {
	choropleth := request.Choropleth
	if choropleth == nil || request.Data == nil {
		return
	}
	dataMap := mapDataToColour(request.Data, choropleth, idPrefix)
	missingValueStyle := "fill: url(#" + request.Filename + "-nodata);"
	for _, feature := range features {
		style := missingValueStyle
		title, ok := feature.Properties[request.Geography.NameProperty]
		if !ok {
			title = ""
		}
		if vc, exists := dataMap[feature.ID]; exists {
			style = "fill: " + vc.colour + ";"
			title = fmt.Sprintf("%v %s%g%s", title, choropleth.ValuePrefix, vc.value, choropleth.ValueSuffix)
		} else {
			title = fmt.Sprintf("%v %s", title, MissingDataText)
		}
		feature.Properties[request.Geography.NameProperty] = title
		appendProperty(feature, "style", style)
	}
}

// mapDataToColour creates a map of DataRow.ID=valueAndColour
func mapDataToColour(data []*models.DataRow, choropleth *models.Choropleth, idPrefix string) map[interface{}]valueAndColour {
	breaks := sortBreaks(choropleth.Breaks, false)

	dataMap := make(map[interface{}]valueAndColour)
	for _, row := range data {
		dataMap[idPrefix+row.ID] = valueAndColour{value: row.Value, colour: getColour(row.Value, breaks)}
	}
	return dataMap
}

// getColour returns the colour for the given value. If the value is below the lowest lowerbound, returns the colour for the lowest.
func getColour(value float64, breaks []*models.ChoroplethBreak) string {
	for _, b := range breaks {
		if value >= b.LowerBound {
			return b.Colour
		}
	}
	return breaks[len(breaks)-1].Colour
}

// sortBreaks returns a copy of the breaks slice, sorted ascending or descending according to asc.
func sortBreaks(breaks []*models.ChoroplethBreak, asc bool) []*models.ChoroplethBreak {
	c := make([]*models.ChoroplethBreak, len(breaks))
	copy(c, breaks)
	sort.Slice(c, func(i, j int) bool {
		if asc {
			return c[i].LowerBound < c[j].LowerBound
		}
		return c[i].LowerBound > c[j].LowerBound
	})
	return c
}

// RenderHorizontalKey creates an SVG containing a horizontally-oriented key for the choropleth
func RenderHorizontalKey(svgRequest *SVGRequest) string {

	geoJSON := svgRequest.geoJSON
	if geoJSON == nil {
		return ""
	}
	request := svgRequest.request

	keyInfo := getHorizontalKeyInfo(svgRequest.ViewBoxWidth, svgRequest)

	content := bytes.NewBufferString("")
	ticks := bytes.NewBufferString("")
	keyClass := getKeyClass(request, "horizontal")
	svgAttributes := fmt.Sprintf(`id="%s-legend-horizontal-svg" class="%s" viewBox="0 0 %.f 90"`, request.Filename, keyClass, svgRequest.ViewBoxWidth)

	fmt.Fprintf(content, `<g id="%s-legend-horizontal-container">`, request.Filename)
	writeHorizontalKeyTitle(request, svgRequest.ViewBoxWidth, content)
	fmt.Fprintf(content, `<g id="%s-legend-horizontal-key" transform="translate(%f, 20)">`, request.Filename, keyInfo.keyX)
	left := 0.0
	breaks := svgRequest.breaks
	for i := 0; i < len(breaks); i++ {
		width := breaks[i].RelativeSize * keyInfo.keyWidth
		fmt.Fprintf(content, `<rect class="keyColour" height="8" width="%f" x="%f" style="stroke-width: 0.5; stroke: black; fill: %s;">`, width, left, breaks[i].Colour)
		content.WriteString(`</rect>`)
		writeHorizontalKeyTick(ticks, left, breaks[i].LowerBound)
		left += width
	}
	writeHorizontalKeyTick(ticks, left, breaks[len(breaks)-1].UpperBound)
	if len(request.Choropleth.ReferenceValueText) > 0 {
		writeHorizontalKeyRefTick(ticks, keyInfo, svgRequest)
	}
	fmt.Fprint(content, ticks.String())

	writeKeyMissingPattern(content, request.Filename, 0.0, 55.0)

	content.WriteString(`</g></g>`)

	if pngConverter == nil || request.IncludeFallbackPng == false {
		return fmt.Sprintf("<svg %s>%s</svg>", svgAttributes, content)
	}
	return pngConverter.IncludeFallbackImage(svgAttributes, content.String())
}

// RenderVerticalKey creates an SVG containing a vertically-oriented key for the choropleth
func RenderVerticalKey(svgRequest *SVGRequest) string {

	geoJSON := svgRequest.geoJSON
	if geoJSON == nil {
		return ""
	}
	request := svgRequest.request
	svgHeight := svgRequest.ViewBoxHeight

	breaks := svgRequest.breaks

	keyHeight := svgHeight * 0.8
	keyWidth, offset := svgRequest.VerticalLegendWidth, svgRequest.verticalKeyOffset

	content := bytes.NewBufferString("")
	ticks := bytes.NewBufferString("")
	keyClass := getKeyClass(request, "vertical")
	attributes := fmt.Sprintf(`id="%s-legend-vertical-svg" class="%s" viewBox="0 0 %.f %.f"`, request.Filename, keyClass, keyWidth, svgHeight)

	fmt.Fprintf(content, `<g id="%s-legend-vertical-container">`, request.Filename)
	fmt.Fprintf(content, `<text x="%f" y="%f" dy=".5em" style="text-anchor: middle;" class="keyText">%s %s</text>`, keyWidth/2, svgHeight*0.05, request.Choropleth.ValuePrefix, request.Choropleth.ValueSuffix)
	fmt.Fprintf(content, `<g id="%s-legend-vertical-key" transform="translate(%f, %f)">`, request.Filename, (keyWidth+offset)/2, svgHeight*0.1)
	position := 0.0
	for i := 0; i < len(breaks); i++ {
		height := breaks[i].RelativeSize * keyHeight
		adjustedPosition := keyHeight - position
		fmt.Fprintf(content, `<rect class="keyColour" height="%f" width="8" y="%f" style="stroke-width: 0.5; stroke: black; fill: %s;">`, height, adjustedPosition-height, breaks[i].Colour)
		content.WriteString(`</rect>`)
		writeVerticalKeyTick(ticks, adjustedPosition, breaks[i].LowerBound)
		position += height
	}
	writeVerticalKeyTick(ticks, keyHeight-position, breaks[len(breaks)-1].UpperBound)
	if len(request.Choropleth.ReferenceValueText) > 0 {
		writeVerticalKeyRefTick(ticks, keyHeight-(keyHeight*svgRequest.referencePos), request.Choropleth.ReferenceValueText, request.Choropleth.ReferenceValue)
	}
	fmt.Fprint(content, ticks.String())
	content.WriteString(`</g>`)

	xPos := (keyWidth - float64(htmlutil.GetApproximateTextWidth(MissingDataText, request.FontSize)+12)) / 2
	writeKeyMissingPattern(content, request.Filename, xPos, svgHeight*0.95)

	content.WriteString(`</g>`)

	if pngConverter == nil || request.IncludeFallbackPng == false {
		return fmt.Sprintf("<svg %s>%s</svg>", attributes, content)
	}
	return pngConverter.IncludeFallbackImage(attributes, content.String())
}

// getKeyClass returns the class of the map key - with an additional class if both keys are rendered.
func getKeyClass(request *models.RenderRequest, keyType string) string {
	keyClass := "map_key_" + keyType
	if hasVerticalLegend(request) && hasHorizontalLegend(request) {
		keyClass = keyClass + " " + keyClass + "_both"
	}
	return keyClass
}

// hasVerticalLegend returns true if the request includes a vertical legend
func hasVerticalLegend(request *models.RenderRequest) bool {
	return request.Choropleth.VerticalLegendPosition == models.LegendPositionBefore ||
		request.Choropleth.VerticalLegendPosition == models.LegendPositionAfter
}

// hasHorizontalLegend returns true if the request includes a horizontal legend
func hasHorizontalLegend(request *models.RenderRequest) bool {
	return request.Choropleth.HorizontalLegendPosition == models.LegendPositionBefore ||
		request.Choropleth.HorizontalLegendPosition == models.LegendPositionAfter
}

// getVerticalLegendWidth determines the approximate width required for the legend
// it also returns an offset for the position of the key. I.e. the middle of the key should be positioned in the middle of the legend, plus the offset.
func getVerticalLegendWidth(request *models.RenderRequest, breaks []*breakInfo) (float64, float64) {
	missingWidth := htmlutil.GetApproximateTextWidth(MissingDataText, request.FontSize) + 12
	titleWidth := htmlutil.GetApproximateTextWidth(request.Choropleth.ValuePrefix+" "+request.Choropleth.ValueSuffix, request.FontSize)
	maxWidth := math.Max(float64(missingWidth), float64(titleWidth))
	keyWidth, offset := getVerticalTickTextWidth(request, breaks)
	return math.Max(maxWidth, keyWidth) + 10, offset
}

// getVerticalTickTextWidth calculates the approximate total width of the ticks on both sides of the key, allowing 38 pixels for the colour bar
// it also returns an offset for the position of the key. I.e. the middle of the key should be positioned in the middle of the legend, plus the offset.
func getVerticalTickTextWidth(request *models.RenderRequest, breaks []*breakInfo) (float64, float64) {
	maxTick := 0.0
	for _, b := range breaks {
		lbound := htmlutil.GetApproximateTextWidth(fmt.Sprintf("%g", b.LowerBound), request.FontSize)
		if lbound > maxTick {
			maxTick = lbound
		}
		ubound := htmlutil.GetApproximateTextWidth(fmt.Sprintf("%g", b.UpperBound), request.FontSize)
		if ubound > maxTick {
			maxTick = ubound
		}
	}
	refTick := htmlutil.GetApproximateTextWidth(request.Choropleth.ReferenceValueText, request.FontSize)
	refValue := htmlutil.GetApproximateTextWidth(fmt.Sprintf("%g", request.Choropleth.ReferenceValue), request.FontSize)
	refWidth := math.Max(refTick, refValue)
	return maxTick + refWidth + 38.0, maxTick - refWidth
}

// writeHorizontalKeyTitle write the title above the key for a horizontal legend, ensuring that the text fits within the svg
func writeHorizontalKeyTitle(request *models.RenderRequest, svgWidth float64, content *bytes.Buffer) {
	textAdjust := ""
	titleText := request.Choropleth.ValuePrefix + " " + request.Choropleth.ValueSuffix
	titleTextLen := htmlutil.GetApproximateTextWidth(titleText, request.FontSize)
	if titleTextLen >= svgWidth {
		textAdjust = fmt.Sprintf(` textLength="%.f" lengthAdjust="spacingAndGlyphs"`, svgWidth-2)
	}
	fmt.Fprintf(content, `<text x="%f" y="6" dy=".5em" style="text-anchor: middle;" class="keyText"%s>%s</text>`, svgWidth/2.0, textAdjust, titleText)
}

// writeHorizontalKeyTick draws a vertical line (the tick) at the given position, labelling it with the given value
func writeHorizontalKeyTick(w *bytes.Buffer, xPos float64, value float64) {
	fmt.Fprintf(w, `<g class="map__tick" transform="translate(%f, 0)">`, xPos)
	w.WriteString(`<line x2="0" y2="15" style="stroke-width: 1; stroke: Black;"></line>`)
	fmt.Fprintf(w, `<text x="0" y="18" dy=".74em" style="text-anchor: middle;" class="keyText">%g</text>`, value)
	w.WriteString(`</g>`)
}

// writeVerticalKeyTick draws a horizontal line (the tick) at the given position, labelling it with the given value
func writeVerticalKeyTick(w *bytes.Buffer, yPos float64, value float64) {
	fmt.Fprintf(w, `<g class="map__tick" transform="translate(0, %f)">`, yPos)
	w.WriteString(`<line x1="8" x2="-15" style="stroke-width: 1; stroke: Black;"></line>`)
	fmt.Fprintf(w, `<text x="-18" y="0" dy="0.32em" style="text-anchor: end;" class="keyText">%g</text>`, value)
	w.WriteString(`</g>`)
}

// writeHorizontalKeyRefTick draws a vertical line at the correct position for the reference value, labelling it with the reference value and reference text.
func writeHorizontalKeyRefTick(w *bytes.Buffer, keyInfo *horizontalKeyInfo, svgRequest *SVGRequest) {
	xPos := keyInfo.keyWidth * svgRequest.referencePos
	svgWidth := svgRequest.ViewBoxWidth
	fmt.Fprintf(w, `<g class="map__tick" transform="translate(%f, 0)">`, xPos)
	w.WriteString(`<line x2="0" y1="8" y2="45" style="stroke-width: 1; stroke: DimGrey;"></line>`)
	textAttr := ""
	if keyInfo.referenceTextLeftLen > xPos+keyInfo.keyX { // adjust the text length so it will fit
		textAttr = fmt.Sprintf(` textLength="%.f" lengthAdjust="spacingAndGlyphs"`, xPos+keyInfo.keyX-1)
	}
	fmt.Fprintf(w, `<text x="0" y="33" dx="-0.1em" dy=".74em" style="text-anchor: end; fill: DimGrey;" class="keyText"%s>%s</text>`, textAttr, keyInfo.referenceTextLeft)
	textAttr = ""
	if keyInfo.referenceTextRightLen > svgWidth-(xPos+keyInfo.keyX) { // adjust the text length so it will fit
		textAttr = fmt.Sprintf(` textLength="%.f" lengthAdjust="spacingAndGlyphs"`, svgWidth-(xPos+keyInfo.keyX)-2)
	}
	fmt.Fprintf(w, `<text x="0" y="33" dx="0.1em" dy=".74em" style="text-anchor: start; fill: DimGrey;" class="keyText"%s>%s</text>`, textAttr, keyInfo.referenceTextRight)
	fmt.Fprintf(w, `</g>`)
}

// writeVerticalKeyRefTick draws a horizontal line at the correct position for the reference value, labelling it with the reference value and reference text.
func writeVerticalKeyRefTick(w *bytes.Buffer, yPos float64, text string, value float64) {
	fmt.Fprintf(w, `<g class="map__tick" transform="translate(0, %f)">`, yPos)
	w.WriteString(`<line x2="45" x1="8" style="stroke-width: 1; stroke: DimGrey;"></line>`)
	fmt.Fprintf(w, `<text x="18" dy="-.32em" style="text-anchor: start; fill: DimGrey;" class="keyText">%s</text>`, text)
	fmt.Fprintf(w, `<text x="18" dy="1em" style="text-anchor: start; fill: DimGrey;" class="keyText">%g</text>`, value)
	w.WriteString(`</g>`)
}

// writeKeyMissingPattern draws a square filled with the missing pattern at the given position, labelling it with MissingDataText
func writeKeyMissingPattern(w *bytes.Buffer, filename string, xPos float64, yPos float64) {
	fmt.Fprintf(w, `<g class="missingPattern" transform="translate(%f, %f)">`, xPos, yPos)
	fmt.Fprintf(w, `<rect class="keyColour" height="8" width="8" style="stroke-width: 0.8; stroke: black; fill: url(#%s-nodata);"></rect>`, filename)
	fmt.Fprintf(w, `<text x="12" dy=".55em" style="text-anchor: start; fill: DimGrey;" class="keyText">%s</text>`, MissingDataText)
	w.WriteString(`</g>`)
}

// breakInfo contains information about the breaks (the boundaries between colours)- lowerBound, upperBound and relative size
type breakInfo struct {
	LowerBound   float64
	UpperBound   float64
	RelativeSize float64
	Colour       string
}

// getSortedBreakInfo returns information about the breaks - lowerBound, upperBound and relative size
// where the lowerBound of the first break is the lowest of the LowerBound and the lowest value in data
// and the upperBound of the last break is the maximum value in the data
// also returns the relative position of the reference value
func getSortedBreakInfo(request *models.RenderRequest) ([]*breakInfo, float64) {

	data := make([]*models.DataRow, len(request.Data))
	copy(data, request.Data)
	sort.Slice(data, func(i, j int) bool { return data[i].Value < data[j].Value })

	breaks := sortBreaks(request.Choropleth.Breaks, true)
	minValue := math.Min(data[0].Value, breaks[0].LowerBound)
	maxValue := request.Choropleth.UpperBound
	if maxValue < breaks[len(breaks)-1].LowerBound {
		maxValue = data[len(data)-1].Value
	}
	totalRange := maxValue - minValue

	breakCount := len(breaks)
	info := make([]*breakInfo, breakCount)
	for i := 0; i < breakCount-1; i++ {
		info[i] = &breakInfo{LowerBound: breaks[i].LowerBound, UpperBound: breaks[i+1].LowerBound, Colour: breaks[i].Colour}
	}
	info[0].LowerBound = minValue
	info[breakCount-1] = &breakInfo{LowerBound: breaks[breakCount-1].LowerBound, UpperBound: maxValue, Colour: breaks[breakCount-1].Colour}
	for _, b := range info {
		b.RelativeSize = (b.UpperBound - b.LowerBound) / totalRange
	}
	referencePos := (request.Choropleth.ReferenceValue - minValue) / totalRange
	return info, referencePos
}

// horizontalKeyInfo contains break info, the width of the key, the x position of the key, and reference tick values
type horizontalKeyInfo struct {
	referenceTextLeft     string
	referenceTextLeftLen  float64
	referenceTextRight    string
	referenceTextRightLen float64
	keyWidth              float64
	keyX                  float64
}

// getHorizontalKeyInfo returns the width of the key, the x position of the key, the breaks within the key, and reference tick values
// (making sure that the longer of the reference value and text is given the most space)
func getHorizontalKeyInfo(svgWidth float64, svgRequest *SVGRequest) *horizontalKeyInfo {
	request := svgRequest.request
	refInfo := getHorizontalRefTextInfo(request)
	info := horizontalKeyInfo{}

	// assume a default width of 90% of svg
	info.keyWidth = svgWidth * 0.9
	info.keyX = (svgWidth - info.keyWidth) / 2

	// half of the upper and lower bound text will sit outside the key
	breaks := svgRequest.breaks
	left := htmlutil.GetApproximateTextWidth(fmt.Sprintf("%g", breaks[0].LowerBound), request.FontSize) / 2
	right := htmlutil.GetApproximateTextWidth(fmt.Sprintf("%g", breaks[len(breaks)-1].UpperBound), request.FontSize) / 2

	// the longer bit of reference text should sit on the side of the tick with the most space
	info.referenceTextLeft = refInfo.referenceTextLong
	info.referenceTextLeftLen = refInfo.referenceTextLongLen
	info.referenceTextRight = refInfo.referenceTextShort
	info.referenceTextRightLen = refInfo.referenceTextShortLen
	if svgRequest.referencePos < 0.5 { // the reference tick is less than halfway - switch the text
		info.referenceTextRight = refInfo.referenceTextLong
		info.referenceTextRightLen = refInfo.referenceTextLongLen
		info.referenceTextLeft = refInfo.referenceTextShort
		info.referenceTextLeftLen = refInfo.referenceTextShortLen
	}
	// now see if reference text is long enough to go beyond the bounds of the key
	refPos := info.keyWidth * svgRequest.referencePos // the actual pixel position of the reference tick within the key
	if refPos-info.referenceTextLeftLen < 0.0-left {
		left = math.Abs(refPos - info.referenceTextLeftLen)
	}
	if (refPos+info.referenceTextRightLen)-info.keyWidth > right {
		right = (refPos + info.referenceTextRightLen) - info.keyWidth
	}
	// if any text goes beyond the bounds of the svg, shorten the key
	if info.keyWidth+left+right > svgWidth {
		info.keyWidth = svgWidth - (left + right)
		info.keyX = left
	}

	return &info
}

// horizontalRefTextInfo contains the reference value and label with information about their length
type horizontalRefTextInfo struct {
	referenceTextShort    string
	referenceTextShortLen float64
	referenceTextLong     string
	referenceTextLongLen  float64
}

// getHorizontalRefTextInfo calculates the approximate width of the reference value and text, dividing them into short and long values.
func getHorizontalRefTextInfo(request *models.RenderRequest) *horizontalRefTextInfo {
	info := horizontalRefTextInfo{}
	refTextLen := htmlutil.GetApproximateTextWidth(request.Choropleth.ReferenceValueText, request.FontSize)
	refValue := fmt.Sprintf("%g", request.Choropleth.ReferenceValue)
	refValueLen := htmlutil.GetApproximateTextWidth(refValue, request.FontSize)
	if refTextLen > refValueLen {
		info.referenceTextLong = request.Choropleth.ReferenceValueText
		info.referenceTextLongLen = refTextLen
		info.referenceTextShort = refValue
		info.referenceTextShortLen = refValueLen
	} else {
		info.referenceTextLong = refValue
		info.referenceTextLongLen = refValueLen
		info.referenceTextShort = request.Choropleth.ReferenceValueText
		info.referenceTextShortLen = refTextLen
	}
	return &info
}
