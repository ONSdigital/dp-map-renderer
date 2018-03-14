package renderer_test

import (
	"bytes"
	"testing"

	"encoding/xml"
	"fmt"

	"regexp"
	"strconv"

	"github.com/ONSdigital/dp-map-renderer/models"
	. "github.com/ONSdigital/dp-map-renderer/renderer"
	"github.com/ONSdigital/dp-map-renderer/testdata"
	"github.com/rubenv/topojson"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/ONSdigital/dp-map-renderer/geojson2svg"
)

var pngConverter = geojson2svg.NewPNGConverter("sh", []string{"-c", `echo "test" >> ` + geojson2svg.ArgPNGFilename})
var expectedFallbackImage = `<img alt="Fallback map image for older browsers" src="data:image/png;base64,dGVzdAo=" />`

func TestRenderSVG(t *testing.T) {

	Convey("Successfully render an svg map", t, func() {
		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}

		result := RenderSVG(renderRequest)

		So(result, ShouldNotBeNil)
		So(result, ShouldStartWith, `<svg width="400" height="748" viewBox="0 0 400 748">`)
	})
}

func TestRenderSVGDoesNotIncludeFallbackPng(t *testing.T) {

	Convey("Successfully render an svg map without fallback png", t, func() {

		UsePNGConverter(nil)
		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}

		result := RenderSVG(renderRequest)

		So(result, ShouldNotBeNil)
		So(result, ShouldStartWith, `<svg `)
		So(result, ShouldNotContainSubstring, `<foreignObject>`)
	})
}

func TestRenderSVGIncludesFallbackPng(t *testing.T) {

	Convey("Successfully render an svg map with fallback png", t, func() {

		UsePNGConverter(pngConverter)

		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}

		result := RenderSVG(renderRequest)

		So(result, ShouldNotBeNil)
		So(result, ShouldStartWith, `<svg `)
		So(result, ShouldContainSubstring, `<foreignObject>`)
		So(result, ShouldContainSubstring, expectedFallbackImage)
	})
}

func TestRenderSVGSucceedsWithNullValues(t *testing.T) {

	Convey("RenderSVG should not fail with null Geography", t, func() {

		renderRequest := &models.RenderRequest{
			Filename: "testname",
		}

		result := RenderSVG(renderRequest)

		So(result, ShouldEqual, "")
	})

	Convey("RenderSVG should not fail with null Topology", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:  "testname",
			Geography: &models.Geography{},
		}

		result := RenderSVG(renderRequest)

		So(result, ShouldEqual, "")
	})

	Convey("RenderSVG should not fail with null Topology.Objects", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:  "testname",
			Geography: &models.Geography{Topojson: simpleTopology()},
		}
		renderRequest.Geography.Topojson.Objects = nil

		result := RenderSVG(renderRequest)

		So(result, ShouldEqual, "")
	})

	Convey("RenderSVG should not fail with null Topology.Arcs", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:  "testname",
			Geography: &models.Geography{Topojson: simpleTopology()},
		}
		renderRequest.Geography.Topojson.Arcs = nil

		result := RenderSVG(renderRequest)

		So(result, ShouldEqual, "")
	})
}

func TestSVGHasWidthAndHeight(t *testing.T) {

	Convey("simpleSVG should be given default width and proportional height", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:  "testname",
			Geography: &models.Geography{Topojson: simpleTopology(), IDProperty: "code", NameProperty: "name"},
		}

		result := RenderSVG(renderRequest)

		svg, e := unmarshalSimpleSVG(result)
		So(e, ShouldBeNil)
		So(svg.Width, ShouldEqual, "400")
		So(svg.Height, ShouldEqual, "329")
	})

	Convey("simpleSVG should use given width and calculate proportional height", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:  "testname",
			Geography: &models.Geography{Topojson: simpleTopology(), IDProperty: "code", NameProperty: "name"},
			Width:     800,
		}

		result := RenderSVG(renderRequest)

		svg, e := unmarshalSimpleSVG(result)
		So(e, ShouldBeNil)
		So(svg.Width, ShouldEqual, "800")
		So(svg.Height, ShouldEqual, "658")
	})

	Convey("simpleSVG should use given width and height", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:  "testname",
			Geography: &models.Geography{Topojson: simpleTopology(), IDProperty: "code", NameProperty: "name"},
			Width:     800,
			Height:    600,
		}

		result := RenderSVG(renderRequest)

		svg, e := unmarshalSimpleSVG(result)
		So(e, ShouldBeNil)
		So(svg.Width, ShouldEqual, "800")
		So(svg.Height, ShouldEqual, "600")
	})
}

