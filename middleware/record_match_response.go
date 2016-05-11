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
	fhir_svr "github.com/intervention-engine/fhir/server"
)

// PostProcessRecordMatchResponse is a middleware function that processes the
// response message received from the record matching system.
func PostProcessRecordMatchResponse() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		logger.Log.Info("PostProcessRecordMatchResponse: Before calling handler")
		ctx.Next()
		resourceType, _ := ctx.Get("Resource")

		logger.Log.WithFields(logrus.Fields{"action": "check resource type exists",
			"method":       ctx.Request.Method,
			"query":        ctx.Request.RequestURI,
			"resourceType": resourceType}).Info("PostProcessRecordMatchResponse")

		if resourceType.(string) == "Bundle" &&
			ctx.Request.Method == "PUT" || ctx.Request.Method == "POST" {
			resource, _ := ctx.Get(resourceType.(string))
			// Need to access Database via global variable 'cuz DB not initializaed when middlewhare is configured
			updateRecordMatchJob(fhir_svr.Database, resource.(*fhir_models.Bundle))
		}
	}

}

func updateRecordMatchJob(db *mgo.Database, respMsg *fhir_models.Bundle) error {
	// Verify this bundle represents a message
	if respMsg.Type == "message" {
		// we care only about response messages
		msgHdr := respMsg.Entry[0].Resource.(*fhir_models.MessageHeader)
		resp := msgHdr.Response

		logger.Log.WithFields(logrus.Fields{"action": "Recognized Bundle of type, message",
			"bundle id": respMsg.Id,
			"msg hdr": msgHdr}).Info("updateRecordMatchJob")

		// verify this is a response for a record-match request
		if resp != nil &&
			msgHdr.Event.Code == "record-match" &&
			msgHdr.Event.System == "http://github.com/mitre/ptmatch/fhir/message-events" {

			reqID := resp.Identifier
			// Determine the collection expected to hold the resource
			c := db.C(ptm_models.GetCollectionName("RecordMatchJob"))
			recMatchJob := &ptm_models.RecordMatchJob{}

			// retrieve the record-match run
			err := c.Find(
				bson.M{"request.message.entry.resource._id": reqID}).One(recMatchJob)
			if err != nil {
				logger.Log.WithFields(logrus.Fields{"func": "updateRecordMatchJob",
					"err":            err,
					"request msg id": reqID}).Warn("Unable to find RecMatchJob assoc w. request")
				return err
			}
			logger.Log.WithFields(logrus.Fields{"action": "found run assoc w. request",
				"result": recMatchJob}).Info("updateRecordMatchJob")

			now := time.Now()

			// check whether the response is already assoc. w/ the record match run object
			count, err := c.Find(bson.M{"_id": recMatchJob.ID,
				"responses.message._id": respMsg.Id}).Count()

			logger.Log.WithFields(logrus.Fields{"action": "look for dupl response",
				"respMsg.Id": respMsg.Id,
				"count":      count}).Info("updateRecordMatchJob")

			if count > 0 {
				// The response message has been processed before
				logger.Log.WithFields(logrus.Fields{"action": "record match response seen before",
					"record match run": recMatchJob.ID,
					"response msg Id":  respMsg.Id}).Info("updateRecordMatchJob")

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

			var respID bson.ObjectId

			// if the bundle id looks like a bson object id, use it; else we need
			// to create a bson id 'cuz IE fhir server only supports those (5/10/16)'
			if bson.IsObjectIdHex(respMsg.Id) {
				respID = bson.ObjectIdHex(respMsg.Id)
			} else {
				logger.Log.WithFields(logrus.Fields{"msg": "Response Msg Id is not BSON Object Id format",
					"rec match run ID": recMatchJob.ID,
					"respMsg.id":       respMsg.Id}).Warn("updateRecordMatchJob")
				respID = bson.NewObjectId()
			}

			// Add the record match response to the record run data
			err = c.UpdateId(recMatchJob.ID,
				bson.M{"$push": bson.M{"responses": bson.M{
					"_id":        respID,
					"meta":       bson.M{"lastUpdatedOn": now, "createdOn": now},
					"receivedOn": now,
					"message":    respMsg,
				}}})

			if err != nil {
				logger.Log.WithFields(logrus.Fields{"msg": "Error adding response to Job Info",
					"rec match run ID": recMatchJob.ID,
					"error":            err}).Warn("eupdateRecordMatchJob")
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
				logger.Log.WithFields(logrus.Fields{"msg": "Error updating response status in run object",
					"rec match run ID": recMatchJob.ID,
					"error":            err}).Warn("updateRecordMatchJob")
				return err
			}
			// Calculate metrics
			_ = calcMetrics(db, recMatchJob, respMsg)
		}
	}
	return nil
}
