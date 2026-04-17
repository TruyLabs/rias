# rias

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

## 💡 Why rias?

AI today is powerful — but forgetful.

- Every conversation starts from zero
- No memory of who you are
- No consistency in decisions

rias changes that.

It gives AI a **persistent brain** that learns, remembers, and evolves with you.

---

## ✨ What it feels like

```bash
rias teach
# "I prefer TDD for business logic"
```

```bash
rias ask "How should I design this system?"
```

→ rias answers based on what it knows about you.

---

## ⚡ Quick Start

```bash
brew tap norenis/kai && brew install kai

rias setup                          # creates ~/.rias/, generates config, registers MCP server
rias auth set-key --provider claude # save your API key
rias                                # start chatting
```

---

## 🧠 The Brain

```
brain/
├── identity/    # who you are
├── opinions/    # what you believe
├── style/       # how you write and code
├── decisions/   # choices you've made
├── knowledge/   # what you know
├── tasks/       # today's tasks
├── goals/       # short, medium, and long-term goals
├── expertise/   # your expertise map
└── feedback/    # session quality ratings
```

Brain files are markdown with YAML frontmatter. rias appends to them as it learns — it never overwrites.

---

## 🛠 Configuration

### Config file location

rias looks for config in this order:

1. `--config <path>` flag (any command)
2. `./config.yaml` in the current directory (local/dev override)
3. `~/.rias/config.yaml` ← **created automatically by `rias setup`**

### Quick provider setup

```bash
rias setup                            # generates ~/.rias/config.yaml
rias auth set-key --provider claude   # stores key in ~/.rias/credentials.json
rias auth status                      # verify all configured providers
```

### Switching providers

Edit `~/.rias/config.yaml` and change the `provider:` line:

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
rias auth set-key --provider claude
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
rias auth set-key --provider openai
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
rias auth set-key --provider gemini
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
  name: rias          # display name shown in chat
  user_name: Kyle     # your name — used in prompts and learning
  proactive_recall: false  # surface related memories even when not directly asked

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
  path: ~/.rias/brain          # where brain files are stored
  max_context_files: 5         # how many brain files to inject per query
  embeddings:
    provider: ""               # "lsi" (built-in) | "ollama" | "" (auto)
    ollama:
      url: http://localhost:11434
      model: nomic-embed-text
  sync:
    git:
      enabled: false
      remote: git@github.com:you/rias-brain.git
      branch: main

sessions_path: ~/.rias/sessions

server:
  listen_addr: 0.0.0.0:8080
  dashboard_pin: ""           # set a PIN to protect the dashboard
```

---

## 🔥 Features

- Persistent memory (brain) across all conversations
- Self-improvement: `reflect`, `expertise`, `feedback`, and `brain scan` keep the brain accurate over time
- Goal tracking with short/medium/long horizons
- Hybrid search (BM25 + vector) with incremental indexing
- 3D vector graph with hover highlighting and ANN link-building (scales to 5000+ nodes)
- Task management — CLI, dashboard, and MCP
- CLI-first workflow
- MCP integration with Claude Code slash commands (21 tools)
- Module system (GitHub PRs, Google Sheets, extensible)
- Dashboard UI with collapsible sidebar, plugin management, and sync controls
- Git-based brain sync
- Proactive recall — surfaces related memories even when not directly asked

---

## 📦 Installation

### Homebrew (macOS / Linux) — recommended

```bash
brew tap norenis/kai
brew install kai
```

### Install script

```bash
curl -fsSL https://raw.githubusercontent.com/TruyLabs/rias/main/install.sh | bash
```

### Build from source

```bash
git clone https://github.com/TruyLabs/rias
cd rias
make install
```

### After installation

Initialize your `~/.rias/` directory and register as an MCP server:

```bash
rias setup
```

This creates `~/.rias/brain/`, `~/.rias/sessions/`, a default `~/.rias/config.yaml`, registers rias in `~/.mcp.json` for Claude Code, and installs `/rias:*` slash commands.

---

## 🧪 Commands

### Core

| Command | Description |
|---------|-------------|
| `rias` | Start an interactive multi-turn chat session |
| `rias ask <question>` | Ask a one-shot question (no session history) |
| `rias teach` | Teach rias something about yourself |
| `rias dashboard` | Launch the brain explorer web UI (auto-opens browser) |
| `rias version` | Print version, commit, and build date |

### Brain

| Command | Description |
|---------|-------------|
| `rias brain` | List all brain files with tags and confidence |
| `rias brain search <query>` | Full-text BM25 search across brain files |
| `rias brain edit <file>` | Open a brain file in `$EDITOR` |
| `rias brain import <files...>` | Import `.md`, `.csv`, or `.xlsx` files into brain |
| `rias brain reorganize` | Analyze brain for duplicates, miscategorizations, and small files (dry-run by default; use `--apply` to execute) |
| `rias brain reorganize dedup` | Find and merge duplicate files |
| `rias brain reorganize recategorize` | Move files to the correct category |
| `rias brain reorganize consolidate` | Merge small related files |
| `rias brain scan` | Detect contradictions between brain entries (requires LLM) |
| `rias brain decay` | Downgrade confidence of stale entries |
| `rias brain migrate` | Recategorize brain files using LLM suggestions |
| `rias reindex` | Rebuild BM25 and vector search indexes manually |

**Import flags:**

```bash
rias brain import notes.md \
  --category knowledge \     # brain subdirectory (default: knowledge)
  --tags "go,testing" \      # comma-separated tags
  --confidence high \        # high | medium | low (default: medium)
  --auto-tag \               # extract tags from content automatically
  --auto-chunk               # chunk large files for better search
