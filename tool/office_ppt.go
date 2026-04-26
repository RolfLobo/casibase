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
	ppt "github.com/casibase/goppt"
)

// ── PowerPoint read ───────────────────────────────────────────────────────────

type pptxReadBuiltin struct{}

func (t *pptxReadBuiltin) GetName() string { return "pptx_read" }

func (t *pptxReadBuiltin) GetDescription() string {
	return `Read text content from a PowerPoint (.pptx) file and return it slide by slide.
- path (required): absolute or relative path to the .pptx file.
Each slide section lists the slide name and all readable text extracted from shapes and speaker notes.`
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

func (t *pptxReadBuiltin) Execute(_ context.Context, arguments map[string]interface{}) (*protocol.CallToolResult, error) {
	path, ok := arguments["path"].(string)
	if !ok || strings.TrimSpace(path) == "" {
		return officeToolError("Missing required parameter: path"), nil
	}

	result, err := readPptxFile(path)
	if err != nil {
		return officeToolError(fmt.Sprintf("Failed to read PowerPoint file: %s", err.Error())), nil
	}
	return officeToolText(result), nil
}

// ── PowerPoint write ──────────────────────────────────────────────────────────

type pptxWriteBuiltin struct{}

func (t *pptxWriteBuiltin) GetName() string { return "pptx_write" }

func (t *pptxWriteBuiltin) GetDescription() string {
	return `Create a PowerPoint (.pptx) file from an array of slide content strings.
- path (required): output path for the .pptx file. Absolute paths are used as-is. Relative paths or bare filenames are resolved inside the current user's Documents folder.
- slides (required): JSON array of slide content strings, one element per slide.
  For each slide the first line becomes the title; remaining lines become the body text.
Creates the file if it does not exist; overwrites otherwise.`
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
				"description": "Array of slide content strings (one element per slide). First line = title, remaining lines = body.",
				"items":       map[string]interface{}{"type": "string"},
			},
		},
		"required": []string{"path", "slides"},
	}
}

func (t *pptxWriteBuiltin) Execute(_ context.Context, arguments map[string]interface{}) (*protocol.CallToolResult, error) {
	path, ok := arguments["path"].(string)
	if !ok || strings.TrimSpace(path) == "" {
		return officeToolError("Missing required parameter: path"), nil
	}

	rawSlides, ok := arguments["slides"].([]interface{})
	if !ok || len(rawSlides) == 0 {
		return officeToolError("Missing or empty required parameter: slides"), nil
	}

	slideTexts := make([]string, 0, len(rawSlides))
	for i, v := range rawSlides {
		s, ok := v.(string)
		if !ok {
			return officeToolError(fmt.Sprintf("slides[%d] must be a string", i)), nil
		}
		slideTexts = append(slideTexts, s)
	}

	resolvedPath := resolveOutputPath(path)
	if err := writePptxFile(path, slideTexts); err != nil {
		return officeToolError(fmt.Sprintf("Failed to write PowerPoint file: %s", err.Error())), nil
	}
	return officeToolText(fmt.Sprintf(
		"Successfully wrote PowerPoint file: %s\n%d slide(s) written",
		resolvedPath, len(slideTexts),
	)), nil
}

// ── goppt helpers ─────────────────────────────────────────────────────────────

