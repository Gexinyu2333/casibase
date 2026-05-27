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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/the-open-agent/openagent/embedsupport"
)

type pptxGenerateBuiltin struct{}

type pptxGenerateArgs struct {
	Path       string          `json:"path"`
	ScriptPath string          `json:"script_path"`
	AssetsDir  string          `json:"assets_dir,omitempty"`
	Data       json.RawMessage `json:"data,omitempty"`
}

type pptxGenerateWorkerResult struct {
	OK         bool   `json:"ok"`
	Path       string `json:"path"`
	SlideCount int    `json:"slideCount"`
	Mode       string `json:"mode"`
	Error      string `json:"error"`
}

type pptxWorkerCandidate struct {
	path               string
	requireNodeModules bool
}

func (t *pptxGenerateBuiltin) GetName() string { return "pptx_generate" }

func (t *pptxGenerateBuiltin) GetDescription() string {
	return `Generate a PowerPoint (.pptx) deck by running a trusted local PptxGenJS module.
- path (required): output .pptx path. Absolute paths are used as-is. Relative paths are resolved inside the current user's Documents folder.
- script_path (required): local .mjs file that exports default async function build(pptx, ctx) or a named build function. The script adds slides to the provided PptxGenJS instance; the worker writes the file.
- data (optional): JSON value passed to ctx.data for content or configuration.
- assets_dir (optional): base directory for ctx.resolveAsset() and relative ctx.imageData() paths. Defaults to the script directory.
Use this for designed, editable decks. The script should call PptxGenJS APIs directly instead of generating HTML.`
}

func (t *pptxGenerateBuiltin) GetInputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Output path for the .pptx file.",
			},
			"script_path": map[string]interface{}{
				"type":        "string",
				"description": "Local .mjs file exporting default build(pptx, ctx) or named build(pptx, ctx).",
			},
			"data": map[string]interface{}{
				"type":        "object",
				"description": "Optional JSON value passed through to the PptxGenJS script as ctx.data.",
			},
			"assets_dir": map[string]interface{}{
				"type":        "string",
				"description": "Optional asset base directory for ctx.resolveAsset() and ctx.imageData(). Defaults to the script directory.",
			},
		},
		"required": []string{"path", "script_path"},
	}
}

// Execute validates the requested deck generation job, writes a short-lived
// worker spec file, then runs the local Node/PptxGenJS worker.
func (t *pptxGenerateBuiltin) Execute(ctx context.Context, arguments map[string]interface{}) (*protocol.CallToolResult, error) {
	argBytes, err := json.Marshal(arguments)
	if err != nil {
		return officeToolError(fmt.Sprintf("Failed to parse parameters: %s", err.Error())), nil
	}

	var args pptxGenerateArgs
	if err := json.Unmarshal(argBytes, &args); err != nil {
		return officeToolError(fmt.Sprintf("Failed to parse parameters: %s", err.Error())), nil
	}

	args.Path = strings.TrimSpace(args.Path)
	if args.Path == "" {
		return officeToolError("Missing required parameter: path"), nil
	}

	// The build script is user/agent-authored PptxGenJS code. This tool only
	// validates and runs it; it does not generate the script itself.
	args.ScriptPath = strings.TrimSpace(args.ScriptPath)
	if args.ScriptPath == "" {
		return officeToolError("Missing required parameter: script_path"), nil
	}
	if !filepath.IsAbs(args.ScriptPath) {
		args.ScriptPath, err = filepath.Abs(args.ScriptPath)
		if err != nil {
			return officeToolError(fmt.Sprintf("Invalid script_path: %s", err.Error())), nil
		}
	}
	scriptInfo, err := os.Stat(args.ScriptPath)
	if err != nil {
		return officeToolError(fmt.Sprintf("Invalid script_path: %s", err.Error())), nil
	}
	if scriptInfo.IsDir() {
		return officeToolError("Invalid script_path: must be a file"), nil
	}

	args.AssetsDir = strings.TrimSpace(args.AssetsDir)
	if args.AssetsDir == "" {
		args.AssetsDir = filepath.Dir(args.ScriptPath)
	} else {
		if !filepath.IsAbs(args.AssetsDir) {
			args.AssetsDir, err = filepath.Abs(args.AssetsDir)
			if err != nil {
				return officeToolError(fmt.Sprintf("Invalid assets_dir: %s", err.Error())), nil
			}
		}
		assetsInfo, err := os.Stat(args.AssetsDir)
		if err != nil {
			return officeToolError(fmt.Sprintf("Invalid assets_dir: %s", err.Error())), nil
		}
		if !assetsInfo.IsDir() {
			return officeToolError("Invalid assets_dir: must be a directory"), nil
		}
	}

	nodePath, err := exec.LookPath("node")
	if err != nil {
		return officeToolError("Node.js was not found; install Node.js to enable PowerPoint generation"), nil
	}
	workerPath, err := findPptxWorkerPath()
	if err != nil {
		return officeToolError(err.Error()), nil
	}

	args.Path = resolveOutputPath(args.Path)

	// Pass the job to Node through a temp JSON file so nested data does not
	// need fragile command-line escaping. The final PPTX is not temporary.
	specFile, err := os.CreateTemp("", "openagent-pptxgenjs-*.json")
	if err != nil {
		return officeToolError(fmt.Sprintf("Failed to create worker spec: %s", err.Error())), nil
	}
	defer os.Remove(specFile.Name())

	if err := json.NewEncoder(specFile).Encode(args); err != nil {
		specFile.Close()
		return officeToolError(fmt.Sprintf("Failed to write worker spec: %s", err.Error())), nil
	}
	if err := specFile.Close(); err != nil {
		return officeToolError(fmt.Sprintf("Failed to close worker spec: %s", err.Error())), nil
	}

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, nodePath, workerPath, specFile.Name())
	// Run from the worker directory so source workers can resolve local node_modules.
	// Bundled and embedded workers do not need node_modules at runtime.
	cmd.Dir = filepath.Dir(workerPath)
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		var workerResult pptxGenerateWorkerResult
		if len(bytes.TrimSpace(output)) > 0 && json.Unmarshal(bytes.TrimSpace(output), &workerResult) == nil && workerResult.Error != "" {
			return officeToolError(fmt.Sprintf("Failed to generate PowerPoint file: %s", workerResult.Error)), nil
		}

		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return officeToolError(fmt.Sprintf("Failed to run PowerPoint worker: %s", detail)), nil
	}

	var workerResult pptxGenerateWorkerResult
	if err := json.Unmarshal(bytes.TrimSpace(output), &workerResult); err != nil {
		return officeToolError(fmt.Sprintf("Failed to parse PowerPoint worker output: %s", err.Error())), nil
	}
	if !workerResult.OK {
		return officeToolError(fmt.Sprintf("Failed to generate PowerPoint file: %s", workerResult.Error)), nil
	}

	mode := workerResult.Mode
	if mode == "" {
		mode = "pptxgenjs"
	}
	return officeToolText(fmt.Sprintf(
		"Successfully generated PowerPoint file: %s\n%d slide(s) written\nmode: %s",
		workerResult.Path, workerResult.SlideCount, mode,
	)), nil
}

