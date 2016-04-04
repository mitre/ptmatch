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
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	logger "github.com/mitre/ptmatch/logger"
	ptm_models "github.com/mitre/ptmatch/models"

	fhir_models "github.com/intervention-engine/fhir/models"
)

func PostProcessRecordMatchResponse(db *mgo.Database) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		logger.Log.Info("PostProcessRecordMatchResponse: Before calling handler")
		ctx.Next()
		resourceType, _ := ctx.Get("Resource")

		logger.Log.WithFields(logrus.Fields{"func": "PostProcessRecordMatchResponse",
			"method":       ctx.Request.Method,
			"query":        ctx.Request.RequestURI,
			"resourceType": resourceType}).Info("check resource type exists")

		if resourceType.(string) == "Bundle" &&
			ctx.Request.Method == "PUT" || ctx.Request.Method == "POST" {
			resource, _ := ctx.Get(resourceType.(string))
			updateRecordMatchJob(db, resource.(*fhir_models.Bundle))
		}
	}

}

func updateRecordMatchJob(db *mgo.Database, respMsg *fhir_models.Bundle) error {
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
			c := db.C(ptm_models.GetCollectionName("RecordMatchJob"))
			recMatchJob := &ptm_models.RecordMatchJob{}

			err := c.Find(
				bson.M{"request.message.entry.resource._id": reqID}).One(recMatchJob)
			if err != nil {
				logger.Log.WithFields(logrus.Fields{"func": "updateRecordMatchJob",
					"err":            err,
					"request msg id": reqID}).Warn("Unable to find RecMatchJob assoc w. request")
				return err
			}
			logger.Log.WithFields(logrus.Fields{"func": "updateRecordMatchJob",
				"result": recMatchJob}).Info("found run assoc w. request")

			now := time.Now()

			// check that the response is already assoc. w/ the record match run object
			count, err := c.Find(bson.M{"_id": recMatchJob.ID,
				"responses.message._id": respMsg.Id}).Count()

			logger.Log.WithFields(logrus.Fields{"func": "updateRecordMatchJob",
				"count": count}).Info("look for dupl response")

			if count > 0 {
				// The response message has been processed before
				logger.Log.WithFields(logrus.Fields{"func": "updateRecordMatchJob",
					"record match run": recMatchJob.ID,
					"response msg Id":  respMsg.Id}).Info("record match response seen before")

				// Record that we've seen this response before
				err = c.UpdateId(recMatchJob.ID,
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
			err = c.UpdateId(recMatchJob.ID,
				bson.M{"$push": bson.M{"responses": bson.M{
					"_id":        respID,
					"meta":       bson.M{"lastUpdatedOn": now, "createdOn": now},
					"receivedOn": now,
					"message":    respMsg,
				}}})

			if err != nil {
				logger.Log.WithFields(logrus.Fields{"func": "updateRecordMatchJob",
					"rec match run ID": recMatchJob.ID,
					"error":            err}).Warn("Error adding response to Job Info")
				return err
			}

			// Add an entry to the record match run status and update lastUpdatedOn
			err = c.UpdateId(recMatchJob.ID,
				bson.M{
					"$currentDate": bson.M{"meta.lastUpdatedOn": bson.M{"$type": "timestamp"}},
					"$push": bson.M{
						"status": bson.M{
							"message":   "Response Received [" + respMsg.Id + "]",
							"createdOn": now}}})

			if err != nil {
				logger.Log.WithFields(logrus.Fields{"func": "updateRecordMatchJob",
					"rec match run ID": recMatchJob.ID,
					"error":            err}).Warn("Error updating response status in run object")
				return err
			}
		}
	}
	return nil
}
