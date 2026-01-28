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

// Package config provides configuration loading and management.
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the full configuration.
type Config struct {
	Inputs   InputsConfig   `yaml:"inputs"`
	Outputs  OutputsConfig  `yaml:"outputs"`
	Pipeline PipelineConfig `yaml:"pipeline"`
}

// InputsConfig configures input adapters.
type InputsConfig struct {
	Chrome   InputConfig `yaml:"chrome"`
	Edge     InputConfig `yaml:"edge"`
	Firefox  InputConfig `yaml:"firefox"`
	Safari   InputConfig `yaml:"safari"`
	Chromium InputConfig `yaml:"chromium"`
	Brave    InputConfig `yaml:"brave"`
}

// InputConfig configures a single input adapter.
type InputConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Profile    string `yaml:"profile"`
	CustomPath string `yaml:"custom_path"`
}

// OutputsConfig configures output adapters.
type OutputsConfig struct {
	Markdown OutputConfig `yaml:"markdown"`
	JSON     OutputConfig `yaml:"json"`
	YAML     OutputConfig `yaml:"yaml"`
}

// OutputConfig configures a single output adapter.
type OutputConfig struct {
	Enabled bool              `yaml:"enabled"`
	Style   string            `yaml:"style"`
	Options map[string]string `yaml:"options"`
}

// PipelineConfig configures the processing pipeline.
type PipelineConfig struct {
	Filter    FilterConfig    `yaml:"filter"`
	Transform TransformConfig `yaml:"transform"`
	Render    RenderConfig    `yaml:"render"`
}

// FilterConfig configures bookmark filtering.
type FilterConfig struct {
	IncludeFolders     []string `yaml:"include_folders"`
	ExcludeFolders     []string `yaml:"exclude_folders"`
	ExcludeURLPatterns []string `yaml:"exclude_url_patterns"`

	// URL protocol filtering
	ExcludeProtocols []string `yaml:"exclude_protocols"` // Protocols to exclude (e.g., data, javascript)
	WarnProtocols    []string `yaml:"warn_protocols"`    // Protocols to warn about but include
	MaxURLLength     int      `yaml:"max_url_length"`    // Exclude URLs longer than this (0 = no limit)
	WarnURLLength    int      `yaml:"warn_url_length"`   // Warn on URLs longer than this (0 = no warning)
}

// TransformConfig configures bookmark transformation.
type TransformConfig struct {
	Deduplicate bool `yaml:"deduplicate"`
	Sort        bool `yaml:"sort"`
}

// RenderConfig configures rendering options.
type RenderConfig struct {
	IncludeMetadata bool `yaml:"include_metadata"`
	IncludeDates    bool `yaml:"include_dates"`
	IncludeTags     bool `yaml:"include_tags"`
	IncludeProfile  bool `yaml:"include_profile"`
	GroupBySource   bool `yaml:"group_by_source"`
}

// Default returns a configuration with sensible defaults.
func Default() Config {
	return Config{
		Inputs: InputsConfig{
			Chrome:  InputConfig{Enabled: true},
			Edge:    InputConfig{Enabled: true},
			Firefox: InputConfig{Enabled: true},
			Safari:  InputConfig{Enabled: true},
		},
		Outputs: OutputsConfig{
			Markdown: OutputConfig{Enabled: true, Style: "textual"},
		},
		Pipeline: PipelineConfig{
			Filter: FilterConfig{
				ExcludeFolders:   []string{"Trash"},
				ExcludeProtocols: []string{"data", "javascript"},
				WarnProtocols:    []string{"file", "chrome", "about", "blob"},
				WarnURLLength:    2048,
			},
			Transform: TransformConfig{
				Deduplicate: false,
			},
			Render: RenderConfig{
				IncludeMetadata: true,
				IncludeDates:    true,
				IncludeTags:     true,
				IncludeProfile:  true,
				GroupBySource:   true,
			},
		},
	}
}

// Load reads configuration from a file, merging with defaults.
func Load(path string) (Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// DefaultPath returns the default config file path.
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".favs", "config.yaml")
}

// LocalPath returns a local config file path if it exists.
func LocalPath() string {
	paths := []string{
		"favs.yaml",
		"favs.yml",
		".favs.yaml",
		".favs.yml",
		"config.yaml",
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

// GetInputConfig returns the config for a specific input adapter.
func (c *Config) GetInputConfig(name string) InputConfig {
	switch name {
	case "chrome":
		return c.Inputs.Chrome
	case "edge":
		return c.Inputs.Edge
	case "firefox":
		return c.Inputs.Firefox
	case "safari":
		return c.Inputs.Safari
	case "chromium":
		return c.Inputs.Chromium
	case "brave":
		return c.Inputs.Brave
	default:
		return InputConfig{}
	}
}

// GetOutputConfig returns the config for a specific output adapter.
func (c *Config) GetOutputConfig(name string) OutputConfig {
	switch name {
	case "markdown":
		return c.Outputs.Markdown
	case "json":
		return c.Outputs.JSON
	case "yaml":
		return c.Outputs.YAML
	default:
		return OutputConfig{}
	}
}
