package geojson2svg_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"testing"

	"github.com/ONSdigital/dp-map-renderer/geojson2svg"
	"github.com/paulmach/go.geojson"
)

// TODO: Convert tests to use Convey

func trimSpace(s string) string {
	res := bytes.NewBufferString("")
	for _, l := range strings.Split(s, "\n") {
		fmt.Fprintf(res, "%s\n", strings.TrimSpace(l))
	}
	return strings.Trim(res.String(), "\n")
}

func insertNewLine(s string) string {
	return strings.Replace(s, "><", ">\n<", -1)
}

func empty(t *testing.T) {
	expected := insertNewLine(`<svg width="400" height="400.45"></svg>`)

	svg := geojson2svg.New()
	got := svg.Draw(400, 400.45)
	if got != expected {
		t.Errorf("\nexpected \n%s\ngot \n%s", expected, got)
	}
}

func withAPoint(t *testing.T) {
	expected := trimSpace(`
		<svg width="400" height="400">
			<circle cx="200.000000" cy="200.000000" r="1"/>
		</svg>
	`)

	svg := geojson2svg.New()
	addGeometry(t, svg, `{"type": "Point", "coordinates": [10.5,20]}`)

	got := svg.Draw(400, 400)
	if got != expected {
		t.Errorf("\nexpected \n%s\ngot \n%s", expected, got)
	}
}

func addGeometry(t *testing.T, svg *geojson2svg.SVG, s string) {
	g, err := geojson.UnmarshalGeometry([]byte(s))
	if err != nil {
		t.Errorf("invalid geometry: %s", s)
	}
	svg.AppendGeometry(g)
}

func withAMultiPoint(t *testing.T) {
	expected := trimSpace(`
		<svg width="400" height="400">
			<g>
				<circle cx="0.000000" cy="400.000000" r="1"/>
				<circle cx="95.238095" cy="0.000000" r="1"/>
			</g>
		</svg>
	`)

	svg := geojson2svg.New()
	addGeometry(t, svg, `{"type": "MultiPoint", "coordinates": [[10.5,20], [20.5,62]]}`)
	got := svg.Draw(400, 400)
	if got != expected {
		t.Errorf("\nexpected \n%s\ngot \n%s", expected, got)
	}
}

func withALineString(t *testing.T) {
	expected := trimSpace(`
		<svg width="400" height="400">
			<path d="M0.000000 291.638796,400.000000 0.000000"/>
		</svg>
	`)

	svg := geojson2svg.New()
	addGeometry(t, svg, `{"type": "LineString", "coordinates": [[10.4,20.5], [40.3,42.3]]}`)
	got := svg.Draw(400, 400)
	if got != expected {
		t.Errorf("\nexpected \n%s\ngot \n%s", expected, got)
	}
}

func withAMultiLineString(t *testing.T) {
	expected := trimSpace(`
		<svg width="400" height="400">
			<g>
				<path d="M0.000000 282.200647,387.055016 0.000000"/>
				<path d="M12.944984 269.255663,400.000000 12.944984"/>
			</g>
		</svg>
	`)

	svg := geojson2svg.New()
	addGeometry(t, svg, `{"type": "MultiLineString", "coordinates": [[[10.4,20.5], [40.3,42.3]], [[11.4,21.5], [41.3,41.3]]]}`)
	got := svg.Draw(400, 400)
	if got != expected {
		t.Errorf("\nexpected \n%s\ngot \n%s", expected, got)
	}
}

func withAPolygonWithoutHoles(t *testing.T) {
	expected := trimSpace(`
		<svg width="400" height="400">
			<path d="M0.000000 271.651090,372.585670 0.000000,122.118380 400.000000,0.000000 271.651090 Z"/>
		</svg>
	`)

	svg := geojson2svg.New()
	addGeometry(t, svg, `{"type": "Polygon", "coordinates": [[[10.4,20.5], [40.3,42.3], [20.2, 10.2], [10.4,20.5]]]}`)
	got := svg.Draw(400, 400)
	if got != expected {
		t.Errorf("\nexpected \n%s\ngot \n%s", expected, got)
	}
}

func withAPolygonWithHoles(t *testing.T) {
	expected := trimSpace(`
		<svg width="400" height="400">
			<path d="M0.000000 400.000000,400.000000 400.000000,400.000000 0.000000,0.000000 0.000000,0.000000 400.000000 M80.000000 320.000000,320.000000 320.000000,320.000000 80.000000,80.000000 80.000000,80.000000 320.000000 Z"/>
		</svg>
	`)

	svg := geojson2svg.New()
	addGeometry(t, svg, `{"type": "Polygon", "coordinates": [
		[[100.0,0.0], [101.0,0.0], [101.0,1.0], [100.0,1.0], [100.0,0.0]],
    [[100.2,0.2], [100.8,0.2], [100.8,0.8], [100.2,0.8], [100.2,0.2]]
	]}`)
	got := svg.Draw(400, 400)
	if got != expected {
		t.Errorf("\nexpected \n%s\ngot \n%s", expected, got)
	}
}

