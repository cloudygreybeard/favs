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

package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudygreybeard/favs/pkg/adapter"
	"github.com/cloudygreybeard/favs/pkg/bookmark"
	"github.com/cloudygreybeard/favs/pkg/config"
	"github.com/cloudygreybeard/favs/pkg/input"
	"github.com/cloudygreybeard/favs/pkg/output"
	"github.com/spf13/cobra"
)

// Input adapter preference order
var inputPreference = []string{"chrome", "firefox", "edge", "safari", "chromium", "brave"}

func runSync(cmd *cobra.Command, args []string) error {
	// Check for list mode
	if list, _ := cmd.Flags().GetBool("list"); list {
		return runListProfiles(cmd)
	}

	// Load config
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Apply flag overrides
	applyFlagOverrides(cmd, &cfg)

	// Determine mode
	allMode, _ := cmd.Flags().GetBool("all")
	browserFlag, _ := cmd.Flags().GetString("browser")
	profileFlag, _ := cmd.Flags().GetString("profile")

	// Collect bookmarks
	collection := bookmark.NewCollection()
	ctx := context.Background()

	if allMode {
		// Read from all available inputs
		if err := readAllInputs(ctx, cfg, collection); err != nil {
			return err
		}
	} else {
		// Read from preferred/specified input
		if err := readPreferredInput(ctx, cfg, browserFlag, profileFlag, collection); err != nil {
			return err
		}
	}

	if collection.Count() == 0 {
		return fmt.Errorf("no bookmarks found")
	}

	// Build filter options from config and flags
	filterOpts := bookmark.FilterOptions{
		IncludeFolders:     cfg.Pipeline.Filter.IncludeFolders,
		ExcludeFolders:     cfg.Pipeline.Filter.ExcludeFolders,
		ExcludeURLPatterns: cfg.Pipeline.Filter.ExcludeURLPatterns,
		ExcludeProtocols:   cfg.Pipeline.Filter.ExcludeProtocols,
		WarnProtocols:      cfg.Pipeline.Filter.WarnProtocols,
		MaxURLLength:       cfg.Pipeline.Filter.MaxURLLength,
		WarnURLLength:      cfg.Pipeline.Filter.WarnURLLength,
	}

	// Apply flag overrides for protocol filtering
	if excludeProtos, _ := cmd.Flags().GetStringSlice("exclude-protocols"); len(excludeProtos) > 0 {
		filterOpts.ExcludeProtocols = excludeProtos
	}
	if warnProtos, _ := cmd.Flags().GetStringSlice("warn-protocols"); len(warnProtos) > 0 {
		filterOpts.WarnProtocols = warnProtos
	}
	if maxLen, _ := cmd.Flags().GetInt("max-url-length"); maxLen > 0 {
		filterOpts.MaxURLLength = maxLen
	}
	if warnLen, _ := cmd.Flags().GetInt("warn-url-length"); warnLen > 0 {
		filterOpts.WarnURLLength = warnLen
	}

	// Apply filters
	filterResult := bookmark.Filter(collection.Bookmarks, filterOpts)

	// Log warnings
	for _, w := range filterResult.Warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
	}

	if filterResult.Excluded > 0 {
		logVerbose("Excluded %d bookmarks by filter rules", filterResult.Excluded)
	}

	filtered := filterResult.Bookmarks
	if cfg.Pipeline.Transform.Deduplicate {
		filtered = bookmark.Deduplicate(filtered)
	}

	// Update collection with filtered bookmarks
	filteredCollection := &bookmark.Collection{
		Bookmarks: filtered,
		Sources:   collection.Sources,
	}

	logVerbose("Source: %s", formatSources(collection.Sources, allMode))
	logVerbose("Bookmarks: %d", len(filtered))

	// Get output adapter
	outputFormat, _ := cmd.Flags().GetString("format")
	outAdapter, ok := adapter.GetOutput(outputFormat)
	if !ok {
		return fmt.Errorf("unknown output format: %s (available: %v)", outputFormat, adapter.ListOutputs())
	}

	// Build render options
	style, _ := cmd.Flags().GetString("style")
	renderOpts := output.RenderOptions{
		IncludeMetadata: cfg.Pipeline.Render.IncludeMetadata,
		IncludeDates:    cfg.Pipeline.Render.IncludeDates,
		IncludeTags:     cfg.Pipeline.Render.IncludeTags,
		IncludeProfile:  cfg.Pipeline.Render.IncludeProfile,
		GroupBySource:   allMode && cfg.Pipeline.Render.GroupBySource,
		SortAlpha:       cfg.Pipeline.Transform.Sort,
		Style:           style,
	}

	// Render output
	data, err := outAdapter.Render(filteredCollection, renderOpts)
	if err != nil {
		return fmt.Errorf("rendering output: %w", err)
	}

	// Write output
	outPath, _ := cmd.Flags().GetString("output")
	if outPath == "" || outPath == "-" {
		fmt.Print(string(data))
	} else {
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}
		if err := os.WriteFile(outPath, data, 0644); err != nil {
			return fmt.Errorf("writing output: %w", err)
		}
		logVerbose("Written to: %s", outPath)
	}

	return nil
}

