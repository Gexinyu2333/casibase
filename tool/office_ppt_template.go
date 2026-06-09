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

package tool

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/the-open-agent/openagent/office"
)

const (
	pptxTemplateDownloadLimit = 100 << 20
)

type (
	pptxTemplateAnalyzeBuiltin struct{}
	pptxTemplateFillBuiltin    struct{}
)

type pptxTemplateAnalyzeArgs struct {
	Template string `json:"template"`
}

type pptxTemplateFillArgs struct {
	Template           string          `json:"template"`
	Path               string          `json:"path"`
	Plan               json.RawMessage `json:"plan"`
	Transition         string          `json:"transition,omitempty"`
	TransitionDuration float64         `json:"transition_duration,omitempty"`
}

func (t *pptxTemplateAnalyzeBuiltin) GetName() string { return "pptx_template_analyze" }

func (t *pptxTemplateAnalyzeBuiltin) GetDescription() string {
	return `Analyze a user-provided PowerPoint template before filling it.
- template (required): local .pptx path or an HTTP(S) URL from a chat attachment.
Returns template_fill_pptx_library.v1 JSON with slide types, text slot IDs, geometry, capacity metrics, tables, charts, and a plan contract. Use the returned IDs to build a template_fill_pptx_plan.v1 plan, then call pptx_template_fill.`
}

func (t *pptxTemplateAnalyzeBuiltin) GetInputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"template": map[string]interface{}{
				"type":        "string",
				"description": "Local .pptx path or HTTP(S) chat attachment URL.",
			},
		},
		"required": []string{"template"},
	}
}

func (t *pptxTemplateAnalyzeBuiltin) Execute(ctx context.Context, arguments map[string]interface{}) (*protocol.CallToolResult, error) {
	var args pptxTemplateAnalyzeArgs
	if result := decodePptxTemplateArguments(arguments, &args); result != nil {
		return result, nil
	}
	args.Template = strings.TrimSpace(args.Template)
	if args.Template == "" {
		return officeToolError("Missing required parameter: template"), nil
	}

	templatePath, cleanup, err := resolvePptxTemplate(ctx, args.Template)
	if err != nil {
		return officeToolError(fmt.Sprintf("Failed to open PowerPoint template: %s", err.Error())), nil
	}
	defer cleanup()

	library, err := office.AnalyzeFile(templatePath, office.DefaultLimits())
	if err != nil {
		return officeToolError(fmt.Sprintf("Failed to analyze PowerPoint template: %s", err.Error())), nil
	}

	data, err := json.MarshalIndent(library, "", "  ")
	if err != nil {
		return officeToolError(fmt.Sprintf("Failed to encode template analysis: %s", err.Error())), nil
	}
	return officeToolText(string(data)), nil
}

func (t *pptxTemplateFillBuiltin) GetName() string { return "pptx_template_fill" }

func (t *pptxTemplateFillBuiltin) GetDescription() string {
	return `Create a new PowerPoint file by deterministically filling and reusing slides from an existing template.
- Call pptx_template_analyze first and use its exact slide, slot, table, and chart IDs.
- template: local .pptx path or HTTP(S) chat attachment URL.
- path: exact output .pptx path; relative paths resolve to the user's Documents folder.
- plan: template_fill_pptx_plan.v1 object. Slides may be selected, repeated, and reordered. Each slide supports replacements, table_edits, chart_edits, and notes.
- Do not insert manual line breaks into titles unless they are intentional; single-line template titles are auto-fitted by default.
- Keep replacement text concise and respect capacity warnings. New text/image or text/text collisions are validation errors: shorten the content or choose another template slide.
- transition defaults to "keep", preserving source transitions and object animations.
The plan is checked before writing. Missing targets, invalid chart data, and new object collisions stop generation; capacity warnings are returned with the successful result.`
}