func withAMultiPolygon(t *testing.T) {
	expected := trimSpace(`
		<svg width="400" height="400">
			<g>
				<path d="M0.000000 96.247241,132.008830 0.000000,43.267108 141.721854,0.000000 96.247241 Z"/>
				<path d="M395.584989 186.754967,400.000000 186.754967,400.000000 182.339956,395.584989 182.339956,395.584989 186.754967 M396.467991 185.871965,399.116998 185.871965,399.116998 183.222958,396.467991 183.222958,396.467991 185.871965 Z"/>
			</g>
		</svg>
	`)

	svg := geojson2svg.New()
	addGeometry(t, svg, `{"type": "MultiPolygon", "coordinates": [
		[
			[[10.4,20.5], [40.3,42.3], [20.2, 10.2], [10.4,20.5]]
		], [
			[[100.0,0.0], [101.0,0.0], [101.0,1.0], [100.0,1.0], [100.0,0.0]],
	    [[100.2,0.2], [100.8,0.2], [100.8,0.8], [100.2,0.8], [100.2,0.2]]
		]
	]}`)
	got := svg.Draw(400, 400)
	if got != expected {
		t.Errorf("\nexpected \n%s\ngot \n%s", expected, got)
	}
}

func withAGeometryCollection(t *testing.T) {
	expected := trimSpace(`
		<svg width="400" height="400">
			<g>
				<path d="M0.000000 291.638796,400.000000 0.000000"/>
				<circle cx="1.337793" cy="298.327759" r="1"/>
			</g>
		</svg>
	`)

	svg := geojson2svg.New()
	addGeometry(t, svg, `{"type": "GeometryCollection", "geometries": [
		{"type": "LineString", "coordinates": [[10.4,20.5], [40.3,42.3]]},
		{"type": "Point", "coordinates": [10.5,20]}
	]}`)
	got := svg.Draw(400, 400)
	if got != expected {
		t.Errorf("\nexpected \n%s\ngot \n%s", expected, got)
	}
}

func withMultipleGeometries(t *testing.T) {
	expected := trimSpace(`
		<svg width="400" height="400">
			<path d="M0.000000 291.638796,400.000000 0.000000"/>
			<circle cx="1.337793" cy="298.327759" r="1"/>
		</svg>
	`)

	svg := geojson2svg.New()
	addGeometry(t, svg, `{"type": "LineString", "coordinates": [[10.4,20.5], [40.3,42.3]]}`)
	addGeometry(t, svg, `{"type": "Point", "coordinates": [10.5,20]}`)
	got := svg.Draw(400, 400)
	if got != expected {
		t.Errorf("\nexpected \n%s\ngot \n%s", expected, got)
	}
}

func withAFeature(t *testing.T) {
	expected := trimSpace(`
		<svg width="400" height="400">
			<circle cx="200.000000" cy="200.000000" r="1"/>
		</svg>
	`)

	svg := geojson2svg.New()
	addFeature(t, svg, `{"type": "Feature", "geometry": {
		"type": "Point",
		"coordinates": [10.5,20]
	}}`)
	got := svg.Draw(400, 400)
	if got != expected {
		t.Errorf("\nexpected \n%s\ngot \n%s", expected, got)
	}
}

func addFeature(t *testing.T, svg *geojson2svg.SVG, s string) {
	f, err := geojson.UnmarshalFeature([]byte(s))
	if err != nil {
		t.Errorf("invalid feature: %s", s)
	}
	svg.AppendFeature(f)
}

func withAFeatureCollection(t *testing.T) {
	expected := trimSpace(`
		<svg width="400" height="400">
			<circle cx="1.337793" cy="298.327759" r="1"/>
			<path d="M0.000000 291.638796,400.000000 0.000000"/>
		</svg>
	`)

	svg := geojson2svg.New()
	addFeatureCollection(t, svg, `{"type": "FeatureCollection", "features": [
		{"type": "Feature", "geometry": {
			"type": "Point",
			"coordinates": [10.5,20]
		}},
		{"type": "Feature", "geometry": {
			"type": "LineString",
			"coordinates": [[10.4,20.5], [40.3,42.3]]
		}}
	]}`)
	got := svg.Draw(400, 400)
	if got != expected {
		t.Errorf("\nexpected \n%s\ngot \n%s", expected, got)
	}
}

