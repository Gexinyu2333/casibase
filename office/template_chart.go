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

const (
	chartAxisDefaultFontPT       = 12.0
	chartAxisMinimumFontPT       = 8.0
	chartAxisMaximumLabelArea    = 0.40
	chartAxisMinimumPlotArea     = 0.55
	chartAxisDefaultOuterMargin  = 0.05
	chartAxisLabelSafetyFactor   = 0.90
	chartAxisCharacterWidthRatio = 0.52
)

type chartShape struct {
	ShapeID   string
	ShapeName string
	RelID     string
	WidthPX   float64
}

type chartCategoryAxis struct {
	Side       string
	FontSizePT float64
	PlotX      float64
	PlotWidth  float64
	HasLayout  bool
}

type chartCategoryLayout struct {
	Applicable              bool
	Changed                 bool
	Fits                    bool
	Side                    string
	FontSizePT              float64
	LabelArea               float64
	PlotX                   float64
	PlotWidth               float64
	LongestCategory         string
	LongestVisualWidth      float64
	SuggestedMaxVisualWidth float64
}

func applyChartEdits(pkg *Package, slide *xmlNode, rels *Relationships, types *ContentTypes, sourceSlide int, newSlidePart string, edits []ChartEdit, nextChart, nextEmbedding *int) error {
	maps := map[string]chartShape{}
	for order, container := range chartContainers(slide) {
		id, name := shapeIdentity(container, order+1)
		chart := container.firstDescendant(nsChart, "chart")
		relID := ""
		if chart != nil {
			relID = chart.attr(nsOfficeRels, "id")
		}
		width := 0.0
		if geometry := containerGeometry(container); geometry.Width != nil {
			width = float64(*geometry.Width)
		}
		info := chartShape{ShapeID: id, ShapeName: name, RelID: relID, WidthPX: width}
		maps[fmt.Sprintf("chart_id:s%02d_ch%s", sourceSlide, id)] = info
		maps["shape_id:"+id] = info
		if name != "" {
			maps["shape_name:"+name] = info
		}
	}
	cloned := map[string]string{}
	var missing []string
	for _, edit := range edits {
		selectors := chartSelectors(edit)
		var info chartShape
		found := false
		for _, selector := range selectors {
			if candidate, ok := maps[selector]; ok {
				info, found = candidate, true
				break
			}
		}
		if !found {
			if !edit.Optional {
				missing = append(missing, selectorLabel(selectors))
			}
			continue
		}
		rel, ok := rels.Find(info.RelID)
		if !ok || rel.Type != RelationshipTypeChart {
			missing = append(missing, fmt.Sprintf("%s relationship=%s", selectorLabel(selectors), info.RelID))
			continue
		}
		newChartPart := cloned[info.RelID]
		if newChartPart == "" {
			sourceChart, err := ResolveTarget(newSlidePart, rel.Target)
			if err != nil {
				return err
			}
			newChartPart = fmt.Sprintf("ppt/charts/chart%d.xml", *nextChart)
			*nextChart++
			if err := cloneAndUpdateChart(pkg, types, sourceChart, newChartPart, edit, info.WidthPX, nextEmbedding); err != nil {
				return err
			}
			target, err := RelativeTarget(newSlidePart, newChartPart)
			if err != nil {
				return err
			}
			rel.Target = target
			cloned[info.RelID] = newChartPart
		} else {
			root, err := pkg.xmlPart(newChartPart)
			if err != nil {
				return err
			}
			if _, err := updateChartXML(root, edit, info.WidthPX); err != nil {
				return err
			}
			data, err := marshalXML(root)
			if err != nil {
				return err
			}
			if err := pkg.SetPart(newChartPart, data); err != nil {
				return err
			}
		}
	}
	if len(missing) != 0 {
		return fmt.Errorf("missing chart edit target(s) on slide %d: %s", sourceSlide, strings.Join(missing, "; "))
	}
	return nil
}