func TestSVGHasCorrectViewBox(t *testing.T) {

	Convey("Report correctly scaled viewbox when height provided for svg", t, func() {
		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}
		renderRequest.Height = 600

		result := RenderSVG(renderRequest)

		svg, e := unmarshalSimpleSVG(result)
		So(e, ShouldBeNil)
		So(svg.Width, ShouldEqual, "400")
		So(svg.Height, ShouldEqual, "600")
		So(svg.ViewBox, ShouldEqual, "0 0 400 748")
	})
}
func TestSVGContainsClassName(t *testing.T) {

	Convey("simpleSVG should assign class to map regions", t, func() {

		UsePNGConverter(nil)

		renderRequest := &models.RenderRequest{
			Filename:  "testname",
			Geography: &models.Geography{Topojson: simpleTopology(), IDProperty: "code", NameProperty: "name"},
		}
		renderRequest.Geography.Topojson.Objects["simplegeojson"].Geometries[0].Properties["class"] = "foo"

		result := RenderSVG(renderRequest)

		svg, e := unmarshalSimpleSVG(result)
		So(e, ShouldBeNil)
		So(len(svg.Paths), ShouldEqual, 2)
		So(svg.Paths[0].Class, ShouldEqual, RegionClassName+" foo")
		So(svg.Paths[1].Class, ShouldEqual, RegionClassName)
	})
}

func TestSVGContainsIDs(t *testing.T) {

	Convey("simpleSVG should assign ids to map regions", t, func() {

		UsePNGConverter(nil)

		renderRequest := &models.RenderRequest{
			Filename:  "testname",
			Geography: &models.Geography{Topojson: simpleTopology(), IDProperty: "code", NameProperty: "name"},
		}

		result := RenderSVG(renderRequest)

		svg, e := unmarshalSimpleSVG(result)
		So(e, ShouldBeNil)
		So(len(svg.Paths), ShouldEqual, 2)
		So(svg.Paths[0].ID, ShouldEqual, "testname-f0")
		So(svg.Paths[1].ID, ShouldEqual, "testname-f1")
	})
}

func TestSVGContainsTitles(t *testing.T) {

	Convey("simpleSVG should assign names as titles to map regions", t, func() {

		UsePNGConverter(nil)

		renderRequest := &models.RenderRequest{
			Filename:  "testname",
			Geography: &models.Geography{Topojson: simpleTopology(), IDProperty: "code", NameProperty: "name"},
		}

		result := RenderSVG(renderRequest)

		So(result, ShouldNotBeNil)
		svg, e := unmarshalSimpleSVG(result)
		So(e, ShouldBeNil)
		So(len(svg.Paths), ShouldEqual, 2)
		So(svg.Paths[0].Title.Value, ShouldEqual, "feature 0")
		So(svg.Paths[1].Title.Value, ShouldEqual, "feature 1")
	})
}

func TestSVGContainsChoroplethColours(t *testing.T) {

	Convey("simpleSVG should use style to colour regions", t, func() {

		UsePNGConverter(nil)

		renderRequest := &models.RenderRequest{
			Filename:   "testname",
			Geography:  &models.Geography{Topojson: simpleTopology(), IDProperty: "code", NameProperty: "name"},
			Choropleth: &models.Choropleth{Breaks: []*models.ChoroplethBreak{{LowerBound: 0, Colour: "red"}, {LowerBound: 11, Colour: "green"}}},
			Data:       []*models.DataRow{{ID: "f0", Value: 10}, {ID: "f1", Value: 20}},
		}

		result := RenderSVG(renderRequest)

		So(result, ShouldNotBeNil)
		svg, e := unmarshalSimpleSVG(result)
		So(e, ShouldBeNil)
		So(len(svg.Paths), ShouldEqual, 2)
		So(svg.Paths[0].Style, ShouldContainSubstring, "fill: red")
		So(svg.Paths[1].Style, ShouldContainSubstring, "fill: green")
	})
}

