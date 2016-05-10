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

	"gopkg.in/mgo.v2"
	rc "github.com/mitre/ptmatch/controllers"
	logger "github.com/mitre/ptmatch/logger"
	mw "github.com/mitre/ptmatch/middleware"

	fhir_svr "github.com/intervention-engine/fhir/server"
)

// Database is a convenience function to obtain a pointer to the fhir server database.
func Database() *mgo.Database {
	return fhir_svr.Database
}

// Setup performs initialization of the middleware and routes.
func Setup(svr *fhir_svr.FHIRServer) {
	logger.Log.WithFields(
		logrus.Fields{"method": "Setup",
			"bundle Mware": svr.MiddlewareConfig["Bundle"]}).Info("before set")

	registerMiddleware(svr)
	registerRoutes(svr)
	logger.Log.WithFields(
		logrus.Fields{"method": "Setup",
			"bundle Mware": svr.MiddlewareConfig["Bundle"]}).Info("mware set?")
}

func registerMiddleware(svr *fhir_svr.FHIRServer) {
	//------------------------
	// Third-party middleware
	//------------------------
	// See https://github.com/thoas/stats
	s := stats.New()
	svr.Engine.Use(func(ctx *gin.Context) {
		beginning, recorder := s.Begin(ctx.Writer)
		ctx.Next()
		s.End(beginning, recorder)
	})
	// Route
	svr.Engine.GET("/stats", func(c *gin.Context) {
		logger.Log.Info("In stats")
		c.JSON(http.StatusOK, s.Data())
	})

	recMatchWatch := mw.PostProcessRecordMatchResponse()

	svr.AddMiddleware("Bundle", recMatchWatch)
}

func registerRoutes(svr *fhir_svr.FHIRServer) {
	controller := rc.ResourceController{}
	controller.DatabaseProvider = Database

	resourceNames := []string{"RecordMatchConfiguration",
		"RecordMatchSystemInterface", "RecordSet"}

	for _, name := range resourceNames {
		svr.Engine.GET("/"+name+"/:id", controller.GetResource)
		svr.Engine.POST("/"+name, controller.CreateResource)
		svr.Engine.PUT("/"+name+"/:id", controller.UpdateResource)
		svr.Engine.DELETE("/"+name+"/:id", controller.DeleteResource)
		svr.Engine.GET("/"+name, controller.GetResources)
	}

	svr.Engine.POST("/AnswerKey", controller.SetAnswerKey)

	name := "RecordMatchJob"
	svr.Engine.GET("/"+name, controller.GetResources)
	svr.Engine.GET("/"+name+"/:id", controller.GetResource)
	svr.Engine.POST("/"+name, controller.CreateRecordMatchJob)
	svr.Engine.PUT("/"+name+"/:id", controller.UpdateResource)
	svr.Engine.DELETE("/"+name+"/:id", controller.DeleteResource)

	svr.Engine.GET("/RecordMatchJobMetrics", controller.GetRecordMatchJobMetrics)
}