func cloneAndUpdateChart(pkg *Package, types *ContentTypes, source, target string, edit ChartEdit, chartWidthPX float64, nextEmbedding *int) error {
	root, err := pkg.xmlPart(source)
	if err != nil {
		return err
	}
	if _, err := updateChartXML(root, edit, chartWidthPX); err != nil {
		return err
	}
	data, err := marshalXML(root)
	if err != nil {
		return err
	}
	if err := pkg.SetPart(target, data); err != nil {
		return err
	}
	if err := types.EnsureOverride(target, ContentTypeChart); err != nil {
		return err
	}
	sourceRels, err := pkg.Relationships(source)
	if err != nil {
		return err
	}
	if len(sourceRels.Items) == 0 {
		return nil
	}
	targetRels := &Relationships{Items: append([]Relationship(nil), sourceRels.Items...)}
	for index := range targetRels.Items {
		rel := &targetRels.Items[index]
		if rel.Mode != TargetInternal || (rel.Type != RelationshipTypeEmbeddedPackage && !strings.HasSuffix(strings.ToLower(rel.Target), ".xlsx")) {
			continue
		}
		workbook, err := ResolveTarget(source, rel.Target)
		if err != nil || !pkg.HasPart(workbook) {
			continue
		}
		workbookData, err := pkg.ReadPart(workbook)
		if err != nil {
			return err
		}
		rewritten, err := rewriteChartWorkbook(workbookData, edit)
		if err != nil {
			return err
		}
		newWorkbook := fmt.Sprintf("ppt/embeddings/templateFillChart%d.xlsx", *nextEmbedding)
		*nextEmbedding++
		if err := pkg.SetPart(newWorkbook, rewritten); err != nil {
			return err
		}
		if err := types.EnsureOverride(newWorkbook, ContentTypeEmbeddedXLSX); err != nil {
			return err
		}
		rel.Target, err = RelativeTarget(target, newWorkbook)
		if err != nil {
			return err
		}
	}
	return pkg.SetRelationships(target, targetRels)
}

func updateChartXML(root *xmlNode, edit ChartEdit, chartWidthPX float64) (chartCategoryLayout, error) {
	if len(edit.Categories) == 0 || len(edit.Series) == 0 {
		return chartCategoryLayout{}, fmt.Errorf("chart edit requires non-empty categories and series")
	}
	plot := root.firstDescendant(nsChart, "plotArea")
	if plot == nil {
		return chartCategoryLayout{}, fmt.Errorf("chart XML has no plotArea")
	}
	var chartType *xmlNode
	for _, child := range plot.Children {
		if strings.HasSuffix(child.Name.Local, "Chart") && len(child.children(nsChart, "ser")) != 0 {
			chartType = child
			break
		}
	}
	if chartType == nil {
		return chartCategoryLayout{}, fmt.Errorf("chart XML has no editable series")
	}
	seriesNodes := chartType.children(nsChart, "ser")
	template := seriesNodes[len(seriesNodes)-1]
	for len(seriesNodes) < len(edit.Series) {
		clone := template.clone()
		chartType.Children = append(chartType.Children, clone)
		seriesNodes = append(seriesNodes, clone)
	}
	keep := map[*xmlNode]bool{}
	for index := 0; index < len(edit.Series); index++ {
		keep[seriesNodes[index]] = true
	}
	filtered := chartType.Children[:0]
	for _, child := range chartType.Children {
		if child.Name.Space == nsChart && child.Name.Local == "ser" && !keep[child] {
			continue
		}
		filtered = append(filtered, child)
	}
	chartType.Children = filtered
	seriesNodes = chartType.children(nsChart, "ser")
	for index, payload := range edit.Series {
		if len(payload.Values) != len(edit.Categories) {
			return chartCategoryLayout{}, fmt.Errorf("chart series values must match categories length")
		}
		series := seriesNodes[index]
		ensureChartChild(series, "idx").setAttr("", "val", strconv.Itoa(index))
		ensureChartChild(series, "order").setAttr("", "val", strconv.Itoa(index))
		setSeriesName(series, payload.Name, index+2)
		setCategoryCache(series, edit.Categories)
		setValueCache(series, payload.Values, index+2)
	}
	layout := adaptHorizontalCategoryAxis(root, edit.Categories, chartWidthPX)
	return layout, nil
}

