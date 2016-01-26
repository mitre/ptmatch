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
	"net/http"

	"github.com/labstack/echo"
	emw "github.com/labstack/echo/middleware"
	"github.com/thoas/stats"

	rc "github.com/mitre/ptmatch/controllers"
	logger "github.com/mitre/ptmatch/logger"
	mw "github.com/mitre/ptmatch/middleware"

	fhir_svr "github.com/intervention-engine/fhir/server"

	"gopkg.in/mgo.v2"
)

type RecMatchServer struct {
	DatabaseName string
	ListenPort   string
	FhirSvr      *fhir_svr.FHIRServer
}

func (svr *RecMatchServer) AddMiddleware(key string, middleware echo.Middleware) {
	svr.FhirSvr.MiddlewareConfig[key] = append(svr.FhirSvr.MiddlewareConfig[key], middleware)
}

func (svr *RecMatchServer) DatabaseHost() string {
	return svr.FhirSvr.DatabaseHost
}

func (svr *RecMatchServer) Router() *echo.Echo {
	return svr.FhirSvr.Echo
}

func Database() *mgo.Database {
	return fhir_svr.Database
}

func SetDatabase(db *mgo.Database) {
	fhir_svr.Database = db
}

func NewServer(databaseHost string, dbName string, listenPort string) *RecMatchServer {
	// TODO Validate database host name
	// TODO Validate listenPort value

	svr := &RecMatchServer{DatabaseName: dbName, ListenPort: listenPort}

	svr.FhirSvr = &fhir_svr.FHIRServer{DatabaseHost: databaseHost,
		MiddlewareConfig: make(map[string][]echo.Middleware)}

	// create echo http routing framework instance
	svr.FhirSvr.Echo = echo.New()

	return svr
}

func (svr *RecMatchServer) Run() {
	var err error

	// Setup the database
	if fhir_svr.MongoSession, err = mgo.Dial(svr.DatabaseHost()); err != nil {
		panic(err)
	}
	logger.Log.Info("Connected to mongodb")
	defer fhir_svr.MongoSession.Close()

	fhir_svr.Database = fhir_svr.MongoSession.DB(svr.DatabaseName)

	registerMiddleware(svr)
	registerRoutes(svr)

	svr.Router().Run(svr.ListenPort)
}

func registerMiddleware(svr *RecMatchServer) {
	svr.Router().Use(emw.Logger())
	svr.Router().Use(emw.Recover())
	svr.Router().Use(emw.Gzip())

	//------------------------
	// Third-party middleware
	//------------------------
	// https://github.com/thoas/stats
	s := stats.New()
	svr.Router().Use(s.Handler)
	// Route
	svr.Router().Get("/stats", func(c *echo.Context) error {
		logger.Log.Info("In stats")
		return c.JSON(http.StatusOK, s.Data())
	})

	//echoSvr.Use(emw.AllowOrigin("*"))

	recMatchWatch := mw.ProcessFhirResource(fhir_svr.Database)
	svr.AddMiddleware("Bundle", recMatchWatch)

}

func registerRoutes(svr *RecMatchServer) {
	svr.Router().Get("/", welcome)

	controller := rc.ResourceController{}
	controller.Database = Database()

	resourceNames := []string{"RecordMatchConfiguration",
		"RecordMatchSystemInterface", "RecordSet"}

	for _, name := range resourceNames {
		svr.Router().Get("/"+name, controller.GetResources)
		svr.Router().Get("/"+name+"/:id", controller.GetResource)
		svr.Router().Post("/"+name, controller.CreateResource)
		svr.Router().Put("/"+name+"/:id", controller.UpdateResource)
		svr.Router().Delete("/"+name+"/:id", controller.DeleteResource)
	}

	name := "RecordMatchRun"
	svr.Router().Get("/"+name, controller.GetResources)
	svr.Router().Get("/"+name+"/:id", controller.GetResource)
	svr.Router().Post("/"+name, controller.CreateRecordMatchRun)
	svr.Router().Put("/"+name+"/:id", controller.UpdateResource)
	svr.Router().Delete("/"+name+"/:id", controller.DeleteResource)
}

func welcome(c *echo.Context) error {
	return c.String(http.StatusOK, "Patient Matching Test Harness Server")
}
