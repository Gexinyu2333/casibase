// Copyright 2026 The OpenAgent Authors. All Rights Reserved.
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
	"github.com/the-open-agent/openagent/object"
)

// GetStoreWordCloud
// @Title GetStoreWordCloud
// @Tag Analysis API
// @Description get word cloud data for a store based on its chat messages
// @Success 200 {object} map[string]int The Response object
// @router /get-store-word-cloud [get]
func (c *ApiController) GetStoreWordCloud() {
	storeName := c.Input().Get("storeName")
	if storeName == "" {
		c.ResponseError("storeName is required")
		return
	}

	_, ok := c.RequireSignedIn()
	if !ok {
		return
	}

	wordCount, err := object.GetStoreWordCloud(storeName)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	c.ResponseOk(wordCount)
}
