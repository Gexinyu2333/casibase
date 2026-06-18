// Copyright 2026 The OpenAgent Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0

package office

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type smartArtNodeRef struct {
	ModelID string
	PresIDs []string
	Index   int
	Text    string
}

type smartArtPresCandidate struct {
	ID    string
	Index int
	Score int
}

func analyzeSmartArts(pkg *Package, slide *xmlNode, ref slideRef, objectByID map[string]*SlideObject) ([]SmartArtInfo, error) {
	rels, err := pkg.Relationships(ref.PartName)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]Relationship, len(rels.Items))
	for _, rel := range rels.Items {
		byID[rel.ID] = rel
	}

	frames := smartArtFrames(slide)
	result := make([]SmartArtInfo, 0, len(frames))
	for order, frame := range frames {
		shapeID, shapeName := shapeIdentity(frame, order+1)
		info := SmartArtInfo{
			SmartArtID: fmt.Sprintf("s%02d_sa%s", ref.Index, shapeID),
			ShapeID:    shapeID,
			ShapeName:  shapeName,
			Editable:   true,
		}
		if object := objectByID[shapeID]; object != nil {
			info.Geometry = object.Geometry
		} else {
			info.Geometry = containerGeometry(frame)
		}

		dataRelID := smartArtRelIDs(frame).attr(nsOfficeRels, "dm")
		rel, ok := byID[dataRelID]
		if !ok || rel.Type != RelationshipTypeDiagramData || rel.Mode != TargetInternal {
			info.Editable = false
			info.Reason = "diagram data relationship not found"
			result = append(result, info)
			continue
		}
		dataPart, err := ResolveTarget(ref.PartName, rel.Target)
		if err != nil || !pkg.HasPart(dataPart) {
			info.Editable = false
			info.Reason = "diagram data part not found"
			result = append(result, info)
			continue
		}
		dataRoot, err := pkg.xmlPart(dataPart)
		if err != nil {
			info.Editable = false
			info.Reason = "diagram data part cannot be parsed"
			result = append(result, info)
			continue
		}
		drawingRoot, _, err := smartArtDrawingRoot(pkg, ref.PartName, rels, dataRoot)
		if err != nil {
			info.Editable = false
			info.Reason = "diagram drawing part cannot be resolved: " + err.Error()
			result = append(result, info)
			continue
		}
		nodes := smartArtNodeRefs(dataRoot, drawingRoot)
		if len(nodes) == 0 {
			info.Editable = false
			info.Reason = "editable SmartArt nodes not found"
		}
		for index, node := range nodes {
			info.Nodes = append(info.Nodes, SmartArtNodeInfo{
				NodeID:         fmt.Sprintf("s%02d_sa%s_n%02d", ref.Index, shapeID, index+1),
				Text:           node.Text,
				ParagraphCount: max(len(strings.Split(node.Text, "\n")), 1),
				Editable:       true,
				modelID:        node.ModelID,
				presIDs:        node.PresIDs,
			})
		}
		result = append(result, info)
	}
	return result, nil
}

func smartArtFrames(root *xmlNode) []*xmlNode {
	var result []*xmlNode
	for _, frame := range root.descendants(nsPresentation, "graphicFrame") {
		data := frame.firstDescendant(nsDrawingML, "graphicData")
		if data != nil && data.attr("", "uri") == "http://schemas.openxmlformats.org/drawingml/2006/diagram" && smartArtRelIDs(frame) != nil {
			result = append(result, frame)
		}
	}
	return result
}

func smartArtRelIDs(frame *xmlNode) *xmlNode {
	return frame.firstDescendant(nsDiagram, "relIds")
}

