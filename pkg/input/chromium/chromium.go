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

// Package chromium provides an input adapter for Chromium-based browsers.
package chromium

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/cloudygreybeard/favs/pkg/adapter"
	"github.com/cloudygreybeard/favs/pkg/bookmark"
	"github.com/cloudygreybeard/favs/pkg/input"
)

// chromiumPaths maps browser names to their config directories per platform.
var chromiumPaths = map[string]map[string]string{
	"chrome": {
		"linux":   ".config/google-chrome",
		"darwin":  "Library/Application Support/Google/Chrome",
		"windows": "Google/Chrome/User Data",
	},
	"edge": {
		"linux":   ".config/microsoft-edge",
		"darwin":  "Library/Application Support/Microsoft Edge",
		"windows": "Microsoft/Edge/User Data",
	},
	"chromium": {
		"linux":   ".config/chromium",
		"darwin":  "Library/Application Support/Chromium",
		"windows": "Chromium/User Data",
	},
	"brave": {
		"linux":   ".config/BraveSoftware/Brave-Browser",
		"darwin":  "Library/Application Support/BraveSoftware/Brave-Browser",
		"windows": "BraveSoftware/Brave-Browser/User Data",
	},
}

var displayNames = map[string]string{
	"chrome":   "Google Chrome",
	"edge":     "Microsoft Edge",
	"chromium": "Chromium",
	"brave":    "Brave",
}

// Difference between Chrome epoch (1601-01-01) and Unix epoch (1970-01-01) in seconds
const chromeToUnixEpochDelta = 11644473600

func init() {
	// Register all Chromium-based browser adapters
	adapter.RegisterInput(New("chrome"))
	adapter.RegisterInput(New("edge"))
	adapter.RegisterInput(New("chromium"))
	adapter.RegisterInput(New("brave"))
}

// Adapter implements input.Adapter for Chromium-based browsers.
type Adapter struct {
	browser  string
	config   input.Config
	profiles []profileInfo
}

type profileInfo struct {
	name string
	path string
}

// New creates a new Chromium adapter for the specified browser.
func New(browser string) *Adapter {
	a := &Adapter{browser: browser}
	a.profiles = a.discoverProfiles()
	return a
}

// Name returns the adapter identifier.
func (a *Adapter) Name() string {
	return a.browser
}

// DisplayName returns a human-friendly name.
func (a *Adapter) DisplayName() string {
	if name, ok := displayNames[a.browser]; ok {
		return name
	}
	return strings.Title(a.browser)
}

// Available returns true if bookmarks are accessible.
func (a *Adapter) Available() bool {
	if a.config.CustomPath != "" {
		_, err := os.Stat(a.config.CustomPath)
		return err == nil
	}
	return len(a.profiles) > 0
}

// Configure applies configuration to the adapter.
func (a *Adapter) Configure(cfg input.Config) error {
	a.config = cfg
	if cfg.CustomPath == "" {
		a.profiles = a.discoverProfiles()
	}
	return nil
}

// Path returns the path being read.
func (a *Adapter) Path() string {
	if a.config.CustomPath != "" {
		return a.config.CustomPath
	}

	if len(a.profiles) == 0 {
		return a.basePath() + " (no profiles found)"
	}

	if len(a.profiles) == 1 {
		return a.profiles[0].path
	}

	var names []string
	for _, p := range a.profiles {
		names = append(names, p.name)
	}
	return a.basePath() + " [" + strings.Join(names, ", ") + "]"
}

// ListProfiles returns available profiles.
func (a *Adapter) ListProfiles() ([]input.ProfileInfo, error) {
	var result []input.ProfileInfo
	for i, p := range a.profiles {
		result = append(result, input.ProfileInfo{
			Name:      p.name,
			Path:      p.path,
			IsDefault: i == 0 || p.name == "Default",
		})
	}
	return result, nil
}

