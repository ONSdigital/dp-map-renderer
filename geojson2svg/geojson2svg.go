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
	"regexp"
	"sort"
	"strings"

	"github.com/paulmach/go.geojson"
)

type ElementType int

const (
	Geometry          ElementType = iota
	Feature           ElementType = iota
	FeatureCollection ElementType = iota
)

const newline = "\n"

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
	useProp    func(string) bool
	padding    Padding
	attributes map[string]string
	elements   []*SVGElement
	titleProp  string
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
	for _, o := range opts {
		o(svg)
	}

	sf := makeScaleFunc(width, height, svg.padding, svg.points(), projection)

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

	attributes := makeAttributes(svg.attributes)
	return fmt.Sprintf(`<svg width="%g" height="%g"%s>%s%s</svg>`, width, height, attributes, content, newline)
}

// AppendGeometry adds a geojson Geometry to the svg.
func (svg *SVG) AppendGeometry(g *geojson.Geometry) {
	svg.elements = append(svg.elements, &SVGElement{geometry: g, elementType: Geometry})
}

// AppendFeature adds a geojson Feature to the svg.
func (svg *SVG) AppendFeature(f *geojson.Feature) {
	svg.elements = append(svg.elements, &SVGElement{feature: f, elementType: Feature})
}

// AppendFeatureCollection adds a geojson FeatureCollection to the svg.
func (svg *SVG) AppendFeatureCollection(fc *geojson.FeatureCollection) {
	svg.elements = append(svg.elements, &SVGElement{featureCollection: fc, elementType: FeatureCollection})
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

func process(sf ScaleFunc, w io.Writer, g *geojson.Geometry, attributes string, title string) {
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
		fmt.Fprintf(w, `%s<g%s>`, newline, attributes)
		if len(title) > 0 {
			fmt.Fprintf(w, `%s<title>%s</title>`, newline, title)
		}
		for _, x := range g.Geometries {
			process(sf, w, x, "", "")
		}
		fmt.Fprintf(w, `%s</g>`, newline)
	}
}

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
	x, y := sf(p[0], p[1])
	endTag := endTag("circle", title)
	fmt.Fprintf(w, `%s<circle cx="%f" cy="%f" r="1"%s%s`, newline, x, y, attributes, endTag)
}

func drawMultiPoint(sf ScaleFunc, w io.Writer, ps [][]float64, attributes string, title string) {
	fmt.Fprintf(w, `%s<g%s>`, newline, attributes)
	if len(title) > 0 {
		fmt.Fprintf(w, `%s<title>%s</title>`, newline, title)
	}
	for _, p := range ps {
		drawPoint(sf, w, p, "", "")
	}
	fmt.Fprintf(w, `%s</g>`, newline)
}

func drawLineString(sf ScaleFunc, w io.Writer, ps [][]float64, attributes string, title string) {
	path := bytes.NewBufferString("M")
	for _, p := range ps {
		x, y := sf(p[0], p[1])
		fmt.Fprintf(path, "%f %f,", x, y)
	}
	endTag := endTag("path", title)
	fmt.Fprintf(w, `%s<path d="%s"%s%s`, newline, trim(path), attributes, endTag)
}

func drawMultiLineString(sf ScaleFunc, w io.Writer, pps [][][]float64, attributes string, title string) {
	fmt.Fprintf(w, `%s<g%s>`, newline, attributes)
	if len(title) > 0 {
		fmt.Fprintf(w, `%s<title>%s</title>`, newline, title)
	}
	for _, ps := range pps {
		drawLineString(sf, w, ps, "", "")
	}
	fmt.Fprintf(w, `%s</g>`, newline)
}

func drawPolygon(sf ScaleFunc, w io.Writer, pps [][][]float64, attributes string, title string) {
	path := bytes.NewBufferString("")
	for _, ps := range pps {
		subPath := bytes.NewBufferString("M")
		for _, p := range ps {
			x, y := sf(p[0], p[1])
			fmt.Fprintf(subPath, "%f %f,", x, y)
		}
		fmt.Fprintf(path, " %s", trim(subPath))
	}
	pathString := trim(path)
	endTag := endTag("path", title)
	fmt.Fprintf(w, `%s<path d="%s Z"%s%s`, newline, pathString, attributes, endTag)
}

func drawMultiPolygon(sf ScaleFunc, w io.Writer, ppps [][][][]float64, attributes string, title string) {
	fmt.Fprintf(w, `%s<g%s>`, newline, attributes)
	if len(title) > 0 {
		fmt.Fprintf(w, `%s<title>%s</title>`, newline, title)
	}
	for _, pps := range ppps {
		drawPolygon(sf, w, pps, "", "")
	}
	fmt.Fprintf(w, `%s</g>`, newline)
}

func trim(s fmt.Stringer) string {
	re := regexp.MustCompile(",$")
	return string(re.ReplaceAll([]byte(strings.TrimSpace(s.String())), []byte("")))
}

// endTag creates an end tag string, "/>" if title is empty, "><title>title</title></tag>" otherwise.
func endTag(tag string, title string) string {
	if len(title) > 0 {
		return fmt.Sprintf("><title>%s</title></%s>", title, tag)
	}
	return "/>"
}

// getFeatureAttributesAndTitle converts the proeprties of the feature into a string of attributes, and extracts the title property into a string
func getFeatureAttributesAndTitle(useProp func(string) bool, titleProp string, feature *geojson.Feature) (string, string) {
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
func makeScaleFunc(width, height float64, padding Padding, ps [][]float64, projection ScaleFunc) ScaleFunc {
	w := width - padding.Left - padding.Right
	h := height - padding.Top - padding.Bottom

	if len(ps) == 0 {
		return func(x, y float64) (float64, float64) { return projection(x, y) }
	}

	if len(ps) == 1 {
		return func(x, y float64) (float64, float64) { return w / 2, h / 2 }
	}

	minX, minY, maxX, maxY := getBoundingRectangle(projection, ps)
	xRes := (maxX - minX) / w
	yRes := (maxY - minY) / h
	res := math.Max(xRes, yRes)

	return func(x, y float64) (float64, float64) {
		x, y = projection(x, y)
		return (x-minX)/res + padding.Left, (maxY-y)/res + padding.Top
	}

}

func getBoundingRectangle(projection ScaleFunc, ps [][]float64) (float64, float64, float64, float64) {
	if len(ps) == 0 || len(ps[0]) == 0 {
		return 0, 0, 0, 0
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
	return minX, minY, maxX, maxY
}

// GetHeightForWidth returns an appropriate height given a desired width.
func (svg *SVG) GetHeightForWidth(width float64, projection ScaleFunc) float64 {
	minX, minY, maxX, maxY := getBoundingRectangle(projection, svg.points())
	svgWidth := maxX - minX
	svgHeight := maxY - minY
	ratio := svgHeight / svgWidth
	return math.Floor((width * ratio) + .5)

}

// MercatorProjection is a projection function that will convert latitude & logitude into x,y coordinates for a Mercator map.
var MercatorProjection = func(longitude, latitude float64) (float64, float64) {
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

	c[0] /= (area * 6)
	c[1] /= (area * 6)
	return c
}
