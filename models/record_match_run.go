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
	"sort"
	"time"

	"gopkg.in/mgo.v2/bson"
)

type RecordMatchRun struct {
	ID                   bson.ObjectId                   `bson:"_id,omitempty" json:"id,omitempty"`
	Meta                 *Meta                           `bson:"meta,omitempty" json:"meta,omitempty"`
	Note                 string                          `bson:"note,omitempty" json:"note,omitempty"`
	RecordMatchContextID bson.ObjectId                   `bson:"recordMatchContextId,omitempty" json:"recordMatchContextId,omitempty"`
	Request              RecordMatchRequest              `bson:"request,omitempty" json:"request,omitempty"`
	Responses            []RecordMatchResponse           `bson:"responses,omitempty" json:"responses,omitempty"`
	Metrics              RecordMatchRunMetrics           `bson:"metrics,omitempty" json:"metrics,omitempty"`
	Status               []RecordMatchRunStatusComponent `bson:"status,omitempty" json:"status,omitempty"`
	// ideally, deduplication or query
	MatchingMode string `bson:"matchingMode,omitempty" json:"matchingMode,omitempty"`
	// fhir resource type of the records being matched (e.g., Patient)
	RecordResourceType string `bson:"recordResourceType,omitempty" json:"recordResourceType,omitempty"`
	// reference to the record matching system interface
	RecordMatchSystemInterfaceID bson.ObjectId `bson:"recordMatchSystemInterfaceId,omitempty" json:"recordMatchSystemInterfaceId,omitempty"`
	MasterRecordSetID            bson.ObjectId `bson:"masterRecordSetId,omitempty" json:"masterRecordSetId,omitempty"`
	QueryRecordSetID             bson.ObjectId `bson:"queryRecordSetId,omitempty" json:"queryRecordSetId,omitempty"`
}

// RecordMatchRunMetrics contains statistics associated with the results reported
// by a record matching system.
type RecordMatchRunMetrics struct {
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

type RecordMatchRunStatusComponent struct {
	Message   string    `bson:"message" json:"message"`
	CreatedOn time.Time `bson:"createdOn,omitempty" json:"createdOn,omitempty"`
}

// Link is not part of FHIR. It is a simplified representation of a suggested
// link (or lack thereof) between two records.
type Link struct {
	Source string  `json:"source"`
	Target string  `json:"target"`
	Match  string  `json:"match"`
	Score  float64 `json:"score"`
}

// LinkSlice is needed as a holder for the functions to work with the sort
// package
type LinkSlice []Link

func (ls LinkSlice) Len() int           { return len(ls) }
func (ls LinkSlice) Swap(i, j int)      { ls[i], ls[j] = ls[j], ls[i] }
func (ls LinkSlice) Less(i, j int) bool { return ls[i].Score < ls[j].Score }

// GetLinks searches all responses to the match run and creates a []Link that is
// sorted by Score.
func (rmr *RecordMatchRun) GetLinks() []Link {
	var links []Link
	for _, response := range rmr.Responses {
		for _, entry := range response.Message.Entry {
			if len(entry.Link) == 2 {
				source := entry.FullUrl
				var target string
				for _, l := range entry.Link {
					if l.Relation == "related" {
						target = l.Url
					}
				}
				var match string
				for _, e := range entry.Search.Extension {
					if e.Url == "http://hl7.org/fhir/StructureDefinition/patient-mpi-match" {
						match = e.ValueCode
					}
				}
				score := *entry.Search.Score
				links = append(links, Link{source, target, match, score})
			}
		}
	}
	sort.Sort(LinkSlice(links))
	return links
}

func (rmr *RecordMatchRun) GetWorstLinks(count int) []Link {
	links := rmr.GetLinks()
	if count >= len(links) {
		return links
	}
	return links[:count]
}

func (rmr *RecordMatchRun) GetBestLinks(count int) []Link {
	links := rmr.GetLinks()
	if count >= len(links) {
		return links
	}
	return links[len(links)-count : len(links)]
}
