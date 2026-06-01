package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/generative-ai-go/genai"
)

// EditFileSchema defines the input schema for the EditFile tool.
var EditFileSchema = &genai.Schema{
	Type: genai.TypeObject,
	Properties: map[string]*genai.Schema{
		"path": {
			Type:        genai.TypeString,
			Description: "The path to the file",
		},
		"old_str": {
			Type:        genai.TypeString,
			Description: "Text to search for - must match exactly and must only have one match exactly",
		},
		"new_str": {
			Type:        genai.TypeString,
			Description: "Text to replace old_str with",
		},
	},
	Required: []string{"path", "old_str", "new_str"},
}

// EditFileDefinition registers the EditFile tool.
var EditFileDefinition = ToolDefinition{
	Name: "edit_file",
	Description: `Make edits to a text file.

Replaces 'old_str' with 'new_str' in the given file. 'old_str' and 'new_str' MUST be different from each other.

If the file specified with path doesn't exist, it will be created.`,
	Parameters: EditFileSchema,
	Function:   EditFile,
}

// EditFile replaces occurrences of oldStr with newStr in the targeted file.
func EditFile(args map[string]any) (string, error) {
	path, okPath := args["path"].(string)
	oldStr, okOld := args["old_str"].(string)
	newStr, okNew := args["new_str"].(string)

	if !okPath || !okOld || !okNew || path == "" || oldStr == newStr {
		return "", fmt.Errorf("invalid input parameters")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && oldStr == "" {
			return createNewFile(path, newStr)
		}
		return "", err
	}

	oldContent := string(content)

	newContent := strings.Replace(oldContent, oldStr, newStr, -1)

	if oldContent == newContent && oldStr != "" {
		return "", fmt.Errorf("old_str not found in file")
	}

	err = os.WriteFile(path, []byte(newContent), 0644)
	if err != nil {
		return "", err
	}

	return "OK", nil
}

// Helper function called by the EditFile tool
func createNewFile(filePath, content string) (string, error) {
	dir := filepath.Dir(filePath)

	if dir != "." {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return "", fmt.Errorf("failed to create directory: %w", err)
		}
	}

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}

	return fmt.Sprintf("Successfully created file %s", filePath), nil
}
