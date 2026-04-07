# kai — Knowledge AI Agent

## Project Overview

**kai** is a CLI-based personal AI knowledge system ("digital twin") written in Go. It learns about the user through conversations, stores knowledge in structured markdown brain files, and exposes that knowledge via CLI, web dashboard, and MCP server.

## Build & Test

```bash
make build          # Build ./kai binary (with version ldflags)
make install        # Install to $GOPATH/bin
make test           # go test ./... -v
make clean          # Remove ./kai binary

go test ./internal/router/... -v    # Test specific package
go test -run TestName ./...         # Run specific test
go vet ./...                         # Lint
```

The binary is always named `kai`. Run it as `./kai` locally after building.

## Architecture

```
cmd/kai/main.go         → cli.Execute()
internal/
  cli/                  → Cobra subcommands (root, ask, teach, brain, mcp, dashboard, sync, auth, setup, reindex)
  router/               → Pipeline: retrieval → prompt → LLM → learning extraction
  brain/                → Brain file storage, BM25 indexing, vector embeddings, chunking
  retriever/            → Hybrid search (BM25 + vector)
  prompt/               → System/user prompt construction with brain context
  provider/             → LLM integrations: claude.go, openai.go, gemini.go
  config/               → YAML config loader (config.yaml)
  auth/                 → API key management (~/.kai/credentials.json)
  session/              → Multi-turn conversation history
  mcp/                  → MCP protocol server (stdio + HTTP transport)
  dashboard/            → Web UI (port 8080): chat, brain explorer, sync status
  sync/                 → Git-based brain sync
```

**Data flow:** `User → Router → Retriever (brain search) → Prompt Builder → Provider (LLM) → Response + Learning Extraction → Brain update`

## Brain Structure

Brain files are markdown with YAML frontmatter in `~/.kai/brain/` (default, configurable via `brain.path`):
```
brain/
  identity/   # Personal info, background
  opinions/   # Beliefs, preferences
  style/      # Writing/coding patterns
  decisions/  # Architectural & life decisions
  knowledge/  # Domain expertise
  index.json          # BM25 inverted index
  vectors.bin.gz      # Vector embeddings
```

Frontmatter format:
```markdown
---
tags: [go, testing]
confidence: high
source: conversation
updated: 2026-03-31
---
Content here.
```

## Configuration

- `config.yaml` — Main config (copy from `config.yaml.example`)
- `~/.kai/credentials.json` — API keys (managed by `kai auth set-key`)
- Providers: `claude`, `openai`, `gemini`, `ollama` (set `provider:` in config)
- Embeddings: `lsi` (built-in) or `ollama` (requires running Ollama)

## Key Conventions

- **No mocks for integration tests** — tests in `router` and `brain` packages use real file I/O
- **Streaming responses** — all providers use streaming; never buffer full responses
- **Brain files are append-friendly** — learning extraction adds to existing files, never overwrites
- **MCP tools mirror CLI** — every brain operation available in CLI must be exposed as MCP tool
- **Config is loaded once** — pass `*config.Config` through; don't reload mid-request
- **Provider interface** is in `internal/provider/types.go` — new providers implement `Provider` (methods: `Chat`, `Stream`)

## Adding a New LLM Provider

The `Provider` interface is defined in `internal/provider/types.go`:
```go
type Provider interface {
    Chat(ctx context.Context, systemPrompt string, messages []Message, opts ...Option) (*Response, error)
    Stream(ctx context.Context, systemPrompt string, messages []Message, opts ...Option) (<-chan Chunk, error)
}
```
Use `ApplyOptions(opts)` inside your implementation to resolve `CallOptions`.

1. Create `internal/provider/<name>.go` implementing the `Provider` interface
2. Register it via `Registry.Register(name, provider)` — see how existing providers are wired in `buildRouter` in `internal/cli/`
3. Add provider config in `internal/config/config.go` under the `Providers` map (`map[string]ProviderConfig`)
4. Update `config.yaml.example` with example settings
5. Test with `go test ./internal/provider/...` using `httptest.NewServer` to mock the API

## Adding a New CLI Command

All command constructors are **unexported** and take no parameters. Config is loaded inside `RunE` via `loadConfig()`.

1. Create `internal/cli/<name>.go` with a `new<Name>Cmd()` func returning `*cobra.Command`
2. Follow the pattern: call `loadConfig()` at the top of `RunE`, then `buildRouter(cfg)` if LLM access is needed
3. Register in `internal/cli/root.go` via `root.AddCommand(new<Name>Cmd())`
4. Only `NewRootCmd()` is exported — keep new command constructors unexported

## Testing

- Unit tests: `*_test.go` alongside source files
- E2e tests: `tests/e2e/` (Playwright, requires `./kai dashboard` running on port 9234)
- Run e2e: `pnpm exec playwright test`

## Release

Built via GitHub Actions (`.github/workflows/build-release.yml`). Releases are triggered by pushing a `v*` tag. Binaries are built for darwin/linux/arm64/amd64.