func (t *pptxTemplateFillBuiltin) GetInputSchema() interface{} {
	stringProperty := func(description string) map[string]interface{} {
		return map[string]interface{}{"type": "string", "description": description}
	}
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"template": stringProperty("Local .pptx path or HTTP(S) chat attachment URL."),
			"path":     stringProperty("Exact output path for the generated .pptx file."),
			"plan": map[string]interface{}{
				"type":        "object",
				"description": "A template_fill_pptx_plan.v1 plan built from pptx_template_analyze output.",
				"properties": map[string]interface{}{
					"schema":      map[string]interface{}{"type": "string", "const": "template_fill_pptx_plan.v1"},
					"source_pptx": stringProperty("Ignored for security; the template argument is always used."),
					"slides": map[string]interface{}{
						"type":     "array",
						"minItems": 1,
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"source_slide": map[string]interface{}{"type": "integer", "minimum": 1},
								"purpose":      stringProperty("Optional semantic purpose."),
								"replacements": map[string]interface{}{
									"type": "array",
									"items": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"slot_id": stringProperty("Text slot ID from analysis."),
											"text":    stringProperty("Replacement text."),
											"preserve_line_breaks": map[string]interface{}{
												"type":        "boolean",
												"description": "Preserve explicit line breaks in a title. Defaults to false for single-line title slots.",
												"default":     false,
											},
										},
										"required": []string{"slot_id", "text"},
									},
								},
								"table_edits": map[string]interface{}{
									"type": "array",
									"items": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"table_id": stringProperty("Table ID from analysis."),
											"cells": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"type": "object",
													"properties": map[string]interface{}{
														"row":  map[string]interface{}{"type": "integer", "minimum": 0},
														"col":  map[string]interface{}{"type": "integer", "minimum": 0},
														"text": stringProperty("Replacement cell text."),
													},
													"required": []string{"row", "col", "text"},
												},
											},
										},
										"required": []string{"table_id", "cells"},
									},
								},
								"chart_edits": map[string]interface{}{
									"type": "array",
									"items": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"chart_id":   stringProperty("Chart ID from analysis."),
											"categories": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
											"series": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"type": "object",
													"properties": map[string]interface{}{
														"name":   stringProperty("Series name."),
														"values": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "number"}},
													},
													"required": []string{"name", "values"},
												},
											},
										},
										"required": []string{"chart_id", "categories", "series"},
									},
								},
								"notes":      stringProperty("Speaker notes for the generated slide."),
								"transition": stringProperty("Optional per-slide transition; keep preserves the source."),
								"transition_duration": map[string]interface{}{
									"type":        "number",
									"description": "Optional per-slide transition duration in seconds.",
									"minimum":     0,
								},
								"advance_after": map[string]interface{}{
									"type":        "number",
									"description": "Optional automatic slide advance delay in seconds.",
									"minimum":     0,
								},
							},
							"required": []string{"source_slide"},
						},
					},
				},
				"required": []string{"schema", "slides"},
			},
			"transition": map[string]interface{}{
				"type":        "string",
				"description": "Default slide transition. Use keep to preserve the source transition.",
				"default":     "keep",
			},
			"transition_duration": map[string]interface{}{
				"type":        "number",
				"description": "Default transition duration in seconds when setting a transition.",
				"minimum":     0,
				"default":     0.5,
			},
		},
		"required": []string{"template", "path", "plan"},
	}
}

