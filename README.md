# Gemini Code Agent in Go

A lightweight, terminal-based AI coding assistant built in Go using the official Google GenAI SDK. This agent features an interactive chat interface and is equipped with file-system tools allowing it to explore, read, and write code under user instructions.

## Key Features

- **Gemini 2.5 Pro Model**: Powered by Google's state-of-the-art model for planning and tool execution.
- **Autonomous Tool Execution**: The agent automatically runs tool operations (like file reads or writes) suggested by the LLM and reports back the results until Gemini finishes its turn.
- **Clean Architecture**: Separated packages for the core agent logic and tool implementations, designed for easy extensibility.

---

## Directory Structure

```
gemini-code-agent-in-golang/
├── .env                  # Configuration for API keys (gitignored)
├── go.mod                # Go module descriptor
├── go.sum                # Go checksum database
├── main.go               # Entry point: loads config and boots agent
└── internal/             # Code private to this repository
    ├── agent/            # Core agent loop and LLM communication
    │   └── agent.go
    └── tools/            # Individual tool implementations
        ├── tools.go      # Shared ToolDefinition struct and registry
        ├── read.go       # ReadFile implementation
        ├── list.go       # ListFiles implementation
        └── edit.go       # EditFile & createNewFile implementation
```

---

## Prerequisites

- **Go**: Version 1.26.3 or higher.
- **Gemini API Key**: A valid Google AI Studio Gemini API key.

---

## Setup & Running

1. **Clone the repository**:
   ```bash
   git clone https://github.com/isaacfrith/gemini-code-agent-in-golang.git
   cd gemini-code-agent-in-golang
   ```

2. **Configure environment variables**:
   Create a `.env` file in the root directory:
   ```bash
   touch .env
   ```
   Add your API key inside:
   ```env
   GEMINI_API_KEY=your_gemini_api_key_here
   ```

3. **Install dependencies**:
   ```bash
   go mod download
   ```

4. **Run the agent**:
   ```bash
   go run main.go
   ```

---

## Available Tools

The agent is registered with three tools to interact with the local workspace:

- **`list_files`**: Lists relative file paths and directories in the target workspace directory.
- **`read_file`**: Reads and returns the contents of a specific file.
- **`edit_file`**: Replaces specific target content or creates new files if they do not exist.
