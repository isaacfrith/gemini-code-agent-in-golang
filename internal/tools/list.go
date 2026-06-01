package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/generative-ai-go/genai"
)

// ListFilesSchema defines the input schema for the ListFiles tool.
var ListFilesSchema = &genai.Schema{
	Type: genai.TypeObject,
	Properties: map[string]*genai.Schema{
		"path": {
			Type:        genai.TypeString,
			Description: "Optional relative path to list files from. Defaults to current directory if not provided.",
		},
	},
}

// ListFilesDefinition registers the ListFiles tool.
var ListFilesDefinition = ToolDefinition{
	Name:        "list_files",
	Description: "List files and directories at a given path. If no path is provided, lists files in the current directory.",
	Parameters:  ListFilesSchema,
	Function:    ListFiles,
}

// ListFiles walks the filesystem at the specified path and returns list of files/dirs.
func ListFiles(args map[string]any) (string, error) {
	dir := "."

	if pathRaw, ok := args["path"]; ok {
		if pathStr, ok := pathRaw.(string); ok && pathStr != "" {
			dir = pathStr
		} else if !ok {
			return "", fmt.Errorf("argument 'path' must be a string")
		}
	}

	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		if relPath != "." {
			if info.IsDir() {
				files = append(files, relPath+"/")
			} else {
				files = append(files, relPath)
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	result, err := json.Marshal(files)
	if err != nil {
		return "", err
	}

	return string(result), nil
}
