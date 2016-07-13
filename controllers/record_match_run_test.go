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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	. "gopkg.in/check.v1"
	"gopkg.in/mgo.v2"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/mitre/ptmatch/logger"
	"github.com/pebbe/util"

	ptm_models "github.com/mitre/ptmatch/models"
)

var (
	mongoSession *mgo.Session
	database     *mgo.Database
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
}

func (s *ServerSuite) TearDownTest(c *C) {
	if database != nil {
		database.C("recordMatchRuns").DropCollection()
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

func (s *ServerSuite) TestGetRecordMatchRunLinks(c *C) {
	resource := ptm_models.InsertResourceFromFile(database, "RecordMatchRun", "../fixtures/record-match-run-responses.json")
	rmr := resource.(*ptm_models.RecordMatchRun)
	provider := func() *mgo.Database { return database }
	handler := GetRecordMatchRunLinksHandler(provider)
	url := fmt.Sprintf("/RecordMatchRunLinks/%s?limit=2", rmr.ID.Hex())
	r, err := http.NewRequest("GET", url, nil)
	util.CheckErr(err)
	e := gin.New()
	rw := httptest.NewRecorder()
	e.GET("/RecordMatchRunLinks/:id", handler)
	e.ServeHTTP(rw, r)
	c.Assert(rw.Code, Equals, http.StatusOK)
	var links []ptm_models.Link
	decoder := json.NewDecoder(rw.Body)
	err = decoder.Decode(&links)
	util.CheckErr(err)
	c.Assert(len(links), Equals, 2)
	lastLink := links[1]
	c.Assert(lastLink.Score, Equals, 0.82)
	c.Assert(lastLink.Source, Equals, "http://localhost:3001/Patient/5616b69a1cd462440e0006ae")
	c.Assert(lastLink.Target, Equals, "http://localhost:3001/Patient/57335da265ddb433bd30f0ee")
	c.Assert(lastLink.Match, Equals, "probable")
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

	// Insert a corresponding record match run to the DB
	recMatchRun := &ptm_models.RecordMatchRun{}
	ptm_models.LoadResourceFromFile("../fixtures/record-match-run-post-01.json", recMatchRun)
	recMatchRun.RecordMatchSystemInterfaceID = recMatchSysIface.ID

	// Build a record match run
	// construct a record match request
	req := newRecordMatchRequest("http://localhost/fhir", recMatchRun, database)
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

	recMatchRun := &ptm_models.RecordMatchRun{}
	recMatchRun.RecordMatchSystemInterfaceID = recMatchSysIface.ID
	recMatchRun.MatchingMode = "query"
	recMatchRun.MasterRecordSetID = masterRecSet.ID
	recMatchRun.QueryRecordSetID = queryRecSet.ID

	// Build a record match run
	// construct a record match request
	req := newRecordMatchRequest("http://replace.me/with/selurl/global", recMatchRun, database)
	buf, _ := req.Message.MarshalJSON()
	logger.Log.WithFields(
		logrus.Fields{"method": "TestNewRecordMatchQueryRequest",
			"request": string(buf)}).Info("")
	c.Assert(req, NotNil)
	c.Assert(req.Message.Type, Equals, "message")
	c.Assert(len(req.Message.Entry), Equals, 3) //MsgHdr + two Param`
}

// TestNewMessageHeader tests that a MessageHeader for a record-match
// request is constructed from information in a RecordMatchRun object.
func (s *ServerSuite) TestNewMessageHeader(c *C) {
	// Insert a record match system interface to the DB
	r := ptm_models.InsertResourceFromFile(database, "RecordMatchSystemInterface", "../fixtures/record-match-sys-if-01.json")
	recMatchSysIface := r.(*ptm_models.RecordMatchSystemInterface)
	c.Assert(*recMatchSysIface, NotNil)

	recMatchRun := &ptm_models.RecordMatchRun{}
	ptm_models.LoadResourceFromFile("../fixtures/record-match-run-post-01.json", recMatchRun)
	recMatchRun.RecordMatchSystemInterfaceID = recMatchSysIface.ID

	// Build a record match run
	// construct a record match request
	src := "http://replace.me/with/selurl/global"
	msgHdr, _ := newMessageHeader(src, recMatchRun, database)
	c.Assert(msgHdr, NotNil)
	c.Assert(msgHdr.Source.Endpoint, Equals, src)
	c.Assert(msgHdr.Event.Code, Equals, "record-match")
}
