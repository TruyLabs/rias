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

## Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/tinhvqbk/kai/main/install.sh | bash
```

The script auto-detects your platform (macOS Intel/Apple Silicon, Linux) and installs to `/usr/local/bin/kai`.

## Manual Download

Download binaries from the [Releases](https://github.com/tinhvqbk/kai/releases) page:

- `kai-darwin-amd64` — macOS Intel
- `kai-darwin-arm64` — macOS Apple Silicon
- `kai-linux-amd64` — Linux x86_64

### Installation
```bash
curl -L https://github.com/tinhvqbk/kai/releases/download/v1.0.0/kai-linux-amd64 -o /usr/local/bin/kai
chmod +x /usr/local/bin/kai
```

## Verify Integrity

Each release includes SHA256 checksums. Verify your download:

```bash
# Download binary and checksum
curl -L https://github.com/tinhvqbk/kai/releases/download/v1.0.0/kai-linux-amd64 -o kai
curl -L https://github.com/tinhvqbk/kai/releases/download/v1.0.0/kai-linux-amd64.sha256 -o kai.sha256

# Verify
sha256sum -c kai.sha256
```

## Usage

```bash
# Start interactive chat
kai

# Ask a question
kai ask "What do I think about testing?"

# Start dashboard
kai dashboard

# View help
kai --help
```

## Configuration

Create `config.yaml` to set your LLM provider:

```yaml
provider: claude
providers:
  claude:
    auth: api_key
    model: claude-sonnet-4-6-20250514
brain:
  path: ./brain
  max_context_files: 10
server:
  listen_addr: 0.0.0.0:8080
```

Set your API key:
```bash
kai auth set-key --provider claude
```

## License

MIT
