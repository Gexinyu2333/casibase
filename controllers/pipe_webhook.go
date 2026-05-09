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

	"github.com/the-open-agent/openagent/object"
	pipepkg "github.com/the-open-agent/openagent/pipe"
)

// ChatWebhookVerify handles the HTTP GET challenge that some platforms (e.g. WhatsApp
// Cloud API) send to verify webhook ownership before they start delivering events.
// The URL format is: /api/chat-webhook/:pipeType/:pipeName
// This endpoint does not require authentication.
// @router /api/chat-webhook/:pipeType/:pipeName [get]
func (c *ApiController) ChatWebhookVerify() {
	pipeType := c.Ctx.Input.Param(":pipeType")
	pipeName := c.Ctx.Input.Param(":pipeName")

	pipeObj, err := object.GetPipeByName("admin", pipeName)
	if err != nil {
		c.Ctx.ResponseWriter.WriteHeader(http.StatusInternalServerError)
		return
	}
	if pipeObj == nil || pipepkg.NormalizeType(pipeObj.Type) != pipeType {
		c.Ctx.ResponseWriter.WriteHeader(http.StatusNotFound)
		return
	}

	provider, err := pipeObj.GetProvider(c.GetAcceptLanguage())
	if err != nil {
		c.Ctx.ResponseWriter.WriteHeader(http.StatusInternalServerError)
		return
	}

	verifier, ok := provider.(pipepkg.WebhookVerifier)
	if !ok {
		c.Ctx.ResponseWriter.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	params := map[string]string{}
	for key, values := range c.Ctx.Request.URL.Query() {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}

	response, err := verifier.VerifyWebhook(params)
	if err != nil {
		c.Ctx.ResponseWriter.WriteHeader(http.StatusBadRequest)
		return
	}

	writePipeWebhookResponse(c, response)
}

// ChatWebhook receives incoming updates from a chat pipe.
// The URL format is: /api/chat-webhook/:pipeType/:pipeName
// This endpoint does not require authentication because it is called by chat platform servers.
// @router /api/chat-webhook/:pipeType/:pipeName [post]
func (c *ApiController) ChatWebhook() {
	pipeType := c.Ctx.Input.Param(":pipeType")
	pipeName := c.Ctx.Input.Param(":pipeName")

	pipeObj, err := object.GetPipeByName("admin", pipeName)
	if err != nil {
		c.Ctx.ResponseWriter.WriteHeader(http.StatusInternalServerError)
		return
	}
	if pipeObj == nil || pipepkg.NormalizeType(pipeObj.Type) != pipeType {
		c.Ctx.ResponseWriter.WriteHeader(http.StatusNotFound)
		return
	}

	provider, err := pipeObj.GetProvider(c.GetAcceptLanguage())
	if err != nil {
		c.Ctx.ResponseWriter.WriteHeader(http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(c.Ctx.Request.Body)
	if err != nil {
		c.Ctx.ResponseWriter.WriteHeader(http.StatusBadRequest)
		return
	}

	immediateResponse, err := getImmediatePipeResponse(provider, body, c.Ctx.Request.Header)
	if err != nil {
		c.Ctx.ResponseWriter.WriteHeader(http.StatusBadRequest)
		return
	}

	incoming, err := provider.ParseWebhookRequest(body)
	if err != nil {
		// Acknowledge malformed updates so chat platforms do not keep retrying them.
		c.Ctx.ResponseWriter.WriteHeader(http.StatusOK)
		return
	}
	if incoming == nil {
		writePipeWebhookResponse(c, immediateResponse)
		return
	}

	if immediateResponse != nil {
		writePipeWebhookResponse(c, immediateResponse)
		go sendPipeAnswer(provider, incoming, c.GetAcceptLanguage())
		return
	}

	sendPipeAnswer(provider, incoming, c.GetAcceptLanguage())
	c.Ctx.ResponseWriter.WriteHeader(http.StatusOK)
}

func getImmediatePipeResponse(provider pipepkg.Pipe, body []byte, header http.Header) (*pipepkg.WebhookResponse, error) {
	responder, ok := provider.(pipepkg.ImmediateWebhookResponder)
	if !ok {
		return nil, nil
	}
	return responder.GetWebhookResponse(body, header)
}

func writePipeWebhookResponse(c *ApiController, response *pipepkg.WebhookResponse) {
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

func sendPipeAnswer(provider pipepkg.Pipe, incoming *pipepkg.IncomingMessage, lang string) {
	answer, _, err := object.GetAnswer("", incoming.Text, lang)
	if err != nil {
		_ = provider.SendMessage(incoming.ChatId, fmt.Sprintf("Error: %v", err))
		return
	}

	_ = provider.SendMessage(incoming.ChatId, answer)
}
