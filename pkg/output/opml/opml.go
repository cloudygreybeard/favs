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

// Package opml provides output adapters for OPML and Netscape HTML formats.
package opml

import (
	"encoding/xml"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/cloudygreybeard/favs/pkg/adapter"
	"github.com/cloudygreybeard/favs/pkg/bookmark"
	"github.com/cloudygreybeard/favs/pkg/output"
)

func init() {
	adapter.RegisterOutput(&OPMLAdapter{})
	adapter.RegisterOutput(&HTMLAdapter{})
}

// OPMLAdapter exports bookmarks to OPML format.
type OPMLAdapter struct{}

// Name returns the adapter identifier.
func (a *OPMLAdapter) Name() string { return "opml" }

// DisplayName returns a human-friendly name.
func (a *OPMLAdapter) DisplayName() string { return "OPML" }

// Extensions returns file extensions for this format.
func (a *OPMLAdapter) Extensions() []string { return []string{".opml", ".xml"} }

// Configure sets up the adapter.
func (a *OPMLAdapter) Configure(cfg output.Config) error { return nil }

// Render exports bookmarks to OPML format.
func (a *OPMLAdapter) Render(collection *bookmark.Collection, opts output.RenderOptions) ([]byte, error) {
	doc := opmlDocument{
		Version: "2.0",
		Head: opmlHead{
			Title:       "Bookmarks Export",
			DateCreated: time.Now().Format(time.RFC1123),
		},
	}

	// Build folder tree
	root := buildFolderTree(collection.Bookmarks)
	doc.Body.Outlines = root.toOutlines()

	data, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling OPML: %w", err)
	}

	return append([]byte(xml.Header), data...), nil
}

// OPML structures for output
type opmlDocument struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Head    opmlHead `xml:"head"`
	Body    opmlBody `xml:"body"`
}

type opmlHead struct {
	Title       string `xml:"title"`
	DateCreated string `xml:"dateCreated"`
}

type opmlBody struct {
	Outlines []opmlOutline `xml:"outline"`
}

type opmlOutline struct {
	Text     string        `xml:"text,attr"`
	Type     string        `xml:"type,attr,omitempty"`
	HTMLURL  string        `xml:"htmlUrl,attr,omitempty"`
	Created  string        `xml:"created,attr,omitempty"`
	Children []opmlOutline `xml:"outline,omitempty"`
}

// folderNode represents a folder in the bookmark hierarchy.
type folderNode struct {
	name      string
	children  map[string]*folderNode
	bookmarks []bookmark.Bookmark
}

func newFolderNode(name string) *folderNode {
	return &folderNode{
		name:     name,
		children: make(map[string]*folderNode),
	}
}

func buildFolderTree(bookmarks []bookmark.Bookmark) *folderNode {
	root := newFolderNode("")

	for _, b := range bookmarks {
		node := root
		for _, folder := range b.FolderPath {
			if _, ok := node.children[folder]; !ok {
				node.children[folder] = newFolderNode(folder)
			}
			node = node.children[folder]
		}
		node.bookmarks = append(node.bookmarks, b)
	}

	return root
}

func (n *folderNode) toOutlines() []opmlOutline {
	var outlines []opmlOutline

	// Add child folders
	for _, child := range n.children {
		outline := opmlOutline{
			Text:     child.name,
			Children: child.toOutlines(),
		}
		outlines = append(outlines, outline)
	}

	// Add bookmarks
	for _, b := range n.bookmarks {
		outline := opmlOutline{
			Text:    b.Title,
			Type:    "link",
			HTMLURL: b.URL,
		}
		if !b.DateAdded.IsZero() {
			outline.Created = b.DateAdded.Format(time.RFC1123)
		}
		outlines = append(outlines, outline)
	}

	return outlines
}

// HTMLAdapter exports bookmarks to Netscape HTML format.
type HTMLAdapter struct{}

// Name returns the adapter identifier.
func (a *HTMLAdapter) Name() string { return "html" }

// DisplayName returns a human-friendly name.
func (a *HTMLAdapter) DisplayName() string { return "Netscape HTML" }

// Extensions returns file extensions for this format.
func (a *HTMLAdapter) Extensions() []string { return []string{".html", ".htm"} }

// Configure sets up the adapter.
func (a *HTMLAdapter) Configure(cfg output.Config) error { return nil }

// Render exports bookmarks to Netscape HTML bookmark format.
func (a *HTMLAdapter) Render(collection *bookmark.Collection, opts output.RenderOptions) ([]byte, error) {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE NETSCAPE-Bookmark-file-1>
<!-- This is an automatically generated file.
     It will be read and overwritten.
     DO NOT EDIT! -->
<META HTTP-EQUIV="Content-Type" CONTENT="text/html; charset=UTF-8">
<TITLE>Bookmarks</TITLE>
<H1>Bookmarks</H1>
<DL><p>
`)

	// Build folder tree and render
	root := buildFolderTree(collection.Bookmarks)
	renderHTMLFolder(&sb, root, 1)

	sb.WriteString("</DL><p>\n")

	return []byte(sb.String()), nil
}

func renderHTMLFolder(sb *strings.Builder, node *folderNode, depth int) {
	indent := strings.Repeat("    ", depth)

	// Render child folders
	for _, child := range node.children {
		sb.WriteString(fmt.Sprintf("%s<DT><H3>%s</H3>\n", indent, html.EscapeString(child.name)))
		sb.WriteString(fmt.Sprintf("%s<DL><p>\n", indent))
		renderHTMLFolder(sb, child, depth+1)
		sb.WriteString(fmt.Sprintf("%s</DL><p>\n", indent))
	}

	// Render bookmarks
	for _, b := range node.bookmarks {
		addDate := ""
		if !b.DateAdded.IsZero() {
			addDate = fmt.Sprintf(" ADD_DATE=\"%d\"", b.DateAdded.Unix())
		}
		sb.WriteString(fmt.Sprintf("%s<DT><A HREF=\"%s\"%s>%s</A>\n",
			indent,
			html.EscapeString(b.URL),
			addDate,
			html.EscapeString(b.Title)))
	}
}
