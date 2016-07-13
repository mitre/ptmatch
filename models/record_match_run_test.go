package models

import (
	"encoding/json"
	"os"

	"github.com/pebbe/util"
	. "gopkg.in/check.v1"
)

type RecordMatchRunSuite struct {
	Run *RecordMatchRun
}

var _ = Suite(&RecordMatchRunSuite{})

func (r *RecordMatchRunSuite) SetUpSuite(c *C) {
	data, err := os.Open("../fixtures/record-match-run-responses.json")
	util.CheckErr(err)
	defer data.Close()

	decoder := json.NewDecoder(data)
	rmr := &RecordMatchRun{}
	err = decoder.Decode(rmr)
	util.CheckErr(err)
	r.Run = rmr
}

func (r *RecordMatchRunSuite) TestGetLinks(c *C) {
	links := r.Run.GetLinks()
	c.Assert(len(links), Equals, 3)
	firstLink := links[0]
	c.Assert(firstLink.Score, Equals, 0.51)
	c.Assert(firstLink.Source, Equals, "http://localhost:3001/Patient/5616b6a11cd462440e001586")
	c.Assert(firstLink.Target, Equals, "http://localhost:3001/Patient/57335e8465ddb433bd30f0ef")
	c.Assert(firstLink.Match, Equals, "probable")

	lastLink := links[2]
	c.Assert(lastLink.Score, Equals, 0.82)
	c.Assert(lastLink.Source, Equals, "http://localhost:3001/Patient/5616b69a1cd462440e0006ae")
	c.Assert(lastLink.Target, Equals, "http://localhost:3001/Patient/57335da265ddb433bd30f0ee")
	c.Assert(lastLink.Match, Equals, "probable")
}
