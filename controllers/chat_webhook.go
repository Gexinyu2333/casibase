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
	"fmt"
	"io"
	"net/http"

	"github.com/the-open-agent/openagent/chat"
	"github.com/the-open-agent/openagent/object"
)

// ChatWebhook receives incoming updates from a chat pipe.
// The URL format is: /api/chat-webhook/:pipeType/:pipeName
// This endpoint does not require authentication because it is called by chat platform servers.
// @router /api/chat-webhook/:pipeType/:pipeName [post]
func (c *ApiController) ChatWebhook() {
	pipeType := c.Ctx.Input.Param(":pipeType")
	pipeName := c.Ctx.Input.Param(":pipeName")

	pipe, err := object.GetPipeByName("admin", pipeName)
	if err != nil {
		c.Ctx.ResponseWriter.WriteHeader(http.StatusInternalServerError)
		return
	}
	if pipe == nil || chat.NormalizeChatProviderType(pipe.Type) != pipeType {
		c.Ctx.ResponseWriter.WriteHeader(http.StatusNotFound)
		return
	}

	chatProviderObj, err := pipe.GetChatProvider(c.GetAcceptLanguage())
	if err != nil {
		c.Ctx.ResponseWriter.WriteHeader(http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(c.Ctx.Request.Body)
	if err != nil {
		c.Ctx.ResponseWriter.WriteHeader(http.StatusBadRequest)
		return
	}

	immediateResponse, err := getImmediateWebhookResponse(chatProviderObj, body, c.Ctx.Request.Header)
	if err != nil {
		c.Ctx.ResponseWriter.WriteHeader(http.StatusBadRequest)
		return
	}

	incoming, err := chatProviderObj.ParseWebhookRequest(body)
	if err != nil {
		// Acknowledge malformed updates so chat platforms do not keep retrying them.
		c.Ctx.ResponseWriter.WriteHeader(http.StatusOK)
		return
	}
	if incoming == nil {
		writeChatWebhookResponse(c, immediateResponse)
		return
	}

	if immediateResponse != nil {
		writeChatWebhookResponse(c, immediateResponse)
		go sendChatPipeAnswer(chatProviderObj, incoming, c.GetAcceptLanguage())
		return
	}

	sendChatPipeAnswer(chatProviderObj, incoming, c.GetAcceptLanguage())
	c.Ctx.ResponseWriter.WriteHeader(http.StatusOK)
}

func getImmediateWebhookResponse(chatProviderObj chat.ChatProvider, body []byte, header http.Header) (*chat.WebhookResponse, error) {
	responder, ok := chatProviderObj.(chat.ImmediateWebhookResponder)
	if !ok {
		return nil, nil
	}
	return responder.GetWebhookResponse(body, header)
}

func writeChatWebhookResponse(c *ApiController, response *chat.WebhookResponse) {
	if response == nil {
		c.Ctx.ResponseWriter.WriteHeader(http.StatusOK)
		return
	}

	if response.ContentType != "" {
		c.Ctx.Output.Header("Content-Type", response.ContentType)
	}
	statusCode := response.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	c.Ctx.ResponseWriter.WriteHeader(statusCode)
	if len(response.Body) > 0 {
		_, _ = c.Ctx.ResponseWriter.Write(response.Body)
	}
}

func sendChatPipeAnswer(chatProviderObj chat.ChatProvider, incoming *chat.IncomingMessage, lang string) {
	answer, _, err := object.GetAnswer("", incoming.Text, lang)
	if err != nil {
		_ = chatProviderObj.SendMessage(incoming.ChatId, fmt.Sprintf("Error: %v", err))
		return
	}

	_ = chatProviderObj.SendMessage(incoming.ChatId, answer)
}