func TestSVGHasMissingValueColourAndCorrectTitle(t *testing.T) {

	Convey("simpleSVG should use style to colour regions, applying style to regions missing data, and modify the title with values", t, func() {

		UsePNGConverter(nil)

		renderRequest := &models.RenderRequest{
			Filename:  "testname",
			Geography: &models.Geography{Topojson: simpleTopology(), IDProperty: "code", NameProperty: "name"},
			Choropleth: &models.Choropleth{MissingValueColor: "white",
				Breaks:      []*models.ChoroplethBreak{{LowerBound: 0, Colour: "red"}, {LowerBound: 11, Colour: "green"}},
				ValuePrefix: "prefix-",
				ValueSuffix: "-suffix"},
			Data: []*models.DataRow{{ID: "f1", Value: 20}},
		}

		result := RenderSVG(renderRequest)

		So(result, ShouldNotBeNil)
		svg, e := unmarshalSimpleSVG(result)
		So(e, ShouldBeNil)
		So(len(svg.Paths), ShouldEqual, 2)
		So(svg.Paths[0].Style, ShouldContainSubstring, "fill: white;")
		So(svg.Paths[1].Style, ShouldContainSubstring, "fill: green;")

		So(svg.Paths[0].Title.Value, ShouldContainSubstring, "feature 0 "+MissingDataText)
		So(svg.Paths[1].Title.Value, ShouldContainSubstring, "feature 1 prefix-20-suffix")
	})
}

func TestRenderVerticalKey(t *testing.T) {
	Convey("RenderVerticalKey should render an svg", t, func() {

		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}

		result := RenderVerticalKey(renderRequest)

		So(result, ShouldNotBeNil)
		So(result, ShouldStartWith, `<svg id="abcd1234-legend-vertical" class="map_key_vertical"`)
		assertKeyContents(result, renderRequest)

	})

}

func TestRenderHorizontalKey(t *testing.T) {
	Convey("RenderHorizontalKey should render an svg", t, func() {

		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}

		result := RenderHorizontalKey(renderRequest)

		So(result, ShouldNotBeNil)
		So(result, ShouldStartWith, `<svg id="abcd1234-legend-horizontal" class="map_key_horizontal" width="400" height="90" viewBox="0 0 400 90">`)
		assertKeyContents(result, renderRequest)
	})

}

func TestRenderHorizontalKeyHasFallbackPng(t *testing.T) {
	Convey("RenderHorizontalKey should render a fallback png", t, func() {

		UsePNGConverter(pngConverter)

		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}

		result := RenderHorizontalKey(renderRequest)

		So(result, ShouldNotBeNil)
		So(result, ShouldContainSubstring, `<foreignObject>`)
		So(result, ShouldContainSubstring, expectedFallbackImage)

	})

}

func TestRenderHorizontalKeyDoesNotHaveFallbackPng(t *testing.T) {
	Convey("RenderHorizontalKey should not render a fallback png", t, func() {

		UsePNGConverter(nil)

		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}

		result := RenderHorizontalKey(renderRequest)

		So(result, ShouldNotBeNil)
		So(result, ShouldStartWith, `<svg `)
		So(result, ShouldNotContainSubstring, `<foreignObject>`)

	})

}
func TestRenderVerticalKeyHasFallbackPng(t *testing.T) {
	Convey("RenderVerticalKey should render a fallback png", t, func() {

		UsePNGConverter(pngConverter)

		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}

		result := RenderVerticalKey(renderRequest)

		So(result, ShouldNotBeNil)
		So(result, ShouldContainSubstring, `<foreignObject>`)
		So(result, ShouldContainSubstring, expectedFallbackImage)

	})

}

func TestRenderVerticalKeyDoesNotHaveFallbackPng(t *testing.T) {
	Convey("RenderVerticalKey should not render a fallback png", t, func() {

		UsePNGConverter(nil)

		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}

		result := RenderVerticalKey(renderRequest)

		So(result, ShouldNotBeNil)
		So(result, ShouldStartWith, `<svg `)
		So(result, ShouldNotContainSubstring, `<foreignObject>`)

	})

}

func TestRenderHorizontalKeyHasCorrectUpperBound(t *testing.T) {
	Convey("RenderHorizontalKey should render an svg with upper bound text as specified in the request", t, func() {

		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}
		renderRequest.Choropleth.UpperBound = 53

		result := RenderHorizontalKey(renderRequest)

		So(result, ShouldNotBeNil)
		So(result, ShouldNotContainSubstring, `>54<`)
		So(result, ShouldContainSubstring, `>53<`)

	})

}

