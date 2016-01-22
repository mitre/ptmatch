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

	fhir_models "github.com/intervention-engine/fhir/models"
)

type RecordMatchResponse struct {
	ID         bson.ObjectId       `bson:"_id,omitempty" json:"id,omitempty"`
	Meta       *Meta               `bson:"meta,omitempty" json:"meta,omitempty"`
	Message    *fhir_models.Bundle `bson:"message,omitempty" json:"message,omitempty"`
	ReceivedOn time.Time           `bson:"receivedOn,omitempty" json:"receivedOn,omitempty"`
}
