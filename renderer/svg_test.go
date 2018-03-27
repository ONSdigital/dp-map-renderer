package renderer_test

import (
	"bytes"
	"testing"

	"encoding/xml"
	"fmt"

	"regexp"
	"strconv"

	"github.com/ONSdigital/dp-map-renderer/geojson2svg"
	"github.com/ONSdigital/dp-map-renderer/models"
	. "github.com/ONSdigital/dp-map-renderer/renderer"
	"github.com/ONSdigital/dp-map-renderer/testdata"
	"github.com/rubenv/topojson"
	. "github.com/smartystreets/goconvey/convey"
)

var pngConverter = geojson2svg.NewPNGConverter("sh", []string{"-c", `echo "test" >> ` + geojson2svg.ArgPNGFilename})
var expectedFallbackImage = `<img alt="Fallback map image for older browsers" src="data:image/png;base64,dGVzdAo=" />`

func TestRenderSVGWithFixedSize(t *testing.T) {

	Convey("Successfully render an svg map", t, func() {
		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}
		renderRequest.DefaultWidth = 400.0
		renderRequest.MaxWidth = 0
		renderRequest.MinWidth = 0

		result := RenderSVG(PrepareSVGRequest(renderRequest))

		So(result, ShouldNotBeNil)
		So(result, ShouldStartWith, `<svg width="400" height="748" id="abcd1234-map-svg" viewBox="0 0 400 748">`)
	})
}

func TestRenderSVGWithResponsiveSize(t *testing.T) {

	Convey("Successfully render an svg map", t, func() {
		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}
		renderRequest.DefaultWidth = 0
		renderRequest.MaxWidth = 300
		renderRequest.MinWidth = 500

		result := RenderSVG(PrepareSVGRequest(renderRequest))

		So(result, ShouldNotBeNil)
		So(result, ShouldStartWith, `<svg id="abcd1234-map-svg" style="width:100%;" viewBox="0 0 400 748">`)
	})
}

func TestRenderSVGDoesNotIncludeFallbackPng(t *testing.T) {

	Convey("Successfully render an svg map without fallback png", t, func() {

		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}

		result := RenderSVG(PrepareSVGRequest(renderRequest))

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
		renderRequest.IncludeFallbackPng = true

		result := RenderSVG(PrepareSVGRequest(renderRequest))

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

		result := RenderSVG(PrepareSVGRequest(renderRequest))

		So(result, ShouldEqual, "")
	})

	Convey("RenderSVG should not fail with null Topology", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:  "testname",
			Geography: &models.Geography{},
		}

		result := RenderSVG(PrepareSVGRequest(renderRequest))

		So(result, ShouldEqual, "")
	})

	Convey("RenderSVG should not fail with null Topology.Objects", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:  "testname",
			Geography: &models.Geography{Topojson: simpleTopology()},
		}
		renderRequest.Geography.Topojson.Objects = nil

		result := RenderSVG(PrepareSVGRequest(renderRequest))

		So(result, ShouldEqual, "")
	})

	Convey("RenderSVG should not fail with null Topology.Arcs", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:  "testname",
			Geography: &models.Geography{Topojson: simpleTopology()},
		}
		renderRequest.Geography.Topojson.Arcs = nil

		result := RenderSVG(PrepareSVGRequest(renderRequest))

		So(result, ShouldEqual, "")
	})
}

func TestSVGIgnoresNilFeatureNames(t *testing.T) {

	Convey("Rendered svg should not include 'nil' in the title when the topology doesn't have the name property", t, func() {
		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}
		renderRequest.Geography.NameProperty = "missing"

		result := RenderSVG(PrepareSVGRequest(renderRequest))

		So(result, ShouldNotContainSubstring, "<nil>")
		So(result, ShouldContainSubstring, "<title> 7% non-UK born")
	})
}

func TestSVGHasWidthAndHeight(t *testing.T) {

	Convey("simpleSVG should be given default width and proportional height", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:  "testname",
			Geography: &models.Geography{Topojson: simpleTopology(), IDProperty: "code", NameProperty: "name"},
		}

		result := RenderSVG(PrepareSVGRequest(renderRequest))

		svg, e := unmarshalSimpleSVG(result)
		So(e, ShouldBeNil)
		So(svg.Width, ShouldEqual, "400")
		So(svg.Height, ShouldEqual, "329")
	})
}

