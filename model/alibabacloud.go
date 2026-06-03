// Copyright 2024 The OpenAgent Authors. All Rights Reserved.
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

package model

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	dashscopesdk "github.com/casibase/dashscope-go-sdk"
	"github.com/casibase/dashscope-go-sdk/wanx"
	"github.com/casibase/dashscopego"
	"github.com/casibase/dashscopego/qwen"
	"github.com/the-open-agent/openagent/i18n"
)

type AlibabacloudModelProvider struct {
	subType     string
	apiKey      string
	temperature float32
	topP        float32
}

func NewAlibabacloudModelProvider(subType string, apiKey string, temperature float32, topP float32) (*AlibabacloudModelProvider, error) {
	return &AlibabacloudModelProvider{
		subType:     subType,
		apiKey:      apiKey,
		temperature: temperature,
		topP:        topP,
	}, nil
}

func isWanxModel(subType string) bool {
	return strings.HasPrefix(subType, "wanx")
}

func isQwenMultimodalModel(subType string) bool {
	return strings.HasPrefix(subType, "qwen3.6-") ||
		strings.HasPrefix(subType, "qwen3-vl-") ||
		strings.HasPrefix(subType, "qwen-vl-") ||
		strings.HasPrefix(subType, "qvq-") ||
		strings.HasPrefix(subType, "qwen2.5-vl-")
}

func (p *AlibabacloudModelProvider) GetPricing() string {
	return `URL:
https://help.aliyun.com/zh/model-studio/billing-for-model-studio

| Model               | sub-type                        | Input Price per 1K characters    | Output Price per 1K characters |
|---------------------|---------------------------------|----------------------------------|--------------------------------|
| Qwen-Long           | qwen-long                       | 0.0005yuan/1,000 tokens          | 0.002yuan/1,000 tokens         |
| Qwen-Turbo          | qwen-turbo                      | 0.002yuan/1,000 tokens           | 0.006yuan/1,000 tokens         |
| Qwen-Plus           | qwen-plus                       | 0.004yuan/1,000 tokens           | 0.012yuan/1,000 tokens         |
| Qwen-Max            | qwen-max                        | 0.04yuan/1,000 tokens            | 0.12yuan/1,000 tokens          |
| Qwen-Max            | qwen-max-longcontext            | 0.04yuan/1,000 tokens            | 0.12yuan/1,000 tokens          |
| Qwen3.6 Plus        | qwen3.6-plus                    | tiered, from 0.002yuan/1,000 tokens | tiered, from 0.012yuan/1,000 tokens |
| Qwen3.6 Flash       | qwen3.6-flash                   | tiered, from 0.0012yuan/1,000 tokens | tiered, from 0.0072yuan/1,000 tokens |
| Qwen3-VL Plus       | qwen3-vl-plus                   | tiered, from 0.001yuan/1,000 tokens | tiered, from 0.010yuan/1,000 tokens |
| Qwen3-VL Flash      | qwen3-vl-flash                  | tiered, from 0.00015yuan/1,000 tokens | tiered, from 0.0015yuan/1,000 tokens |
| Qwen-VL Max         | qwen-vl-max                     | 0.0016yuan/1,000 tokens          | 0.004yuan/1,000 tokens         |
| Qwen-VL Plus        | qwen-vl-plus                    | 0.0008yuan/1,000 tokens          | 0.002yuan/1,000 tokens         |
| Qwen3-235B-a22B     | qwen3-235b-a22b                 | 0.004yuan/1,000 tokens            | 0.04yuan/1,000 tokens         |
| Qwen3-32B           | qwen3-32b            			| 0.002yuan/1,000 tokens            | 0.02yuan/1,000 tokens         |
| DeepSeek-R1         | deepseek-r1                     | 0.004yuan/1,000 tokens           | 0.016yuan/1,000 tokens         |
| DeepSeek-V3         | deepseek-v3                     | 0.002yuan/1,000 tokens           | 0.008yuan/1,000 tokens         |
| DeepSeek-V3.1       | deepseek-v3.1                   | 0.004yuan/1,000 tokens           | 0.012yuan/1,000 tokens         |
| DeepSeek-V3.2       | deepseek-v3.2                   | 0.002yuan/1,000 tokens           | 0.003yuan/1,000 tokens         |
| DeepSeek-R1-Distill | deepseek-r1-distill-qwen-1.5b   | 0.000yuan/1,000 tokens           | 0.000yuan/1,000 tokens         |
| DeepSeek-R1-Distill | deepseek-r1-distill-qwen-7b     | 0.0005yuan/1,000 tokens          | 0.001yuan/1,000 tokens         |
| DeepSeek-R1-Distill | deepseek-r1-distill-qwen-14b    | 0.001yuan/1,000 tokens           | 0.003yuan/1,000 tokens         |
| DeepSeek-R1-Distill | deepseek-r1-distill-qwen-32b    | 0.002yuan/1,000 tokens           | 0.006yuan/1,000 tokens         |
| DeepSeek-R1-Distill | deepseek-r1-distill-llama-8b    | 0.000yuan/1,000 tokens           | 0.000yuan/1,000 tokens         |
| DeepSeek-R1-Distill | deepseek-r1-distill-llama-70b   | 0.000yuan/1,000 tokens           | 0.000yuan/1,000 tokens         |

Image Generation Models:
| Model                 | sub-type                  | Price per image |
|-----------------------|---------------------------|-----------------|
| Wanx2.1 T2I Turbo     | wanx2.1-t2i-turbo         | 0.04yuan/image  |
| Wanx2.1 T2I Plus      | wanx2.1-t2i-plus          | 0.12yuan/image  |
| Wanx V1               | wanx-v1                   | 0.04yuan/image  |
`
}

