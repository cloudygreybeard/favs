# favs

> Your bookmarks as context for AI assistants

A cross-platform CLI tool for aggregating browser bookmarks and converting them to structured formats for AI assistant reference.

## Features

- **Multi-browser support**: Chrome, Edge, Firefox, Safari, Chromium, Brave
- **Multiple output formats**: Markdown, JSON, YAML, OPML, Netscape HTML
- **Import support**: OPML and Netscape HTML bookmark files
- **Pluggable architecture**: Extensible input and output adapters
- **MCP server**: Expose bookmarks to AI assistants via Model Context Protocol
- **Cross-platform**: Linux, macOS, Windows

## Installation

### Homebrew (macOS/Linux)

```bash
brew install cloudygreybeard/tap/favs
```

### Go Install

```bash
go install github.com/cloudygreybeard/favs@latest
```

### From Source

```bash
git clone https://github.com/cloudygreybeard/favs
cd favs
make build
```

## Usage

### Basic Usage

```bash
# Output to stdout (default)
favs

# Write to file
favs -o bookmarks.md

# Specific browser
favs -b firefox

# Specific browser and profile
favs -b chrome -p "Profile 1"

# All browsers and profiles
favs --all

# Verbose output
favs -v
```

### Output Formats

```bash
# Markdown (default) - nested lists
favs --format markdown

# Markdown - table style
favs --format markdown --style table

# Markdown - embedded YAML
favs --format markdown --style yaml

# Pure JSON
favs --format json

# Pure YAML
favs --format yaml

# OPML (for import into other tools)
favs --format opml -o bookmarks.opml

# Netscape HTML (universal browser import format)
favs --format html -o bookmarks.html
```

### Import from File

```bash
# Import from OPML file
favs --input opml --custom-path bookmarks.opml

# Import from Netscape HTML (exported from any browser)
favs --input opml --custom-path bookmarks.html
```

### List Available Adapters

```bash
# List all registered input/output adapters
favs adapters

# List available browser profiles
favs --list
```

### MCP Server Mode

Run as an MCP server for AI assistant integration:

```bash
favs serve
```

Add to your MCP client configuration (e.g., Claude Desktop):

```json
{
  "mcpServers": {
    "favs": {
      "command": "/path/to/favs",
      "args": ["serve"]
    }
  }
}
```

**Available MCP Resources:**
- `favs://all` - All bookmarks (JSON)
- `favs://markdown` - All bookmarks (Markdown)
- `favs://chrome` - Chrome bookmarks
- `favs://firefox` - Firefox bookmarks

**Available MCP Tools:**
- `sync_bookmarks` - Refresh bookmarks from browsers
- `search_bookmarks` - Search bookmarks by title or URL

## URL Filtering

favs filters URLs by protocol and length to exclude problematic bookmarks:

```bash
# Exclude specific protocols (overrides config)
favs --exclude-protocols data,javascript,blob

# Warn on specific protocols (overrides config)
favs --warn-protocols file,chrome,about

# Exclude URLs longer than 4KB
favs --max-url-length 4096

# Warn on URLs longer than 2KB
favs --warn-url-length 2048
```

**Default exclusions:**
- `data:` - Base64-encoded images can be 100KB+
- `javascript:` - Bookmarklets are code, not links

**Default warnings:**
- `file:` - Local file paths
- `chrome:`, `about:` - Browser internal pages
- `blob:` - Blob URLs
- URLs longer than 2048 characters

## Configuration

Create `favs.yaml` in the current directory or `~/.favs/config.yaml`:

```yaml
inputs:
  chrome:
    enabled: true
    profile: ""           # empty = auto-detect
  firefox:
    enabled: true
  safari:
    enabled: true

outputs:
  markdown:
    enabled: true
    style: textual        # textual, table, yaml
  json:
    enabled: false
  yaml:
    enabled: false

pipeline:
  filter:
    exclude_folders: [Trash]
    exclude_url_patterns: []
    exclude_protocols: [data, javascript]
    warn_protocols: [file, chrome, about, blob]
    warn_url_length: 2048
  transform:
    deduplicate: false
    sort: false
  render:
    include_metadata: true
    include_dates: true
    include_tags: true
    include_profile: true
    group_by_source: true
```

## Architecture

favs uses a pluggable adapter architecture that separates bookmark sources (inputs) from output formats (outputs):

```
┌─────────────────────────────────────────┐
│           Adapter Registry              │
├─────────────────────────────────────────┤
│  Input Adapters    │  Output Adapters   │
│  - Chrome          │  - Markdown        │
│  - Firefox         │  - JSON            │
│  - Safari          │  - YAML            │
│  - Edge            │  - OPML            │
│  - Brave           │  - HTML            │
│  - OPML/HTML       │                    │
└─────────────────────────────────────────┘
           │                   │
           ▼                   ▼
┌─────────────────────────────────────────┐
│           Core Pipeline                 │
│  Read → Filter → Transform → Render     │
└─────────────────────────────────────────┘
```

### Adding New Adapters

favs is designed for extensibility. Adding new bookmark sources or output formats is straightforward:

**Input adapters** read bookmarks from sources (browsers, APIs, files):

```go
type input.Adapter interface {
    Name() string                                              // Unique identifier
    DisplayName() string                                       // Human-friendly name
    Available() bool                                           // Can this source be read?
    Configure(cfg Config) error                                // Apply configuration
    Read(ctx context.Context) ([]bookmark.Bookmark, error)     // Fetch bookmarks
    ListProfiles() ([]ProfileInfo, error)                      // List available profiles
    Path() string                                              // Path being read
}
```

**Output adapters** render bookmarks to formats (Markdown, JSON, CSV, HTML):

```go
type output.Adapter interface {
    Name() string                                                            // Unique identifier
    DisplayName() string                                                     // Human-friendly name
    Extensions() []string                                                    // File extensions
    Configure(cfg Config) error                                              // Apply configuration
    Render(collection *bookmark.Collection, opts RenderOptions) ([]byte, error)  // Render output
}
```

For detailed implementation guidance with complete examples, see:

- **[docs/adapters.md](docs/adapters.md)** - Comprehensive adapter development guide
- **[CONTRIBUTING.md](CONTRIBUTING.md)** - Contribution guidelines

## Development

```bash
# Build
make build

# Run tests
make test

# Run linter
make lint

# Build for all platforms
make release
```

## License

Apache License 2.0
