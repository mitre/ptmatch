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

	"gopkg.in/mgo.v2"
)

var (
	MongoSession *mgo.Session
	Database     *mgo.Database
)

type PopHealthServer struct {
	DatabaseHost string
	DatabaseName string
	ListenPort   string
	Router       *echo.Echo
}

func NewServer(databaseHost string, dbName string, listenPort string) *PopHealthServer {
	// TODO Validate database host name
	// TODO Validate listenPort value

	svr := &PopHealthServer{DatabaseHost: databaseHost,
		DatabaseName: dbName, ListenPort: listenPort}

	// create echo http routing framework instance
	svr.Router = echo.New()

	return svr
}

func (svr *PopHealthServer) Run() {
	var err error

	// Setup the database
	if MongoSession, err = mgo.Dial(svr.DatabaseHost); err != nil {
		panic(err)
	}
	logger.Log.Info("Connected to mongodb")
	defer MongoSession.Close()

	Database = MongoSession.DB(svr.DatabaseName)

	registerMiddleware(svr.Router)
	registerRoutes(svr)

	svr.Router.Run(svr.ListenPort)
}

func registerMiddleware(echoSvr *echo.Echo) {
	echoSvr.Use(emw.Logger())
	echoSvr.Use(emw.Recover())
	echoSvr.Use(emw.Gzip())

	//------------------------
	// Third-party middleware
	//------------------------
	// https://github.com/thoas/stats
	s := stats.New()
	echoSvr.Use(s.Handler)
	// Route
	echoSvr.Get("/stats", func(c *echo.Context) error {
		logger.Log.Info("In stats")
		return c.JSON(http.StatusOK, s.Data())
	})

	//echoSvr.Use(emw.AllowOrigin("*"))
}

func registerRoutes(svr *PopHealthServer) {
	svr.Router.Get("/", welcome)

	controller := rc.ResourceController{}
	controller.Database = Database

	resourceNames := []string{"RecordMatchConfiguration",
		"RecordMatchSystemInterface", "RecordSet"}

	for _, name := range resourceNames {
		svr.Router.Get("/"+name, controller.GetResources)
		svr.Router.Get("/"+name+"/:id", controller.GetResource)
		svr.Router.Post("/"+name, controller.CreateResource)
		svr.Router.Put("/"+name+"/:id", controller.UpdateResource)
		svr.Router.Delete("/"+name+"/:id", controller.DeleteResource)
	}

	name := "RecordMatchRun"
	svr.Router.Get("/"+name, controller.GetResources)
	svr.Router.Get("/"+name+"/:id", controller.GetResource)
	svr.Router.Post("/"+name, controller.CreateRecordMatchRun)
	svr.Router.Put("/"+name+"/:id", controller.UpdateResource)
	svr.Router.Delete("/"+name+"/:id", controller.DeleteResource)
}

func welcome(c *echo.Context) error {
	return c.String(http.StatusOK, "PopHealth Server")
}
