package testdata

import (
	"io/ioutil"
	"path/filepath"
	"testing"
)

// LoadExampleAnalyseRequest reads the example request from exampleAnalyseRequest.json
func LoadExampleAnalyseRequest(t *testing.T) []byte {
	return loadTestdata(t, "exampleAnalyseRequest.json")
}

// LoadExampleRequest reads the example request from exampleRequest.json
func LoadExampleRequest(t *testing.T) []byte {
	return loadTestdata(t, "exampleRequest.json")
}

func loadTestdata(t *testing.T, name string) []byte {
	path := filepath.Join("../testdata", name) // relative path
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return bytes
}
