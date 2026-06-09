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
	"math"
	"strings"
)

const textLayoutSafetyFactor = 0.90

type textLayout struct {
	Scale          *float64
	FontSizePX     *float64
	LineCount      int
	OccupiedWidth  *float64
	OccupiedHeight *float64
}

func fallbackFontSize(role string, geometry Geometry, oldParagraphs int) float64 {
	if geometry.Height != nil && oldParagraphs > 0 {
		inferred := float64(*geometry.Height) / float64(oldParagraphs) / 1.25
		if inferred >= 8 && inferred <= 56 {
			return inferred
		}
	}
	switch role {
	case "title_candidate":
		return 28
	case "body_candidate":
		return 16
	default:
		return 14
	}
}

func estimateTextLayout(text, role string, oldParagraphs int, geometry Geometry, metrics TextMetrics, singleLine bool) textLayout {
	return estimateTextLayoutWithSafety(text, role, oldParagraphs, geometry, metrics, singleLine, textLayoutSafetyFactor)
}

func estimateTextLayoutWithSafety(text, role string, oldParagraphs int, geometry Geometry, metrics TextMetrics, singleLine bool, safetyFactor float64) textLayout {
	if geometry.Width == nil || geometry.Height == nil || *geometry.Width <= 0 || *geometry.Height <= 0 {
		return textLayout{LineCount: max(len(strings.Split(text, "\n")), 1)}
	}
	if safetyFactor <= 0 || safetyFactor > 1 {
		safetyFactor = 1
	}
	left, right := float64(metricMargin(metrics, "left", 12)), float64(metricMargin(metrics, "right", 12))
	top, bottom := float64(metricMargin(metrics, "top", 4)), float64(metricMargin(metrics, "bottom", 4))
	usableWidth := math.Max((float64(*geometry.Width)-left-right)*safetyFactor, 1)
	usableHeight := math.Max((float64(*geometry.Height)-top-bottom)*safetyFactor, 1)
	baseFont := fallbackFontSize(role, geometry, oldParagraphs)
	if metrics.FontSizePX != nil && *metrics.FontSizePX > 0 {
		baseFont = *metrics.FontSizePX
	}
	fits := func(scale float64) (bool, int, float64, float64) {
		font := math.Max(baseFont*scale, .01)
		units := usableWidth / math.Max(font*.52, 1)
		switch role {
		case "label_candidate":
			units *= .82
		case "title_candidate":
			units *= .9
		}
		units = math.Max(units, 1)
		lines := 0
		parts := strings.Split(text, "\n")
		if singleLine {
			lines = 1
		} else {
			for _, line := range parts {
				if strings.TrimSpace(line) == "" {
					continue
				}
				lines += max(1, int(math.Ceil(visualWidth(line)/units)))
			}
			lines = max(lines, 1)
		}
		occupiedWidth := visualWidth(text) * font * .52
		if !singleLine {
			occupiedWidth = 0
			for _, line := range parts {
				occupiedWidth = math.Max(occupiedWidth, math.Min(visualWidth(line), units)*font*.52)
			}
			occupiedWidth = math.Min(occupiedWidth, usableWidth)
		}
		lineHeight := font * 1.25
		if metrics.LineSpacing != nil && *metrics.LineSpacing > 0 {
			lineHeight = font * *metrics.LineSpacing
		} else if metrics.LineSpacePX != nil && *metrics.LineSpacePX > 0 {
			lineHeight = *metrics.LineSpacePX
			if metrics.FontSizePX != nil && *metrics.FontSizePX > 0 {
				lineHeight *= font / *metrics.FontSizePX
			}
		}
		occupiedHeight := float64(lines) * math.Max(lineHeight, 1)
		return occupiedWidth <= usableWidth && occupiedHeight <= usableHeight, lines, occupiedWidth, occupiedHeight
	}
	ok, lines, width, height := fits(1)
	scale := 1.0
	if !ok {
		low, high := .001, 1.0
		for range 24 {
			middle := (low + high) / 2
			if fit, _, _, _ := fits(middle); fit {
				low = middle
			} else {
				high = middle
			}
		}
		scale = low
		_, lines, width, height = fits(scale)
	}
	font := baseFont * scale
	width = math.Min(width, usableWidth)
	height = math.Min(height, usableHeight)
	return textLayout{Scale: &scale, FontSizePX: &font, LineCount: lines, OccupiedWidth: &width, OccupiedHeight: &height}
}

func metricMargin(metrics TextMetrics, name string, fallback int) int {
	if value, ok := metrics.MarginsPX[name]; ok {
		return value
	}
	return fallback
}

func occupiedRect(geometry Geometry, metrics TextMetrics, layout textLayout) (map[string]float64, bool) {
	if geometry.X == nil || geometry.Y == nil || geometry.Width == nil || geometry.Height == nil ||
		layout.OccupiedWidth == nil || layout.OccupiedHeight == nil {
		return nil, false
	}
	left := float64(metricMargin(metrics, "left", 12))
	right := float64(metricMargin(metrics, "right", 12))
	top := float64(metricMargin(metrics, "top", 4))
	bottom := float64(metricMargin(metrics, "bottom", 4))
	availableWidth := math.Max(float64(*geometry.Width)-left-right, 0)
	availableHeight := math.Max(float64(*geometry.Height)-top-bottom, 0)
	switch metrics.Alignment {
	case "ctr", "center":
		left += math.Max((availableWidth-*layout.OccupiedWidth)/2, 0)
	case "r", "right":
		left += math.Max(availableWidth-*layout.OccupiedWidth, 0)
	}
	switch metrics.Anchor {
	case "ctr", "mid":
		top += math.Max((availableHeight-*layout.OccupiedHeight)/2, 0)
	case "b", "bottom":
		top += math.Max(availableHeight-*layout.OccupiedHeight, 0)
	}
	return map[string]float64{
		"x": float64(*geometry.X) + left, "y": float64(*geometry.Y) + top,
		"width": *layout.OccupiedWidth, "height": *layout.OccupiedHeight,
	}, true
}

func intersectionArea(first map[string]float64, second Geometry) float64 {
	if second.X == nil || second.Y == nil || second.Width == nil || second.Height == nil {
		return 0
	}
	left := math.Max(first["x"], float64(*second.X))
	top := math.Max(first["y"], float64(*second.Y))
	right := math.Min(first["x"]+first["width"], float64(*second.X+*second.Width))
	bottom := math.Min(first["y"]+first["height"], float64(*second.Y+*second.Height))
	return math.Max(right-left, 0) * math.Max(bottom-top, 0)
}
