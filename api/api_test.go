package api

import (
	"testing"

	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"

	"bytes"

	"github.com/ONSdigital/dp-map-renderer/geojson2svg"
	"github.com/ONSdigital/dp-map-renderer/renderer"
	"github.com/ONSdigital/dp-map-renderer/testdata"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	host          = "http://localhost:80"
	requestSVGURL = host + "/render/svg"
	requestPNGURL = host + "/render/png"
	analyseURL    = host + "/analyse"
)

var saveTestResponse = true

func TestSuccessfullyRenderSVGMap(t *testing.T) {
	Convey("Successfully render an html map with svg images", t, func() {

		renderer.UsePNGConverter(geojson2svg.NewPNGConverter("sh", []string{"-c", "cat testdata/fallback.png >> " + geojson2svg.ArgPNGFilename}))

		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		r, err := http.NewRequest("POST", requestSVGURL, reader)
		So(err, ShouldBeNil)

		w := httptest.NewRecorder()
		api := routes(mux.NewRouter())
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusOK)
		So(w.Header().Get("Content-Type"), ShouldEqual, "text/html")
		So(w.Body.String(), ShouldContainSubstring, "<svg")
		So(w.Body.String(), ShouldContainSubstring, "Non-UK born population, Great Britain, 2015")
		So(w.Body.String(), ShouldNotContainSubstring, "[CSS Here]")
		So(w.Body.String(), ShouldNotContainSubstring, "[javascript Here]")
		if saveTestResponse {
			s := exampleResponseStart + w.Body.String() + exampleResponseEnd
			ioutil.WriteFile("../testdata/exampleResponse.html", []byte(s), 0644)
		}
	})
}

func TestSuccessfullyRenderPNGMap(t *testing.T) {
	Convey("Successfully render an html map with png images", t, func() {

		renderer.UsePNGConverter(geojson2svg.NewPNGConverter("sh", []string{"-c", "cat testdata/fallback.png >> " + geojson2svg.ArgPNGFilename}))

		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		r, err := http.NewRequest("POST", requestPNGURL, reader)
		So(err, ShouldBeNil)

		w := httptest.NewRecorder()
		api := routes(mux.NewRouter())
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusOK)
		So(w.Header().Get("Content-Type"), ShouldEqual, "text/html")
		So(w.Body.String(), ShouldNotContainSubstring, "<svg")
		So(w.Body.String(), ShouldContainSubstring, "<img")
		So(w.Body.String(), ShouldContainSubstring, `width="400"`)
		So(w.Body.String(), ShouldContainSubstring, `src="data:image/png;base64,`)
		So(w.Body.String(), ShouldNotContainSubstring, "[CSS Here]")
		So(w.Body.String(), ShouldNotContainSubstring, "[javascript Here]")
	})
}

func TestSuccessfullyAnalyseData(t *testing.T) {
	Convey("Successfully analyse data and topology", t, func() {
		reader := bytes.NewReader(testdata.LoadExampleAnalyseRequest(t))
		r, err := http.NewRequest("POST", analyseURL, reader)
		So(err, ShouldBeNil)

		w := httptest.NewRecorder()
		api := routes(mux.NewRouter())
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusOK)
		So(w.Header().Get("Content-Type"), ShouldEqual, "application/json")

		if saveTestResponse {
			ioutil.WriteFile("../testdata/exampleAnalyseResponse.json", w.Body.Bytes(), 0644)
		}
	})
}

func TestRejectInvalidRequest(t *testing.T) {
	Convey("Reject invalid render type in url with StatusNotFound", t, func() {
		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		r, err := http.NewRequest("POST", host+"/render/foo", reader)
		So(err, ShouldBeNil)

		w := httptest.NewRecorder()
		api := routes(mux.NewRouter())
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusNotFound)
		So(w.Body.String(), ShouldResemble, "Unknown render type\n")
	})
}

func TestRejectInvalidJSON(t *testing.T) {
	Convey("When an invalid json message is sent, a bad request is returned", t, func() {
		reader := strings.NewReader("{")
		r, err := http.NewRequest("POST", requestSVGURL, reader)
		So(err, ShouldBeNil)

		w := httptest.NewRecorder()
		api := routes(mux.NewRouter())
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusBadRequest)
	})
}

var exampleResponseStart = `
<html>
<head>
	<meta charset="UTF-8">
	<style>
	body {
		font-family: "Open Sans", Helvetica, Arial, sans-serif;
		font-size: 14px;
		font-weight: 400;
	}
	.map__caption {
		font-size: 150%; 
		font-weight: bold;
	}
	.map__subtitle {
		font-size: 75%;
	}
	div.map_key__vertical, div.map {
		display: inline-block;
	}
	.mapRegion {
		stroke: #323132;
		stroke-width: 0.5;
	}
	.mapRegion:hover {
		stroke: purple;
		stroke-width: 1;
	}
	</style>
	<script type="text/javascript" src="http://ariutta.github.io/svg-pan-zoom/dist/svg-pan-zoom.min.js"></script>
</head>
<body>
<p>This page imports the svg-pan-zoom library, which is then used by the map-renderer output to enable panning and zooming.
The renderer output also includes a style block to support responsive resizing.
</p>
`
var exampleResponseEnd = `
<script type="text/javascript" src="https://cdn.ons.gov.uk/vendor/svg-pan-zoom/3.5.2/svg-pan-zoom.min.js"></script>
<script type="text/javascript">
	document.addEventListener("DOMContentLoaded", function() {
		var mapId = "map-abcd1234-map-svg"
		var svg = document.getElementById(mapId);
		if (svg && svg.clientWidth > 0 && svg.hasAttribute("viewBox")) {
			viewBox = svg.getAttribute("viewBox").split(" ") // x1 y1 x2 y2
			heightRatio = parseInt(viewBox[3]) / parseInt(viewBox[2])
			var setSvgHeight = function() {	
				svg.style.height = Math.round(svg.clientWidth * heightRatio) + "px"
				return true;
			};
			setSvgHeight();
			var panZoom = window.panZoom = svgPanZoom('#' + mapId, {minZoom: 0.75, maxZoom: 100, zoomScaleSensitivity: 0.4, mouseWheelZoomEnabled: false, controlIconsEnabled: true, fit: true, center: true});
	
			window.addEventListener('resize', function(){
				setSvgHeight()
				panZoom.resize();
				panZoom.fit();
				panZoom.center();
			});
		}
	});
</script>
</body>
</html>`
