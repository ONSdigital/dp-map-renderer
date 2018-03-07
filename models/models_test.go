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

func TestValidateRenderRequestRejectsMissingFields(t *testing.T) {
	Convey("When a Render request has missing fields, an error is returned", t, func() {
		request := RenderRequest{}
		err := request.ValidateRenderRequest()
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldContainSubstring, "Missing mandatory field(s)")
		So(err.Error(), ShouldContainSubstring, "geography")
		So(err.Error(), ShouldContainSubstring, "data")
	})

	Convey("When a Render request has missing geography fields, an error is returned", t, func() {
		reader := bytes.NewReader(testdata.LoadExampleRequest(t))
		request, _ := CreateRenderRequest(reader)
		request.Geography.Topojson = nil
		request.Geography.IDProperty = ""

		err := request.ValidateRenderRequest()
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldContainSubstring, "Missing mandatory field(s)")
		So(err.Error(), ShouldContainSubstring, "geography.topojson")
		So(err.Error(), ShouldContainSubstring, "geography.id_property")
	})

}

func TestCreateAnalyseRequestFromFile(t *testing.T) {
	Convey("When an analyse request is passed, a valid struct is returned", t, func() {
		reader := bytes.NewReader(testdata.LoadExampleAnalyseRequest(t))
		request, err := CreateAnalyseRequest(reader)

		So(err, ShouldBeNil)
		So(request.ValidateAnalyseRequest(), ShouldBeNil)
		So(request.Geography, ShouldNotBeNil)
		So(len(request.CSV), ShouldBeGreaterThan, 0)
	})

}

func TestCreateAnalyseRequestWithNoBody(t *testing.T) {
	Convey("When an analyse request has no body, an error is returned", t, func() {
		_, err := CreateAnalyseRequest(reader{})
		So(err, ShouldNotBeNil)
		So(err, ShouldEqual, ErrorReadingBody)
	})

	Convey("When an analyse request has an empty body, an error is returned", t, func() {
		filter, err := CreateAnalyseRequest(strings.NewReader("{}"))
		So(err, ShouldNotBeNil)
		So(err, ShouldResemble, ErrorNoData)
		So(filter, ShouldNotBeNil)
	})
}

func TestCreateAnalyseRequestWithInvalidJSON(t *testing.T) {
	Convey("When an analyse request contains json with an invalid syntax, an error is returned", t, func() {
		_, err := CreateAnalyseRequest(strings.NewReader(`{"foo`))
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldResemble, "unexpected end of JSON input")
	})
}

func TestValidateAnalyseRequestRejectsMissingFields(t *testing.T) {
	Convey("When an analyse request has missing fields, an error is returned", t, func() {
		request := AnalyseRequest{}
		err := request.ValidateAnalyseRequest()
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldContainSubstring, "Missing mandatory field(s)")
		So(err.Error(), ShouldContainSubstring, "geography")
		So(err.Error(), ShouldContainSubstring, "csv")
	})

	Convey("When an analyse request has missing geography fields, an error is returned", t, func() {
		request := AnalyseRequest{Geography:&Geography{}, CSV:"foo,bar"}
		err := request.ValidateAnalyseRequest()
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldContainSubstring, "Missing mandatory field(s)")
		So(err.Error(), ShouldNotContainSubstring, "csv")
		So(err.Error(), ShouldContainSubstring, "geography.topojson")
		So(err.Error(), ShouldContainSubstring, "geography.id_property")
	})

}

func TestValidateAnalyseRequestRejectsInvalidValues(t *testing.T) {
	Convey("When an analyse request has indexes below zero, an error is returned", t, func() {
		reader := bytes.NewReader(testdata.LoadExampleAnalyseRequest(t))
		request, _ := CreateAnalyseRequest(reader)
		request.ValueIndex = -1
		request.IDIndex = -2

		err := request.ValidateAnalyseRequest()
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldContainSubstring, "value_index")
		So(err.Error(), ShouldContainSubstring, "id_index")
	})

	Convey("When an analyse request has the same value for value and id indexes, an error is returned", t, func() {
		reader := bytes.NewReader(testdata.LoadExampleAnalyseRequest(t))
		request, _ := CreateAnalyseRequest(reader)
		request.ValueIndex = 0
		request.IDIndex = 0

		err := request.ValidateAnalyseRequest()
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldContainSubstring, "id_index and value_index cannot refer to the same column")
	})

}
