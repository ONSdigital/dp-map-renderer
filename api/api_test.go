package api

import (
	"testing"

	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"

	"bytes"

	"github.com/ONSdigital/dp-map-renderer/testdata"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/ONSdigital/dp-map-renderer/renderer"
	"github.com/ONSdigital/dp-map-renderer/geojson2svg"
)

var (
	host          = "http://localhost:80"
	requestSVGURL = host + "/render/svg"
	requestPNGURL = host + "/render/png"
	analyseURL    = host + "/analyse"
)

var saveTestResponse = true

func TestSuccessfullyRenderSVGMap(t *testing.T) {
	t.Parallel()
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
		if saveTestResponse {
			s := exampleResponseStart + w.Body.String() + exampleResponseEnd
			ioutil.WriteFile("../testdata/exampleResponse.html", []byte(s), 0644)
		}
	})
}

func TestSuccessfullyRenderPNGMap(t *testing.T) {
	t.Parallel()
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
		So(w.Body.String(), ShouldContainSubstring, `src="data:image/png;base64,`)
	})
}

func TestSuccessfullyAnalyseData(t *testing.T) {
	t.Parallel()
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

	Convey("When an invalid json message is sent, a bad request is returned", t, func() {
		reader := strings.NewReader("{")
		r, err := http.NewRequest("POST", requestSVGURL, reader)
		So(err, ShouldBeNil)

		w := httptest.NewRecorder()
		api := routes(mux.NewRouter())
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusBadRequest)

		bodyBytes, _ := ioutil.ReadAll(w.Body)
		response := string(bodyBytes)
		So(response, ShouldResemble, "unexpected end of JSON input\n")
	})
}

var exampleResponseStart = `
<html>
<head>
	<meta charset="UTF-8">
	<style>
	body {
		font-family: sans-serif;
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
	div.map_key__horizontal {
		clear: both;
	}
	.mapRegion:hover {
		stroke: purple;
		stroke-width: 1;
	}
	#abcd1234-legend-horizontal {
		display: none;
	}
	.map svg {
		background-color: LightBlue;
	}
	</style>
	<script type="text/javascript">
	function toggleLegend() {
		vert = document.getElementById("abcd1234-legend-vertical")
		horiz = document.getElementById("abcd1234-legend-horizontal")
		if (( vert.offsetWidth || vert.offsetHeight || vert.getClientRects().length )) {
			vert.style.display = "none"
			horiz.style.display = "block"
		} else {
			horiz.style.display = "none"
			vert.style.display = "block"
		}
	}
	</script>
</head>
<body>
<p>This page has additional styling to set the background colour of the svg, highlight region borders on hover,
	<br/>and position the legend(s). There's also javascript to toggle between the 2 legends (horizontal and vertical)
	- the same can be achieved with css alone using min-width and max-width selectors.
</p>
<button onclick="toggleLegend()">Toggle legend position</button>
`
var exampleResponseEnd = `
</body>
</html>`