package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"gopkg.in/mgo.v2/bson"

	"github.com/intervention-engine/fhir/models"
	"github.com/intervention-engine/ptgen"
	ptm_models "github.com/mitre/ptmatch/models"
)

type PatientFuzzer func(*models.Patient)

type Match struct {
	Source string
	Target string
}

const tagURL = "http://mitre.org/ptmatch/recordSet"

// This application will create a new record set in the patient matching test
// harness. This record set will be associated with a FHIR resource tag.
// It will generate a pair of matching patient records, apply the tag, and
// then upload them to the FHIR server. It will also upload an answer key.
func main() {
	fhirURL := flag.String("fhirURL", "", "URL for the patient matching test harness server")
	recordSetName := flag.String("name", "", "Name of the record set")

	flag.Parse()

	trimmedFhirURL := strings.TrimRight(*fhirURL, "/")

	recordSet := &ptm_models.RecordSet{Name: *recordSetName}

	recordSet.ResourceType = "Patient"
	recordSet.Parameters = generateRecordSetParameters(trimmedFhirURL, *recordSetName)

	recordSetURL := trimmedFhirURL + "/RecordSet"
	rsj, err := json.Marshal(recordSet)
	if err != nil {
		return
	}
	body := bytes.NewReader(rsj)
	resp, err := http.Post(recordSetURL, "application/json", body)
	if err != nil {
		fmt.Printf("Couldn't upload the resource: %s\n", err.Error())
		return
	}
	decoder := json.NewDecoder(resp.Body)
	decoder.Decode(recordSet)

	fuzzers := []PatientFuzzer{AbbreviateFirstName, TransposeBirthDayMonth, ShuffleLastName}
	tagCoding := models.Coding{System: tagURL, Code: tagValue(*recordSetName)}
	meta := &models.Meta{Tag: []models.Coding{tagCoding}}
	patientURL := trimmedFhirURL + "/Patient"
	var matches []Match
	for i := 0; i < 300; i++ {
		patient := ptgen.GenerateDemographics()
		patient.Meta = meta
		source, err := PostAndGetLocation(patient, patientURL)
		if err != nil {
			return
		}

		fuzzer := fuzzers[rand.Intn(3)]
		copy := CopyPatient(&patient)
		fuzzer(copy)
		target, err := PostAndGetLocation(copy, patientURL)
		if err != nil {
			return
		}
		matches = append(matches, Match{source, target})
	}
	bundle := &models.Bundle{}
	bundle.Type = "document"
	bundle.Id = bson.NewObjectId().Hex()
	comp := models.Composition{}
	comp.Date = &models.FHIRDateTime{Time: time.Now(), Precision: models.Timestamp}
	comp.Type = &models.CodeableConcept{Coding: []models.Coding{models.Coding{System: "http://loinc.org", Code: "11503-0"}}}
	comp.Title = "Answer Key for " + *recordSetName
	comp.Status = "final"
	comp.Subject = &models.Reference{Reference: fmt.Sprintf("%s/RecordSet/%s", trimmedFhirURL, recordSet.ID.Hex())}
	compEntry := models.BundleEntryComponent{}
	compEntry.Resource = &comp
	bundle.Entry = append(bundle.Entry, compEntry)
	for _, match := range matches {
		entry := models.BundleEntryComponent{}
		entry.FullUrl = match.Source
		linkType := models.BundleLinkComponent{Relation: "type", Url: "http://hl7.org/fhir/Patient"}
		linkRelated := models.BundleLinkComponent{Relation: "related", Url: match.Target}
		entry.Link = []models.BundleLinkComponent{linkType, linkRelated}
		score := 1.0
		entry.Search = &models.BundleEntrySearchComponent{Score: &score}
		bundle.Entry = append(bundle.Entry, entry)
	}
	PostAnswerKey(bundle, recordSet.ID.Hex(), trimmedFhirURL+"/AnswerKey")
}

