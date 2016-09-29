# ptmatch

This project is an implementation of the services described in [Patient Matching Test Harness Interface](http://mitre.github.io/test-harness-interface/).
It provides the REST/JSON based services to handle the management of record matching system entries, test data sets and matching runs.

This project builds on the [Intervention Engine FHIR Server](https://github.com/intervention-engine/fhir) for all FHIR services. A web
based user interface for this project is provided in the [Patient Match Frontend Project](https://github.com/mitre/ptmatch-frontend).

## Environment

This project currently uses Go 1.7 and is built using the Go toolchain.

To install Go, follow the instructions found at the [Go Website](http://golang.org/doc/install).

Following standard Go practices, you should clone this project to:

    $GOPATH/src/github.com/mitre/ptmatch

Assuming your working directory is $GOPATH/src/github.com/mitre, the git command will look like:

    git clone https://github.com/mitre/ptmatch.git

This project uses [Glide](https://github.com/Masterminds/glide) to manage dependencies. To get all of
the needed run:

    go get github.com/Masterminds/glide
    glide install

To run all of the tests for this project, run:

    go test $(glide novendor)

in this directory.

This project also requires MongoDB 3.2.* or higher. To install MongoDB, refer to the
[MongoDB installation guide](http://docs.mongodb.org/manual/installation/).

To start the application, simply run main.go:

    go run main.go

You can also run the application with the assets flag to serve static assets:

    go run main -assets PATH_TO_ASSETS

In this case, PATH_TO_ASSETS should be a location where a version  
[Patient Match Frontend](https://github.com/mitre/ptmatch-frontend) has been built.

## HEART authentication and authorization:

This server has the ability to authenticate users by acting as a [HEART](http://openid.net/wg/heart/)
compliant OpenID Connect relying party. It can also perform OAuth 2.0 token
introspection in a HEART compliant manner. To enable it, the following command
line flags must be used:

    -heartJWK - The path to the client's private key in [JWK format](https://tools.ietf.org/html/rfc7517). The
                public key must be registered at the OpenID Connect Provider
    -heartOP - The URL of the HEART compliant OpenID Connect Provider
    -heartClientID - The client identifier for this system as registered at the OpenID Connect Provider

Note: The test harness is not able to work with an FHIR server acting as a message broker between
the test harness and record matching system.  This condition is presumed if the
test harness receives a redirect from the message broker. When this occurs, the
test harness persists the record match request in its own database. No issues
should be encountered when the test harness acts as the message broker.

## License

Copyright 2016 The MITRE Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
