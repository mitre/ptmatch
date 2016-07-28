package ptgen

import (
	"math/rand"

	"github.com/intervention-engine/fhir/models"
)

func GenerateBP(ctx Context) []models.Observation {
	sys, dia := models.Observation{}, models.Observation{}
	sys.Code = &models.CodeableConcept{Coding: []models.Coding{{Code: "8480-6", System: "http://loinc.org"}}, Text: "Systolic Blood Pressure"}
	dia.Code = &models.CodeableConcept{Coding: []models.Coding{{Code: "8462-4", System: "http://loinc.org"}}, Text: "Diastolic Blood Pressure"}
	switch ctx.Hypertention {
	case "Normal":
		sys.ValueQuantity = GenerateQuantity(100, 120)
		dia.ValueQuantity = GenerateQuantity(65, 80)
	case "Pre-hypertension":
		sys.ValueQuantity = GenerateQuantity(120, 140)
		dia.ValueQuantity = GenerateQuantity(80, 90)
	case "Hypertension":
		sys.ValueQuantity = GenerateQuantity(140, 180)
		dia.ValueQuantity = GenerateQuantity(90, 120)
	}
	sys.ValueQuantity.Unit = "mmHg"
	dia.ValueQuantity.Unit = "mmHg"

	return []models.Observation{sys, dia}
}

func GenerateCholesterol(ctx Context) []models.Observation {
	ldl, hdl, tri := models.Observation{}, models.Observation{}, models.Observation{}
	ldl.Code = &models.CodeableConcept{Coding: []models.Coding{{Code: "13457-7", System: "http://loinc.org"}}, Text: "Plasma LDL Cholesterol Measurement"}
	hdl.Code = &models.CodeableConcept{Coding: []models.Coding{{Code: "2085-9", System: "http://loinc.org"}}, Text: "Plasma HDL Cholesterol Measurement"}
	tri.Code = &models.CodeableConcept{Coding: []models.Coding{{Code: "3043-7", System: "http://loinc.org"}}, Text: "Plasma Triglyceride Measurement"}

	switch ctx.Cholesterol {
	case "Optimal":
		ldl.ValueQuantity = GenerateQuantity(80, 100)
		hdl.ValueQuantity = GenerateQuantity(60, 70)
		tri.ValueQuantity = GenerateQuantity(100, 140)
	case "Near Optimal":
		ldl.ValueQuantity = GenerateQuantity(100, 130)
		hdl.ValueQuantity = GenerateQuantity(50, 60)
		tri.ValueQuantity = GenerateQuantity(140, 160)
	case "Borderline":
		ldl.ValueQuantity = GenerateQuantity(130, 150)
		hdl.ValueQuantity = GenerateQuantity(40, 60)
		tri.ValueQuantity = GenerateQuantity(160, 200)
	case "High":
		ldl.ValueQuantity = GenerateQuantity(160, 200)
		hdl.ValueQuantity = GenerateQuantity(40, 50)
		tri.ValueQuantity = GenerateQuantity(200, 300)
	case "Very High":
		ldl.ValueQuantity = GenerateQuantity(190, 220)
		hdl.ValueQuantity = GenerateQuantity(30, 40)
		tri.ValueQuantity = GenerateQuantity(300, 400)
	}

	ldl.ValueQuantity.Unit = "mg/dL"
	hdl.ValueQuantity.Unit = "mg/dL"
	tri.ValueQuantity.Unit = "mg/dL"

	return []models.Observation{ldl, hdl, tri}
}

func GenerateWeightAndHeight(ctx Context) []models.Observation {
	w, h := models.Observation{}, models.Observation{}
	w.Code = &models.CodeableConcept{Coding: []models.Coding{{Code: "29463-7", System: "http://loinc.org"}}, Text: "Body Weight"}
	h.Code = &models.CodeableConcept{Coding: []models.Coding{{Code: "8302-2", System: "http://loinc.org"}}, Text: "Body Height"}

	w.ValueQuantity = GenerateQuantity(ctx.Weight-10, ctx.Weight+10)
	height := float64(ctx.Height)
	h.ValueQuantity = &models.Quantity{Value: &height}

	w.ValueQuantity.Unit = "lbs"
	h.ValueQuantity.Unit = "in"

	return []models.Observation{w, h}
}

func GenerateBloodSugars(ctx Context) []models.Observation {
	gluc, ha1c := models.Observation{}, models.Observation{}
	gluc.Code = &models.CodeableConcept{Coding: []models.Coding{{Code: "1558-6", System: "http://loinc.org"}}, Text: "Fasting Glucose"}
	ha1c.Code = &models.CodeableConcept{Coding: []models.Coding{{Code: "4548-4", System: "http://loinc.org"}}, Text: "Hemoglobin A1c"}

	switch ctx.Diabetes {
	case "Normal":
		gluc.ValueQuantity = GenerateQuantity(75, 100)
		ha1c.ValueQuantity = GenerateQuantity(40, 56)
	case "Pre-diabetes":
		gluc.ValueQuantity = GenerateQuantity(100, 125)
		ha1c.ValueQuantity = GenerateQuantity(57, 64)
	case "Diabetes":
		gluc.ValueQuantity = GenerateQuantity(200, 300)
		ha1c.ValueQuantity = GenerateQuantity(65, 80)
	}
	gluc.ValueQuantity.Unit = "mg/dL"
	percentageValue := *ha1c.ValueQuantity.Value / float64(10)
	ha1c.ValueQuantity.Value = &percentageValue
	ha1c.ValueQuantity.Unit = "%"

	return []models.Observation{gluc, ha1c}
}

func GenerateQuantity(min, max int) *models.Quantity {
	q := float64(min + rand.Intn(max-min))
	return &models.Quantity{Value: &q}
}
