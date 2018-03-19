// Package geojson2svg provides the SVG type to convert geojson
// geometries, features and featurecollections into a SVG image.
//
// See the tests for usage examples.
package geojson2svg

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"

	"github.com/paulmach/go.geojson"
	"time"
	"github.com/ONSdigital/dp-map-renderer/health"
)

// ElementType represents the elements that may be represented in an SVG
type ElementType int

// The possible ElementTypes in an SVG
const (
	Geometry          ElementType = iota
	Feature           ElementType = iota
	FeatureCollection ElementType = iota
)

// ScaleFunc accepts x,y coordinates and transforms them, returning a new pair of x,y coordinates.
type ScaleFunc func(float64, float64) (float64, float64)

// SVG represents the SVG that should be created.
// Use the New function to create a SVG. New will handle the default values.
//
// default padding (top: 0, right: 0, bottom: 0, left: 0)
//
// default properties (class)
//
// default attributes ()
type SVG struct {
	useProp      func(string) bool
	padding      Padding
	attributes   map[string]string
	elements     []*SVGElement
	titleProp    string
	pngConverter PNGConverter
	bounds       *boundingRectangle
}

// SVGElement represents a single element of an SVG - a Geometry, Feature or FeatureCollection
type SVGElement struct {
	geometry          *geojson.Geometry
	feature           *geojson.Feature
	featureCollection *geojson.FeatureCollection
	elementType       ElementType
}

// Padding represents the possible padding of the SVG.
type Padding struct{ Top, Right, Bottom, Left float64 }

// An Option represents a single SVG option.
type Option func(*SVG)

// PNGConverter converts an svg file to png. Call either Convert or IncludeFallbackImage - there's no need to call both.
type PNGConverter interface {
	// Convert converts the given svg file to a base64-encoded png
	Convert(svg []byte) ([]byte, error)
	// IncludeFallbackImage generates an svg with the given attributes, content and a fallback image:
	// <svg svgAttributes><switch><g>svgContent</g><foreignObject><image src="data:image/png;base64,..." /></foreignObject></svg>
	IncludeFallbackImage(svgAttributes string, svgContent string) string
}

// boundingRectangle is used to cache the result of calculations in getBoundingRectangle
type boundingRectangle struct {
	minX, minY, maxX, maxY float64
}

// New returns a new SVG that can be used to to draw geojson geometries,
// features and featurecollections.
func New() *SVG {
	return &SVG{
		useProp:    func(prop string) bool { return prop == "class" },
		titleProp:  "",
		attributes: make(map[string]string),
	}
}

// Draw renders the final SVG with the given options to a string.
// All coordinates will be scaled to fit into the svg.
func (svg *SVG) Draw(width, height float64, opts ...Option) string {
	return svg.DrawWithProjection(width, height, func(x, y float64) (float64, float64) { return x, y }, opts...)
}

// DrawWithProjection renders the final SVG with the given options to a string.
// All coordinates will be converted by the given projection, then scaled to fit into the svg.
func (svg *SVG) DrawWithProjection(width, height float64, projection ScaleFunc, opts ...Option) string {
	defer health.TrackTime(time.Now(), "geojson2svg.DrawWithProjection")

	for _, o := range opts {
		o(svg)
	}

	sf := svg.makeScaleFunc(width, height, svg.padding, svg.points(), projection)

	content := bytes.NewBufferString("")
	for _, e := range svg.elements {
		switch e.elementType {
		case Geometry:
			process(sf, content, e.geometry, "", "")
		case Feature:
			as, title := getFeatureAttributesAndTitle(svg.useProp, svg.titleProp, e.feature)
			process(sf, content, e.feature.Geometry, as, title)
		case FeatureCollection:
			for _, f := range e.featureCollection.Features {
				as, title := getFeatureAttributesAndTitle(svg.useProp, svg.titleProp, f)
				process(sf, content, f.Geometry, as, title)
			}
		}
	}

	attributes := fmt.Sprintf(`width="%g" height="%g"%s`, width, height, makeAttributes(svg.attributes))

	if svg.pngConverter == nil {
		return fmt.Sprintf(`<svg %s>%s</svg>`, attributes, content)
	}
	return svg.pngConverter.IncludeFallbackImage(attributes, content.String())
}

