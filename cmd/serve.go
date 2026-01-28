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
	"os/signal"
	"syscall"

	"github.com/cloudygreybeard/favs/pkg/mcp"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run as an MCP server",
	Long: `Runs favs as an MCP (Model Context Protocol) server.

The server communicates via JSON-RPC over stdin/stdout, exposing:

Resources:
  - favs://all        All bookmarks in JSON format
  - favs://markdown   All bookmarks in Markdown format
  - favs://<browser>  Bookmarks from a specific browser

Tools:
  - sync_bookmarks      Refresh bookmarks from browsers
  - search_bookmarks    Search bookmarks by title or URL

Usage with Claude Desktop or similar MCP clients:

Add to your MCP configuration:

  {
    "mcpServers": {
      "favs": {
        "command": "/path/to/favs",
        "args": ["serve"]
      }
    }
  }`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	server := mcp.NewServer(cfg)

	// Handle shutdown gracefully
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	fmt.Fprintln(os.Stderr, "favs MCP server started")
	return server.Run(ctx)
}
