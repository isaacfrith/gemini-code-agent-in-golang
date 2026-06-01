package agent

import (
	"context"
	"fmt"

	"agent/internal/tools"

	"github.com/google/generative-ai-go/genai"
)

// Agent handles interaction with Gemini and tool execution.
type Agent struct {
	client         *genai.Client
	getUserMessage func() (string, bool)
	tools          []tools.ToolDefinition
}

// NewAgent creates a new Agent instance.
func NewAgent(client *genai.Client, getUserMessage func() (string, bool), agentTools []tools.ToolDefinition) *Agent {
	return &Agent{
		client:         client,
		getUserMessage: getUserMessage,
		tools:          agentTools,
	}
}

// Run starts the main interaction loop of the agent.
func (a *Agent) Run(ctx context.Context) error {
	// Initialize the generative model and set generation parameters
	model := a.client.GenerativeModel("gemini-2.5-pro")
	model.SetMaxOutputTokens(1024)

	// Convert and attach tools to the model
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

	// StartChat creates a session that automatically manages conversation history
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
	return chat.SendMessage(ctx, genai.Text(input))
}
