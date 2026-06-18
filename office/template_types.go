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

import "encoding/json"

const (
	LibrarySchema = "template_fill_pptx_library.v1"
	PlanSchema    = "template_fill_pptx_plan.v1"
	CheckSchema   = "template_fill_pptx_check.v1"
)

type Geometry struct {
	X        *int     `json:"x"`
	Y        *int     `json:"y"`
	Width    *int     `json:"width"`
	Height   *int     `json:"height"`
	Rotation *float64 `json:"rotation,omitempty"`
}

type TextMetrics struct {
	FontSizePX  *float64       `json:"font_size_px"`
	Paragraphs  int            `json:"paragraph_count"`
	LineSpacing *float64       `json:"line_spacing"`
	LineSpacePX *float64       `json:"line_spacing_px"`
	Wrap        string         `json:"wrap"`
	Autofit     string         `json:"autofit"`
	MarginsPX   map[string]int `json:"margins_px"`
	Anchor      string         `json:"anchor"`
	Alignment   string         `json:"alignment"`
}

type TextSlot struct {
	SlotID         string      `json:"slot_id"`
	ShapeID        string      `json:"shape_id"`
	ShapeName      string      `json:"shape_name"`
	Role           string      `json:"role"`
	Text           string      `json:"text"`
	ParagraphCount int         `json:"paragraph_count"`
	Geometry       Geometry    `json:"geometry"`
	TextMetrics    TextMetrics `json:"text_metrics"`
	SingleLine     bool        `json:"single_line"`
	ZOrder         *int        `json:"z_order"`
}

type TableCell struct {
	Row            int         `json:"row"`
	Col            int         `json:"col"`
	Text           string      `json:"text"`
	ParagraphCount int         `json:"paragraph_count"`
	Geometry       Geometry    `json:"geometry"`
	TextMetrics    TextMetrics `json:"text_metrics"`
}

type TableRow struct {
	Row   int         `json:"row"`
	Cells []TableCell `json:"cells"`
}

type TableInfo struct {
	TableID     string     `json:"table_id"`
	ShapeID     string     `json:"shape_id"`
	ShapeName   string     `json:"shape_name"`
	Geometry    Geometry   `json:"geometry"`
	RowCount    int        `json:"row_count"`
	ColumnCount int        `json:"column_count"`
	Rows        []TableRow `json:"rows"`
}

type ChartSeries struct {
	Name   string        `json:"name"`
	Values []interface{} `json:"values"`
}

type ChartInfo struct {
	ChartID       string        `json:"chart_id"`
	ShapeID       string        `json:"-"`
	ShapeName     string        `json:"-"`
	ChartType     *string       `json:"chart_type"`
	CategoryCount int           `json:"category_count"`
	SeriesCount   int           `json:"series_count"`
	Categories    []interface{} `json:"categories"`
	Series        []ChartSeries `json:"series"`
	geometry      Geometry
	categoryAxis  *chartCategoryAxis
}

type ImageInfo struct {
	ImageID     string   `json:"image_id"`
	ShapeID     string   `json:"shape_id"`
	ShapeName   string   `json:"shape_name"`
	Description string   `json:"description,omitempty"`
	Geometry    Geometry `json:"geometry"`
}

type SmartArtNodeInfo struct {
	NodeID         string `json:"node_id"`
	Text           string `json:"text"`
	ParagraphCount int    `json:"paragraph_count"`
	Editable       bool   `json:"editable"`
	Reason         string `json:"reason,omitempty"`
	modelID        string
	presIDs        []string
}

type SmartArtInfo struct {
	SmartArtID string             `json:"smartart_id"`
	ShapeID    string             `json:"shape_id"`
	ShapeName  string             `json:"shape_name"`
	Geometry   Geometry           `json:"geometry"`
	Editable   bool               `json:"editable"`
	Reason     string             `json:"reason,omitempty"`
	Nodes      []SmartArtNodeInfo `json:"nodes"`
}

type SlideObject struct {
	ShapeID   string   `json:"shape_id"`
	ShapeName string   `json:"shape_name"`
	Kind      string   `json:"kind"`
	Geometry  Geometry `json:"geometry"`
	ZOrder    int      `json:"z_order"`
	HasText   bool     `json:"has_text"`
}

