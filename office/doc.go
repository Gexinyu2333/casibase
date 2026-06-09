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

// Package ooxmlpkg provides bounded, in-memory access to the OPC ZIP package
// used by PowerPoint files. It manages parts, relationship parts, content
// types, safe part URI resolution, reachability pruning, and atomic writes.
// It also implements the native PPTX template analyze, scaffold, check, and
// fill pipeline used by the Casibase PowerPoint tools.
//
// The package intentionally does not model PresentationML elements. Callers
// that edit slide, chart, or workbook XML should preserve unknown OOXML
// content and write the resulting bytes back with Package.SetPart.
package office
