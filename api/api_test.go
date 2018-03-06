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
)

var (
	host           = "http://localhost:80"
	requestHTMLURL = host + "/render/html"
	requestBody    = `{"title":"map_title", "filename": "file_name", "type":"map_type"}`
)

var saveTestResponse = true

func TestSuccessfullyRenderMap(t *testing.T) {
	t.Parallel()
	Convey("Successfully render an html map", t, func() {
		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		r, err := http.NewRequest("POST", requestHTMLURL, reader)
		So(err, ShouldBeNil)

		w := httptest.NewRecorder()
		api := routes(mux.NewRouter())
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusOK)
		So(w.Header().Get("Content-Type"), ShouldEqual, "text/html")
		So(w.Body.String(), ShouldContainSubstring, "<svg")
		So(w.Body.String(), ShouldContainSubstring, "Non-UK born population, Great Britain, 2015")
		if saveTestResponse {
			s := exampleResponseStart + w.Body.String()
			ioutil.WriteFile("../testdata/exampleResponse.html", []byte(s), 0644)
		}
	})
}

func TestSuccessfullyRenderEmptyMap(t *testing.T) {
	Convey("Successfully render html without an svg", t, func() {
		reader := strings.NewReader(requestBody)
		r, err := http.NewRequest("POST", requestHTMLURL, reader)
		So(err, ShouldBeNil)

		w := httptest.NewRecorder()
		api := routes(mux.NewRouter())
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusOK)
		So(w.Header().Get("Content-Type"), ShouldEqual, "text/html")
		So(w.Body.String(), ShouldNotContainSubstring, "<svg")
		So(w.Body.String(), ShouldContainSubstring, "map_title")
	})
}

func TestRejectInvalidRequest(t *testing.T) {
	t.Parallel()
	Convey("Reject invalid render type in url with StatusNotFound", t, func() {
		reader := strings.NewReader(requestBody)
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
		r, err := http.NewRequest("POST", requestHTMLURL, reader)
		So(err, ShouldBeNil)

		w := httptest.NewRecorder()
		api := routes(mux.NewRouter())
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusBadRequest)

		bodyBytes, _ := ioutil.ReadAll(w.Body)
		response := string(bodyBytes)
		So(response, ShouldResemble, "Bad request - Invalid request body\n")
	})
}

var exampleResponseStart=`
<style>
body {
	font-family: sans-serif;
}
.map__caption {
	font-size: 150%;fill: 
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
<p>This page has additional styling to set the background colour of the svg, highlight region borders on hover,
	<br/>and position the legend(s). There's also javascript to toggle between the 2 legends (horizontal and vertical)
	- the same can be achieved with css alone using min-width and max-width selectors.
</p>
<button onclick="toggleLegend()">Toggle legend position</button>
`