// readPptxFile opens a .pptx file and returns its text content formatted
// as one section per slide.
func readPptxFile(path string) (string, error) {
	reader := &ppt.PPTXReader{}
	pres, err := reader.Read(path)
	if err != nil {
		return "", err
	}

	slideCount := pres.GetSlideCount()
	if slideCount == 0 {
		return "", fmt.Errorf("the PowerPoint file contains no slides")
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Total slides: %d\n", slideCount)

	for i := 0; i < slideCount; i++ {
		slide, err := pres.GetSlide(i)
		if err != nil {
			continue
		}

		name := slide.GetName()
		if name == "" {
			name = fmt.Sprintf("Slide %d", i+1)
		}
		fmt.Fprintf(&sb, "\n=== %s ===\n", name)

		for _, shape := range slide.GetShapes() {
			text := extractShapeText(shape)
			text = strings.TrimSpace(text)
			if text != "" {
				sb.WriteString(text)
				sb.WriteByte('\n')
			}
		}

		if notes := strings.TrimSpace(slide.GetNotes()); notes != "" {
			fmt.Fprintf(&sb, "[Notes] %s\n", notes)
		}
	}
	return sb.String(), nil
}

// extractShapeText recursively extracts all text from a Shape.
func extractShapeText(shape ppt.Shape) string {
	switch shape.GetType() {
	case ppt.ShapeTypeRichText:
		return extractRichTextShapeText(shape.(*ppt.RichTextShape))
	case ppt.ShapeTypeAutoShape:
		return shape.(*ppt.AutoShape).GetText()
	case ppt.ShapeTypeGroup:
		return extractGroupShapeText(shape.(*ppt.GroupShape))
	case ppt.ShapeTypeTable:
		return extractTableShapeText(shape.(*ppt.TableShape))
	default:
		return ""
	}
}

// extractRichTextShapeText collects text from all paragraphs and runs.
func extractRichTextShapeText(rt *ppt.RichTextShape) string {
	var lines []string
	for _, para := range rt.GetParagraphs() {
		var line strings.Builder
		for _, elem := range para.GetElements() {
			switch e := elem.(type) {
			case *ppt.TextRun:
				line.WriteString(e.GetText())
			case *ppt.BreakElement:
				lines = append(lines, line.String())
				line.Reset()
			}
		}
		lines = append(lines, line.String())
	}
	return strings.Join(lines, "\n")
}

// extractGroupShapeText recurses into a group's child shapes.
func extractGroupShapeText(g *ppt.GroupShape) string {
	var parts []string
	for _, child := range g.GetShapes() {
		if t := strings.TrimSpace(extractShapeText(child)); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, "\n")
}

// extractTableShapeText collects text from all table cells, row by row.
func extractTableShapeText(t *ppt.TableShape) string {
	var lines []string
	for _, row := range t.GetRows() {
		var cells []string
		for _, cell := range row {
			if cell == nil {
				cells = append(cells, "")
				continue
			}
			var cellText strings.Builder
			for _, para := range cell.GetParagraphs() {
				for _, elem := range para.GetElements() {
					if tr, ok := elem.(*ppt.TextRun); ok {
						cellText.WriteString(tr.GetText())
					}
				}
			}
			cells = append(cells, cellText.String())
		}
		lines = append(lines, strings.Join(cells, "\t"))
	}
	return strings.Join(lines, "\n")
}

const (
	// Standard 10" × 7.5" slide dimensions in EMU (914400 EMU = 1 inch).
	slideWidth  int64 = 9144000
	slideHeight int64 = 6858000

	// Layout constants for title / body text boxes.
	marginX  int64 = 457200 // 0.5 inch left/right margin
	titleY   int64 = 300000
	titleH   int64 = 1100000
	bodyY    int64 = 1500000
	bodyH    int64 = 5000000
	contentW int64 = 8229600 // slideWidth - 2 * marginX
)

// writePptxFile creates a .pptx from a slice of slide text strings.
// Each string: first line → title text box; remaining lines → body text box.
func writePptxFile(path string, slideTexts []string) error {
	path = resolveOutputPath(path)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", dir, err)
	}

	p := ppt.New()

	for i, content := range slideTexts {
		var slide *ppt.Slide
		if i == 0 {
			slide = p.GetActiveSlide()
		} else {
			slide = p.CreateSlide()
		}

		lines := strings.Split(content, "\n")
		title := strings.TrimSpace(lines[0])
		bodyLines := lines[1:]

		// Title text box.
		titleShape := slide.CreateRichTextShape()
		titleShape.BaseShape.SetOffsetX(marginX).SetOffsetY(titleY).
			SetWidth(contentW).SetHeight(titleH)
		titleShape.SetWordWrap(true)
		titleShape.SetAutoFit(ppt.AutoFitNormal)
		titleShape.GetActiveParagraph().CreateTextRun(title).GetFont().SetBold(true).SetSize(36)

		// Body text box — only when there is non-empty content.
		body := strings.TrimSpace(strings.Join(bodyLines, "\n"))
		if body != "" {
			bodyShape := slide.CreateRichTextShape()
			bodyShape.BaseShape.SetOffsetX(marginX).SetOffsetY(bodyY).
				SetWidth(contentW).SetHeight(bodyH)
			bodyShape.SetWordWrap(true)
			bodyShape.SetAutoFit(ppt.AutoFitNormal)

			firstPara := true
			for _, line := range bodyLines {
				if firstPara {
					bodyShape.GetActiveParagraph().CreateTextRun(line).GetFont().SetSize(24)
					firstPara = false
				} else {
					bodyShape.CreateParagraph().CreateTextRun(line).GetFont().SetSize(24)
				}
			}
		}
	}

	w, err := ppt.NewWriter(p, ppt.WriterPowerPoint2007)
	if err != nil {
		return fmt.Errorf("failed to create writer: %w", err)
	}
	pptxWriter, ok := w.(*ppt.PPTXWriter)
	if !ok {
		return fmt.Errorf("unexpected writer type")
	}
	return pptxWriter.Save(path)
}
