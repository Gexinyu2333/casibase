// Copyright 2026 The OpenAgent Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0

package office

import "fmt"

func (p *Package) ReachableParts() (map[string]struct{}, error) {
	if !p.HasPart(rootRelsPart) {
		return nil, fmt.Errorf("OOXML package is missing %s", rootRelsPart)
	}
	root, err := p.Relationships("")
	if err != nil {
		return nil, err
	}
	reachable := make(map[string]struct{})
	queue := make([]string, 0, len(root.Items))
	if err := p.enqueueRelationshipTargets("", root, &queue); err != nil {
		return nil, err
	}
	for len(queue) > 0 {
		partName := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		if _, visited := reachable[partName]; visited {
			continue
		}
		if !p.HasPart(partName) {
			return nil, fmt.Errorf("relationship target part is missing: %s", partName)
		}
		reachable[partName] = struct{}{}
		rels, err := p.Relationships(partName)
		if err != nil {
			return nil, err
		}
		if err := p.enqueueRelationshipTargets(partName, rels, &queue); err != nil {
			return nil, err
		}
	}
	return reachable, nil
}

func (p *Package) enqueueRelationshipTargets(owner string, rels *Relationships, queue *[]string) error {
	for _, rel := range rels.Items {
		if rel.Mode == TargetExternal {
			continue
		}
		target, err := ResolveTarget(owner, rel.Target)
		if err != nil {
			return fmt.Errorf("resolve relationship %s from %q: %w", rel.ID, owner, err)
		}
		if !p.HasPart(target) {
			return fmt.Errorf("relationship %s from %q targets missing part %s", rel.ID, owner, target)
		}
		*queue = append(*queue, target)
	}
	return nil
}

func (p *Package) PruneUnreachable() error {
	reachable, err := p.ReachableParts()
	if err != nil {
		return err
	}
	contentTypes, err := p.ContentTypes()
	if err != nil {
		return err
	}

	keep := map[string]struct{}{
		contentTypesPart: {},
		rootRelsPart:     {},
	}
	for partName := range reachable {
		keep[partName] = struct{}{}
		relsName, err := RelationshipsPart(partName)
		if err != nil {
			return err
		}
		if p.HasPart(relsName) {
			keep[relsName] = struct{}{}
		}
	}

	for index := 0; index < len(contentTypes.Overrides); {
		name, err := NormalizePartName(contentTypes.Overrides[index].PartName)
		if err != nil {
			return err
		}
		if _, ok := reachable[name]; ok {
			index++
			continue
		}
		contentTypes.Overrides = append(contentTypes.Overrides[:index], contentTypes.Overrides[index+1:]...)
	}

	clone := p.Clone()
	for _, name := range clone.PartNames() {
		if _, ok := keep[name]; ok {
			continue
		}
		if err := clone.DeletePart(name); err != nil {
			return err
		}
	}
	if err := clone.SetContentTypes(contentTypes); err != nil {
		return err
	}
	p.parts = clone.parts
	p.order = clone.order
	return nil
}