```

### 🎯 Goals

| Command | Description |
|---------|-------------|
| `rias goal` | List all goals |
| `rias goal list` | List all goals with horizon and status |
| `rias goal add <text>` | Add a new goal |
| `rias goal done <index>` | Mark a goal as done |

```bash
rias goal add "Ship the MCP server" --horizon short   # short | medium | long
rias goal add "Learn Rust"                             # defaults to medium
rias goal done 0
```

### 🪞 Self-Improvement

| Command | Description |
|---------|-------------|
| `rias reflect` | Analyze session history to extract behavioral patterns into brain |
| `rias expertise` | Show your current expertise map |
| `rias expertise --update` | Regenerate expertise map from all brain files (requires LLM) |
| `rias feedback good` | Rate the last session response as good |
| `rias feedback bad --note "..."` | Rate the last session response as bad, with a note |

```bash
rias reflect              # analyze all sessions
rias reflect --since 7d   # only the last 7 days
rias reflect --since 2w   # only the last 2 weeks
```

### ✅ Tasks

| Command | Description |
|---------|-------------|
| `rias task` | List today's tasks |
| `rias task add <text>` | Add a task for today |
| `rias task done <id>` | Mark a task as done |
| `rias task undone <id>` | Mark a task as not done |
| `rias task rm <id>` | Remove a task |

Tasks are stored as brain files and are also accessible from the dashboard and via MCP.

### 📥 Import

| Command | Description |
|---------|-------------|
| `rias import-history` | Import conversation history from Claude or ChatGPT exports |
| `rias index-repo <path>` | Index a code repository into brain knowledge |

```bash
rias import-history --provider claude  --file ~/Downloads/claude-export.json
rias import-history --provider chatgpt --file ~/Downloads/conversations.json
rias index-repo ~/code/myproject
```

### 🔑 Auth

| Command | Description |
|---------|-------------|
| `rias auth set-key --provider <name>` | Save an API key for a provider |
| `rias auth status` | Show configured vs. unconfigured providers |

### 🔌 Modules (Plugins)

| Command | Description |
|---------|-------------|
| `rias module list` | List available modules and their enabled status |
| `rias module <name>` | Run a specific module (e.g. `rias module github_prs`) |
| `rias module --all` | Run all enabled modules |

Modules pull external data into the brain. Configure them in `config.yaml`:

```yaml
modules:
  - name: github_prs
    enabled: true
    config:
      token: ${GITHUB_TOKEN}
      repos:
        - owner/repo
      state: open
      limit: 20

  - name: google_sheets
    enabled: true
    config:
      api_key: ${GOOGLE_API_KEY}
      spreadsheet_id: "your-sheet-id"
      range: "Sheet1!A:Z"
      category: knowledge
      topic: my-sheet
```

Built-in modules:

| Module | Description |
|--------|-------------|
| `github_prs` | Fetch pull requests from GitHub repositories |
| `google_sheets` | Read a Google Sheet into the brain |

### 🔄 Sync

| Command | Description |
|---------|-------------|
| `rias sync init` | Initialize git sync for the brain |
| `rias sync push` | Push brain changes to remote |
| `rias sync pull` | Pull brain from remote and rebuild index |
| `rias sync status` | Show local vs. remote diff |

Enable git sync in `config.yaml`:

```yaml
brain:
  sync:
    git:
      enabled: true
      remote: git@github.com:you/brain.git
      branch: main