func inspectHorizontalCategoryAxis(root *xmlNode) *chartCategoryAxis {
	if root == nil {
		return nil
	}
	plot := root.firstDescendant(nsChart, "plotArea")
	if plot == nil {
		return nil
	}
	var barChart *xmlNode
	for _, child := range plot.Children {
		if child.Name.Space != nsChart || child.Name.Local != "barChart" {
			continue
		}
		if direction := child.child(nsChart, "barDir"); direction != nil && direction.attr("", "val") == "bar" {
			barChart = child
			break
		}
	}
	if barChart == nil {
		return nil
	}
	axis := plot.child(nsChart, "catAx")
	if axis == nil {
		return nil
	}
	side := "l"
	if position := axis.child(nsChart, "axPos"); position != nil {
		side = position.attr("", "val")
	}
	if side != "l" && side != "r" {
		return nil
	}
	fontSize := chartAxisDefaultFontPT
	if properties := axis.firstDescendant(nsDrawingML, "defRPr"); properties != nil {
		if raw, err := strconv.ParseFloat(properties.attr("", "sz"), 64); err == nil && raw > 0 {
			fontSize = raw / 100
		}
	}
	result := &chartCategoryAxis{
		Side: side, FontSizePT: fontSize,
		PlotX: chartAxisDefaultOuterMargin, PlotWidth: 1 - 2*chartAxisDefaultOuterMargin,
	}
	if manual := chartManualLayout(plot); manual != nil {
		x, xOK := chartLayoutValue(manual, "x")
		width, widthOK := chartLayoutValue(manual, "w")
		if xOK && widthOK && x >= 0 && width > 0 && x+width <= 1.001 {
			result.PlotX = x
			result.PlotWidth = width
			result.HasLayout = true
		}
	}
	return result
}

func calculateHorizontalCategoryLayout(axis *chartCategoryAxis, categories []string, chartWidthPX float64) chartCategoryLayout {
	if axis == nil || chartWidthPX <= 0 || len(categories) == 0 {
		return chartCategoryLayout{}
	}
	longest, longestWidth := longestChartCategory(categories)
	result := chartCategoryLayout{
		Applicable: true, Fits: true, Side: axis.Side, FontSizePT: axis.FontSizePT,
		PlotX: axis.PlotX, PlotWidth: axis.PlotWidth,
		LongestCategory: longest, LongestVisualWidth: longestWidth,
	}
	leftMargin := clampFloat(axis.PlotX, 0, 1)
	rightMargin := clampFloat(1-axis.PlotX-axis.PlotWidth, 0, 1)
	currentArea := leftMargin
	oppositeMargin := rightMargin
	if axis.Side == "r" {
		currentArea = rightMargin
		oppositeMargin = leftMargin
	}
	maximumArea := math.Min(chartAxisMaximumLabelArea, 1-oppositeMargin-chartAxisMinimumPlotArea)
	maximumArea = math.Max(maximumArea, currentArea)
	required := chartCategoryRequiredFraction(longestWidth, axis.FontSizePT, chartWidthPX)
	if required <= currentArea {
		result.LabelArea = currentArea
		result.SuggestedMaxVisualWidth = chartCategoryCapacity(currentArea, axis.FontSizePT, chartWidthPX)
		return result
	}

	fontSize := axis.FontSizePT
	if required > maximumArea {
		fontSize = math.Max(chartAxisMinimumFontPT, axis.FontSizePT*maximumArea/required)
		fontSize = math.Min(fontSize, axis.FontSizePT)
		required = chartCategoryRequiredFraction(longestWidth, fontSize, chartWidthPX)
	}
	labelArea := math.Min(math.Max(required, currentArea), maximumArea)
	result.FontSizePT = fontSize
	result.LabelArea = labelArea
	result.Fits = required <= maximumArea+.000001
	result.SuggestedMaxVisualWidth = chartCategoryCapacity(maximumArea, fontSize, chartWidthPX)
	if axis.Side == "l" {
		rightEdge := axis.PlotX + axis.PlotWidth
		result.PlotX = labelArea
		result.PlotWidth = math.Max(rightEdge-labelArea, chartAxisMinimumPlotArea)
	} else {
		result.PlotX = axis.PlotX
		result.PlotWidth = math.Max(1-axis.PlotX-labelArea, chartAxisMinimumPlotArea)
	}
	result.Changed = math.Abs(result.PlotX-axis.PlotX) > .000001 ||
		math.Abs(result.PlotWidth-axis.PlotWidth) > .000001 ||
		math.Abs(result.FontSizePT-axis.FontSizePT) > .000001
	return result
}

