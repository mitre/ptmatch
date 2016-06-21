# ptmatch

This project is an implementation of the services described in [Patient Matching Test Harness Interface](http://mitre.github.io/test-harness-interface/).
It provides the REST/JSON based services to handle the management of record matching system entries, test data sets and matching runs.

This project builds on the [Intervention Engine FHIR Server](https://github.com/intervention-engine/fhir) for all FHIR services. A web
based user interface for this project is provided in the [Patient Match Frontend Project](https://github.com/mitre/ptmatch-frontend).

## Environment

This project currently uses Go 1.6 and is built using the Go toolchain.

To install Go, follow the instructions found at the [Go Website](http://golang.org/doc/install).

Following standard Go practices, you should clone this project to:

    $GOPATH/src/github.com/mitre/ptmatch

Assuming your working directory is $GOPATH/src/github.com/mitre, the git command will look like:

    git clone https://github.com/mitre/ptmatch.git

This project uses [Godep](https://github.com/tools/godep) to manage dependencies. All of the needed related
libraries are included in the vendor directory.

To run all of the tests for this project, run:

    go test $(go list ./... | grep -v /vendor/)

in this directory.

This project also requires MongoDB 3.2.* or higher. To install MongoDB, refer to the
[MongoDB installation guide](http://docs.mongodb.org/manual/installation/).

To start the application, simply run main.go:

    go run main.go


## API Documentation

When the test harness is running, documentation on the REST API can be accessed at:

<server base url>/ptmatch/api

The default base url for the test harness server is http://localhost:3001

The OPEN API (aka Swagger 2.0) (https://github.com/OAI/OpenAPI-Specification)
YAML file for the test harness' REST API is in the api folder.

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
