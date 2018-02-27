package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/ONSdigital/go-ns/log"
	"github.com/rubenv/topojson"
)

// A list of errors returned from package
var (
	ErrorReadingBody = errors.New("Failed to read message body")
	ErrorParsingBody = errors.New("Failed to parse json body")
	ErrorNoData      = errors.New("Bad request - Missing data in body")
)

// RenderRequest represents a structure for a map render job
type RenderRequest struct {
	Title      string     `json:"title,omitempty"`
	Subtitle   string     `json:"subtitle,omitempty"`
	Source     string     `json:"source,omitempty"`
	SourceLink string     `json:"source_link,omitempty"`
	Filename   string     `json:"filename,omitempty"`
	Footnotes  []string   `json:"footnotes,omitempty"`
	MapType    string     `json:"map_type,omitempty"`
	Geography  Geography  `json:"geography,omitempty"`
	Data       []DataRow  `json:"data,omitempty"`
	Choropleth Choropleth `json:"choropleth,omitempty"`
	Width      float64    `json:"width,omitempty"`
	Height     float64    `json:"height,omitempty"`
}

// Geography holds the topojson topology and supporting information
type Geography struct {
	Topojson     topojson.Topology `json:"topojson,omitempty"`
	IDProperty   string            `json:"id_property,omitempty"`
	NameProperty string            `json:"name_property,omitempty"`
}

// DataRow holds a single row of data.
type DataRow struct {
	ID    string  `json:"id,omitempty"`
	value float64 `json:"value,omitempty"`
}

// Choropleth contains details required to create a choropleth map
type Choropleth struct {
	ReferenceValue     float64           `json:"reference_value,omitempty"`
	ReferenceValueText string            `json:"reference_value_text,omitempty"`
	ValuePrefix        string            `json:"value_prefix,omitempty"`
	ValueSuffix        string            `json:"value_suffix,omitempty"`
	Breaks             []ChoroplethBreak `json:"breaks,omitempty"`
	MissingValueColor  string            `json:"missing_value_color,omitempty"`
}

// ChoroplethBreak represents a single break - the point at which a colour changes
type ChoroplethBreak struct {
	LowerBound float64 `json:"lower_bound,omitempty"` // the lower bound for this colour
	Color      string  `json:"color,omitempty"`
}

// CreateRenderRequest manages the creation of a RenderRequest from a reader
func CreateRenderRequest(reader io.Reader) (*RenderRequest, error) {
	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Error(err, log.Data{"request_body": string(bytes)})
		return nil, ErrorReadingBody
	}
	log.Debug("Render Request: ", log.Data{"request_body": string(bytes)})

	var request RenderRequest
	err = json.Unmarshal(bytes, &request)
	if err != nil {
		log.Error(err, log.Data{"request_body": string(bytes)})
		return nil, ErrorParsingBody
	}

	// This should be the last check before returning RenderRequest
	if len(bytes) == 2 {
		return &request, ErrorNoData
	}

	return &request, nil
}

// ValidateRenderRequest checks the content of the request structure
func (rr *RenderRequest) ValidateRenderRequest() error {

	var missingFields []string

	// TODO: make mandatory fields: width, geography.topojson
	if missingFields != nil {
		return fmt.Errorf("Missing mandatory fields: %v", missingFields)
	}

	return nil
}
