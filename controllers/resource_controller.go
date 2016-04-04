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

package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	logger "github.com/mitre/ptmatch/logger"
	ptm_models "github.com/mitre/ptmatch/models"

	"github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type ResourceController struct {
	DatabaseProvider func() *mgo.Database
}

func (r ResourceController) Database() *mgo.Database {
	return r.DatabaseProvider()
}

func (rc *ResourceController) GetResources(ctx *gin.Context) {
	req := ctx.Request
	resourceType := getResourceType(req.URL)

	logger.Log.WithFields(
		logrus.Fields{"resource type": resourceType}).Info("GetResources")

	resources := ptm_models.NewSliceForResourceName(resourceType, 0, 0)
	c := rc.Database().C(ptm_models.GetCollectionName(resourceType))
	// retrieve all documents in the collection
	// TODO Restrict this to resourc type, just to be extra safe
	err := c.Find(bson.M{}).All(resources)
	if err != nil {
		if err == mgo.ErrNotFound {
			ctx.String(http.StatusNotFound, "Not Found")
			ctx.Abort()
			return
		} else {
			ctx.AbortWithError(http.StatusBadRequest, err)
			return
		}
	}

	ctx.JSON(http.StatusOK, resources)
}

// getResourceType extracts the resource type associated with the
// given resource url.
func getResourceType(url *url.URL) string {
	// The resource type is main part of resource's url
	regex := regexp.MustCompile("^/([a-zA-Z0-9._-]+)/?")
	resourceType := regex.FindStringSubmatch(url.String())[1]
	logger.Log.WithFields(
		logrus.Fields{"url": url, "resource type": resourceType}).Debug("getResourceType")
	return resourceType
}

// LoadResource returns an object from the database that matches the specified
// resource type and object identifier.
func (rc *ResourceController) LoadResource(resourceType string, id bson.ObjectId) (interface{}, error) {
	// Determine the collection expected to hold the resource
	c := rc.Database().C(ptm_models.GetCollectionName(resourceType))
	result := ptm_models.NewStructForResourceName(resourceType)
	err := c.Find(bson.M{"_id": id}).One(result)
	if err != nil {
		return nil, err
	}
	logger.Log.WithFields(logrus.Fields{"result": result}).Debug("LoadResource")
	return result, nil
}

func toBsonObjectID(idString string) (bson.ObjectId, error) {
	var id bson.ObjectId

	logger.Log.WithFields(logrus.Fields{"id": idString}).Debug("toBsonObjectID")
	if bson.IsObjectIdHex(idString) {
		id = bson.ObjectIdHex(idString)
	} else {
		return bson.ObjectId(0), errors.New("Invalid resource identifier: " + idString)
	}
	return id, nil
}

func (rc *ResourceController) GetResource(ctx *gin.Context) {
	var id bson.ObjectId
	req := ctx.Request
	resourceType := getResourceType(req.URL)

	// Validate id as a bson Object ID
	id, err := toBsonObjectID(ctx.Param("id"))
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	logger.Log.WithFields(
		logrus.Fields{"resource type": resourceType, "id": id}).Info("GetResource")

	resource, err := rc.LoadResource(resourceType, id)
	if err != nil {
		if err == mgo.ErrNotFound {
			ctx.String(http.StatusNotFound, "Not Found")
			ctx.Abort()
			return
		} else {
			ctx.AbortWithError(http.StatusBadRequest, err)
			return
		}
	}

	logger.Log.WithFields(logrus.Fields{"resource": resource}).Info("GetResource")

	ctx.JSON(http.StatusOK, resource)
}

