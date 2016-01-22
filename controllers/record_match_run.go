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
	"time"

	"gopkg.in/mgo.v2/bson"

	"github.com/Sirupsen/logrus"
	"github.com/labstack/echo"
	"github.com/satori/go.uuid"

	fhir_models "github.com/intervention-engine/fhir/models"
	logger "github.com/mitre/ptmatch/logger"
	ptm_models "github.com/mitre/ptmatch/models"
)

func (rc *ResourceController) CreateRecordMatchRun(ctx *echo.Context) error {
	req := ctx.Request()
	resourceType := getResourceType(req.URL)
	obj := ptm_models.NewStructForResourceName(resourceType)
	recMatchRun := obj.(*ptm_models.RecordMatchRun)
	if err := ctx.Bind(recMatchRun); err != nil {
		return err
	}

	// retrieve and validate the record match configuration
	recMatchConfigID := recMatchRun.RecordMatchConfigurationID
	logger.Log.WithFields(
		logrus.Fields{"method": "CreateRecordMatchRun",
			"recMatchConfigID": recMatchConfigID}).Info("check recmatch config id")
	if !recMatchConfigID.Valid() {
		// Bad Request: Record Match Configuration is required
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid RecordMatchConfigurationID")
	}
	obj, err := ptm_models.LoadResource(rc.Database, "RecordMatchConfiguration", recMatchConfigID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Unable to find Record Match Configuration")
	}
	recMatchConfig := obj.(*ptm_models.RecordMatchConfiguration)
	if err = validateRecordMatchConfig(recMatchConfig); err != nil {
		return err
	}

	obj, err = ptm_models.LoadResource(rc.Database, "RecordMatchSystemInterface",
		recMatchConfig.RecordMatchSystemInterfaceID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Unable to find Record Match System Interface")
	}
	recMatchSysIface := obj.(*ptm_models.RecordMatchSystemInterface)
	if err != nil {
		return err
	}

	// construct a record match request
	reqMatchRequest := rc.newRecordMatchRequest(recMatchSysIface.ResponseEndpoint, recMatchConfig)
	// attach the request message to the run object
	recMatchRun.Request = *reqMatchRequest

	// Construct http request for the record match run reqMatchRequest
	reqBody, _ := reqMatchRequest.Message.MarshalJSON()

	logger.Log.WithFields(
		logrus.Fields{"method": "CreateRecordMatchRun",
			"request": string(reqBody)}).Info("About to submit request")

	// submit the record match request
	resp, err := http.Post(recMatchSysIface.ServerEndpoint, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	logger.Log.WithFields(
		logrus.Fields{"method": "CreateRecordMatchRun",
			"response": resp}).Info("Request Submission Response")

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
	resource, err := ptm_models.PersistResource(rc.Database, resourceType, recMatchRun)
	if err != nil {
		return err
	}

	/*
		logger.Log.WithFields(
			logrus.Fields{"collection": ptm_models.GetCollectionName(resourceType),
				"res type": resourceType, "id": id}).Info("CreateResource")
	*/
	//reflect.ValueOf(resource).Elem().FieldByName("ID").Set(reflect.ValueOf(id))
	return ctx.JSON(http.StatusCreated, resource)
}

func validateRecordMatchConfig(recMatchConfig *ptm_models.RecordMatchConfiguration) error {
	// TODO check that server, destination, and response endpoints are Set
	// TODO verify that match mode corresponds to number of specified data lists (query, master)
	return nil
}

func (rc *ResourceController) newRecordMatchRequest(srcEndpoint string,
	recMatchConfig *ptm_models.RecordMatchConfiguration) *ptm_models.RecordMatchRequest {

	req := &ptm_models.RecordMatchRequest{ID: bson.NewObjectId()}
	req.Message = &fhir_models.Bundle{}
	req.Message.Id = uuid.NewV4().String()
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
			"num entries": numEntries}).Info("")

	return req
}

func (rc *ResourceController) newMessageHeader(
	srcEndpoint string, recMatchConfig *ptm_models.RecordMatchConfiguration) (*fhir_models.MessageHeader, error) {
	msgHdr := fhir_models.MessageHeader{}
	msgHdr.Id = uuid.NewV4().String()

	// load the record match system Interface referenced in record match config
	obj, err := ptm_models.LoadResource(
		rc.Database, "RecordMatchSystemInterface", recMatchConfig.RecordMatchSystemInterfaceID)
	if err != nil {
		return &msgHdr, err
	}
	recMatchSysIface := obj.(*ptm_models.RecordMatchSystemInterface)

	msgHdr.Source = &fhir_models.MessageHeaderMessageSourceComponent{}
	msgHdr.Source.Endpoint = srcEndpoint

	msgHdr.Destination = make([]fhir_models.MessageHeaderMessageDestinationComponent, 1)
	msgHdr.Destination[0].Name = recMatchSysIface.Name
	msgHdr.Destination[0].Endpoint = recMatchSysIface.DestinationEndpoint

	msgHdr.Event = &fhir_models.Coding{System: "http://github.com/mitre/ptmatch", Code: "record-match"}

	msgHdr.Timestamp = &fhir_models.FHIRDateTime{Time: time.Now(), Precision: fhir_models.Timestamp}

	// TODO Remove when working
	buf, _ := msgHdr.MarshalJSON()
	logger.Log.WithFields(
		logrus.Fields{"method": "newMessageHeader",
			"msgHdr": string(buf)}).Info("")

	return &msgHdr, nil
}

func (rc *ResourceController) addRecordSetParams(recMatchConfig *ptm_models.RecordMatchConfiguration, msg *fhir_models.Bundle) error {

	// retrieve the info for the master record Set
	obj, err := ptm_models.LoadResource(
		rc.Database, "RecordSet", recMatchConfig.MasterRecordSetID)
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
			rc.Database, "RecordSet", recMatchConfig.QueryRecordSetID)
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
