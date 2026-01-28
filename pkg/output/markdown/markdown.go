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

// Package markdown provides an output adapter for markdown format.
package markdown

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/cloudygreybeard/favs/pkg/adapter"
	"github.com/cloudygreybeard/favs/pkg/bookmark"
	"github.com/cloudygreybeard/favs/pkg/output"
)

// Style defines the markdown sub-format.
type Style string

const (
	StyleTextual Style = "textual" // Nested markdown lists
	StyleTable   Style = "table"   // Markdown tables
	StyleYAML    Style = "yaml"    // Embedded YAML in code fence
)

func init() {
	adapter.RegisterOutput(New())
}

// Adapter implements output.Adapter for markdown format.
type Adapter struct {
	config output.Config
	style  Style
}

// New creates a new markdown adapter.
func New() *Adapter {
	return &Adapter{style: StyleTextual}
}

// Name returns the adapter identifier.
func (a *Adapter) Name() string {
	return "markdown"
}

// DisplayName returns a human-friendly name.
func (a *Adapter) DisplayName() string {
	return "Markdown"
}

// Extensions returns supported file extensions.
func (a *Adapter) Extensions() []string {
	return []string{".md", ".markdown"}
}

// Configure applies configuration to the adapter.
func (a *Adapter) Configure(cfg output.Config) error {
	a.config = cfg
	if style, ok := cfg.Options["style"].(string); ok {
		a.style = Style(style)
	}
	return nil
}

// Render converts bookmarks to markdown.
func (a *Adapter) Render(collection *bookmark.Collection, opts output.RenderOptions) ([]byte, error) {
	style := a.style
	if opts.Style != "" {
		style = Style(opts.Style)
	}

	var result string
	switch style {
	case StyleTable:
		result = a.renderTable(collection, opts)
	case StyleYAML:
		result = a.renderYAML(collection, opts)
	default:
		result = a.renderTextual(collection, opts)
	}

	return []byte(result), nil
}

// folder represents a bookmark folder in the hierarchy.
type folder struct {
	Name       string
	Bookmarks  []bookmark.Bookmark
	Subfolders []*folder
}

func (a *Adapter) renderTextual(collection *bookmark.Collection, opts output.RenderOptions) string {
	var sb strings.Builder

	sb.WriteString("# Browser Bookmarks\n\n")

	if opts.IncludeMetadata {
		sb.WriteString(fmt.Sprintf("*Generated: %s*\n", time.Now().Format("2006-01-02 15:04:05")))
		sb.WriteString(fmt.Sprintf("*Platform: %s (%s)*\n", runtime.GOOS, runtime.GOARCH))
		sb.WriteString(fmt.Sprintf("*Source: %s*\n", formatSources(collection.Sources)))
		sb.WriteString(fmt.Sprintf("*Total bookmarks: %d*\n\n", collection.Count()))
	}

	bookmarks := collection.Bookmarks
	if opts.SortAlpha {
		sort.Slice(bookmarks, func(i, j int) bool {
			return strings.ToLower(bookmarks[i].Title) < strings.ToLower(bookmarks[j].Title)
		})
	}

	if opts.GroupBySource {
		grouped := groupBySourceProfile(bookmarks)
		keys := sortedGroupKeys(grouped)

		for _, key := range keys {
			bm := grouped[key]
			if len(bm) == 0 {
				continue
			}

			header := strings.Title(bm[0].Source)
			if opts.IncludeProfile && bm[0].Profile != "" {
				header += " / " + bm[0].Profile
			}
			sb.WriteString(fmt.Sprintf("## %s\n\n", header))

			tree := organizeByFolder(bm)
			a.renderFolder(tree, &sb, 0, opts)
		}
	} else {
		tree := organizeByFolder(bookmarks)
		a.renderFolder(tree, &sb, 0, opts)
	}

	return sb.String()
}

