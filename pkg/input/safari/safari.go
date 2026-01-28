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

// Package safari provides an input adapter for Safari (macOS only).
package safari

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	"github.com/cloudygreybeard/favs/pkg/adapter"
	"github.com/cloudygreybeard/favs/pkg/bookmark"
	"github.com/cloudygreybeard/favs/pkg/input"
	"howett.net/plist"
)

func init() {
	adapter.RegisterInput(New())
}

// Adapter implements input.Adapter for Safari.
type Adapter struct {
	config input.Config
	path   string
}

// New creates a new Safari adapter.
func New() *Adapter {
	a := &Adapter{}
	a.path = a.bookmarkPath()
	return a
}

// Name returns the adapter identifier.
func (a *Adapter) Name() string {
	return "safari"
}

// DisplayName returns a human-friendly name.
func (a *Adapter) DisplayName() string {
	return "Apple Safari"
}

// Available returns true if Safari bookmarks are accessible.
func (a *Adapter) Available() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	if a.path == "" {
		return false
	}
	_, err := os.Stat(a.path)
	return err == nil
}

// Configure applies configuration to the adapter.
func (a *Adapter) Configure(cfg input.Config) error {
	a.config = cfg
	a.path = a.bookmarkPath()
	return nil
}

// Path returns the bookmarks plist path.
func (a *Adapter) Path() string {
	return a.path
}

// ListProfiles returns available profiles (Safari has only one).
func (a *Adapter) ListProfiles() ([]input.ProfileInfo, error) {
	if !a.Available() {
		return nil, nil
	}
	return []input.ProfileInfo{
		{Name: "default", Path: a.path, IsDefault: true},
	}, nil
}

// Read returns all bookmarks from Safari.
func (a *Adapter) Read(ctx context.Context) ([]bookmark.Bookmark, error) {
	if runtime.GOOS != "darwin" {
		return nil, nil
	}

	if a.path == "" {
		return nil, nil
	}

	file, err := os.Open(a.path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var root safariBookmark
	decoder := plist.NewDecoder(file)
	if err := decoder.Decode(&root); err != nil {
		return nil, err
	}

	var bookmarks []bookmark.Bookmark
	a.parseBookmarks(root, []string{}, &bookmarks)

	return bookmarks, nil
}

func (a *Adapter) bookmarkPath() string {
	if a.config.CustomPath != "" {
		return a.config.CustomPath
	}

	if runtime.GOOS != "darwin" {
		return ""
	}

	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Safari", "Bookmarks.plist")
}

type safariBookmark struct {
	WebBookmarkType string            `plist:"WebBookmarkType"`
	Title           string            `plist:"Title"`
	URLString       string            `plist:"URLString"`
	URIDictionary   map[string]string `plist:"URIDictionary"`
	Children        []safariBookmark  `plist:"Children"`
}

func (a *Adapter) parseBookmarks(node safariBookmark, path []string, bookmarks *[]bookmark.Bookmark) {
	switch node.WebBookmarkType {
	case "WebBookmarkTypeLeaf":
		url := node.URLString
		if url == "" && node.URIDictionary != nil {
			url = node.URIDictionary[""]
		}

		title := node.Title
		if title == "" && node.URIDictionary != nil {
			title = node.URIDictionary["title"]
		}
		if title == "" {
			title = url
		}

		if url != "" {
			*bookmarks = append(*bookmarks, bookmark.Bookmark{
				Title:      title,
				URL:        url,
				FolderPath: path,
				Source:     "safari",
				Profile:    "default",
			})
		}

	case "WebBookmarkTypeList":
		currentPath := path
		if node.Title != "" {
			currentPath = append(append([]string{}, path...), node.Title)
		}
		for _, child := range node.Children {
			a.parseBookmarks(child, currentPath, bookmarks)
		}

	default:
		for _, child := range node.Children {
			a.parseBookmarks(child, path, bookmarks)
		}
	}
}
