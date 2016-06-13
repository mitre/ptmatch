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
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/pebbe/util"
	. "gopkg.in/check.v1"
	"gopkg.in/mgo.v2"

	fhir_svr "github.com/intervention-engine/fhir/server"
	logger "github.com/mitre/ptmatch/logger"
	ptm_models "github.com/mitre/ptmatch/models"
)

type ServerSuite struct {
	Server *fhir_svr.FHIRServer
}

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&ServerSuite{})

// runs once
func (s *ServerSuite) SetUpSuite(c *C) {
	var err error

	s.Server = fhir_svr.NewServer("localhost")

	var mongoSession *mgo.Session
	// Set up the database
	if mongoSession, err = mgo.Dial("localhost"); err != nil {
		logger.Log.Error("Cannot connect to MongoDB. Is service running?")
		panic(err)
	}
	c.Assert(mongoSession, NotNil)
	database := mongoSession.DB("ptmatch-test")
	fhir_svr.Database = database

	Setup(s.Server)

	fhir_svr.RegisterRoutes(s.Server.Engine,
		s.Server.MiddlewareConfig, fhir_svr.NewMongoDataAccessLayer(database),
		fhir_svr.Config{})
}

// Clean the database between each test run to avoid conflict
func (s *ServerSuite) TearDownTest(c *C) {
	fhir_svr.Database.C("recordMatchRuns").DropCollection()
	fhir_svr.Database.C("recordMatchContexts").DropCollection()
	fhir_svr.Database.C("recordMatchSystemInterfaces").DropCollection()
	fhir_svr.Database.C("recordSets").DropCollection()
}

func (s *ServerSuite) TestEchoRoutes(c *C) {
	e := gin.New()

	h := func(*gin.Context) {}
	routes := []struct {
		Method string
		Path   string
	}{
		{"GET", "/RecordMatchContext"},
		{"GET", "/RecordMatchContext/:id"},
		{"POST", "/RecordMatchContext"},
		{"PUT", "/RecordMatchContext/:id"},
		{"DELETE", "/RecordMatchContext/:id"},
	}
	for _, r := range routes {
		switch r.Method {
		case "GET":
			e.GET(r.Path, h)
		case "PUT":
			e.PUT(r.Path, h)
		case "POST":
			e.POST(r.Path, h)
		case "DELETE":
			e.DELETE(r.Path, h)
		case "PATCH":
			e.PATCH(r.Path, h)
		}
	}

	for i, r := range e.Routes() {
		c.Assert(routes[i].Method, Equals, r.Method)
		c.Assert(routes[i].Path, Equals, r.Path)
	}
}

func (s *ServerSuite) TestSearch(c *C) {
	ptm_models.InsertResourceFromFile(Database(), "RecordMatchRun", "../fixtures/record-match-run-01.json")
	ptm_models.InsertResourceFromFile(Database(), "RecordMatchRun", "../fixtures/record-match-run-02.json")
	code, body := request("GET", "/RecordMatchRun?recordMatchContextId=56a21d9aa291020ca7dd225f", nil, "", s.Server.Engine)
	c.Assert(code, Equals, http.StatusOK)
	decoder := json.NewDecoder(bytes.NewBufferString(body))
	var runs []ptm_models.RecordMatchRun
	err := decoder.Decode(&runs)
	util.CheckErr(err)
	c.Assert(len(runs), Equals, 1)

}

