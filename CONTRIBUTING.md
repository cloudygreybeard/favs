# Contributing to favs

Thank you for your interest in contributing to favs! This document provides guidelines and information for contributors.

## Code of Conduct

Be respectful, inclusive, and constructive. We welcome contributors of all experience levels.

## Ways to Contribute

- **Report bugs**: Open an issue describing the problem and how to reproduce it
- **Suggest features**: Open an issue describing the feature and its use case
- **Add adapters**: Implement new input sources or output formats
- **Improve documentation**: Fix typos, clarify explanations, add examples
- **Write tests**: Improve test coverage

## Development Setup

```bash
# Clone the repository
git clone https://github.com/cloudygreybeard/favs
cd favs

# Install dependencies
go mod download

# Build
make build

# Run tests
make test

# Run linter
make lint
```

## Project Structure

```
favs/
├── cmd/                    # CLI commands
│   ├── root.go            # Main command and flags
│   ├── sync.go            # Sync logic
│   ├── adapters.go        # List adapters command
│   └── serve.go           # MCP server command
├── pkg/
│   ├── adapter/           # Adapter registry
│   │   └── registry.go    # Global registration
│   ├── bookmark/          # Core domain model
│   │   ├── bookmark.go    # Bookmark struct
│   │   └── filter.go      # Filtering logic
│   ├── config/            # Configuration
│   │   └── config.go      # Config loading
│   ├── input/             # Input adapters
│   │   ├── input.go       # Interface definition
│   │   ├── chromium/      # Chrome/Edge/Brave
│   │   ├── firefox/       # Firefox
│   │   ├── opml/          # OPML/HTML import
│   │   └── safari/        # Safari
│   ├── output/            # Output adapters
│   │   ├── output.go      # Interface definition
│   │   ├── json/          # JSON renderer
│   │   ├── markdown/      # Markdown renderer
│   │   ├── opml/          # OPML/HTML export
│   │   └── yaml/          # YAML renderer
│   └── mcp/               # MCP server
│       └── server.go      # JSON-RPC server
├── main.go                # Entry point
├── Makefile               # Build automation
└── docs/                  # Documentation
    └── adapters.md        # Adapter development guide
```

## Adding New Adapters

The most common contribution is adding new adapters. See [docs/adapters.md](docs/adapters.md) for a comprehensive guide.

### Quick Start: Input Adapter

```go
// pkg/input/myservice/myservice.go
package myservice

import (
    "context"
    "github.com/cloudygreybeard/favs/pkg/adapter"
    "github.com/cloudygreybeard/favs/pkg/bookmark"
    "github.com/cloudygreybeard/favs/pkg/input"
)

func init() {
    adapter.RegisterInput(New())
}

type Adapter struct {
    config input.Config
}

func New() *Adapter {
    return &Adapter{}
}

func (a *Adapter) Name() string        { return "myservice" }
func (a *Adapter) DisplayName() string { return "My Service" }
func (a *Adapter) Available() bool     { return true }
func (a *Adapter) Path() string        { return "api://myservice" }

func (a *Adapter) Configure(cfg input.Config) error {
    a.config = cfg
    return nil
}

func (a *Adapter) ListProfiles() ([]input.ProfileInfo, error) {
    return []input.ProfileInfo{{Name: "default", IsDefault: true}}, nil
}

func (a *Adapter) Read(ctx context.Context) ([]bookmark.Bookmark, error) {
    // Implement bookmark reading logic
    return nil, nil
}
```

### Quick Start: Output Adapter

```go
// pkg/output/myformat/myformat.go
package myformat

import (
    "github.com/cloudygreybeard/favs/pkg/adapter"
    "github.com/cloudygreybeard/favs/pkg/bookmark"
    "github.com/cloudygreybeard/favs/pkg/output"
)

func init() {
    adapter.RegisterOutput(New())
}

type Adapter struct {
    config output.Config
}

func New() *Adapter {
    return &Adapter{}
}

func (a *Adapter) Name() string          { return "myformat" }
func (a *Adapter) DisplayName() string   { return "My Format" }
func (a *Adapter) Extensions() []string  { return []string{".myf"} }

func (a *Adapter) Configure(cfg output.Config) error {
    a.config = cfg
    return nil
}

func (a *Adapter) Render(collection *bookmark.Collection, opts output.RenderOptions) ([]byte, error) {
    // Implement rendering logic
    return nil, nil
}
```

## Pull Request Process

1. **Fork** the repository
2. **Create a branch** for your feature: `git checkout -b feature/my-feature`
3. **Make changes** following the code style
4. **Add tests** for new functionality
5. **Run tests**: `make test`
6. **Run linter**: `make lint`
7. **Commit** with a clear message
8. **Push** to your fork
9. **Open a Pull Request** with a description of changes

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Add doc comments to exported types and functions
- Keep functions focused and reasonably sized
- Handle errors explicitly
- Use meaningful variable names

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(input): add Pinboard input adapter

- Implement API client for Pinboard bookmarks
- Support API token authentication
- Add rate limiting to respect API limits
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `build`, `ci`

## Testing

- Add unit tests for new functionality
- Test edge cases and error conditions
- Use table-driven tests where appropriate

```go
func TestAdapter_Read(t *testing.T) {
    tests := []struct {
        name    string
        setup   func() *Adapter
        want    int
        wantErr bool
    }{
        // test cases
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

## Questions?

Open an issue or start a discussion. We're happy to help!

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
