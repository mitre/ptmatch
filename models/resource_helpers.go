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
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/pebbe/util"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	logger "github.com/mitre/ptmatch/logger"
)

func NewStructForResourceName(name string) interface{} {
	rStruct := StructForResourceName(name)
	structPtr := reflect.New(reflect.TypeOf(rStruct))
	structPtr.Elem().Set(reflect.ValueOf(rStruct))
	return structPtr.Interface()
}

func SliceForResourceName(name string, len int, cap int) interface{} {
	rType := reflect.TypeOf(StructForResourceName(name))
	return reflect.MakeSlice(reflect.SliceOf(rType), len, cap).Interface()
}

func NewSliceForResourceName(name string, len int, cap int) interface{} {
	logger.Log.WithFields(logrus.Fields{"name": name}).Debug("NewSliceForResourceName")
	rSlice := SliceForResourceName(name, len, cap)
	rSlicePtr := reflect.New(reflect.TypeOf(rSlice))
	rSlicePtr.Elem().Set(reflect.ValueOf(rSlice))
	return rSlicePtr.Interface()
}

func StructForResourceName(name string) interface{} {
	logger.Log.WithFields(logrus.Fields{"name": name}).Debug("StructForResourceName")
	switch name {
	case "RecordMatchConfiguration":
		return RecordMatchConfiguration{}
	case "RecordMatchRequest":
		return RecordMatchRequest{}
	case "RecordMatchResponse":
		return RecordMatchResponse{}
	case "RecordMatchJob":
		return RecordMatchJob{}
	case "RecordMatchSystemInterface":
		return RecordMatchSystemInterface{}
	case "RecordSet":
		return RecordSet{}

	default:
		logger.Log.Warn("StructForResourceName() No match for name ", name)
		return nil
	}
}

func GetCollectionName(resourceName string) string {
	return PluralizeLowerResourceName(resourceName)
}

func PluralizeLowerResourceName(name string) string {
	var b []byte = []byte(name)
	// if 1st character is upper case
	if b[0] >= 65 && b[0] <= 90 {
		b[0] += 32
	} else {
		// TODO this doesn't look like a resource type name; return an error
	}

	// append an s
	var buffer bytes.Buffer
	buffer.WriteString(string(b))
	buffer.WriteString("s")
	return buffer.String()

}

/* When FHIR JSON is unmarshalled, types that are interface{} just get unmarshaled to map[string]interface{}.
   This function converts that unmarshaled map to a specific resource type. */
func MapToResource(resourceMap interface{}, asPointer bool) interface{} {
	b, _ := json.Marshal(&resourceMap)
	m := resourceMap.(map[string]interface{})
	t := m["resourceType"]

	if t == nil {
		return nil
	}

	var x interface{} = StructForResourceName(t.(string))

	if x == nil {
		return nil
	}

	json.Unmarshal(b, &x)

	if asPointer {
		return &x
	}
	return x
}

func AddCreatedOn(resource interface{}) {
	m := reflect.ValueOf(resource).Elem().FieldByName("Meta")
	if m.IsNil() {
		newMeta := &Meta{}
		m.Set(reflect.ValueOf(newMeta))
	}
	// Something (unsure if it's mongo or other) is limited to millisecond when precision
	// The issue is manifest after ensuring updates keep the same createOn value
	now := time.Now().Round(time.Millisecond)
	m.Elem().FieldByName("CreatedOn").Set(reflect.ValueOf(now))
}

func UpdateLastUpdatedDate(resource interface{}) {
	m := reflect.ValueOf(resource).Elem().FieldByName("Meta")
	if m.IsNil() {
		newMeta := &Meta{}
		m.Set(reflect.ValueOf(newMeta))
	}
	// Something (unsure if it's mongo or other is limited to millisecon precision)
	now := time.Now().Round(time.Millisecond)
	m.Elem().FieldByName("LastUpdatedOn").Set(reflect.ValueOf(now))
}

// LoadResource returns an object from the database that matches the specified
// resource type and object identifier.
func LoadResource(db *mgo.Database, resourceType string, id bson.ObjectId) (interface{}, error) {
	// Determine the collection expected to hold the resource
	c := db.C(GetCollectionName(resourceType))
	result := NewStructForResourceName(resourceType)
	err := c.Find(bson.M{"_id": id}).One(result)
	if err != nil {
		return nil, err
	}
	logger.Log.WithFields(logrus.Fields{"result": result}).Debug("LoadResource")
	return result, nil
}

func PersistResource(db *mgo.Database, resourceType string, resource interface{}) (interface{}, error) {
	c := db.C(GetCollectionName(resourceType))
	id := bson.NewObjectId()

	logger.Log.WithFields(
		logrus.Fields{"collection": GetCollectionName(resourceType),
			"res type": resourceType, "id": id}).Info("PersistResource")

	reflect.ValueOf(resource).Elem().FieldByName("ID").Set(reflect.ValueOf(id))
	AddCreatedOn(resource)
	UpdateLastUpdatedDate(resource)
	err := c.Insert(resource)
	return resource, err
}

func InsertResourceFromFile(db *mgo.Database, resourceType string, filePath string) interface{} {
	collection := db.C(GetCollectionName(resourceType))
	resource := NewStructForResourceName(resourceType)
	LoadResourceFromFile(filePath, resource)

	// Set a unique identifier to this resource
	rptr := reflect.ValueOf(resource)
	r := rptr.Elem()
	logger.Log.WithFields(logrus.Fields{"kind": r.Kind()}).Debug("InsertResourceFromFile")
	if r.Kind() == reflect.Struct {
		logger.Log.WithFields(logrus.Fields{"method": "InsertResourceFromFile"}).Debug("Recognize struct")
		f := r.FieldByName("ID")
		if f.IsValid() {
			logger.Log.WithFields(logrus.Fields{"method": "InsertResourceFromFile"}).Debug("Id field is valid")
			if f.CanSet() {
				objID := bson.NewObjectId()
				logger.Log.WithFields(
					logrus.Fields{"method": "InsertResourceFromFile", "obj id": objID}).Debug("ID field can be set")
				f.Set(reflect.ValueOf(objID))
			}
		}
	}

	//logger.Log.WithFields(logrus.Fields{"resource": resource}).Info("InsertResourceFromFile")
	err := collection.Insert(resource)
	util.CheckErr(err)
	return resource
}

func LoadResourceFromFile(fileName string, resource interface{}) {
	fstream, err := os.Open(fileName)
	util.WarnErr(err)
	defer fstream.Close()

	decoder := json.NewDecoder(fstream)
	r := &resource
	err = decoder.Decode(r)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{"resource": resource, "error" : err}).Warn("LoadResourceFromFile")
		util.WarnErr(err)
	}

}
