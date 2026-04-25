// Copyright 2026 The Casibase Authors. All Rights Reserved.
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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/carmel/gooxml/document"
	"github.com/casibase/casibase/agent/builtin_tool"
)

// officeSubType enumerates the allowed SubType values for OfficeProvider.
type officeSubType string

const (
	officeSubTypeAll            officeSubType = "All"
	officeSubTypeWordRead       officeSubType = "Word Read"
	officeSubTypeWordWrite      officeSubType = "Word Write"
	officeSubTypeExcelRead      officeSubType = "Excel Read"
	officeSubTypeExcelWrite     officeSubType = "Excel Write"
	officeSubTypePowerPointRead officeSubType = "PowerPoint Read"
	officeSubTypePowerPointWrite officeSubType = "PowerPoint Write"
)

// allOfficeTools is the full ordered list returned when SubType is "All".
var allOfficeTools = []builtin_tool.BuiltinTool{
	&wordReadBuiltin{},
	&wordWriteBuiltin{},
	&excelReadBuiltin{},
	&excelWriteBuiltin{},
	&pptxReadBuiltin{},
	&pptxWriteBuiltin{},
}

// officeToolBySubType maps each specific SubType to its single tool.
var officeToolBySubType = map[officeSubType]builtin_tool.BuiltinTool{
	officeSubTypeWordRead:        &wordReadBuiltin{},
	officeSubTypeWordWrite:       &wordWriteBuiltin{},
	officeSubTypeExcelRead:       &excelReadBuiltin{},
	officeSubTypeExcelWrite:      &excelWriteBuiltin{},
	officeSubTypePowerPointRead:  &pptxReadBuiltin{},
	officeSubTypePowerPointWrite: &pptxWriteBuiltin{},
}

// OfficeProvider is the Tool provider Type "Office".
// It exposes read/write tools for Word (.docx), Excel (.xlsx) and PowerPoint (.pptx).
// Excel and PowerPoint tools are reserved for future implementation.
// SubType controls which tool(s) are exposed: "All" exposes all six; any specific
// SubType (e.g. "Word Read") exposes only that single tool.
type OfficeProvider struct {
	subType officeSubType
}

func (p *OfficeProvider) BuiltinTools() []builtin_tool.BuiltinTool {
	if p.subType == officeSubTypeAll || p.subType == "" {
		return allOfficeTools
	}
	if t, ok := officeToolBySubType[p.subType]; ok {
		return []builtin_tool.BuiltinTool{t}
	}
	return allOfficeTools
}

// ── Word read ─────────────────────────────────────────────────────────────────

type wordReadBuiltin struct{}

func (t *wordReadBuiltin) GetName() string { return "word_read" }

func (t *wordReadBuiltin) GetDescription() string {
	return `Read text content from a Word (.docx) file and return it as plain text.
- path (required): absolute or relative path to the .docx file.`
}

func (t *wordReadBuiltin) GetInputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the .docx file to read.",
			},
		},
		"required": []string{"path"},
	}
}

func (t *wordReadBuiltin) Execute(_ context.Context, arguments map[string]interface{}) (*protocol.CallToolResult, error) {
	path, ok := arguments["path"].(string)
	if !ok || strings.TrimSpace(path) == "" {
		return officeToolError("Missing required parameter: path"), nil
	}

	text, err := readWordFile(path)
	if err != nil {
		return officeToolError(fmt.Sprintf("Failed to read Word file: %s", err.Error())), nil
	}
	if strings.TrimSpace(text) == "" {
		return officeToolError("The Word document is empty"), nil
	}
	return officeToolText(text), nil
}

// ── Word write ────────────────────────────────────────────────────────────────

type wordWriteBuiltin struct{}

func (t *wordWriteBuiltin) GetName() string { return "word_write" }

func (t *wordWriteBuiltin) GetDescription() string {
	return `Write text content to a Word (.docx) file. Creates a new file; overwrites if it already exists.
- path (required): absolute or relative output path for the .docx file.
- content (required): text to write. Each newline becomes a new paragraph.`
}

func (t *wordWriteBuiltin) GetInputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Output path for the .docx file.",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Text content to write. Newlines become paragraph breaks.",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (t *wordWriteBuiltin) Execute(_ context.Context, arguments map[string]interface{}) (*protocol.CallToolResult, error) {
	path, ok := arguments["path"].(string)
	if !ok || strings.TrimSpace(path) == "" {
		return officeToolError("Missing required parameter: path"), nil
	}
	content, ok := arguments["content"].(string)
	if !ok {
		return officeToolError("Missing required parameter: content"), nil
	}

	if err := writeWordFile(path, content); err != nil {
		return officeToolError(fmt.Sprintf("Failed to write Word file: %s", err.Error())), nil
	}
	return officeToolText(fmt.Sprintf("Successfully wrote Word file: %s", path)), nil
}

// ── Excel read (reserved) ─────────────────────────────────────────────────────

type excelReadBuiltin struct{}