func smartArtNodeRefs(dataRoot, drawingRoot *xmlNode) []smartArtNodeRef {
	textShapeIDs := smartArtDrawingTextShapeIDs(drawingRoot)
	candidatesByContent := map[string][]smartArtPresCandidate{}
	presAttrs := map[string]*xmlNode{}
	for _, pt := range dataRoot.descendants(nsDiagram, "pt") {
		if pt.attr("", "type") != "pres" {
			continue
		}
		prSet := pt.child(nsDiagram, "prSet")
		if prSet == nil {
			continue
		}
		contentID := prSet.attr("", "presAssocID")
		if contentID == "" {
			continue
		}
		presID := pt.attr("", "modelId")
		if presID == "" {
			continue
		}
		presAttrs[presID] = prSet
		candidatesByContent[contentID] = append(candidatesByContent[contentID], smartArtPresCandidate{
			ID:    presID,
			Index: smartArtPresStyleIndex(prSet),
			Score: smartArtPresScore(prSet, textShapeIDs[presID]),
		})
	}
	for _, cxn := range dataRoot.descendants(nsDiagram, "cxn") {
		if cxn.attr("", "type") != "presOf" {
			continue
		}
		contentID, presID := cxn.attr("", "srcId"), cxn.attr("", "destId")
		if contentID == "" || presID == "" || presAttrs[presID] == nil {
			continue
		}
		prSet := presAttrs[presID]
		candidatesByContent[contentID] = append(candidatesByContent[contentID], smartArtPresCandidate{
			ID:    presID,
			Index: smartArtPresStyleIndex(prSet),
			Score: smartArtPresScore(prSet, textShapeIDs[presID]),
		})
	}

	var nodes []smartArtNodeRef
	contentOrder := 0
	for _, pt := range dataRoot.descendants(nsDiagram, "pt") {
		modelID := pt.attr("", "modelId")
		if modelID == "" || pt.attr("", "type") != "" {
			continue
		}
		candidates := smartArtBestPresCandidates(candidatesByContent[modelID])
		if len(candidates) == 0 {
			contentOrder++
			continue
		}
		index := contentOrder
		if candidates[0].Index >= 0 {
			index = candidates[0].Index
		}
		nodes = append(nodes, smartArtNodeRef{
			ModelID: modelID,
			PresIDs: smartArtCandidateIDs(candidates),
			Index:   index,
			Text:    strings.Join(paragraphTexts(pt.child(nsDiagram, "t")), "\n"),
		})
		contentOrder++
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		return nodes[i].Index < nodes[j].Index
	})
	return nodes
}

func smartArtDrawingTextShapeIDs(root *xmlNode) map[string]bool {
	result := map[string]bool{}
	if root == nil {
		return result
	}
	for _, shape := range root.descendants(nsDiagram2008, "sp") {
		modelID := shape.attr("", "modelId")
		if modelID != "" && shape.child(nsDiagram2008, "txBody") != nil {
			result[modelID] = true
		}
	}
	return result
}

func smartArtPresStyleIndex(prSet *xmlNode) int {
	raw := prSet.attr("", "presStyleIdx")
	value, err := strconv.Atoi(raw)
	if err != nil {
		return -1
	}
	return value
}

func smartArtPresScore(prSet *xmlNode, hasDrawingText bool) int {
	score := 0
	if hasDrawingText {
		score += 100
	}
	presName := prSet.attr("", "presName")
	styleLabel := prSet.attr("", "presStyleLbl")
	if styleLabel == "node1" {
		score += 60
	}
	if presName == "node" {
		score += 50
	}
	if strings.HasSuffix(presName, "Tx") || strings.Contains(strings.ToLower(presName), "text") {
		score += 25
	}
	lowerName := strings.ToLower(presName)
	lowerLabel := strings.ToLower(styleLabel)
	if strings.Contains(lowerName, "dummy") || strings.Contains(lowerName, "space") ||
		strings.Contains(lowerName, "arrow") || strings.Contains(lowerName, "trans") {
		score -= 80
	}
	if strings.Contains(lowerLabel, "revtx") || strings.Contains(lowerLabel, "trans") {
		score -= 40
	}
	return score
}

