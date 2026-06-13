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
	"path/filepath"
	"strconv"
	"strings"
)

var (
	thanksKeywords  = []string{"thank", "thanks", "q&a", "qa", "contact", "致谢", "谢谢", "感谢", "答疑", "联系方式"}
	tocKeywords     = []string{"agenda", "contents", "content", "outline", "目录", "议程"}
	chapterKeywords = []string{"chapter", "part", "section", "章节", "部分"}
)

func AnalyzeFile(pptxPath string, limits Limits) (*Library, error) {
	absolute, err := filepath.Abs(pptxPath)
	if err != nil {
		return nil, err
	}
	pkg, err := OpenFile(absolute, limits)
	if err != nil {
		return nil, err
	}
	return Analyze(pkg, absolute)
}

func Analyze(pkg *Package, sourcePPTX string) (*Library, error) {
	if pkg == nil {
		return nil, fmt.Errorf("package cannot be nil")
	}
	if err := pkg.ValidatePresentation(); err != nil {
		return nil, err
	}
	presentation, err := pkg.xmlPart("ppt/presentation.xml")
	if err != nil {
		return nil, err
	}
	refs, err := pkg.slideRefs()
	if err != nil {
		return nil, err
	}
	library := &Library{
		Schema:     LibrarySchema,
		SourcePPTX: sourcePPTX,
		CanvasPX:   analyzeCanvas(presentation),
		Slides:     make([]SlideLibraryItem, 0, len(refs)),
		PlanContract: map[string]interface{}{
			"schema": PlanSchema,
			"slides": []interface{}{map[string]interface{}{
				"source_slide": 1,
				"purpose":      "封面 / 章节 / 内容 / 结尾",
				"replacements": []interface{}{map[string]interface{}{
					"slot_id": "s01_sh2", "text": "替换后的文字", "preserve_line_breaks": false,
				}},
				"table_edits": []interface{}{map[string]interface{}{
					"table_id": "s01_tbl3",
					"cells":    []interface{}{map[string]interface{}{"row": 0, "col": 0, "text": "替换后的单元格"}},
				}},
				"chart_edits": []interface{}{map[string]interface{}{
					"chart_id": "s01_ch4", "categories": []string{"A", "B"},
					"series": []interface{}{map[string]interface{}{"name": "系列1", "values": []int{1, 2}}},
				}},
				"image_edits": []interface{}{map[string]interface{}{
					"image_id": "s01_img5", "image_path": "https://example.com/image.png",
				}},
			}},
		},
	}
	for _, ref := range refs {
		slide, err := pkg.xmlPart(ref.PartName)
		if err != nil {
			return nil, err
		}
		layout, master, err := pkg.inheritanceRoots(ref.PartName)
		if err != nil {
			return nil, err
		}
		objects := analyzeObjects(slide)
		objectByID := make(map[string]*SlideObject, len(objects))
		for index := range objects {
			objectByID[objects[index].ShapeID] = &objects[index]
		}
		slots := analyzeSlots(slide, layout, master, ref.Index, objectByID)
		tables := analyzeTables(slide, ref.Index)
		charts, err := analyzeCharts(pkg, slide, ref)
		if err != nil {
			return nil, err
		}
		images, err := analyzeImages(pkg, slide, ref, objectByID)
		if err != nil {
			return nil, err
		}
		var textParts []string
		for _, slot := range slots {
			if slot.Text != "" {
				textParts = append(textParts, slot.Text)
			}
		}
		slideText := strings.Join(textParts, "\n")
		library.Slides = append(library.Slides, SlideLibraryItem{
			SlideIndex: ref.Index, PageType: classifyPageType(ref.Index, len(refs), slideText, len(slots)),
			TextSummary: truncateRunes(slideText, 500), Slots: slots, Tables: tables, Charts: charts,
			Images: images, Objects: objects,
		})
	}
	library.SlideCount = len(library.Slides)
	return library, nil
}

func analyzeCanvas(root *xmlNode) Canvas {
	size := root.firstDescendant(nsPresentation, "sldSz")
	if size == nil {
		return Canvas{}
	}
	return Canvas{Width: emuToPX(size.attr("", "cx")), Height: emuToPX(size.attr("", "cy"))}
}

