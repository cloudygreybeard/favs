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

// Package cmd implements the favs CLI commands.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	// Import adapters to trigger init() registration
	_ "github.com/cloudygreybeard/favs/pkg/input/chromium"
	_ "github.com/cloudygreybeard/favs/pkg/input/firefox"
	_ "github.com/cloudygreybeard/favs/pkg/input/opml"
	_ "github.com/cloudygreybeard/favs/pkg/input/safari"
	_ "github.com/cloudygreybeard/favs/pkg/output/json"
	_ "github.com/cloudygreybeard/favs/pkg/output/markdown"
	_ "github.com/cloudygreybeard/favs/pkg/output/opml"
	_ "github.com/cloudygreybeard/favs/pkg/output/yaml"
)

var (
	cfgFile string
	verbose bool
)

// rootCmd represents the base command.
var rootCmd = &cobra.Command{
	Use:   "favs",
	Short: "Your bookmarks as context for AI assistants",
	Long: `favs aggregates bookmarks from browsers and services,
converting them to structured formats for AI assistant reference.

By default, output goes to stdout. Use -o/--output to write to a file.

Supported browsers:
  - Chrome, Edge, Chromium, Brave (all platforms)
  - Firefox (all platforms)
  - Safari (macOS only)

Output formats:
  - markdown: Nested lists, tables, or embedded YAML
  - json: Structured JSON
  - yaml: Structured YAML

Examples:
  favs                           # First available browser, default profile
  favs -o bookmarks.md           # Write to file
  favs -b firefox                # Specific browser
  favs -b chrome -p Work         # Specific browser and profile
  favs --all                     # All browsers and profiles
  favs --format json             # JSON output
  favs --style table             # Markdown table format
  favs serve                     # Run as MCP server
  favs adapters                  # List available adapters
  favs --list                    # List available browsers/profiles`,
	RunE: runSync,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default: ./favs.yaml or ~/.favs/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output to stderr")

	rootCmd.Flags().StringP("output", "o", "", "output file (default: stdout)")
	rootCmd.Flags().StringP("browser", "b", "", "browser to use (default: first available)")
	rootCmd.Flags().StringP("profile", "p", "", "profile name (default: Default or first found)")
	rootCmd.Flags().Bool("all", false, "read from all available browsers and profiles")
	rootCmd.Flags().Bool("group", true, "group bookmarks by browser (with --all)")
	rootCmd.Flags().Bool("metadata", true, "include metadata header")
	rootCmd.Flags().Bool("nested", true, "use nested list format (textual style)")
	rootCmd.Flags().Bool("sort", false, "sort alphabetically")
	rootCmd.Flags().Bool("list", false, "list available browser profiles and exit")
	rootCmd.Flags().String("style", "textual", "output style: textual, table, or yaml (markdown only)")
	rootCmd.Flags().String("format", "markdown", "output format: markdown, json, or yaml")

	// URL protocol filtering flags
	rootCmd.Flags().StringSlice("exclude-protocols", nil, "protocols to exclude (e.g., data,javascript)")
	rootCmd.Flags().StringSlice("warn-protocols", nil, "protocols that trigger warnings (e.g., file,chrome)")
	rootCmd.Flags().Int("max-url-length", 0, "exclude URLs longer than this (0 = use config default)")
	rootCmd.Flags().Int("warn-url-length", 0, "warn on URLs longer than this (0 = use config default)")

	rootCmd.Version = Version
	rootCmd.SetVersionTemplate(fmt.Sprintf("favs %s (commit: %s, built: %s)\n", Version, Commit, Date))

	// Add subcommands
	rootCmd.AddCommand(adaptersCmd)
}