func smartArtBestPresCandidates(candidates []smartArtPresCandidate) []smartArtPresCandidate {
	if len(candidates) == 0 {
		return nil
	}
	unique := make([]smartArtPresCandidate, 0, len(candidates))
	seen := map[string]bool{}
	for _, candidate := range candidates {
		if candidate.ID == "" || seen[candidate.ID] {
			continue
		}
		seen[candidate.ID] = true
		unique = append(unique, candidate)
	}
	sort.SliceStable(unique, func(i, j int) bool {
		if unique[i].Score != unique[j].Score {
			return unique[i].Score > unique[j].Score
		}
		return unique[i].Index < unique[j].Index
	})
	if len(unique) == 0 || unique[0].Score <= 0 {
		return nil
	}
	bestScore := unique[0].Score
	var result []smartArtPresCandidate
	for _, candidate := range unique {
		if candidate.Score != bestScore {
			break
		}
		result = append(result, candidate)
	}
	return result
}

func smartArtCandidateIDs(candidates []smartArtPresCandidate) []string {
	result := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		result = append(result, candidate.ID)
	}
	return result
}

func checkSmartArts(report *CheckReport, planIndex int, slide *SlideLibraryItem, edits []SmartArtEdit) {
	for _, edit := range edits {
		smartArt, selector := findSmartArt(slide, edit)
		if smartArt == nil {
			if !edit.Optional {
				addCheck(report, "ERROR", CheckResult{
					"plan_slide": planIndex, "source_slide": slide.SlideIndex, "selector": selector,
					"message": "SmartArt target not found in slide library",
				})
			}
			continue
		}
		if !smartArt.Editable {
			addCheck(report, "ERROR", CheckResult{
				"plan_slide": planIndex, "source_slide": slide.SlideIndex, "smartart_id": smartArt.SmartArtID,
				"message": "SmartArt is not editable: " + smartArt.Reason,
			})
			continue
		}
		for index, node := range edit.Nodes {
			target := smartArtNodeByEdit(smartArt, node, index)
			if target == nil {
				if !node.Optional {
					addCheck(report, "ERROR", CheckResult{
						"plan_slide": planIndex, "source_slide": slide.SlideIndex, "smartart_id": smartArt.SmartArtID,
						"node_id": node.NodeID, "message": "SmartArt node target not found",
					})
				}
				continue
			}
			addCheck(report, "OK", CheckResult{
				"plan_slide": planIndex, "source_slide": slide.SlideIndex, "smartart_id": smartArt.SmartArtID,
				"node_id": target.NodeID, "message": "SmartArt node target is valid",
			})
		}
	}
}

func findSmartArt(slide *SlideLibraryItem, edit SmartArtEdit) (*SmartArtInfo, string) {
	for _, selector := range []struct{ key, value string }{{"smartart_id", edit.SmartArtID}, {"shape_id", edit.ShapeID}, {"shape_name", edit.ShapeName}} {
		if selector.value == "" {
			continue
		}
		for index := range slide.SmartArts {
			item := &slide.SmartArts[index]
			if (selector.key == "smartart_id" && item.SmartArtID == selector.value) ||
				(selector.key == "shape_id" && item.ShapeID == selector.value) ||
				(selector.key == "shape_name" && item.ShapeName == selector.value) {
				return item, selector.key + ":" + selector.value
			}
		}
		return nil, selector.key + ":" + selector.value
	}
	return nil, ""
}

func smartArtNodeByEdit(info *SmartArtInfo, edit SmartArtNodeEdit, index int) *SmartArtNodeInfo {
	if edit.NodeID != "" {
		for nodeIndex := range info.Nodes {
			if info.Nodes[nodeIndex].NodeID == edit.NodeID {
				return &info.Nodes[nodeIndex]
			}
		}
		return nil
	}
	if index >= 0 && index < len(info.Nodes) {
		return &info.Nodes[index]
	}
	return nil
}

