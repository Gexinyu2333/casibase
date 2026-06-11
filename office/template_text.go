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
	"strconv"
	"strings"
)

func applyReplacements(slide *xmlNode, sourceSlide int, replacements []Replacement, metadata []TextSlot) error {
	containers := textContainers(slide)
	maps := map[string]*xmlNode{}
	for order, container := range containers {
		id, name := shapeIdentity(container, order+1)
		maps[fmt.Sprintf("slot_id:s%02d_sh%s", sourceSlide, id)] = container
		maps["shape_id:"+id] = container
		if name != "" {
			maps["shape_name:"+name] = container
		}
	}
	slotMaps := map[string]*TextSlot{}
	for index := range metadata {
		slot := &metadata[index]
		slotMaps["slot_id:"+slot.SlotID] = slot
		slotMaps["shape_id:"+slot.ShapeID] = slot
		if slot.ShapeName != "" {
			slotMaps["shape_name:"+slot.ShapeName] = slot
		}
	}
	var missing []string
	for _, replacement := range replacements {
		selectors := replacementSelectors(replacement)
		var container *xmlNode
		var slot *TextSlot
		for _, selector := range selectors {
			if container == nil {
				container = maps[selector]
			}
			if slot == nil {
				slot = slotMaps[selector]
			}
		}
		if container == nil {
			if !replacement.Optional {
				missing = append(missing, selectorLabel(selectors))
			}
			continue
		}
		singleLine := isSingleLineTitle(container)
		if slot != nil {
			singleLine = slot.SingleLine
		}
		if err := setContainerText(container, replacementText(replacement), replacement.PreserveLineBreaks || !singleLine, singleLine && !replacement.PreserveLineBreaks, slot); err != nil {
			return err
		}
	}
	if len(missing) != 0 {
		return fmt.Errorf("missing replacement target(s) on slide %d: %s", sourceSlide, strings.Join(missing, "; "))
	}
	return nil
}

func applyTableEdits(slide *xmlNode, sourceSlide int, edits []TableEdit, metadata []TableInfo) error {
	containers := tableContainers(slide)
	maps := map[string]*xmlNode{}
	for order, container := range containers {
		id, name := shapeIdentity(container, order+1)
		maps[fmt.Sprintf("table_id:s%02d_tbl%s", sourceSlide, id)] = container
		maps["shape_id:"+id] = container
		if name != "" {
			maps["shape_name:"+name] = container
		}
	}
	metaMaps := map[string]*TableInfo{}
	for index := range metadata {
		table := &metadata[index]
		metaMaps["table_id:"+table.TableID] = table
		metaMaps["shape_id:"+table.ShapeID] = table
		if table.ShapeName != "" {
			metaMaps["shape_name:"+table.ShapeName] = table
		}
	}
	var invalid []string
	for _, edit := range edits {
		selectors := tableSelectors(edit)
		var frame *xmlNode
		var tableMeta *TableInfo
		for _, selector := range selectors {
			if frame == nil {
				frame = maps[selector]
			}
			if tableMeta == nil {
				tableMeta = metaMaps[selector]
			}
		}
		if frame == nil {
			if !edit.Optional {
				invalid = append(invalid, selectorLabel(selectors))
			}
			continue
		}
		table := frame.firstDescendant(nsDrawingML, "tbl")
		rows := table.children(nsDrawingML, "tr")
		for _, cellEdit := range edit.Cells {
			if cellEdit.Row < 0 || cellEdit.Row >= len(rows) {
				invalid = append(invalid, fmt.Sprintf("%s row=%d", selectorLabel(selectors), cellEdit.Row))
				continue
			}
			cells := rows[cellEdit.Row].children(nsDrawingML, "tc")
			if cellEdit.Col < 0 || cellEdit.Col >= len(cells) {
				invalid = append(invalid, fmt.Sprintf("%s row=%d col=%d", selectorLabel(selectors), cellEdit.Row, cellEdit.Col))
				continue
			}
			var slot *TextSlot
			if tableMeta != nil {
				if cell := findTableCell(tableMeta, cellEdit.Row, cellEdit.Col); cell != nil {
					slot = &TextSlot{
						Role: "body_candidate", ParagraphCount: cell.ParagraphCount, Geometry: cell.Geometry,
						TextMetrics: cell.TextMetrics,
					}
				}
			}
			if err := setContainerText(cells[cellEdit.Col], tableCellText(cellEdit), true, false, slot); err != nil {
				return err
			}
		}
	}
	if len(invalid) != 0 {
		return fmt.Errorf("invalid table edit target(s) on slide %d: %s", sourceSlide, strings.Join(invalid, "; "))
	}
	return nil
}

