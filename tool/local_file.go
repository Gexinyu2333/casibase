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

package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/the-open-agent/openagent/txt"
)

const (
	localFileDefaultPreviewChars = 1200
	localFileMaxPreviewChars     = 20000
	localFileDefaultReadLimit    = 12000
	localFileMaxReadLimit        = 100000
)

// LocalFileTool is the Tool Type "local_file".
type LocalFileTool struct {
	lang string
}

func (p *LocalFileTool) BuiltinTools() []BuiltinTool {
	return []BuiltinTool{
		&localSpecialDirsBuiltin{},
		&localDocumentsScanBuiltin{lang: p.lang},
		&localDocumentReadBuiltin{lang: p.lang},
		&localTextWriteBuiltin{},
		&localFileMoveBuiltin{},
	}
}

type localDocumentInfo struct {
	Path         string `json:"path"`
	Name         string `json:"name"`
	Extension    string `json:"extension"`
	Size         int64  `json:"size"`
	ModifiedTime string `json:"modifiedTime"`
	TextLength   int    `json:"textLength,omitempty"`
	Preview      string `json:"preview,omitempty"`
	Truncated    bool   `json:"truncated,omitempty"`
	Error        string `json:"error,omitempty"`
}

type localSpecialDirInfo struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

type localSpecialDirs struct {
	Desktop   localSpecialDirInfo `json:"desktop"`
	Documents localSpecialDirInfo `json:"documents"`
	Downloads localSpecialDirInfo `json:"downloads"`
}

type localSpecialDirsResult struct {
	OS          string           `json:"os"`
	Username    string           `json:"username"`
	Home        string           `json:"home"`
	Directories localSpecialDirs `json:"directories"`
}

type localDocumentsScanResult struct {
	Root  string              `json:"root"`
	Count int                 `json:"count"`
	Files []localDocumentInfo `json:"files"`
}

type localDocumentReadResult struct {
	Path       string `json:"path"`
	Extension  string `json:"extension"`
	Offset     int    `json:"offset"`
	Limit      int    `json:"limit"`
	TextLength int    `json:"textLength"`
	Text       string `json:"text"`
	Truncated  bool   `json:"truncated"`
}

type localTextWriteResult struct {
	Path      string `json:"path"`
	ByteSize  int    `json:"byteSize"`
	Overwrote bool   `json:"overwrote"`
}

type localFileMoveResult struct {
	Source    string `json:"source"`
	Target    string `json:"target"`
	Overwrote bool   `json:"overwrote"`
}

func localFileText(text string) *protocol.CallToolResult {
	return &protocol.CallToolResult{
		IsError: false,
		Content: []protocol.Content{
			&protocol.TextContent{Type: "text", Text: text},
		},
	}
}

func localFileError(text string) *protocol.CallToolResult {
	return &protocol.CallToolResult{
		IsError: true,
		Content: []protocol.Content{
			&protocol.TextContent{Type: "text", Text: text},
		},
	}
}

func localFileJSON(v interface{}) *protocol.CallToolResult {
	bs, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return localFileError(err.Error())
	}
	return localFileText(string(bs))
}

func localFileStringArg(arguments map[string]interface{}, key string) string {
	value, _ := arguments[key].(string)
	return strings.TrimSpace(value)
}

func localFileBoolArg(arguments map[string]interface{}, key string) bool {
	value, _ := arguments[key].(bool)
	return value
}

func localFileIntArg(arguments map[string]interface{}, key string, defaultValue int) int {
	value, ok := arguments[key]
	if !ok {
		return defaultValue
	}
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		n, err := v.Int64()
		if err == nil {
			return int(n)
		}
	}
	return defaultValue
}

func localFileRequireAbsolutePath(path, field string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("missing required parameter: %s", field)
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("%s must be an absolute path for the current operating system: %s", field, path)
	}
	return nil
}

func localFileSupportedExt(path string) (string, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	for _, supportedExt := range txt.GetSupportedFileTypes() {
		if ext == supportedExt {
			return ext, true
		}
	}
	return ext, false
}

func localFileClamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func localFileSliceText(text string, offset, limit int) (string, bool) {
	runes := []rune(text)
	if offset > len(runes) {
		return "", false
	}
	end := offset + limit
	truncated := false
	if end < len(runes) {
		truncated = true
	} else {
		end = len(runes)
	}
	return string(runes[offset:end]), truncated
}

