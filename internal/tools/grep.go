package tools

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/generative-ai-go/genai"
)

// GrepSearchSchema defines the schema for grep_search.
var GrepSearchSchema = &genai.Schema{
	Type: genai.TypeObject,
	Properties: map[string]*genai.Schema{
		"query": {
			Type:        genai.TypeString,
			Description: "The text string to search for.",
		},
		"path": {
			Type:        genai.TypeString,
			Description: "Optional relative path to search within. Defaults to the current directory if not provided.",
		},
	},
	Required: []string{"query"},
}

// GrepSearchDefinition registers grep_search.
var GrepSearchDefinition = ToolDefinition{
	Name:        "grep_search",
	Description: "Search for a text query recursively in files within the target directory.",
	Parameters:  GrepSearchSchema,
	Function:    GrepSearch,
}

// GrepSearch performs text matching in the files under a directory.
func GrepSearch(args map[string]any) (string, error) {
	queryRaw, ok := args["query"]
	if !ok {
		return "", fmt.Errorf("missing required argument: query")
	}
	query, ok := queryRaw.(string)
	if !ok {
		return "", fmt.Errorf("argument 'query' must be a string")
	}

	dir := "."
	if pathRaw, ok := args["path"]; ok {
		if pathStr, ok := pathRaw.(string); ok && pathStr != "" {
			dir = pathStr
		}
	}

	var results []string
	maxResults := 100 // Cap results to prevent flooding the context

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden files/directories (like .git)
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." && info.Name() != ".." {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip typical build artifacts, package manager files, and binaries
		if strings.Contains(path, "node_modules/") || strings.Contains(path, "vendor/") {
			return nil
		}

		// Read file line by line
		file, err := os.Open(path)
		if err != nil {
			return nil // Skip files we cannot open
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if strings.Contains(line, query) {
				results = append(results, fmt.Sprintf("%s:%d: %s", path, lineNum, strings.TrimSpace(line)))
				if len(results) >= maxResults {
					return fmt.Errorf("reached maximum result limit of %d", maxResults)
				}
			}
		}
		return nil
	})

	// Walk returns the maxResults err if reached, but we can treat that as successfully terminating early
	if err != nil && !strings.Contains(err.Error(), "reached maximum result limit") {
		return "", err
	}

	if len(results) == 0 {
		return "No matches found.", nil
	}

	return strings.Join(results, "\n"), nil
}