func (t *pptxTemplateFillBuiltin) Execute(ctx context.Context, arguments map[string]interface{}) (*protocol.CallToolResult, error) {
	var args pptxTemplateFillArgs
	if result := decodePptxTemplateArguments(arguments, &args); result != nil {
		return result, nil
	}
	args.Template = strings.TrimSpace(args.Template)
	args.Path = strings.TrimSpace(args.Path)
	if args.Template == "" {
		return officeToolError("Missing required parameter: template"), nil
	}
	if args.Path == "" {
		return officeToolError("Missing required parameter: path"), nil
	}
	if strings.ToLower(filepath.Ext(args.Path)) != ".pptx" {
		return officeToolError("Invalid path: output must use the .pptx extension"), nil
	}
	if len(bytes.TrimSpace(args.Plan)) == 0 || bytes.Equal(bytes.TrimSpace(args.Plan), []byte("null")) {
		return officeToolError("Missing required parameter: plan"), nil
	}
	var plan office.Plan
	if err := json.Unmarshal(args.Plan, &plan); err != nil {
		return officeToolError(fmt.Sprintf("Invalid plan: %s", err.Error())), nil
	}
	if plan.Schema != office.PlanSchema {
		return officeToolError("Invalid plan: schema must be template_fill_pptx_plan.v1"), nil
	}

	templatePath, cleanup, err := resolvePptxTemplate(ctx, args.Template)
	if err != nil {
		return officeToolError(fmt.Sprintf("Failed to open PowerPoint template: %s", err.Error())), nil
	}
	defer cleanup()

	transition := strings.TrimSpace(args.Transition)
	if transition == "" {
		transition = "keep"
	}
	duration := args.TransitionDuration
	if duration == 0 {
		duration = 0.5
	}
	outputPath := ResolveOutputPath(args.Path)
	library, err := office.AnalyzeFile(templatePath, office.DefaultLimits())
	if err != nil {
		return officeToolError(fmt.Sprintf("Failed to fill PowerPoint template: %s", err.Error())), nil
	}
	validationReport, err := office.FillFile(
		templatePath,
		outputPath,
		&plan,
		office.ApplyOptions{
			Transition:         transition,
			TransitionDuration: duration,
			Library:            library,
		},
		office.DefaultLimits(),
	)
	if err != nil {
		if validationReport != nil {
			if reportData, marshalErr := json.MarshalIndent(compactPptxCheckReport(validationReport), "", "  "); marshalErr == nil {
				return officeToolError(fmt.Sprintf(
					"Failed to fill PowerPoint template: %s\nValidation report:\n%s",
					err.Error(), reportData,
				)), nil
			}
		}
		return officeToolError(fmt.Sprintf("Failed to fill PowerPoint template: %s", err.Error())), nil
	}

	reportText := "none"
	if reportData, marshalErr := json.MarshalIndent(compactPptxCheckReport(validationReport), "", "  "); marshalErr == nil {
		reportText = string(reportData)
	}
	return officeToolText(fmt.Sprintf(
		"Successfully filled PowerPoint template: %s\n%d slide(s) written\nValidation report:\n%s",
		outputPath, len(plan.Slides), reportText,
	)), nil
}

func compactPptxCheckReport(report *office.CheckReport) *office.CheckReport {
	if report == nil {
		return nil
	}
	compact := &office.CheckReport{
		Schema:  report.Schema,
		Summary: report.Summary,
		Results: []office.CheckResult{},
	}
	for _, item := range report.Results {
		status, _ := item["status"].(string)
		scale, hasScale := checkResultNumber(item["estimated_font_scale_percent"])
		chartLabelsFit, hasChartFit := item["category_labels_fit"].(bool)
		if status != "ERROR" && !(status == "WARN" && ((hasScale && scale < 60) || (hasChartFit && !chartLabelsFit))) {
			continue
		}
		result := office.CheckResult{}
		for _, key := range []string{
			"status", "plan_slide", "source_slide", "slot_id", "table_id", "chart_id", "selector",
			"new_text", "message", "estimated_font_scale_percent", "capacity_visual_width", "collisions",
			"category_axis_font_size_pt", "category_label_area_percent", "category_labels_fit",
			"longest_category", "longest_category_visual_width", "suggested_max_visual_width",
		} {
			if value, ok := item[key]; ok {
				result[key] = value
			}
		}
		if capacity, ok := item["capacity_visual_width"]; ok {
			result["suggested_max_visual_width"] = capacity
		}
		result["suggestion"] = pptxCheckSuggestion(item)
		compact.Results = append(compact.Results, result)
	}
	return compact
}

func pptxCheckSuggestion(item office.CheckResult) string {
	status, _ := item["status"].(string)
	message, _ := item["message"].(string)
	switch {
	case strings.Contains(message, "overlap"):
		return "Shorten this replacement or choose a source slide with more space; the generated text would overlap another object."
	case strings.Contains(message, "target not found"):
		return "Re-run pptx_template_analyze and use an exact slot, table, or chart ID from the returned library."
	case strings.Contains(message, "out of bounds"):
		return "Use a row and column that exist in the analyzed table."
	case strings.Contains(message, "chart"):
		if labelsFit, ok := item["category_labels_fit"].(bool); ok && !labelsFit {
			return "Shorten the longest category label; the chart already uses the maximum label area and minimum 8pt axis font."
		}
		return "Make every chart series contain exactly one value for each category."
	case status == "WARN":
		return "Shorten this text or choose a larger slot to avoid very small rendered text."
	default:
		return "Revise this plan item using the analyzed template metadata, then retry."
	}
}