type SlideLibraryItem struct {
	SlideIndex  int            `json:"slide_index"`
	PageType    string         `json:"page_type"`
	TextSummary string         `json:"text_summary"`
	Slots       []TextSlot     `json:"slots"`
	Tables      []TableInfo    `json:"tables"`
	Charts      []ChartInfo    `json:"charts"`
	Images      []ImageInfo    `json:"images"`
	SmartArts   []SmartArtInfo `json:"smartarts"`
	Objects     []SlideObject  `json:"objects"`
}

type Canvas struct {
	Width  *int `json:"width"`
	Height *int `json:"height"`
}

type Library struct {
	Schema       string             `json:"schema"`
	SourcePPTX   string             `json:"source_pptx"`
	SlideCount   int                `json:"slide_count"`
	CanvasPX     Canvas             `json:"canvas_px"`
	Slides       []SlideLibraryItem `json:"slides"`
	PlanContract interface{}        `json:"plan_contract"`
}

type Replacement struct {
	SlotID             string   `json:"slot_id,omitempty"`
	ShapeID            string   `json:"shape_id,omitempty"`
	ShapeName          string   `json:"shape_name,omitempty"`
	Text               string   `json:"text,omitempty"`
	Paragraphs         []string `json:"paragraphs,omitempty"`
	PreserveLineBreaks bool     `json:"preserve_line_breaks,omitempty"`
	Optional           bool     `json:"optional,omitempty"`
	OldText            string   `json:"old_text,omitempty"`
}

type TableCellEdit struct {
	Row        int      `json:"row"`
	Col        int      `json:"col"`
	Text       string   `json:"text,omitempty"`
	Paragraphs []string `json:"paragraphs,omitempty"`
	OldText    string   `json:"old_text,omitempty"`
}

type TableEdit struct {
	TableID   string          `json:"table_id,omitempty"`
	ShapeID   string          `json:"shape_id,omitempty"`
	ShapeName string          `json:"shape_name,omitempty"`
	Cells     []TableCellEdit `json:"cells"`
	Optional  bool            `json:"optional,omitempty"`
}

type ImageEdit struct {
	ImageID   string `json:"image_id"`
	ImagePath string `json:"image_path"`
}

type ChartEdit struct {
	ChartID    string        `json:"chart_id,omitempty"`
	ShapeID    string        `json:"shape_id,omitempty"`
	ShapeName  string        `json:"shape_name,omitempty"`
	Categories []string      `json:"categories"`
	Series     []ChartSeries `json:"series"`
	Optional   bool          `json:"optional,omitempty"`
}

type SmartArtNodeEdit struct {
	NodeID     string   `json:"node_id,omitempty"`
	Text       string   `json:"text,omitempty"`
	Paragraphs []string `json:"paragraphs,omitempty"`
	Optional   bool     `json:"optional,omitempty"`
}

type SmartArtEdit struct {
	SmartArtID string             `json:"smartart_id,omitempty"`
	ShapeID    string             `json:"shape_id,omitempty"`
	ShapeName  string             `json:"shape_name,omitempty"`
	Nodes      []SmartArtNodeEdit `json:"nodes"`
	Optional   bool               `json:"optional,omitempty"`
}

type PlanSlide struct {
	SourceSlide        int             `json:"source_slide"`
	Purpose            string          `json:"purpose,omitempty"`
	Replacements       []Replacement   `json:"replacements,omitempty"`
	TableEdits         []TableEdit     `json:"table_edits,omitempty"`
	ChartEdits         []ChartEdit     `json:"chart_edits,omitempty"`
	ImageEdits         []ImageEdit     `json:"image_edits,omitempty"`
	SmartArtEdits      []SmartArtEdit  `json:"smartart_edits,omitempty"`
	Notes              string          `json:"notes,omitempty"`
	SpeakerNotes       string          `json:"speaker_notes,omitempty"`
	Transition         json.RawMessage `json:"transition,omitempty"`
	TransitionDuration *float64        `json:"transition_duration,omitempty"`
	AdvanceAfter       *float64        `json:"advance_after,omitempty"`
}

type Plan struct {
	Schema     string      `json:"schema"`
	SourcePPTX string      `json:"source_pptx,omitempty"`
	Slides     []PlanSlide `json:"slides"`
}

type CheckSummary struct {
	OK    int `json:"ok"`
	Warn  int `json:"warn"`
	Error int `json:"error"`
}

type CheckResult map[string]interface{}

type CheckReport struct {
	Schema  string        `json:"schema"`
	Summary CheckSummary  `json:"summary"`
	Results []CheckResult `json:"results"`
}

type ApplyOptions struct {
	Transition         string
	TransitionDuration float64
	Library            *Library
}
