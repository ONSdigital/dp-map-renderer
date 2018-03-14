package geojson2svg

import (
	"io/ioutil"
	"math/rand"
	"os"
	"github.com/ONSdigital/go-ns/log"
	"os/exec"
	"strings"
	"bytes"
	"encoding/base64"
	"fmt"
)

const (
	// ArgSVGFilename is text that will be replaced with the actual svg filename when invoking the PNGConverter executable
	ArgSVGFilename = "<SVG>"
	// ArgPNGFilename is text that will be replaced with name of the png file to write when invoking the PNGConverter executable
	ArgPNGFilename = "<PNG>"
)

// PNGConverter invokes an executable file to convert an svg file to png
type executablePNGConverter struct {
	Executable string
	Arguments  []string
}

// NewPNGConverter creates a new PNGConverter that invokes an executable to perform the conversion.
// Parameters:
// executable - the path to the executable that converts an svg to png.
// arguments - the arguments passed to the executable. These should include:
// 		geojson2svg.ArgSVGFilename as the name of the svg file to convert
// 		geojson2svg.ArgPNGFilename as the name of the png file to create
func NewPNGConverter(executable string, arguments []string) PNGConverter {
	return &executablePNGConverter{Executable: executable, Arguments: arguments}
}

// Convert converts the given svg file to a base64-encoded png
func (exe *executablePNGConverter) Convert(svg []byte) ([]byte, error) {

	tempName := "temp_" + randomString(8)
	tempSVG := tempName + ".svg"
	tempPNG := tempName + ".png"

	defer deleteTemporaryFiles(tempSVG, tempPNG)

	err := ioutil.WriteFile(tempSVG, svg, 0666)
	if err != nil {
		log.Error(err, log.Data{"_message": "Unable to write svg file"})
		return nil, err
	}

	args := make([]string, len(exe.Arguments))
	for i, s := range exe.Arguments {
		args[i] = strings.Replace(s, ArgSVGFilename, tempSVG, -1)
		args[i] = strings.Replace(args[i], ArgPNGFilename, tempPNG, -1)
	}

	cmd := exec.Command(exe.Executable, args...)
	var out bytes.Buffer
	cmd.Stderr = &out
	err = cmd.Run()
	if err != nil {
		log.Error(err, log.Data{"Command": exe.Executable, "arguments": args, "stderr": out.String()})
		return nil, err
	}

	png, err := ioutil.ReadFile(tempPNG)
	if err != nil {
		log.Error(err, log.Data{"_message": "Unable to read png file"})
		return nil, err
	}

	imgBase64Str := base64.StdEncoding.EncodeToString(png)

	return []byte(imgBase64Str), nil
}

// IncludeFallbackImage inserts a foreignObject with a fallback png image, if we have been configured with a PNGConverter.
func (exe *executablePNGConverter) IncludeFallbackImage(attributes string, content string) string {
	svgString := fmt.Sprintf(`<svg %s>%s%s</svg>`, attributes, content, newline)
	png, err := exe.Convert([]byte(svgString))
	pngString := "<p>Unsupported Browser</p>"
	if err == nil {
		pngString = fmt.Sprintf(`<img alt="Fallback map image for older browsers" src="data:image/png;base64,%s" />`, string(png))
	} else {
		log.Error(err, log.Data{"_message": "Unable to include fallback png"})
	}
	svgString = fmt.Sprintf(svgSwitchTemplate, attributes, content, pngString)
	return svgString
}

const svgSwitchTemplate = `<svg %s>
	<switch>
		<g>
%s
		</g>
		<foreignObject>%s</foreignObject>
	</switch>
</svg>`

// randomString creates a random string of length n consisting of upper and lowercase letters
// thanks to https://stackoverflow.com/a/31832326
func randomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// deleteTemporaryFiles checks to see if each of the files exist, and tries to delete them if so.
func deleteTemporaryFiles(files ...string) {
	for _, s := range files {
		if _, err := os.Stat(s); err == nil {
			e := os.Remove(s)
			if e != nil {
				log.Debug(e.Error(), log.Data{"problem": "Unable to delete temporary file", "file": s})
			}
		}
	}
}