func (a *Adapter) renderFolder(f *folder, sb *strings.Builder, indent int, opts output.RenderOptions) {
	// Render folder heading (skip root)
	if f.Name != "" && f.Name != "Bookmarks" {
		sb.WriteString(fmt.Sprintf("%s- **%s**\n", strings.Repeat("  ", indent), f.Name))
		indent++
	}

	// Render bookmarks
	bookmarks := f.Bookmarks
	if opts.SortAlpha {
		sort.Slice(bookmarks, func(i, j int) bool {
			return strings.ToLower(bookmarks[i].Title) < strings.ToLower(bookmarks[j].Title)
		})
	}

	for _, b := range bookmarks {
		a.renderBookmark(b, sb, indent, opts)
	}

	// Render subfolders
	subfolders := f.Subfolders
	if opts.SortAlpha {
		sort.Slice(subfolders, func(i, j int) bool {
			return strings.ToLower(subfolders[i].Name) < strings.ToLower(subfolders[j].Name)
		})
	}

	for _, sf := range subfolders {
		a.renderFolder(sf, sb, indent, opts)
	}
}

func (a *Adapter) renderBookmark(b bookmark.Bookmark, sb *strings.Builder, indent int, opts output.RenderOptions) {
	title := strings.ReplaceAll(b.Title, "[", "\\[")
	title = strings.ReplaceAll(title, "]", "\\]")

	prefix := strings.Repeat("  ", indent) + "- "
	line := fmt.Sprintf("%s[%s](%s)", prefix, title, b.URL)

	var meta []string
	if opts.IncludeDates && !b.DateAdded.IsZero() {
		meta = append(meta, b.DateAdded.Format("2006-01-02"))
	}
	if opts.IncludeTags && len(b.Tags) > 0 {
		for _, tag := range b.Tags {
			meta = append(meta, "#"+tag)
		}
	}

	if len(meta) > 0 {
		line += " *(" + strings.Join(meta, ", ") + ")*"
	}

	sb.WriteString(line + "\n")
}

func (a *Adapter) renderTable(collection *bookmark.Collection, opts output.RenderOptions) string {
	var sb strings.Builder

	sb.WriteString("# Browser Bookmarks\n\n")

	if opts.IncludeMetadata {
		sb.WriteString(fmt.Sprintf("*Generated: %s*\n", time.Now().Format("2006-01-02 15:04:05")))
		sb.WriteString(fmt.Sprintf("*Platform: %s (%s)*\n", runtime.GOOS, runtime.GOARCH))
		sb.WriteString(fmt.Sprintf("*Source: %s*\n", formatSources(collection.Sources)))
		sb.WriteString(fmt.Sprintf("*Total bookmarks: %d*\n\n", collection.Count()))
	}

	bookmarks := collection.Bookmarks
	if opts.SortAlpha {
		sort.Slice(bookmarks, func(i, j int) bool {
			return strings.ToLower(bookmarks[i].Title) < strings.ToLower(bookmarks[j].Title)
		})
	}

	if opts.GroupBySource {
		grouped := groupBySourceProfile(bookmarks)
		keys := sortedGroupKeys(grouped)

		for _, key := range keys {
			bm := grouped[key]
			if len(bm) == 0 {
				continue
			}

			header := strings.Title(bm[0].Source)
			if opts.IncludeProfile && bm[0].Profile != "" {
				header += " / " + bm[0].Profile
			}
			sb.WriteString(fmt.Sprintf("## %s\n\n", header))

			a.renderTableSection(bm, &sb, opts)
			sb.WriteString("\n")
		}
	} else {
		a.renderTableSection(bookmarks, &sb, opts)
	}

	return sb.String()
}

func (a *Adapter) renderTableSection(bookmarks []bookmark.Bookmark, sb *strings.Builder, opts output.RenderOptions) {
	headers := []string{"Title", "Folder"}
	if opts.IncludeDates {
		headers = append(headers, "Date")
	}
	if opts.IncludeTags {
		headers = append(headers, "Tags")
	}

	sb.WriteString("| " + strings.Join(headers, " | ") + " |\n")
	sb.WriteString("|" + strings.Repeat("---|", len(headers)) + "\n")

	for _, b := range bookmarks {
		title := escapeTableCell(b.Title)
		link := fmt.Sprintf("[%s](%s)", title, b.URL)
		folder := escapeTableCell(strings.Join(b.FolderPath, "/"))

		row := []string{link, folder}

		if opts.IncludeDates {
			date := ""
			if !b.DateAdded.IsZero() {
				date = b.DateAdded.Format("2006-01-02")
			}
			row = append(row, date)
		}

		if opts.IncludeTags {
			tags := ""
			if len(b.Tags) > 0 {
				tagParts := make([]string, len(b.Tags))
				for i, t := range b.Tags {
					tagParts[i] = "#" + t
				}
				tags = strings.Join(tagParts, " ")
			}
			row = append(row, tags)
		}

		sb.WriteString("| " + strings.Join(row, " | ") + " |\n")
	}
}