func analyzeSlots(slide, layout, master *xmlNode, slideIndex int, objectByID map[string]*SlideObject) []TextSlot {
	containers := textContainers(slide)
	result := make([]TextSlot, 0, len(containers))
	for order, container := range containers {
		shapeID, shapeName := shapeIdentity(container, order+1)
		paragraphs := paragraphTexts(container)
		text := strings.Join(paragraphs, "\n")
		chain := inheritedChain(container, layout, master)
		geometry := containerGeometry(container)
		for _, inherited := range chain {
			candidate := containerGeometry(inherited)
			if candidate.Width != nil && candidate.Height != nil {
				geometry = candidate
				break
			}
		}
		metrics := textMetrics(chain, len(paragraphs))
		placeholder, _, _ := placeholderKey(container)
		role := slotRole(text, shapeName, placeholder, metrics, geometry, len(paragraphs), len(container.descendants(nsDrawingML, "t")))
		var zOrder *int
		if object := objectByID[shapeID]; object != nil {
			object.Geometry = geometry
			object.HasText = text != ""
			zOrder = ptrInt(object.ZOrder)
		}
		result = append(result, TextSlot{
			SlotID: fmt.Sprintf("s%02d_sh%s", slideIndex, shapeID), ShapeID: shapeID, ShapeName: shapeName,
			Role: role, Text: text, ParagraphCount: len(paragraphs), Geometry: geometry, TextMetrics: metrics,
			SingleLine: (role == "title_candidate" || role == "label_candidate") && !strings.Contains(text, "\n") && metrics.Wrap == "none",
			ZOrder:     zOrder,
		})
	}
	return result
}

func analyzeTables(slide *xmlNode, slideIndex int) []TableInfo {
	containers := tableContainers(slide)
	result := make([]TableInfo, 0, len(containers))
	for order, container := range containers {
		shapeID, shapeName := shapeIdentity(container, order+1)
		geometry := containerGeometry(container)
		table := container.firstDescendant(nsDrawingML, "tbl")
		rows := table.children(nsDrawingML, "tr")
		totalHeight := 0
		for _, row := range rows {
			if value, ok := intAttr(row, "", "h"); ok {
				totalHeight += value
			}
		}
		y := valueOrZero(geometry.Y)
		info := TableInfo{
			TableID: fmt.Sprintf("s%02d_tbl%s", slideIndex, shapeID), ShapeID: shapeID, ShapeName: shapeName,
			Geometry: geometry, RowCount: len(rows), Rows: make([]TableRow, 0, len(rows)),
		}
		for rowIndex, row := range rows {
			cells := row.children(nsDrawingML, "tc")
			info.ColumnCount = max(info.ColumnCount, len(cells))
			rowHeight := 0
			if geometry.Height != nil {
				if raw, ok := intAttr(row, "", "h"); ok && totalHeight > 0 {
					rowHeight = int(math.Round(float64(raw) / float64(totalHeight) * float64(*geometry.Height)))
				} else {
					rowHeight = int(math.Round(float64(*geometry.Height) / float64(max(len(rows), 1))))
				}
			}
			cellWidth := 0
			if geometry.Width != nil {
				cellWidth = int(math.Round(float64(*geometry.Width) / float64(max(len(cells), 1))))
			}
			rowInfo := TableRow{Row: rowIndex, Cells: make([]TableCell, 0, len(cells))}
			for colIndex, cell := range cells {
				paragraphs := paragraphTexts(cell)
				x := valueOrZero(geometry.X) + colIndex*cellWidth
				cellGeometry := Geometry{X: ptrInt(x), Y: ptrInt(y), Width: ptrInt(cellWidth), Height: ptrInt(rowHeight)}
				rowInfo.Cells = append(rowInfo.Cells, TableCell{
					Row: rowIndex, Col: colIndex, Text: strings.Join(paragraphs, "\n"), ParagraphCount: len(paragraphs),
					Geometry: cellGeometry, TextMetrics: textMetrics([]*xmlNode{cell}, len(paragraphs)),
				})
			}
			info.Rows = append(info.Rows, rowInfo)
			y += rowHeight
		}
		result = append(result, info)
	}
	return result
}