func readPreferredInput(ctx context.Context, cfg config.Config, browserFlag, profileFlag string, collection *bookmark.Collection) error {
	var targetInput input.Adapter

	if browserFlag != "" {
		// Specific browser requested
		inp, ok := adapter.GetInput(browserFlag)
		if !ok {
			return fmt.Errorf("unknown browser: %s", browserFlag)
		}
		targetInput = inp
	} else {
		// Find first available by preference
		for _, name := range inputPreference {
			inp, ok := adapter.GetInput(name)
			if !ok {
				continue
			}
			inputCfg := cfg.GetInputConfig(name)
			if !inputCfg.Enabled {
				continue
			}
			if inp.Available() {
				targetInput = inp
				break
			}
		}
	}

	if targetInput == nil {
		return fmt.Errorf("no available browser found")
	}

	// Configure the input adapter
	inputCfg := cfg.GetInputConfig(targetInput.Name())
	if profileFlag != "" {
		inputCfg.Profile = profileFlag
	} else if inputCfg.Profile == "" {
		inputCfg.Profile = "Default"
	}

	if err := targetInput.Configure(input.Config{
		Enabled:    true,
		Profile:    inputCfg.Profile,
		CustomPath: inputCfg.CustomPath,
	}); err != nil {
		return fmt.Errorf("configuring %s: %w", targetInput.Name(), err)
	}

	logVerbose("Browser %s: reading from %s", targetInput.Name(), targetInput.Path())

	bookmarks, err := targetInput.Read(ctx)
	if err != nil {
		return fmt.Errorf("reading from %s: %w", targetInput.Name(), err)
	}

	collection.Add(bookmarks, bookmark.SourceInfo{
		Name:    targetInput.Name(),
		Profile: inputCfg.Profile,
		Path:    targetInput.Path(),
	})

	return nil
}

func readAllInputs(ctx context.Context, cfg config.Config, collection *bookmark.Collection) error {
	for _, name := range inputPreference {
		inp, ok := adapter.GetInput(name)
		if !ok {
			continue
		}

		inputCfg := cfg.GetInputConfig(name)
		if !inputCfg.Enabled {
			continue
		}

		if !inp.Available() {
			continue
		}

		// Configure without specific profile to get all
		if err := inp.Configure(input.Config{
			Enabled:    true,
			Profile:    "", // Empty = read all profiles
			CustomPath: inputCfg.CustomPath,
		}); err != nil {
			logVerbose("Browser %s: config error - %v", name, err)
			continue
		}

		logVerbose("Browser %s: reading from %s", name, inp.Path())

		bookmarks, err := inp.Read(ctx)
		if err != nil {
			logVerbose("Browser %s: error - %v", name, err)
			continue
		}

		if len(bookmarks) > 0 {
			logVerbose("Browser %s: %d bookmarks", name, len(bookmarks))

			// Determine profile from bookmarks
			profile := ""
			if len(bookmarks) > 0 {
				profile = bookmarks[0].Profile
			}

			collection.Add(bookmarks, bookmark.SourceInfo{
				Name:    name,
				Profile: profile,
				Path:    inp.Path(),
			})
		}
	}

	return nil
}

func runListProfiles(cmd *cobra.Command) error {
	fmt.Println("Available browser profiles:")
	fmt.Println()

	for _, name := range inputPreference {
		inp, ok := adapter.GetInput(name)
		if !ok {
			continue
		}

		status := "not available"
		if inp.Available() {
			status = "available"
		}

		fmt.Printf("  %s (%s)\n", inp.DisplayName(), status)
		fmt.Printf("    Path: %s\n", inp.Path())

		if inp.Available() {
			profiles, err := inp.ListProfiles()
			if err == nil && len(profiles) > 0 {
				fmt.Printf("    Profiles:\n")
				for _, p := range profiles {
					def := ""
					if p.IsDefault {
						def = " (default)"
					}
					fmt.Printf("      - %s%s\n", p.Name, def)
				}
			}
		}
		fmt.Println()
	}

	return nil
}

func formatSources(sources []bookmark.SourceInfo, allMode bool) string {
	if allMode {
		return "all / all"
	}
	if len(sources) == 0 {
		return "none"
	}
	s := sources[0]
	return fmt.Sprintf("%s / %s", s.Name, s.Profile)
}

func applyFlagOverrides(cmd *cobra.Command, cfg *config.Config) {
	if cmd.Flags().Changed("metadata") {
		cfg.Pipeline.Render.IncludeMetadata, _ = cmd.Flags().GetBool("metadata")
	}
	if cmd.Flags().Changed("group") {
		cfg.Pipeline.Render.GroupBySource, _ = cmd.Flags().GetBool("group")
	}
	if cmd.Flags().Changed("sort") {
		cfg.Pipeline.Transform.Sort, _ = cmd.Flags().GetBool("sort")
	}
}

func loadConfig() (config.Config, error) {
	path := cfgFile
	if path == "" {
		path = config.LocalPath()
	}
	if path == "" {
		path = config.DefaultPath()
	}

	cfg, err := config.Load(path)
	if err != nil && !os.IsNotExist(err) {
		return cfg, err
	}

	return cfg, nil
}

func logVerbose(format string, args ...interface{}) {
	if verbose {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}
