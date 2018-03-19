package analyser_test

import (
	"testing"

	"bytes"

	"github.com/ONSdigital/dp-map-renderer/analyser"
	"github.com/ONSdigital/dp-map-renderer/models"
	"github.com/ONSdigital/dp-map-renderer/testdata"
	. "github.com/smartystreets/goconvey/convey"
)

func TestAnalyseData(t *testing.T) {
	Convey("AnalyseData should parse data correctly", t, func() {

		exampleAnalyseRequest := testdata.LoadExampleAnalyseRequest(t)
		reader := bytes.NewReader(exampleAnalyseRequest)
		request, err := models.CreateAnalyseRequest(reader)
		if err != nil {
			t.Fatal(err)
		}

		result, err := analyser.AnalyseData(request)

		So(err, ShouldBeNil)
		So(result, ShouldNotBeNil)
		So(len(result.Data), ShouldEqual, 415)

		errors := filterMessages(result, "error")
		So(len(errors), ShouldEqual, 1)
		s := errors[0].Text
		So(s, ShouldContainSubstring, "IDs of 42 rows could not be found in the topology")
		So(s, ShouldContainSubstring, "E10000002")

		warnings := filterMessages(result, "warn")
		So(len(warnings), ShouldEqual, 1)
		s = warnings[0].Text
		So(s, ShouldContainSubstring, "7 rows have missing (or non-numeric) values and could not be parsed")
		So(s, ShouldContainSubstring, "E06000053")

		info := filterMessages(result, "info")
		So(len(info), ShouldEqual, 1)
		So(info[0].Text, ShouldContainSubstring, "373") // number of rows parsed successfully
		So(info[0].Text, ShouldContainSubstring, "422") // total number of rows
		So(result.MinValue, ShouldEqual, 0.0)
		So(result.MaxValue, ShouldEqual, 54.0)
		So(len(result.Breaks), ShouldEqual, 10)
		So(result.Breaks[0], ShouldResemble, []float64{0.0, 22.0})
		So(result.Breaks[9], ShouldResemble, []float64{0.0, 4.0, 7.0, 10.0, 13.0, 16.0, 20.0, 26.0, 33.0, 39.0, 46.0})
		So(result.BestFitClassCount, ShouldEqual, 5)

	})

}

func TestAnalyseDataShouldReturnErrorWhenUnableToParse(t *testing.T) {
	Convey("AnalyseData should return an error message and no data when unable to parse csv", t, func() {

		reader := bytes.NewReader(testdata.LoadExampleAnalyseRequest(t))
		request, err := models.CreateAnalyseRequest(reader)
		if err != nil {
			t.Fatal(err)
		}
		request.CSV = ""

		result, err := analyser.AnalyseData(request)

		So(err, ShouldNotBeNil)
		So(result, ShouldBeNil)
	})

}

func TestAnalyseDataShouldReturnErrorWhenNoRowsHaveData(t *testing.T) {
	Convey("AnalyseData should return an error message and no data when no rows have values", t, func() {

		reader := bytes.NewReader(testdata.LoadExampleAnalyseRequest(t))
		request, err := models.CreateAnalyseRequest(reader)
		if err != nil {
			t.Fatal(err)
		}
		request.CSV = "S12000013,Eilean Siar (Western Isles),a\nS12000023,Orkney Islands,b"
		request.HasHeaderRow = false

		result, err := analyser.AnalyseData(request)

		So(err, ShouldNotBeNil)
		So(result, ShouldBeNil)

		So(err.Error(), ShouldContainSubstring, "No CSV rows had a numeric value - could not read data")

	})

}

func TestAnalyseDataShouldReturnErrorWhenDataDoesNotMatchTopology(t *testing.T) {
	Convey("AnalyseData should return an error message and no data when no rows have ids that match the topology", t, func() {

		reader := bytes.NewReader(testdata.LoadExampleAnalyseRequest(t))
		request, err := models.CreateAnalyseRequest(reader)
		if err != nil {
			t.Fatal(err)
		}
		request.CSV = "xxx,Eilean Siar (Western Isles),0\nyyy,Orkney Islands,0"
		request.HasHeaderRow = false

		result, err := analyser.AnalyseData(request)

		So(err, ShouldNotBeNil)
		So(result, ShouldBeNil)

		So(err.Error(), ShouldContainSubstring, "Data does not match Topology - IDs in the data do not match any IDs in the topology")

	})

	Convey("AnalyseData should return an error message and no data when no rows have ids that match the topology - because the geography IDProperty is wrong", t, func() {

		reader := bytes.NewReader(testdata.LoadExampleAnalyseRequest(t))
		request, err := models.CreateAnalyseRequest(reader)
		if err != nil {
			t.Fatal(err)
		}
		request.Geography.IDProperty = "no such property"

		result, err := analyser.AnalyseData(request)

		So(err, ShouldNotBeNil)
		So(result, ShouldBeNil)

		So(err.Error(), ShouldContainSubstring, "Data does not match Topology - IDs in the data do not match any IDs in the topology")

	})

}

