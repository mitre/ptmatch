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

	"github.com/gin-gonic/gin"
	"github.com/thoas/stats"

	rc "github.com/mitre/ptmatch/controllers"
	logger "github.com/mitre/ptmatch/logger"
	"gopkg.in/mgo.v2"

	fhir_svr "github.com/intervention-engine/fhir/server"
)

// Database is a convenience function to obtain a pointer to the fhir server database.
func Database() *mgo.Database {
	return fhir_svr.Database
}

// Setup performs initialization of the middleware and routes.
func Setup(e *gin.Engine) {
	registerMiddleware(e)
	registerRoutes(e)
}

func registerMiddleware(e *gin.Engine) {
	//------------------------
	// Third-party middleware
	//------------------------
	// See https://github.com/thoas/stats
	s := stats.New()
	e.Use(func(ctx *gin.Context) {
		beginning, recorder := s.Begin(ctx.Writer)
		ctx.Next()
		s.End(beginning, recorder)
	})
	// Route
	e.GET("/stats", func(c *gin.Context) {
		logger.Log.Info("In stats")
		c.JSON(http.StatusOK, s.Data())
	})
}

func registerRoutes(e *gin.Engine) {
	controller := rc.ResourceController{}
	controller.DatabaseProvider = Database

	resourceNames := []string{"RecordMatchContext",
		"RecordMatchSystemInterface", "RecordSet"}

	for _, name := range resourceNames {
		e.GET("/"+name+"/:id", controller.GetResource)
		e.POST("/"+name, controller.CreateResource)
		e.PUT("/"+name+"/:id", controller.UpdateResource)
		e.DELETE("/"+name+"/:id", controller.DeleteResource)
		e.GET("/"+name, controller.GetResources)
	}

	e.POST("/AnswerKey", controller.SetAnswerKey)

	name := "RecordMatchRun"
	e.GET("/"+name, controller.GetResources)
	e.GET("/"+name+"/:id", controller.GetResource)
	e.POST("/"+name, rc.CreateRecordMatchRunHandler(Database))
	e.PUT("/"+name+"/:id", controller.UpdateResource)
	e.DELETE("/"+name+"/:id", controller.DeleteResource)

	e.GET("/RecordMatchRunMetrics", rc.GetRecordMatchRunMetricsHandler(Database))
	e.GET("/RecordMatchRunLinks/:id", rc.GetRecordMatchRunLinksHandler(Database))

	e.Static("/ptmatch/api/", "api")
}
