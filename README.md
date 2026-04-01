# kai

Knowledge AI — your digital twin. A CLI-based AI agent that learns about you through conversations and can answer questions, make decisions, and write in your style.

## Features

- **Brain** — persistent knowledge store with YAML frontmatter files organized by category (identity, opinions, style, decisions, knowledge)
- **Semantic Search** — BM25 full-text search with chunking, pseudo-relevance feedback, and relevance scoring (recency decay, access frequency, confidence)
- **Dashboard** — web UI with chat interface, brain exploration, and sync status monitoring
- **Streaming Chat** — real-time token-by-token responses with dual modes (brain-augmented QA or free LLM chat)
- **Multi-Provider** — supports Claude (Anthropic) and OpenAI, with configurable timeouts and custom base URLs (e.g. Ollama)
- **Git Sync** — Git integration for brain persistence and version control across systems
- **MCP Server** — Model Context Protocol server with stdio and HTTP transports for integration with Claude Code, Cursor, and other MCP clients
- **Teaching Mode** — interactive mode where you tell kai about yourself; it extracts structured learnings and saves them to the brain

## Installation

### Option 1: Pre-built Binary (Recommended)

**macOS (via Homebrew - coming soon):**
```bash
brew install tinhvqbk/kai/kai
```

**macOS / Linux (via curl):**
```bash
curl -fsSL https://raw.githubusercontent.com/tinhvqbk/kai/main/install.sh | bash
```

