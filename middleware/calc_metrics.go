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

package middleware

import (
	//	"reflect"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	logger "github.com/mitre/ptmatch/logger"
	ptm_models "github.com/mitre/ptmatch/models"

	fhir_models "github.com/intervention-engine/fhir/models"
)

func calcMetrics(db *mgo.Database, recMatchJob *ptm_models.RecordMatchJob,
	respMsg *fhir_models.Bundle) error {

	answerKey, _ := getAnswerKey(db, recMatchJob)
	totalResults := 0
	var answerMap map[string]*groundTruth

	if answerKey != nil {
		// store answer key info in map
		answerMap, totalResults = buildAnswerMap(answerKey)
	}

	metrics := recMatchJob.Metrics

	logger.Log.WithFields(logrus.Fields{
		"metrics": metrics}).Info("calcMetrics")

	//expectedResourceType := recMatchJob.RecordResourceType
	matchCount := 0
	truePositiveCount := 0
	falsePositiveCount := 0

	for i, entry := range respMsg.Entry {
		//			logger.Log.WithFields(logrus.Fields{
		//			"i": i,
		//		"entry type": rtype,
		//	"entry kind": reflect.ValueOf(entry.Resource).Kind()}).Info("calcMetrics")

		// Results are in untyped entry w/ links and search result
		if entry.Resource == nil {
			refURL := entry.FullUrl
			if refURL != "" && entry.Search != nil && len(entry.Link) > 0 {
				score := *entry.Search.Score
				//				logger.Log.WithFields(logrus.Fields{
				//					"i":        i,
				//					"full url": refURL,
				//					"search":   score}).Info("calcMetrics")
				if score > 0 {
					for _, link := range entry.Link {
						if strings.EqualFold("related", link.Relation) {
							linkedURL := link.Url
							matchCount++
							logger.Log.WithFields(logrus.Fields{
								"i":        i,
								"full url": refURL,
								"link url": linkedURL,
								"search":   score}).Info("calcMetrics")
							// if we have an answer key to compare against
							if answerKey != nil {
								truth := answerMap[refURL]
								if truth != nil {
									// look for linked URL in array of known linked records
									idx := indexOf(truth.linkedURLs, linkedURL)
									if idx >= 0 {
										truePositiveCount++
										truth.numFound[idx]++
									} else {
										falsePositiveCount++
									}
								} else if answerMap[linkedURL] != nil {
									truth := answerMap[linkedURL]
									// look for reference URL in array of known linked records
									idx := indexOf(truth.linkedURLs, refURL)
									if idx >= 0 {
										truePositiveCount++
										truth.numFound[idx]++
									} else {
										falsePositiveCount++
									}
								} else {
									// no entry found in answer key; this is a false positive
									falsePositiveCount++
								}
							}
						}
					}
				}
			}
		}
	}

	logger.Log.WithFields(logrus.Fields{
		"truePositive":  truePositiveCount,
		"falsePositive": falsePositiveCount,
		"matchCount":    matchCount}).Info("calcMetrics")

	metrics.MatchCount += matchCount
	if answerKey != nil && totalResults > 0 {
		metrics.TruePositiveCount += truePositiveCount
		metrics.FalsePositiveCount += falsePositiveCount
		metrics.Precision = float32(metrics.TruePositiveCount) / float32(metrics.TruePositiveCount+metrics.FalsePositiveCount)
		metrics.Recall = float32(metrics.TruePositiveCount) / float32(totalResults)
		metrics.F1 = 2.0 * ((metrics.Precision * metrics.Recall) / (metrics.Precision + metrics.Recall))
	}

	now := time.Now()

	c := db.C(ptm_models.GetCollectionName("RecordMatchJob"))
	// Add an entry to the record match run status and update lastUpdatedOn
	err := c.UpdateId(recMatchJob.ID,
		bson.M{
			"$currentDate": bson.M{"meta.lastUpdatedOn": bson.M{"$type": "timestamp"}},
			"$set":         bson.M{"metrics": metrics},
			"$push": bson.M{
				"status": bson.M{
					"message":   "Metrics Updated [" + respMsg.Id + "]",
					"createdOn": now}}})

	if err != nil {
		logger.Log.WithFields(logrus.Fields{"msg": "Error updating metrics in record match run",
			"rec match run ID": recMatchJob.ID,
			"error":            err}).Warn("calcMetrics")
		return err
	}

	return nil
}

