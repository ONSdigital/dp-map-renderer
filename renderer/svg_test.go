package renderer_test

import (
	"bytes"
	"testing"

	"github.com/ONSdigital/dp-map-renderer/models"
	"github.com/ONSdigital/dp-map-renderer/testdata"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/rubenv/topojson"
	. "github.com/ONSdigital/dp-map-renderer/renderer"
	"encoding/xml"
)

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

func TestRenderSVGSucceedsWithNullValues(t *testing.T) {

	Convey("RenderSVG should not fail with null Geography", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:"testname",
		}

		result := RenderSVG(renderRequest)

		So(result, ShouldEqual, "")
	})

	Convey("RenderSVG should not fail with null Topology", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:"testname",
			Geography: &models.Geography{},
		}

		result := RenderSVG(renderRequest)

		So(result, ShouldEqual, "")
	})

	Convey("RenderSVG should not fail with null Topology.Objects", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:"testname",
			Geography: &models.Geography{Topojson:simpleTopology()},
		}
		renderRequest.Geography.Topojson.Objects = nil

		result := RenderSVG(renderRequest)

		So(result, ShouldEqual, "")
	})

	Convey("RenderSVG should not fail with null Topology.Arcs", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:"testname",
			Geography: &models.Geography{Topojson:simpleTopology()},
		}
		renderRequest.Geography.Topojson.Arcs = nil

		result := RenderSVG(renderRequest)

		So(result, ShouldEqual, "")
	})
}

func TestSVGHasWidthAndHeight(t *testing.T) {

	Convey("SVG should be given default width and proportional height", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:"testname",
			Geography: &models.Geography{Topojson:simpleTopology(), IDProperty:"code", NameProperty:"name"},
		}

		result := RenderSVG(renderRequest)

		svg, e := unmarshalSimpleSVG(result)
		So(e, ShouldBeNil)
		So(svg.Width, ShouldEqual, "400")
		So(svg.Height, ShouldEqual, "329")
	})

	Convey("SVG should use given width and calculate proportional height", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:"testname",
			Geography: &models.Geography{Topojson:simpleTopology(), IDProperty:"code", NameProperty:"name"},
			Width: 800,
		}

		result := RenderSVG(renderRequest)

		svg, e := unmarshalSimpleSVG(result)
		So(e, ShouldBeNil)
		So(svg.Width, ShouldEqual, "800")
		So(svg.Height, ShouldEqual, "658")
	})

	Convey("SVG should use given width and height", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:"testname",
			Geography: &models.Geography{Topojson:simpleTopology(), IDProperty:"code", NameProperty:"name"},
			Width: 800,
			Height: 600,
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

	Convey("SVG should assign class to map regions", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:"testname",
			Geography: &models.Geography{Topojson:simpleTopology(), IDProperty:"code", NameProperty:"name"},
		}
		renderRequest.Geography.Topojson.Objects["simplegeojson"].Geometries[0].Properties["class"] = "foo"

		result := RenderSVG(renderRequest)

		svg, e := unmarshalSimpleSVG(result)
		So(e, ShouldBeNil)
		So(len(svg.Paths), ShouldEqual, 2)
		So(svg.Paths[0].Class, ShouldEqual, REGION_CLASS_NAME + " foo")
		So(svg.Paths[1].Class, ShouldEqual, REGION_CLASS_NAME)
	})
}

func TestSVGContainsIDs(t *testing.T) {

	Convey("SVG should assign ids to map regions", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:"testname",
			Geography: &models.Geography{Topojson:simpleTopology(), IDProperty:"code", NameProperty:"name"},
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

	Convey("SVG should assign names as titles to map regions", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:"testname",
			Geography: &models.Geography{Topojson:simpleTopology(), IDProperty:"code", NameProperty:"name"},
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

	Convey("SVG should use style to colour regions", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:   "testname",
			Geography:  &models.Geography{Topojson: simpleTopology(), IDProperty: "code", NameProperty: "name"},
			Choropleth: &models.Choropleth{Breaks: []*models.ChoroplethBreak{{LowerBound: 0, Color: "red"}, {LowerBound: 11, Color: "green"}}},
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

	Convey("SVG should use style to colour regions, applying style to regions missing data, and modify the title with values", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:  "testname",
			Geography: &models.Geography{Topojson: simpleTopology(), IDProperty: "code", NameProperty: "name"},
			Choropleth: &models.Choropleth{MissingValueColor: "white",
				Breaks: []*models.ChoroplethBreak{{LowerBound: 0, Color: "red"}, {LowerBound: 11, Color: "green"}},
				ValuePrefix:"prefix-",
				ValueSuffix:"-suffix"},
			Data: []*models.DataRow{{ID: "f1", Value: 20}},
		}

		result := RenderSVG(renderRequest)

		So(result, ShouldNotBeNil)
		svg, e := unmarshalSimpleSVG(result)
		So(e, ShouldBeNil)
		So(len(svg.Paths), ShouldEqual, 2)
		So(svg.Paths[0].Style, ShouldContainSubstring, "fill: white;")
		So(svg.Paths[1].Style, ShouldContainSubstring, "fill: green;")

		So(svg.Paths[0].Title.Value, ShouldContainSubstring, "feature 0 " + MISSING_DATA_TEXT)
		So(svg.Paths[1].Title.Value, ShouldContainSubstring, "feature 1 prefix-20-suffix")
	})
}

// simpleTopology returns a topology with 2 features: code=f0, name=feature 0; code=f1, name=feature 1
func simpleTopology() *topojson.Topology {
	simpleTopology, _ := topojson.UnmarshalTopology([]byte(`{"type":"Topology","objects":{"simplegeojson":{"type":"GeometryCollection","geometries":[{"type":"Polygon","arcs":[[0]],"properties":{"code":"f0","name":"feature 0"}},{"type":"Polygon","arcs":[[1]],"properties":{"code":"f1","name":"feature 1"}}]}},"arcs":[[[47.13148713111877,9.53216215939578],[47.13148713111877,9.53216215939578],[47.13148713111877,9.53216215939578],[47.13148713111877,9.53216215939578]],[[47.128000259399414,9.52858586376412],[47.132699489593506,9.52858586376412],[47.132699489593506,9.532394934735397],[47.128000259399414,9.532394934735397],[47.128000259399414,9.52858586376412]]],"bbox":[47.128000259399414,9.52858586376412,47.132699489593506,9.532394934735397]}`))
	return simpleTopology
}

// definition of an SVG sufficient to get details for a simple topology
type SVG struct {
	Paths   []Path `xml:"path"`
	Width   string `xml:"width,attr"`
	Height  string `xml:"height,attr"`
	ViewBox string `xml:"viewBox,attr"`
}

type Path struct {
	D     string `xml:"d,attr"`
	ID    string `xml:"id,attr"`
	Style string `xml:"style,attr"`
	Class string `xml:"class,attr"`
	Title Title  `xml:"title"`
}

type Title struct {
	Value string `xml:",chardata"`
}

func unmarshalSimpleSVG(source string) (*SVG, error) {
	svg := &SVG{}
	err := xml.Unmarshal([]byte(source), svg)
	return svg, err
}