func adaptHorizontalCategoryAxis(root *xmlNode, categories []string, chartWidthPX float64) chartCategoryLayout {
	axisInfo := inspectHorizontalCategoryAxis(root)
	layout := calculateHorizontalCategoryLayout(axisInfo, categories, chartWidthPX)
	if !layout.Applicable || !layout.Changed {
		return layout
	}
	plot := root.firstDescendant(nsChart, "plotArea")
	axis := plot.child(nsChart, "catAx")
	manual := ensureChartManualLayout(plot, axisInfo)
	setChartLayoutValue(manual, "x", layout.PlotX)
	setChartLayoutValue(manual, "w", layout.PlotWidth)
	setChartAxisFontSize(axis, layout.FontSizePT)
	return layout
}

func longestChartCategory(categories []string) (string, float64) {
	longest := ""
	width := 0.0
	for _, category := range categories {
		candidate := visualWidth(category)
		if candidate > width {
			longest = category
			width = candidate
		}
	}
	return longest, width
}

func chartCategoryRequiredFraction(visualUnits, fontSizePT, chartWidthPX float64) float64 {
	fontPX := fontSizePT * pxPerInch / 72
	padding := fontPX * .5
	requiredPX := (visualUnits*fontPX*chartAxisCharacterWidthRatio + padding) / chartAxisLabelSafetyFactor
	return requiredPX / chartWidthPX
}

func chartCategoryCapacity(labelArea, fontSizePT, chartWidthPX float64) float64 {
	fontPX := fontSizePT * pxPerInch / 72
	available := labelArea*chartWidthPX*chartAxisLabelSafetyFactor - fontPX*.5
	return math.Max(available/(fontPX*chartAxisCharacterWidthRatio), 0)
}

func chartManualLayout(plot *xmlNode) *xmlNode {
	if layout := plot.child(nsChart, "layout"); layout != nil {
		return layout.child(nsChart, "manualLayout")
	}
	return nil
}

func ensureChartManualLayout(plot *xmlNode, axis *chartCategoryAxis) *xmlNode {
	layout := plot.child(nsChart, "layout")
	if layout == nil {
		layout = element(nsChart, "layout")
		plot.Children = append([]*xmlNode{layout}, plot.Children...)
	}
	manual := layout.child(nsChart, "manualLayout")
	if manual == nil {
		manual = element(nsChart, "manualLayout")
		manual.Children = []*xmlNode{
			element(nsChart, "layoutTarget", plainAttr("val", "inner")),
			element(nsChart, "xMode", plainAttr("val", "edge")),
			element(nsChart, "x", plainAttr("val", formatChartFraction(axis.PlotX))),
			element(nsChart, "w", plainAttr("val", formatChartFraction(axis.PlotWidth))),
		}
		layout.Children = append(layout.Children, manual)
	}
	return manual
}

func chartLayoutValue(manual *xmlNode, local string) (float64, bool) {
	node := manual.child(nsChart, local)
	if node == nil {
		return 0, false
	}
	value, err := strconv.ParseFloat(node.attr("", "val"), 64)
	return value, err == nil
}

func setChartLayoutValue(manual *xmlNode, local string, value float64) {
	node := manual.child(nsChart, local)
	if node == nil {
		node = element(nsChart, local)
		manual.Children = append(manual.Children, node)
	}
	node.setAttr("", "val", formatChartFraction(value))
}

func formatChartFraction(value float64) string {
	return strconv.FormatFloat(clampFloat(value, 0, 1), 'f', 8, 64)
}

