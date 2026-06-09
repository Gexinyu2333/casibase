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
	"path"
	"strings"
)

var sharedRelationshipTypes = map[string]bool{
	RelationshipTypeSlideLayout: true,
	RelationshipTypeSlideMaster: true,
	RelationshipTypeNotesMaster: true,
	RelationshipTypeTheme:       true,
	"http://schemas.openxmlformats.org/officeDocument/2006/relationships/presProps":   true,
	"http://schemas.openxmlformats.org/officeDocument/2006/relationships/viewProps":   true,
	"http://schemas.openxmlformats.org/officeDocument/2006/relationships/tableStyles": true,
}

func cloneSlidePrivateParts(pkg *Package, rels *Relationships, newSlidePart string, types *ContentTypes, allocator *PartAllocator) error {
	return clonePrivateRelationships(pkg, rels, newSlidePart, types, allocator, map[string]string{})
}

func clonePrivateRelationships(pkg *Package, rels *Relationships, owner string, types *ContentTypes, allocator *PartAllocator, cloned map[string]string) error {
	for index := range rels.Items {
		rel := &rels.Items[index]
		if rel.Mode == TargetExternal || sharedRelationshipTypes[rel.Type] ||
			rel.Type == RelationshipTypeChart || rel.Type == RelationshipTypeNotesSlide || rel.Type == RelationshipTypeSlide {
			continue
		}
		sourcePart, err := ResolveTarget(owner, rel.Target)
		if err != nil || !pkg.HasPart(sourcePart) {
			continue
		}
		contentType, explicit := explicitContentType(types, sourcePart)
		if !explicit {
			continue
		}
		newPart := cloned[sourcePart]
		if newPart == "" {
			newPart = allocator.NextSibling(sourcePart, "_tf")
			data, err := pkg.ReadPart(sourcePart)
			if err != nil {
				return err
			}
			if err := pkg.SetPart(newPart, data); err != nil {
				return err
			}
			if err := types.EnsureOverride(newPart, contentType); err != nil {
				return err
			}
			cloned[sourcePart] = newPart
			sourceRels, err := pkg.Relationships(sourcePart)
			if err != nil {
				return err
			}
			if len(sourceRels.Items) != 0 {
				subRels := &Relationships{Items: append([]Relationship(nil), sourceRels.Items...)}
				if err := clonePrivateRelationships(pkg, subRels, newPart, types, allocator, cloned); err != nil {
					return err
				}
				if err := pkg.SetRelationships(newPart, subRels); err != nil {
					return err
				}
			}
		}
		target, err := RelativeTarget(owner, newPart)
		if err != nil {
			return err
		}
		rel.Target = target
	}
	return nil
}

func explicitContentType(types *ContentTypes, partName string) (string, bool) {
	normalized, err := NormalizePartName(partName)
	if err != nil {
		return "", false
	}
	for _, override := range types.Overrides {
		item, err := NormalizePartName(override.PartName)
		if err == nil && item == normalized {
			return override.ContentType, true
		}
	}
	return "", false
}

func nextNumberedPart(pkg *Package, directory, prefix, suffix string) int {
	maxNumber := 0
	for _, name := range pkg.PartNames() {
		maxNumber = max(maxNumber, partBaseNumber(name, directory, prefix, suffix))
	}
	return maxNumber + 1
}

func nextSiblingName(pkg *Package, source, marker string) string {
	directory := path.Dir(source)
	extension := path.Ext(source)
	stem := strings.TrimSuffix(path.Base(source), extension)
	for index := 1; ; index++ {
		candidate := path.Join(directory, fmt.Sprintf("%s%s%d%s", stem, marker, index, extension))
		if !pkg.HasPart(candidate) {
			return candidate
		}
	}
}
