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

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/intervention-engine/fhir/auth"
	fhirSvr "github.com/intervention-engine/fhir/server"
	"github.com/mitre/heart"
	"github.com/mitre/ptmatch/middleware"
	"github.com/mitre/ptmatch/server"
)

func main() {
	// Check environment variable for a linked MongoDB container
	mongoHost := os.Getenv("MONGO_PORT_27017_TCP_ADDR")
	if mongoHost == "" {
		mongoHost = "localhost"
	}

	s := fhirSvr.NewServer(mongoHost)

	assetPath := flag.String("assets", "", "Path to static assets to host")
	jwkPath := flag.String("heartJWK", "", "Path the JWK for the HEART client")
	clientID := flag.String("heartClientID", "", "Client ID registered with the OP")
	opURL := flag.String("heartOP", "", "URL for the OpenID Provider")
	sessionSecret := flag.String("secret", "", "Secret for the cookie session")

	flag.Parse()

	var authConfig auth.Config

	if *jwkPath != "" {
		if *clientID == "" || *opURL == "" {
			fmt.Println("You must provide both a client ID and OP URL for HEART mode")
			return
		}
		secret := *sessionSecret
		if secret == "" {
			secret = "reallySekret"
		}
		heart.SetUpRoutes(*jwkPath, *clientID, *opURL,
			"http://localhost:3001", secret, s.Engine)
	}

	recMatchWatch := middleware.PostProcessRecordMatchResponse()
	s.AddMiddleware("Bundle", recMatchWatch)

	ar := func(e *gin.Engine) {
		server.Setup(e)

		if *assetPath != "" {
			e.StaticFile("/", fmt.Sprintf("%s/index.html", *assetPath))
			e.Static("/assets", fmt.Sprintf("%s/assets", *assetPath))
		}
	}

	s.AfterRoutes = append(s.AfterRoutes, ar)
	s.Run(fhirSvr.Config{Auth: authConfig,
		ServerURL:    "http://localhost:3001",
		DatabaseName: "fhir"})
}