// AppendGeometry adds a geojson Geometry to the svg.
func (svg *SVG) AppendGeometry(g *geojson.Geometry) {
	svg.elements = append(svg.elements, &SVGElement{geometry: g, elementType: Geometry})
	svg.clearCache()
}

// AppendFeature adds a geojson Feature to the svg.
func (svg *SVG) AppendFeature(f *geojson.Feature) {
	svg.elements = append(svg.elements, &SVGElement{feature: f, elementType: Feature})
	svg.clearCache()
}

// AppendFeatureCollection adds a geojson FeatureCollection to the svg.
func (svg *SVG) AppendFeatureCollection(fc *geojson.FeatureCollection) {
	svg.elements = append(svg.elements, &SVGElement{featureCollection: fc, elementType: FeatureCollection})
	svg.clearCache()
}

// clearCache deletes all internal cached values
func (svg *SVG) clearCache() {
	svg.bounds = nil
}

// WithAttribute adds the key value pair as attribute to the
// resulting SVG root element.
func WithAttribute(k, v string) Option {
	return func(svg *SVG) {
		svg.attributes[k] = v
	}
}

// WithAttributes adds the map of key value pairs as attributes to the
// resulting SVG root element.
func WithAttributes(as map[string]string) Option {
	return func(svg *SVG) {
		for k, v := range as {
			svg.attributes[k] = v
		}
	}
}

// WithPadding configures the SVG to use the specified padding.
func WithPadding(p Padding) Option {
	return func(svg *SVG) {
		svg.padding = p
	}
}

// WithTitles configures the SVG to include a title element for each feature with the given property.
func WithTitles(titleProperty string) Option {
	return func(svg *SVG) {
		svg.titleProp = titleProperty
	}
}

// WithPNGFallback configures the SVG to include a png image as a foreignObject fallback for browsers that don't support svg
func WithPNGFallback(converter PNGConverter) Option {
	return func(svg *SVG) {
		svg.pngConverter = converter
	}
}

// UseProperties configures which geojson properties should be copied to the
// resulting SVG element.
func UseProperties(props []string) Option {
	return func(svg *SVG) {
		svg.useProp = func(prop string) bool {
			for _, p := range props {
				if p == prop {
					return true
				}
			}
			return false
		}
	}
}

// points returns an array of all coordinates (points) in the svg. Note that these points have not had any projection applied.
func (svg *SVG) points() [][]float64 {
	ps := [][]float64{}
	for _, e := range svg.elements {
		switch e.elementType {
		case Geometry:
			ps = append(ps, collect(e.geometry)...)
		case Feature:
			ps = append(ps, collect(e.feature.Geometry)...)
		case FeatureCollection:
			for _, f := range e.featureCollection.Features {
				ps = append(ps, collect(f.Geometry)...)
			}
		}
	}
	return ps
}

// process draws the given geometry to the svg canvas (the writer)
func process(sf ScaleFunc, w io.Writer, g *geojson.Geometry, attributes string, title string) {
	defer health.RecordTime(time.Now(), "geojson2svg.process")
	switch {
	case g.IsPoint():
		drawPoint(sf, w, g.Point, attributes, title)
	case g.IsMultiPoint():
		drawMultiPoint(sf, w, g.MultiPoint, attributes, title)
	case g.IsLineString():
		drawLineString(sf, w, g.LineString, attributes, title)
	case g.IsMultiLineString():
		drawMultiLineString(sf, w, g.MultiLineString, attributes, title)
	case g.IsPolygon():
		drawPolygon(sf, w, g.Polygon, attributes, title)
	case g.IsMultiPolygon():
		drawMultiPolygon(sf, w, g.MultiPolygon, attributes, title)
	case g.IsCollection():
		fmt.Fprintf(w, `<g%s>`, attributes)
		if len(title) > 0 {
			fmt.Fprintf(w, `<title>%s</title>`, title)
		}
		for _, x := range g.Geometries {
			process(sf, w, x, "", "")
		}
		w.Write([]byte(`</g>`))
	}
}

