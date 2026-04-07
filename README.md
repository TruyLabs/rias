# KAI

![License](https://img.shields.io/badge/license-MIT-green)
![CLI](https://img.shields.io/badge/interface-CLI-blue)
![AI](https://img.shields.io/badge/powered_by-AI-purple)
![Go](https://img.shields.io/badge/built_with-Go-00ADD8)
![MCP](https://img.shields.io/badge/MCP-supported-black)
![Status](https://img.shields.io/badge/status-active-success)

> Your digital twin — powered by knowledge, not prompts.

---

## 🚀 Demo

<img width="800" height="400" alt="image" src="https://github.com/user-attachments/assets/474d9684-e570-48fb-9173-d42b9adebd4f" />


---

## Why KAI?

AI today is powerful — but forgetful.

- Every conversation starts from zero
- No memory of who you are
- No consistency in decisions

KAI changes that.

It gives AI a **persistent brain** that learns, remembers, and evolves with you.

---

## What it feels like

```bash
kai teach
# "I prefer TDD for business logic"
```

```bash
kai ask "How should I design this system?"
```

→ KAI answers based on what it knows about you.

---

## ⚡ Quick Start

```bash
brew tap norenis/kai && brew install kai

kai setup                          # creates ~/.kai/, generates config, registers MCP server
kai auth set-key --provider claude # save your API key
kai                                # start chatting
```

---

## 🧠 Core Concept: The Brain

```
brain/
├── identity/    # who you are
├── opinions/    # what you believe
├── style/       # how you write and code
├── decisions/   # choices you've made
└── knowledge/   # what you know
```

Brain files are markdown with YAML frontmatter. KAI appends to them as it learns — it never overwrites.

---

## 🛠 Configuration

### Config file location

kai looks for config in this order:

1. `--config <path>` flag (any command)
2. `./config.yaml` in the current directory (local/dev override)
3. `~/.kai/config.yaml` ← **created automatically by `kai setup`**

### Quick provider setup

```bash
kai setup                            # generates ~/.kai/config.yaml
kai auth set-key --provider claude   # stores key in ~/.kai/credentials.json
kai auth status                      # verify all configured providers
```

### Switching providers

Edit `~/.kai/config.yaml` and change the `provider:` line:

```yaml
provider: openai   # claude | openai | gemini | ollama
```

### Provider configuration

<details>
<summary><strong>Claude (Anthropic)</strong></summary>

```yaml
provider: claude
providers:
  claude:
    auth: api_key
    model: claude-sonnet-4-6-20250514
```

```bash
kai auth set-key --provider claude
# Paste your key from https://console.anthropic.com
```
</details>

<details>
<summary><strong>OpenAI</strong></summary>

```yaml
provider: openai
providers:
  openai:
    auth: api_key
    model: gpt-4o
```

```bash
kai auth set-key --provider openai
# Paste your key from https://platform.openai.com
```
</details>

<details>
<summary><strong>Gemini (Google)</strong></summary>

```yaml
provider: gemini
providers:
  gemini:
    auth: api_key
    model: gemini-1.5-pro
```

```bash
kai auth set-key --provider gemini
# Paste your key from https://aistudio.google.com
```
</details>

<details>
<summary><strong>Ollama (local, no API key)</strong></summary>

```yaml
provider: openai
providers:
  openai:
    base_url: http://localhost:11434/v1
    model: llama3
```

No API key needed. Start Ollama first: `ollama serve`
</details>

### Full config reference

```yaml
agent:
  name: kai           # display name shown in chat
  user_name: Kyle     # your name — used in prompts and learning

provider: claude      # active provider: claude | openai | gemini | ollama

providers:
  claude:
    auth: api_key
    model: claude-sonnet-4-6-20250514
    # base_url: https://api.anthropic.com   # optional: custom endpoint
    # timeout_sec: 120
  openai:
    auth: api_key
    model: gpt-4o
    # base_url: https://api.openai.com      # optional: change for Ollama etc.
    # timeout_sec: 120

brain:
  path: ~/.kai/brain          # where brain files are stored
  max_context_files: 5        # how many brain files to inject per query
  embeddings:
    provider: ""              # "lsi" (built-in) | "ollama" | "" (auto)
    ollama:
      url: http://localhost:11434
      model: nomic-embed-text
  sync:
    git:
      enabled: false
      remote: git@github.com:you/kai-brain.git
      branch: main

sessions_path: ~/.kai/sessions

server:
  listen_addr: 0.0.0.0:8080
  dashboard_pin: ""           # set a PIN to protect the dashboard
```

---

## 🔥 Features

- Persistent memory (brain)
- Hybrid search (BM25 + vector)
- CLI-first workflow
- MCP integration
- Dashboard UI

---

## 📦 Installation

### Homebrew (macOS / Linux) — recommended

```bash
brew tap norenis/kai
brew install kai
```

### Install script

```bash
curl -fsSL https://raw.githubusercontent.com/norenis/kai/main/install.sh | bash
```

### Build from source

```bash
git clone https://github.com/norenis/kai
cd kai
make install
```

### After installation

Initialize your `~/.kai/` directory and register as an MCP server:

```bash
kai setup
```

This creates `~/.kai/brain/`, `~/.kai/sessions/`, a default `~/.kai/config.yaml`, and registers kai in `~/.mcp.json` for Claude Code.

---

## 🧪 Commands

### Core

| Command | Description |
|---------|-------------|
| `kai` | Start an interactive multi-turn chat session |
| `kai ask <question>` | Ask a one-shot question (no session history) |
| `kai teach` | Teach KAI something about yourself |
| `kai dashboard` | Launch the brain explorer web UI (auto-opens browser) |
| `kai version` | Print version, commit, and build date |

### Brain

| Command | Description |
|---------|-------------|
| `kai brain` | List all brain files with tags and confidence |
| `kai brain search <query>` | Full-text BM25 search across brain files |
| `kai brain edit <file>` | Open a brain file in `$EDITOR` |
| `kai brain import <files...>` | Import `.md`, `.csv`, or `.xlsx` files into brain |
| `kai brain reorganize` | Analyze brain for duplicates, miscategorizations, and small files (dry-run by default; use `--apply` to execute) |
| `kai brain reorganize dedup` | Find and merge duplicate files |
| `kai brain reorganize recategorize` | Move files to the correct category |
| `kai brain reorganize consolidate` | Merge small related files |
| `kai reindex` | Rebuild BM25 and vector search indexes manually |

**Import flags:**

```bash
kai brain import notes.md \
  --category knowledge \     # brain subdirectory (default: knowledge)
  --tags "go,testing" \      # comma-separated tags
  --confidence high \        # high | medium | low (default: medium)
  --auto-tag \               # extract tags from content automatically
  --auto-chunk               # chunk large files for better search
```

### Auth

| Command | Description |
|---------|-------------|
| `kai auth set-key --provider <name>` | Save an API key for a provider |
| `kai auth status` | Show configured vs. unconfigured providers |

### Sync

| Command | Description |
|---------|-------------|
| `kai sync init` | Initialize git sync for the brain |
| `kai sync push` | Push brain changes to remote |
| `kai sync pull` | Pull brain from remote and rebuild index |
| `kai sync status` | Show local vs. remote diff |

Enable git sync in `config.yaml`:

```yaml
brain:
  sync:
    git:
      enabled: true
      remote: git@github.com:you/brain.git
      branch: main
```

### MCP

| Command | Description |
|---------|-------------|
| `kai mcp` | Start MCP server over stdio (default) |
| `kai mcp --transport http --addr :8081` | Start MCP server over HTTP (requires `KAI_MCP_TOKEN`) |
| `kai setup` | One-time setup: create brain, config, and register with Claude Code |

---

## 🔌 MCP Integration

KAI exposes all brain operations as MCP tools — meaning Claude Code, Cursor, and VS Code can read and write your brain directly.

### Claude Code (one command)

```bash
kai setup
```

That's it. Restart Claude Code and KAI appears as an MCP server.

> Use `kai setup --local` to register the installed binary instead of `go run`.

### Cursor

Add to `~/.cursor/mcp.json` (global) or `.cursor/mcp.json` (per-project):

```json
{
  "mcpServers": {
    "kai": {
      "type": "stdio",
      "command": "kai",
      "args": ["mcp"]
    }
  }
}
```

Restart Cursor. KAI tools will appear in the MCP panel.

### VS Code

VS Code MCP support is available via extensions (Cline, Continue, or GitHub Copilot with MCP enabled). Add to your extension's MCP config:

```json
{
  "mcpServers": {
    "kai": {
      "type": "stdio",
      "command": "kai",
      "args": ["mcp"]
    }
  }
}
```

Refer to your extension's documentation for the exact config file location.

---

## 🧬 Philosophy

> AI should remember who you are.

---

## 📄 License

MIT
