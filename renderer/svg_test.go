package renderer_test

import (
	"bytes"
	"testing"

	"github.com/ONSdigital/dp-map-renderer/models"
	"github.com/ONSdigital/dp-map-renderer/testdata"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/rubenv/topojson"
	. "github.com/ONSdigital/dp-map-renderer/renderer"
	"strings"
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
		So(result, ShouldStartWith, `<svg width="400" height="748">`)
	})
}

func TestSVGSucceedsWithNullValues(t *testing.T) {

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

		So(result, ShouldNotBeNil)
		So(result, ShouldContainSubstring, `svg width="400" height="329"`)
	})

	Convey("SVG should use given width and calculate proportional height", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:"testname",
			Geography: &models.Geography{Topojson:simpleTopology(), IDProperty:"code", NameProperty:"name"},
			Width: 800,
		}

		result := RenderSVG(renderRequest)

		So(result, ShouldNotBeNil)
		So(result, ShouldContainSubstring, `svg width="800" height="658"`)
	})

	Convey("SVG should use given width and height", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:"testname",
			Geography: &models.Geography{Topojson:simpleTopology(), IDProperty:"code", NameProperty:"name"},
			Width: 800,
			Height: 600,
		}

		result := RenderSVG(renderRequest)

		So(result, ShouldNotBeNil)
		So(result, ShouldContainSubstring, `svg width="800" height="600"`)
	})
}

func TestSVGContainsClassName(t *testing.T) {

	Convey("SVG should assign class to map regions", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:"testname",
			Geography: &models.Geography{Topojson:simpleTopology(), IDProperty:"code", NameProperty:"name"},
		}

		result := RenderSVG(renderRequest)

		So(result, ShouldNotBeNil)
		So(strings.Count(result, `class="` + REGION_CLASS_NAME + `"`), ShouldEqual, 2)
	})

	Convey("SVG should append to existing class names", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:"testname",
			Geography: &models.Geography{Topojson:simpleTopology(), IDProperty:"code", NameProperty:"name"},
		}
		renderRequest.Geography.Topojson.Objects["simplegeojson"].Geometries[0].Properties["class"] = "foo"

		result := RenderSVG(renderRequest)

		So(result, ShouldNotBeNil)
		So(strings.Count(result, `class="` + REGION_CLASS_NAME + `"`), ShouldEqual, 1)
		So(result, ShouldContainSubstring, `class="foo ` + REGION_CLASS_NAME + `"`)
	})
}

func TestSVGContainsIDs(t *testing.T) {

	Convey("SVG should assign ids to map regions", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:"testname",
			Geography: &models.Geography{Topojson:simpleTopology(), IDProperty:"code", NameProperty:"name"},
		}

		result := RenderSVG(renderRequest)

		So(result, ShouldNotBeNil)
		So(result, ShouldContainSubstring, `id="testname-f1"`)
		So(result, ShouldContainSubstring, `id="testname-f2"`)
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
		So(result, ShouldContainSubstring, `<title>feature 1</title>`)
		So(result, ShouldContainSubstring, `<title>feature 2</title>`)
	})
}

// simpleTopology returns a topology with 2 features: code=f1, name=feature 1; code=f2, name=feature 2
func simpleTopology() *topojson.Topology {
	simpleTopology, _ := topojson.UnmarshalTopology([]byte(`{"type":"Topology","objects":{"simplegeojson":{"type":"GeometryCollection","geometries":[{"type":"Polygon","arcs":[[0]],"properties":{"code":"f1","name":"feature 1"}},{"type":"Polygon","arcs":[[1]],"properties":{"code":"f2","name":"feature 2"}}]}},"arcs":[[[47.13148713111877,9.53216215939578],[47.13148713111877,9.53216215939578],[47.13148713111877,9.53216215939578],[47.13148713111877,9.53216215939578]],[[47.128000259399414,9.52858586376412],[47.132699489593506,9.52858586376412],[47.132699489593506,9.532394934735397],[47.128000259399414,9.532394934735397],[47.128000259399414,9.52858586376412]]],"bbox":[47.128000259399414,9.52858586376412,47.132699489593506,9.532394934735397]}`))
	return simpleTopology
}