func setContainerText(container *xmlNode, text string, preserveLineBreaks, singleLine bool, slot *TextSlot) error {
	if singleLine && !preserveLineBreaks {
		text = collapseTitleLines(text)
	}
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}
	body := textBody(container)
	if body == nil {
		return fmt.Errorf("matched shape does not contain a text body")
	}
	paragraphs := body.children(nsDrawingML, "p")
	if len(lines) > 1 {
		if len(paragraphs) == 0 {
			paragraphs = []*xmlNode{element(nsDrawingML, "p")}
		}
		templates := make([]*xmlNode, len(paragraphs))
		for index, paragraph := range paragraphs {
			templates[index] = paragraph.clone()
		}
		body.removeChildren(nsDrawingML, "p")
		for index, line := range lines {
			template := templates[min(index, len(templates)-1)].clone()
			setParagraphText(template, line)
			body.Children = append(body.Children, template)
		}
	} else {
		nodes := container.descendants(nsDrawingML, "t")
		if len(nodes) == 0 {
			paragraph := body.child(nsDrawingML, "p")
			if paragraph == nil {
				paragraph = element(nsDrawingML, "p")
				body.Children = append(body.Children, paragraph)
			}
			nodes = ensureParagraphTextNodes(paragraph)
		}
		nodes[0].Text = text
		for _, node := range nodes[1:] {
			node.Text = ""
		}
	}
	setNormalAutofit(body, singleLine)
	if slot != nil {
		layout := estimateTextLayout(text, slot.Role, max(slot.ParagraphCount, 1), slot.Geometry, slot.TextMetrics, singleLine)
		if layout.Scale != nil && *layout.Scale > 0 {
			base := fallbackFontSize(slot.Role, slot.Geometry, max(slot.ParagraphCount, 1))
			if slot.TextMetrics.FontSizePX != nil && *slot.TextMetrics.FontSizePX > 0 {
				base = *slot.TextMetrics.FontSizePX
			}
			setExplicitFontScale(container, *layout.Scale, base)
		}
	}
	return nil
}

func textBody(container *xmlNode) *xmlNode {
	if body := container.firstDescendant(nsPresentation, "txBody"); body != nil {
		return body
	}
	return container.firstDescendant(nsDrawingML, "txBody")
}

func setParagraphText(paragraph *xmlNode, text string) {
	nodes := paragraph.descendants(nsDrawingML, "t")
	if len(nodes) == 0 {
		nodes = ensureParagraphTextNodes(paragraph)
	}
	nodes[0].Text = text
	for _, node := range nodes[1:] {
		node.Text = ""
	}
}

func ensureParagraphTextNodes(paragraph *xmlNode) []*xmlNode {
	run := paragraph.child(nsDrawingML, "r")
	if run == nil {
		run = element(nsDrawingML, "r")
		insertBeforeParagraphEnd(paragraph, run)
	}
	text := run.child(nsDrawingML, "t")
	if text == nil {
		text = element(nsDrawingML, "t")
		run.Children = append(run.Children, text)
	}
	return []*xmlNode{text}
}

func insertBeforeParagraphEnd(paragraph, child *xmlNode) {
	for index, current := range paragraph.Children {
		if current.Name.Space == nsDrawingML && current.Name.Local == "endParaRPr" {
			paragraph.Children = append(paragraph.Children, nil)
			copy(paragraph.Children[index+1:], paragraph.Children[index:])
			paragraph.Children[index] = child
			return
		}
	}
	paragraph.Children = append(paragraph.Children, child)
}

func setNormalAutofit(body *xmlNode, singleLine bool) {
	properties := body.child(nsDrawingML, "bodyPr")
	if properties == nil {
		properties = element(nsDrawingML, "bodyPr")
		body.Children = append([]*xmlNode{properties}, body.Children...)
	}
	for _, local := range []string{"noAutofit", "spAutoFit", "normAutofit"} {
		properties.removeChildren(nsDrawingML, local)
	}
	properties.Children = append(properties.Children, element(nsDrawingML, "normAutofit"))
	if singleLine {
		properties.setAttr("", "wrap", "none")
	} else if properties.attr("", "wrap") == "none" {
		properties.setAttr("", "wrap", "square")
	}
}