func indexOf(s []string, e string) int {
	for i, a := range s {
		if a == e {
			return i
		}
	}
	return -1
}

func getAnswerKey(db *mgo.Database, recMatchJob *ptm_models.RecordMatchJob) (*fhir_models.Bundle, error) {
	var answerKey *fhir_models.Bundle

	// If deduplication mode, try to find an answer key w/ the master record set
	if recMatchJob.MatchingMode == ptm_models.Deduplication {
		masterRecSet := &ptm_models.RecordSet{}
		c := db.C(ptm_models.GetCollectionName("RecordSet"))
		// retrieve the master record set
		err := c.Find(
			bson.M{"_id": recMatchJob.MasterRecordSetID}).One(masterRecSet)
		if err != nil {
			logger.Log.WithFields(logrus.Fields{"func": "updateRecordMatchJob",
				"err":          err,
				"msg":          "Unable to find master record set",
				"record setid": recMatchJob.MasterRecordSetID}).Warn("calcMetrics")
			return nil, err
		}

		logger.Log.WithFields(logrus.Fields{
			"rec set":            masterRecSet,
			"answer key entries": len(masterRecSet.AnswerKey.Entry)}).Info("calcMetrics")

		if len(masterRecSet.AnswerKey.Entry) > 1 {
			answerKey = &masterRecSet.AnswerKey
			logger.Log.WithFields(logrus.Fields{
				"answer key entries": len(masterRecSet.AnswerKey.Entry)}).Info("calcMetrics")
		}
	} else if recMatchJob.MatchingMode == ptm_models.Query {
		logger.Log.WithFields(logrus.Fields{
			"msg": "Calculating Metrics for Query Mode Not Supoprted"}).Warn("calcMetrics")
		return nil, nil
	}

	return answerKey, nil
}

func buildAnswerMap(answerKey *fhir_models.Bundle) (map[string]*groundTruth, int) {
	totalResults := 0
	m := make(map[string]*groundTruth)
	for i, entry := range answerKey.Entry {
		//		rtype := reflect.TypeOf(entry.Resource)
		//		logger.Log.WithFields(logrus.Fields{
		//			"answer":     i,
		//			"entry type": rtype,
		//			"entry kind": reflect.ValueOf(entry.Resource).Kind()}).Info("calcMetrics")

		// Results are in untyped entry w/ links and search result
		if entry.Resource == nil {

			refURL := entry.FullUrl
			if refURL != "" && entry.Search != nil && len(entry.Link) > 0 {
				score := *entry.Search.Score
				// Allow for possibility of true negatives being expressed in answer key
				if score > 0 {
					for _, link := range entry.Link {
						if strings.EqualFold("related", link.Relation) {
							linkedURL := link.Url
							logger.Log.WithFields(logrus.Fields{
								"answer":   i,
								"full url": refURL,
								"link url": linkedURL,
								"search":   score}).Info("buildAnswerMap")
							// look for existing entry
							item := m[refURL]
							if item == nil {
								item = &groundTruth{}
								item.linkedURLs = make([]string, 1)
								item.numFound = make([]int, 1)
								item.linkedURLs[0] = linkedURL
								item.numFound[0] = 1
								m[refURL] = item
								totalResults++
							}
						}
					}
				}
			}
		}
	}
	return m, totalResults
}

type groundTruth struct {
	linkedURLs []string
	numFound   []int
}
