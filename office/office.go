// Copyright 2026 The OpenAgent Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0

package office

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const encryptedZipFlag = 1

type Limits struct {
	MaxArchiveSize      int64
	MaxPartSize         int64
	MaxUncompressedSize int64
	MaxParts            int
}

func DefaultLimits() Limits {
	return Limits{
		MaxArchiveSize:      128 << 20,
		MaxPartSize:         128 << 20,
		MaxUncompressedSize: 512 << 20,
		MaxParts:            20_000,
	}
}

type part struct {
	data   []byte
	header zip.FileHeader
}

type Package struct {
	parts  map[string]*part
	order  []string
	limits Limits
}

func OpenFile(path string, limits Limits) (*Package, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("OOXML package path is a directory")
	}
	limits = normalizedLimits(limits)
	if info.Size() > limits.MaxArchiveSize {
		return nil, fmt.Errorf("OOXML package exceeds archive size limit")
	}
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open OOXML ZIP package: %w", err)
	}
	defer reader.Close()
	return loadPackage(reader.File, info.Size(), limits)
}

func OpenBytes(data []byte, limits Limits) (*Package, error) {
	limits = normalizedLimits(limits)
	if int64(len(data)) > limits.MaxArchiveSize {
		return nil, fmt.Errorf("OOXML package exceeds archive size limit")
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open OOXML ZIP package: %w", err)
	}
	return loadPackage(reader.File, int64(len(data)), limits)
}

func normalizedLimits(limits Limits) Limits {
	defaults := DefaultLimits()
	if limits.MaxArchiveSize <= 0 {
		limits.MaxArchiveSize = defaults.MaxArchiveSize
	}
	if limits.MaxPartSize <= 0 {
		limits.MaxPartSize = defaults.MaxPartSize
	}
	if limits.MaxUncompressedSize <= 0 {
		limits.MaxUncompressedSize = defaults.MaxUncompressedSize
	}
	if limits.MaxParts <= 0 {
		limits.MaxParts = defaults.MaxParts
	}
	return limits
}

func loadPackage(files []*zip.File, archiveSize int64, limits Limits) (*Package, error) {
	if archiveSize > limits.MaxArchiveSize {
		return nil, fmt.Errorf("OOXML package exceeds archive size limit")
	}
	p := &Package{
		parts:  make(map[string]*part, len(files)),
		order:  make([]string, 0, len(files)),
		limits: limits,
	}
	caseNames := make(map[string]string, len(files))
	var total int64
	for _, file := range files {
		if file.FileInfo().IsDir() {
			directory := strings.TrimSuffix(file.Name, "/")
			if _, err := NormalizePartName(directory); err != nil {
				return nil, fmt.Errorf("invalid ZIP directory name %q", file.Name)
			}
			continue
		}
		if file.Flags&encryptedZipFlag != 0 {
			return nil, fmt.Errorf("encrypted ZIP entries are not supported: %s", file.Name)
		}
		name, err := NormalizePartName(file.Name)
		if err != nil || name != file.Name {
			return nil, fmt.Errorf("invalid ZIP part name %q", file.Name)
		}
		folded := strings.ToLower(name)
		if existing, ok := caseNames[folded]; ok {
			if existing == name {
				return nil, fmt.Errorf("duplicate OOXML part %q", name)
			}
			return nil, fmt.Errorf("case-conflicting OOXML parts %q and %q", existing, name)
		}
		if len(p.order) >= limits.MaxParts {
			return nil, fmt.Errorf("OOXML package exceeds part count limit")
		}
		if file.UncompressedSize64 > uint64(limits.MaxPartSize) {
			return nil, fmt.Errorf("OOXML part exceeds size limit: %s", name)
		}
		if file.UncompressedSize64 > uint64(limits.MaxUncompressedSize-total) {
			return nil, fmt.Errorf("OOXML package exceeds uncompressed size limit")
		}

		data, err := readZipPart(file, limits.MaxPartSize)
		if err != nil {
			return nil, fmt.Errorf("read OOXML part %s: %w", name, err)
		}
		total += int64(len(data))
		if total > limits.MaxUncompressedSize {
			return nil, fmt.Errorf("OOXML package exceeds uncompressed size limit")
		}

		header := file.FileHeader
		header.Name = name
		p.parts[name] = &part{data: data, header: header}
		p.order = append(p.order, name)
		caseNames[folded] = name
	}
	return p, nil
}

func readZipPart(file *zip.File, limit int64) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("part exceeds size limit")
	}
	return data, nil
}

func (p *Package) HasPart(name string) bool {
	normalized, err := NormalizePartName(name)
	if err != nil {
		return false
	}
	_, ok := p.parts[normalized]
	return ok
}

func (p *Package) ReadPart(name string) ([]byte, error) {
	normalized, err := NormalizePartName(name)
	if err != nil {
		return nil, err
	}
	item, ok := p.parts[normalized]
	if !ok {
		return nil, fmt.Errorf("OOXML part not found: %s", normalized)
	}
	return bytes.Clone(item.data), nil
}