func (p *AlibabacloudModelProvider) calculatePrice(modelResult *ModelResult, lang string) error {
	price := 0.0

	if isWanxModel(p.subType) {
		imagePriceTable := map[string]float64{
			"wanx2.1-t2i-turbo": 0.04,
			"wanx2.1-t2i-plus":  0.12,
			"wanx-v1":           0.04,
		}
		unitPrice, ok := imagePriceTable[p.subType]
		if !ok {
			unitPrice = 0.04
		}
		price = float64(modelResult.ImageCount) * unitPrice
		modelResult.TotalPrice = price
		modelResult.Currency = "CNY"
		return nil
	}

	if priceItem, ok := getAlibabacloudTieredPrice(p.subType, modelResult.PromptTokenCount); ok {
		inputPrice := getPrice(modelResult.PromptTokenCount, priceItem[0])
		outputPrice := getPrice(modelResult.ResponseTokenCount, priceItem[1])
		modelResult.TotalPrice = inputPrice + outputPrice
		modelResult.Currency = "CNY"
		return nil
	}

	// Prices are CNY per 1K tokens, converted from Alibaba Model Studio
	// per-1M-token pricing: https://help.aliyun.com/zh/model-studio/model-pricing (checked 2026-06-03).
	priceTable := map[string][2]float64{
		"qwen-long":                     {0.0005, 0.002},
		"qwen-turbo":                    {0.002, 0.006},
		"qwen-plus":                     {0.004, 0.012},
		"qwen-max":                      {0.040, 0.120},
		"qwen-max-longcontext":          {0.040, 0.120},
		"qwen-vl-max":                   {0.0016, 0.004},
		"qwen-vl-plus":                  {0.0008, 0.002},
		"qvq-max":                       {0.008, 0.032},
		"qvq-plus":                      {0.002, 0.005},
		"qwen3-235b-a22b":               {0.004, 0.04},
		"qwen3-32b":                     {0.002, 0.02},
		"deepseek-r1":                   {0.004, 0.016},
		"deepseek-v3":                   {0.002, 0.008},
		"deepseek-v3.1":                 {0.004, 0.012},
		"deepseek-v3.2":                 {0.002, 0.003},
		"deepseek-r1-distill-qwen-1.5b": {0.000, 0.000},
		"deepseek-r1-distill-qwen-7b":   {0.001, 0.003},
		"deepseek-r1-distill-qwen-14b ": {0.002, 0.006},
		"deepseek-r1-distill-qwen-32b":  {0.000, 0.000},
		"deepseek-r1-distill-llama-8b":  {0.000, 0.000},
	}

	if priceItem, ok := priceTable[p.subType]; ok {
		inputPrice := getPrice(modelResult.PromptTokenCount, priceItem[0])
		outputPrice := getPrice(modelResult.ResponseTokenCount, priceItem[1])
		price = inputPrice + outputPrice
	} else {
		return fmt.Errorf(i18n.Translate(lang, "embedding:calculatePrice() error: unknown model type: %s"), p.subType)
	}

	modelResult.TotalPrice = price
	modelResult.Currency = "CNY"
	return nil
}

