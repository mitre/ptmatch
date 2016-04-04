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
	"strings"
	"time"

	"gopkg.in/mgo.v2/bson"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/satori/go.uuid"

	fhir_models "github.com/intervention-engine/fhir/models"
	ptm_http "github.com/mitre/ptmatch/http"
	logger "github.com/mitre/ptmatch/logger"
	ptm_models "github.com/mitre/ptmatch/models"
)

// CreateRecordMatchJob creates a new Record Match Job and constructs and
// sends a Record Match request message.
func (rc *ResourceController) CreateRecordMatchJob(ctx *gin.Context) {
	req := ctx.Request
	resourceType := getResourceType(req.URL)
	obj := ptm_models.NewStructForResourceName(resourceType)
	recMatchJob := obj.(*ptm_models.RecordMatchJob)
	if err := ctx.Bind(recMatchJob); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// retrieve and validate the record match configuration
	recMatchConfigID := recMatchJob.RecordMatchConfigurationID
	logger.Log.WithFields(
		logrus.Fields{"method": "CreateRecordMatchJob",
			"recMatchConfigID": recMatchConfigID}).Debug("check recmatch config id")
	if !recMatchConfigID.Valid() {
		// Bad Request: Record Match Configuration is required
		ctx.String(http.StatusBadRequest, "Invalid RecordMatchConfigurationID")
		ctx.Abort()
		return
	}
	// Retrieve the RecordMatchConfiguration specified in the run object
	obj, err := ptm_models.LoadResource(rc.Database(), "RecordMatchConfiguration", recMatchConfigID)
	if err != nil {
		ctx.String(http.StatusBadRequest, "Unable to find Record Match Configuration")
		ctx.Abort()
		return
	}
	recMatchConfig := obj.(*ptm_models.RecordMatchConfiguration)
	if !isValidRecordMatchConfig(recMatchConfig) {
		ctx.String(http.StatusBadRequest, "Invalid Record Match Configuration")
		ctx.Abort()
		return
	}

	// Retrieve the info about the record matcher
	obj, err = ptm_models.LoadResource(rc.Database(), "RecordMatchSystemInterface",
		recMatchConfig.RecordMatchSystemInterfaceID)
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
	reqMatchRequest := rc.newRecordMatchRequest(recMatchSysIface.ResponseEndpoint, recMatchConfig)
	// attach the request message to the run object
	recMatchJob.Request = *reqMatchRequest

	// Construct body of the http request for the record match request
	reqBody, _ := reqMatchRequest.Message.MarshalJSON()

	svrEndpoint := prepEndpoint(recMatchSysIface.ServerEndpoint, reqMatchRequest.Message.Id)

	logger.Log.WithFields(
		logrus.Fields{"method": "CreateRecordMatchJob",
			"server endpoint": svrEndpoint,
			"request":         reqMatchRequest}).Info("About to submit request")

	reqMatchRequest.SubmittedOn = time.Now()
	// submit the record match request
	resp, err := ptm_http.Put(svrEndpoint, "application/json+fhir",
		bytes.NewReader(reqBody))
	if err != nil {
		logger.Log.WithFields(
			logrus.Fields{"method": "CreateRecordMatchJob",
				"err": err}).Error("Sending Record Match Request")
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Store status, Sent, with the run object
	recMatchJob.Status = make([]ptm_models.RecordMatchJobStatusComponent, 1)
	recMatchJob.Status[0].CreatedOn = time.Now()
	// if a success code was received
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		recMatchJob.Status[0].Message = "Request Sent [" + resp.Status + "]"
	} else {
		recMatchJob.Status[0].Message = "Error Sending Request to Record Matcher [" + resp.Status + "]"
	}

	// Persist the record match run
	resource, err := ptm_models.PersistResource(rc.Database(), resourceType, recMatchJob)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	/*
		logger.Log.WithFields(
			logrus.Fields{"collection": ptm_models.GetCollectionName(resourceType),
				"res type": resourceType, "id": id}).Info("CreateResource")
	*/
	//reflect.ValueOf(resource).Elem().FieldByName("ID").Set(reflect.ValueOf(id))
	ctx.JSON(http.StatusCreated, resource)
}

