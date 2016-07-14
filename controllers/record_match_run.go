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

package controllers

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/satori/go.uuid"

	fhir_models "github.com/intervention-engine/fhir/models"
	ptm_http "github.com/mitre/ptmatch/http"
	logger "github.com/mitre/ptmatch/logger"
	ptm_models "github.com/mitre/ptmatch/models"
)

// CreateRecordMatchRunHandler creates a HandlerFunc that creates a new
// RecordMatchRun and constructs and sends a Record Match request message.
func CreateRecordMatchRunHandler(provider func() *mgo.Database) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		recMatchRun := &ptm_models.RecordMatchRun{}
		if err := ctx.Bind(recMatchRun); err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		if !isValidRecordMatchRun(recMatchRun) {
			// Bad Request: Record Match Run contains invalid content
			ctx.String(http.StatusBadRequest, "Invalid RecordMatchRun content")
			ctx.Abort()
			return
		}

		// retrieve and validate the record match context
		recMatchContextID := recMatchRun.RecordMatchContextID
		logger.Log.WithFields(
			logrus.Fields{"method": "CreateRecordMatchRun",
				"recMatchContextID": recMatchContextID}).Debug("check recmatch config id")
		if !recMatchContextID.Valid() {
			// Bad Request: Record Match Context is optional but must be valid, if provided
			ctx.String(http.StatusBadRequest, "Invalid RecordMatchContextID")
			ctx.Abort()
			return
		}

		// Retrieve the info about the record matcher
		obj, err := ptm_models.LoadResource(provider(), "RecordMatchSystemInterface",
			recMatchRun.RecordMatchSystemInterfaceID)
		if err != nil {
			ctx.String(http.StatusBadRequest, "Unable to find Record Match System Interface")
			ctx.Abort()
			return
		}
		recMatchSysIface := obj.(*ptm_models.RecordMatchSystemInterface)
		if !isValidRecordMatchSysIface(recMatchSysIface) {
			ctx.String(http.StatusBadRequest, "Invalid Record Match System Interface")
			ctx.Abort()
			return
		}

		// construct a record match request
		reqMatchRequest := newRecordMatchRequest(recMatchSysIface.ResponseEndpoint, recMatchRun, provider())
		// attach the request message to the run object
		recMatchRun.Request = *reqMatchRequest

		// Construct body of the http request for the record match request
		reqBody, _ := reqMatchRequest.Message.MarshalJSON()

		svrEndpoint := prepEndpoint(recMatchSysIface.ServerEndpoint, reqMatchRequest.Message.Id)

		logger.Log.WithFields(
			logrus.Fields{"method": "CreateRecordMatchRun",
				"server endpoint": svrEndpoint,
				"reqBody":         string(reqBody[:]),
				"message":         reqMatchRequest.Message,
				"request":         reqMatchRequest}).Info("About to submit request")

		reqMatchRequest.SubmittedOn = time.Now()
		// submit the record match request
		resp, err := ptm_http.Put(svrEndpoint, "application/json+fhir",
			bytes.NewReader(reqBody))
		if err != nil {
			logger.Log.WithFields(
				logrus.Fields{"method": "CreateRecordMatchRun",
					"err": err}).Error("Sending Record Match Request")
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		// Store status, Sent, with the run object
		recMatchRun.Status = make([]ptm_models.RecordMatchRunStatusComponent, 1)
		recMatchRun.Status[0].CreatedOn = time.Now()
		// if a success code was received
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			recMatchRun.Status[0].Message = "Request Sent [" + resp.Status + "]"
		} else {
			recMatchRun.Status[0].Message = "Error Sending Request to Record Matcher [" + resp.Status + "]"
		}

		// Persist the record match run
		resource, err := ptm_models.PersistResource(provider(), "RecordMatchRun", recMatchRun)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		ctx.JSON(http.StatusCreated, resource)
	}
}

