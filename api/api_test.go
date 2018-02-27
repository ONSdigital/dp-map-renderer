package api

import (
	"testing"

	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/ONSdigital/dp-map-renderer/testdata"
	"bytes"
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
			ioutil.WriteFile("../testdata/exampleResponse.html", w.Body.Bytes(), 0644)
		}
	})
}

func TestSuccessfullyRenderEmptyMap(t *testing.T) {
	t.Parallel()
	Convey("Successfully render an html map", t, func() {
		reader := strings.NewReader(requestBody)
		r, err := http.NewRequest("POST", requestHTMLURL, reader)
		So(err, ShouldBeNil)

		w := httptest.NewRecorder()
		api := routes(mux.NewRouter())
		api.router.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, http.StatusOK)
		So(w.Header().Get("Content-Type"), ShouldEqual, "text/html")
		So(w.Body.String(), ShouldContainSubstring, "<svg")
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