func isValidRecordMatchConfig(rmc *ptm_models.RecordMatchConfiguration) bool {
	isValid := false

	if rmc.RecordMatchSystemInterfaceID.Valid() {
		// verify that match mode corresponds to number of specified data lists (query, master)
		if rmc.MatchingMode == ptm_models.Deduplication {
			isValid = rmc.MasterRecordSetID.Valid()
		} else if rmc.MatchingMode == ptm_models.Query {
			isValid = rmc.MasterRecordSetID.Valid() && rmc.QueryRecordSetID.Valid()
		}
	}

	return isValid
}

func isValidRecordMatchSysIface(rmsi *ptm_models.RecordMatchSystemInterface) bool {
	isValid := false

	// check that server, destination, and response endpoints are Set
	// TODO check that server, destination, and response endpoint values seem reasonable
	if rmsi.DestinationEndpoint != "" &&
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

func (rc *ResourceController) newRecordMatchRequest(srcEndpoint string,
	recMatchConfig *ptm_models.RecordMatchConfiguration) *ptm_models.RecordMatchRequest {

	req := &ptm_models.RecordMatchRequest{ID: bson.NewObjectId()}
	req.Message = &fhir_models.Bundle{}
	// 2/2016 - Intervention Engine FHIR Server only supports Hex bson ObjectID for Id
	req.Message.Id = bson.NewObjectId().Hex()
	req.Message.Type = "message"

	// deduplication has 2 entries (hdr +_one data); query has 3 (hdr + 2 data)
	numEntries := 2
	if recMatchConfig.MatchingMode == ptm_models.Query {
		numEntries = 3
	}
	req.Message.Entry = make([]fhir_models.BundleEntryComponent, numEntries)

	msgHdr, err := rc.newMessageHeader(srcEndpoint, recMatchConfig)
	if err != nil {
		//TODO What should I do here?  panic?
		panic(fmt.Sprintf("Not IMPL: New Msg Hdr returned error: %s", err.Error()))
	}
	req.Message.Entry[0].Resource = msgHdr
	req.Message.Entry[0].FullUrl = "urn:uuid:" + msgHdr.Id

	rc.addRecordSetParams(recMatchConfig, req.Message)
	msgHdr.Data = make([]fhir_models.Reference, numEntries-1)

	msgHdr.Data[0].Reference = req.Message.Entry[1].FullUrl

	if numEntries == 3 {
		msgHdr.Data[1].Reference = req.Message.Entry[2].FullUrl
	}

	ptm_models.AddCreatedOn(req)

	logger.Log.WithFields(
		logrus.Fields{"method": "NewRecordMatchRequest",
			"match mode":  recMatchConfig.MatchingMode,
			"num entries": numEntries}).Debug("")

	return req
}

func (rc *ResourceController) newMessageHeader(
	srcEndpoint string, recMatchConfig *ptm_models.RecordMatchConfiguration) (*fhir_models.MessageHeader, error) {
	msgHdr := fhir_models.MessageHeader{}
	msgHdr.Id = uuid.NewV4().String()

	// load the record match system Interface referenced in record match config
	obj, err := ptm_models.LoadResource(
		rc.Database(), "RecordMatchSystemInterface", recMatchConfig.RecordMatchSystemInterfaceID)
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
		logrus.Fields{"method": "newMessageHeader",
			"msgHdr": string(buf)}).Info("")

	return &msgHdr, nil
}

func (rc *ResourceController) addRecordSetParams(recMatchConfig *ptm_models.RecordMatchConfiguration, msg *fhir_models.Bundle) error {

	// retrieve the info for the master record Set
	obj, err := ptm_models.LoadResource(
		rc.Database(), "RecordSet", recMatchConfig.MasterRecordSetID)
	if err != nil {
		return err
	}
	masterRecSet := obj.(*ptm_models.RecordSet)
	params := buildParams("master", masterRecSet)
	msg.Entry[1].Resource = params
	msg.Entry[1].FullUrl = "urn:uuid:" + params.Id

	logger.Log.WithFields(
		logrus.Fields{"method": "addRecordSetParams",
			"match mode":   recMatchConfig.MatchingMode,
			"masterRecSet": masterRecSet}).Info("")

	if recMatchConfig.MatchingMode == ptm_models.Query {
		// retrieve the info for the query record set
		obj, err := ptm_models.LoadResource(
			rc.Database(), "RecordSet", recMatchConfig.QueryRecordSetID)
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
