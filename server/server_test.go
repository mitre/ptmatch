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

	"github.com/Sirupsen/logrus"
	"github.com/labstack/echo"
	. "gopkg.in/check.v1"
	"gopkg.in/mgo.v2"

	fhir_svr "github.com/intervention-engine/fhir/server"
	logger "github.com/mitre/ptmatch/logger"
	ptm_models "github.com/mitre/ptmatch/models"
)

type ServerSuite struct {
	Server *RecMatchServer
}

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&ServerSuite{})

// runs once
func (s *ServerSuite) SetUpSuite(c *C) {
	var err error

	s.Server = NewServer("localhost", "recmatch-test", ":8882")

	var mongoSession *mgo.Session
	// Set up the database
	if mongoSession, err = mgo.Dial(s.Server.DatabaseHost()); err != nil {
		panic(err)
	}

	fhir_svr.MongoSession = mongoSession
	//	database = mongoSession.DB(s.Server.DatabaseName)
	SetDatabase(fhir_svr.MongoSession.DB(s.Server.DatabaseName))

	registerMiddleware(s.Server)
	registerRoutes(s.Server)
	fhir_svr.RegisterRoutes(s.Server.FhirSvr.Echo, s.Server.FhirSvr.MiddlewareConfig)
}

// SetUpTest is invoked before each test
func (s *ServerSuite) SetUpTest(c *C) {

}

func (s *ServerSuite) TearDownTest(c *C) {
	/*
		Database().C("recordMatchConfigurations").DropCollection()
		Database().C("recordMatchSystemInterfaces").DropCollection()
		Database().C("recordMatchSets").DropCollection()
		Database().C("recordMatchRuns").DropCollection()
	*/
}

func (s *ServerSuite) TearDownSuite(c *C) {
	//	Database().DropDatabase()
	fhir_svr.MongoSession.Close()
}

func (s *ServerSuite) TestEchoRoutes(c *C) {
	e := echo.New()
	h := func(*echo.Context) error { return nil }
	routes := []echo.Route{
		{echo.GET, "/RecordMatchConfiguration", h},
		{echo.GET, "/RecordMatchConfiguration/:id", h},
		{echo.POST, "/RecordMatchConfiguration", h},
		{echo.PUT, "/RecordMatchConfiguration/:id", h},
		{echo.DELETE, "/RecordMatchConfiguration/:id", h},
	}
	for _, r := range routes {
		switch r.Method {
		case echo.GET:
			e.Get(r.Path, h)
		case echo.PUT:
			e.Put(r.Path, h)
		case echo.POST:
			e.Post(r.Path, h)
		case echo.DELETE:
			e.Delete(r.Path, h)
		case echo.PATCH:
			e.Patch(r.Path, h)
		}
	}

	for i, r := range e.Routes() {
		c.Assert(routes[i].Method, Equals, r.Method)
		c.Assert(routes[i].Path, Equals, r.Path)
	}
}

func (s *ServerSuite) TestGetRecordMatchConfigurations(c *C) {
	recs := [4]*ptm_models.RecordMatchConfiguration{}

	// Add record match configurations
	for i := 0; i < len(recs); i++ {
		r := ptm_models.InsertResourceFromFile(Database(), "RecordMatchConfiguration", "../fixtures/record-match-config-01.json")
		recs[i] = r.(*ptm_models.RecordMatchConfiguration)
	}

	e := s.Server.Router()

	// retrieve collection of record match configurations
	code, body := request(echo.GET, "/RecordMatchConfiguration", nil, "", e)
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
		path := "/RecordMatchConfiguration/" + rec.ID.Hex()
		code, body = request(echo.GET, path, nil, "", e)
		c.Assert(code, Equals, http.StatusOK)
		// confirm response body contains resource ID
		pat := regexp.MustCompile(rec.ID.Hex())
		c.Assert(pat.MatchString(body), Equals, true)
	}

	// Delete the resources created above
	for _, rec := range recs {
		path := "/RecordMatchConfiguration/" + rec.ID.Hex()
		logger.Log.Debug("About to call DELETE: " + path)
		code, body = request(echo.DELETE, path, nil, "", e)
		c.Assert(code, Equals, http.StatusNoContent)
	}

	code, body = request(echo.GET, "/RecordMatchConfiguration", nil, "", e)
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

	e := s.Server.Router()

	code, body := request(echo.POST, "/RecordMatchSystemInterface",
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
	var meta *ptm_models.Meta = resource.Meta
	c.Assert(meta.CreatedOn, NotNil)
	// verify that a lastUpdateOn date was added to the resource
	c.Assert(meta.LastUpdatedOn, NotNil)

	// Delete the resource created above
	path := "/RecordMatchSystemInterface/" + resource.ID.Hex()
	code, body = request(echo.DELETE, path, nil, "", e)
	c.Assert(code, Equals, http.StatusNoContent)

	// Verify the resource was deleted
	code, body = request(echo.GET, path, nil, "", e)
	c.Assert(code, Equals, http.StatusNotFound)
}

func (s *ServerSuite) TestPutRecordMatchSystemInterface(c *C) {
	buf, err := ioutil.ReadFile("../fixtures/record-match-sys-if-01.json")
	c.Assert(err, IsNil)

	e := s.Server.Router()

	// Post resource
	code, body := request(echo.POST, "/RecordMatchSystemInterface",
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
	var meta *ptm_models.Meta = resource.Meta
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
	// Submit a known change; the resource ID in the URL should override the one in the body
	code, body = request(echo.PUT, path,
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
	code, body = request(echo.DELETE, path, nil, "", e)
	c.Assert(code, Equals, http.StatusNoContent)
}

func (s *ServerSuite) TestPostRecordSet(c *C) {
	buf, err := ioutil.ReadFile("../fixtures/record-set-01.json")
	c.Assert(err, IsNil)

	e := s.Server.Router()

	code, body := request(echo.POST, "/RecordSet",
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
	var meta *ptm_models.Meta = resource.Meta
	c.Assert(meta.CreatedOn, NotNil)
	// verify that a lastUpdateOn date was added to the resource
	c.Assert(meta.LastUpdatedOn, NotNil)

	// Delete the resource created above
	path := "/RecordSet/" + resource.ID.Hex()
	code, body = request(echo.DELETE, path, nil, "", e)
	c.Assert(code, Equals, http.StatusNoContent)

	// Verify the resource was deleted
	code, body = request(echo.GET, path, nil, "", e)
	c.Assert(code, Equals, http.StatusNotFound)
}

func request(method, path string, body io.Reader, ct string, e *echo.Echo) (int, string) {
	r, _ := http.NewRequest(method, path, body)
	if body != nil && ct != "" {
		r.Header.Set("Content-Type", ct)
	}
	w := httptest.NewRecorder()
	e.ServeHTTP(w, r)
	return w.Code, w.Body.String()
}
