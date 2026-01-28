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

// Package bookmark provides the core bookmark model.
//
// This package defines the data structures that flow between input adapters
// (which read bookmarks from sources) and output adapters (which render
// bookmarks to various formats). It serves as the common language of the
// application, independent of any specific source or format.
//
// # Core Types
//
// Bookmark is a single bookmark with all its metadata:
//
//	b := bookmark.Bookmark{
//	    Title:      "GitHub",
//	    URL:        "https://github.com",
//	    FolderPath: []string{"Bookmarks", "Dev", "Tools"},
//	    DateAdded:  time.Now(),
//	    Source:     "chrome",
//	    Profile:    "Default",
//	    Tags:       []string{"dev", "git"},
//	}
//
// Collection aggregates bookmarks from multiple sources:
//
//	collection := bookmark.NewCollection()
//	collection.Add(chromeBookmarks, bookmark.SourceInfo{Name: "chrome", Profile: "Default"})
//	collection.Add(firefoxBookmarks, bookmark.SourceInfo{Name: "firefox"})
//
// # Design Principles
//
//  1. Source-agnostic: Bookmark fields are generic enough to represent
//     bookmarks from any source (browsers, APIs, files).
//
//  2. Metadata-rich: Include optional fields (DateAdded, Tags) that some
//     sources provide, even if others don't.
//
//  3. Hierarchical: FolderPath preserves folder structure from sources
//     that support it, enabling hierarchical output rendering.
//
//  4. Attributable: Source and Profile fields allow tracking where each
//     bookmark came from, useful for grouping and debugging.
package bookmark

import (
	"time"
)

// Bookmark represents a single bookmark from any source.
//
// All fields except URL are optional, but adapters should populate
// as many fields as the source provides for the best output quality.
type Bookmark struct {
	// Title is the display name of the bookmark.
	// If empty, renderers typically fall back to the URL.
	Title string

	// URL is the bookmark's target address (required).
	URL string

	// FolderPath is the hierarchical folder structure.
	// Example: ["Bookmarks Bar", "Work", "Tools"]
	// Empty slice means the bookmark is at the root level.
	FolderPath []string

	// DateAdded is when the bookmark was created.
	// Zero value means the date is unknown.
	DateAdded time.Time

	// Source identifies which input adapter produced this bookmark.
	// Should match the adapter's Name() return value.
	// Examples: "chrome", "firefox", "pinboard"
	Source string

	// Profile identifies the profile/account within the source.
	// Examples: "Default", "Profile 1", "work@example.com"
	Profile string

	// Tags are labels or categories assigned to the bookmark.
	// Not all sources support tags (Firefox does, Chrome doesn't).
	Tags []string
}

// Collection is a set of bookmarks aggregated from one or more sources.
//
// Collections track which sources contributed bookmarks, enabling
// output adapters to group by source or show source attribution.
type Collection struct {
	// Bookmarks is the aggregated list of all bookmarks.
	Bookmarks []Bookmark

	// Sources describes where the bookmarks came from.
	Sources []SourceInfo
}

// SourceInfo describes a bookmark source that contributed to a collection.
type SourceInfo struct {
	// Name is the adapter identifier (e.g., "chrome", "firefox").
	Name string

	// Profile is the profile/account within the source.
	Profile string

	// Path is the filesystem path or URI that was read.
	Path string

	// Count is the number of bookmarks from this source.
	Count int
}

// NewCollection creates a new empty collection.
func NewCollection() *Collection {
	return &Collection{
		Bookmarks: []Bookmark{},
		Sources:   []SourceInfo{},
	}
}

// Add appends bookmarks to the collection with source attribution.
// The source's Count field is automatically set to len(bookmarks).
func (c *Collection) Add(bookmarks []Bookmark, source SourceInfo) {
	c.Bookmarks = append(c.Bookmarks, bookmarks...)
	source.Count = len(bookmarks)
	c.Sources = append(c.Sources, source)
}

// Count returns the total number of bookmarks in the collection.
func (c *Collection) Count() int {
	return len(c.Bookmarks)
}
