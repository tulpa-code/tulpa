# Tulpa Development Guide

## Build & Development Commands

```bash
# Build the application
task build
go build .

# Run with CLI args
task run -- --help
go run . --help

# Run tests
task test
go test ./...

# Run specific test
go test -run TestRegexCache ./internal/llm/tools/

# Lint and format
task lint
task lint-fix
task fmt

# Install dependencies
task install
go install .

# Development with profiling
task dev
TULPA_PROFILE=true go run .
```

## Code Style Guidelines

### General
- Go 1.25.0+ with green tea GC experiment enabled
- Use `gofumpt` for formatting (stricter than gofmt)
- Import organization: stdlib, third-party, internal packages
- Use table-driven tests with `t.Parallel()` when possible

### Testing
- Use `github.com/stretchr/testify/require` for assertions
- Test files end with `_test.go`
- Use `t.Helper()` in helper functions
- Benchmarks use `b.Loop()` instead of `for i := 0; i < b.N; i++`

### Error Handling
- Wrap errors with context using `fmt.Errorf`
- Use structured logging with `log/slog`
- Check error variables explicitly, don't ignore returns

### Naming & Types
- Use PascalCase for exported, camelCase for unexported
- Interface names end in `-er` (e.g., `Service`, `Client`)
- Use generics for type safety where applicable
- Context should be first parameter: `ctx context.Context`

### Concurrency
- Use `sync.WaitGroup` for goroutine coordination
- Prefer channels for communication
- Use `csync.Map` for concurrent-safe maps
- Use `testing/synctest` for time-sensitive tests

### Project Structure
- Main entry point in `main.go`
- Internal packages in `internal/`
- Use dependency injection pattern
- Separate concerns (cmd, app, domain, infrastructure)