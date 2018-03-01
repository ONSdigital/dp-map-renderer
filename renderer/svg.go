package renderer

import (
	"github.com/ONSdigital/dp-map-renderer/models"
	"github.com/ONSdigital/dp-map-renderer/geojson2svg"
	"github.com/paulmach/go.geojson"
	"fmt"
)

// REGION_CLASS_NAME is the name of the class assigned to all map regions (denoted by features in the input topology)
const REGION_CLASS_NAME = "mapRegion"

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

	setFeatureIDs(geoJSON.Features, request.Geography.IDProperty, request.Filename+"-")
	setClassProperty(geoJSON.Features, REGION_CLASS_NAME)

	svg := geojson2svg.New()
	svg.AppendFeatureCollection(geoJSON)

	width, height := getWidthAndHeight(request, svg)
	return svg.DrawWithProjection(width, height, geojson2svg.MercatorProjection, geojson2svg.WithTitles(request.Geography.NameProperty))
}

// getWidthAndHeight extracts width and height from the request,
// defaulting the width if missing and determining the height proportionally to the width if missing
// note that this means that the request should only specify height if it has specified width, otherwise the dimensions will be wrong.
func getWidthAndHeight(request *models.RenderRequest, svg *geojson2svg.SVG) (float64, float64) {
	width := request.Width
	if width <= 0 {
		width = 400.0;
	}
	height := request.Height
	if height <= 0 {
		height = svg.GetHeightForWidth(width, geojson2svg.MercatorProjection)
	}
	return width, height
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

// setClassProperty populates a class property in each feature with the given class name, appending it to any existing class property.
func setClassProperty(features []*geojson.Feature, className string) {
	for _, feature := range features {
		s := className
		if val, ok := feature.Properties["class"]; ok {
			s = fmt.Sprintf("%v %s", val, className)
		}
		feature.Properties["class"] = s
	}
}
