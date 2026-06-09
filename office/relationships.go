// Copyright 2026 The OpenAgent Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0

package office

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type TargetMode uint8

const (
	TargetInternal TargetMode = iota
	TargetExternal
)

type Relationship struct {
	ID     string
	Type   string
	Target string
	Mode   TargetMode
}

type Relationships struct {
	Items []Relationship
}

type relationshipsXML struct {
	XMLName xml.Name          `xml:"http://schemas.openxmlformats.org/package/2006/relationships Relationships"`
	Items   []relationshipXML `xml:"Relationship"`
}

type relationshipXML struct {
	ID         string `xml:"Id,attr"`
	Type       string `xml:"Type,attr"`
	Target     string `xml:"Target,attr"`
	TargetMode string `xml:"TargetMode,attr,omitempty"`
}

func (p *Package) Relationships(owner string) (*Relationships, error) {
	relsName, err := RelationshipsPart(owner)
	if err != nil {
		return nil, err
	}
	item, ok := p.parts[relsName]
	if !ok {
		return &Relationships{}, nil
	}
	var payload relationshipsXML
	decoder := xml.NewDecoder(bytes.NewReader(item.data))
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("parse relationships %s: %w", relsName, err)
	}
	result := &Relationships{Items: make([]Relationship, 0, len(payload.Items))}
	seen := make(map[string]struct{}, len(payload.Items))
	for _, raw := range payload.Items {
		mode, err := parseTargetMode(raw.TargetMode)
		if err != nil {
			return nil, fmt.Errorf("parse relationships %s: %w", relsName, err)
		}
		rel := Relationship{
			ID:     raw.ID,
			Type:   raw.Type,
			Target: raw.Target,
			Mode:   mode,
		}
		if err := validateRelationship(owner, rel); err != nil {
			return nil, fmt.Errorf("parse relationships %s: %w", relsName, err)
		}
		if _, ok := seen[rel.ID]; ok {
			return nil, fmt.Errorf("parse relationships %s: duplicate relationship ID %q", relsName, rel.ID)
		}
		seen[rel.ID] = struct{}{}
		result.Items = append(result.Items, rel)
	}
	return result, nil
}

func (p *Package) SetRelationships(owner string, rels *Relationships) error {
	if rels == nil {
		return fmt.Errorf("relationships cannot be nil")
	}
	relsName, err := RelationshipsPart(owner)
	if err != nil {
		return err
	}
	payload := relationshipsXML{Items: make([]relationshipXML, 0, len(rels.Items))}
	seen := make(map[string]struct{}, len(rels.Items))
	for _, rel := range rels.Items {
		if err := validateRelationship(owner, rel); err != nil {
			return err
		}
		if _, ok := seen[rel.ID]; ok {
			return fmt.Errorf("duplicate relationship ID %q", rel.ID)
		}
		seen[rel.ID] = struct{}{}
		raw := relationshipXML{
			ID:     rel.ID,
			Type:   rel.Type,
			Target: rel.Target,
		}
		if rel.Mode == TargetExternal {
			raw.TargetMode = "External"
		}
		payload.Items = append(payload.Items, raw)
	}
	data, err := marshalControlXML(payload)
	if err != nil {
		return err
	}
	return p.SetPart(relsName, data)
}

func parseTargetMode(value string) (TargetMode, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "internal":
		return TargetInternal, nil
	case "external":
		return TargetExternal, nil
	default:
		return TargetInternal, fmt.Errorf("unsupported relationship TargetMode %q", value)
	}
}

func validateRelationship(owner string, rel Relationship) error {
	if strings.TrimSpace(rel.ID) == "" {
		return fmt.Errorf("relationship ID is empty")
	}
	if strings.TrimSpace(rel.Type) == "" {
		return fmt.Errorf("relationship %s type is empty", rel.ID)
	}
	if rel.Target == "" {
		return fmt.Errorf("relationship %s target is empty", rel.ID)
	}
	switch rel.Mode {
	case TargetInternal:
		if _, err := ResolveTarget(owner, rel.Target); err != nil {
			return fmt.Errorf("relationship %s: %w", rel.ID, err)
		}
	case TargetExternal:
		// External targets are intentionally opaque so malformed but
		// round-trippable hyperlinks remain usable.
	default:
		return fmt.Errorf("relationship %s has an invalid target mode", rel.ID)
	}
	return nil
}

func (r *Relationships) Find(id string) (*Relationship, bool) {
	for index := range r.Items {
		if r.Items[index].ID == id {
			return &r.Items[index], true
		}
	}
	return nil, false
}

func (r *Relationships) Add(item Relationship) error {
	if _, ok := r.Find(item.ID); ok {
		return fmt.Errorf("duplicate relationship ID %q", item.ID)
	}
	if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.Type) == "" || item.Target == "" {
		return fmt.Errorf("relationship ID, type, and target are required")
	}
	if item.Mode != TargetInternal && item.Mode != TargetExternal {
		return fmt.Errorf("relationship %s has an invalid target mode", item.ID)
	}
	r.Items = append(r.Items, item)
	return nil
}

func (r *Relationships) Remove(id string) bool {
	for index := range r.Items {
		if r.Items[index].ID == id {
			r.Items = append(r.Items[:index], r.Items[index+1:]...)
			return true
		}
	}
	return false
}

func (r *Relationships) RemoveByType(relType string) int {
	kept := r.Items[:0]
	removed := 0
	for _, item := range r.Items {
		if item.Type == relType {
			removed++
			continue
		}
		kept = append(kept, item)
	}
	r.Items = kept
	return removed
}

var numericRID = regexp.MustCompile(`^rId([0-9]+)$`)

func (r *Relationships) NextID() string {
	maxID := 0
	for _, item := range r.Items {
		match := numericRID.FindStringSubmatch(item.ID)
		if len(match) != 2 {
			continue
		}
		value, err := strconv.Atoi(match[1])
		if err == nil && value > maxID {
			maxID = value
		}
	}
	return fmt.Sprintf("rId%d", maxID+1)
}

func marshalControlXML(value any) ([]byte, error) {
	data, err := xml.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), append(data, '\n')...), nil
}
