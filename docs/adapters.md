# Adapter Development Guide

favs uses a pluggable adapter architecture that separates bookmark sources (inputs) from output formats (outputs). This guide explains how to create new adapters.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Adapter Registry                         │
│  adapter.RegisterInput()         adapter.RegisterOutput()   │
└─────────────────────────────────────────────────────────────┘
        │                                    │
        ▼                                    ▼
┌───────────────────┐              ┌───────────────────┐
│  Input Adapters   │              │  Output Adapters  │
│  input.Adapter    │              │  output.Adapter   │
├───────────────────┤              ├───────────────────┤
│ • chromium        │              │ • markdown        │
│ • firefox         │              │ • json            │
│ • safari          │              │ • yaml            │
│ • (your adapter)  │              │ • (your adapter)  │
└───────────────────┘              └───────────────────┘
        │                                    ▲
        ▼                                    │
┌─────────────────────────────────────────────────────────────┐
│                    Core Pipeline                            │
│                                                             │
│   Read() ──► bookmark.Collection ──► Filter ──► Render()   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Core Types

### bookmark.Bookmark

The fundamental data structure passed between adapters:

```go
// pkg/bookmark/bookmark.go

type Bookmark struct {
    Title      string      // Display title
    URL        string      // Bookmark URL
    FolderPath []string    // Hierarchical folder path: ["Bookmarks", "Work", "Tools"]
    DateAdded  time.Time   // When the bookmark was created
    Source     string      // Adapter name that produced this bookmark
    Profile    string      // Profile/account identifier
    Tags       []string    // Labels/tags (if supported by source)
}
```

### bookmark.Collection

A set of bookmarks with source metadata:

```go
type Collection struct {
    Bookmarks []Bookmark
    Sources   []SourceInfo
}

type SourceInfo struct {
    Name    string  // Adapter name
    Profile string  // Profile identifier
    Path    string  // Path or URI that was read
    Count   int     // Number of bookmarks from this source
}
```

---

## Input Adapters

Input adapters read bookmarks from external sources (browsers, APIs, files).

### Interface

```go
// pkg/input/input.go

type Adapter interface {
    // Identity
    Name() string        // Unique identifier (e.g., "pinboard", "raindrop")
    DisplayName() string // Human-friendly name (e.g., "Pinboard", "Raindrop.io")

    // Availability
    Available() bool     // Can this source be read right now?
    Path() string        // Path/URI being read (for logging/debugging)

    // Configuration
    Configure(cfg Config) error  // Apply runtime configuration

    // Reading
    Read(ctx context.Context) ([]bookmark.Bookmark, error)  // Fetch bookmarks
    ListProfiles() ([]ProfileInfo, error)                   // List available profiles
}

type Config struct {
    Enabled    bool                   // Is this adapter enabled?
    Profile    string                 // Specific profile to read
    CustomPath string                 // Override default path
    Options    map[string]interface{} // Adapter-specific options
}

type ProfileInfo struct {
    Name      string
    Path      string
    IsDefault bool
}
```

### Implementation Example: Pinboard API