// collect collects all points in the given geometry into a slice of []float64
func collect(g *geojson.Geometry) (ps [][]float64) {
	switch {
	case g.IsPoint():
		ps = append(ps, g.Point)
	case g.IsMultiPoint():
		ps = append(ps, g.MultiPoint...)
	case g.IsLineString():
		ps = append(ps, g.LineString...)
	case g.IsMultiLineString():
		for _, x := range g.MultiLineString {
			ps = append(ps, x...)
		}
	case g.IsPolygon():
		for _, x := range g.Polygon {
			ps = append(ps, x...)
		}
	case g.IsMultiPolygon():
		for _, xs := range g.MultiPolygon {
			for _, x := range xs {
				ps = append(ps, x...)
			}
		}
	case g.IsCollection():
		for _, g := range g.Geometries {
			ps = append(ps, collect(g)...)
		}
	}
	return ps
}

func drawPoint(sf ScaleFunc, w io.Writer, p []float64, attributes string, title string) {
	defer health.RecordTime(time.Now(), "geojson2svg.drawPoint")
	x, y := sf(p[0], p[1])
	endTag := endTag("circle", title)
	fmt.Fprintf(w, `<circle cx="%f" cy="%f" r="1"%s%s`, x, y, attributes, endTag)
}

func drawMultiPoint(sf ScaleFunc, w io.Writer, ps [][]float64, attributes string, title string) {
	defer health.RecordTime(time.Now(), "geojson2svg.drawMultiPoint")
	fmt.Fprintf(w, `<g%s>`, attributes)
	if len(title) > 0 {
		fmt.Fprintf(w, `<title>%s</title>`, title)
	}
	for _, p := range ps {
		drawPoint(sf, w, p, "", "")
	}
	w.Write([]byte(`</g>`))
}

func drawLineString(sf ScaleFunc, w io.Writer, ps [][]float64, attributes string, title string) {
	defer health.RecordTime(time.Now(), "geojson2svg.drawLineString")
	path := bytes.NewBufferString("M")
	for _, p := range ps {
		x, y := sf(p[0], p[1])
		fmt.Fprintf(path, "%f %f,", x, y)
	}
	endTag := endTag("path", title)
	fmt.Fprintf(w, `<path d="%s"%s%s`, trim(path), attributes, endTag)
}

func drawMultiLineString(sf ScaleFunc, w io.Writer, pps [][][]float64, attributes string, title string) {
	defer health.RecordTime(time.Now(), "geojson2svg.drawMultiLineString")
	fmt.Fprintf(w, `<g%s>`, attributes)
	if len(title) > 0 {
		fmt.Fprintf(w, `<title>%s</title>`, title)
	}
	for _, ps := range pps {
		drawLineString(sf, w, ps, "", "")
	}
	w.Write([]byte(`</g>`))
}

var singleSpace = []byte(" ")

func drawPolygon(sf ScaleFunc, w io.Writer, pps [][][]float64, attributes string, title string) {
	defer health.RecordTime(time.Now(), "geojson2svg.drawPolygon")
	path := bytes.NewBufferString("")
	for i, ps := range pps {
		subPath := bytes.NewBufferString("M")
		for _, p := range ps {
			x, y := sf(p[0], p[1])
			fmt.Fprintf(subPath, "%f %f,", x, y)
		}
		if i > 0 {
			path.Write(singleSpace)
		}
		path.Write([]byte(trim(subPath)))
	}
	//fmt.Fprintf(w, `<path d="%s Z"%s%s`, trim(path), attributes, endTag("path", title))
	w.Write([]byte(`<path d="` + trim(path) + ` Z"` + attributes + endTag("path", title)))
}