// Read returns all bookmarks from the browser.
func (a *Adapter) Read(ctx context.Context) ([]bookmark.Bookmark, error) {
	if a.config.CustomPath != "" {
		return a.readFromPath(a.config.CustomPath, "custom")
	}

	if len(a.profiles) == 0 {
		return nil, nil
	}

	// If specific profile requested, find it
	if a.config.Profile != "" {
		for _, profile := range a.profiles {
			if profile.name == a.config.Profile {
				return a.readFromPath(profile.path, profile.name)
			}
		}

		// If "Default" was requested but not found, use first available
		if a.config.Profile == "Default" && len(a.profiles) > 0 {
			first := a.profiles[0]
			return a.readFromPath(first.path, first.name)
		}

		return nil, nil
	}

	// No profile specified: read all profiles
	var allBookmarks []bookmark.Bookmark
	for _, profile := range a.profiles {
		bookmarks, err := a.readFromPath(profile.path, profile.name)
		if err != nil {
			continue
		}
		allBookmarks = append(allBookmarks, bookmarks...)
	}

	return allBookmarks, nil
}

func (a *Adapter) basePath() string {
	paths, ok := chromiumPaths[a.browser]
	if !ok {
		return ""
	}

	relPath, ok := paths[runtime.GOOS]
	if !ok {
		return ""
	}

	var base string
	if runtime.GOOS == "windows" {
		base = os.Getenv("LOCALAPPDATA")
	} else {
		base, _ = os.UserHomeDir()
	}

	return filepath.Join(base, relPath)
}

func (a *Adapter) discoverProfiles() []profileInfo {
	if a.config.CustomPath != "" {
		return nil
	}

	basePath := a.basePath()
	if basePath == "" {
		return nil
	}

	entries, err := os.ReadDir(basePath)
	if err != nil {
		return nil
	}

	var profiles []profileInfo

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		if name == "Default" || strings.HasPrefix(name, "Profile ") {
			bookmarkPath := filepath.Join(basePath, name, "Bookmarks")
			if _, err := os.Stat(bookmarkPath); err == nil {
				profiles = append(profiles, profileInfo{
					name: name,
					path: bookmarkPath,
				})
			}
		}
	}

	return profiles
}

func (a *Adapter) readFromPath(path, profile string) ([]bookmark.Bookmark, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var chromiumData struct {
		Roots map[string]json.RawMessage `json:"roots"`
	}
	if err := json.Unmarshal(data, &chromiumData); err != nil {
		return nil, err
	}

	var bookmarks []bookmark.Bookmark
	for _, rootData := range chromiumData.Roots {
		var node chromiumNode
		if err := json.Unmarshal(rootData, &node); err != nil {
			continue
		}
		if node.Type == "folder" {
			a.parseFolder(node, []string{}, profile, &bookmarks)
		}
	}

	return bookmarks, nil
}

type chromiumNode struct {
	Type      string         `json:"type"`
	Name      string         `json:"name"`
	URL       string         `json:"url"`
	DateAdded string         `json:"date_added"`
	Children  []chromiumNode `json:"children"`
}

func (a *Adapter) parseFolder(node chromiumNode, path []string, profile string, bookmarks *[]bookmark.Bookmark) {
	currentPath := path
	if node.Name != "" {
		currentPath = append(append([]string{}, path...), node.Name)
	}

	for _, child := range node.Children {
		switch child.Type {
		case "url":
			*bookmarks = append(*bookmarks, bookmark.Bookmark{
				Title:      child.Name,
				URL:        child.URL,
				FolderPath: currentPath,
				DateAdded:  parseChromiumDate(child.DateAdded),
				Source:     a.browser,
				Profile:    profile,
			})
		case "folder":
			a.parseFolder(child, currentPath, profile, bookmarks)
		}
	}
}

func parseChromiumDate(dateStr string) time.Time {
	if dateStr == "" {
		return time.Time{}
	}

	chromeMicroseconds, err := strconv.ParseInt(dateStr, 10, 64)
	if err != nil || chromeMicroseconds == 0 {
		return time.Time{}
	}

	chromeSeconds := chromeMicroseconds / 1000000
	unixSeconds := chromeSeconds - chromeToUnixEpochDelta
	microRemainder := chromeMicroseconds % 1000000

	return time.Unix(unixSeconds, microRemainder*1000)
}
