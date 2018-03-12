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
	"github.com/ThinkingLogic/jenks"
	"sort"
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

	values := extractValues(parseInfo.rows)
	breaks := jenks.AllNaturalBreaks(values, 11)
	for i := range breaks {
		breaks[i] = jenks.Round(breaks[i], values)
	}

	classCount := bestFitClassCount(values, breaks)

	return &models.AnalyseResponse{Data: parseInfo.rows, Messages: messages, Breaks: breaks, MinValue:values[0], MaxValue:values[len(values)-1], BestFitClassCount:classCount}, nil
}

// extractValues extracts and sorts the values in rows.
func extractValues(rows []*models.DataRow) []float64 {
	values := make([]float64, len(rows))
	for i := range rows {
		values[i] = rows[i].Value
	}
	sort.Float64s(values)
	return values
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

// bestFitClassCount tries to find the breaks that best fit the data in the fewest classes.
// This is purely a best guess suggestion
func bestFitClassCount(data []float64, allBreaks [][]float64) int {

	const classCountFactor = 0.2
	goodnessFactor := 1.0 - classCountFactor

	maxClasses := float64(len(allBreaks[len(allBreaks)-1]))
	bestCount := 0
	bestFitness := 0.0
	for _, breaks := range allBreaks {
		goodness := goodnessOfVarianceFit(data, breaks)
		classFitness := 1.0 - (float64(len(breaks)) / maxClasses) // fewer classes are fitter
		fitness := ( (goodness * goodnessFactor) + (classFitness * classCountFactor) ) / 2.0
		if fitness > bestFitness {
			bestCount = len(breaks)
			bestFitness = fitness
		}
	}

	return bestCount
}

// goodnessOfVarianceFit returns a value indicating how well the given breaks fit the data,
// with 0 being no fit, and 1 being a perfect fit.
// thanks to: https://stats.stackexchange.com/a/144075
func goodnessOfVarianceFit(data []float64, classes []float64) float64 {
	ssdData := sumOfSquaredDeviations(data)

	// we need the upper bounds for classes
	upperBounds := make([]float64, len(classes))
	copy(upperBounds, classes[1:])
	upperBounds[len(classes)-1] = data[len(data)-1] + 1.0

	ssdClasses := []float64{}
	remainingData := data
	for _, ub := range upperBounds {
		idx := sort.SearchFloat64s(remainingData, ub)
		cls := remainingData[:idx]
		remainingData = remainingData[idx:]
		ssdClasses = append(ssdClasses, sumOfSquaredDeviations(cls))
	}
	ssdc := sum(ssdClasses)
	return (ssdData - ssdc) / ssdData
}

func sum(data []float64)float64 {
	sum := 0.0
	for _, i := range data {
		sum += i
	}
	return sum
}

func sumOfSquaredDeviations(data []float64) float64 {
	mean := sum(data) / float64(len(data))

	ssd := 0.0
	for _, i := range data {
		ssd += math.Pow(i - mean, 2)
	}
	return ssd
}