package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

func main() {
	// 1. Load the .env file FIRST
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file")
	}

	ctx := context.Background()

	// 2. Now initialize the Gemini client (it will successfully find the key)
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	scanner := bufio.NewScanner(os.Stdin)
	getUserMessage := func() (string, bool) {
		if !scanner.Scan() {
			return "", false
		}
		return scanner.Text(), true
	}

	tools := []ToolDefinition{ReadFileDefinition, ListFilesDefinition, EditFileDefinition}
	agent := NewAgent(client, getUserMessage, tools)
	err = agent.Run(ctx)
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}

func NewAgent(client *genai.Client, getUserMessage func() (string, bool), tools []ToolDefinition) *Agent {
	return &Agent{
		client:         client,
		getUserMessage: getUserMessage,
		tools:          tools,
	}
}

type Agent struct {
	client         *genai.Client
	getUserMessage func() (string, bool)
	tools          []ToolDefinition
}

func (a *Agent) Run(ctx context.Context) error {
	// Initialize the generative model and set generation parameters
	model := a.client.GenerativeModel("gemini-2.5-pro")
	model.SetMaxOutputTokens(1024)

	// --- NEW: Convert and attach tools to the model HERE ---
	var funcDecls []*genai.FunctionDeclaration
	for _, tool := range a.tools {
		funcDecls = append(funcDecls, &genai.FunctionDeclaration{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.Parameters,
		})
	}

	model.Tools = []*genai.Tool{
		{
			FunctionDeclarations: funcDecls,
		},
	}
	// --------------------------------------------------------

	// StartChat creates a session that automatically manages conversation history
	// AND inherits the tools we just attached to the model.
	chat := model.StartChat()

	fmt.Println("Chat with Gemini (use 'ctrl-c' to quit)")

	for {
		fmt.Print("\u001b[94mYou\u001b[0m: ")
		userInput, ok := a.getUserMessage()
		if !ok {
			break
		}

		// Initial request to Gemini
		resp, err := a.runInference(ctx, chat, userInput)
		if err != nil {
			return err
		}

		// Inner loop: Keep communicating with Gemini automatically as long as it asks to use tools
		for {
			var calledTool bool
			var toolResponses []genai.Part

			if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
				for _, part := range resp.Candidates[0].Content.Parts {

					// 1. If Gemini replies with standard text, print it
					if text, ok := part.(genai.Text); ok {
						fmt.Printf("\u001b[93mGemini\u001b[0m: %s\n", text)
					}

					// 2. If Gemini wants to use a tool, execute it automatically
					if fc, ok := part.(genai.FunctionCall); ok {
						fmt.Printf("\u001b[95m[Agent Action: Using tool '%s']\u001b[0m\n", fc.Name)
						calledTool = true

						// Find the requested tool in our registry
						var resultStr string
						var resultErr error
						found := false
						for _, t := range a.tools {
							if t.Name == fc.Name {
								resultStr, resultErr = t.Function(fc.Args)
								found = true
								break
							}
						}

						// Prepare the result payload
						var responsePayload map[string]any
						if !found {
							responsePayload = map[string]any{"error": "tool not found"}
						} else if resultErr != nil {
							responsePayload = map[string]any{"error": resultErr.Error()}
						} else {
							responsePayload = map[string]any{"result": resultStr}
						}

						// Package it into a FunctionResponse
						toolResponses = append(toolResponses, genai.FunctionResponse{
							Name:     fc.Name,
							Response: responsePayload,
						})
					}
				}
			}

			// 3. If a tool was executed, immediately send the result back to Gemini so it can continue
			if calledTool {
				resp, err = chat.SendMessage(ctx, toolResponses...)
				if err != nil {
					return err
				}
			} else {
				// No tools were called, meaning Gemini is done with its turn. Break to user input.
				break
			}
		}
	}

	return nil
}

func (a *Agent) runInference(ctx context.Context, chat *genai.ChatSession, input string) (*genai.GenerateContentResponse, error) {
	// The chat session already knows about the tools from the model initialization
	return chat.SendMessage(ctx, genai.Text(input))
}