func applySmartArtEdits(pkg *Package, slide *xmlNode, rels *Relationships, sourceSlide int, slidePart string, edits []SmartArtEdit) error {
	if len(edits) == 0 {
		return nil
	}
	frames := smartArtFrames(slide)
	maps := map[string]*xmlNode{}
	for order, frame := range frames {
		id, name := shapeIdentity(frame, order+1)
		maps[fmt.Sprintf("smartart_id:s%02d_sa%s", sourceSlide, id)] = frame
		maps["shape_id:"+id] = frame
		if name != "" {
			maps["shape_name:"+name] = frame
		}
	}

	var missing []string
	for _, edit := range edits {
		selectors := smartArtSelectors(edit)
		var frame *xmlNode
		for _, selector := range selectors {
			if frame == nil {
				frame = maps[selector]
			}
		}
		if frame == nil {
			if !edit.Optional {
				missing = append(missing, selectorLabel(selectors))
			}
			continue
		}
		if err := applySmartArtEdit(pkg, frame, rels, sourceSlide, slidePart, edit); err != nil {
			return err
		}
	}
	if len(missing) != 0 {
		return fmt.Errorf("missing SmartArt target(s) on slide %d: %s", sourceSlide, strings.Join(missing, "; "))
	}
	return nil
}

func applySmartArtEdit(pkg *Package, frame *xmlNode, rels *Relationships, sourceSlide int, slidePart string, edit SmartArtEdit) error {
	shapeID, _ := shapeIdentity(frame, 0)
	relIDs := smartArtRelIDs(frame)
	if relIDs == nil {
		return fmt.Errorf("SmartArt %s on slide %d is missing diagram relationship IDs", shapeID, sourceSlide)
	}
	dataPart, err := relatedPartByID(slidePart, rels, relIDs.attr(nsOfficeRels, "dm"), RelationshipTypeDiagramData)
	if err != nil {
		return fmt.Errorf("SmartArt %s on slide %d: %w", shapeID, sourceSlide, err)
	}
	dataRoot, err := pkg.xmlPart(dataPart)
	if err != nil {
		return err
	}
	drawingRoot, drawingPart, err := smartArtDrawingRoot(pkg, slidePart, rels, dataRoot)
	if err != nil {
		return fmt.Errorf("SmartArt %s on slide %d: %w", shapeID, sourceSlide, err)
	}
	nodeRefs := smartArtNodeRefs(dataRoot, drawingRoot)
	if len(nodeRefs) == 0 {
		return fmt.Errorf("SmartArt %s on slide %d has no editable nodes", shapeID, sourceSlide)
	}

	var missing []string
	for index, nodeEdit := range edit.Nodes {
		nodeIndex := index
		if nodeEdit.NodeID != "" {
			nodeIndex = smartArtNodeIndex(sourceSlide, shapeID, nodeEdit.NodeID, len(nodeRefs))
		}
		if nodeIndex < 0 || nodeIndex >= len(nodeRefs) {
			if !nodeEdit.Optional {
				missing = append(missing, nodeEdit.NodeID)
			}
			continue
		}
		text := smartArtEditText(nodeEdit)
		ref := nodeRefs[nodeIndex]
		if !setSmartArtDataText(dataRoot, ref.ModelID, text) {
			return fmt.Errorf("SmartArt %s on slide %d: node data not found: %s", shapeID, sourceSlide, ref.ModelID)
		}
		if drawingRoot != nil {
			for _, presID := range ref.PresIDs {
				setSmartArtDrawingText(drawingRoot, presID, text)
			}
		}
	}
	if len(missing) != 0 {
		return fmt.Errorf("missing SmartArt node target(s) on slide %d: %s", sourceSlide, strings.Join(missing, "; "))
	}

	data, err := marshalXML(dataRoot)
	if err != nil {
		return err
	}
	if err := pkg.SetPart(dataPart, data); err != nil {
		return err
	}
	if drawingRoot != nil {
		data, err := marshalXML(drawingRoot)
		if err != nil {
			return err
		}
		if err := pkg.SetPart(drawingPart, data); err != nil {
			return err
		}
	}
	return nil
}

