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

package bookmark

import (
	"fmt"
	"regexp"
	"strings"
)

// FilterOptions configures bookmark filtering.
type FilterOptions struct {
	IncludeFolders     []string // Only include bookmarks in these folders
	ExcludeFolders     []string // Exclude bookmarks in these folders
	ExcludeURLPatterns []string // Exclude URLs matching these regex patterns

	// URL protocol filtering
	ExcludeProtocols []string // Protocols to exclude (e.g., "data", "javascript")
	WarnProtocols    []string // Protocols to warn about but include
	MaxURLLength     int      // Exclude URLs longer than this (0 = no limit)
	WarnURLLength    int      // Warn on URLs longer than this (0 = no warning)
}

// FilterResult contains the filtered bookmarks and any warnings generated.
type FilterResult struct {
	Bookmarks []Bookmark
	Warnings  []string
	Excluded  int // Count of excluded bookmarks
}

// Filter applies filters to a collection of bookmarks.
func Filter(bookmarks []Bookmark, opts FilterOptions) FilterResult {
	var patterns []*regexp.Regexp
	for _, p := range opts.ExcludeURLPatterns {
		if re, err := regexp.Compile(p); err == nil {
			patterns = append(patterns, re)
		}
	}

	// Build protocol lookup maps for efficiency
	excludeProtos := make(map[string]bool)
	for _, p := range opts.ExcludeProtocols {
		excludeProtos[strings.ToLower(p)] = true
	}
	warnProtos := make(map[string]bool)
	for _, p := range opts.WarnProtocols {
		warnProtos[strings.ToLower(p)] = true
	}

	var result FilterResult
	for _, b := range bookmarks {
		folderStr := strings.Join(b.FolderPath, "/")
		excluded := false
		var reason string

		// Extract protocol from URL
		proto := extractProtocol(b.URL)

		// Check protocol exclusion
		if excludeProtos[proto] {
			excluded = true
			reason = fmt.Sprintf("excluded protocol '%s'", proto)
		}

		// Check URL length exclusion
		if !excluded && opts.MaxURLLength > 0 && len(b.URL) > opts.MaxURLLength {
			excluded = true
			reason = fmt.Sprintf("URL length %d exceeds max %d", len(b.URL), opts.MaxURLLength)
		}

		// Check folder inclusion
		if !excluded && len(opts.IncludeFolders) > 0 {
			matched := false
			for _, inc := range opts.IncludeFolders {
				if strings.Contains(folderStr, inc) {
					matched = true
					break
				}
			}
			if !matched {
				excluded = true
				reason = "not in included folders"
			}
		}

		// Check folder exclusion
		if !excluded {
			for _, exc := range opts.ExcludeFolders {
				if strings.Contains(folderStr, exc) {
					excluded = true
					reason = fmt.Sprintf("in excluded folder '%s'", exc)
					break
				}
			}
		}

		// Check URL pattern exclusion
		if !excluded {
			for _, p := range patterns {
				if p.MatchString(b.URL) {
					excluded = true
					reason = "matches excluded URL pattern"
					break
				}
			}
		}

		if excluded {
			result.Excluded++
			// Generate warning for excluded bookmarks (optional, could be verbose mode only)
			_ = reason // reason available for verbose logging if needed
			continue
		}

		// Generate warnings for included bookmarks
		if warnProtos[proto] {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("bookmark '%s' uses protocol '%s': %s", truncate(b.Title, 40), proto, truncate(b.URL, 60)))
		}

		if opts.WarnURLLength > 0 && len(b.URL) > opts.WarnURLLength {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("bookmark '%s' has long URL (%d chars): %s", truncate(b.Title, 40), len(b.URL), truncate(b.URL, 60)))
		}

		result.Bookmarks = append(result.Bookmarks, b)
	}

	return result
}

// extractProtocol extracts the protocol/scheme from a URL.
func extractProtocol(url string) string {
	idx := strings.Index(url, ":")
	if idx <= 0 {
		return ""
	}
	return strings.ToLower(url[:idx])
}

// truncate shortens a string to maxLen, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// Deduplicate removes duplicate bookmarks by URL.
func Deduplicate(bookmarks []Bookmark) []Bookmark {
	seen := make(map[string]bool)
	var result []Bookmark

	for _, b := range bookmarks {
		if !seen[b.URL] {
			seen[b.URL] = true
			result = append(result, b)
		}
	}

	return result
}
