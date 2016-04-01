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

package server

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/Sirupsen/logrus"

	. "gopkg.in/check.v1"
	"gopkg.in/mgo.v2/bson"

	fhir_models "github.com/intervention-engine/fhir/models"
	logger "github.com/mitre/ptmatch/logger"
	ptm_models "github.com/mitre/ptmatch/models"
)

func (s *ServerSuite) TestRecordMatchJobResponse(c *C) {
	var r interface{}
	// Insert a record match run object to the DB
	//	r := ptm_models.InsertResourceFromFile(Database(), "RecordMatchJob", "../fixtures/record-match-run-01.json")
	recMatchJob := &ptm_models.RecordMatchJob{}
	ptm_models.LoadResourceFromFile("../fixtures/record-match-run-01.json", recMatchJob)
	//	recMatchJob := r.(*ptm_models.RecordMatchJob)
	c.Assert(*recMatchJob, NotNil)
	// confirm initial state - zero responses assoc. w/ run
	c.Assert(len(recMatchJob.Responses), Equals, 0)
	// Assign New Identifier to request message
	//	recMatchJob.Request.Message.Enry[0].Resource.ID = bson.NewObjectId()
	reqMsg := recMatchJob.Request.Message
	reqMsgHdr := reqMsg.Entry[0].Resource.(*fhir_models.MessageHeader)
	recMatchJob.Request.ID = bson.NewObjectId()
	reqMsg.Id = bson.NewObjectId().Hex()
	reqMsgHdr.Id = bson.NewObjectId().Hex()

	ptm_models.PersistResource(Database(), "RecordMatchJob", recMatchJob)

	logger.Log.WithFields(
		logrus.Fields{"func": "TestRecordMatchJobResponse",
			"recMatchJob": recMatchJob}).Info("after insert recMatchJob")

	respMsg := &fhir_models.Bundle{}
	// Load text of a record match ack message
	ptm_models.LoadResourceFromFile("../fixtures/record-match-ack-01.json", respMsg)
	//respMsg := r.(*fhir_models.Bundle)
	c.Assert(*respMsg, NotNil)

	// Ensure the response references the request
	//	reqMsg := recMatchJob.Request.Message
	c.Assert(reqMsg, NotNil)
	c.Assert(reqMsg.Type, Equals, "message")
	c.Assert(len(reqMsg.Entry) > 1, Equals, true)
	c.Assert(reqMsg.Entry[0].Resource, NotNil)
	//	reqMsgHdr := reqMsg.Entry[0].Resource.(*fhir_models.MessageHeader)
	respMsgHdr := respMsg.Entry[0].Resource.(*fhir_models.MessageHeader)
	respMsgHdr.Response.Identifier = reqMsgHdr.Id

	buf, _ := respMsg.MarshalJSON()
	logger.Log.WithFields(
		logrus.Fields{"func": "TestRecordMatchJobResponse",
			"resp msg": string(buf)}).Info("prep to POST")

	e := s.Server.Router()

	code, body := request("POST", "/Bundle",
		bytes.NewReader(buf), "application/json", e)
	c.Assert(code, Equals, http.StatusCreated)
	c.Assert(body, NotNil)
	logger.Log.Info("Post Record Match response: " + body)

	// Load the record match run object -- this time from database
	r, err := ptm_models.LoadResource(Database(), "RecordMatchJob", recMatchJob.ID)
	c.Assert(err, IsNil)
	c.Assert(r, NotNil)
	recMatchJob = r.(*ptm_models.RecordMatchJob)

	logger.Log.WithFields(
		logrus.Fields{"func": "TestRecordMatchJobResponse",
			"recMatchJob": recMatchJob}).Info("after recv response")

	// The response should be attached to the record match run ObjectId
	c.Assert(len(recMatchJob.Responses), Equals, 1)
	c.Assert(recMatchJob.Responses[0].ID, NotNil)
	c.Assert(recMatchJob.Responses[0].Message, NotNil)
	var respMsg1 *fhir_models.Bundle
	respMsg1 = recMatchJob.Responses[0].Message

	// After inserting into database, we've lost knowledge about type of resource in response message
	// so we use hack to decode to and then encode from json to get map to struct
	respMsgHdr1 := &fhir_models.MessageHeader{}
	mapToStruct(respMsg1.Entry[0].Resource.(bson.M), respMsgHdr1)
	c.Assert(respMsgHdr1.Response.Identifier, Equals, respMsgHdr.Response.Identifier)
}

func mapToStruct(m map[string]interface{}, val interface{}) error {
	tmp, err := json.Marshal(m)
	if err != nil {
		return err
	}
	err = json.Unmarshal(tmp, val)
	if err != nil {
		return err
	}
	return nil
}
