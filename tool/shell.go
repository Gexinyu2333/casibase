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
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
)

// ShellTool is the Tool Type "shell".
type ShellTool struct{}

func (p *ShellTool) BuiltinTools() []BuiltinTool {
	return []BuiltinTool{&shellBuiltin{}}
}

type shellBuiltin struct{}

func (s *shellBuiltin) GetName() string {
	return "shell"
}

func (s *shellBuiltin) GetDescription() string {
	return `Execute a shell command and return its output.
- command (required): the shell command to run (e.g. "ls -la", "echo hello").
- timeout: execution timeout in seconds (default 30, max 300).
- workdir: working directory for the command (default: current directory).`
}

func (s *shellBuiltin) GetInputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The shell command to execute.",
			},
			"timeout": map[string]interface{}{
				"type":        "number",
				"description": "Execution timeout in seconds (default 30, max 300).",
			},
			"workdir": map[string]interface{}{
				"type":        "string",
				"description": "Working directory for the command (default: current directory).",
			},
		},
		"required": []string{"command"},
	}
}

func (s *shellBuiltin) Execute(ctx context.Context, arguments map[string]interface{}) (*protocol.CallToolResult, error) {
	command, ok := arguments["command"].(string)
	if !ok || strings.TrimSpace(command) == "" {
		return &protocol.CallToolResult{
			IsError: true,
			Content: []protocol.Content{
				&protocol.TextContent{Type: "text", Text: "Missing required parameter: command"},
			},
		}, nil
	}

	timeoutSecs := 30.0
	if t, ok := arguments["timeout"].(float64); ok && t > 0 {
		timeoutSecs = t
		if timeoutSecs > 300 {
			timeoutSecs = 300
		}
	}

	workdir := ""
	if wd, ok := arguments["workdir"].(string); ok {
		workdir = wd
	}

	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSecs)*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(execCtx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(execCtx, "sh", "-c", command)
	}
	if workdir != "" {
		cmd.Dir = workdir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	if runErr != nil {
		var parts []string
		parts = append(parts, fmt.Sprintf("Error: %s", runErr.Error()))
		if stderrStr != "" {
			parts = append(parts, fmt.Sprintf("Stderr:\n%s", stderrStr))
		}
		if stdoutStr != "" {
			parts = append(parts, fmt.Sprintf("Stdout:\n%s", stdoutStr))
		}
		return &protocol.CallToolResult{
			IsError: true,
			Content: []protocol.Content{
				&protocol.TextContent{Type: "text", Text: strings.Join(parts, "\n")},
			},
		}, nil
	}

	result := stdoutStr
	if stderrStr != "" {
		if result != "" {
			result = fmt.Sprintf("%s\nStderr:\n%s", result, stderrStr)
		} else {
			result = fmt.Sprintf("Stderr:\n%s", stderrStr)
		}
	}
	if result == "" {
		result = "(no output)"
	}

	return &protocol.CallToolResult{
		IsError: false,
		Content: []protocol.Content{
			&protocol.TextContent{Type: "text", Text: result},
		},
	}, nil
}