func drawMultiPolygon(sf ScaleFunc, w io.Writer, ppps [][][][]float64, attributes string, title string) {
	defer health.RecordTime(time.Now(), "geojson2svg.drawMultiPolygon")
	fmt.Fprintf(w, `<g%s>`, attributes)
	if len(title) > 0 {
		fmt.Fprintf(w, `<title>%s</title>`, title)
	}
	for _, pps := range ppps {
		drawPolygon(sf, w, pps, "", "")
	}
	w.Write([]byte(`</g>`))
}

func trim(s *bytes.Buffer) string {
	defer health.RecordTime(time.Now(), "geojson2svg.trim")
	return strings.Trim(s.String(), " ,")
}

// endTag creates an end tag string, "/>" if title is empty, "><title>title</title></tag>" otherwise.
func endTag(tag string, title string) string {
	defer health.RecordTime(time.Now(), "geojson2svg.endTag")
	if len(title) > 0 {
		return fmt.Sprintf("><title>%s</title></%s>", title, tag)
	}
	return "/>"
}

// getFeatureAttributesAndTitle converts the properties of the feature into a string of attributes, and extracts the title property into a string
func getFeatureAttributesAndTitle(useProp func(string) bool, titleProp string, feature *geojson.Feature) (string, string) {
	defer health.RecordTime(time.Now(), "geojson2svg.getFeatureAttributesAndTitle")
	attrs := make(map[string]string)
	id, isString := feature.ID.(string)
	if isString && len(id) > 0 {
		attrs["id"] = id
	}
	for k, v := range feature.Properties {
		if useProp(k) {
			attrs[k] = fmt.Sprintf("%v", v)
		}
	}
	titleString := ""
	if title, ok := feature.Properties[titleProp]; ok {
		titleString = fmt.Sprintf("%v", title)
	}
	return makeAttributes(attrs), titleString
}

