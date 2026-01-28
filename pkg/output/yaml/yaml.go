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

// Package yaml provides an output adapter for YAML format.
package yaml

import (
	"runtime"
	"time"

	"github.com/cloudygreybeard/favs/pkg/adapter"
	"github.com/cloudygreybeard/favs/pkg/bookmark"
	"github.com/cloudygreybeard/favs/pkg/output"
	"gopkg.in/yaml.v3"
)

func init() {
	adapter.RegisterOutput(New())
}

// Adapter implements output.Adapter for YAML format.
type Adapter struct {
	config output.Config
}

// New creates a new YAML adapter.
func New() *Adapter {
	return &Adapter{}
}

// Name returns the adapter identifier.
func (a *Adapter) Name() string {
	return "yaml"
}

// DisplayName returns a human-friendly name.
func (a *Adapter) DisplayName() string {
	return "YAML"
}

// Extensions returns supported file extensions.
func (a *Adapter) Extensions() []string {
	return []string{".yaml", ".yml"}
}

// Configure applies configuration to the adapter.
func (a *Adapter) Configure(cfg output.Config) error {
	a.config = cfg
	return nil
}

// Render converts bookmarks to YAML.
func (a *Adapter) Render(collection *bookmark.Collection, opts output.RenderOptions) ([]byte, error) {
	doc := Document{
		Bookmarks: make([]BookmarkEntry, 0, len(collection.Bookmarks)),
	}

	if opts.IncludeMetadata {
		doc.Metadata = &Metadata{
			Generated: time.Now().Format(time.RFC3339),
			Platform:  runtime.GOOS + "/" + runtime.GOARCH,
			Total:     collection.Count(),
			Sources:   make([]SourceEntry, 0, len(collection.Sources)),
		}
		for _, s := range collection.Sources {
			doc.Metadata.Sources = append(doc.Metadata.Sources, SourceEntry{
				Name:    s.Name,
				Profile: s.Profile,
				Path:    s.Path,
				Count:   s.Count,
			})
		}
	}

	for _, b := range collection.Bookmarks {
		entry := BookmarkEntry{
			Title:  b.Title,
			URL:    b.URL,
			Folder: joinFolder(b.FolderPath),
		}

		if opts.IncludeDates && !b.DateAdded.IsZero() {
			date := b.DateAdded.Format("2006-01-02")
			entry.DateAdded = date
		}

		if opts.IncludeTags && len(b.Tags) > 0 {
			entry.Tags = b.Tags
		}

		if opts.IncludeProfile {
			entry.Source = b.Source
			entry.Profile = b.Profile
		}

		doc.Bookmarks = append(doc.Bookmarks, entry)
	}

	return yaml.Marshal(doc)
}

func joinFolder(path []string) string {
	if len(path) == 0 {
		return ""
	}
	result := path[0]
	for i := 1; i < len(path); i++ {
		result += "/" + path[i]
	}
	return result
}

// Document is the top-level YAML structure.
type Document struct {
	Metadata  *Metadata       `yaml:"metadata,omitempty"`
	Bookmarks []BookmarkEntry `yaml:"bookmarks"`
}

// Metadata contains generation information.
type Metadata struct {
	Generated string        `yaml:"generated"`
	Platform  string        `yaml:"platform"`
	Total     int           `yaml:"total"`
	Sources   []SourceEntry `yaml:"sources,omitempty"`
}

// SourceEntry describes a bookmark source.
type SourceEntry struct {
	Name    string `yaml:"name"`
	Profile string `yaml:"profile,omitempty"`
	Path    string `yaml:"path,omitempty"`
	Count   int    `yaml:"count"`
}

// BookmarkEntry is a single bookmark in the YAML output.
type BookmarkEntry struct {
	Title     string   `yaml:"title"`
	URL       string   `yaml:"url"`
	Folder    string   `yaml:"folder,omitempty"`
	DateAdded string   `yaml:"date_added,omitempty"`
	Tags      []string `yaml:"tags,omitempty"`
	Source    string   `yaml:"source,omitempty"`
	Profile   string   `yaml:"profile,omitempty"`
}
