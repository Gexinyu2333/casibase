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
	"path"
	"strings"
)

type PartAllocator struct {
	used map[string]struct{}
}

func NewPartAllocator(p *Package) *PartAllocator {
	used := make(map[string]struct{}, len(p.parts))
	for name := range p.parts {
		used[strings.ToLower(name)] = struct{}{}
	}
	return &PartAllocator{used: used}
}

func (a *PartAllocator) NextNumbered(dir, stem, ext string) string {
	dir = strings.Trim(dir, "/")
	first := path.Join(dir, fmt.Sprintf("%s1%s", stem, ext))
	if _, err := NormalizePartName(first); err != nil {
		return ""
	}
	for number := 1; ; number++ {
		name := path.Join(dir, fmt.Sprintf("%s%d%s", stem, number, ext))
		if a.reserve(name) {
			return name
		}
	}
}

func (a *PartAllocator) NextSibling(source, marker string) string {
	normalized, err := NormalizePartName(source)
	if err != nil {
		return ""
	}
	dir := path.Dir(normalized)
	if dir == "." {
		dir = ""
	}
	ext := path.Ext(normalized)
	stem := strings.TrimSuffix(path.Base(normalized), ext)
	for number := 1; ; number++ {
		name := path.Join(dir, fmt.Sprintf("%s%s%d%s", stem, marker, number, ext))
		if a.reserve(name) {
			return name
		}
	}
}

func (a *PartAllocator) reserve(name string) bool {
	normalized, err := NormalizePartName(name)
	if err != nil {
		return false
	}
	key := strings.ToLower(normalized)
	if _, exists := a.used[key]; exists {
		return false
	}
	a.used[key] = struct{}{}
	return true
}