func setExplicitFontScale(container *xmlNode, scale, baseFontPX float64) {
	fallback := max(int(math.Round(baseFontPX*72/96*100)), 1)
	for _, local := range []string{"rPr", "defRPr", "endParaRPr"} {
		for _, properties := range container.descendants(nsDrawingML, local) {
			properties.setAttr("", "sz", scaledFontSize(properties.attr("", "sz"), scale, fallback))
		}
	}
	for _, local := range []string{"r", "fld"} {
		for _, run := range container.descendants(nsDrawingML, local) {
			properties := run.child(nsDrawingML, "rPr")
			if properties == nil {
				properties = element(nsDrawingML, "rPr")
				run.Children = append([]*xmlNode{properties}, run.Children...)
			}
			if properties.attr("", "sz") == "" {
				properties.setAttr("", "sz", scaledFontSize("", scale, fallback))
			}
		}
	}
	for _, paragraph := range container.descendants(nsDrawingML, "p") {
		properties := paragraph.child(nsDrawingML, "pPr")
		if properties == nil {
			properties = element(nsDrawingML, "pPr")
			paragraph.Children = append([]*xmlNode{properties}, paragraph.Children...)
		}
		defaultRun := properties.child(nsDrawingML, "defRPr")
		if defaultRun == nil {
			defaultRun = element(nsDrawingML, "defRPr")
			properties.Children = append(properties.Children, defaultRun)
		}
		if defaultRun.attr("", "sz") == "" {
			defaultRun.setAttr("", "sz", scaledFontSize("", scale, fallback))
		}
		endRun := paragraph.child(nsDrawingML, "endParaRPr")
		if endRun == nil {
			endRun = element(nsDrawingML, "endParaRPr")
			paragraph.Children = append(paragraph.Children, endRun)
		}
		if endRun.attr("", "sz") == "" {
			endRun.setAttr("", "sz", scaledFontSize("", scale, fallback))
		}
	}
}

func scaledFontSize(raw string, scale float64, fallback int) string {
	size, err := strconv.Atoi(raw)
	if err != nil {
		size = fallback
	}
	return strconv.Itoa(max(int(math.Round(float64(size)*scale)), 1))
}

func isSingleLineTitle(container *xmlNode) bool {
	kind, _, placeholder := placeholderKey(container)
	if placeholder && (kind == "title" || kind == "ctrTitle") {
		return true
	}
	_, name := shapeIdentity(container, 0)
	lower := strings.ToLower(name)
	if strings.Contains(lower, "title") || strings.Contains(name, "标题") {
		return len(container.descendants(nsDrawingML, "p")) <= 1
	}
	body := textBody(container)
	properties := body
	if body != nil {
		properties = body.child(nsDrawingML, "bodyPr")
	}
	return properties != nil && properties.attr("", "wrap") == "none" && len(container.descendants(nsDrawingML, "p")) <= 1
}

func replacementSelectors(value Replacement) []string {
	var result []string
	if value.SlotID != "" {
		result = append(result, "slot_id:"+value.SlotID)
	}
	if value.ShapeID != "" {
		result = append(result, "shape_id:"+value.ShapeID)
	}
	if value.ShapeName != "" {
		result = append(result, "shape_name:"+value.ShapeName)
	}
	return result
}

func tableSelectors(value TableEdit) []string {
	var result []string
	if value.TableID != "" {
		result = append(result, "table_id:"+value.TableID)
	}
	if value.ShapeID != "" {
		result = append(result, "shape_id:"+value.ShapeID)
	}
	if value.ShapeName != "" {
		result = append(result, "shape_name:"+value.ShapeName)
	}
	return result
}

func chartSelectors(value ChartEdit) []string {
	var result []string
	if value.ChartID != "" {
		result = append(result, "chart_id:"+value.ChartID)
	}
	if value.ShapeID != "" {
		result = append(result, "shape_id:"+value.ShapeID)
	}
	if value.ShapeName != "" {
		result = append(result, "shape_name:"+value.ShapeName)
	}
	return result
}

func selectorLabel(selectors []string) string {
	if len(selectors) == 0 {
		return "<missing selector>"
	}
	return strings.Join(selectors, ", ")
}