// CreateResource creates an instance of the resource associated with
// the request url, writes the body of the request into the new object,
// and then persists the object in the database.  A unique identifier is
// created and associated with the object.  A copy of the object that was
// stored in the database is returned in the response.
func (rc *ResourceController) CreateResource(ctx *gin.Context) {
	req := ctx.Request
	resourceType := getResourceType(req.URL)
	resource := ptm_models.NewStructForResourceName(resourceType)
	if err := ctx.Bind(resource); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	res, err := ptm_models.PersistResource(rc.Database(), resourceType, resource)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	id := reflect.ValueOf(res).Elem().FieldByName("ID").String()

	logger.Log.WithFields(
		logrus.Fields{"res type": resourceType, "id": id}).Info("CreateResource")

	ctx.Header("Location", responseURL(req, resourceType, id).String())

	ctx.JSON(http.StatusCreated, res)
}

func (rc *ResourceController) UpdateResource(ctx *gin.Context) {
	var id bson.ObjectId

	// Section 9.6 of RFC 2616 says to return 201 if resource didn't already exist
	// and 200 or 204, otherwise
	// http://www.w3.org/Protocols/rfc2616/rfc2616-sec9.html#sec9.6
	var statusCode int = http.StatusOK

	req := ctx.Request
	resourceType := getResourceType(req.URL)

	// Validate id as a bson Object ID
	id, err := toBsonObjectID(ctx.Param("id"))
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	var createdOn reflect.Value

	// Determine if the resource already exists
	existing, err := rc.LoadResource(resourceType, id)
	if err != nil {
		if err == mgo.ErrNotFound {
			statusCode = http.StatusCreated
		} else {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	} else {
		//		reflect.ValueOf(&n).Elem().FieldByName("N").Set(reflect.ValueOf(ft))
		metaField := reflect.ValueOf(existing).Elem().FieldByName("Meta")
		createdOn = metaField.Elem().FieldByName("CreatedOn")
		logger.Log.WithFields(
			logrus.Fields{"createdOn": createdOn}).Info("UpdateResource")
	}

	resource := ptm_models.NewStructForResourceName(resourceType)
	if err := ctx.Bind(resource); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c := rc.Database().C(ptm_models.GetCollectionName(resourceType))
	// Force the ID provided in the URL to be in the resource object
	reflect.ValueOf(resource).Elem().FieldByName("ID").Set(reflect.ValueOf(id))
	ptm_models.UpdateLastUpdatedDate(resource)
	// Ensure the creation date does not change`
	metaField := reflect.ValueOf(resource).Elem().FieldByName("Meta")
	metaField.Elem().FieldByName("CreatedOn").Set(createdOn)
	createdOn2 := metaField.Elem().FieldByName("CreatedOn")
	logger.Log.WithFields(
		logrus.Fields{"createdOn2": createdOn2}).Info("UpdateResource")
	err = c.Update(bson.M{"_id": id}, resource)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	logger.Log.WithFields(
		logrus.Fields{"collection": ptm_models.GetCollectionName(resourceType),
			"res type": resourceType, "id": id, "createdOn": createdOn}).Info("UpdateResource")

	ctx.Header("Location", responseURL(req, resourceType, id.Hex()).String())

	ctx.JSON(statusCode, resource)
}

// DeleteResource handles requests to delete a specific resource.
func (rc *ResourceController) DeleteResource(ctx *gin.Context) {
	var id bson.ObjectId
	req := ctx.Request
	resourceType := getResourceType(req.URL)

	// Validate id as a bson Object ID
	id, err := toBsonObjectID(ctx.Param("id"))
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	logger.Log.WithFields(
		logrus.Fields{"resource type": resourceType, "id": id, "coll": ptm_models.GetCollectionName(resourceType)}).Info("DeleteResource")

	// Determine the collection expected to hold the resource
	c := rc.Database().C(ptm_models.GetCollectionName(resourceType))
	err = c.Remove(bson.M{"_id": id})
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

func responseURL(r *http.Request, paths ...string) *url.URL {
	responseURL := url.URL{}
	if r.TLS == nil {
		responseURL.Scheme = "http"
	} else {
		responseURL.Scheme = "https"
	}
	responseURL.Host = r.Host
	responseURL.Path = fmt.Sprintf("/%s", strings.Join(paths, "/"))

	return &responseURL
}