type ToolDefinition struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Parameters  *genai.Schema `json:"parameters"` // Replaces anthropic.ToolInputSchemaParam

	// Gemini natively returns function arguments as a map[string]any,
	// so using this signature saves you from having to marshal/unmarshal JSON bytes.
	Function func(args map[string]any) (string, error)
}

// 1. Explicitly define the parameter schema using Gemini's native types
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

// 2. Define the ReadFile tool matching the Gemini-friendly signature
var ReadFileDefinition = ToolDefinition{
	Name:        "read_file",
	Description: "Read the contents of a given relative file path. Use this when you want to see what's inside a file. Do not use this with directory names.",
	Parameters:  ReadFileSchema,
	Function:    ReadFile,
}

// 3. Updated ReadFile to extract parameters from map[string]any
func ReadFile(args map[string]any) (string, error) {
	// Extract and type-assert the path string from the arguments map
	pathRaw, ok := args["path"]
	if !ok {
		return "", fmt.Errorf("missing required argument: path")
	}

	path, ok := pathRaw.(string)
	if !ok {
		return "", fmt.Errorf("argument 'path' must be a string")
	}

	// Execute the core logic
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

var ListFilesSchema = &genai.Schema{
	Type: genai.TypeObject,
	Properties: map[string]*genai.Schema{
		"path": {
			Type:        genai.TypeString,
			Description: "Optional relative path to list files from. Defaults to current directory if not provided.",
		},
	},
}

// 2. Define the ListFiles tool matching the Gemini-friendly signature
var ListFilesDefinition = ToolDefinition{
	Name:        "list_files",
	Description: "List files and directories at a given path. If no path is provided, lists files in the current directory.",
	Parameters:  ListFilesSchema,
	Function:    ListFiles,
}

// 3. Updated ListFiles to extract parameters from map[string]any
func ListFiles(args map[string]any) (string, error) {
	dir := "."

	// Safely extract the optional path argument if it exists
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

	// Marshal the slice of filenames back to a JSON string for Gemini to read
	result, err := json.Marshal(files)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

// 1. Explicitly define the parameter schema using Gemini's native types
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

var EditFileDefinition = ToolDefinition{
	Name: "edit_file",
	Description: `Make edits to a text file.

Replaces 'old_str' with 'new_str' in the given file. 'old_str' and 'new_str' MUST be different from each other.

If the file specified with path doesn't exist, it will be created.`,
	Parameters: EditFileSchema,
	Function:   EditFile,
}

func EditFile(args map[string]any) (string, error) {
	// Extract arguments safely using type assertions
	path, okPath := args["path"].(string)
	oldStr, okOld := args["old_str"].(string)
	newStr, okNew := args["new_str"].(string)

	// Enforce your initial validation checks
	if !okPath || !okOld || !okNew || path == "" || oldStr == newStr {
		return "", fmt.Errorf("invalid input parameters")
	}

	// Read the targeted file
	content, err := os.ReadFile(path)
	if err != nil {
		// If the file doesn't exist and oldStr is empty, create a new file
		if os.IsNotExist(err) && oldStr == "" {
			return createNewFile(path, newStr)
		}
		return "", err
	}

	oldContent := string(content)

	// Perform the string replacement globally (-1 replaces all instances)
	newContent := strings.Replace(oldContent, oldStr, newStr, -1)

	// If the file content didn't change and we weren't targeting an empty string, error out
	if oldContent == newContent && oldStr != "" {
		return "", fmt.Errorf("old_str not found in file")
	}

	// Write the modified content back out to disk
	err = os.WriteFile(path, []byte(newContent), 0644)
	if err != nil {
		return "", err
	}

	return "OK", nil
}

// Helper function called by the EditFile tool
func createNewFile(filePath, content string) (string, error) {
	// 1. Extract the directory path
	dir := filepath.Dir(filePath)

	// 2. Create the directory tree if it doesn't exist
	if dir != "." {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return "", fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// 3. Write the file content
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}

	return fmt.Sprintf("Successfully created file %s", filePath), nil
}
