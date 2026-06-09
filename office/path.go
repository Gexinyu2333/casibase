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

// NormalizePartName converts an OPC part URI to the package's canonical form:
// forward slashes without a leading slash.
func NormalizePartName(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("OOXML part name is empty")
	}
	if strings.ContainsAny(name, "\\?#") {
		return "", fmt.Errorf("invalid OOXML part name %q", name)
	}
	if strings.HasPrefix(name, "//") {
		return "", fmt.Errorf("invalid OOXML part name %q", name)
	}
	name = strings.TrimPrefix(name, "/")
	if name == "" || strings.HasSuffix(name, "/") {
		return "", fmt.Errorf("invalid OOXML part name %q", name)
	}
	if len(name) >= 2 && name[1] == ':' {
		return "", fmt.Errorf("invalid OOXML part name %q", name)
	}
	for _, segment := range strings.Split(name, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return "", fmt.Errorf("invalid OOXML part name %q", name)
		}
	}
	if cleaned := path.Clean(name); cleaned != name || cleaned == "." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("invalid OOXML part name %q", name)
	}
	return name, nil
}

// ResolveTarget resolves an internal relationship target against its owner.
// An empty owner represents the package root.
func ResolveTarget(ownerPart, target string) (string, error) {
	if target == "" {
		return "", fmt.Errorf("relationship target is empty")
	}
	if strings.ContainsAny(target, "\\?#") {
		return "", fmt.Errorf("invalid internal relationship target %q", target)
	}
	if len(target) >= 2 && target[1] == ':' {
		return "", fmt.Errorf("invalid internal relationship target %q", target)
	}

	base := ""
	if ownerPart != "" {
		owner, err := NormalizePartName(ownerPart)
		if err != nil {
			return "", err
		}
		base = path.Dir(owner)
		if base == "." {
			base = ""
		}
	}

	var resolved string
	if strings.HasPrefix(target, "/") {
		resolved = strings.TrimPrefix(target, "/")
	} else {
		resolved = path.Join(base, target)
	}
	if resolved == "." || resolved == "" || strings.HasPrefix(resolved, "../") {
		return "", fmt.Errorf("relationship target escapes package root: %q", target)
	}
	return NormalizePartName(resolved)
}

// RelativeTarget returns a relationship target from ownerPart to targetPart.
func RelativeTarget(ownerPart, targetPart string) (string, error) {
	target, err := NormalizePartName(targetPart)
	if err != nil {
		return "", err
	}
	if ownerPart == "" {
		return target, nil
	}
	owner, err := NormalizePartName(ownerPart)
	if err != nil {
		return "", err
	}
	base := path.Dir(owner)
	if base == "." {
		base = ""
	}
	return relativeSlashPath(base, target), nil
}

func relativeSlashPath(base, target string) string {
	baseParts := splitPartPath(base)
	targetParts := splitPartPath(target)
	common := 0
	for common < len(baseParts) && common < len(targetParts) && baseParts[common] == targetParts[common] {
		common++
	}
	parts := make([]string, 0, len(baseParts)-common+len(targetParts)-common)
	for range baseParts[common:] {
		parts = append(parts, "..")
	}
	parts = append(parts, targetParts[common:]...)
	if len(parts) == 0 {
		return path.Base(target)
	}
	return strings.Join(parts, "/")
}

func splitPartPath(value string) []string {
	if value == "" || value == "." {
		return nil
	}
	return strings.Split(value, "/")
}

// RelationshipsPart returns the relationship part belonging to ownerPart.
func RelationshipsPart(ownerPart string) (string, error) {
	if ownerPart == "" {
		return rootRelsPart, nil
	}
	owner, err := NormalizePartName(ownerPart)
	if err != nil {
		return "", err
	}
	dir, base := path.Dir(owner), path.Base(owner)
	if dir == "." {
		return path.Join("_rels", base+".rels"), nil
	}
	return path.Join(dir, "_rels", base+".rels"), nil
}

// OwnerPartFromRelationships maps a relationship part back to its owner.
func OwnerPartFromRelationships(name string) (string, error) {
	normalized, err := NormalizePartName(name)
	if err != nil {
		return "", err
	}
	if normalized == rootRelsPart {
		return "", nil
	}
	dir, base := path.Dir(normalized), path.Base(normalized)
	if !strings.HasSuffix(base, ".rels") || path.Base(dir) != "_rels" {
		return "", fmt.Errorf("%q is not an OOXML relationship part", name)
	}
	ownerBase := strings.TrimSuffix(base, ".rels")
	parent := path.Dir(dir)
	if parent == "." {
		return NormalizePartName(ownerBase)
	}
	return NormalizePartName(path.Join(parent, ownerBase))
}
