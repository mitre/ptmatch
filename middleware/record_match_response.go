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

func PostProcessRecordMatchResponse(db *mgo.Database) echo.MiddlewareFunc {
	return func(hf echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx *echo.Context) error {
			logger.Log.Info("PostProcessRecordMatchResponse: Before calling handler")
			err := hf(ctx)
			if err != nil {
				return err
			}
			resourceType := ctx.Get("Resource")

			logger.Log.WithFields(logrus.Fields{"func": "PostProcessRecordMatchResponse",
				"method":       ctx.Request().Method,
				"resourceType": resourceType}).Info("check resource type exists")

			if resourceType.(string) == "Bundle" &&
				ctx.Request().Method == "PUT" || ctx.Request().Method == "POST" {
				resource := ctx.Get(resourceType.(string))
				updateRecordMatchRun(db, resource.(*fhir_models.Bundle))
			}
			return nil
		}
	}
}

/*
func handleResponseBundle(db *mgo.Database, b *fhir_models.Bundle) error {
	// Verify this bundle represents a message
	if b.Type == "message" {
		// we care only about response messages
		msgHdr := b.Entry[0].Resource.(*fhir_models.MessageHeader)
		resp := msgHdr.Response

		logger.Log.WithFields(logrus.Fields{"func": "handleResponseBundle",
			"msg hdr": msgHdr}).Info("Recognized Bundle of type, message")

		// verify this is a response for a record-match request
		if resp != nil &&
			msgHdr.Event.Code == "record-match" &&
			msgHdr.Event.System == "http://github.com/mitre/ptmatch/fhir/message-events" {

			// Find the record match run object w/ the record-match request w/ the id in the response
			err := updateRecordMatchRun(db, resp.Identifier, b)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
*/
func updateRecordMatchRun(db *mgo.Database, respMsg *fhir_models.Bundle) error {
	// Verify this bundle represents a message
	if respMsg.Type == "message" {
		// we care only about response messages
		msgHdr := respMsg.Entry[0].Resource.(*fhir_models.MessageHeader)
		resp := msgHdr.Response

		logger.Log.WithFields(logrus.Fields{"func": "handleResponseBundle",
			"msg hdr": msgHdr}).Info("Recognized Bundle of type, message")

		// verify this is a response for a record-match request
		if resp != nil &&
			msgHdr.Event.Code == "record-match" &&
			msgHdr.Event.System == "http://github.com/mitre/ptmatch/fhir/message-events" {

			reqID := resp.Identifier
			// Determine the collection expected to hold the resource
			c := db.C(ptm_models.GetCollectionName("RecordMatchRun"))
			recMatchRun := &ptm_models.RecordMatchRun{}

			err := c.Find(
				bson.M{"request.message.entry.resource._id": reqID}).One(recMatchRun)
			if err != nil {
				logger.Log.WithFields(logrus.Fields{"func": "updateRecordMatchRun",
					"err":            err,
					"request msg id": reqID}).Warn("Unable to find RecMatchRun assoc w. request")
				return err
			}
			logger.Log.WithFields(logrus.Fields{"func": "updateRecordMatchRun",
				"result": recMatchRun}).Info("found run assoc w. request")

			now := time.Now()

			// check that the response is already assoc. w/ the record match run object
			count, err := c.Find(bson.M{"_id": recMatchRun.ID,
				"responses.message._id": respMsg.Id}).Count()

			logger.Log.WithFields(logrus.Fields{"func": "updateRecordMatchRun",
				"count": count}).Info("look for dupl response")

			if count > 0 {
				// The response message has been processed before
				logger.Log.WithFields(logrus.Fields{"func": "updateRecordMatchRun",
					"record match run": recMatchRun.ID,
					"response msg Id":  respMsg.Id}).Info("record match response seen before")

				// Record that we've seen this response before
				err = c.UpdateId(recMatchRun.ID,
					bson.M{
						"$currentDate": bson.M{"meta.lastUpdatedOn": bson.M{"$type": "timestamp"}},
						"$push": bson.M{
							"status": bson.M{
								"message":   "Duplicate Response Received and Ignored [" + respMsg.Id + "]",
								"createdOn": now}}})

				return nil
			}

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
					"error":            err}).Warn("Error adding response to Run Info")
				return err
			}

			// Add an entry to the record match run status and update lastUpdatedOn
			err = c.UpdateId(recMatchRun.ID,
				bson.M{
					"$currentDate": bson.M{"meta.lastUpdatedOn": bson.M{"$type": "timestamp"}},
					"$push": bson.M{
						"status": bson.M{
							"message":   "Response Received [" + respMsg.Id + "]",
							"createdOn": now}}})

			if err != nil {
				logger.Log.WithFields(logrus.Fields{"func": "updateRecordMatchRun",
					"rec match run ID": recMatchRun.ID,
					"error":            err}).Warn("Error updating response status in run object")
				return err
			}
		}
	}
	return nil
}