func localFileReadDocument(path, ext, lang string) (string, error) {
	return txt.GetParsedTextFromUrl(path, ext, lang)
}

func localFileDirectory(path string) localSpecialDirInfo {
	info, err := os.Stat(path)
	return localSpecialDirInfo{
		Path:   path,
		Exists: err == nil && info.IsDir(),
	}
}

func localFileDesktopPath(home string) string {
	desktop := filepath.Join(home, "Desktop")
	if runtime.GOOS == "windows" {
		oneDriveDesktop := filepath.Join(home, "OneDrive", "Desktop")
		if localFileDirectory(oneDriveDesktop).Exists {
			return oneDriveDesktop
		}
	}
	return desktop
}

type localSpecialDirsBuiltin struct{}

func (b *localSpecialDirsBuiltin) GetName() string {
	return "local_special_dirs"
}

func (b *localSpecialDirsBuiltin) GetDescription() string {
	return `Return common local directories for the operating system user running the OpenAgent backend process.
- No required parameters.
- Returns os, username, home, and Desktop/Documents/Downloads paths with existence flags.
- Use this before scanning when the user says "my Desktop", "my Documents", or "Downloads" without an absolute path.`
}

func (b *localSpecialDirsBuiltin) GetInputSchema() interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (b *localSpecialDirsBuiltin) Execute(_ context.Context, _ map[string]interface{}) (*protocol.CallToolResult, error) {
	currentUser, err := user.Current()
	if err != nil {
		return localFileError(fmt.Sprintf("failed to read current process user: %s", err.Error())), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return localFileError(fmt.Sprintf("failed to read current process user home: %s", err.Error())), nil
	}
	if !filepath.IsAbs(home) {
		return localFileError(fmt.Sprintf("current process user home is not absolute: %s", home)), nil
	}

	return localFileJSON(localSpecialDirsResult{
		OS:       runtime.GOOS,
		Username: currentUser.Username,
		Home:     home,
		Directories: localSpecialDirs{
			Desktop:   localFileDirectory(localFileDesktopPath(home)),
			Documents: localFileDirectory(filepath.Join(home, "Documents")),
			Downloads: localFileDirectory(filepath.Join(home, "Downloads")),
		},
	}), nil
}

type localDocumentsScanBuiltin struct {
	lang string
}

func (b *localDocumentsScanBuiltin) GetName() string {
	return "local_documents_scan"
}

func (b *localDocumentsScanBuiltin) GetDescription() string {
	return `Scan a local directory for supported documents and return a JSON manifest with metadata, text previews, and parse errors.
- root (required): absolute directory path for the current operating system.
- preview_chars: maximum text preview characters per file (default 1200, max 20000).
- max_files: optional maximum number of supported files to return.`
}

func (b *localDocumentsScanBuiltin) GetInputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"root": map[string]interface{}{
				"type":        "string",
				"description": "Absolute directory path to scan.",
			},
			"preview_chars": map[string]interface{}{
				"type":        "number",
				"description": "Maximum text preview characters per file (default 1200, max 20000).",
			},
			"max_files": map[string]interface{}{
				"type":        "number",
				"description": "Optional maximum number of supported files to return.",
			},
		},
		"required": []string{"root"},
	}
}

