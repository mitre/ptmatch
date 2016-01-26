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

var _ = Suite(&ServerSuite{"localhost", "pophealth-test", ":8882"})

// runs once
func (s *ServerSuite) SetUpSuite(c *C) {
	var err error

	// Set up the database
	if mongoSession, err = mgo.Dial(s.DatabaseHost); err != nil {
		panic(err)
	}

	database = mongoSession.DB(s.DatabaseName)

	controller = ResourceController{}
	controller.Database = database
}

func (s *ServerSuite) TearDownTest(c *C) {
	database.C("recordMatchConfigurations").DropCollection()
	database.C("recordMatchSystemInterfaces").DropCollection()
}

func (s *ServerSuite) TearDownSuite(c *C) {
	database.DropDatabase()
	mongoSession.Close()
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

	// Insert a corresponding record match configuration to the DB
	recMatchConfig := &ptm_models.RecordMatchConfiguration{}
	ptm_models.LoadResourceFromFile("../fixtures/record-match-config-01.json", recMatchConfig)
	recMatchConfig.RecordMatchSystemInterfaceID = recMatchSysIface.ID
	recMatchConfig.MasterRecordSetID = masterRecSet.ID
	ptm_models.PersistResource(controller.Database, "RecordMatchConfiguration", recMatchConfig)
	c.Assert(*recMatchConfig, NotNil)

	// Build a record match run
	// construct a record match request
	req := controller.newRecordMatchRequest("http://replace.me/with/selurl/global", recMatchConfig)
	buf, _ := req.Message.MarshalJSON()
	logger.Log.WithFields(
		logrus.Fields{"method": "TestNewRecordMatchDedupRequest",
			"request": string(buf)}).Info("")
	c.Assert(req, NotNil)
	c.Assert(req.Message.Type, Equals, "message")
	c.Assert(len(req.Message.Entry), Equals, 2)
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

	// Insert a corresponding record match configuration to the DB
	recMatchConfig := &ptm_models.RecordMatchConfiguration{}
	ptm_models.LoadResourceFromFile("../fixtures/record-match-config-01.json", recMatchConfig)
	recMatchConfig.RecordMatchSystemInterfaceID = recMatchSysIface.ID
	recMatchConfig.MasterRecordSetID = masterRecSet.ID
	recMatchConfig.QueryRecordSetID = queryRecSet.ID
	recMatchConfig.MatchingMode = ptm_models.Query
	ptm_models.PersistResource(controller.Database, "RecordMatchConfiguration", recMatchConfig)
	c.Assert(*recMatchConfig, NotNil)

	// Build a record match run
	// construct a record match request
	req := controller.newRecordMatchRequest("http://replace.me/with/selurl/global", recMatchConfig)
	buf, _ := req.Message.MarshalJSON()
	logger.Log.WithFields(
		logrus.Fields{"method": "TestNewRecordMatchQueryRequest",
			"request": string(buf)}).Info("")
	c.Assert(req, NotNil)
	c.Assert(req.Message.Type, Equals, "message")
	c.Assert(len(req.Message.Entry), Equals, 3)
}

func (s *ServerSuite) TestNewMessageHeader(c *C) {
	c.Assert(controller.Database, NotNil)
	// Insert a record match system interface to the DB
	r := ptm_models.InsertResourceFromFile(database, "RecordMatchSystemInterface", "../fixtures/record-match-sys-if-01.json")
	recMatchSysIface := r.(*ptm_models.RecordMatchSystemInterface)
	c.Assert(*recMatchSysIface, NotNil)

	// Insert a corresponding record match configuration to the DB
	recMatchConfig := &ptm_models.RecordMatchConfiguration{}
	ptm_models.LoadResourceFromFile("../fixtures/record-match-config-01.json", recMatchConfig)
	recMatchConfig.RecordMatchSystemInterfaceID = recMatchSysIface.ID
	ptm_models.PersistResource(controller.Database, "RecordMatchConfiguration", recMatchConfig)
	c.Assert(*recMatchConfig, NotNil)

	// Build a record match run
	// construct a record match request
	msgHdr, _ := controller.newMessageHeader("http://replace.me/with/selurl/global", recMatchConfig)
	c.Assert(msgHdr, NotNil)
}
