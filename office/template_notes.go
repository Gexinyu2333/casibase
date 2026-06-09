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
	"encoding/xml"
	"regexp"
	"strings"
)

var (
	markdownBoldA = regexp.MustCompile(`\*\*(.+?)\*\*`)
	markdownBoldB = regexp.MustCompile(`__(.+?)__`)
	markdownHead  = regexp.MustCompile(`^#+\s*`)
)

func markdownToPlainText(value string) string {
	stripBold := func(text string) string {
		text = markdownBoldA.ReplaceAllString(text, "$1")
		return markdownBoldB.ReplaceAllString(text, "$1")
	}
	var lines []string
	for _, raw := range strings.Split(value, "\n") {
		trimmed := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(raw, "#"):
			text := stripBold(strings.TrimSpace(markdownHead.ReplaceAllString(raw, "")))
			if text != "" {
				lines = append(lines, text, "")
			}
		case strings.HasPrefix(trimmed, "- "):
			lines = append(lines, "• "+stripBold(strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))))
		case trimmed != "":
			lines = append(lines, stripBold(trimmed))
		default:
			lines = append(lines, "")
		}
	}
	result := lines[:0]
	previousEmpty := false
	for _, line := range lines {
		if line != "" {
			result = append(result, line)
			previousEmpty = false
		} else if !previousEmpty {
			result = append(result, "")
			previousEmpty = true
		}
	}
	return strings.TrimSpace(strings.Join(result, "\n"))
}

func findNotesMasterPart(pkg *Package) (string, bool) {
	if part, ok, err := pkg.relatedPart("ppt/presentation.xml", RelationshipTypeNotesMaster); err == nil && ok {
		return part, true
	}
	for _, name := range pkg.PartNames() {
		if !strings.HasPrefix(name, "ppt/notesSlides/notesSlide") || !strings.HasSuffix(name, ".xml") {
			continue
		}
		if part, ok, err := pkg.relatedPart(name, RelationshipTypeNotesMaster); err == nil && ok {
			return part, true
		}
	}
	return "", false
}

func setSlideNotes(pkg *Package, slidePart string, slideNumber int, rels *Relationships, notes string, types *ContentTypes) error {
	rels.RemoveByType(RelationshipTypeNotesSlide)
	notes = strings.TrimSpace(notes)
	if notes == "" {
		return nil
	}
	notesPart := "ppt/notesSlides/notesSlide" + itoa(slideNumber) + ".xml"
	root := createNotesXML(markdownToPlainText(notes))
	data, err := marshalXML(root)
	if err != nil {
		return err
	}
	if err := pkg.SetPart(notesPart, data); err != nil {
		return err
	}
	if err := types.EnsureOverride(notesPart, ContentTypeNotesSlide); err != nil {
		return err
	}
	target, err := RelativeTarget(slidePart, notesPart)
	if err != nil {
		return err
	}
	if err := rels.Add(Relationship{ID: rels.NextID(), Type: RelationshipTypeNotesSlide, Target: target, Mode: TargetInternal}); err != nil {
		return err
	}
	notesRels := &Relationships{}
	if master, ok := findNotesMasterPart(pkg); ok {
		masterTarget, err := RelativeTarget(notesPart, master)
		if err != nil {
			return err
		}
		notesRels.Items = append(notesRels.Items, Relationship{ID: notesRels.NextID(), Type: RelationshipTypeNotesMaster, Target: masterTarget, Mode: TargetInternal})
	}
	slideTarget, err := RelativeTarget(notesPart, slidePart)
	if err != nil {
		return err
	}
	notesRels.Items = append(notesRels.Items, Relationship{ID: notesRels.NextID(), Type: RelationshipTypeSlide, Target: slideTarget, Mode: TargetInternal})
	return pkg.SetRelationships(notesPart, notesRels)
}