func addFeatureCollection(t *testing.T, svg *geojson2svg.SVG, s string) {
	f, err := geojson.UnmarshalFeatureCollection([]byte(s))
	if err != nil {
		t.Errorf("invalid feature: %s", s)
	}
	svg.AppendFeatureCollection(f)
}

func TestSVG(t *testing.T) {
	tcs := []struct {
		name string
		test func(*testing.T)
	}{
		{"empty svg", empty},
		{"svg with a point", withAPoint},
		{"svg with a multipoint", withAMultiPoint},
		{"svg with a linestring", withALineString},
		{"svg with a multilinestring", withAMultiLineString},
		{"svg with a polygon without holes", withAPolygonWithoutHoles},
		{"svg with a polygon with holes", withAPolygonWithHoles},
		{"svg with a multipolygon", withAMultiPolygon},
		{"svg with a geometry collection", withAGeometryCollection},
		{"svg with multiple geometries", withMultipleGeometries},
		{"svg with a feature", withAFeature},
		{"svg with a feature collection", withAFeatureCollection},
	}

	for _, tc := range tcs {
		t.Run(tc.name, tc.test)
	}
}

func TestSVGAttributeOptions(t *testing.T) {
	tcs := []struct {
		name string
		fn   func(*testing.T)
	}{
		{"should add the passed attribute to the svg tag", withAttributeOption},
		{"should add the passed attributes to the svg tag", withAttributesOption},
		{"latest attribute wins", withAttributeMultipleTimesOption},
		{"no attributes are lost", withAttributesNothingIsLostOption},
	}

	for _, tc := range tcs {
		t.Run(tc.name, tc.fn)
	}
}

func withAttributeOption(t *testing.T) {
	want := insertNewLine(`<svg width="200" height="200" class="a_class" id="the_id"></svg>`)
	svg := geojson2svg.New()
	got := svg.Draw(200, 200,
		geojson2svg.WithAttribute("id", "the_id"),
		geojson2svg.WithAttribute("class", "a_class"))

	if got != want {
		t.Errorf("wanted\n%s, got\n%s", want, got)
	}
}

func withAttributeMultipleTimesOption(t *testing.T) {
	want := insertNewLine(`<svg width="200" height="200" class="a_class_2" id="the_id_2"></svg>`)
	svg := geojson2svg.New()
	got := svg.Draw(200, 200,
		geojson2svg.WithAttribute("id", "the_id"),
		geojson2svg.WithAttribute("class", "a_class"),
		geojson2svg.WithAttribute("class", "a_class_2"),
		geojson2svg.WithAttribute("id", "the_id_2"))

	if got != want {
		t.Errorf("wanted\n%s\ngot\n%s", want, got)
	}
}

func withAttributesOption(t *testing.T) {
	want := insertNewLine(`<svg width="200" height="200" class="a_class" id="the_id"></svg>`)

	attributes := map[string]string{
		"id":    "the_id",
		"class": "a_class",
	}

	svg := geojson2svg.New()
	got := svg.Draw(200, 200, geojson2svg.WithAttributes(attributes))

	if got != want {
		t.Errorf("wanted\n%s\ngot\n%s", want, got)
	}
}

func withAttributesNothingIsLostOption(t *testing.T) {
	want := insertNewLine(`<svg width="200" height="200" class="a_class_2" id="the_id"></svg>`)

	attributesA := map[string]string{"id": "the_id", "class": "a_class"}
	attributesB := map[string]string{"class": "a_class_2"}

	svg := geojson2svg.New()
	got := svg.Draw(200, 200,
		geojson2svg.WithAttributes(attributesA),
		geojson2svg.WithAttributes(attributesB))

	if got != want {
		t.Errorf("wanted\n%s\ngot\n%s", want, got)
	}
}

