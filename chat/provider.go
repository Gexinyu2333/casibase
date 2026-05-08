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

package chat

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/the-open-agent/openagent/i18n"
	"github.com/the-open-agent/openagent/proxy"
)

type IncomingMessage struct {
	ChatId   string
	UserId   string
	Text     string
	Username string
}

type ChatProvider interface {
	SendMessage(chatId string, text string) error
	ParseWebhookRequest(body []byte) (*IncomingMessage, error)
	SetWebhook(webhookUrl string) error
}

type WebhookResponse struct {
	StatusCode  int
	ContentType string
	Body        []byte
}

type ImmediateWebhookResponder interface {
	GetWebhookResponse(body []byte, header http.Header) (*WebhookResponse, error)
}

func NormalizeChatProviderType(typ string) string {
	return strings.ToLower(strings.ReplaceAll(typ, " ", "-"))
}

func GetChatProvider(typ string, clientSecret string, providerKey string, lang string) (ChatProvider, error) {
	var p ChatProvider
	var err error

	if typ == "Telegram" {
		p, err = NewTelegramChatProvider(clientSecret, proxy.ProxyHttpClient)
	} else if typ == "Discord" {
		p, err = NewDiscordChatProvider(clientSecret, providerKey, proxy.ProxyHttpClient)
	} else {
		return nil, fmt.Errorf(i18n.Translate(lang, "object:the chat provider type: %s is not supported"), typ)
	}

	if err != nil {
		return nil, err
	}

	return p, nil
}
