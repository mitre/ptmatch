/*
Copyright 2016 The MITRE Corporation. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package models

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

type RecordMatchJob struct {
	ID                         bson.ObjectId                   `bson:"_id,omitempty" json:"id,omitempty"`
	Meta                       *Meta                           `bson:"meta,omitempty" json:"meta,omitempty"`
	Note                       string                          `bson:"note,omitempty" json:"note,omitempty"`
	RecordMatchConfigurationID bson.ObjectId                   `bson:"recordMatchConfigurationId,omitempty" json:"recordMatchConfigurationId,omitempty"`
	Request                    RecordMatchRequest              `bson:"request,omitempty" json:"request,omitempty"`
	Responses                  []RecordMatchResponse           `bson:"responses,omitempty" json:"responses,omitempty"`
	Metrics                    RecordMatchJobMetrics           `bson:"metrics,omitempty" json:"metrics,omitempty"`
	Status                     []RecordMatchJobStatusComponent `bson:"status,omitempty" json:"status,omitempty"`
	MatchingMode               string                          `bson:"matchingMode,omitempty" json:"matchingMode,omitempty"`
	// fhir resource type of the records being matched (e.g., Patient)
	RecordResourceType string `bson:"recordResourceType,omitempty" json:"recordResourceType,omitempty"`
	// reference to the record matching system interface
	RecordMatchSystemInterfaceID bson.ObjectId `bson:"recordMatchSystemInterfaceId,omitempty" json:"recordMatchSystemInterfaceId,omitempty"`
	MasterRecordSetID            bson.ObjectId `bson:"masterRecordSetId,omitempty" json:"masterRecordSetId,omitempty"`
	QueryRecordSetID             bson.ObjectId `bson:"queryRecordSetId,omitempty" json:"queryRecordSetId,omitempty"`
}

// RecordMatchJobMetrics contains statistics associated with the results reported
// by a record matching system.
type RecordMatchJobMetrics struct {
	F1                 float32 `bson:"f1,omitempty" json:"f1,omitempty"`
	Precision          float32 `bson:"precision,omitempty" json:"precision,omitempty"`
	Recall             float32 `bson:"recall,omitempty" json:"recall,omitempty"`
	MatchCount         int     `bson:"matchCount,omitempty" json:"matchCount,omitempty"`
	TruePositiveCount  int     `bson:"truePositiveCount,omitempty" json:"truePositiveCount,omitempty"`
	FalsePositiveCount int     `bson:"falsePositiveCount,omitempty" json:"falsePositiveCount,omitempty"`
	FRecall            float32 `bson:"FRecall,omitempty" json:"FRecall,omitempty"`
	FPrecision         float32 `bson:"FPrecision,omitempty" json:"FPrecision,omitempty"`
	MAP                float32 `bson:"MAP,omitempty" json:"MAP,omitempty"`
}

type RecordMatchJobStatusComponent struct {
	Message   string    `bson:"message" json:"message"`
	CreatedOn time.Time `bson:"createdOn,omitempty" json:"createdOn,omitempty"`
}