func setChartAxisFontSize(axis *xmlNode, fontSizePT float64) {
	if axis == nil {
		return
	}
	txPr := axis.child(nsChart, "txPr")
	if txPr == nil {
		txPr = element(nsChart, "txPr")
		insertBeforeChartChild(axis, txPr, "crossAx")
	}
	bodyPr := txPr.child(nsDrawingML, "bodyPr")
	if bodyPr == nil {
		bodyPr = element(nsDrawingML, "bodyPr")
		txPr.Children = append([]*xmlNode{bodyPr}, txPr.Children...)
	}
	paragraph := txPr.child(nsDrawingML, "p")
	if paragraph == nil {
		paragraph = element(nsDrawingML, "p")
		txPr.Children = append(txPr.Children, paragraph)
	}
	paragraphProperties := paragraph.child(nsDrawingML, "pPr")
	if paragraphProperties == nil {
		paragraphProperties = element(nsDrawingML, "pPr")
		paragraph.Children = append([]*xmlNode{paragraphProperties}, paragraph.Children...)
	}
	defaultRun := paragraphProperties.child(nsDrawingML, "defRPr")
	if defaultRun == nil {
		defaultRun = element(nsDrawingML, "defRPr")
		paragraphProperties.Children = append(paragraphProperties.Children, defaultRun)
	}
	endRun := paragraph.child(nsDrawingML, "endParaRPr")
	if endRun == nil {
		endRun = element(nsDrawingML, "endParaRPr")
		paragraph.Children = append(paragraph.Children, endRun)
	}
	raw := strconv.Itoa(int(math.Round(fontSizePT * 100)))
	for _, local := range []string{"rPr", "defRPr", "endParaRPr"} {
		for _, properties := range txPr.descendants(nsDrawingML, local) {
			properties.setAttr("", "sz", raw)
		}
	}
}

func insertBeforeChartChild(parent, child *xmlNode, before string) {
	for index, candidate := range parent.Children {
		if candidate.Name.Space == nsChart && candidate.Name.Local == before {
			parent.Children = append(parent.Children, nil)
			copy(parent.Children[index+1:], parent.Children[index:])
			parent.Children[index] = child
			return
		}
	}
	parent.Children = append(parent.Children, child)
}

func clampFloat(value, low, high float64) float64 {
	return math.Min(math.Max(value, low), high)
}

func ensureChartChild(parent *xmlNode, local string) *xmlNode {
	if child := parent.child(nsChart, local); child != nil {
		return child
	}
	child := element(nsChart, local)
	parent.Children = append(parent.Children, child)
	return child
}

func setSeriesName(series *xmlNode, name string, column int) {
	tx := ensureChartChild(series, "tx")
	tx.Children = nil
	ref := element(nsChart, "strRef")
	formula := element(nsChart, "f")
	formula.Text = fmt.Sprintf("Sheet1!$%s$1", excelColumn(column))
	cache := element(nsChart, "strCache")
	writeChartCache(cache, []interface{}{name}, false)
	ref.Children = []*xmlNode{formula, cache}
	tx.Children = []*xmlNode{ref}
}

func setCategoryCache(series *xmlNode, categories []string) {
	category := ensureChartChild(series, "cat")
	ref := element(nsChart, "strRef")
	formula := element(nsChart, "f")
	formula.Text = fmt.Sprintf("Sheet1!$A$2:$A$%d", len(categories)+1)
	cache := element(nsChart, "strCache")
	values := make([]interface{}, len(categories))
	for index := range categories {
		values[index] = categories[index]
	}
	writeChartCache(cache, values, false)
	ref.Children = []*xmlNode{formula, cache}
	category.Children = []*xmlNode{ref}
}

func setValueCache(series *xmlNode, values []interface{}, column int) {
	value := ensureChartChild(series, "val")
	ref := element(nsChart, "numRef")
	formula := element(nsChart, "f")
	formula.Text = fmt.Sprintf("Sheet1!$%s$2:$%s$%d", excelColumn(column), excelColumn(column), len(values)+1)
	cache := element(nsChart, "numCache")
	writeChartCache(cache, values, true)
	ref.Children = []*xmlNode{formula, cache}
	value.Children = []*xmlNode{ref}
}

