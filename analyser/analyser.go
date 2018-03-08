package analyser

import (
	"github.com/ONSdigital/dp-map-renderer/models"
	"strings"
	"io"
	"fmt"
	"encoding/csv"
	"math"
	"strconv"
	"github.com/ONSdigital/go-ns/log"
	"github.com/rubenv/topojson"
)

// AnalyseData analyses the given topology and csv file to confirm that they match, returning the csv converted to json
func AnalyseData(request *models.AnalyseRequest) (*models.AnalyseResponse, error) {

	parseInfo, err := parseData(request.CSV, request.IDIndex, request.ValueIndex, request.HasHeaderRow)
	if err != nil {
		return nil, err
	}

	messages := parseInfo.messages

	ids := getTopologyIDs(request.Geography.Topojson, request.Geography.IDProperty)
	unmatchedRows := []string{}
	for _, row := range parseInfo.rows {
		id := ids[row.ID]
		if len(id) == 0 {
			unmatchedRows = append(unmatchedRows, row.ID)
		}
	}
	if len(unmatchedRows) == len(parseInfo.rows) {
		return nil, fmt.Errorf("Data does not match Topology - IDs in the data do not match any IDs in the topology (using property '%s' to identify features in the topology)", request.Geography.IDProperty)
	}
	if len(unmatchedRows) > 0 {
		messages = append(messages, &models.Message{Level:"error", Text:fmt.Sprintf("IDs of %d rows could not be found in the topology. Row IDs: [%v]", len(unmatchedRows), strings.Join(unmatchedRows, ", "))})
	}

	count := len(parseInfo.rows) - len(unmatchedRows)
	messages = append(messages, &models.Message{Level:"info", Text: fmt.Sprintf("Successfully processed %d of %d rows", count, parseInfo.totalRows)})
	return &models.AnalyseResponse{Data:parseInfo.rows, Messages:messages}, nil
}

// parseData parses the csv file into a slice of DataRows, returning it along with messages about the number of rows parsed and any failed rows.
func parseData(csvSource string, idIndex int, valueIndex int, hasHeader bool) (*parseInfo, error){
	r := csv.NewReader(strings.NewReader(csvSource))
	r.FieldsPerRecord = -1 // allow variable count of fields per record

	if hasHeader {
		r.Read()
	}

	requiredColumns := int(math.Max(float64(idIndex), float64(valueIndex))) + 1

	missingColumns := []int{}
	missingValues := []string{}
	rows := []*models.DataRow{}

	i := 0
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		i++
		if err != nil {
			log.Error(err, log.Data{"_message": "Error reading CSV"})
			return nil, fmt.Errorf("Error reading CSV: %v", err.Error())
		}
		if len(record) < requiredColumns {
			missingColumns = append(missingColumns, i)
			continue
		}
		id := record[idIndex]
		value, err := strconv.ParseFloat(record[valueIndex], 64)
		if err != nil {
			missingValues = append(missingValues, id)
			continue
		}
		rows = append(rows, &models.DataRow{ID: id, Value:value})
	}
	if len(missingColumns) == i {
		return nil, fmt.Errorf("All CSV rows had fewer than %d columns - could not read data", requiredColumns)
	}
	if len(missingValues) == i {
		return nil, fmt.Errorf("No CSV rows had a numeric value - could not read data")
	}

	messages := []*models.Message{}
	if len(missingColumns) > 0 {
		rowNumbers := strings.Join(strings.Fields(fmt.Sprint(missingColumns)), ", ")
		messages = append(messages, &models.Message{Level:"warn", Text: fmt.Sprintf("%d rows have missing columns and could not be parsed. Row numbers: %v", len(missingColumns), rowNumbers)})
	}
	if len(missingValues) > 0 {
		messages = append(messages, &models.Message{Level:"warn", Text: fmt.Sprintf("%d rows have missing (or non-numeric) values and could not be parsed. Row IDs: [%v]", len(missingValues), strings.Join(missingValues, ", "))})
	}

	return &parseInfo{rows: rows, messages: messages, totalRows: i}, nil
}

// getTopologyIDs extracts the id from each object in the topology, using the given idProperty first, or the ID if no such property found
func getTopologyIDs(topology *topojson.Topology, idProperty string) map[string]string {
	o := []*topojson.Geometry{}
	for _, v := range topology.Objects {
		o = append(o, v)
	}
	return getGeographyIDs(o, idProperty)
}

// getGeographyIDs extracts the id from each geometry, using the given idProperty first, or the ID if no such property found
func getGeographyIDs(topologyObjects []*topojson.Geometry, idProperty string) map[string]string {
	m := make(map[string]string)
	for _, o := range topologyObjects {
		if o.Type == "GeometryCollection" {
			om := getGeographyIDs(o.Geometries, idProperty)
			for k, v := range om {
				m[k] = v
			}
		} else {
			id, isString := o.Properties[idProperty].(string)
			if isString && len(id) > 0 {
				m[id] = id
			} else {
				m[o.ID] = o.ID
			}
		}
	}
	return m
}

// parseInfo contains information about the rows parsed from the csv
type parseInfo struct {
	rows []*models.DataRow
	messages []*models.Message
	totalRows int
}