```go
// pkg/input/pinboard/pinboard.go
package pinboard

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "time"

    "github.com/cloudygreybeard/favs/pkg/adapter"
    "github.com/cloudygreybeard/favs/pkg/bookmark"
    "github.com/cloudygreybeard/favs/pkg/input"
)

const apiBase = "https://api.pinboard.in/v1"

// Register on package import
func init() {
    adapter.RegisterInput(New())
}

// Adapter reads bookmarks from Pinboard API.
type Adapter struct {
    config   input.Config
    apiToken string
}

// New creates a new Pinboard adapter.
func New() *Adapter {
    return &Adapter{}
}

// Name returns the adapter identifier.
func (a *Adapter) Name() string {
    return "pinboard"
}

// DisplayName returns a human-friendly name.
func (a *Adapter) DisplayName() string {
    return "Pinboard"
}

// Available returns true if Pinboard can be accessed.
// Checks for API token in config or environment.
func (a *Adapter) Available() bool {
    return a.getAPIToken() != ""
}

// Path returns the API endpoint being accessed.
func (a *Adapter) Path() string {
    return apiBase + "/posts/all"
}

// Configure applies configuration to the adapter.
func (a *Adapter) Configure(cfg input.Config) error {
    a.config = cfg
    
    // Check for API token in options
    if token, ok := cfg.Options["api_token"].(string); ok {
        a.apiToken = token
    }
    
    return nil
}

// ListProfiles returns available profiles.
// Pinboard has a single account, so we return one profile.
func (a *Adapter) ListProfiles() ([]input.ProfileInfo, error) {
    if !a.Available() {
        return nil, nil
    }
    return []input.ProfileInfo{
        {Name: "default", Path: a.Path(), IsDefault: true},
    }, nil
}

// Read fetches all bookmarks from Pinboard.
func (a *Adapter) Read(ctx context.Context) ([]bookmark.Bookmark, error) {
    token := a.getAPIToken()
    if token == "" {
        return nil, fmt.Errorf("pinboard: API token not configured")
    }

    // Build request
    url := fmt.Sprintf("%s/posts/all?auth_token=%s&format=json", apiBase, token)
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }

    // Execute request
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("pinboard: API returned %d", resp.StatusCode)
    }

    // Parse response
    var posts []pinboardPost
    if err := json.NewDecoder(resp.Body).Decode(&posts); err != nil {
        return nil, err
    }

    // Convert to bookmarks
    bookmarks := make([]bookmark.Bookmark, 0, len(posts))
    for _, post := range posts {
        bookmarks = append(bookmarks, bookmark.Bookmark{
            Title:      post.Description,
            URL:        post.Href,
            FolderPath: []string{"Pinboard"},  // Pinboard is flat
            DateAdded:  parseTime(post.Time),
            Source:     "pinboard",
            Profile:    "default",
            Tags:       splitTags(post.Tags),
        })
    }

    return bookmarks, nil
}

// getAPIToken retrieves the API token from config or environment.
func (a *Adapter) getAPIToken() string {
    if a.apiToken != "" {
        return a.apiToken
    }
    return os.Getenv("PINBOARD_API_TOKEN")
}

// pinboardPost represents a Pinboard bookmark from the API.
type pinboardPost struct {
    Href        string `json:"href"`
    Description string `json:"description"`
    Tags        string `json:"tags"`
    Time        string `json:"time"`
}

func parseTime(s string) time.Time {
    t, _ := time.Parse(time.RFC3339, s)
    return t
}

func splitTags(s string) []string {
    if s == "" {
        return nil
    }
    // Pinboard uses space-separated tags
    // Implementation left as exercise
    return nil
}
```

### Registration

Adapters self-register via `init()`. To include in the build, import in `cmd/root.go`:

```go
import (
    _ "github.com/cloudygreybeard/favs/pkg/input/pinboard"
)
```

---

## Output Adapters

Output adapters convert bookmark collections to specific formats.

### Interface

```go
// pkg/output/output.go

type Adapter interface {
    // Identity
    Name() string          // Unique identifier (e.g., "csv", "html")
    DisplayName() string   // Human-friendly name (e.g., "CSV", "HTML")
    Extensions() []string  // File extensions (e.g., [".csv"])

    // Configuration
    Configure(cfg Config) error

    // Rendering
    Render(collection *bookmark.Collection, opts RenderOptions) ([]byte, error)
}

type Config struct {
    Enabled bool
    Options map[string]interface{}
}

type RenderOptions struct {
    IncludeMetadata bool   // Include generation timestamp, source info
    IncludeDates    bool   // Include bookmark creation dates
    IncludeTags     bool   // Include tags
    IncludeProfile  bool   // Include source/profile info per bookmark
    GroupBySource   bool   // Group bookmarks by source
    SortAlpha       bool   // Sort alphabetically
    Style           string // Adapter-specific style variant
}
```

### Implementation Example: CSV Output

