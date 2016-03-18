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
	"encoding/json"

	"gopkg.in/mgo.v2/bson"
)

const (
	Deduplication = "deduplication"
	Query         = "query"
)

type RecordMatchConfiguration struct {
	ID   bson.ObjectId `bson:"_id,omitempty" json:"id,omitempty"`
	Meta *Meta         `bson:"meta,omitempty" json:"meta,omitempty"`
	// human-friendly name assoc. w/ this record matching configuration
	Name string `bson:"name,omitempty" json:"name,omitempty"`
	// descriptive remarks assoc. w/ this interface to the record match system
	Description string `bson:"description,omitempty" json:"description,omitempty"`
	// ideally, deduplication or query
	MatchingMode string `bson:"matchingMode,omitempty" json:"matchingMode,omitempty"`
	// fhir resource type of the records being matched (e.g., Patient)
	RecordResourceType string `bson:"recordResourceType,omitempty" json:"recordResourceType,omitempty"`
	// reference to the record matching system interface
	RecordMatchSystemInterfaceID bson.ObjectId `bson:"recordMatchSystemInterfaceId,omitempty" json:"recordMatchSystemInterfaceId,omitempty"`
	MasterRecordSetID            bson.ObjectId `bson:"masterRecordSetId,omitempty" json:"masterRecordSetId,omitempty"`
	QueryRecordSetID             bson.ObjectId `bson:"queryRecordSetId,omitempty" json:"queryRecordSetId,omitempty"`
	//RecordMatchJobs []bson.ObjectId `bson:"recordMatchJobs,omitempty" json:"recordMatchJobs,omitempty" json:"recordMatchJobs,omitempty"`
}

// MarshalJSON - custom marshaller to add the resourceType property, as required by the specification
func (resource *RecordMatchConfiguration) MarshalJSON() ([]byte, error) {
	x := struct {
		ResourceType string `json:"resourceType"`
		RecordMatchConfiguration
	}{
		ResourceType:             "RecordMatchConfiguration",
		RecordMatchConfiguration: *resource,
	}
	return json.Marshal(x)
}