func (s *ServerSuite) TestGetRecordMatchContexts(c *C) {
	recs := [4]*ptm_models.RecordMatchContext{}

	// Add record match contexts
	for i := 0; i < len(recs); i++ {
		r := ptm_models.InsertResourceFromFile(Database(), "RecordMatchContext", "../fixtures/record-match-context-01.json")
		recs[i] = r.(*ptm_models.RecordMatchContext)
	}

	e := s.Server.Engine

	// retrieve collection of record match contexts
	code, body := request("GET", "/RecordMatchContext", nil, "", e)
	c.Assert(code, Equals, http.StatusOK)
	logger.Log.Debug("response body: " + body)

	// check that ID of each created resource is in response
	for i, rec := range recs {
		logger.Log.Debug("Chk in Body i: " + strconv.Itoa(i) + " resource id: " + rec.ID.Hex())
		// confirm response body contains resource ID
		pat := regexp.MustCompile(rec.ID.Hex())
		c.Assert(pat.MatchString(body), Equals, true)
	}

	// Try to Get each resource created above
	for _, rec := range recs {
		path := "/RecordMatchContext/" + rec.ID.Hex()
		code, body = request("GET", path, nil, "", e)
		c.Assert(code, Equals, http.StatusOK)
		// confirm response body contains resource ID
		pat := regexp.MustCompile(rec.ID.Hex())
		c.Assert(pat.MatchString(body), Equals, true)
	}

	// Delete the resources created above
	for _, rec := range recs {
		path := "/RecordMatchContext/" + rec.ID.Hex()
		logger.Log.Debug("About to call DELETE: " + path)
		code, body = request("DELETE", path, nil, "", e)
		c.Assert(code, Equals, http.StatusNoContent)
	}

	code, body = request("GET", "/RecordMatchContext", nil, "", e)
	c.Assert(code, Equals, http.StatusOK)

	for i, rec := range recs {
		logger.Log.Debug("Delete, i: " + strconv.Itoa(i) + " resource id: " + rec.ID.Hex())
		// confirm response body no longer contains resource ID
		pat := regexp.MustCompile(rec.ID.Hex())
		c.Assert(pat.MatchString(body), Equals, false)
	}
}

func (s *ServerSuite) TestPostRecordMatchSystemInterface(c *C) {
	buf, err := ioutil.ReadFile("../fixtures/record-match-sys-if-01.json")
	c.Assert(err, IsNil)

	e := s.Server.Engine

	code, body := request("POST", "/RecordMatchSystemInterface",
		bytes.NewReader(buf), "application/json", e)
	c.Assert(code, Equals, http.StatusCreated)
	c.Assert(body, NotNil)
	logger.Log.Info("response body: " + body)

	// Verify the response can be decoded into the expected resource type
	resource := ptm_models.RecordMatchSystemInterface{}
	decoder := json.NewDecoder(bytes.NewBufferString(body))
	r := &resource
	err = decoder.Decode(r)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{"err": err, "resource": resource,
			"func": "TestPostRecordMatchSystemInterface"}).Info("check error")
	}
	c.Assert(err, IsNil)

	// verify that a createdOn date was added to the resource
	var meta = resource.Meta
	c.Assert(meta.CreatedOn, NotNil)
	// verify that a lastUpdateOn date was added to the resource
	c.Assert(meta.LastUpdatedOn, NotNil)

	// Delete the resource created above
	path := "/RecordMatchSystemInterface/" + resource.ID.Hex()
	code, body = request("DELETE", path, nil, "", e)
	c.Assert(code, Equals, http.StatusNoContent)

	// Verify the resource was deleted
	code, body = request("GET", path, nil, "", e)
	c.Assert(code, Equals, http.StatusNotFound)
}