func writeChartCache(cache *xmlNode, values []interface{}, numeric bool) {
	cache.Children = nil
	if numeric {
		format := element(nsChart, "formatCode")
		format.Text = "General"
		cache.Children = append(cache.Children, format)
	}
	count := element(nsChart, "ptCount")
	count.setAttr("", "val", strconv.Itoa(len(values)))
	cache.Children = append(cache.Children, count)
	for index, value := range values {
		point := element(nsChart, "pt", plainAttr("idx", strconv.Itoa(index)))
		node := element(nsChart, "v")
		node.Text = fmt.Sprint(value)
		point.Children = []*xmlNode{node}
		cache.Children = append(cache.Children, point)
	}
}

func excelColumn(index int) string {
	var result string
	for index > 0 {
		index--
		result = string(rune('A'+index%26)) + result
		index /= 26
	}
	if result == "" {
		return "A"
	}
	return result
}

func rewriteChartWorkbook(data []byte, edit ChartEdit) ([]byte, error) {
	workbook, err := OpenBytes(data, DefaultLimits())
	if err != nil {
		return nil, err
	}
	sheetPart := "xl/worksheets/sheet1.xml"
	if root, err := workbook.xmlPart("xl/workbook.xml"); err == nil {
		if sheets := root.firstDescendant(nsSpreadsheetML, "sheets"); sheets != nil && len(sheets.Children) != 0 {
			relID := sheets.Children[0].attr(nsOfficeRels, "id")
			if rels, relErr := workbook.Relationships("xl/workbook.xml"); relErr == nil {
				if rel, ok := rels.Find(relID); ok {
					if resolved, resolveErr := ResolveTarget("xl/workbook.xml", rel.Target); resolveErr == nil {
						sheetPart = resolved
					}
				}
			}
		}
	}
	if !workbook.HasPart(sheetPart) {
		return data, nil
	}
	sheet, err := workbook.xmlPart(sheetPart)
	if err != nil {
		return nil, err
	}
	sheetData := sheet.firstDescendant(nsSpreadsheetML, "sheetData")
	if sheetData == nil {
		sheetData = element(nsSpreadsheetML, "sheetData")
		sheet.Children = append(sheet.Children, sheetData)
	}
	sheetData.Children = nil
	rows := make([][]interface{}, 0, len(edit.Categories)+1)
	header := []interface{}{"Category"}
	for index, series := range edit.Series {
		name := series.Name
		if name == "" {
			name = fmt.Sprintf("系列%d", index+1)
		}
		header = append(header, name)
	}
	rows = append(rows, header)
	for rowIndex, category := range edit.Categories {
		row := []interface{}{category}
		for _, series := range edit.Series {
			row = append(row, series.Values[rowIndex])
		}
		rows = append(rows, row)
	}
	for rowIndex, values := range rows {
		row := element(nsSpreadsheetML, "row", plainAttr("r", strconv.Itoa(rowIndex+1)))
		for colIndex, value := range values {
			row.Children = append(row.Children, spreadsheetCell(value, rowIndex+1, colIndex+1))
		}
		sheetData.Children = append(sheetData.Children, row)
	}
	xmlData, err := marshalXML(sheet)
	if err != nil {
		return nil, err
	}
	if err := workbook.SetPart(sheetPart, xmlData); err != nil {
		return nil, err
	}
	return workbook.Bytes()
}

func spreadsheetCell(value interface{}, row, col int) *xmlNode {
	cell := element(nsSpreadsheetML, "c", plainAttr("r", fmt.Sprintf("%s%d", excelColumn(col), row)))
	switch value.(type) {
	case int, int32, int64, float32, float64:
		node := element(nsSpreadsheetML, "v")
		node.Text = fmt.Sprint(value)
		cell.Children = []*xmlNode{node}
	default:
		cell.setAttr("", "t", "inlineStr")
		inline := element(nsSpreadsheetML, "is")
		text := element(nsSpreadsheetML, "t")
		text.Text = fmt.Sprint(value)
		inline.Children = []*xmlNode{text}
		cell.Children = []*xmlNode{inline}
	}
	return cell
}