func TestSVGPaddingOption(t *testing.T) {
	tcs := []struct {
		name     string
		data     string
		padding  geojson2svg.Padding
		expected string
	}{
		{"without padding",
			"[[0,0], [0,400], [400,400], [400,0]]",
			geojson2svg.Padding{Top: 0, Right: 0, Bottom: 0, Left: 0},
			insertNewLine(`<svg width="200" height="200"><path d="M0.000000 200.000000,0.000000 0.000000,200.000000 0.000000,200.000000 200.000000"/></svg>`)},
		{"with padding",
			"[[0,0], [0,400], [400,400], [400,0]]",
			geojson2svg.Padding{Top: 5, Right: 5, Bottom: 5, Left: 5},
			insertNewLine(`<svg width="200" height="200"><path d="M5.000000 195.000000,5.000000 5.000000,195.000000 5.000000,195.000000 195.000000"/></svg>`)},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			svg := geojson2svg.New()
			addGeometry(t, svg, fmt.Sprintf(`{"type": "LineString", "coordinates": %s}`, tc.data))
			padding := geojson2svg.WithPadding(tc.padding)
			got := svg.Draw(200, 200, padding)
			if got != tc.expected {
				t.Errorf("\nexpected \n%s\ngot \n%s", tc.expected, got)
			}
		})
	}
}

