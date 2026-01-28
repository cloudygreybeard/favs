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

// Package input provides the InputAdapter interface for bookmark sources.
//
// Input adapters are responsible for reading bookmarks from external sources
// such as browsers, bookmark services, or file formats. Each adapter is
// registered with the global registry and can be discovered at runtime.
//
// # Implementing an Input Adapter
//
// To create a new input adapter:
//
//  1. Create a new package under pkg/input/
//  2. Implement the Adapter interface
//  3. Register via init() using adapter.RegisterInput()
//  4. Import in cmd/root.go to include in the build
//
// Example:
//
//	package myservice
//
//	import (
//	    "context"
//	    "github.com/cloudygreybeard/favs/pkg/adapter"
//	    "github.com/cloudygreybeard/favs/pkg/bookmark"
//	    "github.com/cloudygreybeard/favs/pkg/input"
//	)
//
//	func init() {
//	    adapter.RegisterInput(New())
//	}
//
//	type Adapter struct {
//	    config input.Config
//	}
//
//	func New() *Adapter { return &Adapter{} }
//
//	func (a *Adapter) Name() string        { return "myservice" }
//	func (a *Adapter) DisplayName() string { return "My Service" }
//	func (a *Adapter) Available() bool     { return true }
//	func (a *Adapter) Path() string        { return "https://myservice.com" }
//
//	func (a *Adapter) Configure(cfg input.Config) error {
//	    a.config = cfg
//	    return nil
//	}
//
//	func (a *Adapter) ListProfiles() ([]input.ProfileInfo, error) {
//	    return []input.ProfileInfo{{Name: "default", IsDefault: true}}, nil
//	}
//
//	func (a *Adapter) Read(ctx context.Context) ([]bookmark.Bookmark, error) {
//	    // Implement reading logic here
//	    return nil, nil
//	}
//
// See docs/adapters.md for comprehensive documentation.
package input

import (
	"context"

	"github.com/cloudygreybeard/favs/pkg/bookmark"
)

// Adapter is the interface for bookmark input sources.
//
// Input adapters read bookmarks from external sources and convert them
// to the common bookmark.Bookmark format. Adapters should handle their
// own authentication, caching, and error recovery.
type Adapter interface {
	// Name returns the unique adapter identifier used in configuration
	// and command-line flags. Should be lowercase, alphanumeric.
	// Examples: "chrome", "firefox", "pinboard", "raindrop"
	Name() string

	// DisplayName returns a human-friendly name for UI display.
	// Examples: "Google Chrome", "Mozilla Firefox", "Pinboard"
	DisplayName() string

	// Available returns true if this input source can be read.
	// This method should be fast and not perform network I/O.
	// Check for existence of data files, configured credentials, etc.
	Available() bool

	// Path returns the path, URI, or description of what is being read.
	// Used for logging and debugging. Examples:
	//   - "/Users/name/Library/Application Support/Google/Chrome"
	//   - "https://api.pinboard.in/v1"
	//   - "~/bookmarks.html"
	Path() string

	// Configure applies runtime configuration to the adapter.
	// Called before Read() with user-specified options.
	// Should validate configuration and return errors for invalid settings.
	Configure(cfg Config) error

	// ListProfiles returns available profiles/accounts for this source.
	// Returns nil if the source doesn't support multiple profiles.
	// Used by the --list command to show available options.
	ListProfiles() ([]ProfileInfo, error)

	// Read fetches all bookmarks from this source.
	// Should respect context cancellation for long-running operations.
	// Populate bookmark.Source with Name() for proper attribution.
	Read(ctx context.Context) ([]bookmark.Bookmark, error)
}

// Config holds adapter-specific configuration passed at runtime.
type Config struct {
	// Enabled indicates whether this adapter should be used.
	Enabled bool

	// Profile specifies which profile to read.
	// Empty string means use default or read all profiles.
	Profile string

	// CustomPath overrides the default path/location for this source.
	CustomPath string

	// Options holds adapter-specific key-value options.
	// Common keys include "api_token", "username", etc.
	Options map[string]interface{}
}

// ProfileInfo describes an available profile within an input source.
// Browsers often have multiple profiles (Default, Work, Personal).
// Services may have multiple accounts.
type ProfileInfo struct {
	// Name is the profile identifier (e.g., "Default", "Profile 1").
	Name string

	// Path is the filesystem path or URI for this profile.
	Path string

	// IsDefault indicates if this is the default/primary profile.
	IsDefault bool
}