func analyzeCharts(pkg *Package, slide *xmlNode, ref slideRef) ([]ChartInfo, error) {
	rels, err := pkg.Relationships(ref.PartName)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]Relationship, len(rels.Items))
	for _, rel := range rels.Items {
		byID[rel.ID] = rel
	}
	containers := chartContainers(slide)
	result := make([]ChartInfo, 0, len(containers))
	for order, container := range containers {
		shapeID, shapeName := shapeIdentity(container, order+1)
		info := ChartInfo{
			ChartID: fmt.Sprintf("s%02d_ch%s", ref.Index, shapeID), ShapeID: shapeID, ShapeName: shapeName,
			Categories: []interface{}{}, Series: []ChartSeries{},
			geometry: containerGeometry(container),
		}
		chartRef := container.firstDescendant(nsChart, "chart")
		if chartRef != nil {
			rel := byID[chartRef.attr(nsOfficeRels, "id")]
			if rel.Type == RelationshipTypeChart && rel.Mode == TargetInternal {
				partName, resolveErr := ResolveTarget(ref.PartName, rel.Target)
				if resolveErr == nil && pkg.HasPart(partName) {
					root, parseErr := pkg.xmlPart(partName)
					if parseErr == nil {
						readChartData(root, &info)
						info.categoryAxis = inspectHorizontalCategoryAxis(root)
					}
				}
			}
		}
		result = append(result, info)
	}
	return result, nil
}

func readChartData(root *xmlNode, info *ChartInfo) {
	plot := root.firstDescendant(nsChart, "plotArea")
	if plot == nil {
		return
	}
	var chartType *xmlNode
	for _, child := range plot.Children {
		if strings.HasSuffix(child.Name.Local, "Chart") && len(child.children(nsChart, "ser")) != 0 {
			chartType = child
			break
		}
	}
	if chartType == nil {
		return
	}
	kind := chartType.Name.Local
	info.ChartType = &kind
	seriesNodes := chartType.children(nsChart, "ser")
	if len(seriesNodes) != 0 {
		info.Categories = chartCacheValues(seriesNodes[0].child(nsChart, "cat"), false)
	}
	for index, series := range seriesNodes {
		name := fmt.Sprintf("系列%d", index+1)
		if tx := series.child(nsChart, "tx"); tx != nil {
			values := chartCacheValues(tx, false)
			if len(values) != 0 {
				name = fmt.Sprint(values[0])
			} else if direct := tx.child(nsChart, "v"); direct != nil && strings.TrimSpace(textContent(direct)) != "" {
				name = strings.TrimSpace(textContent(direct))
			}
		}
		info.Series = append(info.Series, ChartSeries{Name: name, Values: chartCacheValues(series.child(nsChart, "val"), true)})
	}
	info.CategoryCount = len(info.Categories)
	info.SeriesCount = len(info.Series)
}

func chartCacheValues(parent *xmlNode, numeric bool) []interface{} {
	if parent == nil {
		return []interface{}{}
	}
	cache := parent.firstDescendant(nsChart, "strCache")
	if cache == nil {
		cache = parent.firstDescendant(nsChart, "numCache")
	}
	if cache == nil {
		return []interface{}{}
	}
	result := make([]interface{}, 0)
	for _, point := range cache.children(nsChart, "pt") {
		valueNode := point.child(nsChart, "v")
		value := ""
		if valueNode != nil {
			value = textContent(valueNode)
		}
		if numeric {
			if number, err := strconv.ParseFloat(value, 64); err == nil {
				if number == math.Trunc(number) {
					result = append(result, int(number))
				} else {
					result = append(result, number)
				}
				continue
			}
		}
		result = append(result, value)
	}
	return result
}

func analyzeObjects(slide *xmlNode) []SlideObject {
	tree := slide.firstDescendant(nsPresentation, "spTree")
	if tree == nil {
		return []SlideObject{}
	}
	var result []SlideObject
	var walk func(*xmlNode, *objectTransform)
	walk = func(parent *xmlNode, parentTransform *objectTransform) {
		for _, child := range parent.Children {
			if child.Name.Space != nsPresentation {
				continue
			}
			if child.Name.Local == "grpSp" {
				groupTransform := readObjectTransform(child, true)
				if groupTransform != nil && parentTransform != nil {
					absolute := absoluteObjectGeometry(groupTransform, parentTransform)
					groupTransform.X = float64(*absolute.X) / pxPerInch * emuPerInch
					groupTransform.Y = float64(*absolute.Y) / pxPerInch * emuPerInch
					groupTransform.Width = float64(*absolute.Width) / pxPerInch * emuPerInch
					groupTransform.Height = float64(*absolute.Height) / pxPerInch * emuPerInch
				}
				walk(child, groupTransform)
				continue
			}
			switch child.Name.Local {
			case "sp", "pic", "cxnSp", "graphicFrame":
			default:
				continue
			}
			shapeID, shapeName := shapeIdentity(child, len(result)+1)
			geometry := Geometry{}
			if transform := readObjectTransform(child, false); transform != nil {
				geometry = absoluteObjectGeometry(transform, parentTransform)
			} else {
				rotation := 0.0
				geometry.Rotation = &rotation
			}
			result = append(result, SlideObject{
				ShapeID: shapeID, ShapeName: shapeName, Kind: child.Name.Local,
				Geometry: geometry, ZOrder: len(result), HasText: len(paragraphTexts(child)) != 0,
			})
		}
	}
	walk(tree, nil)
	return result
}

