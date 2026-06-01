package tools

import (
	"github.com/google/generative-ai-go/genai"
)

// ToolDefinition wraps tool specifications and their execution handler
type ToolDefinition struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Parameters  *genai.Schema `json:"parameters"`

	// Function is the actual handler. Gemini returns arguments as map[string]any,
	// which avoids having to manually marshal/unmarshal.
	Function func(args map[string]any) (string, error)
}
