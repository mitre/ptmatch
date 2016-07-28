package ptgen

import (
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/icrowley/fake"
	"github.com/intervention-engine/fhir/models"
	"github.com/jmcvetta/randutil"
)

// Context contains information about the patient that can be used when
// generating information
type Context struct {
	Smoker       string
	Hypertention string
	Alcohol      string
	Cholesterol  string
	Diabetes     string
	Height       int
	Weight       int
	BirthDate    time.Time
}

func GeneratePatient() []interface{} {
	ctx := NewContext()
	tempID := strconv.FormatInt(rand.Int63(), 10)
	pt := GenerateDemographics()
	ctx.Height, ctx.Weight = initialHeightAndWeight(pt.Gender)
	ctx.BirthDate = pt.BirthDate.Time
	pt.Id = tempID
	md := LoadConditions()
	mmd := LoadMedications()
	conditions := GenerateConditions(ctx, md)
	var m []interface{}
	m = append(m, &pt)
	for i := range conditions {
		c := conditions[i]
		c.Patient = &models.Reference{Reference: "cid:" + tempID}
		m = append(m, &c)
		conditionMetadata := conditionByName(c.Code.Text, md)
		med := GenerateMedication(conditionMetadata.MedicationID, c.OnsetDateTime, c.AbatementDateTime, mmd)
		if med != nil {
			med.Patient = &models.Reference{Reference: "cid:" + tempID}
			m = append(m, med)
		}
	}

	for i := 0; i < 3; i++ {
		t := time.Now()
		encounterDate := &models.FHIRDateTime{Time: t.AddDate(-i, rand.Intn(2), rand.Intn(5)), Precision: models.Date}
		encounter := models.Encounter{}
		encounter.Type = []models.CodeableConcept{{Coding: []models.Coding{{Code: "99213", System: "http://www.ama-assn.org/go/cpt"}}, Text: "Office Visit"}}
		encounter.Period = &models.Period{Start: encounterDate}
		encounter.Patient = &models.Reference{Reference: "cid:" + tempID}
		m = append(m, &encounter)
		obs := GenerateBP(ctx)
		obs = append(obs, GenerateBloodSugars(ctx)...)
		obs = append(obs, GenerateWeightAndHeight(ctx)...)
		for j := range obs {
			o := obs[j]
			o.EffectiveDateTime = encounterDate
			o.Subject = &models.Reference{Reference: "cid:" + tempID}
			m = append(m, &o)
		}
	}

	return m
}

func GenerateDemographics() models.Patient {
	patient := models.Patient{}
	patient.Gender = strings.ToLower(fake.Gender())
	name := models.HumanName{}
	var firstName string
	if patient.Gender == "male" {
		firstName = fake.MaleFirstName()
	} else {
		firstName = fake.FemaleFirstName()
	}
	name.Given = []string{firstName}
	name.Family = []string{fake.LastName()}
	patient.Name = []models.HumanName{name}
	patient.BirthDate = &models.FHIRDateTime{Time: RandomBirthDate(), Precision: models.Date}
	patient.Address = []models.Address{GenerateAddress()}
	return patient
}

// RandomBirthDate generates a random birth date between 65 and 85 years ago
func RandomBirthDate() time.Time {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomYears := r.Intn(20)
	yearsAgo := randomYears + 65
	randomMonth := r.Intn(11)
	randomDay := r.Intn(28)
	t := time.Now()
	return t.AddDate(-yearsAgo, -randomMonth, -randomDay).Truncate(time.Hour * 24)
}

func GenerateAddress() models.Address {
	address := models.Address{}
	address.Line = []string{fake.Street()}
	address.City = fake.City()
	address.State = fake.StateAbbrev()
	address.PostalCode = fake.Zip()
	return address
}

// NewContext generates a new context with randomly populated content
func NewContext() Context {
	ctx := Context{}
	smokingChoices := []randutil.Choice{
		{2, "Smoker"},
		{3, "Non-smoker"},
		{1, "Ex-smoker"}}
	sc, _ := randutil.WeightedChoice(smokingChoices)
	ctx.Smoker = sc.Item.(string)

	alcoholChoices := []randutil.Choice{
		{2, "Occasional"},
		{1, "Heavy"},
		{1, "None"}}
	ac, _ := randutil.WeightedChoice(alcoholChoices)
	ctx.Alcohol = ac.Item.(string)

	cholesterolChoices := []randutil.Choice{
		{3, "Optimal"},
		{1, "Near Optimal"},
		{2, "Borderline"},
		{1, "High"},
		{2, "Very High"}}
	cc, _ := randutil.WeightedChoice(cholesterolChoices)
	ctx.Cholesterol = cc.Item.(string)

	hc, _ := randutil.ChoiceString([]string{"Normal", "Pre-hypertension", "Hypertension"})
	ctx.Hypertention = hc

	dc, _ := randutil.ChoiceString([]string{"Normal", "Pre-diabetes", "Diabetes"})
	ctx.Diabetes = dc
	return ctx
}

func initialHeightAndWeight(gender string) (h, w int) {
	if gender == "male" {
		h = 60 + rand.Intn(20)
	} else {
		h = 55 + rand.Intn(20)
	}
	minBMI := float64(18)
	englishBMIConstant := float64(703)
	minWeight := (minBMI / englishBMIConstant) * math.Pow(float64(h), float64(2))
	w = int(minWeight) + rand.Intn(200)
	return
}
