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
	"github.com/the-open-agent/openagent/util"
)

// ChatWebhook receives incoming updates from a Chat provider.
// The URL format is: /api/chat-webhook/:providerType/:providerName
// This endpoint does not require authentication because it is called by chat provider servers.
// @router /api/chat-webhook/:providerType/:providerName [post]
func (c *ApiController) ChatWebhook() {
	providerType := c.Ctx.Input.Param(":providerType")
	providerName := c.Ctx.Input.Param(":providerName")

	provider, err := object.GetProvider(util.GetIdFromOwnerAndName("admin", providerName))
	if err != nil {
		c.Ctx.ResponseWriter.WriteHeader(http.StatusInternalServerError)
		return
	}
	if provider == nil || provider.Category != "Chat" || chat.NormalizeChatProviderType(provider.Type) != providerType {
		c.Ctx.ResponseWriter.WriteHeader(http.StatusNotFound)
		return
	}

	chatProviderObj, err := provider.GetChatProvider(c.GetAcceptLanguage())
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
		// Acknowledge malformed updates so chat providers do not keep retrying them.
		c.Ctx.ResponseWriter.WriteHeader(http.StatusOK)
		return
	}
	if incoming == nil {
		writeChatWebhookResponse(c, immediateResponse)
		return
	}

	if immediateResponse != nil {
		writeChatWebhookResponse(c, immediateResponse)
		go sendChatProviderAnswer(chatProviderObj, provider.ClientId, incoming, c.GetAcceptLanguage())
		return
	}

	sendChatProviderAnswer(chatProviderObj, provider.ClientId, incoming, c.GetAcceptLanguage())
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

func sendChatProviderAnswer(chatProviderObj chat.ChatProvider, modelProvider string, incoming *chat.IncomingMessage, lang string) {
	// Use clientId as the model provider name; empty string falls back to default store.
	answer, _, err := object.GetAnswer(modelProvider, incoming.Text, lang)
	if err != nil {
		_ = chatProviderObj.SendMessage(incoming.ChatId, fmt.Sprintf("Error: %v", err))
		return
	}

	_ = chatProviderObj.SendMessage(incoming.ChatId, answer)
}

// SetChatWebhook calls the provider API to register the webhook URL for the given provider.
// @Title SetChatWebhook
// @Tag Provider API
// @Description set webhook for a Chat provider
// @Param   id     query    string  true  "The id of the provider (owner/name)"
// @Success 200 {object} controllers.Response The Response object
// @router /api/set-chat-webhook [post]
func (c *ApiController) SetChatWebhook() {
	id := c.Input().Get("id")

	provider, err := object.GetProvider(id)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	if provider == nil {
		c.ResponseError("provider not found")
		return
	}
	if provider.Category != "Chat" {
		c.ResponseError("provider is not a Chat provider")
		return
	}

	chatProviderObj, err := provider.GetChatProvider(c.GetAcceptLanguage())
	if err != nil {
		c.ResponseError(err.Error())
		return
	}

	if provider.Domain == "" {
		c.ResponseError("Domain is not set on this provider")
		return
	}

	webhookUrl := fmt.Sprintf("%s/api/chat-webhook/%s/%s", provider.Domain, chat.NormalizeChatProviderType(provider.Type), provider.Name)
	if err = chatProviderObj.SetWebhook(webhookUrl); err != nil {
		c.ResponseError(err.Error())
		return
	}

	c.ResponseOk(webhookUrl)
}
