package geojson2svg_test

import (
	"testing"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/ONSdigital/dp-map-renderer/geojson2svg"
	"encoding/base64"
)

func Test_ConvertShouldFailWhenExecutableDoesNotExist(t *testing.T) {
	Convey("Should invoke (non-existent) executable and return error", t, func() {

		converter := geojson2svg.NewPNGConverter("executableThatDoesNotExist", []string{"<SVG>", "-o", "<PNG>"})
		So(converter, ShouldNotBeNil)

		result, e := converter.Convert([]byte(`<svg><rect height="8" width="8" style="stroke-width: 0.8; stroke: black; fill: Blue;"></rect></svg>`))
		So(e, ShouldNotBeNil)
		So(result, ShouldBeNil)
	})
}

func Test_ConvertShouldInvokeExecutableAndBase64EncodeTheResult(t *testing.T) {
	Convey("Should invoke executable and base 64 encode the output", t, func() {

		converter := geojson2svg.NewPNGConverter("sh", []string{"-c", "cat " + geojson2svg.ArgSVGFilename + " >> " + geojson2svg.ArgPNGFilename})
		So(converter, ShouldNotBeNil)

		result, e := converter.Convert([]byte("MySVG"))
		So(e, ShouldBeNil)
		So(string(result), ShouldResemble, base64.StdEncoding.EncodeToString([]byte("MySVG")))
	})
}
