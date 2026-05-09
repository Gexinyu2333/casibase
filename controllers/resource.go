// Copyright 2025 The OpenAgent Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controllers

import (
	"encoding/json"
	"fmt"
	"mime"
	"path/filepath"
	"strings"

	"github.com/beego/beego/utils/pagination"
	"github.com/the-open-agent/openagent/object"
	"github.com/the-open-agent/openagent/util"
)

// GetGlobalResources
// @Title GetGlobalResources
// @Tag Resource API
// @Description get global resources with pagination
// @Success 200 {array} object.Resource The Response object
// @router /get-global-resources [get]
func (c *ApiController) GetGlobalResources() {
	owner := c.Input().Get("owner")
	limit := c.Input().Get("pageSize")
	page := c.Input().Get("p")
	field := c.Input().Get("field")
	value := c.Input().Get("value")
	sortField := c.Input().Get("sortField")
	sortOrder := c.Input().Get("sortOrder")

	if limit == "" || page == "" {
		resources, err := object.GetGlobalResources(owner)
		if err != nil {
			c.ResponseError(err.Error())
			return
		}
		c.ResponseOk(resources)
	} else {
		if !c.RequireAdmin() {
			return
		}

		limitInt := util.ParseInt(limit)
		count, err := object.GetResourceCount(owner, field, value)
		if err != nil {
			c.ResponseError(err.Error())
			return
		}

		paginator := pagination.SetPaginator(c.Ctx, limitInt, count)
		resources, err := object.GetPaginationResources(owner, paginator.Offset(), limitInt, field, value, sortField, sortOrder)
		if err != nil {
			c.ResponseError(err.Error())
			return
		}

		c.ResponseOk(resources, count)
	}
}

// GetResource
// @Title GetResource
// @Tag Resource API
// @Description get resource by id
// @Param id query string true "The id (owner/name) of the resource"
// @Success 200 {object} object.Resource The Response object
// @router /get-resource [get]
func (c *ApiController) GetResource() {
	id := c.Input().Get("id")

	resource, err := object.GetResource(id)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	c.ResponseOk(resource)
}

// UpdateResource
// @Title UpdateResource
// @Tag Resource API
// @Description update resource metadata
// @Param id query string true "The id (owner/name) of the resource"
// @Param body body object.Resource true "The resource object"
// @Success 200 {object} controllers.Response The Response object
// @router /update-resource [post]
func (c *ApiController) UpdateResource() {
	id := c.Input().Get("id")

	var resource object.Resource
	err := json.NewDecoder(c.Ctx.Request.Body).Decode(&resource)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	success, err := object.UpdateResource(id, &resource)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	c.ResponseOk(success)
}

// AddResource
// @Title AddResource
// @Tag Resource API
// @Description add resource record
// @Param body body object.Resource true "The resource object"
// @Success 200 {object} controllers.Response The Response object
// @router /add-resource [post]
func (c *ApiController) AddResource() {
	var resource object.Resource
	err := json.NewDecoder(c.Ctx.Request.Body).Decode(&resource)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	success, err := object.AddResource(&resource)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	c.ResponseOk(success)
}

// DeleteResource
// @Title DeleteResource
// @Tag Resource API
// @Description delete resource record and its file from storage
// @Param body body object.Resource true "The resource object"
// @Success 200 {object} controllers.Response The Response object
// @router /delete-resource [post]
func (c *ApiController) DeleteResource() {
	var resource object.Resource
	err := json.NewDecoder(c.Ctx.Request.Body).Decode(&resource)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	err = object.DeleteResourceFile(&resource, c.GetAcceptLanguage())
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	success, err := object.DeleteResource(&resource)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	c.ResponseOk(success)
}

// UploadResource
// @Title UploadResource
// @Tag Resource API
// @Description upload a file (multipart/form-data) and record it as a resource
// @Param file formData file true "The file to upload"
// @Param category formData string false "Resource category: avatar (default), chat, document"
// @Param objectType formData string false "Associated object type: store, task, message, chat"
// @Param objectId formData string false "Associated object id (owner/name)"
// @Success 200 {object} controllers.Response The Response object (returns fileUrl)
// @router /upload-resource [post]
func (c *ApiController) UploadResource() {
	userName, ok := c.RequireSignedIn()
	if !ok {
		return
	}

	category := c.GetString("category")
	objectType := c.GetString("objectType")
	objectId := c.GetString("objectId")
	if category == "" {
		category = "avatar"
	}

	file, header, err := c.GetFile("file")
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	defer file.Close()

	fileName := header.Filename
	fileSize := int(header.Size)

	fileBytes := make([]byte, fileSize)
	_, err = file.Read(fileBytes)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	// Detect MIME type and file type category
	ext := strings.ToLower(filepath.Ext(fileName))
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = mime.TypeByExtension(ext)
	}
	fileTypeParts := strings.SplitN(mimeType, "/", 2)
	fileType := "unknown"
	if len(fileTypeParts) > 0 {
		fileType = fileTypeParts[0]
	}

	fullFilePath := fmt.Sprintf("openagent/resources/%s/%s/%s", category, userName, fileName)

	host := c.Ctx.Request.Host
	origin := getOriginFromHost(host)
	fileUrl, err := object.UploadFileToStorageSafe(fullFilePath, fileBytes, origin, c.GetAcceptLanguage())
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	resource := object.NewResourceFromUpload("admin", userName, category, fileName, fileType, ext, fileUrl, fullFilePath, fileSize, objectType, objectId)
	_, err = object.AddResource(resource)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	c.ResponseOk(fileUrl, fullFilePath)
}
