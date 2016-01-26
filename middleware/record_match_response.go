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
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/labstack/echo"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	logger "github.com/mitre/ptmatch/logger"
	ptm_models "github.com/mitre/ptmatch/models"

	fhir_models "github.com/intervention-engine/fhir/models"
)

func ProcessFhirResource(db *mgo.Database) echo.MiddlewareFunc {
	return func(hf echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx *echo.Context) error {
			err := hf(ctx)
			if err != nil {
				return err
			}
			resourceType := ctx.Get("Resource")
			if resourceType != nil {
				resource := ctx.Get(resourceType.(string))
				handleResource(db, resource)
			}
			return nil
		}
	}
}

func handleResource(db *mgo.Database, resource interface{}) error {
	logger.Log.WithFields(logrus.Fields{"func": "handleResponseMessage",
		"resource": resource}).Info("Entering")

	switch r := resource.(type) {
	case *fhir_models.Bundle:
		// Verify this bundle represents a message
		if r.Type == "message" {
			// we care only about response messages
			msgHdr := r.Entry[0].Resource.(*fhir_models.MessageHeader)
			resp := msgHdr.Response
			logger.Log.WithFields(logrus.Fields{"func": "handleResponseMessage",
				"msg hdr": msgHdr}).Info("Recognized Bundle of type, message")
			// verify this is a response for a record-match request
			if resp != nil &&
				msgHdr.Event.Code == "record-match" &&
				msgHdr.Event.System == "http://github.com/mitre/ptmatch/fhir/message-events" {

				logger.Log.WithFields(logrus.Fields{"func": "handleResponseMessage",
					"resp ID": resp.Identifier}).Info("About to update recmatch run")

				// Find the record match run object w/ the record-match request w/ the id in the response
				err := updateRecordMatchRun(db, resp.Identifier, resource.(*fhir_models.Bundle))
				if err != nil {
					return err
				}
			}
		}
		return nil
	default:
		return nil
	}
}

func updateRecordMatchRun(db *mgo.Database, reqID string, respMsg *fhir_models.Bundle) error {
	resourceType := "RecordMatchRun"
	// Determine the collection expected to hold the resource
	c := db.C(ptm_models.GetCollectionName(resourceType))
	recMatchRun := ptm_models.NewStructForResourceName(resourceType).(*ptm_models.RecordMatchRun)
	err := c.Find(bson.M{"request.message.entry.resource._id": reqID}).One(recMatchRun)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{"func": "updateRecordMatchRun",
			"request msg id": reqID}).Warn("Unable to find RecMatchRun assoc w. request")
		return err
	}
	logger.Log.WithFields(logrus.Fields{"func": "updateRecordMatchRun",
		"result": recMatchRun}).Info("found run assoc w. request")

	now := time.Now()
	respID := bson.NewObjectId()

	// Add the record match response to the record run data
	err = c.UpdateId(recMatchRun.ID,
		bson.M{"$push": bson.M{"responses": bson.M{
			"_id":        respID,
			"meta":       bson.M{"lastUpdatedOn": now, "createdOn": now},
			"receivedOn": now,
			"message":    respMsg,
		}}})

	if err != nil {
		logger.Log.WithFields(logrus.Fields{"func": "updateRecordMatchRun",
			"rec match run ID": recMatchRun.ID,
			"error":            err}).Warn("Error adding response to run")
		return err
	}

	// Add an entry to the record match run status and update lastUpdatedOn
	err = c.UpdateId(recMatchRun.ID,
		bson.M{
			"$currentDate": bson.M{"meta.lastUpdatedOn": bson.M{"$type": "timestamp"}},
			"$push": bson.M{
				"status": bson.M{
					"message":   "Response Received",
					"createdOn": now}}})

	if err != nil {
		logger.Log.WithFields(logrus.Fields{"func": "updateRecordMatchRun",
			"rec match run ID": recMatchRun.ID,
			"error":            err}).Warn("Error updating response status in run object")
		return err
	}
	return nil
}
