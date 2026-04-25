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
	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/casibase/casibase/agent/builtin_tool"
)

// officeSubType enumerates the allowed SubType values for OfficeProvider.
type officeSubType string

const (
	officeSubTypeAll             officeSubType = "All"
	officeSubTypeWordRead        officeSubType = "Word Read"
	officeSubTypeWordWrite       officeSubType = "Word Write"
	officeSubTypeExcelRead       officeSubType = "Excel Read"
	officeSubTypeExcelWrite      officeSubType = "Excel Write"
	officeSubTypePowerPointRead  officeSubType = "PowerPoint Read"
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
// It exposes read/write tools for Word (.docx), Excel (.xlsx), and PowerPoint (.pptx).
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

// ── shared response helpers ───────────────────────────────────────────────────

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
