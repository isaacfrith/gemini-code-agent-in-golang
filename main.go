package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	"agent/internal/agent"
	"agent/internal/tools"

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

	// Define the tools we want the agent to use
	agentTools := []tools.ToolDefinition{
		tools.ReadFileDefinition,
		tools.ListFilesDefinition,
		tools.EditFileDefinition,
	}

	// Initialize the Agent
	a := agent.NewAgent(client, getUserMessage, agentTools)
	err = a.Run(ctx)
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}