func (t *excelReadBuiltin) GetName() string { return "excel_read" }

func (t *excelReadBuiltin) GetDescription() string {
	return `Read data from an Excel (.xlsx) file and return it as CSV-formatted text.
- path (required): absolute or relative path to the .xlsx file.
- sheet: sheet name to read (default: first sheet).
NOTE: This tool is reserved and not yet implemented.`
}

func (t *excelReadBuiltin) GetInputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the .xlsx file to read.",
			},
			"sheet": map[string]interface{}{
				"type":        "string",
				"description": "Sheet name to read (default: first sheet).",
			},
		},
		"required": []string{"path"},
	}
}

func (t *excelReadBuiltin) Execute(_ context.Context, _ map[string]interface{}) (*protocol.CallToolResult, error) {
	return officeToolError("excel_read is not yet implemented"), nil
}

// ── Excel write (reserved) ────────────────────────────────────────────────────

type excelWriteBuiltin struct{}

func (t *excelWriteBuiltin) GetName() string { return "excel_write" }

func (t *excelWriteBuiltin) GetDescription() string {
	return `Write data to an Excel (.xlsx) file.
- path (required): output path for the .xlsx file.
- data (required): CSV-formatted text representing the cell data.
- sheet: sheet name (default: Sheet1).
NOTE: This tool is reserved and not yet implemented.`
}

func (t *excelWriteBuiltin) GetInputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Output path for the .xlsx file.",
			},
			"data": map[string]interface{}{
				"type":        "string",
				"description": "CSV-formatted text representing the rows and columns to write.",
			},
			"sheet": map[string]interface{}{
				"type":        "string",
				"description": "Sheet name (default: Sheet1).",
			},
		},
		"required": []string{"path", "data"},
	}
}

func (t *excelWriteBuiltin) Execute(_ context.Context, _ map[string]interface{}) (*protocol.CallToolResult, error) {
	return officeToolError("excel_write is not yet implemented"), nil
}

// ── PowerPoint read (reserved) ────────────────────────────────────────────────

type pptxReadBuiltin struct{}

func (t *pptxReadBuiltin) GetName() string { return "pptx_read" }

func (t *pptxReadBuiltin) GetDescription() string {
	return `Read text content from a PowerPoint (.pptx) file, slide by slide.
- path (required): absolute or relative path to the .pptx file.
NOTE: This tool is reserved and not yet implemented.`
}

func (t *pptxReadBuiltin) GetInputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the .pptx file to read.",
			},
		},
		"required": []string{"path"},
	}
}

func (t *pptxReadBuiltin) Execute(_ context.Context, _ map[string]interface{}) (*protocol.CallToolResult, error) {
	return officeToolError("pptx_read is not yet implemented"), nil
}

// ── PowerPoint write (reserved) ───────────────────────────────────────────────

type pptxWriteBuiltin struct{}

func (t *pptxWriteBuiltin) GetName() string { return "pptx_write" }

func (t *pptxWriteBuiltin) GetDescription() string {
	return `Create a PowerPoint (.pptx) file from a list of slide texts.
- path (required): output path for the .pptx file.
- slides (required): JSON array of slide content strings, one element per slide.
NOTE: This tool is reserved and not yet implemented.`
}

func (t *pptxWriteBuiltin) GetInputSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Output path for the .pptx file.",
			},
			"slides": map[string]interface{}{
				"type":        "array",
				"description": "Array of slide content strings (one per slide).",
				"items":       map[string]interface{}{"type": "string"},
			},
		},
		"required": []string{"path", "slides"},
	}
}

func (t *pptxWriteBuiltin) Execute(_ context.Context, _ map[string]interface{}) (*protocol.CallToolResult, error) {
	return officeToolError("pptx_write is not yet implemented"), nil
}

// ── gooxml helpers ────────────────────────────────────────────────────────────

func readWordFile(path string) (string, error) {
	doc, err := document.Open(path)
	if err != nil {
		return "", err
	}

	var parts []string
	for _, para := range doc.Paragraphs() {
		var paraText string
		for _, run := range para.Runs() {
			paraText += run.Text()
		}
		parts = append(parts, paraText)
	}
	return strings.Join(parts, "\n"), nil
}

func writeWordFile(path string, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", dir, err)
	}

	doc := document.New()
	for _, line := range strings.Split(content, "\n") {
		para := doc.AddParagraph()
		run := para.AddRun()
		run.AddText(line)
	}
	return doc.SaveToFile(path)
}

// ── response helpers ──────────────────────────────────────────────────────────

func officeToolText(text string) *protocol.CallToolResult {
	return &protocol.CallToolResult{
		IsError: false,
		Content: []protocol.Content{
			&protocol.TextContent{Type: "text", Text: text},
		},
	}
}

func officeToolError(text string) *protocol.CallToolResult {
	return &protocol.CallToolResult{
		IsError: true,
		Content: []protocol.Content{
			&protocol.TextContent{Type: "text", Text: text},
		},
	}
}
