package renderer

import (
	"github.com/ONSdigital/dp-map-renderer/models"
	"github.com/ONSdigital/dp-map-renderer/geojson2svg"
	"github.com/paulmach/go.geojson"
)

// RenderSVG generates an SVG map for the given request
func RenderSVG(request *models.RenderRequest) string {
	geoJSON := request.Geography.Topojson.ToGeoJSON()

	setFeatureIDs(geoJSON.Features, "AREACD", request.Filename + "_")

	svg := geojson2svg.New()
	svg.AppendFeatureCollection(geoJSON)

	width, height := getWidthAndHeight(request, svg)
	return svg.DrawWithProjection(width, height, geojson2svg.MercatorProjection)
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
