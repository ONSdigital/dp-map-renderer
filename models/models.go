package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/ONSdigital/go-ns/log"
	"github.com/json-iterator/go"
	"github.com/rubenv/topojson"
)

// A list of errors returned from package
var (
	ErrorReadingBody = errors.New("Failed to read message body")
	ErrorNoData      = errors.New("Bad request - Missing data in body")
)

// possible values for the 2 LegendPositions. 'None' is the default.
var (
	LegendPositionBefore = "before"
	LegendPositionAfter  = "after"
)

// RenderRequest represents a structure for a map render job
type RenderRequest struct {
	Title              string      `json:"title,omitempty"`
	Subtitle           string      `json:"subtitle,omitempty"`
	Source             string      `json:"source,omitempty"`
	SourceLink         string      `json:"source_link,omitempty"`
	Licence            string      `json:"licence,omitempty"`
	Filename           string      `json:"filename,omitempty"`
	Footnotes          []string    `json:"footnotes,omitempty"`
	MapType            string      `json:"map_type,omitempty"`
	Geography          *Geography  `json:"geography,omitempty"`
	Data               []*DataRow  `json:"data,omitempty"` // ID's in Data should match values of IDProperty in Geography
	Choropleth         *Choropleth `json:"choropleth,omitempty"`
	DefaultWidth       float64     `json:"width,omitempty"`     // used when determining the viewBox dimensions and the switch point between displaying the horizontal and vertical legends in responsive design. Optional if min and max width specified
	MinWidth           float64     `json:"min_width,omitempty"` // the minimum width in a responsive design. optional.
	MaxWidth           float64     `json:"max_width,omitempty"` // the maximum width in a responsive design. Required if min width specified.
	IncludeFallbackPng bool        `json:"include_fallback_png"`
	FontSize           int         `json:"font_size"`
}

// Geography holds the topojson topology and supporting information
type Geography struct {
	Topojson     *topojson.Topology `json:"topojson,omitempty"`
	IDProperty   string             `json:"id_property,omitempty"`
	NameProperty string             `json:"name_property,omitempty"`
}

// DataRow holds a single row of data.
type DataRow struct {
	ID    string  `json:"id,omitempty"`
	Value float64 `json:"value,omitempty"`
}

// Choropleth contains details required to create a choropleth map
type Choropleth struct {
	ReferenceValue           float64            `json:"reference_value,omitempty"`
	ReferenceValueText       string             `json:"reference_value_text,omitempty"`
	ValuePrefix              string             `json:"value_prefix,omitempty"`
	ValueSuffix              string             `json:"value_suffix,omitempty"`
	Breaks                   []*ChoroplethBreak `json:"breaks,omitempty"`
	UpperBound               float64            `json:"upper_bound,omitempty"`                 // used only in displaying the upperbound in the legend
	HorizontalLegendPosition string             `json:"horizontal_legend_position, omitempty"` // before, after or none (the default)
	VerticalLegendPosition   string             `json:"vertical_legend_position, omitempty"`   // before, after or none (the default)
}

// ChoroplethBreak represents a single break - the point at which a colour changes
type ChoroplethBreak struct {
	LowerBound float64 `json:"lower_bound"` // the lower bound for this colour
	Colour     string  `json:"color,omitempty"`
}

// AnalyseRequest represents the structure of a request to analyse data and ensure it matches a topology
type AnalyseRequest struct {
	Geography    *Geography `json:"geography"`
	CSV          string     `json:"csv"`
	IDIndex      int        `json:"id_index"`
	ValueIndex   int        `json:"value_index"`
	HasHeaderRow bool       `json:"has_header_row"`
}

// AnalyseResponse represents the structure of an analyse data response
type AnalyseResponse struct {
	Data              []*DataRow  `json:"data"`
	Messages          []*Message  `json:"messages"`
	Breaks            [][]float64 `json:"breaks"`
	BestFitClassCount int         `json:"best_fit_class_count"`
	MinValue          float64     `json:"min_value"`
	MaxValue          float64     `json:"max_value"`
}

// Message represents a message with a level type
type Message struct {
	Level string `json:"level"`
	Text  string `json:"text"`
}

// CreateRenderRequest manages the creation of a RenderRequest from a reader
func CreateRenderRequest(reader io.Reader) (*RenderRequest, error) {

	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Error(err, log.Data{"request_body": string(bytes)})
		return nil, ErrorReadingBody
	}

	var request RenderRequest
	err = jsoniter.Unmarshal(bytes, &request)
	if err != nil {
		log.Error(err, log.Data{"request_body": string(bytes)})
		return nil, err
	}

	// This should be the last check before returning RenderRequest
	if len(bytes) == 2 {
		return &request, ErrorNoData
	}

	return &request, nil
}

// ValidateRenderRequest checks the content of the request structure
func (r *RenderRequest) ValidateRenderRequest() error {

	var missingFields []string

	if r.Geography == nil {
		missingFields = append(missingFields, "geography")
	} else {
		if r.Geography.Topojson == nil {
			missingFields = append(missingFields, "geography.topojson")
		}
		if len(r.Geography.IDProperty) == 0 {
			missingFields = append(missingFields, "geography.id_property")
		}
	}

	if len(r.Data) == 0 {
		missingFields = append(missingFields, "data")
	}

	if missingFields != nil {
		return fmt.Errorf("Missing mandatory field(s): %v", missingFields)
	}

	return nil
}

// CreateAnalyseRequest manages the creation of an AnalyseRequest from a reader
func CreateAnalyseRequest(reader io.Reader) (*AnalyseRequest, error) {
	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Error(err, log.Data{"request_body": string(bytes)})
		return nil, ErrorReadingBody
	}

	var request AnalyseRequest
	err = json.Unmarshal(bytes, &request)
	if err != nil {
		log.Error(err, log.Data{"request_body": string(bytes)})
		return nil, err
	}

	// This should be the last check before returning RenderRequest
	if len(bytes) == 2 {
		return &request, ErrorNoData
	}

	return &request, nil
}

// ValidateAnalyseRequest checks the content of the request structure
func (r *AnalyseRequest) ValidateAnalyseRequest() error {

	var missingFields []string

	if r.Geography == nil {
		missingFields = append(missingFields, "geography")
	} else {
		if r.Geography.Topojson == nil {
			missingFields = append(missingFields, "geography.topojson")
		}
		if len(r.Geography.IDProperty) == 0 {
			missingFields = append(missingFields, "geography.id_property")
		}
	}

	if len(r.CSV) == 0 {
		missingFields = append(missingFields, "csv")
	}

	if missingFields != nil {
		return fmt.Errorf("Missing mandatory field(s): %v", missingFields)
	}
	if r.IDIndex < 0 || r.ValueIndex < 0 {
		return fmt.Errorf("id_index and value_index must be >=0: id_index=%v, value_index=%v", r.IDIndex, r.ValueIndex)
	}
	if r.IDIndex == r.ValueIndex {
		return fmt.Errorf("id_index and value_index cannot refer to the same column: id_index=%v, value_index=%v", r.IDIndex, r.ValueIndex)
	}
	return nil
}