```

### 🤖 MCP

| Command | Description |
|---------|-------------|
| `rias mcp` | Start MCP server over stdio (default) |
| `rias mcp --transport http --addr :8081` | Start MCP server over HTTP (requires `RIAS_MCP_TOKEN`) |
| `rias setup` | One-time setup: create brain, config, and register with Claude Code |

---

## 🔌 MCP Integration

rias exposes brain operations as MCP tools — Claude Code, Cursor, and VS Code can read and write your brain directly.

### Claude Code (one command)

```bash
rias setup
```

That's it. Restart Claude Code and rias appears as an MCP server.

> Use `rias setup --local` to register the installed binary instead of `go run`.

### Cursor

Add to `~/.cursor/mcp.json` (global) or `.cursor/mcp.json` (per-project):

```json
{
  "mcpServers": {
    "rias": {
      "type": "stdio",
      "command": "rias",
      "args": ["mcp"]
    }
  }
}
```

Restart Cursor. rias tools will appear in the MCP panel.

### VS Code

VS Code MCP support is available via extensions (Cline, Continue, or GitHub Copilot with MCP enabled). Add to your extension's MCP config:

```json
{
  "mcpServers": {
    "rias": {
      "type": "stdio",
      "command": "rias",
      "args": ["mcp"]
    }
  }
}
```

### MCP Tools

| Tool | Description |
|------|-------------|
| `brain_list` | List all brain files with tags and confidence |
| `brain_read` | Read a brain file's content and metadata |
| `brain_write` | Write or update a brain file directly |
| `brain_search` | Full-text BM25 search across brain files |
| `brain_reorganize` | Analyze and reorganize brain files |
| `ask` | Ask rias a question using brain context |
| `teach` | Teach rias something (LLM or direct mode) |
| `tasks` | Manage today's task list |
| `module_list` | List available modules |
| `module_run` | Run a module to import external data |
| `setup_commands` | Get slash command files for Claude Code |

### Claude Code Slash Commands

After `rias setup`, these slash commands are available in Claude Code:

| Command | Description |
|---------|-------------|
| `/rias:ask <question>` | Ask rias using brain context |
| `/rias:teach <input>` | Teach rias something new |
| `/rias:brain-list` | List all brain files |
| `/rias:brain-read <path>` | Read a brain file |
| `/rias:brain-search <query>` | Search brain by keywords |
| `/rias:brain-write` | Write or update a brain file |
| `/rias:brain-reorganize` | Analyze brain for reorganization |
| `/rias:module-list` | List available plugins |
| `/rias:module-run <name>` | Run a plugin module |

These are installed to `~/.claude/commands/rias/` automatically during setup.

---

## Performance

### Vector graph link-building

The graph uses a projection-sort sliding window (ANN) instead of all-pairs cosine comparison.

| Nodes | Old O(n²) comparisons | New O(n×W) comparisons | Speedup |
|------:|----------------------:|----------------------:|--------:|
| 100   | 4,950                 | 8,000                 | —       |
| 500   | 124,750               | 40,000                | ~3×     |
| 1,000 | 499,500               | 80,000                | ~6×     |
| 5,000 | 12,497,500            | 400,000               | ~31×    |

W = 80 (topK × 20). Node cap raised from 500 → 5,000.

### Incremental indexing

Only modified brain files are re-embedded on each run. A manifest tracks per-file hashes so unchanged files are skipped entirely.

| Brain size | Full reindex | Incremental (1 file changed) |
|-----------:|-------------:|-----------------------------:|
| 50 files   | ~8s          | ~0.2s                        |
| 200 files  | ~30s         | ~0.2s                        |
| 1,000 files| ~150s        | ~0.2s                        |

*Times are approximate and depend on embedding provider (Ollama local vs. API).*

---

## Comparison

| Feature | rias | mem0 | Notion AI | ChatGPT Memory |
|---------|:----:|:----:|:---------:|:--------------:|
| Runs locally / offline | ✅ | ❌ | ❌ | ❌ |
| Open source | ✅ | ✅ | ❌ | ❌ |
| Structured brain files (plain markdown) | ✅ | ❌ | ❌ | ❌ |
| BM25 + vector hybrid search | ✅ | ✅ | ❌ | ❌ |
| 3D knowledge graph | ✅ | ❌ | ❌ | ❌ |
| MCP server (Claude Code, Cursor, VS Code) | ✅ | ❌ | ❌ | ❌ |
| Multi-provider LLM (Claude, OpenAI, Gemini, Ollama) | ✅ | ✅ | ❌ | ❌ |
| Git-based brain sync | ✅ | ❌ | ❌ | ❌ |
| Task management | ✅ | ❌ | ✅ | ❌ |
| Goal tracking | ✅ | ❌ | ✅ | ❌ |
| Self-improvement (reflect, expertise, feedback) | ✅ | ❌ | ❌ | ❌ |
| External data modules (GitHub, Sheets, …) | ✅ | ❌ | ❌ | ❌ |
| No subscription required | ✅ | ❌ | ❌ | ❌ |

---

## Philosophy

> AI should remember who you are.

---

## License

MIT