type objectTransform struct {
	X, Y, Width, Height                     float64
	Rotation                                float64
	ChildX, ChildY, ChildWidth, ChildHeight float64
}

func readObjectTransform(node *xmlNode, group bool) *objectTransform {
	var transform *xmlNode
	if group {
		if properties := node.child(nsPresentation, "grpSpPr"); properties != nil {
			transform = properties.child(nsDrawingML, "xfrm")
		}
	} else if node.Name.Local == "graphicFrame" {
		transform = node.child(nsPresentation, "xfrm")
	} else if properties := node.child(nsPresentation, "spPr"); properties != nil {
		transform = properties.child(nsDrawingML, "xfrm")
	}
	if transform == nil {
		return nil
	}
	offset, extent := transform.child(nsDrawingML, "off"), transform.child(nsDrawingML, "ext")
	if offset == nil || extent == nil {
		return nil
	}
	x, xOK := floatAttr(offset, "", "x")
	y, yOK := floatAttr(offset, "", "y")
	width, widthOK := floatAttr(extent, "", "cx")
	height, heightOK := floatAttr(extent, "", "cy")
	if !xOK || !yOK || !widthOK || !heightOK {
		return nil
	}
	result := &objectTransform{X: x, Y: y, Width: width, Height: height}
	if rotation, ok := floatAttr(transform, "", "rot"); ok {
		result.Rotation = rotation / 60000
	}
	childOffset, childExtent := transform.child(nsDrawingML, "chOff"), transform.child(nsDrawingML, "chExt")
	if childOffset != nil && childExtent != nil {
		result.ChildX, _ = floatAttr(childOffset, "", "x")
		result.ChildY, _ = floatAttr(childOffset, "", "y")
		result.ChildWidth, _ = floatAttr(childExtent, "", "cx")
		result.ChildHeight, _ = floatAttr(childExtent, "", "cy")
	}
	return result
}

func absoluteObjectGeometry(value, parent *objectTransform) Geometry {
	x, y, width, height := value.X, value.Y, value.Width, value.Height
	if parent != nil && parent.ChildWidth != 0 && parent.ChildHeight != 0 {
		scaleX, scaleY := parent.Width/parent.ChildWidth, parent.Height/parent.ChildHeight
		x = parent.X + (value.X-parent.ChildX)*scaleX
		y = parent.Y + (value.Y-parent.ChildY)*scaleY
		width, height = value.Width*scaleX, value.Height*scaleY
	}
	rotation := math.Mod(value.Rotation, 360)
	if parent != nil {
		rotation = math.Mod(rotation+parent.Rotation, 360)
	}
	if rotation != 0 {
		radians := rotation * math.Pi / 180
		boundingWidth := math.Abs(width*math.Cos(radians)) + math.Abs(height*math.Sin(radians))
		boundingHeight := math.Abs(width*math.Sin(radians)) + math.Abs(height*math.Cos(radians))
		x += (width - boundingWidth) / 2
		y += (height - boundingHeight) / 2
		width, height = boundingWidth, boundingHeight
	}
	px := func(value float64) *int {
		result := int(math.Round(value / emuPerInch * pxPerInch))
		return &result
	}
	rotation = math.Round(rotation*100) / 100
	return Geometry{X: px(x), Y: px(y), Width: px(width), Height: px(height), Rotation: &rotation}
}

func classifyPageType(index, total int, text string, slotCount int) string {
	normalized := strings.ToLower(text)
	if index == 1 {
		return "cover_candidate"
	}
	if index == total || containsAny(normalized, thanksKeywords) {
		return "ending_candidate"
	}
	if containsAny(normalized, tocKeywords) {
		return "toc_candidate"
	}
	if containsAny(normalized, chapterKeywords) || (slotCount <= 2 && len([]rune(text)) <= 80) {
		return "chapter_candidate"
	}
	return "content_candidate"
}

func containsAny(value string, terms []string) bool {
	for _, term := range terms {
		if strings.Contains(value, term) {
			return true
		}
	}
	return false
}

func valueOrZero(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