func (b *localDocumentsScanBuiltin) Execute(_ context.Context, arguments map[string]interface{}) (*protocol.CallToolResult, error) {
	root := localFileStringArg(arguments, "root")
	if err := localFileRequireAbsolutePath(root, "root"); err != nil {
		return localFileError(err.Error()), nil
	}

	info, err := os.Stat(root)
	if err != nil {
		return localFileError(fmt.Sprintf("failed to access root: %s", err.Error())), nil
	}
	if !info.IsDir() {
		return localFileError("root must be a directory"), nil
	}

	previewChars := localFileIntArg(arguments, "preview_chars", localFileDefaultPreviewChars)
	previewChars = localFileClamp(previewChars, 0, localFileMaxPreviewChars)
	maxFiles := localFileIntArg(arguments, "max_files", 0)

	var files []localDocumentInfo
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		ext, ok := localFileSupportedExt(path)
		if !ok {
			return nil
		}
		if maxFiles > 0 && len(files) >= maxFiles {
			return filepath.SkipAll
		}

		item := localDocumentInfo{
			Path:      path,
			Name:      d.Name(),
			Extension: ext,
		}
		if fileInfo, statErr := d.Info(); statErr == nil {
			item.Size = fileInfo.Size()
			item.ModifiedTime = fileInfo.ModTime().Format("2006-01-02 15:04:05")
		}

		text, readErr := localFileReadDocument(path, ext, b.lang)
		if readErr != nil {
			item.Error = readErr.Error()
		} else {
			item.TextLength = len([]rune(text))
			item.Preview, item.Truncated = localFileSliceText(text, 0, previewChars)
		}
		files = append(files, item)
		return nil
	})
	if walkErr != nil {
		return localFileError(fmt.Sprintf("failed to scan root: %s", walkErr.Error())), nil
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return localFileJSON(localDocumentsScanResult{
		Root:  root,
		Count: len(files),
		Files: files,
	}), nil
}

type localDocumentReadBuiltin struct {
	lang string
}

func (b *localDocumentReadBuiltin) GetName() string {
	return "local_document_read"
}

func (b *localDocumentReadBuiltin) GetDescription() string {
	return `Read text from one supported local document.
- path (required): absolute file path for the current operating system.
- offset: character offset in extracted text (default 0).
- limit: maximum characters to return (default 12000, max 100000).`
}

func (b *localDocumentReadBuiltin) GetInputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Absolute path to a supported document.",
			},
			"offset": map[string]interface{}{
				"type":        "number",
				"description": "Character offset in extracted text (default 0).",
			},
			"limit": map[string]interface{}{
				"type":        "number",
				"description": "Maximum characters to return (default 12000, max 100000).",
			},
		},
		"required": []string{"path"},
	}
}

func (b *localDocumentReadBuiltin) Execute(_ context.Context, arguments map[string]interface{}) (*protocol.CallToolResult, error) {
	path := localFileStringArg(arguments, "path")
	if err := localFileRequireAbsolutePath(path, "path"); err != nil {
		return localFileError(err.Error()), nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return localFileError(fmt.Sprintf("failed to access path: %s", err.Error())), nil
	}
	if info.IsDir() {
		return localFileError("path must be a file"), nil
	}
	ext, ok := localFileSupportedExt(path)
	if !ok {
		return localFileError(fmt.Sprintf("unsupported file type: %s", ext)), nil
	}

	offset := localFileIntArg(arguments, "offset", 0)
	if offset < 0 {
		offset = 0
	}
	limit := localFileIntArg(arguments, "limit", localFileDefaultReadLimit)
	limit = localFileClamp(limit, 0, localFileMaxReadLimit)

	text, err := localFileReadDocument(path, ext, b.lang)
	if err != nil {
		return localFileError(fmt.Sprintf("failed to read document: %s", err.Error())), nil
	}
	section, truncated := localFileSliceText(text, offset, limit)

	return localFileJSON(localDocumentReadResult{
		Path:       path,
		Extension:  ext,
		Offset:     offset,
		Limit:      limit,
		TextLength: len([]rune(text)),
		Text:       section,
		Truncated:  truncated,
	}), nil
}

type localTextWriteBuiltin struct{}

func (b *localTextWriteBuiltin) GetName() string {
	return "local_text_write"
}

func (b *localTextWriteBuiltin) GetDescription() string {
	return `Write Markdown or plain text content to a local file.
- path (required): absolute output file path for the current operating system.
- content (required): text content to write.
- overwrite: set true to replace an existing file (default false).`
}

