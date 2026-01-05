# AGENTS.md

Development guide for agentic coding agents working in the Handeln repository.

## Project Overview

**Handeln** is an AI Agent Toolbox built in Go, organized as a Go workspace with two modules:
- Root module (`gosuda.org/handeln`) - intended for higher-level agent logic
- `koppel` module (`gosuda.org/koppel`) - core AI provider abstractions and chat logic

The project uses an interface-driven architecture to abstract multiple AI model providers (OpenAI, Anthropic, Gemini) into a unified interface.

## Build/Test Commands

This is a standard Go project using the built-in Go toolchain. All commands should be run from the project root unless specified otherwise.

### Core Commands
```bash
# Build all packages in workspace
go build ./...

# Run all tests across workspace
go test ./...

# Run tests for specific package
go test ./koppel/provider/openai/...

# Run single test function
go test -run TestToChatParams ./koppel/provider/openai -v

# Static analysis/linting
go vet ./...

# Format code (standard Go formatting)
go fmt ./...

# Check if any files need formatting
find . -name "*.go" -exec gofmt -l {} \;
```

### Module-Specific Commands
```bash
# Work in the koppel module specifically
cd koppel && go test ./...
cd koppel && go mod tidy  # Clean up dependencies
```

## Code Style Guidelines

### General Go Conventions
- Use standard Go formatting (`go fmt`)
- Follow idiomatic Go naming conventions (camelCase for variables, PascalCase for exported types)
- Use functional options pattern for configuration (see `provider.Option`)
- Prefer interfaces over concrete types

### Import Organization
- Group imports in three sections: standard library, third-party, local packages
- Use absolute import paths with module prefixes (e.g., `"gosuda.org/koppel/provider"`)
- No unused imports (use `go mod tidy` to clean up)

### Type and Interface Patterns
- Define interfaces in `provider.go` with clear method signatures
- Use polymorphic interfaces for message parts (`Part` interface with `TextPart`, `BlobPart`, etc.)
- Implement custom `MarshalJSON`/`UnmarshalJSON` for complex types
- Use reflection for schema generation (see `tool.GenerateSchema`)

### Error Handling
- Return errors as the last return value
- Use explicit error checks, don't ignore errors
- Wrap errors with context when appropriate
- Don't use panic for expected error conditions

### Testing Patterns
- Place test files alongside source files (`*_test.go`)
- Use table-driven tests for multiple test cases
- Test interface compliance with var declarations (`var _ Provider = (*ConcreteProvider)(nil)`)
- Focus on testing business logic, not external API calls

## Project Structure

```
Handeln/
├── go.work              # Workspace configuration
├── go.mod               # Root module
├── koppel/              # Core library module
│   ├── go.mod          # Library dependencies
│   ├── provider/       # AI provider interfaces and implementations
│   │   ├── provider.go # Core interfaces and types
│   │   ├── openai/     # OpenAI provider
│   │   ├── anthropic/  # Anthropic provider
│   │   └── gemini/     # Gemini provider
│   ├── chat/           # Session management
│   └── tool/           # Tool/function calling support
└── AGENTS.md           # This file
```

## Key Interfaces and Types

### Provider Interface
```go
type Provider interface {
    GenerateContent(ctx context.Context, model string, messages []Message, options ...Option) (Response, error)
    GenerateContentStream(ctx context.Context, model string, messages []Message, options ...Option) (StreamResponse, error)
}
```

### Message Parts
- `TextPart` - Plain text content
- `BlobPart` - Binary data with MIME type
- `ThoughtPart` - Model reasoning/thinking
- `ToolCallPart` - Function call requests
- `ToolResultPart` - Function call results

### Tool Definitions
Tools are defined using Go structs with reflection-based schema generation:
```go
type MyToolInput struct {
    Param1 string `json:"param1" description:"Parameter description"`
    Param2 int    `json:"param2,omitempty"`
}

toolDef, err := tool.FromStruct("my_tool", "Tool description", MyToolInput{})
```

## Development Workflow

1. **Make changes** to source files
2. **Run tests**: `go test ./...`
3. **Check formatting**: `gofmt -l .` (should return nothing)
4. **Static analysis**: `go vet ./...`
5. **Build verification**: `go build ./...`

## Adding New Providers

When adding a new AI provider:

1. Create a new package under `koppel/provider/`
2. Implement the `Provider` interface
3. Add conversion logic for message parts and tools
4. Create comprehensive tests including interface compliance
5. Follow existing patterns in OpenAI/Anthropic/Gemini providers

## Dependencies

The project uses official SDKs for AI providers:
- OpenAI: `github.com/openai/openai-go/v3`
- Anthropic: `github.com/anthropics/anthropic-sdk-go`
- Gemini: `google.golang.org/genai`

All dependencies are managed through Go modules. Use `go mod tidy` to clean up unused dependencies.

## Notes

- This is a library project, not an executable application
- The workspace structure allows for multi-module development
- Modern Go features are used (Go 1.25.5, iterators available since 1.23)
- No external build tools or CI/CD configuration present
- Code is self-documenting through clear interface definitions