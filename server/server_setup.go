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

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
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

func (svr *RecMatchServer) AddMiddleware(key string, middleware gin.HandlerFunc) {
	svr.FhirSvr.AddMiddleware(key, middleware)
	//	svr.FhirSvr.MiddlewareConfig[key] = append(svr.FhirSvr.MiddlewareConfig[key], middleware)
}

func (svr *RecMatchServer) DatabaseHost() string {
	return svr.FhirSvr.DatabaseHost
}

func (svr *RecMatchServer) Router() *gin.Engine {
	return svr.FhirSvr.Engine
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

	svr.FhirSvr = fhir_svr.NewServer(databaseHost)

	return svr
}

func (svr *RecMatchServer) Run() {
	logger.Log.WithFields(
		logrus.Fields{"method": "Run",
			"bundle Mware": svr.FhirSvr.MiddlewareConfig["Bundle"]}).Info("before set")

	registerMiddleware(svr)
	registerRoutes(svr)
	logger.Log.WithFields(
		logrus.Fields{"method": "Run",
			"bundle Mware": svr.FhirSvr.MiddlewareConfig["Bundle"]}).Info("mware set?")

	svr.FhirSvr.Run(fhir_svr.Config{})
}

func registerMiddleware(svr *RecMatchServer) {

	//------------------------
	// Third-party middleware
	//------------------------
	// See https://github.com/thoas/stats
	s := stats.New()
	svr.FhirSvr.Engine.Use(func(ctx *gin.Context) {
		beginning, recorder := s.Begin(ctx.Writer)
		ctx.Next()
		s.End(beginning, recorder)
	})
	// Route
	svr.FhirSvr.Engine.GET("/stats", func(c *gin.Context) {
		logger.Log.Info("In stats")
		c.JSON(http.StatusOK, s.Data())
	})

	recMatchWatch := mw.PostProcessRecordMatchResponse(fhir_svr.Database)

	//	recMatchWatch := mw.PostProcessFhirResource("PUT", fhir_svr.Database)
	svr.AddMiddleware("Bundle", recMatchWatch)
}

func registerRoutes(svr *RecMatchServer) {
	svr.FhirSvr.Engine.GET("/", welcome)

	controller := rc.ResourceController{}
	controller.DatabaseProvider = Database

	resourceNames := []string{"RecordMatchConfiguration",
		"RecordMatchSystemInterface", "RecordSet"}

	for _, name := range resourceNames {
		svr.FhirSvr.Engine.GET("/"+name, controller.GetResources)
		svr.FhirSvr.Engine.GET("/"+name+"/:id", controller.GetResource)
		svr.FhirSvr.Engine.POST("/"+name, controller.CreateResource)
		svr.FhirSvr.Engine.PUT("/"+name+"/:id", controller.UpdateResource)
		svr.FhirSvr.Engine.DELETE("/"+name+"/:id", controller.DeleteResource)
	}

	name := "RecordMatchJob"
	svr.FhirSvr.Engine.GET("/"+name, controller.GetResources)
	svr.FhirSvr.Engine.GET("/"+name+"/:id", controller.GetResource)
	svr.FhirSvr.Engine.POST("/"+name, controller.CreateRecordMatchJob)
	svr.FhirSvr.Engine.PUT("/"+name+"/:id", controller.UpdateResource)
	svr.FhirSvr.Engine.DELETE("/"+name+"/:id", controller.DeleteResource)
}

func welcome(c *gin.Context) {
	c.String(http.StatusOK, "Patient Matching Test Harness Server")
}