func TestSVGContainsClassName(t *testing.T) {

	Convey("simpleSVG should assign class to map regions", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:  "testname",
			Geography: &models.Geography{Topojson: simpleTopology(), IDProperty: "code", NameProperty: "name"},
		}
		renderRequest.Geography.Topojson.Objects["simplegeojson"].Geometries[0].Properties["class"] = "foo"

		result := RenderSVG(PrepareSVGRequest(renderRequest))

		svg, e := unmarshalSimpleSVG(result)
		So(e, ShouldBeNil)
		So(len(svg.Paths), ShouldEqual, 2)
		So(svg.Paths[0].Class, ShouldEqual, RegionClassName+" foo")
		So(svg.Paths[1].Class, ShouldEqual, RegionClassName)
	})
}

func TestSVGContainsIDs(t *testing.T) {

	Convey("simpleSVG should assign ids to map regions", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:  "testname",
			Geography: &models.Geography{Topojson: simpleTopology(), IDProperty: "code", NameProperty: "name"},
		}

		result := RenderSVG(PrepareSVGRequest(renderRequest))

		svg, e := unmarshalSimpleSVG(result)
		So(e, ShouldBeNil)
		So(len(svg.Paths), ShouldEqual, 2)
		So(svg.Paths[0].ID, ShouldEqual, "testname-f0")
		So(svg.Paths[1].ID, ShouldEqual, "testname-f1")
	})
}

func TestSVGContainsTitles(t *testing.T) {

	Convey("simpleSVG should assign names as titles to map regions", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:  "testname",
			Geography: &models.Geography{Topojson: simpleTopology(), IDProperty: "code", NameProperty: "name"},
		}

		result := RenderSVG(PrepareSVGRequest(renderRequest))

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

		renderRequest := &models.RenderRequest{
			Filename:   "testname",
			Geography:  &models.Geography{Topojson: simpleTopology(), IDProperty: "code", NameProperty: "name"},
			Choropleth: &models.Choropleth{Breaks: []*models.ChoroplethBreak{{LowerBound: 0, Colour: "red"}, {LowerBound: 11, Colour: "green"}}},
			Data:       []*models.DataRow{{ID: "f0", Value: 10}, {ID: "f1", Value: 20}},
		}

		result := RenderSVG(PrepareSVGRequest(renderRequest))

		So(result, ShouldNotBeNil)
		svg, e := unmarshalSimpleSVG(result)
		So(e, ShouldBeNil)
		So(len(svg.Paths), ShouldEqual, 2)
		So(svg.Paths[0].Style, ShouldContainSubstring, "fill: red")
		So(svg.Paths[1].Style, ShouldContainSubstring, "fill: green")
	})
}

func TestSVGHasMissingValuePatternAndCorrectTitle(t *testing.T) {

	Convey("simpleSVG should use style to colour regions, applying style to regions missing data, and modify the title with values", t, func() {

		renderRequest := &models.RenderRequest{
			Filename:  "testname",
			Geography: &models.Geography{Topojson: simpleTopology(), IDProperty: "code", NameProperty: "name"},
			Choropleth: &models.Choropleth{
				Breaks:      []*models.ChoroplethBreak{{LowerBound: 0, Colour: "red"}, {LowerBound: 11, Colour: "green"}},
				ValuePrefix: "prefix-",
				ValueSuffix: "-suffix"},
			Data: []*models.DataRow{{ID: "f1", Value: 20}},
		}

		result := RenderSVG(PrepareSVGRequest(renderRequest))

		So(result, ShouldNotBeNil)
		So(result, ShouldContainSubstring, `<defs><pattern id="testname-nodata"`)
		svg, e := unmarshalSimpleSVG(result)
		So(e, ShouldBeNil)
		So(len(svg.Paths), ShouldEqual, 2)
		So(svg.Paths[0].Style, ShouldContainSubstring, "fill: url(#testname-nodata);")
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

		result := RenderVerticalKey(PrepareSVGRequest(renderRequest))

		So(result, ShouldNotBeNil)
		So(result, ShouldStartWith, `<svg id="abcd1234-legend-vertical-svg" class="map_key_vertical`)
		So(getWidth(result), ShouldEqual, 122)
		assertKeyContents(result, renderRequest)

	})
}