func (p *Package) SetPart(name string, data []byte) error {
	normalized, err := NormalizePartName(name)
	if err != nil {
		return err
	}
	if int64(len(data)) > p.limits.MaxPartSize {
		return fmt.Errorf("OOXML part exceeds size limit: %s", normalized)
	}
	if current := p.uncompressedSize(); current-int64(p.partSize(normalized))+int64(len(data)) > p.limits.MaxUncompressedSize {
		return fmt.Errorf("OOXML package exceeds uncompressed size limit")
	}
	for existing := range p.parts {
		if existing != normalized && strings.EqualFold(existing, normalized) {
			return fmt.Errorf("OOXML part %q conflicts with %q by case", normalized, existing)
		}
	}
	if item, ok := p.parts[normalized]; ok {
		item.data = bytes.Clone(data)
		return nil
	}
	if len(p.parts) >= p.limits.MaxParts {
		return fmt.Errorf("OOXML package exceeds part count limit")
	}
	header := zip.FileHeader{
		Name:     normalized,
		Method:   zip.Deflate,
		Modified: time.Now().UTC(),
	}
	p.parts[normalized] = &part{data: bytes.Clone(data), header: header}
	p.order = append(p.order, normalized)
	return nil
}

func (p *Package) DeletePart(name string) error {
	normalized, err := NormalizePartName(name)
	if err != nil {
		return err
	}
	if _, ok := p.parts[normalized]; !ok {
		return fmt.Errorf("OOXML part not found: %s", normalized)
	}
	delete(p.parts, normalized)
	for index, existing := range p.order {
		if existing == normalized {
			p.order = append(p.order[:index], p.order[index+1:]...)
			break
		}
	}
	return nil
}

func (p *Package) PartNames() []string {
	return append([]string(nil), p.order...)
}

func (p *Package) Clone() *Package {
	clone := &Package{
		parts:  make(map[string]*part, len(p.parts)),
		order:  append([]string(nil), p.order...),
		limits: p.limits,
	}
	for name, item := range p.parts {
		header := item.header
		header.Extra = bytes.Clone(item.header.Extra)
		clone.parts[name] = &part{
			data:   bytes.Clone(item.data),
			header: header,
		}
	}
	return clone
}

func (p *Package) Bytes() ([]byte, error) {
	var buffer bytes.Buffer
	if err := p.writeTo(&buffer); err != nil {
		return nil, err
	}
	if int64(buffer.Len()) > p.limits.MaxArchiveSize {
		return nil, fmt.Errorf("serialized OOXML package exceeds archive size limit")
	}
	return buffer.Bytes(), nil
}

func (p *Package) WriteFileAtomic(output string) error {
	dir := filepath.Dir(output)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, "."+filepath.Base(output)+"-*.tmp")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	cleanup := func() {
		_ = temp.Close()
		_ = os.Remove(tempName)
	}
	defer cleanup()

	counting := &limitWriter{writer: temp, remaining: p.limits.MaxArchiveSize}
	if err := p.writeTo(counting); err != nil {
		return err
	}
	if err := temp.Sync(); err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := replaceFile(tempName, output); err != nil {
		return err
	}
	return nil
}

func (p *Package) writeTo(writer io.Writer) error {
	archive := zip.NewWriter(writer)
	for _, name := range p.order {
		item, ok := p.parts[name]
		if !ok {
			continue
		}
		header := item.header
		header.Name = name
		header.Flags &^= encryptedZipFlag
		header.CRC32 = 0
		header.CompressedSize = 0
		header.CompressedSize64 = 0
		header.UncompressedSize = 0
		header.UncompressedSize64 = 0
		partWriter, err := archive.CreateHeader(&header)
		if err != nil {
			_ = archive.Close()
			return fmt.Errorf("create OOXML ZIP part %s: %w", name, err)
		}
		if _, err := partWriter.Write(item.data); err != nil {
			_ = archive.Close()
			return fmt.Errorf("write OOXML ZIP part %s: %w", name, err)
		}
	}
	if err := archive.Close(); err != nil {
		return fmt.Errorf("close OOXML ZIP package: %w", err)
	}
	return nil
}

func (p *Package) ValidatePresentation() error {
	required := []string{contentTypesPart, rootRelsPart, "ppt/presentation.xml", "ppt/_rels/presentation.xml.rels"}
	for _, name := range required {
		if !p.HasPart(name) {
			return fmt.Errorf("invalid PPTX package: missing %s", name)
		}
	}
	contentTypes, err := p.ContentTypes()
	if err != nil {
		return err
	}
	if contentType, ok := contentTypes.Lookup("ppt/presentation.xml"); !ok || contentType != ContentTypePresentation {
		return fmt.Errorf("invalid PPTX package: unexpected presentation content type %q", contentType)
	}
	rootRels, err := p.Relationships("")
	if err != nil {
		return err
	}
	for _, rel := range rootRels.Items {
		if rel.Type != RelationshipTypeOfficeDocument || rel.Mode != TargetInternal {
			continue
		}
		target, err := ResolveTarget("", rel.Target)
		if err == nil && target == "ppt/presentation.xml" {
			return nil
		}
	}
	return fmt.Errorf("invalid PPTX package: root officeDocument relationship is missing")
}

func (p *Package) uncompressedSize() int64 {
	var total int64
	for _, item := range p.parts {
		total += int64(len(item.data))
	}
	return total
}

func (p *Package) partSize(name string) int {
	if item, ok := p.parts[name]; ok {
		return len(item.data)
	}
	return 0
}

type limitWriter struct {
	writer    io.Writer
	remaining int64
}

func (w *limitWriter) Write(data []byte) (int, error) {
	if int64(len(data)) > w.remaining {
		return 0, errors.New("serialized OOXML package exceeds archive size limit")
	}
	n, err := w.writer.Write(data)
	w.remaining -= int64(n)
	return n, err
}
