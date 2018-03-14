package renderer

import (
	"bytes"
	"fmt"
	"math"
	"sort"

	"regexp"

	g2s "github.com/ONSdigital/dp-map-renderer/geojson2svg"
	"github.com/ONSdigital/dp-map-renderer/models"
	"github.com/paulmach/go.geojson"
)

// RegionClassName is the name of the class assigned to all map regions (denoted by features in the input topology)
const RegionClassName = "mapRegion"

// MissingDataText is the text appended to the title of a region that has missing data
const MissingDataText = "data unavailable"

var pngConverter g2s.PNGConverter

// UsePNGConverter assigns a PNGConverter that will be used to generate fallback png images for svgs.
func UsePNGConverter(p g2s.PNGConverter) {
	pngConverter = p
}

type valueAndColour struct {
	value  float64
	colour string
}

// RenderSVG generates an SVG map for the given request
func RenderSVG(request *models.RenderRequest) string {

	geoJSON := getGeoJSON(request)
	if geoJSON == nil {
		return ""
	}

	idPrefix := request.Filename + "-"
	setFeatureIDs(geoJSON.Features, request.Geography.IDProperty, idPrefix)
	setClassProperty(geoJSON.Features, RegionClassName)
	setChoroplethColoursAndTitles(geoJSON.Features, request, idPrefix)

	svg := g2s.New()
	svg.AppendFeatureCollection(geoJSON)

	width, height, vbHeight := getWidthAndHeight(request, svg)
	return svg.DrawWithProjection(width, height, g2s.MercatorProjection,
		g2s.UseProperties([]string{"style", "class"}),
		g2s.WithTitles(request.Geography.NameProperty),
		g2s.WithAttribute("viewBox", fmt.Sprintf("0 0 %g %g", width, vbHeight)),
		g2s.WithPNGFallback(pngConverter))
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

// getWidthAndHeight extracts width and height from the request,
// defaulting the width if missing and determining the height proportionally to the width if missing
// note that this means that the request should only specify height if it has specified width, otherwise the dimensions will be wrong.
// The third response argument is the height that should be used for the viewBox - this may be different to the height specified in the request.
func getWidthAndHeight(request *models.RenderRequest, svg *g2s.SVG) (float64, float64, float64) {
	width := request.Width
	if width <= 0 {
		width = 400.0
	}
	viewBoxHeight := svg.GetHeightForWidth(width, g2s.MercatorProjection)
	height := request.Height
	if height <= 0 {
		height = viewBoxHeight
	}
	return width, height, viewBoxHeight
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

func setChoroplethColoursAndTitles(features []*geojson.Feature, request *models.RenderRequest, idPrefix string) {
	choropleth := request.Choropleth
	if choropleth == nil || request.Data == nil {
		return
	}
	dataMap := mapDataToColour(request.Data, choropleth, idPrefix)
	missingValueStyle := ""
	if len(choropleth.MissingValueColor) > 0 {
		missingValueStyle = "fill: " + choropleth.MissingValueColor + ";"
	}
	for _, feature := range features {
		style := missingValueStyle
		title := feature.Properties[request.Geography.NameProperty]
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
func RenderHorizontalKey(request *models.RenderRequest) string {

	geoJSON := getGeoJSON(request)
	if geoJSON == nil {
		return ""
	}
	svg := g2s.New()
	svg.AppendFeatureCollection(geoJSON)
	svgWidth, _, _ := getWidthAndHeight(request, svg)

	breaks, referencePos := getSortedBreakInfo(request)

	keyWidth := svgWidth * 0.9
	content := bytes.NewBufferString("")
	ticks := bytes.NewBufferString("")
	attributes := fmt.Sprintf(`id="%s-legend-horizontal" class="map_key_horizontal" width="%g" height="90" viewBox="0 0 %g 90"`, request.Filename, svgWidth, svgWidth)

	fmt.Fprintf(content, `%s<g id="%s-legend-horizontal-container">`, "\n", request.Filename)
	fmt.Fprintf(content, `%s<text x="%f" y="6" dy=".5em" style="text-anchor: middle;" class="keyText">%s %s</text>`, "\n", keyWidth/2.0, request.Choropleth.ValuePrefix, request.Choropleth.ValueSuffix)
	fmt.Fprintf(content, `%s<g id="%s-legend-horizontal-key" transform="translate(%f, 20)">`, "\n", request.Filename, svgWidth*0.05)
	left := 0.0
	for i := 0; i < len(breaks); i++ {
		width := breaks[i].RelativeSize * keyWidth
		fmt.Fprintf(content, `%s<rect class="keyColour" height="8" width="%f" x="%f" style="stroke-width: 0.5; stroke: black; fill: %s;">`, "\n", width, left, breaks[i].Colour)
		fmt.Fprintf(content, `</rect>`)
		writeKeyTick(ticks, left, breaks[i].LowerBound)
		left += width
	}
	writeKeyTick(ticks, left, breaks[len(breaks)-1].UpperBound)
	if len(request.Choropleth.ReferenceValueText) > 0 {
		writeKeyReferenceTick(ticks, keyWidth*referencePos, request.Choropleth.ReferenceValueText, request.Choropleth.ReferenceValue)
	}
	fmt.Fprint(content, ticks.String())
	if len(request.Choropleth.MissingValueColor) > 0 {
		writeKeyMissingColour(content, request.Choropleth.MissingValueColor, 0.0, 55.0)
	}
	fmt.Fprintf(content, `%s</g>%s</g>%s`, "\n", "\n", "\n")

	if pngConverter == nil {
		return fmt.Sprintf("<svg %s>%s</svg>", attributes, content)
	}
	return pngConverter.IncludeFallbackImage(attributes, content.String())
}

// RenderVerticalKey creates an SVG containing a vertically-oriented key for the choropleth
func RenderVerticalKey(request *models.RenderRequest) string {

	geoJSON := getGeoJSON(request)
	if geoJSON == nil {
		return ""
	}
	svg := g2s.New()
	svg.AppendFeatureCollection(geoJSON)
	_, svgHeight, _ := getWidthAndHeight(request, svg)

	breaks, referencePos := getSortedBreakInfo(request)

	keyHeight := svgHeight * 0.8
	keyWidth := getVerticalKeyWidth(request, breaks)
	content := bytes.NewBufferString("")
	ticks := bytes.NewBufferString("")
	attributes := fmt.Sprintf(`id="%s-legend-vertical" class="map_key_vertical" height="%g" width="%g" viewBox="0 0 %g %g"`, request.Filename, svgHeight, keyWidth, keyWidth, svgHeight)

	fmt.Fprintf(content, `%s<g id="%s-legend-vertical-container">`, "\n", request.Filename)
	fmt.Fprintf(content, `%s<text x="%f" y="%f" dy=".5em" style="text-anchor: middle;" class="keyText">%s %s</text>`, "\n", keyWidth/2, svgHeight*0.05, request.Choropleth.ValuePrefix, request.Choropleth.ValueSuffix)
	fmt.Fprintf(content, `%s<g id="%s-legend-vertical-key" transform="translate(%f, %f)">`, "\n", request.Filename, keyWidth/2, svgHeight*0.1)
	position := 0.0
	for i := 0; i < len(breaks); i++ {
		height := breaks[i].RelativeSize * keyHeight
		adjustedPosition := keyHeight - position
		fmt.Fprintf(content, `%s<rect class="keyColour" height="%f" width="8" y="%f" style="stroke-width: 0.5; stroke: black; fill: %s;">`, "\n", height, adjustedPosition-height, breaks[i].Colour)
		fmt.Fprintf(content, `</rect>`)
		writeVerticalKeyTick(ticks, adjustedPosition, breaks[i].LowerBound)
		position += height
	}
	writeVerticalKeyTick(ticks, keyHeight-position, breaks[len(breaks)-1].UpperBound)
	if len(request.Choropleth.ReferenceValueText) > 0 {
		writeVerticalKeyReferenceTick(ticks, keyHeight-(keyHeight*referencePos), request.Choropleth.ReferenceValueText, request.Choropleth.ReferenceValue)
	}
	fmt.Fprint(content, ticks.String())
	fmt.Fprintf(content, `%s</g>`, "\n")
	if len(request.Choropleth.MissingValueColor) > 0 {
		xPos := (keyWidth - float64(getTextWidth(MissingDataText, 12))) / 2
		writeKeyMissingColour(content, request.Choropleth.MissingValueColor, xPos, svgHeight*0.95)
	}
	fmt.Fprintf(content, `%s</g>%s`, "\n", "\n")

	if pngConverter == nil {
		return fmt.Sprintf("<svg %s>%s</svg>", attributes, content)
	}
	return pngConverter.IncludeFallbackImage(attributes, content.String())
}

// getVerticalKeyWidth determines the approximate width required for the key
func getVerticalKeyWidth(request *models.RenderRequest, breaks []*breakInfo) float64 {
	missingWidth := getTextWidth(MissingDataText, 12)
	titleWidth := getTextWidth(request.Choropleth.ValuePrefix+" "+request.Choropleth.ValueSuffix, 0)
	maxWidth := math.Max(float64(missingWidth), float64(titleWidth))
	return math.Max(maxWidth, getTickTextWidth(request, breaks)) + 10
}

// getTickTextWidth calculates the approximate total width of the ticks on both sides of the key, allowing 36 pixels for the colour bar
func getTickTextWidth(request *models.RenderRequest, breaks []*breakInfo) float64 {
	maxTick := 0
	for _, b := range breaks {
		lbound := getTextWidth(fmt.Sprintf("%g", b.LowerBound), 0)
		if lbound > maxTick {
			maxTick = lbound
		}
		ubound := getTextWidth(fmt.Sprintf("%g", b.UpperBound), 0)
		if ubound > maxTick {
			maxTick = ubound
		}
	}
	refTick := getTextWidth(request.Choropleth.ReferenceValueText, 0)
	refValue := getTextWidth(fmt.Sprintf("%g", request.Choropleth.ReferenceValue), 0)
	return float64(maxTick) + math.Max(float64(refTick), float64(refValue)) + 36
}

var narrowChars = regexp.MustCompile(`[1ijlt;:,.'!]`)
var medChars = regexp.MustCompile(`[abcdefghknoprsuvxyz ]`)

// getTextWidth determines the (approximate) width required to display the given text
// this method gives a very rough approximation. To do better would involve generating an image and measuring its width - requiring an external tool such as inkscape or image magick
func getTextWidth(text string, margin int) int {
	const emWidth = 11
	const medWidth = 9
	const narrowWidth = 4
	length := len(text)
	narrow := length - len(narrowChars.ReplaceAllLiteralString(text, ""))
	med := length - len(medChars.ReplaceAllLiteralString(text, ""))
	wide := length - narrow - med
	return wide*emWidth + narrow*narrowWidth + med*medWidth + margin
}

func writeKeyTick(w *bytes.Buffer, xPos float64, value float64) {
	fmt.Fprintf(w, `%s<g class="tick" transform="translate(%f, 0)">`, "\n", xPos)
	fmt.Fprintf(w, `%s<line x2="0" y2="15" style="stroke-width: 1; stroke: Black;"></line>`, "\n")
	fmt.Fprintf(w, `%s<text x="0" y="18" dy=".74em" style="text-anchor: middle;" class="keyText">%g</text>`, "\n", value)
	fmt.Fprintf(w, `%s</g>`, "\n")
}

func writeVerticalKeyTick(w *bytes.Buffer, yPos float64, value float64) {
	fmt.Fprintf(w, `%s<g class="tick" transform="translate(0, %f)">`, "\n", yPos)
	fmt.Fprintf(w, `%s<line x1="8" x2="-15" style="stroke-width: 1; stroke: Black;"></line>`, "\n")
	fmt.Fprintf(w, `%s<text x="-18" y="0" dy="0.32em" style="text-anchor: end;" class="keyText">%g</text>`, "\n", value)
	fmt.Fprintf(w, `%s</g>`, "\n")
}

func writeKeyReferenceTick(w *bytes.Buffer, xPos float64, text string, value float64) {
	fmt.Fprintf(w, `%s<g class="tick" transform="translate(%f, 0)">`, "\n", xPos)
	fmt.Fprintf(w, `%s<line x2="0" y1="8" y2="45" style="stroke-width: 1; stroke: DimGrey;"></line>`, "\n")
	fmt.Fprintf(w, `%s<text x="0" y="33" dx="-0.1em" dy=".74em" style="text-anchor: end; fill: DimGrey;" class="keyText">%s</text>`, "\n", text)
	fmt.Fprintf(w, `<text x="0" y="33" dx="0.1em" dy=".74em" style="text-anchor: start; fill: DimGrey;" class="keyText">%g</text>`, value)
	fmt.Fprintf(w, `%s</g>`, "\n")
}

func writeVerticalKeyReferenceTick(w *bytes.Buffer, yPos float64, text string, value float64) {
	fmt.Fprintf(w, `%s<g class="tick" transform="translate(0, %f)">`, "\n", yPos)
	fmt.Fprintf(w, `%s<line x2="45" x1="8" style="stroke-width: 1; stroke: DimGrey;"></line>`, "\n")
	fmt.Fprintf(w, `%s<text x="18" dy="-.32em" style="text-anchor: start; fill: DimGrey;" class="keyText">%s</text>`, "\n", text)
	fmt.Fprintf(w, `<text x="18" dy="1em" style="text-anchor: start; fill: DimGrey;" class="keyText">%g</text>`, value)
	fmt.Fprintf(w, `%s</g>`, "\n")
}

func writeKeyMissingColour(w *bytes.Buffer, colour string, xPos float64, yPos float64) {
	fmt.Fprintf(w, `%s<g class="missingColour" transform="translate(%f, %f)">`, "\n", xPos, yPos)
	fmt.Fprintf(w, `%s<rect class="keyColour" height="8" width="8" style="stroke-width: 0.8; stroke: black; fill: %s;"></rect>`, "\n", colour)
	fmt.Fprintf(w, `%s<text x="12" dy=".55em" style="text-anchor: start; fill: DimGrey;" class="keyText">%s</text>`, "\n", MissingDataText)
	fmt.Fprintf(w, `%s</g>`, "\n")
}

type breakInfo struct {
	LowerBound   float64
	UpperBound   float64
	RelativeSize float64
	Colour       string
}

// getRelativeBreakSizes return information about the breaks - lowerBound, upperBound and relative size
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