```go
// pkg/output/csv/csv.go
package csv

import (
    "bytes"
    "encoding/csv"
    "strings"

    "github.com/cloudygreybeard/favs/pkg/adapter"
    "github.com/cloudygreybeard/favs/pkg/bookmark"
    "github.com/cloudygreybeard/favs/pkg/output"
)

func init() {
    adapter.RegisterOutput(New())
}

// Adapter renders bookmarks as CSV.
type Adapter struct {
    config output.Config
}

func New() *Adapter {
    return &Adapter{}
}

func (a *Adapter) Name() string          { return "csv" }
func (a *Adapter) DisplayName() string   { return "CSV" }
func (a *Adapter) Extensions() []string  { return []string{".csv"} }

func (a *Adapter) Configure(cfg output.Config) error {
    a.config = cfg
    return nil
}

func (a *Adapter) Render(collection *bookmark.Collection, opts output.RenderOptions) ([]byte, error) {
    var buf bytes.Buffer
    w := csv.NewWriter(&buf)

    // Write header
    header := []string{"Title", "URL", "Folder"}
    if opts.IncludeDates {
        header = append(header, "Date Added")
    }
    if opts.IncludeTags {
        header = append(header, "Tags")
    }
    if opts.IncludeProfile {
        header = append(header, "Source", "Profile")
    }
    w.Write(header)

    // Write bookmarks
    for _, b := range collection.Bookmarks {
        row := []string{
            b.Title,
            b.URL,
            strings.Join(b.FolderPath, "/"),
        }
        if opts.IncludeDates {
            date := ""
            if !b.DateAdded.IsZero() {
                date = b.DateAdded.Format("2006-01-02")
            }
            row = append(row, date)
        }
        if opts.IncludeTags {
            row = append(row, strings.Join(b.Tags, ", "))
        }
        if opts.IncludeProfile {
            row = append(row, b.Source, b.Profile)
        }
        w.Write(row)
    }

    w.Flush()
    return buf.Bytes(), w.Error()
}
```

---

## Best Practices

### Input Adapters

1. **Check availability gracefully**: `Available()` should not panic or make network calls
2. **Support context cancellation**: Honor `ctx.Done()` in `Read()`
3. **Handle missing credentials**: Return empty results, not errors, when not configured
4. **Populate all Bookmark fields**: Fill `Source` and `Profile` for proper attribution
5. **Parse dates when available**: Helps with sorting and filtering

### Output Adapters

1. **Respect RenderOptions**: Check each option and adjust output accordingly
2. **Handle empty collections**: Return valid (possibly empty) output
3. **Use proper encoding**: Escape special characters for the format
4. **Include metadata when requested**: Generation time, source info, counts

### General

1. **Register in init()**: Ensures adapter is available without manual wiring
2. **Document configuration options**: Explain what Options keys are supported
3. **Add tests**: Cover happy path and error cases
4. **Log sparingly**: Use the verbose flag for debug output

---

## Testing Your Adapter

```go
func TestAdapter_Read(t *testing.T) {
    adapter := New()
    
    // Configure with test credentials
    err := adapter.Configure(input.Config{
        Options: map[string]interface{}{
            "api_token": os.Getenv("TEST_API_TOKEN"),
        },
    })
    if err != nil {
        t.Fatalf("Configure failed: %v", err)
    }

    if !adapter.Available() {
        t.Skip("Adapter not available (missing credentials)")
    }

    bookmarks, err := adapter.Read(context.Background())
    if err != nil {
        t.Fatalf("Read failed: %v", err)
    }

    // Verify bookmarks have required fields
    for _, b := range bookmarks {
        if b.URL == "" {
            t.Error("Bookmark missing URL")
        }
        if b.Source == "" {
            t.Error("Bookmark missing Source")
        }
    }
}
```

---

## Configuration Integration

To make your adapter configurable via `favs.yaml`, update `pkg/config/config.go`:

```go
type InputsConfig struct {
    // ... existing adapters ...
    Pinboard InputConfig `yaml:"pinboard"`
}
```

And add to `GetInputConfig()`:

```go
case "pinboard":
    return c.Inputs.Pinboard
```

---

## Questions?

Open an issue or discussion on GitHub. We're happy to help you contribute!
