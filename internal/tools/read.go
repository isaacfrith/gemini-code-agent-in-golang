package tools

import (
	"fmt"
	"os"

	"github.com/google/generative-ai-go/genai"
)

// ReadFileSchema defines the input schema for the ReadFile tool.
var ReadFileSchema = &genai.Schema{
	Type: genai.TypeObject,
	Properties: map[string]*genai.Schema{
		"path": {
			Type:        genai.TypeString,
			Description: "The relative path of a file in the working directory.",
		},
	},
	Required: []string{"path"},
}

// ReadFileDefinition registers the ReadFile tool.
var ReadFileDefinition = ToolDefinition{
	Name:        "read_file",
	Description: "Read the contents of a given relative file path. Use this when you want to see what's inside a file. Do not use this with directory names.",
	Parameters:  ReadFileSchema,
	Function:    ReadFile,
}

// ReadFile reads the file's contents from disk.
func ReadFile(args map[string]any) (string, error) {
	pathRaw, ok := args["path"]
	if !ok {
		return "", fmt.Errorf("missing required argument: path")
	}

	path, ok := pathRaw.(string)
	if !ok {
		return "", fmt.Errorf("argument 'path' must be a string")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
