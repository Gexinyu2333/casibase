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
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
)

func pictureContainers(root *xmlNode) []*xmlNode {
	return root.descendants(nsPresentation, "pic")
}

func analyzeImages(
	pkg *Package,
	slide *xmlNode,
	ref slideRef,
	objectByID map[string]*SlideObject,
) ([]ImageInfo, error) {
	rels, err := pkg.Relationships(ref.PartName)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]Relationship, len(rels.Items))
	for _, rel := range rels.Items {
		byID[rel.ID] = rel
	}

	containers := pictureContainers(slide)
	result := make([]ImageInfo, 0, len(containers))

	for order, container := range containers {
		shapeID, shapeName := shapeIdentity(container, order+1)
		description := ""
		if cNvPr := container.firstDescendant(nsPresentation, "cNvPr"); cNvPr != nil {
			description = cNvPr.attr("", "descr")
		}

		blip := firstContainerBlip(container)
		if blip == nil {
			continue
		}
		embedID := blip.attr(nsOfficeRels, "embed")
		if embedID == "" {
			continue
		}
		rel, ok := byID[embedID]
		if !ok || rel.Type != RelationshipTypeImage || rel.Mode != TargetInternal {
			continue
		}
		partName, resolveErr := ResolveTarget(ref.PartName, rel.Target)
		if resolveErr != nil || !pkg.HasPart(partName) {
			continue
		}

		geometry := Geometry{}
		if obj := objectByID[shapeID]; obj != nil {
			geometry = obj.Geometry
		}

		result = append(result, ImageInfo{
			ImageID:     fmt.Sprintf("s%02d_img%s", ref.Index, shapeID),
			ShapeID:     shapeID,
			ShapeName:   shapeName,
			Description: description,
			Geometry:    geometry,
		})
	}
	return result, nil
}

func firstContainerBlip(pic *xmlNode) *xmlNode {
	blipFill := pic.child(nsPresentation, "blipFill")
	if blipFill == nil {
		return nil
	}
	return blipFill.child(nsDrawingML, "blip")
}

type imagePayload struct {
	Data        []byte
	Extension   string
	ContentType string
}

func readImagePayload(path string, limit int64) (imagePayload, error) {
	if path == "" {
		return imagePayload{}, fmt.Errorf("image path is empty")
	}
	info, err := os.Stat(path)
	if err != nil {
		return imagePayload{}, fmt.Errorf("image file not found: %w", err)
	}
	if info.IsDir() {
		return imagePayload{}, fmt.Errorf("image path is a directory")
	}
	if info.Size() > limit {
		return imagePayload{}, fmt.Errorf("image file exceeds the %d MB size limit", limit>>20)
	}

	file, err := os.Open(path)
	if err != nil {
		return imagePayload{}, fmt.Errorf("cannot read image file: %w", err)
	}
	defer file.Close()

	limited := io.LimitReader(file, limit+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return imagePayload{}, fmt.Errorf("cannot read image file: %w", err)
	}
	if int64(len(data)) > limit {
		return imagePayload{}, fmt.Errorf("image file exceeds the %d MB size limit", limit>>20)
	}
	if len(data) == 0 {
		return imagePayload{}, fmt.Errorf("image file is empty")
	}

	// image.DecodeConfig is used only to detect the format; dimensions are intentionally
	// ignored because image replacement preserves the template frame's geometry as-is.
	_, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return imagePayload{}, fmt.Errorf("cannot decode image: %w", err)
	}

	switch format {
	case "png":
		return imagePayload{Data: data, Extension: ".png", ContentType: ContentTypeImagePNG}, nil
	case "jpeg":
		return imagePayload{Data: data, Extension: ".jpeg", ContentType: ContentTypeImageJPEG}, nil
	default:
		return imagePayload{}, fmt.Errorf("unsupported image format %q; only PNG and JPEG are supported", format)
	}
}

type imageShape struct {
	Node *xmlNode
	Blip *xmlNode
}

func applyImageEdits(
	pkg *Package,
	slide *xmlNode,
	rels *Relationships,
	types *ContentTypes,
	allocator *PartAllocator,
	sourceSlide int,
	newSlidePart string,
	edits []ImageEdit,
) error {
	if len(edits) == 0 {
		return nil
	}

	targets := make(map[string]imageShape)
	for _, pic := range pictureContainers(slide) {
		blip := firstContainerBlip(pic)
		if blip == nil {
			continue
		}
		embedID := blip.attr(nsOfficeRels, "embed")
		if embedID == "" {
			continue
		}
		shapeID, _ := shapeIdentity(pic, 0)
		imageID := fmt.Sprintf("s%02d_img%s", sourceSlide, shapeID)
		targets[imageID] = imageShape{Node: pic, Blip: blip}
	}

	for _, edit := range edits {
		target, ok := targets[edit.ImageID]
		if !ok {
			return fmt.Errorf("image edit %s on slide %d: image target not found", edit.ImageID, sourceSlide)
		}

		payload, err := readImagePayload(edit.ImagePath, pkg.limits.MaxPartSize)
		if err != nil {
			return fmt.Errorf("image edit %s on slide %d: %s", edit.ImageID, sourceSlide, err.Error())
		}

		mediaPart := allocator.NextNumbered("ppt/media", "templateFillImage", payload.Extension)

		if err := pkg.SetPart(mediaPart, payload.Data); err != nil {
			return err
		}
		if err := types.EnsureOverride(mediaPart, payload.ContentType); err != nil {
			return err
		}

		relID := rels.NextID()
		targetName, err := RelativeTarget(newSlidePart, mediaPart)
		if err != nil {
			return err
		}
		if err := rels.Add(Relationship{
			ID:     relID,
			Type:   RelationshipTypeImage,
			Target: targetName,
			Mode:   TargetInternal,
		}); err != nil {
			return err
		}

		target.Blip.setAttr(nsOfficeRels, "embed", relID)
	}

	return nil
}