// Tiered prices are CNY per 1K tokens, converted from Alibaba Model Studio
// per-1M-token pricing: https://help.aliyun.com/zh/model-studio/model-pricing (checked 2026-06-03).
func getAlibabacloudTieredPrice(subType string, promptTokenCount int) ([2]float64, bool) {
	switch {
	case strings.HasPrefix(subType, "qwen3.6-plus"):
		if promptTokenCount > 256000 {
			return [2]float64{0.008, 0.048}, true
		}
		return [2]float64{0.002, 0.012}, true
	case strings.HasPrefix(subType, "qwen3.6-flash"):
		if promptTokenCount > 256000 {
			return [2]float64{0.0048, 0.0288}, true
		}
		return [2]float64{0.0012, 0.0072}, true
	case strings.HasPrefix(subType, "qwen3-vl-plus"):
		if promptTokenCount > 128000 {
			return [2]float64{0.003, 0.030}, true
		} else if promptTokenCount > 32000 {
			return [2]float64{0.0015, 0.015}, true
		}
		return [2]float64{0.001, 0.010}, true
	case strings.HasPrefix(subType, "qwen3-vl-flash"):
		if promptTokenCount > 128000 {
			return [2]float64{0.0006, 0.006}, true
		} else if promptTokenCount > 32000 {
			return [2]float64{0.0003, 0.003}, true
		}
		return [2]float64{0.00015, 0.0015}, true
	}
	return [2]float64{}, false
}

func (p *AlibabacloudModelProvider) queryWanx(ctx context.Context, question string, writer io.Writer, lang string) (*ModelResult, error) {
	flusher, ok := writer.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf(i18n.Translate(lang, "model:writer does not implement http.Flusher"))
	}

	cli := dashscopesdk.NewTongyiClient(p.subType, p.apiKey)
	req := &wanx.ImageSynthesisRequest{
		Model: p.subType,
		Input: wanx.ImageSynthesisInput{
			Prompt: question,
		},
		Params: wanx.ImageSynthesisParams{
			N:    1,
			Size: "1024*1024",
		},
		// Do not ask the SDK to download the image bytes: GetImage uses the same
		// HTTP options as DashScope API calls, including Content-Type: application/json,
		// which breaks OSS presigned URL signature verification (SignatureDoesNotMatch).
		// The task result URL is returned and embedded; the browser loads it with a plain GET.
		Download: false,
	}

	imgBlobs, err := cli.CreateImageGeneration(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(imgBlobs) == 0 {
		return nil, fmt.Errorf("empty image generation response")
	}

	blob := imgBlobs[0]
	var imgSrc string
	if len(blob.Data) > 0 {
		b64 := base64.StdEncoding.EncodeToString(blob.Data)
		imgSrc = fmt.Sprintf("data:%s;base64,%s", blob.ImgType, b64)
	} else {
		imgSrc = blob.ImgURL
	}

	html := fmt.Sprintf("<img src=\"%s\" width=\"100%%\" height=\"auto\">", imgSrc)
	if _, err = fmt.Fprint(writer, html); err != nil {
		return nil, err
	}
	flusher.Flush()

	modelResult := &ModelResult{
		ImageCount:      1,
		TotalTokenCount: 1,
	}
	if err = p.calculatePrice(modelResult, lang); err != nil {
		return nil, err
	}
	return modelResult, nil
}

