# POTUS - Power Of The Universal Shell

An open-source, provider-agnostic AI coding assistant for your terminal. POTUS brings the power of Claude, GPT, and local models directly to your command line with full tool support for file editing, git operations, code search, and more.

## Features

- **Multi-Provider Support** - Use Anthropic (Claude), OpenAI (GPT), or Ollama (local models)
- **14 Built-in Tools** - File operations, bash execution, git commands, code search, web fetching
- **Diff Preview & Confirmation** - Review changes before they're applied (like Claude Code)
- **Context Management** - Smart token tracking with automatic conversation compaction
- **Beautiful TUI** - Styled terminal interface with streaming responses
- **Secure Auth** - API keys stored securely with `potus auth login`
- **Project Context** - Automatically loads `POTUS.md` or `CLAUDE.md` for project-specific instructions

## Installation

### Using Go

```bash
go install github.com/taaha3244/potus/cmd/potus@latest
```

### Download Binary

Download the latest release for your platform from [GitHub Releases](https://github.com/taaha3244/potus/releases):

```bash
# macOS (Apple Silicon)
curl -fsSL https://github.com/taaha3244/potus/releases/latest/download/potus_Darwin_arm64.tar.gz | tar -xz
sudo mv potus /usr/local/bin/

# macOS (Intel)
curl -fsSL https://github.com/taaha3244/potus/releases/latest/download/potus_Darwin_x86_64.tar.gz | tar -xz
sudo mv potus /usr/local/bin/

# Linux (amd64)
curl -fsSL https://github.com/taaha3244/potus/releases/latest/download/potus_Linux_x86_64.tar.gz | tar -xz
sudo mv potus /usr/local/bin/

# Linux (arm64)
curl -fsSL https://github.com/taaha3244/potus/releases/latest/download/potus_Linux_arm64.tar.gz | tar -xz
sudo mv potus /usr/local/bin/
```

### Verify Installation

```bash
potus --version
```

## Quick Start

### 1. Authenticate with your provider

```bash
# For Anthropic (Claude)
potus auth login anthropic
# Enter your API key when prompted

# For OpenAI
potus auth login openai

# List configured providers
potus auth list
```

Alternatively, set environment variables:
```bash
export ANTHROPIC_API_KEY=sk-ant-...
export OPENAI_API_KEY=sk-...
```

### 2. Start chatting

```bash
# Start POTUS in the current directory
potus

# Or specify a directory
potus --dir /path/to/project

# Use a specific model
potus --model anthropic/claude-sonnet-4-5
potus --model openai/gpt-5.2
potus --model ollama/qwen2.5-coder:32b
```

## IDE Integration

### VS Code

Add POTUS as an integrated terminal task:

1. Open `.vscode/tasks.json` (create if it doesn't exist)
2. Add the following configuration:

```json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "POTUS",
      "type": "shell",
      "command": "potus",
      "args": ["--dir", "${workspaceFolder}"],
      "presentation": {
        "reveal": "always",
        "panel": "new",
        "focus": true
      },
      "problemMatcher": []
    }
  ]
}
```

3. Run with `Cmd+Shift+P` (Mac) or `Ctrl+Shift+P` (Windows/Linux) → "Tasks: Run Task" → "POTUS"

Or add a keyboard shortcut in `keybindings.json`:
```json
{
  "key": "cmd+shift+p",
  "command": "workbench.action.tasks.runTask",
  "args": "POTUS"
}
```

### Cursor

Cursor uses the same VS Code task system. Follow the VS Code instructions above.

Alternatively, open Cursor's integrated terminal (`Ctrl+`` `) and run:
```bash
potus
```

### Terminal (CLI)

POTUS works in any terminal:

```bash
# Navigate to your project
cd ~/projects/my-app

# Start POTUS
potus
```

## Available Tools

| Tool | Description |
|------|-------------|
| `file_read` | Read file contents with optional line ranges |
| `file_write` | Create new files |
| `file_edit` | Search and replace edits in existing files |
| `file_delete` | Delete files |
| `bash` | Execute shell commands with safety checks |
| `git_status` | Show working tree status |
| `git_diff` | Show changes between commits |
| `git_commit` | Create commits with staged changes |
| `git_log` | View commit history |
| `git_branch` | List and manage branches |
| `search_files` | Find files by glob patterns (*.go, **/*.ts) |
| `search_content` | Search text within files (grep-like) |
| `web_fetch` | Fetch and extract content from URLs |
| `web_search` | Search the web using DuckDuckGo |

## Confirmation Flow

When POTUS wants to modify files or run commands, you'll see a diff preview:

```
-> file_edit: main.go
--- main.go
+++ main.go
-    fmt.Println("hello")
+    fmt.Println("hello world")

[y] approve  [n] deny  [a] always allow
```

- Press `y` to approve the change
- Press `n` to deny (POTUS will know and can try something else)
- Press `a` to always allow this tool (saved to `.potus/settings.json`)

## Configuration

POTUS uses hierarchical configuration:

1. Built-in defaults
2. Global: `~/.config/potus/config.yaml`
3. Project: `.potus/config.yaml`
4. Local: `.potus/config.local.yaml` (gitignored)
5. Environment variables: `POTUS_*`
6. Command-line flags

### Example Configuration

Create `.potus/config.yaml` in your project:

```yaml
providers:
  anthropic:
    api_key_env: ANTHROPIC_API_KEY
    default_model: claude-sonnet-4-5-20250929

  openai:
    api_key_env: OPENAI_API_KEY
    default_model: gpt-5.2

  ollama:
    endpoint: http://localhost:11434
    default_model: qwen2.5-coder:32b

agents:
  default:
    model: anthropic/claude-sonnet-4-5
    max_tokens: 8192
    temperature: 0.7

context:
  max_tokens: 100000
  warn_threshold: 0.8
  auto_compact: true
  load_project_context: true
  project_context_files:
    - POTUS.md
    - CLAUDE.md
    - .cursorrules
```

## Project Context

Create a `POTUS.md` (or `CLAUDE.md`) in your project root to give POTUS context about your codebase:

```markdown
# Project Context

This is a Go web application using:
- Fiber for HTTP routing
- GORM for database access
- PostgreSQL database

## Code Style
- Use table-driven tests
- Keep functions under 50 lines
- Always handle errors explicitly
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Enter` | Send message |
| `Ctrl+C` / `Esc` | Quit |
| `y` | Approve tool (during confirmation) |
| `n` | Deny tool (during confirmation) |
| `a` | Always allow tool (during confirmation) |

## Supported Models

### Anthropic
- Claude Opus 4.5
- Claude Sonnet 4.5
- Claude Sonnet 4
- Claude Haiku 4.5

### OpenAI
- GPT-5.2
- GPT-5
- GPT-5 Mini
- GPT-4.1
- O4 Mini

### Ollama (Local)
Any model available in your Ollama installation:
- qwen2.5-coder
- llama3.2
- codellama
- deepseek-coder
- etc.

## Building from Source

For contributors or if you prefer to build locally:

```bash
git clone https://github.com/taaha3244/potus.git
cd potus
make build
make install
```

## Contributing

```bash
# Run tests
make test

# Format code
make fmt

# Lint
make lint
```

1. Fork the repository
2. Create a feature branch
3. Write tests for your changes
4. Ensure all tests pass: `make test`
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file

## Acknowledgments

Inspired by [Claude Code](https://github.com/anthropics/claude-code), [OpenCode](https://github.com/anomalyco/opencode), and [Aider](https://github.com/Aider-AI/aider).

---

**Built for developers who love the terminal**
