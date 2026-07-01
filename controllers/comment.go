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
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/beego/beego/utils/pagination"
	"github.com/the-open-agent/openagent/object"
	"github.com/the-open-agent/openagent/util"
)

const (
	defaultCommentPageSize = 10
	maxCommentPageSize     = 50
	maxCommentLength       = 1000
)

type commentTarget struct {
	Owner string
}

func (c *ApiController) responseCommentError(message string) {
	switch message {
	case "Unsupported comment target type",
		"Comment target does not exist",
		"Comment content cannot be empty",
		"Parent comment does not exist",
		"Comment does not exist":
		c.ResponseError(c.T("comment:" + message))
	default:
		c.ResponseError(message)
	}
}

func getCommentPageSize(value string) int {
	pageSize := util.ParseInt(value)
	if pageSize <= 0 {
		return defaultCommentPageSize
	}
	if pageSize > maxCommentPageSize {
		return maxCommentPageSize
	}
	return pageSize
}

func resolveCommentTarget(targetType string, targetKey string) (*commentTarget, error) {
	if targetType != object.CommentTargetTypeAgentHub {
		return nil, fmt.Errorf("Unsupported comment target type")
	}

	owner, name, err := util.GetOwnerAndNameFromIdWithError(targetKey)
	if err != nil {
		return nil, err
	}

	store, err := object.GetStore(util.GetIdFromOwnerAndName(owner, name))
	if err != nil {
		return nil, err
	}
	if store == nil || store.PublishState != "Published" {
		return nil, fmt.Errorf("Comment target does not exist")
	}

	return &commentTarget{Owner: store.Owner}, nil
}

// GetComments
// @Title GetComments
// @Tag Comment API
// @Description get comments by target
// @Param targetType query string true "The target type"
// @Param targetKey query string true "The target key"
// @Success 200 {array} object.Comment The Response object
// @router /get-comments [get]
func (c *ApiController) GetComments() {
	targetType := c.Input().Get("targetType")
	targetKey := c.Input().Get("targetKey")
	pageSize := getCommentPageSize(c.Input().Get("pageSize"))

	_, err := resolveCommentTarget(targetType, targetKey)
	if err != nil {
		c.responseCommentError(err.Error())
		return
	}

	count, err := object.GetCommentCount(targetType, targetKey)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	paginator := pagination.SetPaginator(c.Ctx, pageSize, count)
	comments, err := object.GetPaginationComments(targetType, targetKey, paginator.Offset(), pageSize)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	c.ResponseOk(comments, count)
}

// AddComment
// @Title AddComment
// @Tag Comment API
// @Description add a comment
// @Param body body object.Comment true "The details of the comment"
// @Success 200 {object} controllers.Response The Response object
// @router /add-comment [post]
func (c *ApiController) AddComment() {
	username, ok := c.RequireSignedIn()
	if !ok {
		return
	}
	if util.IsAnonymousUserByUsername(username) {
		c.ResponseError(c.T("auth:Please sign in first"))
		return
	}

	var comment object.Comment
	err := json.Unmarshal(c.Ctx.Input.RequestBody, &comment)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	comment.Owner = username
	comment.Content = strings.TrimSpace(comment.Content)
	if comment.Content == "" {
		c.responseCommentError("Comment content cannot be empty")
		return
	}
	if utf8.RuneCountInString(comment.Content) > maxCommentLength {
		c.ResponseError(fmt.Sprintf(c.T("comment:Comment content cannot be longer than %d characters"), maxCommentLength))
		return
	}

	_, err = resolveCommentTarget(comment.TargetType, comment.TargetKey)
	if err != nil {
		c.responseCommentError(err.Error())
		return
	}

	if comment.ParentOwner != "" || comment.ParentName != "" {
		parent, err := object.GetComment(comment.ParentOwner, comment.ParentName)
		if err != nil {
			c.ResponseError(err.Error())
			return
		}
		if parent == nil || parent.TargetType != comment.TargetType || parent.TargetKey != comment.TargetKey {
			c.responseCommentError("Parent comment does not exist")
			return
		}
		if parent.RootOwner == "" && parent.RootName == "" {
			comment.RootOwner = parent.Owner
			comment.RootName = parent.Name
		} else {
			comment.RootOwner = parent.RootOwner
			comment.RootName = parent.RootName
		}
	} else {
		comment.ParentOwner = ""
		comment.ParentName = ""
		comment.RootOwner = ""
		comment.RootName = ""
	}

	success, err := object.AddComment(&comment)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	c.ResponseOk(success)
}

// DeleteComment
// @Title DeleteComment
// @Tag Comment API
// @Description delete a comment
// @Param body body object.Comment true "The details of the comment"
// @Success 200 {object} controllers.Response The Response object
// @router /delete-comment [post]
func (c *ApiController) DeleteComment() {
	username, ok := c.RequireSignedIn()
	if !ok {
		return
	}

	var request object.Comment
	err := json.Unmarshal(c.Ctx.Input.RequestBody, &request)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	comment, err := object.GetComment(request.Owner, request.Name)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	if comment == nil {
		c.responseCommentError("Comment does not exist")
		return
	}

	target, err := resolveCommentTarget(comment.TargetType, comment.TargetKey)
	if err != nil {
		c.responseCommentError(err.Error())
		return
	}
	if username != comment.Owner && username != target.Owner && !c.IsAdmin() {
		c.ResponseError(c.T("auth:Unauthorized operation"))
		return
	}

	success, err := object.DeleteComment(comment)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	c.ResponseOk(success)
}
