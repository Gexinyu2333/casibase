// Copyright 2026 The OpenAgent Authors. All Rights Reserved.
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

const (
	ContentTypeRelationships = "application/vnd.openxmlformats-package.relationships+xml"
	ContentTypeXML           = "application/xml"

	ContentTypePresentation  = "application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"
	ContentTypeSlide         = "application/vnd.openxmlformats-officedocument.presentationml.slide+xml"
	ContentTypeSlideLayout   = "application/vnd.openxmlformats-officedocument.presentationml.slideLayout+xml"
	ContentTypeSlideMaster   = "application/vnd.openxmlformats-officedocument.presentationml.slideMaster+xml"
	ContentTypeNotesSlide    = "application/vnd.openxmlformats-officedocument.presentationml.notesSlide+xml"
	ContentTypeNotesMaster   = "application/vnd.openxmlformats-officedocument.presentationml.notesMaster+xml"
	ContentTypeChart         = "application/vnd.openxmlformats-officedocument.drawingml.chart+xml"
	ContentTypeEmbeddedXLSX  = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	ContentTypeTheme         = "application/vnd.openxmlformats-officedocument.theme+xml"
	ContentTypeThemeOverride = "application/vnd.openxmlformats-officedocument.themeOverride+xml"
	ContentTypeImagePNG      = "image/png"
	ContentTypeImageJPEG     = "image/jpeg"
	ContentTypeImageGIF      = "image/gif"
	ContentTypeImageSVG      = "image/svg+xml"

	RelationshipTypeOfficeDocument  = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument"
	RelationshipTypeSlide           = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide"
	RelationshipTypeSlideLayout     = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout"
	RelationshipTypeSlideMaster     = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster"
	RelationshipTypeNotesSlide      = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/notesSlide"
	RelationshipTypeNotesMaster     = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/notesMaster"
	RelationshipTypeChart           = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/chart"
	RelationshipTypeEmbeddedPackage = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/package"
	RelationshipTypeTheme           = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme"
	RelationshipTypeImage           = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/image"
)

const (
	contentTypesPart = "[Content_Types].xml"
	rootRelsPart     = "_rels/.rels"

	contentTypesNamespace  = "http://schemas.openxmlformats.org/package/2006/content-types"
	relationshipsNamespace = "http://schemas.openxmlformats.org/package/2006/relationships"
)