func TestRenderVerticalKeyWithoutReferenceValue(t *testing.T) {
	Convey("RenderVerticalKey should not render any reference tick", t, func() {

		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}
		renderRequest.Choropleth.ReferenceValue = 0
		renderRequest.Choropleth.ReferenceValueText = ""

		result := RenderVerticalKey(PrepareSVGRequest(renderRequest))

		So(result, ShouldNotBeNil)
		So(result, ShouldStartWith, `<svg id="abcd1234-legend-vertical-svg" class="map_key_vertical`)
		So(getWidth(result), ShouldEqual, 122)

	})
}

func TestRenderHorizontalKey(t *testing.T) {
	Convey("RenderHorizontalKey should render an svg", t, func() {

		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}

		result := RenderHorizontalKey(PrepareSVGRequest(renderRequest))

		So(result, ShouldNotBeNil)
		So(result, ShouldStartWith, `<svg id="abcd1234-legend-horizontal-svg" class="map_key_horizontal`)
		So(result, ShouldContainSubstring, ` viewBox="0 0 400 90"`)
		So(result, ShouldContainSubstring, `<text x="200.000000" y="6" dy=".5em" style="text-anchor: middle;" class="keyText">`)
		So(result, ShouldContainSubstring, `<g id="abcd1234-legend-horizontal-key" transform="translate(20.000000, 20)">`)
		So(result, ShouldContainSubstring, `<g class="map__tick" transform="translate(360.000000, 0)">`)
		assertKeyContents(result, renderRequest)
	})

}

func TestRenderHorizontalKeyWithLongTitle(t *testing.T) {
	Convey("RenderHorizontalKey should render an svg and adjust title text to fit within the bounds", t, func() {

		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}
		renderRequest.Choropleth.ValuePrefix = "This is a long-ish prefix"
		renderRequest.Choropleth.ValueSuffix = "This is a longer suffix. It needs to be wider than the svg to test that the text is compressed"

		result := RenderHorizontalKey(PrepareSVGRequest(renderRequest))

		So(result, ShouldNotBeNil)
		So(result, ShouldStartWith, `<svg id="abcd1234-legend-horizontal-svg" class="map_key_horizontal`)
		So(result, ShouldContainSubstring, ` viewBox="0 0 400 90"`)
		So(result, ShouldContainSubstring, `<text x="200.000000" y="6" dy=".5em" style="text-anchor: middle;" class="keyText" textLength="398" lengthAdjust="spacingAndGlyphs">`)
		So(result, ShouldContainSubstring, `<g id="abcd1234-legend-horizontal-key" transform="translate(20.000000, 20)">`)
		So(result, ShouldContainSubstring, `<g class="map__tick" transform="translate(360.000000, 0)">`)
		assertKeyContents(result, renderRequest)
	})

}

func TestRenderHorizontalKeyWithLongReferenceText(t *testing.T) {
	Convey("RenderHorizontalKey should render an svg and adjust reference text position to maximise use of space", t, func() {

		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}
		renderRequest.Choropleth.ReferenceValueText = "This is a longer bit of text"

		result := RenderHorizontalKey(PrepareSVGRequest(renderRequest))

		So(result, ShouldNotBeNil)
		So(result, ShouldStartWith, `<svg id="abcd1234-legend-horizontal-svg" class="map_key_horizontal`)
		So(result, ShouldContainSubstring, ` viewBox="0 0 400 90"`)
		So(result, ShouldContainSubstring, `<text x="200.000000" y="6" dy=".5em" style="text-anchor: middle;" class="keyText">`)
		So(result, ShouldContainSubstring, `<g id="abcd1234-legend-horizontal-key" transform="translate(20.000000, 20)">`)
		So(result, ShouldContainSubstring, `<g class="map__tick" transform="translate(360.000000, 0)">`)
		assertKeyContents(result, renderRequest)
	})

}

func TestRenderHorizontalKeyWithLongerReferenceTextOnLeft(t *testing.T) {
	Convey("RenderHorizontalKey should render an svg and adjust the key width to accommodate long reference text", t, func() {

		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}
		renderRequest.Choropleth.ReferenceValue = 28
		renderRequest.Choropleth.ReferenceValueText = "This is a much longer bit of text that will shorten the key"

		result := RenderHorizontalKey(PrepareSVGRequest(renderRequest))

		So(result, ShouldNotBeNil)
		So(result, ShouldStartWith, `<svg id="abcd1234-legend-horizontal-svg" class="map_key_horizontal`)
		So(result, ShouldContainSubstring, ` viewBox="0 0 400 90"`)
		So(result, ShouldContainSubstring, `<text x="200.000000" y="6" dy=".5em" style="text-anchor: middle;" class="keyText">`)
		So(result, ShouldContainSubstring, `<g id="abcd1234-legend-horizontal-key" transform="translate(164.010933, 20)">`)
		So(result, ShouldContainSubstring, `<g class="map__tick" transform="translate(228.588667, 0)">`)
		assertKeyContents(result, renderRequest)
	})

}