func (a *Adapter) renderYAML(collection *bookmark.Collection, opts output.RenderOptions) string {
	var sb strings.Builder

	sb.WriteString("# Browser Bookmarks\n\n")

	if opts.IncludeMetadata {
		sb.WriteString(fmt.Sprintf("*Generated: %s*\n", time.Now().Format("2006-01-02 15:04:05")))
		sb.WriteString(fmt.Sprintf("*Platform: %s (%s)*\n", runtime.GOOS, runtime.GOARCH))
		sb.WriteString(fmt.Sprintf("*Source: %s*\n", formatSources(collection.Sources)))
		sb.WriteString(fmt.Sprintf("*Total bookmarks: %d*\n\n", collection.Count()))
	}

	bookmarks := collection.Bookmarks
	if opts.SortAlpha {
		sort.Slice(bookmarks, func(i, j int) bool {
			return strings.ToLower(bookmarks[i].Title) < strings.ToLower(bookmarks[j].Title)
		})
	}

	sb.WriteString("```yaml\n")
	sb.WriteString("bookmarks:\n")

	for _, b := range bookmarks {
		sb.WriteString(fmt.Sprintf("  - title: %s\n", yamlEscape(b.Title)))
		sb.WriteString(fmt.Sprintf("    url: %s\n", b.URL))

		if len(b.FolderPath) > 0 {
			sb.WriteString(fmt.Sprintf("    folder: %s\n", strings.Join(b.FolderPath, "/")))
		}

		if opts.IncludeDates && !b.DateAdded.IsZero() {
			sb.WriteString(fmt.Sprintf("    date: %s\n", b.DateAdded.Format("2006-01-02")))
		}

		if opts.IncludeTags && len(b.Tags) > 0 {
			sb.WriteString(fmt.Sprintf("    tags: [%s]\n", strings.Join(b.Tags, ", ")))
		}

		if opts.IncludeProfile {
			sb.WriteString(fmt.Sprintf("    source: %s\n", b.Source))
			if b.Profile != "" {
				sb.WriteString(fmt.Sprintf("    profile: %s\n", b.Profile))
			}
		}

		sb.WriteString("\n")
	}

	sb.WriteString("```\n")

	return sb.String()
}

// Helper functions

func formatSources(sources []bookmark.SourceInfo) string {
	var parts []string
	for _, s := range sources {
		part := s.Name
		if s.Profile != "" {
			part += "/" + s.Profile
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, ", ")
}

func groupBySourceProfile(bookmarks []bookmark.Bookmark) map[string][]bookmark.Bookmark {
	result := make(map[string][]bookmark.Bookmark)
	for _, b := range bookmarks {
		key := b.Source + "/" + b.Profile
		result[key] = append(result[key], b)
	}
	return result
}

func sortedGroupKeys(m map[string][]bookmark.Bookmark) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func organizeByFolder(bookmarks []bookmark.Bookmark) *folder {
	root := &folder{Name: "Bookmarks"}

	for _, b := range bookmarks {
		current := root
		for _, name := range b.FolderPath {
			var child *folder
			for _, sf := range current.Subfolders {
				if sf.Name == name {
					child = sf
					break
				}
			}
			if child == nil {
				child = &folder{Name: name}
				current.Subfolders = append(current.Subfolders, child)
			}
			current = child
		}
		current.Bookmarks = append(current.Bookmarks, b)
	}

	return root
}

func escapeTableCell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

func yamlEscape(s string) string {
	if strings.ContainsAny(s, ":#[]{}|>&*!?,\\\"'\n") || strings.HasPrefix(s, "-") || strings.HasPrefix(s, " ") {
		s = strings.ReplaceAll(s, "\"", "\\\"")
		return "\"" + s + "\""
	}
	return s
}
