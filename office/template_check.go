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

package office

import (
	"fmt"
	"math"
	"strings"
)

func CheckPlan(library *Library, plan *Plan) *CheckReport {
	report := &CheckReport{Schema: CheckSchema, Results: []CheckResult{}}
	if library == nil || plan == nil {
		addCheck(report, "ERROR", CheckResult{"message": "library and plan are required"})
		return report
	}
	slides := make(map[int]*SlideLibraryItem, len(library.Slides))
	for index := range library.Slides {
		slides[library.Slides[index].SlideIndex] = &library.Slides[index]
	}
	for planIndex, item := range plan.Slides {
		source := slides[item.SourceSlide]
		if source == nil {
			addCheck(report, "ERROR", CheckResult{
				"plan_slide": planIndex + 1, "source_slide": item.SourceSlide,
				"message": "source slide not found in slide library",
			})
			continue
		}
		checkReplacements(report, planIndex+1, source, item.Replacements)
		checkTables(report, planIndex+1, source, item.TableEdits)
		checkCharts(report, planIndex+1, source, item.ChartEdits)
		checkImages(report, planIndex+1, source, item.ImageEdits)
	}
	return report
}

func checkReplacements(report *CheckReport, planIndex int, slide *SlideLibraryItem, replacements []Replacement) {
	for _, replacement := range replacements {
		slot, selector := findSlot(slide, replacement)
		if slot == nil {
			addCheck(report, "ERROR", CheckResult{
				"plan_slide": planIndex, "source_slide": slide.SlideIndex, "selector": selector,
				"message": "replacement target not found in slide library",
			})
			continue
		}
		text := replacementText(replacement)
		singleLine := slot.SingleLine && !replacement.PreserveLineBreaks
		if singleLine {
			text = collapseTitleLines(text)
		}
		oldWidth, newWidth := visualWidth(slot.Text), visualWidth(text)
		oldParagraphs := max(slot.ParagraphCount, 1)
		newParagraphs := 0
		for _, line := range strings.Split(text, "\n") {
			if strings.TrimSpace(line) != "" {
				newParagraphs++
			}
		}
		newParagraphs = max(newParagraphs, 1)
		layout := estimateTextLayout(text, slot.Role, oldParagraphs, slot.Geometry, slot.TextMetrics, singleLine)
		status, message := fitStatus(slot, oldWidth, newWidth, newParagraphs, layout)
		oldLayout := estimateTextLayoutWithSafety(slot.Text, slot.Role, oldParagraphs, slot.Geometry, slot.TextMetrics, slot.SingleLine, 1)
		collisions := collisionErrors(slide, slot, oldLayout, layout)
		if len(collisions) != 0 {
			status = "ERROR"
			message = "replacement creates new overlap with another slide object"
		}
		result := CheckResult{
			"plan_slide": planIndex, "source_slide": slide.SlideIndex, "slot_id": slot.SlotID,
			"role": slot.Role, "old_len": displayWidth(oldWidth), "new_len": displayWidth(newWidth),
			"old_visual_width": displayWidth(oldWidth), "new_visual_width": displayWidth(newWidth),
			"capacity_visual_width": capacityWidth(slot, oldWidth), "estimated_font_scale_percent": percent(layout.Scale),
			"final_font_size_px": roundedPointer(layout.FontSizePX, 2), "estimated_line_count": layout.LineCount,
			"single_line": singleLine, "collisions": collisions,
			"ratio":          math.Round(newWidth/math.Max(oldWidth, 1)*100) / 100,
			"old_paragraphs": oldParagraphs, "new_paragraphs": newParagraphs,
			"message": message, "old_text": slot.Text, "new_text": text,
		}
		addCheck(report, status, result)
	}
}