func findPptxWorkerPath() (string, error) {
	var candidates []pptxWorkerCandidate
	if exeDir, err := pptxExecutableDir(); err == nil {
		candidates = append(candidates,
			pptxWorkerCandidate{path: filepath.Join(exeDir, "pptx-worker", "worker.bundle.mjs")},
			pptxWorkerCandidate{path: filepath.Join(exeDir, "pptx-worker", "worker.mjs")},
		)
	}
	candidates = append(candidates,
		pptxWorkerCandidate{path: filepath.Join("tool", "pptx-worker", "worker.mjs"), requireNodeModules: true},
		pptxWorkerCandidate{path: filepath.Join("pptx-worker", "worker.mjs"), requireNodeModules: true},
		pptxWorkerCandidate{path: filepath.Join("tool", "pptx-worker", "worker.bundle.mjs")},
		pptxWorkerCandidate{path: filepath.Join("pptx-worker", "worker.bundle.mjs")},
	)

	for _, candidate := range candidates {
		workerInfo, err := os.Stat(candidate.path)
		if err != nil {
			continue
		}
		if workerInfo.IsDir() {
			continue
		}
		absPath, err := filepath.Abs(candidate.path)
		if err != nil {
			absPath = candidate.path
		}
		if candidate.requireNodeModules && !sourcePptxWorkerReady(absPath) {
			continue
		}
		return absPath, nil
	}

	embeddedWorker := embedsupport.PptxWorkerFS()
	if embeddedWorker != nil {
		exeDir, err := pptxExecutableDir()
		if err != nil {
			return "", err
		}
		return writeEmbeddedPptxWorker(embeddedWorker, exeDir)
	}

	return "", fmt.Errorf("PowerPoint worker not found next to the executable or in tool/pptx-worker; build with -tags embed or place worker.bundle.mjs or worker.mjs in pptx-worker")
}

func sourcePptxWorkerReady(workerPath string) bool {
	workerDir := filepath.Dir(workerPath)
	for _, dep := range []string{
		filepath.Join("node_modules", "pptxgenjs"),
		filepath.Join("node_modules", "@fortawesome", "free-solid-svg-icons"),
	} {
		info, err := os.Stat(filepath.Join(workerDir, dep))
		if err != nil || !info.IsDir() {
			return false
		}
	}
	return true
}

// writeEmbeddedPptxWorker extracts the bundled worker script to
// <rootDir>/pptx-worker/worker.mjs the first time it is needed.
// On the next call to findPptxWorkerPath the extracted file is found in the
// exe-dir candidates list before this function is ever reached, so the write
// effectively happens only once (until the file is removed or the binary is
// replaced).
func writeEmbeddedPptxWorker(source fs.FS, rootDir string) (string, error) {
	data, err := fs.ReadFile(source, "worker.bundle.mjs")
	if err != nil {
		return "", fmt.Errorf("Failed to read embedded PowerPoint worker: %s", err.Error())
	}

	workerPath := filepath.Join(rootDir, "pptx-worker", "worker.mjs")
	if err = os.MkdirAll(filepath.Dir(workerPath), 0o755); err != nil {
		return "", fmt.Errorf("Failed to prepare embedded PowerPoint worker: %s", err.Error())
	}
	if err = os.WriteFile(workerPath, data, 0o644); err != nil {
		return "", fmt.Errorf("Failed to write embedded PowerPoint worker: %s", err.Error())
	}
	return workerPath, nil
}

func pptxExecutableDir() (string, error) {
	exePath, err := os.Executable()
	if err == nil {
		return filepath.Dir(exePath), nil
	}
	wd, wdErr := os.Getwd()
	if wdErr != nil {
		return "", err
	}
	return wd, nil
}