// makeAttributes converts the given map into a string with each key="value" pair in sorted order
func makeAttributes(as map[string]string) string {
	defer health.RecordTime(time.Now(), "geojson2svg.makeAttributes")
	keys := make([]string, 0, len(as))
	for k := range as {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	res := bytes.NewBufferString("")
	for _, k := range keys {
		fmt.Fprintf(res, ` %s="%s"`, k, as[k])
	}
	return res.String()
}

// makeScaleFunc creates a function that will scale a pair of coordinates so that they fit within the width and height,
// passing them through the projection first.
func (svg *SVG) makeScaleFunc(width, height float64, padding Padding, ps [][]float64, projection ScaleFunc) ScaleFunc {
	w := width - padding.Left - padding.Right
	h := height - padding.Top - padding.Bottom

	if len(ps) == 0 {
		return func(x, y float64) (float64, float64) { return projection(x, y) }
	}

	if len(ps) == 1 {
		return func(x, y float64) (float64, float64) { return w / 2, h / 2 }
	}

	minX, minY, maxX, maxY := svg.getBoundingRectangle(projection, ps)
	xRes := (maxX - minX) / w
	yRes := (maxY - minY) / h
	res := math.Max(xRes, yRes)

	return func(x, y float64) (float64, float64) {
		x, y = projection(x, y)
		return (x-minX)/res + padding.Left, (maxY-y)/res + padding.Top
	}

}

// getBoundingRectangle calculates (and caches) the minX, minY, maxX, maxY coordinates of the svg
func (svg *SVG) getBoundingRectangle(projection ScaleFunc, ps [][]float64) (float64, float64, float64, float64) {
	if svg.bounds == nil {
		svg.bounds = calcBoundingRectangle(projection, ps)
	}
	return svg.bounds.minX, svg.bounds.minY, svg.bounds.maxX, svg.bounds.maxY
}

// calcBoundingRectangle calculates the minX, minY, maxX, maxY coordinates of the svg.
func calcBoundingRectangle(projection ScaleFunc, ps [][]float64) *boundingRectangle {
	defer health.RecordTime(time.Now(), "geojson2svg.calcBoundingRectangle")
	if len(ps) == 0 || len(ps[0]) == 0 {
		return &boundingRectangle{}
	}
	minX, minY := projection(ps[0][0], ps[0][1])
	maxX, maxY := projection(ps[0][0], ps[0][1])
	for _, p := range ps[1:] {
		x, y := projection(p[0], p[1])
		minX = math.Min(minX, x)
		maxX = math.Max(maxX, x)
		minY = math.Min(minY, y)
		maxY = math.Max(maxY, y)
	}
	return &boundingRectangle{minX, minY, maxX, maxY}
}

// GetHeightForWidth returns an appropriate height given a desired width.
func (svg *SVG) GetHeightForWidth(width float64, projection ScaleFunc) float64 {
	minX, minY, maxX, maxY := svg.getBoundingRectangle(projection, svg.points())
	svgWidth := maxX - minX
	svgHeight := maxY - minY
	ratio := svgHeight / svgWidth
	return math.Floor((width * ratio) + .5)

}

// MercatorProjection is a projection function that will convert latitude & logitude into x,y coordinates for a Mercator map.
var MercatorProjection = func(longitude, latitude float64) (float64, float64) {
	defer health.RecordTime(time.Now(), "geojson2svg.MercatorProjection")
	// https://stackoverflow.com/questions/38270132/topojson-d3-map-with-longitude-latitude
	mapWidth, mapHeight := 100.0, 100.0
	// get x value
	x := (longitude + 180) * (mapWidth / 360)

	// convert from degrees to radians
	latRad := latitude * math.Pi / 180

	// get y value
	mercN := math.Log(math.Tan((math.Pi / 4) + (latRad / 2)))
	y := (mapHeight / 2) - (mapHeight * mercN / (2 * math.Pi))
	// invert the y-axis to put the map the right way up
	return x, mapHeight - y
}

// areaOfPolygon returns the signed area of the polygon described by the path
func areaOfPolygon(sf ScaleFunc, path [][]float64) float64 {
	defer health.RecordTime(time.Now(), "geojson2svg.areaOfPolygon")
	s := 0.0

	for i := 0; i < len(path)-1; i++ {
		i0, i1 := sf(path[i][0], path[i][1])
		j0, j1 := sf(path[i+1][0], path[i+1][1])
		s += float64(i0*j1 - j0*i1)
	}

	return 0.5 * s
}

// Centroid calculates the centroid of the exterior ring of a polygon using
// the formula at http://en.wikipedia.org/wiki/Centroid#Centroid_of_polygon
// but be careful as this applies Euclidean principles to decidedly non-
// Euclidean geometry. In other words, it will fail miserably for polygons
// near the poles, polygons that straddle the dateline, and for large polygons
// where Euclidean approximations break down.
// adapted from https://github.com/kpawlik/geojson/issues/3
func Centroid(sf ScaleFunc, poly [][][]float64) []float64 {
	defer health.RecordTime(time.Now(), "geojson2svg.Centroid")

	// find the path describing the largest polygon by area
	var ring [][]float64
	area := 0.0
	for _, path := range poly {
		pathArea := areaOfPolygon(sf, path)
		if pathArea >= area {
			area = pathArea
			ring = path
		}
	}

	c := []float64{0, 0}
	for i := 0; i < len(ring)-1; i++ {
		i0, i1 := sf(ring[i][0], ring[i][1])
		j0, j1 := sf(ring[i+1][0], ring[i+1][1])
		c[0] += (i0 + j0) * (i0*j1 - j0*i1)
		c[1] += (i1 + j1) * (i0*j1 - j0*i1)
	}

	c[0] /= area * 6
	c[1] /= area * 6
	return c
}
