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
	"path"
	"strconv"
	"strings"
	"unicode"
)

const (
	emuPerInch = 914400.0
	pxPerInch  = 96.0
)

type slideRef struct {
	Index    int
	RelID    string
	PartName string
	RelsName string
}

func (p *Package) slideRefs() ([]slideRef, error) {
	presentation, err := p.xmlPart("ppt/presentation.xml")
	if err != nil {
		return nil, err
	}
	rels, err := p.Relationships("ppt/presentation.xml")
	if err != nil {
		return nil, err
	}
	byID := make(map[string]Relationship, len(rels.Items))
	for _, rel := range rels.Items {
		byID[rel.ID] = rel
	}
	list := presentation.firstDescendant(nsPresentation, "sldIdLst")
	if list == nil {
		return []slideRef{}, nil
	}
	var result []slideRef
	for _, item := range list.children(nsPresentation, "sldId") {
		relID := item.attr(nsOfficeRels, "id")
		rel, ok := byID[relID]
		if !ok || rel.Type != RelationshipTypeSlide || rel.Mode != TargetInternal {
			continue
		}
		partName, err := ResolveTarget("ppt/presentation.xml", rel.Target)
		if err != nil {
			return nil, err
		}
		relsName, err := RelationshipsPart(partName)
		if err != nil {
			return nil, err
		}
		result = append(result, slideRef{Index: len(result) + 1, RelID: relID, PartName: partName, RelsName: relsName})
	}
	return result, nil
}

func (p *Package) xmlPart(name string) (*xmlNode, error) {
	data, err := p.ReadPart(name)
	if err != nil {
		return nil, err
	}
	root, err := parseXML(data)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", name, err)
	}
	return root, nil
}

func emuToPX(raw string) *int {
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil
	}
	result := int(math.Round(float64(value) / emuPerInch * pxPerInch))
	return &result
}

func shapeIdentity(container *xmlNode, order int) (string, string) {
	property := container.firstDescendant(nsPresentation, "cNvPr")
	if property == nil {
		return strconv.Itoa(order), ""
	}
	id := property.attr("", "id")
	if id == "" {
		id = strconv.Itoa(order)
	}
	return id, property.attr("", "name")
}

func textContainers(root *xmlNode) []*xmlNode {
	var result []*xmlNode
	for _, local := range []string{"sp", "graphicFrame"} {
		for _, candidate := range root.descendants(nsPresentation, local) {
			if candidate.firstDescendant(nsPresentation, "txBody") != nil ||
				candidate.firstDescendant(nsDrawingML, "txBody") != nil ||
				candidate.firstDescendant(nsDrawingML, "t") != nil {
				result = append(result, candidate)
			}
		}
	}
	return result
}

func tableContainers(root *xmlNode) []*xmlNode {
	var result []*xmlNode
	for _, candidate := range root.descendants(nsPresentation, "graphicFrame") {
		if candidate.firstDescendant(nsDrawingML, "tbl") != nil {
			result = append(result, candidate)
		}
	}
	return result
}

func chartContainers(root *xmlNode) []*xmlNode {
	var result []*xmlNode
	for _, candidate := range root.descendants(nsPresentation, "graphicFrame") {
		if candidate.firstDescendant(nsChart, "chart") != nil {
			result = append(result, candidate)
		}
	}
	return result
}

func paragraphTexts(container *xmlNode) []string {
	var paragraphs []string
	for _, paragraph := range container.descendants(nsDrawingML, "p") {
		var builder strings.Builder
		for _, text := range paragraph.descendants(nsDrawingML, "t") {
			builder.WriteString(textContent(text))
		}
		value := strings.TrimSpace(builder.String())
		if value != "" {
			paragraphs = append(paragraphs, value)
		}
	}
	if len(paragraphs) != 0 {
		return paragraphs
	}
	var builder strings.Builder
	for _, text := range container.descendants(nsDrawingML, "t") {
		builder.WriteString(textContent(text))
	}
	value := strings.TrimSpace(builder.String())
	if value != "" {
		return []string{value}
	}
	return []string{}
}

func containerGeometry(container *xmlNode) Geometry {
	var transform *xmlNode
	if properties := container.child(nsPresentation, "spPr"); properties != nil {
		transform = properties.child(nsDrawingML, "xfrm")
	}
	if transform == nil {
		transform = container.child(nsPresentation, "xfrm")
	}
	if transform == nil {
		transform = container.firstDescendant(nsDrawingML, "xfrm")
	}
	if transform == nil {
		return Geometry{}
	}
	offset := transform.child(nsDrawingML, "off")
	extent := transform.child(nsDrawingML, "ext")
	var result Geometry
	if offset != nil {
		result.X = emuToPX(offset.attr("", "x"))
		result.Y = emuToPX(offset.attr("", "y"))
	}
	if extent != nil {
		result.Width = emuToPX(extent.attr("", "cx"))
		result.Height = emuToPX(extent.attr("", "cy"))
	}
	return result
}

func placeholderKey(container *xmlNode) (string, string, bool) {
	placeholder := container.firstDescendant(nsPresentation, "ph")
	if placeholder == nil {
		return "", "", false
	}
	kind := placeholder.attr("", "type")
	if kind == "" {
		kind = "body"
	}
	return kind, placeholder.attr("", "idx"), true
}

func placeholderMap(root *xmlNode) map[string]*xmlNode {
	result := map[string]*xmlNode{}
	if root == nil {
		return result
	}
	for _, container := range root.descendants(nsPresentation, "sp") {
		kind, index, ok := placeholderKey(container)
		if ok {
			result[kind+"\x00"+index] = container
		}
	}
	return result
}

