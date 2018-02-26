package testdata

import (
	"io/ioutil"
	"path/filepath"
	"testing"
)

// LoadExampleRequest reads the example request from examplerequest.json
func LoadExampleRequest(t *testing.T) []byte {
	return loadTestdata(t, "exampleRequest.json")
}

// LoadExampleHandsonTable reads the example parse request containing html from handsontable in exampleHandsonTable.json
func LoadExampleHandsonTable(t *testing.T) []byte {
	return loadTestdata(t, "exampleHandsonTable.json")
}

func loadTestdata(t *testing.T, name string) []byte {
	path := filepath.Join("../testdata", name) // relative path
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return bytes
}
