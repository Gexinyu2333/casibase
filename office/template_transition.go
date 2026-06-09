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
	"encoding/json"
	"fmt"
	"strconv"
)

type transitionSpec struct {
	Effect       string   `json:"effect"`
	Duration     *float64 `json:"duration"`
	AdvanceAfter *float64 `json:"advance_after"`
}

var transitionElements = map[string]struct {
	Element string
	Attrs   map[string]string
}{
	"fade":       {"fade", nil},
	"push":       {"push", map[string]string{"dir": "l"}},
	"wipe":       {"wipe", map[string]string{"dir": "l"}},
	"split":      {"split", map[string]string{"orient": "horz", "dir": "out"}},
	"cover":      {"cover", map[string]string{"dir": "l"}},
	"uncover":    {"pull", map[string]string{"dir": "l"}},
	"randomBars": {"randomBar", map[string]string{"dir": "vert"}},
	"wheel":      {"wheel", map[string]string{"spokes": "1"}},
	"circle":     {"circle", nil},
	"diamond":    {"diamond", nil},
	"plus":       {"plus", nil},
	"zoom":       {"zoom", map[string]string{"dir": "in"}},
}

func resolveTransition(item PlanSlide, defaultEffect string, defaultDuration float64) (string, float64, *float64, error) {
	effect, duration := defaultEffect, defaultDuration
	advance := item.AdvanceAfter
	if len(item.Transition) != 0 && string(item.Transition) != "null" {
		var asString string
		if json.Unmarshal(item.Transition, &asString) == nil {
			effect = asString
		} else {
			var spec transitionSpec
			if err := json.Unmarshal(item.Transition, &spec); err != nil {
				return "", 0, nil, fmt.Errorf("invalid slide transition: %w", err)
			}
			if spec.Effect != "" {
				effect = spec.Effect
			}
			if spec.Duration != nil {
				duration = *spec.Duration
			}
			if spec.AdvanceAfter != nil {
				advance = spec.AdvanceAfter
			}
		}
	}
	if item.TransitionDuration != nil {
		duration = *item.TransitionDuration
	}
	if effect != "" && effect != "keep" && effect != "none" {
		if _, ok := transitionElements[effect]; !ok {
			return "", 0, nil, fmt.Errorf("unknown transition effect %q", effect)
		}
	}
	return effect, duration, advance, nil
}

func setSlideTransition(slide *xmlNode, effect string, duration float64, advance *float64) {
	if effect == "" || effect == "keep" {
		return
	}
	existing := -1
	for index, child := range slide.Children {
		if child.Name.Space == nsPresentation && child.Name.Local == "transition" {
			existing = index
			break
		}
	}
	if existing >= 0 {
		slide.Children = append(slide.Children[:existing], slide.Children[existing+1:]...)
	}
	if effect == "none" {
		return
	}
	info := transitionElements[effect]
	transition := element(nsPresentation, "transition")
	transition.setAttr(nsP14, "dur", strconv.Itoa(int(duration*1000)))
	if advance != nil {
		transition.setAttr("", "advTm", strconv.Itoa(int(*advance*1000)))
	}
	child := element(nsPresentation, info.Element)
	for key, value := range info.Attrs {
		child.setAttr("", key, value)
	}
	transition.Children = append(transition.Children, child)
	insert := len(slide.Children)
	for index, child := range slide.Children {
		if child.Name.Space == nsPresentation && child.Name.Local == "timing" {
			insert = index
			break
		}
		if child.Name.Space == nsPresentation && child.Name.Local == "clrMapOvr" {
			insert = index + 1
		}
	}
	slide.Children = append(slide.Children, nil)
	copy(slide.Children[insert+1:], slide.Children[insert:])
	slide.Children[insert] = transition
}