func inheritedChain(container, layout, master *xmlNode) []*xmlNode {
	result := []*xmlNode{container}
	kind, index, ok := placeholderKey(container)
	if !ok {
		return result
	}
	key := kind + "\x00" + index
	if inherited := placeholderMap(layout)[key]; inherited != nil {
		result = append(result, inherited)
	}
	if inherited := placeholderMap(master)[key]; inherited != nil {
		result = append(result, inherited)
	}
	return result
}

func (p *Package) relatedPart(owner, relType string) (string, bool, error) {
	rels, err := p.Relationships(owner)
	if err != nil {
		return "", false, err
	}
	for _, rel := range rels.Items {
		if rel.Type != relType || rel.Mode != TargetInternal {
			continue
		}
		target, err := ResolveTarget(owner, rel.Target)
		return target, err == nil, err
	}
	return "", false, nil
}

func (p *Package) inheritanceRoots(slidePart string) (*xmlNode, *xmlNode, error) {
	layoutPart, ok, err := p.relatedPart(slidePart, RelationshipTypeSlideLayout)
	if err != nil || !ok {
		return nil, nil, err
	}
	layout, err := p.xmlPart(layoutPart)
	if err != nil {
		return nil, nil, err
	}
	masterPart, ok, err := p.relatedPart(layoutPart, RelationshipTypeSlideMaster)
	if err != nil || !ok {
		return layout, nil, err
	}
	master, err := p.xmlPart(masterPart)
	return layout, master, err
}

func textMetrics(chain []*xmlNode, paragraphCount int) TextMetrics {
	result := TextMetrics{
		Paragraphs: paragraphCount,
		Wrap:       "square",
		Autofit:    "none",
		MarginsPX:  map[string]int{},
		Anchor:     "t",
		Alignment:  "l",
	}
	var sizes []float64
	for _, container := range chain {
		for _, local := range []string{"rPr", "defRPr"} {
			for _, properties := range container.descendants(nsDrawingML, local) {
				if value, ok := floatAttr(properties, "", "sz"); ok {
					sizes = append(sizes, value/100*96/72)
				}
			}
		}
		if len(sizes) != 0 {
			break
		}
	}
	for _, size := range sizes {
		if result.FontSizePX == nil || size > *result.FontSizePX {
			value := math.Round(size*100) / 100
			result.FontSizePX = &value
		}
	}
	for _, container := range chain {
		body := container.firstDescendant(nsDrawingML, "bodyPr")
		if body == nil {
			continue
		}
		if value := body.attr("", "wrap"); value != "" {
			result.Wrap = value
		}
		if value := body.attr("", "anchor"); value != "" {
			result.Anchor = value
		}
		if body.child(nsDrawingML, "normAutofit") != nil {
			result.Autofit = "normal"
		} else if body.child(nsDrawingML, "spAutoFit") != nil {
			result.Autofit = "shape"
		} else if body.child(nsDrawingML, "noAutofit") != nil {
			result.Autofit = "none"
		}
		for key, attribute := range map[string]string{"left": "lIns", "right": "rIns", "top": "tIns", "bottom": "bIns"} {
			if px := emuToPX(body.attr("", attribute)); px != nil {
				result.MarginsPX[key] = *px
			}
		}
		break
	}
	for _, container := range chain {
		for _, paragraph := range container.descendants(nsDrawingML, "pPr") {
			if alignment := paragraph.attr("", "algn"); alignment != "" {
				result.Alignment = alignment
			}
			if spacing := paragraph.child(nsDrawingML, "lnSpc"); spacing != nil {
				if percent := spacing.child(nsDrawingML, "spcPct"); percent != nil {
					if value, ok := floatAttr(percent, "", "val"); ok {
						value /= 100000
						result.LineSpacing = &value
					}
				}
				if points := spacing.child(nsDrawingML, "spcPts"); points != nil {
					if value, ok := floatAttr(points, "", "val"); ok {
						value = value / 100 * 96 / 72
						result.LineSpacePX = &value
					}
				}
			}
			return result
		}
	}
	return result
}

func slotRole(text, name, placeholder string, metrics TextMetrics, geometry Geometry, paragraphs, textNodes int) string {
	lowerName := strings.ToLower(name)
	if placeholder == "title" || placeholder == "ctrTitle" || strings.Contains(lowerName, "title") || strings.Contains(name, "标题") {
		return "title_candidate"
	}
	if metrics.FontSizePX != nil && *metrics.FontSizePX >= 30 && geometry.Y != nil && *geometry.Y < 100 &&
		geometry.Width != nil && *geometry.Width >= 300 {
		return "title_candidate"
	}
	if placeholder == "body" || placeholder == "subTitle" || placeholder == "obj" {
		return "body_candidate"
	}
	if paragraphs > 1 || (metrics.Wrap != "none" && (len([]rune(text)) >= 36 || textNodes >= 3)) || len([]rune(text)) >= 72 {
		return "body_candidate"
	}
	return "label_candidate"
}

func visualWidth(value string) float64 {
	var width float64
	for _, char := range value {
		if unicode.IsSpace(char) {
			continue
		}
		if char >= 0x1100 && (char <= 0x115f || char >= 0x2e80) {
			width += 2
		} else {
			width++
		}
	}
	return width
}

func ptrInt(value int) *int { return &value }

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) > limit {
		runes = runes[:limit]
	}
	return string(runes)
}

func partBaseNumber(name, directory, prefix, suffix string) int {
	if path.Dir(name) != directory {
		return 0
	}
	base := path.Base(name)
	if !strings.HasPrefix(base, prefix) || !strings.HasSuffix(base, suffix) {
		return 0
	}
	value, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(base, prefix), suffix))
	return value
}