func (s *ServerSuite) TestPutRecordMatchSystemInterface(c *C) {
	buf, err := ioutil.ReadFile("../fixtures/record-match-sys-if-01.json")
	c.Assert(err, IsNil)

	e := s.Server.Engine

	// Post resource
	code, body := request("POST", "/RecordMatchSystemInterface",
		bytes.NewReader(buf), "application/json", e)
	c.Assert(code, Equals, http.StatusCreated)

	// Decode response to get the resource identifier
	resource := ptm_models.RecordMatchSystemInterface{}
	decoder := json.NewDecoder(bytes.NewBufferString(body))
	r := &resource
	err = decoder.Decode(r)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{"resource": resource,
			"error": err,
			"func":  "TestPutRecordMatchSystemInterface"}).Info("Check Error")
	}
	c.Assert(err, IsNil)

	origDesc := resource.Description

	// grab the original values of metadata fields
	var meta = resource.Meta
	c.Assert(meta.CreatedOn, NotNil)
	// verify that a lastUpdateOn date was added to the resource
	c.Assert(meta.LastUpdatedOn, NotNil)
	origCreatedOn := meta.CreatedOn
	origLastUpdatedOn := meta.LastUpdatedOn

	buf, err = ioutil.ReadFile("../fixtures/record-match-sys-if-02.json")
	c.Assert(err, IsNil)

	path := "/RecordMatchSystemInterface/" + resource.ID.Hex()
	logger.Log.WithFields(logrus.Fields{"path": path,
		"func": "TestPutRecordMatchSystemInterface"}).Info("Prepare for Put")
	time.Sleep(time.Millisecond)
	// Submit a known change; the resource ID in the URL should override the one in the body
	code, body = request("PUT", path,
		bytes.NewReader(buf), "application/json", e)
	c.Assert(code, Equals, http.StatusOK)
	c.Assert(body, NotNil)

	// Decode response to get the updated resource
	resource = ptm_models.RecordMatchSystemInterface{}
	decoder = json.NewDecoder(bytes.NewBufferString(body))
	r = &resource
	err = decoder.Decode(r)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{"resource": resource,
			"error": err,
			"func":  "TestPutRecordMatchSystemInterface"}).Info("after putting update")
	}
	c.Assert(err, IsNil)

	logger.Log.WithFields(logrus.Fields{"orig creation": origCreatedOn,
		"creation": resource.Meta.CreatedOn,
		"func":     "TestPutRecordMatchSystemInterface"}).Info("check vars")

	// Verify the description has changed
	c.Assert(resource.Description, Not(Equals), origDesc)
	// Update should not affect the creation date
	c.Assert(resource.Meta.CreatedOn, Equals, origCreatedOn)
	c.Assert(resource.Meta.LastUpdatedOn, Not(Equals), origLastUpdatedOn)

	// Delete the resource created above
	code, body = request("DELETE", path, nil, "", e)
	c.Assert(code, Equals, http.StatusNoContent)
}

func (s *ServerSuite) TestPostRecordSet(c *C) {
	buf, err := ioutil.ReadFile("../fixtures/record-set-01.json")
	c.Assert(err, IsNil)

	e := s.Server.Engine

	code, body := request("POST", "/RecordSet",
		bytes.NewReader(buf), "application/json", e)
	c.Assert(code, Equals, http.StatusCreated)
	c.Assert(body, NotNil)
	logger.Log.Debug("Post Record Set response: " + body)

	// Verify the response can be decoded into the expected resource type
	resource := ptm_models.RecordSet{}
	decoder := json.NewDecoder(bytes.NewBufferString(body))
	r := &resource
	err = decoder.Decode(r)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{"err": err, "resource": resource,
			"func": "TestPostRecordSet"}).Info("check decode error")
	}
	c.Assert(err, IsNil)

	// verify that a createdOn date was added to the resource
	var meta = resource.Meta
	c.Assert(meta.CreatedOn, NotNil)
	// verify that a lastUpdateOn date was added to the resource
	c.Assert(meta.LastUpdatedOn, NotNil)

	// Delete the resource created above
	path := "/RecordSet/" + resource.ID.Hex()
	code, body = request("DELETE", path, nil, "", e)
	c.Assert(code, Equals, http.StatusNoContent)

	// Verify the resource was deleted
	code, body = request("GET", path, nil, "", e)
	c.Assert(code, Equals, http.StatusNotFound)
}

func request(method, path string, body io.Reader, ct string, e *gin.Engine) (int, string) {
	r, _ := http.NewRequest(method, path, body)
	if body != nil && ct != "" {
		r.Header.Set("Content-Type", ct)
	}
	w := httptest.NewRecorder()
	e.ServeHTTP(w, r)
	return w.Code, w.Body.String()
}
