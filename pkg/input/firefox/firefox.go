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

// Package firefox provides an input adapter for Firefox.
package firefox

import (
	"context"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/cloudygreybeard/favs/pkg/adapter"
	"github.com/cloudygreybeard/favs/pkg/bookmark"
	"github.com/cloudygreybeard/favs/pkg/input"
	_ "github.com/mattn/go-sqlite3"
)

// firefoxPaths maps platform to Firefox profiles directory.
var firefoxPaths = map[string]string{
	"linux":   ".mozilla/firefox",
	"darwin":  "Library/Application Support/Firefox/Profiles",
	"windows": "Mozilla/Firefox/Profiles",
}

func init() {
	adapter.RegisterInput(New())
}

// Adapter implements input.Adapter for Firefox.
type Adapter struct {
	config  input.Config
	path    string
	profile string
}

// New creates a new Firefox adapter.
func New() *Adapter {
	a := &Adapter{}
	a.path, a.profile = a.findDatabase()
	return a
}

// Name returns the adapter identifier.
func (a *Adapter) Name() string {
	return "firefox"
}

// DisplayName returns a human-friendly name.
func (a *Adapter) DisplayName() string {
	return "Mozilla Firefox"
}

// Available returns true if Firefox bookmarks are accessible.
func (a *Adapter) Available() bool {
	if a.path == "" {
		return false
	}
	_, err := os.Stat(a.path)
	return err == nil
}

// Configure applies configuration to the adapter.
func (a *Adapter) Configure(cfg input.Config) error {
	a.config = cfg
	a.path, a.profile = a.findDatabase()
	return nil
}

// Path returns the database path.
func (a *Adapter) Path() string {
	return a.path
}

// ListProfiles returns available Firefox profiles.
func (a *Adapter) ListProfiles() ([]input.ProfileInfo, error) {
	profilesDir := a.profilesDir()
	if profilesDir == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		return nil, nil
	}

	var profiles []input.ProfileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			placesPath := filepath.Join(profilesDir, entry.Name(), "places.sqlite")
			if _, err := os.Stat(placesPath); err == nil {
				profiles = append(profiles, input.ProfileInfo{
					Name:      entry.Name(),
					Path:      placesPath,
					IsDefault: entry.Name() == a.profile,
				})
			}
		}
	}

	return profiles, nil
}

// Read returns all bookmarks from Firefox.
func (a *Adapter) Read(ctx context.Context) ([]bookmark.Bookmark, error) {
	if a.path == "" {
		return nil, nil
	}

	// Firefox locks the database, so copy it first
	tmpFile, err := os.CreateTemp("", "firefox-places-*.sqlite")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	srcFile, err := os.Open(a.path)
	if err != nil {
		return nil, err
	}
	defer srcFile.Close()

	if _, err := io.Copy(tmpFile, srcFile); err != nil {
		return nil, err
	}
	tmpFile.Close()

	db, err := sql.Open("sqlite3", tmpFile.Name()+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	return a.readFromDB(db)
}

func (a *Adapter) profilesDir() string {
	relPath, ok := firefoxPaths[runtime.GOOS]
	if !ok {
		return ""
	}

	var base string
	if runtime.GOOS == "windows" {
		base = os.Getenv("APPDATA")
	} else {
		base, _ = os.UserHomeDir()
	}

	return filepath.Join(base, relPath)
}

func (a *Adapter) findDatabase() (string, string) {
	if a.config.CustomPath != "" {
		profile := filepath.Base(filepath.Dir(a.config.CustomPath))
		return a.config.CustomPath, profile
	}

	profilesDir := a.profilesDir()
	if profilesDir == "" {
		return "", ""
	}

	if _, err := os.Stat(profilesDir); err != nil {
		return "", ""
	}

	// If profile specified, use it directly
	if a.config.Profile != "" {
		return filepath.Join(profilesDir, a.config.Profile, "places.sqlite"), a.config.Profile
	}

	// Find first profile with places.sqlite
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		return "", ""
	}

	for _, entry := range entries {
		if entry.IsDir() {
			placesPath := filepath.Join(profilesDir, entry.Name(), "places.sqlite")
			if _, err := os.Stat(placesPath); err == nil {
				return placesPath, entry.Name()
			}
		}
	}

	return "", ""
}

