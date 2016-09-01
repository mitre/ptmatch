package ptgen

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"time"

	"golang.org/x/tools/container/intsets"

	"github.com/intervention-engine/fhir/models"
)

type ConditionMetadata struct {
	ID                   int    `json:"condition_id"`
	ICD9                 string `json:"icd9code"`
	Display              string `json:"display"`
	MedicationID         int    `json:"medication_id"`
	Overnights           string `json:"overnights"`
	AbatementChance      int    `json:"abatementChance"`
	Fatal                bool   `json:"healOrDeath"`
	MortalityChance      int    `json:"mortalityChance"`
	MortalityTime        string `json:"mortalityTime"`
	RecoveryEstimate     string `json:"recoveryEstimate"`
	ProcedureChance      int    `json:"procedureChance"`
	ProcedureSuccess     int    `json:"procedureSuccess"`
	CheckUp              string `json:"checkUp"`
	ProcedureDescription string `json:"procedureDescription"`
	ProcedureCode        string `json:"procedureCode"`
	ProcedureName        string `json:"procedureCodeName"`
}

func LoadConditions() []ConditionMetadata {
	j, err := Asset("data/conditions.json")
	if err != nil {
		panic("Can't get the condition data")
	}
	decoder := json.NewDecoder(bytes.NewReader(j))
	md := []ConditionMetadata{}
	decoder.Decode(&md)
	return md
}

func GenerateConditions(ctx Context, md []ConditionMetadata) []models.Condition {
	conditions := []models.Condition{}
	if ctx.Hypertention == "Hypertension" {
		ht := generateCondition("Hypertension", 2, md)
		conditions = append(conditions, ht)
		complication := rand.Intn(5)
		if complication == 1 {
			chf := models.Condition{VerificationStatus: "confirmed"}
			chfmd := conditionByName("Congestive Heart Failure", md)
			chf.Code = &models.CodeableConcept{Coding: []models.Coding{{Code: chfmd.ICD9, System: "http://hl7.org/fhir/sid/icd-9"}}, Text: chfmd.Display}
			chf.OnsetDateTime = &models.FHIRDateTime{Time: ht.OnsetDateTime.Time.AddDate(rand.Intn(2), rand.Intn(10), rand.Intn(28)), Precision: models.Date}
			conditions = append(conditions, chf)
		}
		if complication == 2 {
			phd := models.Condition{VerificationStatus: "confirmed"}
			phdmd := conditionByName("Pulmonary Heart Disease", md)
			phd.Code = &models.CodeableConcept{Coding: []models.Coding{{Code: phdmd.ICD9, System: "http://hl7.org/fhir/sid/icd-9"}}, Text: phdmd.Display}
			phd.OnsetDateTime = &models.FHIRDateTime{Time: ht.OnsetDateTime.Time.AddDate(rand.Intn(2), rand.Intn(10), rand.Intn(28)), Precision: models.Date}
			conditions = append(conditions, phd)
		}
	}
	if ctx.Diabetes == "Diabetes" {
		dia := generateCondition("Diabetes", 2, md)
		conditions = append(conditions, dia)
	}

	if ctx.Smoker == "Smoker" {
		complication := rand.Intn(5)
		if complication == 1 {
			e := generateCondition("Emphysema", 2, md)
			conditions = append(conditions, e)
		}
		if complication == 2 {
			lc := generateCondition("Lung Cancer", 2, md)
			conditions = append(conditions, lc)
		}
	}
	// per http://www.cdc.gov/dhdsp/data_statistics/fact_sheets/fs_atrial_fibrillation.htm
	var afibChance int
	if time.Now().AddDate(-65, 0, 0).After(ctx.BirthDate) {
		afibChance = 9
	} else {
		afibChance = 2
	}

	afibDiceRoll := rand.Intn(100)
	if afibDiceRoll <= afibChance {
		afib := generateCondition("Atrial Fibrillation", 3, md)
		conditions = append(conditions, afib)
	}

	otherConditions := rand.Intn(3)
	previouslySelected := &intsets.Sparse{}
	for index := 0; index < otherConditions; index++ {
		randomCondition := 2 + rand.Intn(76)
		if !previouslySelected.Has(randomCondition) {
			rmd := conditionByID(randomCondition, md)
			rc := generateCondition(rmd.Display, index, md)
			previouslySelected.Insert(randomCondition)
			conditions = append(conditions, rc)
		}
	}

	return conditions
}

func generateCondition(name string, yearOffset int, md []ConditionMetadata) models.Condition {
	c := models.Condition{VerificationStatus: "confirmed"}
	cmd := conditionByName(name, md)
	c.Code = &models.CodeableConcept{Coding: []models.Coding{{Code: cmd.ICD9, System: "http://hl7.org/fhir/sid/icd-9"}}, Text: cmd.Display}
	c.OnsetDateTime = &models.FHIRDateTime{Time: randomOnset(yearOffset), Precision: models.Date}
	recoveryDiceRoll := rand.Intn(100)
	if recoveryDiceRoll <= cmd.AbatementChance {
		switch cmd.RecoveryEstimate {
		case "week":
			c.AbatementDateTime = &models.FHIRDateTime{Time: c.OnsetDateTime.Time.AddDate(0, 0, 7), Precision: models.Date}
		case "threeMonths":
			c.AbatementDateTime = &models.FHIRDateTime{Time: c.OnsetDateTime.Time.AddDate(0, 3, 0), Precision: models.Date}
		case "sixMonths":
			c.AbatementDateTime = &models.FHIRDateTime{Time: c.OnsetDateTime.Time.AddDate(0, 6, 0), Precision: models.Date}
		case "threeYears":
			c.AbatementDateTime = &models.FHIRDateTime{Time: c.OnsetDateTime.Time.AddDate(3, 0, 0), Precision: models.Date}
		}
	}

	return c
}

func conditionByName(name string, md []ConditionMetadata) *ConditionMetadata {
	for _, c := range md {
		if c.Display == name {
			return &c
		}
	}
	return nil
}

func conditionByID(id int, md []ConditionMetadata) *ConditionMetadata {
	for _, c := range md {
		if c.ID == id {
			return &c
		}
	}
	return nil
}

func randomOnset(minYearsAgo int) time.Time {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomYears := minYearsAgo + r.Intn(3)
	randomMonth := r.Intn(11)
	randomDay := r.Intn(28)
	t := time.Now()
	return t.AddDate(-randomYears, -randomMonth, -randomDay).Truncate(time.Hour * 24)
}