func PostAnswerKey(answerKey *models.Bundle, recordSetId, fhirUrl string) error {
	akj, _ := json.Marshal(answerKey)
	akb := bytes.NewReader(akj)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fmt.Printf("Got record set id of %s\n", recordSetId)
	writer.WriteField("recordSetId", recordSetId)
	part, err := writer.CreateFormFile("answerKey", "answerKey.json")
	if err != nil {
		return err
	}
	_, err = io.Copy(part, akb)
	if err != nil {
		return err
	}
	contentType := writer.FormDataContentType()
	err = writer.Close()
	if err != nil {
		return err
	}

	_, err = http.Post(fhirUrl, contentType, body)
	return err
}

func PostAndGetLocation(resource interface{}, url string) (string, error) {
	rsj, err := json.Marshal(resource)
	if err != nil {
		return "", err
	}
	body := bytes.NewReader(rsj)
	resp, err := http.Post(url, "application/json", body)
	defer resp.Body.Close()
	if err != nil {
		fmt.Printf("Couldn't upload the resource: %s\n", err.Error())
		return "", err
	}
	if resp.StatusCode == http.StatusCreated && len(resp.Header["Location"]) == 1 {
		return resp.Header["Location"][0], nil
	}
	return "", errors.New("Couldn't find a location for the resource")
}

func CopyPatient(source *models.Patient) *models.Patient {
	target := &models.Patient{}
	target.Gender = source.Gender
	name := models.HumanName{}
	name.Given = []string{source.Name[0].Given[0]}
	name.Family = []string{source.Name[0].Family[0]}
	target.Name = []models.HumanName{name}
	target.BirthDate = &models.FHIRDateTime{Time: source.BirthDate.Time, Precision: models.Date}
	target.Address = source.Address
	target.Meta = source.Meta
	return target
}

func AbbreviateFirstName(patient *models.Patient) {
	firstName := patient.Name[0].Given[0]
	firstInitial := firstName[:1]
	patient.Name[0].Given[0] = firstInitial + "."
}

// Only does something if the day of the birthday is <= 12
func TransposeBirthDayMonth(patient *models.Patient) {
	birthDate := patient.BirthDate.Time
	if birthDate.Day() <= 12 {
		transposedDate := time.Date(birthDate.Year(), time.Month(birthDate.Day()), int(birthDate.Month()), 0, 0, 0, 0, birthDate.Location())
		patient.BirthDate.Time = transposedDate
	}
}

func ShuffleLastName(patient *models.Patient) {
	lastName := patient.Name[0].Family[0]
	if len(lastName) >= 3 {
		shuffledName := lastName[:1] + lastName[2:3] + lastName[1:2] + lastName[3:]
		patient.Name[0].Family[0] = shuffledName
	}
}

func PatientToRecord(patient *models.Patient) []string {
	record := []string{patient.Id, patient.Name[0].Given[0], patient.Name[0].Family[0], patient.Gender}
	birthDate := patient.BirthDate.Time.Format("2006-01-02")
	record = append(record, birthDate)
	address := patient.Address[0]
	record = append(record, address.Line[0])
	record = append(record, address.City)
	record = append(record, address.State)
	record = append(record, address.PostalCode)
	return record
}

func tagValue(recordSetName string) string {
	return strings.Replace(recordSetName, " ", "", -1)
}

func generateRecordSetParameters(fhirURL, recordSetName string) *models.Parameters {
	parameters := models.Parameters{}

	resourceParam := fhirURL + "/Patient"
	urlPcc := models.ParametersParameterComponent{Name: "resourceUrl", ValueString: resourceParam}
	tagParam := tagValue(recordSetName)
	tagPcc := models.ParametersParameterComponent{Name: "_tag", ValueString: tagParam}
	parameters.Parameter = []models.ParametersParameterComponent{urlPcc, tagPcc}
	return &parameters
}
