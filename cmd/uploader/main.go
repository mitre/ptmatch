package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	fhir_models "github.com/intervention-engine/fhir/models"
	ptm_models "github.com/mitre/ptmatch/models"
)

const tagURL = "http://mitre.org/ptmatch/recordSet"

// This application will create a new record set in the patient matching test
// harness. This record set will be associated with a FHIR resource tag.
// It will go through a directory of patient resources, in JSON format, read
// them in, apply the tag, and then upload them to the FHIR server.
func main() {
	fhirURL := flag.String("fhirURL", "", "URL for the patient matching test harness server")
	recordSetName := flag.String("name", "", "Name of the record set")
	path := flag.String("path", "", "Path to the JSON files")

	flag.Parse()

	argsToName := map[string]string{"fhirURL": *fhirURL, "name": *recordSetName, "path": *path}
	for argName, argValue := range argsToName {
		if argValue == "" {
			fmt.Printf("You must provide an argument for %s\n", argName)
			return
		}
	}

	trimmedFhirURL := strings.TrimRight(*fhirURL, "/")

	recordSet := &ptm_models.RecordSet{Name: *recordSetName}

	recordSet.ResourceType = "Patient"
	recordSet.Parameters = generateRecordSetParameters(trimmedFhirURL, *recordSetName)
	rsj, _ := json.Marshal(recordSet)
	body := bytes.NewReader(rsj)

	recordSetURL := trimmedFhirURL + "/RecordSet"
	http.Post(recordSetURL, "application/json", body)

	files, err := ioutil.ReadDir(*path)
	if err != nil {
		fmt.Printf("Couldn't read the directory: %s\n", err.Error())
		return
	}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") {
			jsonBlob, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", *path, file.Name()))
			patient := &fhir_models.Patient{}
			if err != nil {
				fmt.Printf("Couldn't read the JSON file: %s\n", err.Error())
				return
			}
			json.Unmarshal(jsonBlob, patient)
			tagCoding := fhir_models.Coding{System: tagURL, Code: tagValue(*recordSetName)}
			meta := &fhir_models.Meta{Tag: []fhir_models.Coding{tagCoding}}
			patient.Meta = meta
			pj, _ := json.Marshal(patient)
			pb := bytes.NewReader(pj)
			patientURL := trimmedFhirURL + "/Patient"
			resp, err := http.Post(patientURL, "application/json", pb)
			if err != nil {
				fmt.Printf("Couldn't upload patient: %s\n", err.Error())
				return
			}
			if resp.StatusCode != http.StatusCreated {
				fmt.Sprintf("Unexpected status code when creating a patient: %d\n", resp.StatusCode)
				return
			}
		}
	}
}

func tagValue(recordSetName string) string {
	return strings.Replace(recordSetName, " ", "", -1)
}

func generateRecordSetParameters(fhirURL, recordSetName string) *fhir_models.Parameters {
	parameters := fhir_models.Parameters{}

	resourceParam := fhirURL + "/Patient"
	urlPcc := fhir_models.ParametersParameterComponent{Name: "resourceUrl", ValueString: resourceParam}
	tagParam := tagValue(recordSetName)
	tagPcc := fhir_models.ParametersParameterComponent{Name: "_tag", ValueString: tagParam}
	parameters.Parameter = []fhir_models.ParametersParameterComponent{urlPcc, tagPcc}
	return &parameters
}
