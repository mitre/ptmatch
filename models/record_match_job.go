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
	RecordMatchConfigurationID bson.ObjectId                   `bson:"recordMatchConfigurationId,omitempty" json:"recordMatchConfigurationId,omitempty"`
	Request                    RecordMatchRequest              `bson:"request,omitempty" json:"request,omitempty"`
	Responses                  []RecordMatchResponse           `bson:"responses,omitempty" json:"responses,omitempty"`
	Status                     []RecordMatchJobStatusComponent `bson:"status,omitempty" json:"status,omitempty"`
}

type RecordMatchJobStatusComponent struct {
	Message   string    `bson:"message" json:"message"`
	CreatedOn time.Time `bson:"createdOn,omitempty" json:"createdOn,omitempty"`
}