func createNotesXML(notes string) *xmlNode {
	root := element(nsPresentation, "notes")
	common := element(nsPresentation, "cSld")
	tree := element(nsPresentation, "spTree")
	group := element(nsPresentation, "nvGrpSpPr")
	group.Children = []*xmlNode{
		element(nsPresentation, "cNvPr", plainAttr("id", "1"), plainAttr("name", "")),
		element(nsPresentation, "cNvGrpSpPr"), element(nsPresentation, "nvPr"),
	}
	groupProperties := element(nsPresentation, "grpSpPr")
	transform := element(nsDrawingML, "xfrm")
	transform.Children = []*xmlNode{
		element(nsDrawingML, "off", plainAttr("x", "0"), plainAttr("y", "0")),
		element(nsDrawingML, "ext", plainAttr("cx", "0"), plainAttr("cy", "0")),
		element(nsDrawingML, "chOff", plainAttr("x", "0"), plainAttr("y", "0")),
		element(nsDrawingML, "chExt", plainAttr("cx", "0"), plainAttr("cy", "0")),
	}
	groupProperties.Children = []*xmlNode{transform}
	tree.Children = append(tree.Children, group, groupProperties, slideImagePlaceholder(), notesPlaceholder())
	common.Children = []*xmlNode{tree}
	color := element(nsPresentation, "clrMapOvr")
	color.Children = []*xmlNode{element(nsDrawingML, "masterClrMapping")}
	root.Children = []*xmlNode{common, color}

	body := tree.Children[len(tree.Children)-1].child(nsPresentation, "txBody")
	for _, line := range strings.Split(notes, "\n") {
		paragraph := element(nsDrawingML, "p")
		if strings.TrimSpace(line) == "" {
			paragraph.Children = []*xmlNode{element(nsDrawingML, "endParaRPr", plainAttr("lang", "zh-CN"), plainAttr("dirty", "0"))}
		} else {
			run := element(nsDrawingML, "r")
			run.Children = []*xmlNode{
				element(nsDrawingML, "rPr", plainAttr("lang", "zh-CN"), plainAttr("dirty", "0")),
				{Name: xmlName(nsDrawingML, "t"), Text: line},
			}
			paragraph.Children = []*xmlNode{run}
		}
		body.Children = append(body.Children, paragraph)
	}
	if notes == "" {
		body.Children = append(body.Children, element(nsDrawingML, "p"))
	}
	return root
}

func slideImagePlaceholder() *xmlNode {
	shape := element(nsPresentation, "sp")
	nonVisual := element(nsPresentation, "nvSpPr")
	nonVisualProps := element(nsPresentation, "cNvPr", plainAttr("id", "2"), plainAttr("name", "Slide Image Placeholder 1"))
	nonVisualShape := element(nsPresentation, "cNvSpPr")
	locks := element(nsDrawingML, "spLocks",
		plainAttr("noGrp", "1"),
		plainAttr("noRot", "1"),
		plainAttr("noChangeAspect", "1"),
	)
	nonVisualShape.Children = []*xmlNode{locks}
	app := element(nsPresentation, "nvPr")
	app.Children = []*xmlNode{element(nsPresentation, "ph", plainAttr("type", "sldImg"))}
	nonVisual.Children = []*xmlNode{nonVisualProps, nonVisualShape, app}
	shape.Children = []*xmlNode{nonVisual, element(nsPresentation, "spPr")}
	return shape
}

func notesPlaceholder() *xmlNode {
	shape := element(nsPresentation, "sp")
	nonVisual := element(nsPresentation, "nvSpPr")
	nonVisualProps := element(nsPresentation, "cNvPr", plainAttr("id", "3"), plainAttr("name", "Notes Placeholder 2"))
	nonVisualShape := element(nsPresentation, "cNvSpPr")
	nonVisualShape.Children = []*xmlNode{element(nsDrawingML, "spLocks", plainAttr("noGrp", "1"))}
	app := element(nsPresentation, "nvPr")
	app.Children = []*xmlNode{element(nsPresentation, "ph", plainAttr("type", "body"), plainAttr("idx", "1"))}
	nonVisual.Children = []*xmlNode{nonVisualProps, nonVisualShape, app}
	body := element(nsPresentation, "txBody")
	body.Children = []*xmlNode{element(nsDrawingML, "bodyPr"), element(nsDrawingML, "lstStyle")}
	shape.Children = []*xmlNode{nonVisual, element(nsPresentation, "spPr"), body}
	return shape
}

func xmlName(space, local string) xml.Name {
	return xml.Name{Space: space, Local: local}
}

func itoa(value int) string {
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}
	var buffer [20]byte
	index := len(buffer)
	for value > 0 {
		index--
		buffer[index] = digits[value%10]
		value /= 10
	}
	return string(buffer[index:])
}