func checkTables(report *CheckReport, planIndex int, slide *SlideLibraryItem, edits []TableEdit) {
	for _, edit := range edits {
		table, selector := findTable(slide, edit)
		if table == nil {
			addCheck(report, "ERROR", CheckResult{
				"plan_slide": planIndex, "source_slide": slide.SlideIndex, "selector": selector,
				"message": "table target not found in slide library",
			})
			continue
		}
		for _, editCell := range edit.Cells {
			cell := findTableCell(table, editCell.Row, editCell.Col)
			if cell == nil {
				addCheck(report, "ERROR", CheckResult{
					"plan_slide": planIndex, "source_slide": slide.SlideIndex, "selector": selector,
					"message": fmt.Sprintf("table cell out of bounds: row=%d col=%d", editCell.Row, editCell.Col),
				})
				continue
			}
			layout := estimateTextLayout(tableCellText(editCell), "body_candidate", max(cell.ParagraphCount, 1), cell.Geometry, cell.TextMetrics, false)
			status, message := "OK", "table cell target and capacity are valid"
			if layout.Scale != nil && *layout.Scale < .55 {
				status, message = "WARN", "table cell requires aggressive auto-fit shrinking"
			}
			addCheck(report, status, CheckResult{
				"plan_slide": planIndex, "source_slide": slide.SlideIndex, "table_id": table.TableID,
				"row": editCell.Row, "col": editCell.Col, "estimated_font_scale_percent": percent(layout.Scale),
				"final_font_size_px": roundedPointer(layout.FontSizePX, 2), "estimated_line_count": layout.LineCount,
				"message": message,
			})
		}
	}
}

func checkCharts(report *CheckReport, planIndex int, slide *SlideLibraryItem, edits []ChartEdit) {
	for _, edit := range edits {
		chart, selector := findChart(slide, edit)
		if chart == nil {
			addCheck(report, "ERROR", CheckResult{
				"plan_slide": planIndex, "source_slide": slide.SlideIndex, "selector": selector,
				"message": "chart target not found in slide library",
			})
			continue
		}
		if len(edit.Series) == 0 {
			addCheck(report, "ERROR", CheckResult{
				"plan_slide": planIndex, "source_slide": slide.SlideIndex, "selector": selector,
				"message": "chart edit requires categories list and non-empty series list",
			})
			continue
		}
		valid := true
		for _, series := range edit.Series {
			if len(series.Values) != len(edit.Categories) {
				valid = false
				break
			}
		}
		if !valid {
			addCheck(report, "ERROR", CheckResult{
				"plan_slide": planIndex, "source_slide": slide.SlideIndex, "selector": selector,
				"message": "each chart series needs values matching categories length",
			})
			continue
		}
		status := "OK"
		message := "chart edit target and data shape are valid"
		result := CheckResult{
			"plan_slide": planIndex, "source_slide": slide.SlideIndex, "chart_id": chart.ChartID,
			"category_count": len(edit.Categories), "series_count": len(edit.Series),
		}
		if chart.geometry.Width != nil {
			layout := calculateHorizontalCategoryLayout(chart.categoryAxis, edit.Categories, float64(*chart.geometry.Width))
			if layout.Applicable {
				result["category_axis_font_size_pt"] = math.Round(layout.FontSizePT*100) / 100
				result["category_label_area_percent"] = math.Round(layout.LabelArea*1000) / 10
				result["longest_category"] = layout.LongestCategory
				result["longest_category_visual_width"] = displayWidth(layout.LongestVisualWidth)
				result["suggested_max_visual_width"] = displayWidth(layout.SuggestedMaxVisualWidth)
				result["category_labels_fit"] = layout.Fits
				if !layout.Fits {
					status = "WARN"
					message = "horizontal chart category labels may be truncated even after using the maximum label area and 8pt font"
				}
			}
		}
		result["message"] = message
		addCheck(report, status, result)
	}
}

func addCheck(report *CheckReport, status string, result CheckResult) {
	result["status"] = status
	report.Results = append(report.Results, result)
	switch status {
	case "OK":
		report.Summary.OK++
	case "WARN":
		report.Summary.Warn++
	default:
		report.Summary.Error++
	}
}

func findSlot(slide *SlideLibraryItem, edit Replacement) (*TextSlot, string) {
	selectors := []struct{ key, value string }{{"slot_id", edit.SlotID}, {"shape_id", edit.ShapeID}, {"shape_name", edit.ShapeName}}
	for _, selector := range selectors {
		if selector.value == "" {
			continue
		}
		for index := range slide.Slots {
			slot := &slide.Slots[index]
			if (selector.key == "slot_id" && slot.SlotID == selector.value) ||
				(selector.key == "shape_id" && slot.ShapeID == selector.value) ||
				(selector.key == "shape_name" && slot.ShapeName == selector.value) {
				return slot, selector.key + ":" + selector.value
			}
		}
	}
	for _, selector := range selectors {
		if selector.value != "" {
			return nil, selector.key + ":" + selector.value
		}
	}
	return nil, ""
}

