# kai

Knowledge AI — your digital twin. A CLI-based AI agent that learns about you through conversations and can answer questions, make decisions, and write in your style.

## Features

- **Brain** — persistent knowledge store with YAML frontmatter files organized by category (identity, opinions, style, decisions, knowledge)
- **Semantic Search** — full-text TF-IDF search with field boosts, stemming, and relevance scoring (recency decay, access frequency, confidence)
- **Multi-Provider** — supports Claude (Anthropic) and OpenAI, with configurable timeouts and custom base URLs (e.g. Ollama)
- **MCP Server** — Model Context Protocol server with stdio and HTTP transports for integration with Claude Code, Cursor, and other MCP clients
- **Teaching Mode** — interactive mode where you tell kai about yourself; it extracts structured learnings and saves them to the brain
- **Google Sheets** — read spreadsheets and save them as brain knowledge files

## Quick Start

```bash
# Clone and build
git clone https://github.com/tinhvqbk/kai.git
cd kai
make build

# Set up config
cp config.yaml.example config.yaml
# Edit config.yaml with your provider and API key

# Set your API key
./kai auth set-key --provider claude

# Start chatting
./kai

# Or ask a one-shot question
./kai ask "What do I think about testing?"
```

## Configuration

Copy `config.yaml.example` to `config.yaml`:

```yaml
provider: claude
providers:
  claude:
    auth: api_key
    model: claude-sonnet-4-6-20250514
    # timeout_sec: 120
  openai:
    auth: api_key
    model: gpt-4o
brain:
  path: ./brain
  max_context_files: 10
sessions_path: ./sessions
server:
  listen_addr: 0.0.0.0:8080
```

API keys can be set via:
1. `api_key` field in config.yaml
2. `kai auth set-key --provider <name>` (stored in `~/.kai/credentials.json`)

## Commands

| Command | Description |
|---------|-------------|
| `kai` | Interactive chat mode |
| `kai ask <question>` | One-shot question |
| `kai teach` | Teaching mode — tell kai about yourself |
| `kai brain` | List all brain files |
| `kai brain search <query>` | Search brain knowledge |
| `kai brain edit <file>` | Open a brain file in `$EDITOR` |
| `kai auth set-key` | Save an API key |
| `kai auth status` | Show configured providers |
| `kai mcp` | Start MCP server (stdio) |
| `kai mcp --transport http` | Start MCP server (HTTP with bearer auth) |
| `kai version` | Print version info |

## MCP Server

kai exposes these tools via MCP:

| Tool | Description | Requires LLM |
|------|-------------|---------------|
| `brain_list` | List all brain files | No |
| `brain_read` | Read a brain file | No |
| `brain_write` | Write/update a brain file | No |
| `brain_search` | Search brain by keywords | No |
| `ask` | Ask kai a question | **No** (context mode) |
| `teach` | Teach kai something new | **No** (direct mode) |
| `sheet_read` | Read a Google Sheet into brain | No |

### Using with Claude Code (no LLM config needed)

When kai runs as an MCP server inside Claude Code, **Claude is the LLM**. You don't need to configure a provider — kai's brain tools work independently:

- `ask` — returns brain context + system prompt so Claude Code answers as your digital twin
- `brain_search` / `brain_read` — Claude Code retrieves knowledge
- `brain_write` — Claude Code writes knowledge directly
- `teach` (direct mode) — Claude Code extracts the learning and passes structured fields:

```
teach(category="opinions", topic="testing", content="Prefers TDD for business logic", tags="testing,tdd", confidence="high")
```

**All 7 tools work with zero API keys** — just point the MCP config at kai and start using it. When an LLM provider is configured, `ask` and `teach` can also run the full pipeline internally.

### Stdio (default)

```json
{
  "mcpServers": {
    "kai": {
      "command": "./kai",
      "args": ["mcp"]
    }
  }
}
```

### HTTP

```bash
export KAI_MCP_TOKEN="your-secret-token"
./kai mcp --transport http --addr 0.0.0.0:8080
```

## Brain Structure

```
brain/
├── identity/      # Who you are
├── opinions/      # What you think
├── style/         # How you write/code
├── decisions/     # Architectural and life decisions
└── knowledge/     # Domain knowledge
```

Each file uses YAML frontmatter:

```markdown
---
tags:
  - go
  - testing
confidence: high
source: conversation
updated: 2026-03-26
---
Prefers table-driven tests and avoids mocks when possible.
```

## Docker

```bash
# Set your MCP token
export KAI_MCP_TOKEN="your-secret-token"

# Start kai as HTTP MCP server
docker compose up -d

# kai is now available at http://localhost:8080/mcp
```

Brain data and sessions persist in Docker volumes (`kai-brain`, `kai-sessions`).

To run without an LLM provider (brain-only mode), just omit the `config.yaml` mount:

```yaml
services:
  kai:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - kai-brain:/app/brain
    environment:
      - KAI_MCP_TOKEN=${KAI_MCP_TOKEN}
    command: ["mcp", "--transport", "http", "--addr", "0.0.0.0:8080"]

volumes:
  kai-brain:
```

## Building

```bash
make build    # Build with version info
make test     # Run all tests
make install  # Install to $GOPATH/bin
make clean    # Remove binary
```

Version is injected at build time via ldflags. Use `VERSION` to override:

```bash
make build VERSION=v1.0.0
```

## License

MIT
