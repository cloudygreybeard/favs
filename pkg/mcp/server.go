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

// Package mcp provides an MCP (Model Context Protocol) server for bookmarks.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/cloudygreybeard/favs/pkg/adapter"
	"github.com/cloudygreybeard/favs/pkg/bookmark"
	"github.com/cloudygreybeard/favs/pkg/config"
	"github.com/cloudygreybeard/favs/pkg/input"
	"github.com/cloudygreybeard/favs/pkg/output"
)

// Server implements an MCP server for bookmark resources.
type Server struct {
	config  config.Config
	cache   *bookmark.Collection
	cacheMu sync.RWMutex
}

// NewServer creates a new MCP server.
func NewServer(cfg config.Config) *Server {
	return &Server{config: cfg}
}

// Run starts the MCP server, reading JSON-RPC from stdin and writing to stdout.
func (s *Server) Run(ctx context.Context) error {
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var req Request
		if err := decoder.Decode(&req); err != nil {
			if err == io.EOF {
				return nil
			}
			continue
		}

		resp := s.handleRequest(ctx, &req)
		if err := encoder.Encode(resp); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding response: %v\n", err)
		}
	}
}

func (s *Server) handleRequest(ctx context.Context, req *Request) *Response {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "resources/list":
		return s.handleResourcesList(req)
	case "resources/read":
		return s.handleResourcesRead(ctx, req)
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	default:
		return errorResponse(req.ID, -32601, "Method not found")
	}
}

func (s *Server) handleInitialize(req *Request) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"serverInfo": map[string]string{
				"name":    "favs",
				"version": "1.0.0",
			},
			"capabilities": map[string]interface{}{
				"resources": map[string]bool{
					"subscribe":   false,
					"listChanged": false,
				},
				"tools": map[string]interface{}{},
			},
		},
	}
}

func (s *Server) handleResourcesList(req *Request) *Response {
	resources := []Resource{
		{
			URI:         "favs://all",
			Name:        "All Bookmarks",
			Description: "All browser bookmarks in JSON format",
			MimeType:    "application/json",
		},
		{
			URI:         "favs://markdown",
			Name:        "Bookmarks (Markdown)",
			Description: "All browser bookmarks in Markdown format",
			MimeType:    "text/markdown",
		},
	}

	// Add per-browser resources
	for _, name := range adapter.ListInputs() {
		inp, ok := adapter.GetInput(name)
		if !ok || !inp.Available() {
			continue
		}
		resources = append(resources, Resource{
			URI:         fmt.Sprintf("favs://%s", name),
			Name:        fmt.Sprintf("%s Bookmarks", inp.DisplayName()),
			Description: fmt.Sprintf("Bookmarks from %s", inp.DisplayName()),
			MimeType:    "application/json",
		})
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"resources": resources,
		},
	}
}

func (s *Server) handleResourcesRead(ctx context.Context, req *Request) *Response {
	var params struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResponse(req.ID, -32602, "Invalid params")
	}

	collection, err := s.getBookmarks(ctx, params.URI)
	if err != nil {
		return errorResponse(req.ID, -32000, err.Error())
	}

	// Determine format from URI
	format := "json"
	if params.URI == "favs://markdown" {
		format = "markdown"
	}

	outAdapter, ok := adapter.GetOutput(format)
	if !ok {
		return errorResponse(req.ID, -32000, "Output adapter not found")
	}

	data, err := outAdapter.Render(collection, output.DefaultRenderOptions())
	if err != nil {
		return errorResponse(req.ID, -32000, err.Error())
	}

	mimeType := "application/json"
	if format == "markdown" {
		mimeType = "text/markdown"
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"uri":      params.URI,
					"mimeType": mimeType,
					"text":     string(data),
				},
			},
		},
	}
}

func (s *Server) handleToolsList(req *Request) *Response {
	tools := []Tool{
		{
			Name:        "sync_bookmarks",
			Description: "Refresh bookmarks from all available browsers",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "search_bookmarks",
			Description: "Search bookmarks by title or URL",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query",
					},
				},
				"required": []string{"query"},
			},
		},
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}
}

func (s *Server) handleToolsCall(ctx context.Context, req *Request) *Response {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResponse(req.ID, -32602, "Invalid params")
	}

	switch params.Name {
	case "sync_bookmarks":
		return s.toolSyncBookmarks(ctx, req)
	case "search_bookmarks":
		return s.toolSearchBookmarks(ctx, req, params.Arguments)
	default:
		return errorResponse(req.ID, -32602, "Unknown tool")
	}
}

func (s *Server) toolSyncBookmarks(ctx context.Context, req *Request) *Response {
	// Clear cache to force refresh
	s.cacheMu.Lock()
	s.cache = nil
	s.cacheMu.Unlock()

	collection, err := s.getBookmarks(ctx, "favs://all")
	if err != nil {
		return errorResponse(req.ID, -32000, err.Error())
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("Synced %d bookmarks from %d sources", collection.Count(), len(collection.Sources)),
				},
			},
		},
	}
}

func (s *Server) toolSearchBookmarks(ctx context.Context, req *Request, args json.RawMessage) *Response {
	var searchArgs struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(args, &searchArgs); err != nil {
		return errorResponse(req.ID, -32602, "Invalid search arguments")
	}

	collection, err := s.getBookmarks(ctx, "favs://all")
	if err != nil {
		return errorResponse(req.ID, -32000, err.Error())
	}

	// Simple search
	var matches []bookmark.Bookmark
	query := searchArgs.Query
	for _, b := range collection.Bookmarks {
		if containsIgnoreCase(b.Title, query) || containsIgnoreCase(b.URL, query) {
			matches = append(matches, b)
		}
	}

	// Format results
	var results []map[string]string
	for _, b := range matches {
		results = append(results, map[string]string{
			"title": b.Title,
			"url":   b.URL,
		})
	}

	resultJSON, _ := json.MarshalIndent(results, "", "  ")

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("Found %d matches:\n%s", len(matches), string(resultJSON)),
				},
			},
		},
	}
}

func (s *Server) getBookmarks(ctx context.Context, uri string) (*bookmark.Collection, error) {
	// Check cache first
	s.cacheMu.RLock()
	if s.cache != nil {
		cached := s.cache
		s.cacheMu.RUnlock()
		return cached, nil
	}
	s.cacheMu.RUnlock()

	// Read from all available inputs
	collection := bookmark.NewCollection()

	for _, name := range adapter.ListInputs() {
		inp, ok := adapter.GetInput(name)
		if !ok {
			continue
		}

		inputCfg := s.config.GetInputConfig(name)
		if !inputCfg.Enabled {
			continue
		}

		if !inp.Available() {
			continue
		}

		if err := inp.Configure(input.Config{
			Enabled:    true,
			Profile:    "",
			CustomPath: inputCfg.CustomPath,
		}); err != nil {
			continue
		}

		bookmarks, err := inp.Read(ctx)
		if err != nil {
			continue
		}

		if len(bookmarks) > 0 {
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

	// Update cache
	s.cacheMu.Lock()
	s.cache = collection
	s.cacheMu.Unlock()

	return collection, nil
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && containsIgnoreCaseImpl(s, substr)))
}

func containsIgnoreCaseImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalIgnoreCase(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func equalIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func errorResponse(id interface{}, code int, message string) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
		},
	}
}

// MCP Protocol types

// Request represents a JSON-RPC request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC response.
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

// Error represents a JSON-RPC error.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Resource represents an MCP resource.
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// Tool represents an MCP tool.
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}