func TestRenderHorizontalKeyWithLongerReferenceTextOnRight(t *testing.T) {
	Convey("RenderHorizontalKey should render an svg and adjust the key width to accommodate long reference text", t, func() {

		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}
		renderRequest.Choropleth.ReferenceValue = 13
		renderRequest.Choropleth.ReferenceValueText = "This is a much longer bit of text that will shorten the key"

		result := RenderHorizontalKey(PrepareSVGRequest(renderRequest))

		So(result, ShouldNotBeNil)
		So(result, ShouldStartWith, `<svg id="abcd1234-legend-horizontal-svg" class="map_key_horizontal`)
		So(result, ShouldContainSubstring, ` viewBox="0 0 400 90"`)
		So(result, ShouldContainSubstring, `<text x="200.000000" y="6" dy=".5em" style="text-anchor: middle;" class="keyText">`)
		So(result, ShouldContainSubstring, `<g id="abcd1234-legend-horizontal-key" transform="translate(3.700200, 20)">`)
		So(result, ShouldContainSubstring, `<g class="map__tick" transform="translate(318.955533, 0)">`)
		assertKeyContents(result, renderRequest)
	})

}

func TestRenderHorizontalKeyClassChangesWhenVerticalKeyAlsoPresent(t *testing.T) {
	Convey("RenderHorizontalKey should include an additional class when vertical key also present", t, func() {

		UsePNGConverter(pngConverter)

		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}
		renderRequest.Choropleth.HorizontalLegendPosition = models.LegendPositionBefore
		renderRequest.Choropleth.VerticalLegendPosition = models.LegendPositionBefore

		result := RenderHorizontalKey(PrepareSVGRequest(renderRequest))

		So(result, ShouldNotBeNil)
		So(result, ShouldContainSubstring, `class="map_key_horizontal map_key_horizontal_both"`)
	})

	Convey("RenderHorizontalKey should not include additional class when vertical key absent", t, func() {

		UsePNGConverter(pngConverter)

		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}
		renderRequest.Choropleth.VerticalLegendPosition = "none"
		renderRequest.Choropleth.HorizontalLegendPosition = models.LegendPositionBefore

		result := RenderHorizontalKey(PrepareSVGRequest(renderRequest))

		So(result, ShouldNotBeNil)
		So(result, ShouldContainSubstring, `class="map_key_horizontal"`)
	})

}

func TestRenderVerticalKeyClassChangesWhenHorizontalKeyAlsoPresent(t *testing.T) {
	Convey("RenderVerticalKey should include an additional class when horizontal key also present", t, func() {

		UsePNGConverter(pngConverter)

		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}
		renderRequest.Choropleth.VerticalLegendPosition = models.LegendPositionBefore
		renderRequest.Choropleth.HorizontalLegendPosition = models.LegendPositionBefore

		result := RenderVerticalKey(PrepareSVGRequest(renderRequest))

		So(result, ShouldNotBeNil)
		So(result, ShouldContainSubstring, `class="map_key_vertical map_key_vertical_both"`)
	})

	Convey("RenderVerticalKey should not include additional class when horizontal key absent", t, func() {

		UsePNGConverter(pngConverter)

		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}
		renderRequest.Choropleth.VerticalLegendPosition = models.LegendPositionBefore
		renderRequest.Choropleth.HorizontalLegendPosition = "none"

		result := RenderVerticalKey(PrepareSVGRequest(renderRequest))

		So(result, ShouldNotBeNil)
		So(result, ShouldContainSubstring, `class="map_key_vertical"`)
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
		renderRequest.IncludeFallbackPng = true

		result := RenderHorizontalKey(PrepareSVGRequest(renderRequest))

		So(result, ShouldNotBeNil)
		So(result, ShouldContainSubstring, `<foreignObject>`)
		So(result, ShouldContainSubstring, expectedFallbackImage)

	})

}

