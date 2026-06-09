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
	"path"
	"strings"
)

type DefaultContentType struct {
	Extension   string
	ContentType string
}

type OverrideContentType struct {
	PartName    string
	ContentType string
}

type ContentTypes struct {
	Defaults  []DefaultContentType
	Overrides []OverrideContentType
}

type contentTypesXML struct {
	XMLName   xml.Name          `xml:"http://schemas.openxmlformats.org/package/2006/content-types Types"`
	Defaults  []contentDefault  `xml:"Default"`
	Overrides []contentOverride `xml:"Override"`
}

type contentDefault struct {
	Extension   string `xml:"Extension,attr"`
	ContentType string `xml:"ContentType,attr"`
}

type contentOverride struct {
	PartName    string `xml:"PartName,attr"`
	ContentType string `xml:"ContentType,attr"`
}

func (p *Package) ContentTypes() (*ContentTypes, error) {
	item, ok := p.parts[contentTypesPart]
	if !ok {
		return nil, fmt.Errorf("OOXML package is missing %s", contentTypesPart)
	}
	var payload contentTypesXML
	decoder := xml.NewDecoder(bytes.NewReader(item.data))
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("parse %s: %w", contentTypesPart, err)
	}
	result := &ContentTypes{
		Defaults:  make([]DefaultContentType, 0, len(payload.Defaults)),
		Overrides: make([]OverrideContentType, 0, len(payload.Overrides)),
	}
	defaults := make(map[string]string, len(payload.Defaults))
	for _, item := range payload.Defaults {
		extension := strings.TrimPrefix(strings.TrimSpace(item.Extension), ".")
		if extension == "" || strings.ContainsAny(extension, `/\`) || strings.TrimSpace(item.ContentType) == "" {
			return nil, fmt.Errorf("invalid default content type for extension %q", item.Extension)
		}
		key := strings.ToLower(extension)
		if existing, ok := defaults[key]; ok {
			if existing != item.ContentType {
				return nil, fmt.Errorf("conflicting default content types for extension %q", extension)
			}
			return nil, fmt.Errorf("duplicate default content type for extension %q", extension)
		}
		defaults[key] = item.ContentType
		result.Defaults = append(result.Defaults, DefaultContentType{Extension: extension, ContentType: item.ContentType})
	}
	overrides := make(map[string]string, len(payload.Overrides))
	for _, item := range payload.Overrides {
		partName, err := NormalizePartName(item.PartName)
		if err != nil || !strings.HasPrefix(item.PartName, "/") || strings.TrimSpace(item.ContentType) == "" {
			return nil, fmt.Errorf("invalid content type override for part %q", item.PartName)
		}
		if existing, ok := overrides[partName]; ok {
			if existing != item.ContentType {
				return nil, fmt.Errorf("conflicting content type overrides for part %q", partName)
			}
			return nil, fmt.Errorf("duplicate content type override for part %q", partName)
		}
		overrides[partName] = item.ContentType
		result.Overrides = append(result.Overrides, OverrideContentType{PartName: partName, ContentType: item.ContentType})
	}
	return result, nil
}

func (p *Package) SetContentTypes(types *ContentTypes) error {
	if types == nil {
		return fmt.Errorf("content types cannot be nil")
	}
	payload := contentTypesXML{
		Defaults:  make([]contentDefault, 0, len(types.Defaults)),
		Overrides: make([]contentOverride, 0, len(types.Overrides)),
	}
	seenDefaults := make(map[string]string, len(types.Defaults))
	for _, item := range types.Defaults {
		extension := strings.TrimPrefix(strings.TrimSpace(item.Extension), ".")
		if extension == "" || strings.ContainsAny(extension, `/\`) || strings.TrimSpace(item.ContentType) == "" {
			return fmt.Errorf("invalid default content type for extension %q", item.Extension)
		}
		key := strings.ToLower(extension)
		if existing, ok := seenDefaults[key]; ok {
			return fmt.Errorf("duplicate or conflicting default content type for extension %q: %q and %q", extension, existing, item.ContentType)
		}
		seenDefaults[key] = item.ContentType
		payload.Defaults = append(payload.Defaults, contentDefault{Extension: extension, ContentType: item.ContentType})
	}
	seenOverrides := make(map[string]string, len(types.Overrides))
	for _, item := range types.Overrides {
		partName, err := NormalizePartName(item.PartName)
		if err != nil || strings.TrimSpace(item.ContentType) == "" {
			return fmt.Errorf("invalid content type override for part %q", item.PartName)
		}
		if existing, ok := seenOverrides[partName]; ok {
			return fmt.Errorf("duplicate or conflicting content type override for part %q: %q and %q", partName, existing, item.ContentType)
		}
		seenOverrides[partName] = item.ContentType
		payload.Overrides = append(payload.Overrides, contentOverride{PartName: "/" + partName, ContentType: item.ContentType})
	}
	data, err := marshalControlXML(payload)
	if err != nil {
		return err
	}
	return p.SetPart(contentTypesPart, data)
}

func (c *ContentTypes) Lookup(partName string) (string, bool) {
	normalized, err := NormalizePartName(partName)
	if err != nil {
		return "", false
	}
	for _, item := range c.Overrides {
		itemName, err := NormalizePartName(item.PartName)
		if err == nil && itemName == normalized {
			return item.ContentType, true
		}
	}
	extension := strings.TrimPrefix(path.Ext(normalized), ".")
	for _, item := range c.Defaults {
		if strings.EqualFold(strings.TrimPrefix(item.Extension, "."), extension) {
			return item.ContentType, true
		}
	}
	return "", false
}

func (c *ContentTypes) EnsureOverride(partName, contentType string) error {
	normalized, err := NormalizePartName(partName)
	if err != nil {
		return err
	}
	if strings.TrimSpace(contentType) == "" {
		return fmt.Errorf("content type is empty")
	}
	for _, item := range c.Overrides {
		existing, err := NormalizePartName(item.PartName)
		if err != nil || existing != normalized {
			continue
		}
		if item.ContentType != contentType {
			return fmt.Errorf("part %q already has content type %q", normalized, item.ContentType)
		}
		return nil
	}
	c.Overrides = append(c.Overrides, OverrideContentType{PartName: normalized, ContentType: contentType})
	return nil
}

func (c *ContentTypes) RemoveOverride(partName string) bool {
	normalized, err := NormalizePartName(partName)
	if err != nil {
		return false
	}
	for index, item := range c.Overrides {
		existing, err := NormalizePartName(item.PartName)
		if err == nil && existing == normalized {
			c.Overrides = append(c.Overrides[:index], c.Overrides[index+1:]...)
			return true
		}
	}
	return false
}
