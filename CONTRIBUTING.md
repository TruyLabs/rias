# Contributing to kai

## Development Setup

```bash
git clone https://github.com/norenis/kai.git
cd kai
cp config.yaml.example config.yaml
# Edit config.yaml with your API key and provider
make build
./kai version
```

## Running Tests

```bash
make test               # All tests
go test ./internal/... -v -run TestName   # Specific test
go vet ./...            # Linting
```

Integration tests use real file I/O — no mocks for brain or router packages.

## Project Structure

```
cmd/kai/          Entry point
internal/
  brain/          Knowledge storage, BM25 indexing, vector embeddings
  cli/            Cobra subcommands
  config/         YAML config loader
  dashboard/      Web UI
  mcp/            MCP protocol server
  provider/       LLM integrations (claude, openai, gemini)
  retriever/      Hybrid search
  router/         Conversation pipeline
  session/        Chat history
  sync/           Git brain sync
```

See `CLAUDE.md` for architecture details, conventions, and guides for adding providers or commands.

## Pull Requests

- Keep changes focused — one thing per PR
- Add tests for new behaviour
- Run `make test` and `go vet ./...` before opening a PR
- Update `CLAUDE.md` if you change architecture or conventions

## Releasing

Push a version tag to trigger the build workflow:

```bash
git tag v1.2.3
git push origin v1.2.3
```

GitHub Actions builds binaries for macOS (Intel + Apple Silicon) and Linux, then creates the release automatically.