func assertKeyContents(result string, renderRequest *models.RenderRequest) {
	So(result, ShouldContainSubstring, renderRequest.Choropleth.ValuePrefix)
	So(result, ShouldContainSubstring, renderRequest.Choropleth.ValueSuffix)
	for _, b := range renderRequest.Choropleth.Breaks {
		So(result, ShouldContainSubstring, "fill: "+b.Colour)
		So(result, ShouldContainSubstring, fmt.Sprintf("%g", b.LowerBound))
	}
	So(result, ShouldContainSubstring, "fill: "+renderRequest.Choropleth.MissingValueColor)
	So(result, ShouldContainSubstring, MissingDataText)
	// ensure all text has a class applied
	textElements := regexp.MustCompile("<text").FindAllStringIndex(result, -1)
	withClass := regexp.MustCompile(`<text[^>]*class="[^"]*keyText[^>]*"`).FindAllStringIndex(result, -1)
	So(len(withClass), ShouldEqual, len(textElements))
}

func TestRenderVerticalKeyWidth(t *testing.T) {
	Convey("RenderVerticalKey should adjust width to acommodate the text", t, func() {

		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}
		renderRequest.Choropleth.ValueSuffix = "short text"

		result := RenderVerticalKey(renderRequest)

		defaultWidth := getWidth(result)

		Convey("A long title should cause the width to increase", func() {
			renderRequest.Choropleth.ValueSuffix = "some text long enough to increase the width...."
			width := getWidth(RenderVerticalKey(renderRequest))

			So(width, ShouldBeGreaterThan, defaultWidth)

		})

		Convey("A long reference text should cause the width to increase", func() {
			renderRequest.Choropleth.ValueSuffix = "short text"
			renderRequest.Choropleth.ReferenceValueText = "some text long enough to increase the width..."
			width := getWidth(RenderVerticalKey(renderRequest))

			So(width, ShouldBeGreaterThan, defaultWidth)

		})

		Convey("Long tick values should cause the width to increase", func() {
			renderRequest.Choropleth.ValueSuffix = "short text"
			renderRequest.Choropleth.ReferenceValueText = ".."
			renderRequest.Choropleth.Breaks[0].LowerBound = 123456.123456
			renderRequest.Choropleth.ReferenceValue = 123456.123456
			width := getWidth(RenderVerticalKey(renderRequest))

			So(width, ShouldBeGreaterThan, defaultWidth)

		})
	})

}
func getWidth(result string) int {
	widthRE := regexp.MustCompile(`width="([\d]+)"`)
	submatch := widthRE.FindStringSubmatch(result)
	So(len(submatch), ShouldEqual, 2)
	width, err := strconv.Atoi(submatch[1])
	So(err, ShouldBeNil)
	return width
}

// simpleTopology returns a topology with 2 features: code=f0, name=feature 0; code=f1, name=feature 1
func simpleTopology() *topojson.Topology {
	simpleTopology, _ := topojson.UnmarshalTopology([]byte(`{"type":"Topology","objects":{"simplegeojson":{"type":"GeometryCollection","geometries":[{"type":"Polygon","arcs":[[0]],"properties":{"code":"f0","name":"feature 0"}},{"type":"Polygon","arcs":[[1]],"properties":{"code":"f1","name":"feature 1"}}]}},"arcs":[[[47.13148713111877,9.53216215939578],[47.13148713111877,9.53216215939578],[47.13148713111877,9.53216215939578],[47.13148713111877,9.53216215939578]],[[47.128000259399414,9.52858586376412],[47.132699489593506,9.52858586376412],[47.132699489593506,9.532394934735397],[47.128000259399414,9.532394934735397],[47.128000259399414,9.52858586376412]]],"bbox":[47.128000259399414,9.52858586376412,47.132699489593506,9.532394934735397]}`))
	return simpleTopology
}

// definition of an SVG sufficient to get details for a simple topology
type simpleSVG struct {
	Paths   []path `xml:"path"`
	Width   string `xml:"width,attr"`
	Height  string `xml:"height,attr"`
	ViewBox string `xml:"viewBox,attr"`
}

type path struct {
	D     string `xml:"d,attr"`
	ID    string `xml:"id,attr"`
	Style string `xml:"style,attr"`
	Class string `xml:"class,attr"`
	Title title  `xml:"title"`
}

type title struct {
	Value string `xml:",chardata"`
}

func unmarshalSimpleSVG(source string) (*simpleSVG, error) {
	svg := &simpleSVG{}
	err := xml.Unmarshal([]byte(source), svg)
	return svg, err
}