func TestAnalyseDataShouldReturnErrorWhenNoRowsHaveEnoughColumns(t *testing.T) {
	Convey("AnalyseData should return an error message and no data when all rows have too few columns", t, func() {

		reader := bytes.NewReader(testdata.LoadExampleAnalyseRequest(t))
		request, err := models.CreateAnalyseRequest(reader)
		if err != nil {
			t.Fatal(err)
		}
		request.CSV = "S12000013,Eilean Siar (Western Isles)\nS12000023,Orkney Islands"
		request.HasHeaderRow = false

		result, err := analyser.AnalyseData(request)

		So(err, ShouldNotBeNil)
		So(result, ShouldBeNil)

		So(err.Error(), ShouldContainSubstring, "All CSV rows had fewer than 3 columns - could not read data")

	})

}

func TestAnalyseDataShouldReturnResponseWithWarnings(t *testing.T) {
	Convey("AnalyseData should returns a response with warnings when some rows have valid data", t, func() {

		reader := bytes.NewReader(testdata.LoadExampleAnalyseRequest(t))
		request, err := models.CreateAnalyseRequest(reader)
		if err != nil {
			t.Fatal(err)
		}
		request.CSV = "S12000013,Eilean Siar (Western Isles),1\nS12000023,Orkney Islands,b\nMyUnknownID,Invalid,2"
		request.HasHeaderRow = false

		result, err := analyser.AnalyseData(request)

		So(err, ShouldBeNil)
		So(result, ShouldNotBeNil)
		So(len(result.Data), ShouldEqual, 2)

		errors := filterMessages(result, "error")
		So(len(errors), ShouldEqual, 1)
		So(errors[0].Text, ShouldContainSubstring, "could not be found in the topology")
		So(errors[0].Text, ShouldContainSubstring, "MyUnknownID")

		warnings := filterMessages(result, "warn")
		So(len(warnings), ShouldEqual, 1)
		So(warnings[0].Text, ShouldContainSubstring, "could not be parsed")
		So(warnings[0].Text, ShouldContainSubstring, "S12000023")

		info := filterMessages(result, "info")
		So(len(info), ShouldEqual, 1)
		So(info[0].Text, ShouldContainSubstring, "1") // number of rows parsed
		So(info[0].Text, ShouldContainSubstring, "3") // total number of rows
	})

}

func TestAnalyseDataShouldReturnResponseWithWarningsForMissingRows(t *testing.T) {
	Convey("AnalyseData should returns a response with warnings when some rows have too few columns", t, func() {

		reader := bytes.NewReader(testdata.LoadExampleAnalyseRequest(t))
		request, err := models.CreateAnalyseRequest(reader)
		if err != nil {
			t.Fatal(err)
		}
		request.CSV = "S12000013,Eilean Siar (Western Isles),1\nS12000023,Orkney Islands\nS12000027,Shetland Islands"
		request.HasHeaderRow = false

		result, err := analyser.AnalyseData(request)

		So(err, ShouldBeNil)
		So(result, ShouldNotBeNil)
		So(len(result.Data), ShouldEqual, 1)

		warnings := filterMessages(result, "warn")
		So(len(warnings), ShouldEqual, 1)
		s := warnings[0].Text
		So(s, ShouldContainSubstring, "rows have missing columns and could not be parsed")
		So(s, ShouldContainSubstring, "[2")
		So(s, ShouldContainSubstring, "3]")

		info := filterMessages(result, "info")
		So(len(info), ShouldEqual, 1)
		So(info[0].Text, ShouldContainSubstring, "1") // number of rows parsed
		So(info[0].Text, ShouldContainSubstring, "3") // total number of rows
	})

}

func filterMessages(response *models.AnalyseResponse, level string) []*models.Message {
	m := []*models.Message{}
	for _, msg := range response.Messages {
		if msg.Level == level {
			m = append(m, msg)
		}
	}
	return m
}
