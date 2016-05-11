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

	if answerKey != nil {
		// store answer key info in map
	}

	metrics := recMatchJob.Metrics

	logger.Log.WithFields(logrus.Fields{
		"metrics": metrics}).Info("calcMetrics")

	//expectedResourceType := recMatchJob.RecordResourceType
	matchCount := 0

	for i, entry := range respMsg.Entry {
		//			logger.Log.WithFields(logrus.Fields{
		//			"i": i,
		//		"entry type": rtype,
		//	"entry kind": reflect.ValueOf(entry.Resource).Kind()}).Info("calcMetrics")

		// Results are in untyped entry w/ links and search result
		if entry.Resource == nil {
			baseRec := entry.FullUrl
			if baseRec != "" && entry.Search != nil && len(entry.Link) > 0 {
				score := *entry.Search.Score
				//				logger.Log.WithFields(logrus.Fields{
				//					"i":        i,
				//					"full url": baseRec,
				//					"search":   score}).Info("calcMetrics")
				if score > 0 {
					for _, link := range entry.Link {
						if strings.EqualFold("related", link.Relation) {
							linkedURL := link.Url
							matchCount++
							logger.Log.WithFields(logrus.Fields{
								"i":        i,
								"full url": baseRec,
								"link url": linkedURL,
								"search":   score}).Info("calcMetrics")
							// if we have an answer key to compare against
							if answerKey != nil {

							}
						}
					}
				}
			}
		}
	}
	logger.Log.WithFields(logrus.Fields{
		"matchCount": matchCount}).Info("calcMetrics")

	metrics.MatchCount += matchCount

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
