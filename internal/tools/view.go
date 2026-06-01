package tools

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
)

// ViewFileLinesSchema defines the schema for view_file_lines.
var ViewFileLinesSchema = &genai.Schema{
	Type: genai.TypeObject,
	Properties: map[string]*genai.Schema{
		"path": {
			Type:        genai.TypeString,
			Description: "The relative path of the file to view.",
		},
		"start_line": {
			Type:        genai.TypeInteger,
			Description: "The starting line number to view (1-indexed, inclusive).",
		},
		"end_line": {
			Type:        genai.TypeInteger,
			Description: "The ending line number to view (1-indexed, inclusive).",
		},
	},
	Required: []string{"path", "start_line", "end_line"},
}

// ViewFileLinesDefinition registers view_file_lines.
var ViewFileLinesDefinition = ToolDefinition{
	Name:        "view_file_lines",
	Description: "View a specific line range of a text file. Use this to avoid reading very large files entirely.",
	Parameters:  ViewFileLinesSchema,
	Function:    ViewFileLines,
}

// ViewFileLines extracts specific line ranges from a file.
func ViewFileLines(args map[string]any) (string, error) {
	pathRaw, ok := args["path"]
	if !ok {
		return "", fmt.Errorf("missing required argument: path")
	}
	path, ok := pathRaw.(string)
	if !ok {
		return "", fmt.Errorf("argument 'path' must be a string")
	}

	startLineRaw, ok := args["start_line"]
	if !ok {
		return "", fmt.Errorf("missing required argument: start_line")
	}
	var startLine int
	switch val := startLineRaw.(type) {
	case float64:
		startLine = int(val)
	case int:
		startLine = val
	case int64:
		startLine = val
	default:
		return "", fmt.Errorf("argument 'start_line' must be an integer")
	}

	endLineRaw, ok := args["end_line"]
	if !ok {
		return "", fmt.Errorf("missing required argument: end_line")
	}
	var endLine int
	switch val := endLineRaw.(type) {
	case float64:
		endLine = int(val)
	case int:
		endLine = val
	case int64:
		endLine = val
	default:
		return "", fmt.Errorf("argument 'end_line' must be an integer")
	}

	if startLine < 1 {
		return "", fmt.Errorf("start_line must be 1 or greater")
	}
	if endLine < startLine {
		return "", fmt.Errorf("end_line must be greater than or equal to start_line")
	}

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	currentLine := 0
	for scanner.Scan() {
		currentLine++
		if currentLine >= startLine && currentLine <= endLine {
			lines = append(lines, scanner.Text())
		}
		if currentLine > endLine {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	if len(lines) == 0 {
		return "No lines found in the specified range.", nil
	}

	return strings.Join(lines, "\n"), nil
}