func GetRecordMatchRunMetricsHandler(provider func() *mgo.Database) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		resourceType := "RecordMatchRun"

		recordMatchSystemInterfaceId := ctx.Query("recordMatchSystemInterfaceId")
		validRecordMatchSystemInterfaceId := len(recordMatchSystemInterfaceId) > 1 && len(recordMatchSystemInterfaceId) <= 24 && bson.IsObjectIdHex(recordMatchSystemInterfaceId)
		recordSetId := ctx.Query("recordSetId")
		validRecordSetId := len(recordSetId) > 1 && len(recordSetId) <= 24 && bson.IsObjectIdHex(recordSetId)

		logger.Log.WithFields(
			logrus.Fields{"resource type": resourceType, "rec match sys": recordMatchSystemInterfaceId, "record set": recordSetId}).Info("GetRecordMatchRunMetrics")

		resources := ptm_models.NewSliceForResourceName(resourceType, 0, 0)
		c := provider().C(ptm_models.GetCollectionName(resourceType))

		var query *mgo.Query

		if validRecordSetId {
			logger.Log.WithFields(
				// find the record match runs with masterRecordSetId or queryRecordSetId == record set id
				logrus.Fields{"validRecord Set Id": validRecordSetId, "record set": recordSetId}).Info("GetRecordMatchRunMetrics")

			recordSetBsonId, _ := ptm_models.ToBsonObjectID(recordSetId)
			query = c.Find(bson.M{"$or": []bson.M{bson.M{"masterRecordSetId": recordSetBsonId}, bson.M{"queryRecordSetId": recordSetBsonId}}})

		} else if validRecordMatchSystemInterfaceId {
			recordMatchSystemInterfaceBsonId, _ := ptm_models.ToBsonObjectID(recordMatchSystemInterfaceId)
			query = c.Find(bson.M{"recordMatchSystemInterfaceId": recordMatchSystemInterfaceBsonId})

		} else { // no query parameters were provided
			// get all record runs with, primarily, metrics only
			// retrieve all documents in the collection
			// TODO Restrict this to resourc type, just to be extra safe
			query = c.Find(bson.M{})
		}

		// constrain which fields are returned
		err := query.Select(bson.M{"meta": 1, "metrics": 1, "recordMatchSystemInterfaceId": 1, "matchingMode": 1, "recordResourceType": 1, "masterRecordSetId": 1, "queryRecordSetId": 1, "recordMatchContextId": 1}).All(resources)

		if err != nil {
			if err == mgo.ErrNotFound {
				ctx.String(http.StatusNotFound, "Not Found")
				ctx.Abort()
				return
			} else {
				ctx.AbortWithError(http.StatusBadRequest, err)
				return
			}
		}

		ctx.JSON(http.StatusOK, resources)
	}
}

func GetRecordMatchRunLinksHandler(provider func() *mgo.Database) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		idString := ctx.Param("id")
		id := bson.ObjectIdHex(idString)
		c := provider().C(ptm_models.GetCollectionName("RecordMatchRun"))
		rmr := &ptm_models.RecordMatchRun{}
		err := c.Find(bson.M{"_id": id}).One(rmr)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		category := ctx.Query("category")
		limitString := ctx.Query("limit")
		limit, err := strconv.ParseInt(limitString, 10, 0)
		if err != nil || limit == 0 {
			limit = 10
		}
		var links []ptm_models.Link
		if category == "worst" {
			links = rmr.GetWorstLinks(int(limit))
		} else {
			links = rmr.GetBestLinks(int(limit))
		}
		ctx.JSON(http.StatusOK, links)
	}
}

func isValidRecordMatchRun(rmr *ptm_models.RecordMatchRun) bool {
	isValid := false

	if rmr.RecordMatchSystemInterfaceID.Valid() {
		// verify that match mode corresponds to number of specified data lists (query, master)
		if rmr.MatchingMode == ptm_models.Deduplication {
			isValid = rmr.MasterRecordSetID.Valid()
		} else if rmr.MatchingMode == ptm_models.Query {
			isValid = rmr.MasterRecordSetID.Valid() && rmr.QueryRecordSetID.Valid()
		}
	}
	// validate the record match context, if provided
	if isValid && !rmr.RecordMatchContextID.Valid() {
		isValid = false
	}

	return isValid
}

func isValidRecordMatchSysIface(rmsi *ptm_models.RecordMatchSystemInterface) bool {
	isValid := false

	// check that server, destination, and response endpoints are Set
	// TODO check that server, destination, and response endpoint values seem reasonable
	if rmsi.ID.Valid() && rmsi.DestinationEndpoint != "" &&
		rmsi.ServerEndpoint != "" && rmsi.ResponseEndpoint != "" {
		isValid = true
	}
	return isValid
}

func prepEndpoint(baseURL, id string) string {
	result := baseURL

	// if server base doesn't end in/
	if !strings.HasSuffix(baseURL, "/") {
		if !strings.HasSuffix(baseURL, "/Bundle") {
			result += "/Bundle/"
		} else {
			result += "/"
		}
	} else {
		if !strings.HasSuffix(baseURL, "/Bundle/") {
			result += "Bundle/"
		}
	}
	result += id

	return result
}