func findTable(slide *SlideLibraryItem, edit TableEdit) (*TableInfo, string) {
	for _, selector := range []struct{ key, value string }{{"table_id", edit.TableID}, {"shape_id", edit.ShapeID}, {"shape_name", edit.ShapeName}} {
		if selector.value == "" {
			continue
		}
		for index := range slide.Tables {
			item := &slide.Tables[index]
			if (selector.key == "table_id" && item.TableID == selector.value) ||
				(selector.key == "shape_id" && item.ShapeID == selector.value) ||
				(selector.key == "shape_name" && item.ShapeName == selector.value) {
				return item, selector.key + ":" + selector.value
			}
		}
		return nil, selector.key + ":" + selector.value
	}
	return nil, ""
}

func findChart(slide *SlideLibraryItem, edit ChartEdit) (*ChartInfo, string) {
	for _, selector := range []struct{ key, value string }{{"chart_id", edit.ChartID}, {"shape_id", edit.ShapeID}, {"shape_name", edit.ShapeName}} {
		if selector.value == "" {
			continue
		}
		for index := range slide.Charts {
			item := &slide.Charts[index]
			if (selector.key == "chart_id" && item.ChartID == selector.value) ||
				(selector.key == "shape_id" && item.ShapeID == selector.value) ||
				(selector.key == "shape_name" && item.ShapeName == selector.value) {
				return item, selector.key + ":" + selector.value
			}
		}
		return nil, selector.key + ":" + selector.value
	}
	return nil, ""
}

func findTableCell(table *TableInfo, row, col int) *TableCell {
	for rowIndex := range table.Rows {
		if table.Rows[rowIndex].Row != row {
			continue
		}
		for cellIndex := range table.Rows[rowIndex].Cells {
			if table.Rows[rowIndex].Cells[cellIndex].Col == col {
				return &table.Rows[rowIndex].Cells[cellIndex]
			}
		}
	}
	return nil
}

func fitStatus(slot *TextSlot, oldWidth, newWidth float64, newParagraphs int, layout textLayout) (string, string) {
	if layout.Scale != nil && *layout.Scale < .55 {
		return "WARN", "text requires aggressive auto-fit shrinking and may be difficult to read"
	}
	if layout.Scale != nil && *layout.Scale < 1 {
		return "WARN", "text will be auto-fit inside the original text box"
	}
	ratio := newWidth / math.Max(oldWidth, 1)
	if slot.Role == "label_candidate" || (oldWidth <= 8 && slot.ParagraphCount <= 1) {
		if newWidth > math.Max(oldWidth, oldWidth*1.25) {
			return "WARN", "short label exceeds original visual width; rewrite shorter"
		}
		return "OK", "short label fits original visual width"
	}
	if slot.Role == "title_candidate" && slot.ParagraphCount <= 1 {
		limit := 1.35
		if oldWidth <= 12 {
			limit = 1.15
		}
		if ratio > limit {
			return "WARN", "title is too long for the original slot; rewrite first"
		}
		return "OK", "title stays near original capacity"
	}
	if newParagraphs > max(slot.ParagraphCount+2, slot.ParagraphCount*2, 2) {
		return "WARN", "body paragraph count changed too much; auto-fit may reduce readability"
	}
	if ratio > 3 {
		return "WARN", "text is much longer than source slot; auto-fit may reduce readability"
	}
	return "OK", "within estimated slot capacity"
}

func collisionErrors(slide *SlideLibraryItem, slot *TextSlot, oldLayout, newLayout textLayout) []interface{} {
	oldRect, oldOK := occupiedRect(slot.Geometry, slot.TextMetrics, oldLayout)
	newRect, newOK := occupiedRect(slot.Geometry, slot.TextMetrics, newLayout)
	if !oldOK || !newOK {
		return []interface{}{}
	}
	result := []interface{}{}
	for _, item := range slide.Objects {
		if item.ShapeID == slot.ShapeID || item.Kind == "cxnSp" {
			continue
		}
		if !item.HasText && (slot.ZOrder == nil || item.ZOrder <= *slot.ZOrder) {
			continue
		}
		oldOverlap := intersectionArea(oldRect, item.Geometry)
		newOverlap := intersectionArea(newRect, item.Geometry)
		growth := newOverlap - oldOverlap
		newArea := math.Max(newRect["width"]*newRect["height"], 1)
		if growth <= math.Max(16, oldOverlap*.25) || newOverlap/newArea < .03 {
			continue
		}
		result = append(result, map[string]interface{}{
			"shape_id": item.ShapeID, "shape_name": item.ShapeName, "kind": item.Kind, "z_order": item.ZOrder,
			"old_overlap_px2": math.Round(oldOverlap*10) / 10, "new_overlap_px2": math.Round(newOverlap*10) / 10,
		})
	}
	return result
}

