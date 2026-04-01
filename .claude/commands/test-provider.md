Test a specific LLM provider integration for kai.

Provider to test: `$ARGUMENTS`

Steps:
1. Read `internal/provider/$ARGUMENTS.go` to understand the implementation
2. Read `internal/provider/${ARGUMENTS}_test.go` if it exists
3. Run the provider tests: `go test ./internal/provider/... -run $ARGUMENTS -v`
4. If tests don't exist, create them using `httptest.NewServer` to mock the API — never use real API keys in tests
5. Also run a quick build check: `make build && ./kai auth status`

Show any test failures with full output and suggest fixes.
