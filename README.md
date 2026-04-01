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

![kai_demo](https://github.com/user-attachments/assets/c81bc457-3faf-4534-bb36-c9b9ffb3d5e9)


> Replace this with a real GIF showing: teach → ask → answer

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

kai auth set-key --provider claude
kai setup        # registers as MCP server for Claude Code
kai              # start chatting
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

Copy the example config and set your provider:

```bash
cp config.yaml.example config.yaml
```

```yaml
# config.yaml
provider: claude          # claude | openai | gemini | ollama
brain:
  path: ./brain
  max_context_files: 5
```

Set your API key:

```bash
kai auth set-key --provider claude
kai auth status            # verify
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
