// Copyright 2026 cloudygreybeard
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package output provides the OutputAdapter interface for bookmark renderers.
//
// Output adapters convert bookmark collections into specific file formats
// or data representations. Each adapter is registered with the global
// registry and can be selected at runtime via --format flag.
//
// # Implementing an Output Adapter
//
// To create a new output adapter:
//
//  1. Create a new package under pkg/output/
//  2. Implement the Adapter interface
//  3. Register via init() using adapter.RegisterOutput()
//  4. Import in cmd/root.go to include in the build
//
// Example:
//
//	package csv
//
//	import (
//	    "github.com/cloudygreybeard/favs/pkg/adapter"
//	    "github.com/cloudygreybeard/favs/pkg/bookmark"
//	    "github.com/cloudygreybeard/favs/pkg/output"
//	)
//
//	func init() {
//	    adapter.RegisterOutput(New())
//	}
//
//	type Adapter struct {
//	    config output.Config
//	}
//
//	func New() *Adapter { return &Adapter{} }
//
//	func (a *Adapter) Name() string          { return "csv" }
//	func (a *Adapter) DisplayName() string   { return "CSV" }
//	func (a *Adapter) Extensions() []string  { return []string{".csv"} }
//
//	func (a *Adapter) Configure(cfg output.Config) error {
//	    a.config = cfg
//	    return nil
//	}
//
//	func (a *Adapter) Render(collection *bookmark.Collection, opts output.RenderOptions) ([]byte, error) {
//	    // Implement CSV rendering logic here
//	    return nil, nil
//	}
//
// See docs/adapters.md for comprehensive documentation.
package output

import (
	"github.com/cloudygreybeard/favs/pkg/bookmark"
)

// Adapter is the interface for bookmark output renderers.
//
// Output adapters take a collection of bookmarks and render them
// to a specific format (Markdown, JSON, CSV, HTML, etc.). Adapters
// should respect RenderOptions to control what information is included.
type Adapter interface {
	// Name returns the unique adapter identifier used in --format flag.
	// Should be lowercase, alphanumeric.
	// Examples: "markdown", "json", "yaml", "csv", "html"
	Name() string

	// DisplayName returns a human-friendly name for UI display.
	// Examples: "Markdown", "JSON", "YAML", "CSV", "HTML"
	DisplayName() string

	// Extensions returns file extensions supported by this format.
	// First extension is the default. Used for filename suggestions.
	// Examples: [".md", ".markdown"], [".json"], [".csv"]
	Extensions() []string

	// Configure applies runtime configuration to the adapter.
	// Called before Render() with user-specified options.
	Configure(cfg Config) error

	// Render converts bookmarks to the output format.
	// Returns the rendered content as bytes.
	// Should respect all applicable RenderOptions.
	Render(collection *bookmark.Collection, opts RenderOptions) ([]byte, error)
}

// Config holds adapter-specific configuration passed at runtime.
type Config struct {
	// Enabled indicates whether this adapter should be used.
	Enabled bool

	// Options holds adapter-specific key-value options.
	// Common keys include "style", "template", etc.
	Options map[string]interface{}
}

// RenderOptions configures what information to include in the output.
// Adapters should check each option and adjust output accordingly.
type RenderOptions struct {
	// IncludeMetadata adds a header with generation time, source info, counts.
	IncludeMetadata bool

	// IncludeDates includes the date each bookmark was added.
	IncludeDates bool

	// IncludeTags includes bookmark tags/labels.
	IncludeTags bool

	// IncludeProfile includes source and profile attribution per bookmark.
	IncludeProfile bool

	// GroupBySource groups bookmarks by their source adapter.
	// When true, output should have sections for each source.
	GroupBySource bool

	// SortAlpha sorts bookmarks alphabetically by title.
	SortAlpha bool

	// Style specifies a format variant (adapter-specific).
	// For markdown: "textual", "table", "yaml"
	Style string
}

// DefaultRenderOptions returns sensible defaults for rendering.
// All metadata is included, no sorting, no specific style.
func DefaultRenderOptions() RenderOptions {
	return RenderOptions{
		IncludeMetadata: true,
		IncludeDates:    true,
		IncludeTags:     true,
		IncludeProfile:  true,
		GroupBySource:   true,
		SortAlpha:       false,
		Style:           "",
	}
}