func (b *localTextWriteBuiltin) GetInputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Absolute output file path.",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Markdown or plain text content to write.",
			},
			"overwrite": map[string]interface{}{
				"type":        "boolean",
				"description": "Replace an existing file when true (default false).",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (b *localTextWriteBuiltin) Execute(_ context.Context, arguments map[string]interface{}) (*protocol.CallToolResult, error) {
	path := localFileStringArg(arguments, "path")
	if err := localFileRequireAbsolutePath(path, "path"); err != nil {
		return localFileError(err.Error()), nil
	}
	content, ok := arguments["content"].(string)
	if !ok {
		return localFileError("missing required parameter: content"), nil
	}
	overwrite := localFileBoolArg(arguments, "overwrite")

	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return localFileError("path must be a file"), nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return localFileError(fmt.Sprintf("failed to create parent directory: %s", err.Error())), nil
	}

	overwrote := false
	if overwrite {
		if _, err := os.Stat(path); err == nil {
			overwrote = true
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return localFileError(fmt.Sprintf("failed to write file: %s", err.Error())), nil
		}
	} else {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			if os.IsExist(err) {
				return localFileError("target file already exists; set overwrite to true to replace it"), nil
			}
			return localFileError(fmt.Sprintf("failed to create file: %s", err.Error())), nil
		}
		if _, err := file.WriteString(content); err != nil {
			_ = file.Close()
			return localFileError(fmt.Sprintf("failed to write file: %s", err.Error())), nil
		}
		if err := file.Close(); err != nil {
			return localFileError(fmt.Sprintf("failed to close file: %s", err.Error())), nil
		}
	}

	return localFileJSON(localTextWriteResult{
		Path:      path,
		ByteSize:  len([]byte(content)),
		Overwrote: overwrote,
	}), nil
}

type localFileMoveBuiltin struct{}

func (b *localFileMoveBuiltin) GetName() string {
	return "local_file_move"
}

func (b *localFileMoveBuiltin) GetDescription() string {
	return `Move one local file after explicit user confirmation.
- source (required): absolute source file path for the current operating system.
- target (required): absolute target file path for the current operating system.
- confirmed (required): must be true; call only after the user explicitly confirms the move plan.
- overwrite: set true to replace an existing target file (default false).`
}

func (b *localFileMoveBuiltin) GetInputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"source": map[string]interface{}{
				"type":        "string",
				"description": "Absolute source file path.",
			},
			"target": map[string]interface{}{
				"type":        "string",
				"description": "Absolute target file path.",
			},
			"confirmed": map[string]interface{}{
				"type":        "boolean",
				"description": "Must be true after explicit user confirmation.",
			},
			"overwrite": map[string]interface{}{
				"type":        "boolean",
				"description": "Replace an existing target file when true (default false).",
			},
		},
		"required": []string{"source", "target", "confirmed"},
	}
}

func (b *localFileMoveBuiltin) Execute(_ context.Context, arguments map[string]interface{}) (*protocol.CallToolResult, error) {
	if !localFileBoolArg(arguments, "confirmed") {
		return localFileError("confirmed must be true before moving files"), nil
	}
	source := localFileStringArg(arguments, "source")
	target := localFileStringArg(arguments, "target")
	if err := localFileRequireAbsolutePath(source, "source"); err != nil {
		return localFileError(err.Error()), nil
	}
	if err := localFileRequireAbsolutePath(target, "target"); err != nil {
		return localFileError(err.Error()), nil
	}
	if filepath.Clean(source) == filepath.Clean(target) {
		return localFileError("source and target must be different files"), nil
	}

	sourceInfo, err := os.Stat(source)
	if err != nil {
		return localFileError(fmt.Sprintf("failed to access source: %s", err.Error())), nil
	}
	if sourceInfo.IsDir() {
		return localFileError("source must be a file"), nil
	}
	if info, err := os.Stat(target); err == nil && info.IsDir() {
		return localFileError("target must be a file"), nil
	}

	overwrite := localFileBoolArg(arguments, "overwrite")
	targetExists := false
	if _, err := os.Stat(target); err == nil {
		targetExists = true
		if !overwrite {
			return localFileError("target file already exists; set overwrite to true to replace it"), nil
		}
	} else if !os.IsNotExist(err) {
		return localFileError(fmt.Sprintf("failed to access target: %s", err.Error())), nil
	}

	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return localFileError(fmt.Sprintf("failed to create target directory: %s", err.Error())), nil
	}
	if targetExists {
		if err := os.Remove(target); err != nil {
			return localFileError(fmt.Sprintf("failed to remove existing target: %s", err.Error())), nil
		}
	}
	if err := os.Rename(source, target); err != nil {
		return localFileError(fmt.Sprintf("failed to move file: %s", err.Error())), nil
	}

	return localFileJSON(localFileMoveResult{
		Source:    source,
		Target:    target,
		Overwrote: targetExists,
	}), nil
}
