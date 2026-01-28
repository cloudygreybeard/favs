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

// Package opml provides an input adapter for OPML and Netscape HTML bookmark files.
package opml

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/cloudygreybeard/favs/pkg/adapter"
	"github.com/cloudygreybeard/favs/pkg/bookmark"
	"github.com/cloudygreybeard/favs/pkg/input"
)

func init() {
	adapter.RegisterInput(&Adapter{})
}

// Adapter reads bookmarks from OPML or Netscape HTML files.
type Adapter struct {
	path string
}

// Name returns the adapter identifier.
func (a *Adapter) Name() string { return "opml" }

// DisplayName returns a human-friendly name.
func (a *Adapter) DisplayName() string { return "OPML/HTML Import" }

// Available returns true if a file path is configured.
func (a *Adapter) Available() bool { return a.path != "" }

// Path returns the configured file path.
func (a *Adapter) Path() string { return a.path }

// Configure sets up the adapter with the given configuration.
func (a *Adapter) Configure(cfg input.Config) error {
	a.path = cfg.CustomPath
	return nil
}

// ListProfiles returns an empty list (not applicable for file import).
func (a *Adapter) ListProfiles() ([]input.ProfileInfo, error) {
	return nil, nil
}

// Read imports bookmarks from the configured file.
func (a *Adapter) Read(ctx context.Context) ([]bookmark.Bookmark, error) {
	if a.path == "" {
		return nil, fmt.Errorf("no file path configured")
	}

	data, err := os.ReadFile(a.path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	content := string(data)

	// Detect format and parse accordingly
	if strings.Contains(content, "<!DOCTYPE NETSCAPE-Bookmark-file") ||
		strings.Contains(content, "<DL>") {
		return a.parseNetscapeHTML(content)
	}

	return a.parseOPML(data)
}

// OPML structures
type opmlDocument struct {
	XMLName xml.Name `xml:"opml"`
	Body    opmlBody `xml:"body"`
}

type opmlBody struct {
	Outlines []opmlOutline `xml:"outline"`
}

type opmlOutline struct {
	Text     string        `xml:"text,attr"`
	Title    string        `xml:"title,attr"`
	Type     string        `xml:"type,attr"`
	HTMLURL  string        `xml:"htmlUrl,attr"`
	XMLURL   string        `xml:"xmlUrl,attr"`
	Created  string        `xml:"created,attr"`
	Children []opmlOutline `xml:"outline"`
}

func (a *Adapter) parseOPML(data []byte) ([]bookmark.Bookmark, error) {
	var doc opmlDocument
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing OPML: %w", err)
	}

	var bookmarks []bookmark.Bookmark
	a.walkOPML(doc.Body.Outlines, nil, &bookmarks)
	return bookmarks, nil
}

func (a *Adapter) walkOPML(outlines []opmlOutline, path []string, bookmarks *[]bookmark.Bookmark) {
	for _, o := range outlines {
		title := o.Text
		if title == "" {
			title = o.Title
		}

		// If it has children, it's a folder
		if len(o.Children) > 0 {
			newPath := append(path, title)
			a.walkOPML(o.Children, newPath, bookmarks)
			continue
		}

		// Get URL (prefer htmlUrl, fall back to xmlUrl for feeds)
		url := o.HTMLURL
		if url == "" {
			url = o.XMLURL
		}
		if url == "" {
			continue
		}

		b := bookmark.Bookmark{
			Title:      title,
			URL:        url,
			FolderPath: append([]string{}, path...),
			Source:     "opml",
			Profile:    "import",
		}

		// Parse created date if present
		if o.Created != "" {
			if t, err := time.Parse(time.RFC1123, o.Created); err == nil {
				b.DateAdded = t
			}
		}

		*bookmarks = append(*bookmarks, b)
	}
}

// parseNetscapeHTML parses Netscape bookmark HTML format.
// This is the format exported by most browsers.
func (a *Adapter) parseNetscapeHTML(content string) ([]bookmark.Bookmark, error) {
	var bookmarks []bookmark.Bookmark
	var currentPath []string

	// Regex patterns
	folderPattern := regexp.MustCompile(`<DT><H3[^>]*>([^<]+)</H3>`)
	linkPattern := regexp.MustCompile(`<DT><A HREF="([^"]+)"[^>]*(?:ADD_DATE="(\d+)")?[^>]*>([^<]+)</A>`)
	dlStartPattern := regexp.MustCompile(`<DL>`)
	dlEndPattern := regexp.MustCompile(`</DL>`)

	lines := strings.Split(content, "\n")
	pendingFolder := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for folder header
		if matches := folderPattern.FindStringSubmatch(line); len(matches) > 1 {
			pendingFolder = matches[1]
			continue
		}

		// Check for DL start (descend into folder)
		if dlStartPattern.MatchString(line) {
			if pendingFolder != "" {
				currentPath = append(currentPath, pendingFolder)
				pendingFolder = ""
			}
			continue
		}

		// Check for DL end (ascend from folder)
		if dlEndPattern.MatchString(line) && len(currentPath) > 0 {
			currentPath = currentPath[:len(currentPath)-1]
			continue
		}

		// Check for bookmark link
		if matches := linkPattern.FindStringSubmatch(line); len(matches) > 3 {
			url := matches[1]
			addDateStr := matches[2]
			title := matches[3]

			b := bookmark.Bookmark{
				Title:      title,
				URL:        url,
				FolderPath: append([]string{}, currentPath...),
				Source:     "html",
				Profile:    "import",
			}

			// Parse Unix timestamp for ADD_DATE
			if addDateStr != "" {
				if ts, err := parseUnixTimestamp(addDateStr); err == nil {
					b.DateAdded = ts
				}
			}

			bookmarks = append(bookmarks, b)
		}
	}

	return bookmarks, nil
}

func parseUnixTimestamp(s string) (time.Time, error) {
	var ts int64
	if _, err := fmt.Sscanf(s, "%d", &ts); err != nil {
		return time.Time{}, err
	}
	return time.Unix(ts, 0), nil
}