Or manually download from [GitHub Releases](https://github.com/tinhvqbk/kai/releases):
```bash
# macOS Intel
curl -L https://github.com/tinhvqbk/kai/releases/download/v1.0.0/kai-darwin-amd64 -o /usr/local/bin/kai
chmod +x /usr/local/bin/kai

# macOS Apple Silicon
curl -L https://github.com/tinhvqbk/kai/releases/download/v1.0.0/kai-darwin-arm64 -o /usr/local/bin/kai
chmod +x /usr/local/bin/kai

# Linux x86_64
curl -L https://github.com/tinhvqbk/kai/releases/download/v1.0.0/kai-linux-amd64 -o /usr/local/bin/kai
chmod +x /usr/local/bin/kai
```

### Option 2: Build from Source

```bash
# Clone and build
git clone https://github.com/tinhvqbk/kai-code.git
cd kai-code
make build

# Set your API key
./kai auth set-key --provider claude

# Start chatting
./kai
```

### Option 3: Go Install

```bash
go install github.com/tinhvqbk/kai/cmd/kai@latest
```

## Quick Start

```bash
# Set up config
cp config.yaml.example config.yaml
# Edit config.yaml with your provider and API key

# Set your API key (if not already done)
kai auth set-key --provider claude

# Start chatting
kai

# Or ask a one-shot question
kai ask "What do I think about testing?"

# Start the dashboard
kai dashboard
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
  chunk_size: 1000
sessions_path: ./sessions
server:
  listen_addr: 0.0.0.0:8080
sync:
  git:
    enabled: false
    remote: "https://github.com/user/kai-brain.git"
    branch: main
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
| `kai brain import <files...>` | Import .md, .csv, or .xlsx files into brain |
| `kai brain reorganize` | Deduplicate and consolidate brain files |
| `kai reindex` | Rebuild BM25 and vector search indexes |
| `kai dashboard` | Start dashboard web UI (http://localhost:8080) |
| `kai sync` | Sync brain with configured backends (Git) |
| `kai auth set-key` | Save an API key |
| `kai auth status` | Show configured providers |
| `kai mcp` | Start MCP server (stdio) |
| `kai mcp --transport http` | Start MCP server (HTTP with bearer auth) |
| `kai version` | Print version info |

## Importing Knowledge

You can import markdown and CSV files directly into your brain via **CLI or Dashboard** with optional auto-tagging:

### CLI Import

```bash
# Import a single markdown file
kai brain import notes.md

# Import with auto-extracted tags
kai brain import notes.md --auto-tag

# Import CSV files (automatically converts to markdown tables)
kai brain import data.csv

# Import Excel files (all sheets converted to markdown tables)
kai brain import spreadsheet.xlsx

# Multiple files with auto-tag and manual tags
kai brain import *.md *.csv *.xlsx --auto-tag --tags "important,project-x" --confidence high

# Auto-tag + auto-chunk for large files
kai brain import book.md --auto-tag --auto-chunk
```

### Dashboard Import

Use the **Import** page in the dashboard for drag-and-drop file uploads:
1. Navigate to the **Import** tab
2. Drag files onto the upload area or click to select
3. Configure category, tags, and confidence level
4. **Enable auto-tagging** to extract tags from content
5. Click **Import Files**

**Supported formats:**
- **Markdown** (.md) — imported as-is
- **CSV** (.csv) — automatically converted to markdown table format
- **Excel** (.xlsx) — all sheets converted to markdown tables with headers

**Options:**
- `-c, --category` or **Category** — Brain subdirectory category (default: `knowledge`)
- `-t, --tags` or **Tags** — Comma-separated tags to apply (combined with auto-extracted tags)
- `-C, --confidence` or **Confidence** — Confidence level: `high`, `medium`, or `low` (default: `medium`)
- `-a, --auto-tag` or **Auto-extract tags** — Extract meaningful keywords from content (merges with manual tags)
- `-k, --auto-chunk` or **Auto-chunk** — Break large files into semantic chunks for better search

**How auto-tagging works:**
- Extracts keywords from file headers and content
- Identifies meaningful 2-word phrases
- Merges with any manually-specified tags
- Deduplicates tags automatically
- Returns top 5 most relevant tags

Files are stored in the specified category directory with metadata frontmatter and are immediately indexed for search.

## Dashboard

kai includes a web UI for interactive chat and brain management:

```bash
# Start dashboard on http://localhost:8080
kai dashboard
```

Features:
- **Chat Interface** — dual-mode chat with brain-augmented QA or free LLM responses
- **Streaming Responses** — real-time token-by-token output with smooth animations
- **Brain Explorer** — browse and search brain files
- **Sync Status** — monitor Git sync status
- **Configuration** — view and manage sync backends
- **Vector Dashboard** — 3D visualization of semantic document relationships

## Hybrid Search & Vector Embeddings

kai uses **hybrid search** combining BM25 full-text indexing with vector embeddings for better semantic understanding:

### How It Works
- **Automatic** — indexes are rebuilt automatically when you write, teach, or reorganize brain files
- **Dual providers:**
  - **LSI** (default) — corpus-derived embeddings, built-in, no external dependencies
  - **Ollama** (optional) — higher-quality vector embeddings via local Ollama service

### Manual Reindex

Rebuild indexes manually if needed:

```bash
# Rebuild BM25 and vector indexes (LSI if no Ollama)
kai reindex

# Output shows:
# ✓ Indexes rebuilt successfully
#   Documents: 42
#   Chunks: 156
#   Inverted index terms: 892
#   Vector embeddings: 156 chunks
#   Vector provider: ollama
```

### Configuration

```yaml
brain:
  embeddings:
    provider: ""           # "" = auto (try Ollama, fall back to LSI)
    ollama:
      url: http://localhost:11434
      model: nomic-embed-text    # Other options: mxbai-embed-large, all-minilm
```

**Providers:**
- Empty (default) — tries Ollama, falls back to LSI
- `"ollama"` — only use Ollama (requires service running)
- `"lsi"` — only use LSI (corpus-derived, offline)

**Note:** Vector embeddings require ≥5 documents minimum. If fewer documents exist, `kai reindex` will show why vectors weren't created.

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

### Quick Setup

```bash
# One command — creates brain, config, and registers MCP server (no install needed)
go run github.com/tinhvqbk/kai/cmd/kai@latest setup
```

### MCP Configuration

Add to your `~/.mcp.json` (or let `kai setup` do it automatically):

**Via Go module (recommended — always runs latest, no install):**

```json
{
  "mcpServers": {
    "kai": {
      "type": "stdio",
      "command": "go",
      "args": ["run", "github.com/tinhvqbk/kai/cmd/kai@latest", "mcp", "--config", "~/.kai/config.yaml"],
      "env": {}
    }
  }
}
```

**Via git repo directly (pin to a branch or commit):**

```json
{
  "mcpServers": {
    "kai": {
      "type": "stdio",
      "command": "go",
      "args": ["run", "github.com/tinhvqbk/kai/cmd/kai@main", "mcp", "--config", "~/.kai/config.yaml"],
      "env": {}
    }
  }
}
```

**Via local binary (fastest startup):**

```bash
go install github.com/tinhvqbk/kai/cmd/kai@latest
kai setup --local
```

```json
{
  "mcpServers": {
    "kai": {
      "type": "stdio",
      "command": "/path/to/kai",
      "args": ["mcp", "--config", "~/.kai/config.yaml"],
      "env": {}
    }
  }
}
```

### HTTP

```bash
export KAI_MCP_TOKEN="your-secret-token"
kai mcp --transport http --addr 0.0.0.0:8080
```

## Git Sync

kai can sync your brain across systems using Git:

```bash
# Sync brain with git
kai sync
```

**Git**: Uses a dedicated brain repository branch for version control and multi-device sync.

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