func newRecordMatchRequest(srcEndpoint string,
	recMatchRun *ptm_models.RecordMatchRun, db *mgo.Database) *ptm_models.RecordMatchRequest {

	req := &ptm_models.RecordMatchRequest{ID: bson.NewObjectId()}
	req.Message = &fhir_models.Bundle{}
	// 2/2016 - Intervention Engine FHIR Server only supports Hex bson ObjectID for Id
	req.Message.Id = bson.NewObjectId().Hex()
	req.Message.Type = "message"

	// deduplication has 2 entries (hdr +_one data); query has 3 (hdr + 2 data)
	numEntries := 2
	if recMatchRun.MatchingMode == ptm_models.Query {
		numEntries = 3
	}
	req.Message.Entry = make([]fhir_models.BundleEntryComponent, numEntries)

	msgHdr, err := newMessageHeader(srcEndpoint, recMatchRun, db)
	if err != nil {
		//TODO What should I do here?  panic?
		panic(fmt.Sprintf("Not IMPL: New Msg Hdr returned error: %s", err.Error()))
	}
	req.Message.Entry[0].Resource = msgHdr
	req.Message.Entry[0].FullUrl = "urn:uuid:" + msgHdr.Id

	addRecordSetParams(recMatchRun, req.Message, db)
	msgHdr.Data = make([]fhir_models.Reference, numEntries-1)

	msgHdr.Data[0].Reference = req.Message.Entry[1].FullUrl

	if numEntries == 3 {
		msgHdr.Data[1].Reference = req.Message.Entry[2].FullUrl
	}

	ptm_models.AddCreatedOn(req)

	logger.Log.WithFields(
		logrus.Fields{
			"match mode":  recMatchRun.MatchingMode,
			"num entries": numEntries}).Debug("NewRecordMatchRequest")

	return req
}

// newMessageHeader constructs a FHIR MessageHeader resource using the information
// associated with the given RecordMatchRun.
func newMessageHeader(
	srcEndpoint string, recMatchRun *ptm_models.RecordMatchRun, db *mgo.Database) (*fhir_models.MessageHeader, error) {
	msgHdr := fhir_models.MessageHeader{}
	msgHdr.Id = uuid.NewV4().String()

	// load the record match system Interface referenced in record match run
	obj, err := ptm_models.LoadResource(
		db, "RecordMatchSystemInterface", recMatchRun.RecordMatchSystemInterfaceID)
	if err != nil {
		return &msgHdr, err
	}
	recMatchSysIface := obj.(*ptm_models.RecordMatchSystemInterface)

	msgHdr.Source = &fhir_models.MessageHeaderMessageSourceComponent{}
	msgHdr.Source.Endpoint = srcEndpoint

	msgHdr.Destination = make([]fhir_models.MessageHeaderMessageDestinationComponent, 1)
	msgHdr.Destination[0].Name = recMatchSysIface.Name
	msgHdr.Destination[0].Endpoint = recMatchSysIface.DestinationEndpoint

	msgHdr.Event = &fhir_models.Coding{
		System: "http://github.com/mitre/ptmatch/fhir/message-events",
		Code:   "record-match"}

	msgHdr.Timestamp = &fhir_models.FHIRDateTime{Time: time.Now(), Precision: fhir_models.Timestamp}

	// TODO Remove MarshalJSON() and Log calls when working
	buf, _ := msgHdr.MarshalJSON()
	logger.Log.WithFields(
		logrus.Fields{
			"msgHdr": string(buf)}).Info("newMessageHeader")

	return &msgHdr, nil
}

func addRecordSetParams(recMatchRun *ptm_models.RecordMatchRun, msg *fhir_models.Bundle, db *mgo.Database) error {
	// retrieve the info for the master record Set
	obj, err := ptm_models.LoadResource(db, "RecordSet", recMatchRun.MasterRecordSetID)
	if err != nil {
		return err
	}
	masterRecSet := obj.(*ptm_models.RecordSet)
	params := buildParams("master", masterRecSet)
	msg.Entry[1].Resource = params
	msg.Entry[1].FullUrl = "urn:uuid:" + params.Id

	logger.Log.WithFields(
		logrus.Fields{"method": "addRecordSetParams",
			"match mode":   recMatchRun.MatchingMode,
			"masterRecSet": masterRecSet}).Debug("addRecordSetParams")

	if recMatchRun.MatchingMode == ptm_models.Query {
		// retrieve the info for the query record set
		obj, err := ptm_models.LoadResource(
			db, "RecordSet", recMatchRun.QueryRecordSetID)
		if err != nil {
			return err
		}
		queryRecSet := obj.(*ptm_models.RecordSet)
		params = buildParams(ptm_models.Query, queryRecSet)
		msg.Entry[2].Resource = params
		msg.Entry[2].FullUrl = "urn:uuid:" + params.Id

		logger.Log.WithFields(
			logrus.Fields{"method": "addRecordSetParams",
				"queryRecSet": queryRecSet}).Info("")

	}

	return nil
}

func buildParams(setType string, recSet *ptm_models.RecordSet) *fhir_models.Parameters {
	params := fhir_models.Parameters{}
	params.Id = uuid.NewV4().String()

	params.Parameter = make([]fhir_models.ParametersParameterComponent, 3)

	params.Parameter[0].Name = "type"
	params.Parameter[0].ValueString = setType

	params.Parameter[1].Name = "resourceType"
	params.Parameter[1].ValueString = recSet.ResourceType

	params.Parameter[2].Name = "searchExpression"
	params.Parameter[2].Resource = recSet.Parameters

	return &params
}