func TestRenderHorizontalKeyDoesNotHaveFallbackPng(t *testing.T) {
	Convey("RenderHorizontalKey should not render a fallback png", t, func() {

		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}
		renderRequest.IncludeFallbackPng = false

		result := RenderHorizontalKey(PrepareSVGRequest(renderRequest))

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
		renderRequest.IncludeFallbackPng = true

		result := RenderVerticalKey(PrepareSVGRequest(renderRequest))

		So(result, ShouldNotBeNil)
		So(result, ShouldContainSubstring, `<foreignObject>`)
		So(result, ShouldContainSubstring, expectedFallbackImage)

	})

}

func TestRenderVerticalKeyDoesNotHaveFallbackPng(t *testing.T) {
	Convey("RenderVerticalKey should not render a fallback png", t, func() {

		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}
		renderRequest.IncludeFallbackPng = false

		result := RenderVerticalKey(PrepareSVGRequest(renderRequest))

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

		result := RenderHorizontalKey(PrepareSVGRequest(renderRequest))

		So(result, ShouldNotBeNil)
		So(result, ShouldNotContainSubstring, `>54<`)
		So(result, ShouldContainSubstring, `>53<`)

	})

}

func TestRenderVerticalKeyWidth(t *testing.T) {
	Convey("RenderVerticalKey should adjust width to acommodate the text", t, func() {

		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		renderRequest, err := models.CreateRenderRequest(reader)
		if err != nil {
			t.Fatal(err)
		}
		renderRequest.Choropleth.ValueSuffix = "short text"

		result := RenderVerticalKey(PrepareSVGRequest(renderRequest))

		defaultWidth := getWidth(result)

		Convey("A long title should cause the width to increase", func() {
			renderRequest.Choropleth.ValueSuffix = "some text long enough to increase the width...."
			width := getWidth(RenderVerticalKey(PrepareSVGRequest(renderRequest)))

			So(width, ShouldBeGreaterThan, defaultWidth)

		})

		Convey("A long reference text should cause the width to increase", func() {
			renderRequest.Choropleth.ValueSuffix = "short text"
			renderRequest.Choropleth.ReferenceValueText = "some text long enough to increase the width..."
			width := getWidth(RenderVerticalKey(PrepareSVGRequest(renderRequest)))

			So(width, ShouldBeGreaterThan, defaultWidth)

		})

		Convey("Long tick values should cause the width to increase", func() {
			renderRequest.Choropleth.ValueSuffix = "short text"
			renderRequest.Choropleth.ReferenceValueText = ".."
			renderRequest.Choropleth.Breaks[0].LowerBound = 123456.123456
			renderRequest.Choropleth.ReferenceValue = 123456.123456
			width := getWidth(RenderVerticalKey(PrepareSVGRequest(renderRequest)))

			So(width, ShouldBeGreaterThan, defaultWidth)

		})
	})

}

func assertKeyContents(result string, renderRequest *models.RenderRequest) {
	So(result, ShouldContainSubstring, renderRequest.Choropleth.ValuePrefix)
	So(result, ShouldContainSubstring, renderRequest.Choropleth.ValueSuffix)
	for _, b := range renderRequest.Choropleth.Breaks {
		So(result, ShouldContainSubstring, "fill: "+b.Colour)
		So(result, ShouldContainSubstring, fmt.Sprintf("%g", b.LowerBound))
	}
	So(result, ShouldContainSubstring, "fill: url(#"+renderRequest.Filename+"-nodata)")
	So(result, ShouldContainSubstring, MissingDataText)
	// ensure all text has a class applied
	textElements := regexp.MustCompile("<text").FindAllStringIndex(result, -1)
	withClass := regexp.MustCompile(`<text[^>]*class="[^"]*keyText[^>]*"`).FindAllStringIndex(result, -1)
	So(len(withClass), ShouldEqual, len(textElements))
	// look for the reference text if it should be present
	referenceTick := regexp.MustCompile(`<line [^>]*stroke: DimGrey;[^>]*></line>`).FindString(result)
	if len(renderRequest.Choropleth.ReferenceValueText) == 0 {
		So(len(referenceTick), ShouldEqual, 0)
	} else {
		So(len(referenceTick), ShouldBeGreaterThan, 0)
		So(result, ShouldContainSubstring, renderRequest.Choropleth.ReferenceValueText)
	}

}

func getWidth(result string) int {
	widthRE := regexp.MustCompile(`viewBox="0 0 ([\d]+) \d+"`)
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
