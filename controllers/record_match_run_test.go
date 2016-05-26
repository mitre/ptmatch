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
	"testing"

	. "gopkg.in/check.v1"
	"gopkg.in/mgo.v2"

	"github.com/Sirupsen/logrus"
	"github.com/mitre/ptmatch/logger"

	ptm_models "github.com/mitre/ptmatch/models"
)

var (
	mongoSession *mgo.Session
	database     *mgo.Database
	controller   ResourceController
)

type ServerSuite struct {
	DatabaseHost string
	DatabaseName string
	ListenPort   string
}

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&ServerSuite{"localhost", "ptmatch-test", ":8882"})

// runs once
func (s *ServerSuite) SetUpSuite(c *C) {
	var err error

	// Set up the database
	if mongoSession, err = mgo.Dial(s.DatabaseHost); err != nil {
		logger.Log.Error("Cannot connect to MongoDB. Is service running?")
		panic(err)
	}
	c.Assert(mongoSession, NotNil)

	database = mongoSession.DB(s.DatabaseName)
	c.Assert(database, NotNil)

	controller = ResourceController{}
	controller.DatabaseProvider = func() *mgo.Database { return database }
}

func (s *ServerSuite) TearDownTest(c *C) {
	if database != nil {
		database.C("recordMatchContexts").DropCollection()
		database.C("recordMatchSystemInterfaces").DropCollection()
	}
}

func (s *ServerSuite) TearDownSuite(c *C) {
	if database != nil {
		database.DropDatabase()
	}
	if mongoSession != nil {
		mongoSession.Close()
	}
}

func (s *ServerSuite) TestNewRecordMatchDedupRequest(c *C) {
	// Insert a record match system interface to the DB
	r := ptm_models.InsertResourceFromFile(database, "RecordMatchSystemInterface", "../fixtures/record-match-sys-if-01.json")
	recMatchSysIface := r.(*ptm_models.RecordMatchSystemInterface)
	c.Assert(*recMatchSysIface, NotNil)

	// Insert a record set to the DB
	r = ptm_models.InsertResourceFromFile(database, "RecordSet", "../fixtures/record-set-01.json")
	masterRecSet := r.(*ptm_models.RecordSet)
	c.Assert(*masterRecSet, NotNil)

	// Insert a corresponding record match context to the DB
	recMatchContext := &ptm_models.RecordMatchContext{}
	ptm_models.LoadResourceFromFile("../fixtures/record-match-context-01.json", recMatchContext)
	ptm_models.PersistResource(controller.Database(), "RecordMatchContext", recMatchContext)
	c.Assert(*recMatchContext, NotNil)

	// Build a record match run
	// construct a record match request
	req := controller.newRecordMatchRequest("http://localhost/fhir", recMatchContext)
	buf, _ := req.Message.MarshalJSON()
	logger.Log.WithFields(
		logrus.Fields{"method": "TestNewRecordMatchDedupRequest",
			"request": string(buf)}).Info("")
	c.Assert(req, NotNil)
	c.Assert(req.Message.Type, Equals, "message")
	c.Assert(len(req.Message.Entry), Equals, 2) //MsgHdr + Param`
}

func (s *ServerSuite) TestNewRecordMatchQueryRequest(c *C) {
	// Insert a record match system interface to the DB
	r := ptm_models.InsertResourceFromFile(database, "RecordMatchSystemInterface", "../fixtures/record-match-sys-if-01.json")
	recMatchSysIface := r.(*ptm_models.RecordMatchSystemInterface)
	c.Assert(*recMatchSysIface, NotNil)

	// Insert a record set to the DB
	r = ptm_models.InsertResourceFromFile(database, "RecordSet", "../fixtures/record-set-01.json")
	masterRecSet := r.(*ptm_models.RecordSet)
	c.Assert(*masterRecSet, NotNil)

	// Insert a 2nd record set to the DB
	r = ptm_models.InsertResourceFromFile(database, "RecordSet", "../fixtures/record-set-02.json")
	queryRecSet := r.(*ptm_models.RecordSet)
	c.Assert(*queryRecSet, NotNil)

	// Insert a corresponding record match context to the DB
	recMatchContext := &ptm_models.RecordMatchContext{}
	ptm_models.LoadResourceFromFile("../fixtures/record-match-context-01.json", recMatchContext)
	ptm_models.PersistResource(controller.Database(), "RecordMatchContext", recMatchContext)
	c.Assert(*recMatchContext, NotNil)

	// Build a record match run
	// construct a record match request
	req := controller.newRecordMatchRequest("http://replace.me/with/selurl/global", recMatchContext)
	buf, _ := req.Message.MarshalJSON()
	logger.Log.WithFields(
		logrus.Fields{"method": "TestNewRecordMatchQueryRequest",
			"request": string(buf)}).Info("")
	c.Assert(req, NotNil)
	c.Assert(req.Message.Type, Equals, "message")
	c.Assert(len(req.Message.Entry), Equals, 3) //MsgHdr + two Param`
}

// TestNewMessageHeader tests that a MessageHeader for a record-match
// request is constructed from information in a RecordMatchContext object.
func (s *ServerSuite) TestNewMessageHeader(c *C) {
	c.Assert(controller.Database, NotNil)
	// Insert a record match system interface to the DB
	r := ptm_models.InsertResourceFromFile(database, "RecordMatchSystemInterface", "../fixtures/record-match-sys-if-01.json")
	recMatchSysIface := r.(*ptm_models.RecordMatchSystemInterface)
	c.Assert(*recMatchSysIface, NotNil)

	// Insert a corresponding record match context to the DB
	recMatchContext := &ptm_models.RecordMatchContext{}
	ptm_models.LoadResourceFromFile("../fixtures/record-match-context-01.json", recMatchContext)
	ptm_models.PersistResource(controller.Database(), "RecordMatchContext", recMatchContext)
	c.Assert(*recMatchContext, NotNil)

	// Build a record match run
	// construct a record match request
	src := "http://replace.me/with/selurl/global"
	msgHdr, _ := controller.newMessageHeader(src, recMatchContext)
	c.Assert(msgHdr, NotNil)
	c.Assert(msgHdr.Source.Endpoint, Equals, src)
	c.Assert(msgHdr.Event.Code, Equals, "record-match")
}
