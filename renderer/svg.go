package renderer

import (
	"github.com/ONSdigital/dp-map-renderer/models"
	g2s "github.com/ONSdigital/dp-map-renderer/geojson2svg"
	"github.com/paulmach/go.geojson"
	"fmt"
	"sort"
)

// REGION_CLASS_NAME is the name of the class assigned to all map regions (denoted by features in the input topology)
const REGION_CLASS_NAME = "mapRegion"
// MISSING_DATA_TEXT is the text appended to the title of a region that has missing data
const MISSING_DATA_TEXT = "(missing data)"

type valueAndColour struct {
	value  float64
	colour string
}

// RenderSVG generates an SVG map for the given request
func RenderSVG(request *models.RenderRequest) string {

	// sanity check
	if request.Geography == nil ||
		request.Geography.Topojson == nil ||
		len(request.Geography.Topojson.Arcs) == 0 ||
		len(request.Geography.Topojson.Objects) == 0 {
		return ""
	}

	geoJSON := request.Geography.Topojson.ToGeoJSON()

	idPrefix := request.Filename + "-"
	setFeatureIDs(geoJSON.Features, request.Geography.IDProperty, idPrefix)
	setClassProperty(geoJSON.Features, REGION_CLASS_NAME)
	setChoroplethColoursAndTitles(geoJSON.Features, request, idPrefix)

	svg := g2s.New()
	svg.AppendFeatureCollection(geoJSON)

	width, height, vbHeight := getWidthAndHeight(request, svg)
	return svg.DrawWithProjection(width, height, g2s.MercatorProjection,
		g2s.UseProperties([]string{"style", "class"}),
		g2s.WithTitles(request.Geography.NameProperty),
		g2s.WithAttribute("viewBox", fmt.Sprintf("0 0 %g %g", width, vbHeight)))
}

// getWidthAndHeight extracts width and height from the request,
// defaulting the width if missing and determining the height proportionally to the width if missing
// note that this means that the request should only specify height if it has specified width, otherwise the dimensions will be wrong.
// The third argument is the height that should be used for the viewBox - this may be different to the height specified in the request.
func getWidthAndHeight(request *models.RenderRequest, svg *g2s.SVG) (float64, float64, float64) {
	width := request.Width
	if width <= 0 {
		width = 400.0;
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
			title = fmt.Sprintf("%v (missing data)", title)
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
		dataMap[idPrefix + row.ID] = valueAndColour{value:row.Value, colour:getColour(row.Value, breaks)}
	}
	return dataMap
}

// getColour returns the colour for the given value. If the value is below the lowest lowerbound, returns the colour for the lowest.
func getColour(value float64, breaks []*models.ChoroplethBreak) string {
	for _, b := range breaks {
		if value >= b.LowerBound {
			return b.Color
		}
	}
	return breaks[len(breaks)-1].Color
}

// sortBreaks returns a copy of the breaks slice, sorted ascending or descending according to asc.
func sortBreaks(breaks []*models.ChoroplethBreak, asc bool) []*models.ChoroplethBreak {
	c := make([]*models.ChoroplethBreak, len(breaks))
	copy(c, breaks)
	sort.Slice(c, func(i, j int) bool {
		if asc {
			return c[i].LowerBound < c[j].LowerBound
		} else {
			return c[i].LowerBound > c[j].LowerBound
		}
	})
	return c;
}