func TestFeatureProperties(t *testing.T) {
	tcs := []struct {
		name      string
		feature   string
		usedProps []string
		expected  string
	}{
		{"no props (point)",
			`{"type": "Feature", "geometry": { "type": "Point", "coordinates": [10.5,20] }}`,
			nil,
			insertNewLine(`<svg width="400" height="400"><circle cx="200.000000" cy="200.000000" r="1"/></svg>`)},
		{"with class (point)",
			`{"type": "Feature", "properties": {"class": "class"}, "geometry": { "type": "Point", "coordinates": [10.5,20] }}`,
			nil,
			insertNewLine(`<svg width="400" height="400"><circle cx="200.000000" cy="200.000000" r="1" class="class"/></svg>`)},
		{"with class and unused (point)",
			`{"type": "Feature", "properties": {"class": "class", "style": "stroke:1"}, "geometry": { "type": "Point", "coordinates": [10.5,20] }}`,
			nil,
			insertNewLine(`<svg width="400" height="400"><circle cx="200.000000" cy="200.000000" r="1" class="class"/></svg>`)},
		{"with unused (point)",
			`{"type": "Feature", "properties": {"style": "stroke:1"}, "geometry": { "type": "Point", "coordinates": [10.5,20] }}`,
			nil,
			insertNewLine(`<svg width="400" height="400"><circle cx="200.000000" cy="200.000000" r="1"/></svg>`)},
		{"with added props (point)",
			`{"type": "Feature", "properties": {"style": "stroke:1"}, "geometry": { "type": "Point", "coordinates": [10.5,20] }}`,
			[]string{"style"},
			insertNewLine(`<svg width="400" height="400"><circle cx="200.000000" cy="200.000000" r="1" style="stroke:1"/></svg>`)},
		{"with class removed (point)",
			`{"type": "Feature", "properties": {"class": "class"}, "geometry": { "type": "Point", "coordinates": [10.5,20] }}`,
			[]string{},
			insertNewLine(`<svg width="400" height="400"><circle cx="200.000000" cy="200.000000" r="1"/></svg>`)},

		{"no props (linestring)",
			`{"type": "Feature", "geometry": { "type": "LineString", "coordinates": [[10.4,20.5], [40.3,42.3]] }}`,
			nil,
			insertNewLine(`<svg width="400" height="400"><path d="M0.000000 291.638796,400.000000 0.000000"/></svg>`)},
		{"with class (linestring)",
			`{"type": "Feature", "properties": {"class": "class"}, "geometry": { "type": "LineString", "coordinates": [[10.4,20.5], [40.3,42.3]] }}`,
			nil,
			insertNewLine(`<svg width="400" height="400"><path d="M0.000000 291.638796,400.000000 0.000000" class="class"/></svg>`)},
		{"with class and unused (linestring)",
			`{"type": "Feature", "properties": {"class": "class", "style": "stroke:1"}, "geometry": { "type": "LineString", "coordinates": [[10.4,20.5], [40.3,42.3]] }}`,
			nil,
			insertNewLine(`<svg width="400" height="400"><path d="M0.000000 291.638796,400.000000 0.000000" class="class"/></svg>`)},
		{"with unused (linestring)",
			`{"type": "Feature", "properties": {"style": "stroke:1"}, "geometry": { "type": "LineString", "coordinates": [[10.4,20.5], [40.3,42.3]] }}`,
			nil,
			insertNewLine(`<svg width="400" height="400"><path d="M0.000000 291.638796,400.000000 0.000000"/></svg>`)},
		{"with added props (linestring)",
			`{"type": "Feature", "properties": {"style": "stroke:1"}, "geometry": { "type": "LineString", "coordinates": [[10.4,20.5], [40.3,42.3]] }}`,
			[]string{"style"},
			insertNewLine(`<svg width="400" height="400"><path d="M0.000000 291.638796,400.000000 0.000000" style="stroke:1"/></svg>`)},
		{"with class removed (linestring)",
			`{"type": "Feature", "properties": {"class": "class"}, "geometry": { "type": "LineString", "coordinates": [[10.4,20.5], [40.3,42.3]] }}`,
			[]string{},
			insertNewLine(`<svg width="400" height="400"><path d="M0.000000 291.638796,400.000000 0.000000"/></svg>`)},

		{"no props (polygon)",
			`{"type": "Feature", "geometry": { "type": "Polygon", "coordinates": [[[10.4,20.5], [40.3,42.3], [20.2, 10.2], [10.4,20.5]]] }}`,
			nil,
			insertNewLine(`<svg width="400" height="400"><path d="M0.000000 271.651090,372.585670 0.000000,122.118380 400.000000,0.000000 271.651090 Z"/></svg>`)},
		{"with class (polygon)",
			`{"type": "Feature", "properties": {"class": "class"}, "geometry": { "type": "Polygon", "coordinates": [[[10.4,20.5], [40.3,42.3], [20.2, 10.2], [10.4,20.5]]] }}`,
			nil,
			insertNewLine(`<svg width="400" height="400"><path d="M0.000000 271.651090,372.585670 0.000000,122.118380 400.000000,0.000000 271.651090 Z" class="class"/></svg>`)},
		{"with class and unused (polygon)",
			`{"type": "Feature", "properties": {"class": "class", "style": "stroke:1"}, "geometry": { "type": "Polygon", "coordinates": [[[10.4,20.5], [40.3,42.3], [20.2, 10.2], [10.4,20.5]]] }}`,
			nil,
			insertNewLine(`<svg width="400" height="400"><path d="M0.000000 271.651090,372.585670 0.000000,122.118380 400.000000,0.000000 271.651090 Z" class="class"/></svg>`)},
		{"with unused (polygon)",
			`{"type": "Feature", "properties": {"style": "stroke:1"}, "geometry": { "type": "Polygon", "coordinates": [[[10.4,20.5], [40.3,42.3], [20.2, 10.2], [10.4,20.5]]] }}`,
			nil,
			insertNewLine(`<svg width="400" height="400"><path d="M0.000000 271.651090,372.585670 0.000000,122.118380 400.000000,0.000000 271.651090 Z"/></svg>`)},
		{"with added props (polygon)",
			`{"type": "Feature", "properties": {"style": "stroke:1"}, "geometry": { "type": "Polygon", "coordinates": [[[10.4,20.5], [40.3,42.3], [20.2, 10.2], [10.4,20.5]]] }}`,
			[]string{"style"},
			insertNewLine(`<svg width="400" height="400"><path d="M0.000000 271.651090,372.585670 0.000000,122.118380 400.000000,0.000000 271.651090 Z" style="stroke:1"/></svg>`)},
		{"with class removed (polygon)",
			`{"type": "Feature", "properties": {"class": "class"}, "geometry": { "type": "Polygon", "coordinates": [[[10.4,20.5], [40.3,42.3], [20.2, 10.2], [10.4,20.5]]] }}`,
			[]string{},
			insertNewLine(`<svg width="400" height="400"><path d="M0.000000 271.651090,372.585670 0.000000,122.118380 400.000000,0.000000 271.651090 Z"/></svg>`)},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			svg := geojson2svg.New()
			addFeature(t, svg, tc.feature)

			var got string
			if tc.usedProps != nil {
				got = svg.Draw(400, 400, geojson2svg.UseProperties(tc.usedProps))
			} else {
				got = svg.Draw(400, 400)
			}
			if got != tc.expected {
				tt.Errorf("\nexpected \n%s\ngot \n%s", tc.expected, got)
			}
		})
	}
}

func TestExample(t *testing.T) {
	exampleFile := path.Join("testdata", "example.json")
	geojson, err := ioutil.ReadFile(exampleFile)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	svgFile := path.Join("testdata", "example.svg")
	want, err := ioutil.ReadFile(svgFile)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	svg := geojson2svg.New()
	addFeatureCollection(t, svg, string(geojson))

	got := svg.Draw(1000, 510,
		geojson2svg.WithAttribute("xmlns", "http://www.w3.org/2000/svg"),
		geojson2svg.UseProperties([]string{"style"}),
		geojson2svg.WithPadding(geojson2svg.Padding{
			Top:    10,
			Right:  10,
			Bottom: 10,
			Left:   10,
		}))
	//ioutil.WriteFile("testdata/example.svg", []byte(got), 0644)
	if got != string(want) {
		t.Errorf("\nexpected \n%s\ngot \n%s", string(want), got)
	}
}
