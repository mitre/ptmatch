FHIR Patient Generator [![Build Status](https://travis-ci.org/intervention-engine/ptgen.svg?branch=master)](https://travis-ci.org/intervention-engine/ptgen)
============================================================================================================================================================

The *ptgen* project is a (partial) Go port of https://github.com/jnazarian1/Patient-Generator. This Go library generates synthetic patient records using the [HL7 FHIR DSTU2](http://hl7.org/fhir/DSTU2/index.html) models defined in the Intervention Engine [fhir](https://github.com/interventionengine/fhir) project.

*NOTE: Due to Intervention Engine's prominent use case, all synthetic records are tuned to a geriatric population. At this time, patient demographics, simple observations, office visits, conditions, and medications are generated.*

Building ptgen Locally
----------------------

For information on installing and running the full Intervention Engine stack, please see [Building and Running the Intervention Engine Stack in a Development Environment](https://github.com/intervention-engine/ie/blob/master/docs/dev_install.md).

The *ptgen* project is a Go library. For information related specifically to building the code in this repository (*ptgen*), please refer to the following sections in the above guide:

-	(Prerequisite) [Install Git](https://github.com/intervention-engine/ie/blob/master/docs/dev_install.md#install-git)
-	(Prerequisite) [Install Go](https://github.com/intervention-engine/ie/blob/master/docs/dev_install.md#install-go)
-	[Clone ptgen Repository](https://github.com/intervention-engine/ie/blob/master/docs/dev_install.md#clone-ptgen-repository)

To build the *ptgen* library, you must install its dependencies via `go get` first, and then build it:

```
$ cd $GOPATH/src/github.com/intervention-engine/ptgen
$ go get
$ go build
```

For information on using the *generate* tool to create synthetic patient records and upload them to a FHIR server, please refer to the [generate](https://github.com/intervention-engine/tools#generate) section of the [tools](https://github.com/intervention-engine/tools) repository README.

Using ptgen as a library
------------------------

The following is a simple example of generating the FHIR resources to represent a single synthetic patient:

```go
import "github.com/intervention-engine/ptgen"

func ExamplePtGeneration() []interface{} {
	return ptgen.GeneratePatient()
}
```

License
-------

Copyright 2016 The MITRE Corporation

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
