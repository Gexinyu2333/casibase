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

package object

import "strings"

const (
	storeMinMessageCountForReview = 200
	storeMinVectorCountForReview  = 100
)

// CheckStorePendingReviewEligibility checks whether a store qualifies to be submitted for hub review.
// storeName is the store's DB name (used to count its messages/vectors).
// Returns whether it is eligible, and the i18n keys of every unmet requirement.
func CheckStorePendingReviewEligibility(store *Store, storeName string) (bool, []string, error) {
	var failedChecks []string

	if store.DisplayName == "" || strings.Contains(store.DisplayName, "New Store") {
		failedChecks = append(failedChecks, "store:Please set a custom display name for this agent (the default \"New Store\" name is not allowed)")
	}

	if store.Avatar == "" || strings.Contains(store.Avatar, "openagent.png") || strings.Contains(store.Avatar, "casibase.png") {
		failedChecks = append(failedChecks, "store:Please upload a custom avatar for this agent (the default avatar is not allowed)")
	}

	messageCount, err := adapter.engine.Count(&Message{Store: storeName})
	if err != nil {
		return false, nil, err
	}
	if messageCount < storeMinMessageCountForReview {
		failedChecks = append(failedChecks, "store:This agent needs at least 200 messages before it can be submitted for review")
	}

	vectorCount, err := adapter.engine.Count(&Vector{Store: storeName})
	if err != nil {
		return false, nil, err
	}
	if vectorCount < storeMinVectorCountForReview {
		failedChecks = append(failedChecks, "store:This agent needs at least 100 vectors before it can be submitted for review")
	}

	return len(failedChecks) == 0, failedChecks, nil
}

func InitStoreCount() {
	emptyStoreMessage := &Message{}
	has, err := adapter.engine.Where("store = ?", "").Or("store IS NULL").Get(emptyStoreMessage)
	if err != nil {
		panic(err)
	}

	if !has {
		return
	}

	chats, err := GetGlobalChats()
	if err != nil {
		panic(err)
	}

	chatMap := map[string]*Chat{}
	for _, chat := range chats {
		chatMap[chat.Name] = chat
	}

	messages, err := GetGlobalMessages()
	if err != nil {
		panic(err)
	}

	for _, message := range messages {
		if message.Store != "" {
			continue
		}

		chat, ok := chatMap[message.Chat]
		if !ok || chat.Store == "" {
			continue
		}

		message.Store = chat.Store
		_, err = UpdateMessage(message.GetId(), message, false)
		if err != nil {
			panic(err)
		}
	}
}

func PopulateStoreCounts(stores []*Store) error {
	for _, store := range stores {
		chatCount, err := adapter.engine.Count(&Chat{Store: store.Name})
		if err != nil {
			return err
		}

		messageCount, err := adapter.engine.Count(&Message{Store: store.Name})
		if err != nil {
			return err
		}

		vectorCount, err := adapter.engine.Count(&Vector{Store: store.Name})
		if err != nil {
			return err
		}

		store.ChatCount = int(chatCount)
		store.MessageCount = int(messageCount)
		store.VectorCount = int(vectorCount)
	}

	return nil
}