func (a *Adapter) readFromDB(db *sql.DB) ([]bookmark.Bookmark, error) {
	// Build folder hierarchy and identify tag folders
	folders := make(map[int64]struct {
		Parent int64
		Title  string
	})

	var tagsRootID int64 = 4

	folderRows, err := db.Query("SELECT id, parent, title FROM moz_bookmarks WHERE type = 2")
	if err != nil {
		return nil, err
	}
	defer folderRows.Close()

	for folderRows.Next() {
		var id, parent int64
		var title sql.NullString
		if err := folderRows.Scan(&id, &parent, &title); err != nil {
			continue
		}
		folders[id] = struct {
			Parent int64
			Title  string
		}{Parent: parent, Title: title.String}
	}

	// Build tag lookup
	tagsByURL := make(map[string][]string)
	tagRows, err := db.Query(`
		SELECT p.url, tag_folder.title
		FROM moz_bookmarks b
		JOIN moz_places p ON b.fk = p.id
		JOIN moz_bookmarks tag_folder ON b.parent = tag_folder.id
		WHERE tag_folder.parent = ?
		  AND p.url IS NOT NULL
		  AND tag_folder.title IS NOT NULL
	`, tagsRootID)
	if err == nil {
		defer tagRows.Close()
		for tagRows.Next() {
			var url, tag string
			if err := tagRows.Scan(&url, &tag); err == nil {
				tagsByURL[url] = append(tagsByURL[url], tag)
			}
		}
	}

	// Get bookmarks
	rows, err := db.Query(`
		SELECT b.id, b.title, p.url, b.parent, b.dateAdded
		FROM moz_bookmarks b
		JOIN moz_places p ON b.fk = p.id
		WHERE b.type = 1 
		  AND p.url IS NOT NULL
		  AND p.url NOT LIKE 'place:%'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookmarks []bookmark.Bookmark
	seen := make(map[string]bool)

	for rows.Next() {
		var id int64
		var title sql.NullString
		var url string
		var parentID int64
		var dateAdded sql.NullInt64

		if err := rows.Scan(&id, &title, &url, &parentID, &dateAdded); err != nil {
			continue
		}

		if isUnderTagsRoot(parentID, folders, tagsRootID) {
			continue
		}

		if seen[url] {
			continue
		}
		seen[url] = true

		// Build folder path
		var folderPath []string
		currentParent := parentID
		for {
			folder, ok := folders[currentParent]
			if !ok {
				break
			}
			if folder.Title != "" {
				folderPath = append([]string{folder.Title}, folderPath...)
			}
			currentParent = folder.Parent
		}

		bookmarkTitle := title.String
		if bookmarkTitle == "" {
			bookmarkTitle = url
		}

		var addedTime time.Time
		if dateAdded.Valid && dateAdded.Int64 > 0 {
			addedTime = time.Unix(0, dateAdded.Int64*1000)
		}

		bookmarks = append(bookmarks, bookmark.Bookmark{
			Title:      bookmarkTitle,
			URL:        url,
			FolderPath: folderPath,
			DateAdded:  addedTime,
			Source:     "firefox",
			Profile:    a.profile,
			Tags:       tagsByURL[url],
		})
	}

	return bookmarks, nil
}

func isUnderTagsRoot(folderID int64, folders map[int64]struct {
	Parent int64
	Title  string
}, tagsRootID int64) bool {
	current := folderID
	for i := 0; i < 10; i++ {
		if current == tagsRootID {
			return true
		}
		folder, ok := folders[current]
		if !ok {
			return false
		}
		current = folder.Parent
	}
	return false
}
