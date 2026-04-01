Add a new LLM provider to kai named `$ARGUMENTS`.

Steps:

1. Read `internal/provider/types.go` — the `Provider` interface has two methods:
   - `Chat(ctx, systemPrompt, messages, opts...) (*Response, error)` — single response
   - `Stream(ctx, systemPrompt, messages, opts...) (<-chan Chunk, error)` — streaming

2. Read `internal/provider/claude.go` for a reference implementation, including how to use `ApplyOptions(opts)` to resolve `CallOptions`

3. Read `internal/provider/registry.go` to understand `Registry.Register(name, provider)`

4. Create `internal/provider/$ARGUMENTS.go` implementing the `Provider` interface

5. Find where providers are registered (grep for `Registry` in `internal/cli/`) and register the new provider there

6. Add config struct in `internal/config/config.go` under the `Providers map[string]ProviderConfig`

7. Add example config block to `config.yaml.example`

8. Write tests in `internal/provider/${ARGUMENTS}_test.go` using `httptest.NewServer` — never use real API keys

9. Run `make build && go test ./internal/provider/...` to verify

Show me the Provider interface and an existing implementation before writing any code.