func (p *AlibabacloudModelProvider) QueryText(question string, writer io.Writer, history []*RawMessage, prompt string, knowledgeMessages []*RawMessage, toolSession *ToolSession, lang string) (*ModelResult, error) {
	ctx := context.Background()
	flusher, ok := writer.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf(i18n.Translate(lang, "model:writer does not implement http.Flusher"))
	}

	if isWanxModel(p.subType) {
		return p.queryWanx(ctx, question, writer, lang)
	}

	if isQwenMultimodalModel(p.subType) {
		const baseUrl = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		localProvider, err := NewLocalModelProvider("Custom-think", "custom-model", p.apiKey, p.temperature, p.topP, 0, 0, baseUrl, p.subType, 0, 0, "CNY")
		if err != nil {
			return nil, err
		}

		modelResult, err := localProvider.QueryText(question, writer, history, prompt, knowledgeMessages, toolSession, lang)
		if err != nil {
			return nil, err
		}

		err = p.calculatePrice(modelResult, lang)
		if err != nil {
			return nil, err
		}
		return modelResult, nil
	}

	cli := dashscopego.NewTongyiClient(p.subType, p.apiKey)

	if strings.HasPrefix(question, "$OpenAgentDryRun$") {
		modelResult, err := getDefaultModelResult(p.subType, question, "")
		if err != nil {
			return nil, fmt.Errorf(i18n.Translate(lang, "model:cannot calculate tokens"))
		}
		if getContextLength(p.subType) > modelResult.TotalTokenCount {
			return modelResult, nil
		} else {
			return nil, fmt.Errorf(i18n.Translate(lang, "model:exceed max tokens"))
		}
	}

	params := qwen.DefaultParameters().
		SetTemperature(float64(p.temperature)).
		SetTopP(float64(p.topP)).
		SetIncrementalOutput(true)

	if toolSession != nil && toolSession.McpToolSet != nil && toolSession.McpToolSet.WebSearchEnabled {
		params.SetEnableSearch(true)
		params.SetSearchOptions(&qwen.SearchOptions{
			ForcedSearch:        true,
			EnableSource:        true,
			EnableCitation:      true,
			PrependSearchResult: true,
		})
	}

	streamCallbackFn := func(ctx context.Context, typ string, chunk []byte) error {
		data := string(chunk)
		if data == "" {
			return nil
		}
		return flushDataThink(data, typ, writer, lang)
	}

	req := &qwen.Request[*qwen.TextContent]{
		Model: p.subType,
		Input: qwen.Input[*qwen.TextContent]{
			Messages: buildMessages(question, history, prompt, knowledgeMessages),
		},
		Parameters:  params,
		StreamingFn: streamCallbackFn,
	}

	resp, err := cli.CreateCompletion(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.Output.SearchInfo != nil && resp.Output.SearchInfo.SearchResults != nil && len(resp.Output.SearchInfo.SearchResults) > 0 {
		searchResultsJSON, _ := json.Marshal(resp.Output.SearchInfo.SearchResults)
		flushDataThink(string(searchResultsJSON), "search", writer, lang)
	}

	modelResult := &ModelResult{
		PromptTokenCount:   resp.Usage.InputTokens,
		ResponseTokenCount: resp.Usage.OutputTokens,
		TotalTokenCount:    resp.Usage.TotalTokens,
	}

	err = p.calculatePrice(modelResult, lang)
	if err != nil {
		return nil, err
	}

	flusher.Flush()
	return modelResult, nil
}

func buildMessages(question string, history []*RawMessage, prompt string, knowledgeMessages []*RawMessage) []qwen.Message[*qwen.TextContent] {
	systemMessages := getSystemMessages(prompt, knowledgeMessages)
	var messages []qwen.Message[*qwen.TextContent]
	for _, systemMsg := range systemMessages {
		content := &qwen.TextContent{Text: systemMsg.Text}
		messages = append(messages, qwen.Message[*qwen.TextContent]{
			Role:    "system",
			Content: content,
		})
	}

	for i := len(history) - 1; i >= 0; i-- {
		historyMessage := history[i]
		content := &qwen.TextContent{Text: historyMessage.Text}
		role := "user"
		if historyMessage.Author == "AI" {
			role = "assistant"
		}
		messages = append(messages, qwen.Message[*qwen.TextContent]{
			Role:    role,
			Content: content,
		})
	}

	questionContent := &qwen.TextContent{Text: question}
	messages = append(messages, qwen.Message[*qwen.TextContent]{
		Role:    "user",
		Content: questionContent,
	})

	return messages
}

func (p *AlibabacloudModelProvider) ListModels() ([]string, error) {
	return unsupportedListModels("Alibaba Cloud")
}
