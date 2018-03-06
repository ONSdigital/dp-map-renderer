package models

import (
	"fmt"
	"strings"
	"testing"

	"bytes"

	"github.com/ONSdigital/dp-map-renderer/testdata"
	. "github.com/smartystreets/goconvey/convey"
)

// A Mock io.reader to trigger errors on reading
type reader struct {
}

func (f reader) Read(bytes []byte) (int, error) {
	return 0, fmt.Errorf("Reader failed")
}

func TestCreateRenderRequestWithValidJSON(t *testing.T) {
	Convey("When a render request has a minimally valid json body, a valid struct is returned", t, func() {
		reader := strings.NewReader(`{"title":"map_title", "filename":"filename"}`)
		request, err := CreateRenderRequest(reader)

		So(err, ShouldBeNil)
		So(request.ValidateRenderRequest(), ShouldBeNil)
		So(request.Title, ShouldEqual, "map_title")
		So(request.Filename, ShouldEqual, "filename")
	})

}

func TestCreateRenderRequestFromFile(t *testing.T) {
	Convey("When a render request is passed, a valid struct is returned", t, func() {
		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		request, err := CreateRenderRequest(reader)

		So(err, ShouldBeNil)
		So(request.ValidateRenderRequest(), ShouldBeNil)
		So(request.Title, ShouldEqual, "Non-UK born population, Great Britain, 2015")
		So(request.Filename, ShouldEqual, "abcd1234")
	})

}

func TestCreateRenderRequestWithNoBody(t *testing.T) {
	Convey("When a render request has no body, an error is returned", t, func() {
		_, err := CreateRenderRequest(reader{})
		So(err, ShouldNotBeNil)
		So(err, ShouldEqual, ErrorReadingBody)
	})

	Convey("When a render request has an empty body, an error is returned", t, func() {
		filter, err := CreateRenderRequest(strings.NewReader("{}"))
		So(err, ShouldNotBeNil)
		So(err, ShouldResemble, ErrorNoData)
		So(filter, ShouldNotBeNil)
	})
}

func TestCreateRenderRequestWithInvalidJSON(t *testing.T) {
	Convey("When a render request contains json with an invalid syntax, and error is returned", t, func() {
		_, err := CreateRenderRequest(strings.NewReader(`{"foo`))
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldResemble, "unexpected end of JSON input")
	})
}
