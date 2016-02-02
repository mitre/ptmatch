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

import "gopkg.in/mgo.v2/bson"

type RecordMatchSystemInterface struct {
	ID   bson.ObjectId `bson:"_id,omitempty" json:"id,omitempty"`
	Meta *Meta         `bson:"meta,omitempty" json:"meta,omitempty"`
	// human-friendly name assoc. with this record match system interface
	Name string `bson:"name,omitempty" json:"name,omitempty"`
	// descriptive remarks assoc. w/ this interface to the record match system
	Description string `bson:"description,omitempty" json:"description,omitempty"`
	// address to the record match system that will process the request
	DestinationEndpoint string `bson:"destinationEndpoint,omitempty" json:"destinationEndpoint,omitempty"`
	// address to the FHIR Server to which the record-match message should be PUT, may be same or different from ultimate message destination
	ServerEndpoint string `bson:"serverEndpoint,omitempty" json:"serverEndpoint,omitempty"`
	// address to which record match direct response messages
	ResponseEndpoint string `bson:"responseEndpoint,omitempty" json:"responseEndpoint,omitempty"`
}
