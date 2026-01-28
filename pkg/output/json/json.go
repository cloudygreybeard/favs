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

// Package json provides an output adapter for JSON format.
package json

import (
	"encoding/json"
	"runtime"
	"time"

	"github.com/cloudygreybeard/favs/pkg/adapter"
	"github.com/cloudygreybeard/favs/pkg/bookmark"
	"github.com/cloudygreybeard/favs/pkg/output"
)

func init() {
	adapter.RegisterOutput(New())
}

// Adapter implements output.Adapter for JSON format.
type Adapter struct {
	config output.Config
}

// New creates a new JSON adapter.
func New() *Adapter {
	return &Adapter{}
}

// Name returns the adapter identifier.
func (a *Adapter) Name() string {
	return "json"
}

// DisplayName returns a human-friendly name.
func (a *Adapter) DisplayName() string {
	return "JSON"
}

// Extensions returns supported file extensions.
func (a *Adapter) Extensions() []string {
	return []string{".json"}
}

// Configure applies configuration to the adapter.
func (a *Adapter) Configure(cfg output.Config) error {
	a.config = cfg
	return nil
}

// Render converts bookmarks to JSON.
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
			Folder: b.FolderPath,
		}

		if opts.IncludeDates && !b.DateAdded.IsZero() {
			date := b.DateAdded.Format("2006-01-02")
			entry.DateAdded = &date
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

	return json.MarshalIndent(doc, "", "  ")
}

// Document is the top-level JSON structure.
type Document struct {
	Metadata  *Metadata       `json:"metadata,omitempty"`
	Bookmarks []BookmarkEntry `json:"bookmarks"`
}

// Metadata contains generation information.
type Metadata struct {
	Generated string        `json:"generated"`
	Platform  string        `json:"platform"`
	Total     int           `json:"total"`
	Sources   []SourceEntry `json:"sources,omitempty"`
}

// SourceEntry describes a bookmark source.
type SourceEntry struct {
	Name    string `json:"name"`
	Profile string `json:"profile,omitempty"`
	Path    string `json:"path,omitempty"`
	Count   int    `json:"count"`
}

// BookmarkEntry is a single bookmark in the JSON output.
type BookmarkEntry struct {
	Title     string   `json:"title"`
	URL       string   `json:"url"`
	Folder    []string `json:"folder,omitempty"`
	DateAdded *string  `json:"date_added,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	Source    string   `json:"source,omitempty"`
	Profile   string   `json:"profile,omitempty"`
}