func capacityWidth(slot *TextSlot, oldWidth float64) interface{} {
	if slot.Geometry.Width == nil || slot.Geometry.Height == nil {
		return nil
	}
	font := fallbackFontSize(slot.Role, slot.Geometry, max(slot.ParagraphCount, 1))
	if slot.TextMetrics.FontSizePX != nil && *slot.TextMetrics.FontSizePX > 0 {
		font = *slot.TextMetrics.FontSizePX
	}
	width := math.Max(float64(*slot.Geometry.Width-metricMargin(slot.TextMetrics, "left", 12)-metricMargin(slot.TextMetrics, "right", 12)), float64(*slot.Geometry.Width)*.72) * textLayoutSafetyFactor
	height := math.Max(float64(*slot.Geometry.Height-metricMargin(slot.TextMetrics, "top", 4)-metricMargin(slot.TextMetrics, "bottom", 4)), float64(*slot.Geometry.Height)*.72) * textLayoutSafetyFactor
	lineHeight := font * 1.25
	if slot.TextMetrics.LineSpacing != nil && *slot.TextMetrics.LineSpacing > 0 {
		lineHeight = font * *slot.TextMetrics.LineSpacing
	}
	if slot.TextMetrics.LineSpacePX != nil && *slot.TextMetrics.LineSpacePX > 0 {
		lineHeight = *slot.TextMetrics.LineSpacePX
	}
	maxLines := max(int(height/math.Max(lineHeight, 1)), 1)
	units := width / math.Max(font*.52, 1)
	switch slot.Role {
	case "label_candidate":
		units *= .7
	case "title_candidate":
		units *= .85
	}
	value := math.Max(units*float64(maxLines), oldWidth)
	return displayWidth(value)
}

func replacementText(value Replacement) string {
	if value.Paragraphs != nil {
		return strings.Join(value.Paragraphs, "\n")
	}
	return value.Text
}

func tableCellText(value TableCellEdit) string {
	if value.Paragraphs != nil {
		return strings.Join(value.Paragraphs, "\n")
	}
	return value.Text
}

func collapseTitleLines(value string) string {
	var result string
	for _, raw := range strings.Split(value, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if result != "" && isASCIIWord(result[len(result)-1]) && isASCIIWord(line[0]) {
			result += " "
		}
		result += line
	}
	return result
}

func isASCIIWord(value byte) bool {
	return value < 128 && (value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9')
}

func percent(value *float64) interface{} {
	if value == nil {
		return nil
	}
	return math.Round(*value*1000) / 10
}

func roundedPointer(value *float64, places int) interface{} {
	if value == nil {
		return nil
	}
	scale := math.Pow10(places)
	return math.Round(*value*scale) / scale
}

func displayWidth(value float64) interface{} {
	if value == math.Trunc(value) {
		return int(value)
	}
	return math.Round(value*10) / 10
}

func checkImages(report *CheckReport, planIndex int, slide *SlideLibraryItem, edits []ImageEdit) {
	seen := make(map[string]bool)
	for _, edit := range edits {
		if edit.ImageID == "" {
			addCheck(report, "ERROR", CheckResult{
				"plan_slide": planIndex, "source_slide": slide.SlideIndex,
				"message": "image edit requires a non-empty image_id",
			})
			continue
		}
		if edit.ImagePath == "" {
			addCheck(report, "ERROR", CheckResult{
				"plan_slide": planIndex, "source_slide": slide.SlideIndex,
				"image_id": edit.ImageID,
				"message":  "image edit requires a non-empty image_path",
			})
			continue
		}
		if seen[edit.ImageID] {
			addCheck(report, "ERROR", CheckResult{
				"plan_slide": planIndex, "source_slide": slide.SlideIndex,
				"image_id": edit.ImageID,
				"message":  "duplicate image_id in the same plan slide",
			})
			continue
		}
		seen[edit.ImageID] = true

		found := false
		for index := range slide.Images {
			if slide.Images[index].ImageID == edit.ImageID {
				found = true
				break
			}
		}
		if !found {
			addCheck(report, "ERROR", CheckResult{
				"plan_slide": planIndex, "source_slide": slide.SlideIndex,
				"image_id": edit.ImageID,
				"message":  "image target not found in slide library",
			})
			continue
		}
		addCheck(report, "OK", CheckResult{
			"plan_slide": planIndex, "source_slide": slide.SlideIndex,
			"image_id": edit.ImageID,
			"message":  "image edit target is valid",
		})
	}
}
