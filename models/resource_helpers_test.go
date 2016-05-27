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

package models

import (
	"testing"

	. "gopkg.in/check.v1"
)

type ServerSuite struct {
}

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&ServerSuite{})

// runs once
func (s *ServerSuite) SetUpSuite(c *C) {
}

func (s *ServerSuite) TestPluralizeLowerResourceName(c *C) {
	var names = []string{"RecordMatchContext",
		"SomeUnknownName", "RecordSet", "Message"}
	var expected = []string{"recordMatchContexts",
		"someUnknownNames", "recordSets", "messages"}

	for i, name := range names {
		actual := PluralizeLowerResourceName(name)
		c.Assert(actual, Equals, expected[i])
	}
}