func checkResultNumber(value interface{}) (float64, bool) {
	switch number := value.(type) {
	case float64:
		return number, true
	case float32:
		return float64(number), true
	case int:
		return float64(number), true
	case int64:
		return float64(number), true
	default:
		return 0, false
	}
}

func decodePptxTemplateArguments(arguments map[string]interface{}, target interface{}) *protocol.CallToolResult {
	data, err := json.Marshal(arguments)
	if err != nil {
		return officeToolError(fmt.Sprintf("Failed to parse parameters: %s", err.Error()))
	}
	if err := json.Unmarshal(data, target); err != nil {
		return officeToolError(fmt.Sprintf("Failed to parse parameters: %s", err.Error()))
	}
	return nil
}

func resolvePptxTemplate(ctx context.Context, value string) (string, func(), error) {
	parsed, err := url.Parse(value)
	if err != nil {
		return "", func() {}, fmt.Errorf("invalid template location: %w", err)
	}
	if parsed.Scheme == "http" || parsed.Scheme == "https" {
		return downloadPptxTemplate(ctx, parsed)
	}
	if parsed.Scheme != "" && strings.Contains(value, "://") {
		return "", func() {}, fmt.Errorf("unsupported URL scheme %q; only HTTP(S) is allowed", parsed.Scheme)
	}

	path, err := filepath.Abs(value)
	if err != nil {
		return "", func() {}, err
	}
	if err := validatePptxPackage(path); err != nil {
		return "", func() {}, err
	}
	return path, func() {}, nil
}

func downloadPptxTemplate(ctx context.Context, location *url.URL) (string, func(), error) {
	client := &http.Client{
		Timeout: 90 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return errors.New("redirected to a non-HTTP(S) URL")
			}
			return nil
		},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, location.String(), nil)
	if err != nil {
		return "", func() {}, err
	}
	response, err := client.Do(request)
	if err != nil {
		return "", func() {}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", func() {}, fmt.Errorf("download returned HTTP %d", response.StatusCode)
	}
	if response.ContentLength > pptxTemplateDownloadLimit {
		return "", func() {}, fmt.Errorf("template exceeds the 100 MB download limit")
	}

	file, err := os.CreateTemp("", "openagent-pptx-template-*.pptx")
	if err != nil {
		return "", func() {}, err
	}
	path := file.Name()
	cleanup := func() { _ = os.Remove(path) }
	limited := io.LimitReader(response.Body, pptxTemplateDownloadLimit+1)
	written, copyErr := io.Copy(file, limited)
	closeErr := file.Close()
	if copyErr != nil {
		cleanup()
		return "", func() {}, copyErr
	}
	if closeErr != nil {
		cleanup()
		return "", func() {}, closeErr
	}
	if written > pptxTemplateDownloadLimit {
		cleanup()
		return "", func() {}, fmt.Errorf("template exceeds the 100 MB download limit")
	}
	if err := validatePptxPackage(path); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return path, cleanup, nil
}

func validatePptxPackage(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("template path is a directory")
	}
	if strings.ToLower(filepath.Ext(path)) != ".pptx" {
		return fmt.Errorf("template must use the .pptx extension")
	}
	reader, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("template is not a valid ZIP/PPTX package: %w", err)
	}
	defer reader.Close()
	required := map[string]bool{
		"[Content_Types].xml":  false,
		"ppt/presentation.xml": false,
	}
	for _, entry := range reader.File {
		if _, ok := required[entry.Name]; ok {
			required[entry.Name] = true
		}
	}
	for name, found := range required {
		if !found {
			return fmt.Errorf("invalid PPTX package: missing %s", name)
		}
	}
	return nil
}