func relatedPartByID(owner string, rels *Relationships, id, relType string) (string, error) {
	rel, ok := rels.Find(id)
	if !ok || rel.Type != relType || rel.Mode != TargetInternal {
		return "", fmt.Errorf("relationship %s not found", id)
	}
	partName, err := ResolveTarget(owner, rel.Target)
	if err != nil {
		return "", err
	}
	return partName, nil
}

func smartArtDrawingRoot(pkg *Package, slidePart string, rels *Relationships, dataRoot *xmlNode) (*xmlNode, string, error) {
	ext := dataRoot.firstDescendant(nsDiagram2008, "dataModelExt")
	if ext == nil || ext.attr("", "relId") == "" {
		return nil, "", nil
	}
	partName, err := relatedPartByID(slidePart, rels, ext.attr("", "relId"), RelationshipTypeDiagramDrawing)
	if err != nil {
		return nil, "", err
	}
	root, err := pkg.xmlPart(partName)
	if err != nil {
		return nil, "", err
	}
	return root, partName, nil
}

func smartArtNodeIndex(sourceSlide int, shapeID, nodeID string, count int) int {
	prefix := fmt.Sprintf("s%02d_sa%s_n", sourceSlide, shapeID)
	if !strings.HasPrefix(nodeID, prefix) {
		return -1
	}
	value, err := strconv.Atoi(strings.TrimPrefix(nodeID, prefix))
	if err != nil || value < 1 || value > count {
		return -1
	}
	return value - 1
}

func setSmartArtDataText(dataRoot *xmlNode, modelID, text string) bool {
	for _, pt := range dataRoot.descendants(nsDiagram, "pt") {
		if pt.attr("", "modelId") != modelID {
			continue
		}
		prSet := pt.child(nsDiagram, "prSet")
		if prSet == nil {
			prSet = element(nsDiagram, "prSet")
			pt.Children = append([]*xmlNode{prSet}, pt.Children...)
		}
		prSet.removeAttr("", "phldr")
		body := pt.child(nsDiagram, "t")
		if body == nil {
			body = element(nsDiagram, "t")
			pt.Children = append(pt.Children, body)
		}
		setSmartArtTextBody(body, strings.Split(text, "\n"))
		return true
	}
	return false
}

func setSmartArtDrawingText(drawingRoot *xmlNode, presID, text string) bool {
	for _, shape := range drawingRoot.descendants(nsDiagram2008, "sp") {
		if shape.attr("", "modelId") != presID {
			continue
		}
		body := shape.child(nsDiagram2008, "txBody")
		if body == nil {
			return false
		}
		setSmartArtTextBody(body, strings.Split(text, "\n"))
		return true
	}
	return false
}

func setSmartArtTextBody(body *xmlNode, lines []string) {
	if len(lines) == 0 {
		lines = []string{""}
	}
	templates := body.children(nsDrawingML, "p")
	if len(templates) == 0 {
		templates = []*xmlNode{element(nsDrawingML, "p")}
	}
	for index := range templates {
		templates[index] = templates[index].clone()
	}
	body.removeChildren(nsDrawingML, "p")
	for index, line := range lines {
		paragraph := templates[min(index, len(templates)-1)].clone()
		setParagraphText(paragraph, line)
		body.Children = append(body.Children, paragraph)
	}
}

func smartArtEditText(edit SmartArtNodeEdit) string {
	if edit.Paragraphs != nil {
		return strings.Join(edit.Paragraphs, "\n")
	}
	return edit.Text
}

func smartArtSelectors(value SmartArtEdit) []string {
	var result []string
	if value.SmartArtID != "" {
		result = append(result, "smartart_id:"+value.SmartArtID)
	}
	if value.ShapeID != "" {
		result = append(result, "shape_id:"+value.ShapeID)
	}
	if value.ShapeName != "" {
		result = append(result, "shape_name:"+value.ShapeName)
	}
	return result
}
