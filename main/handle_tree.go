package main

import (
	"github.com/labstack/echo/v4"
	"strings"
)

func handleGetTree(c echo.Context) error {
	type FileEntry struct {
		Frontmatter map[string]any        `json:"frontmatter"`
		Title       string                `json:"title"`
		Level       *int                  `json:"level,omitempty"`
		Type        string                `json:"type,omitempty"`
		Path        string                `json:"path"`
		Children    map[string]*FileEntry `json:"children,omitempty"`
	}

	var resp FileEntry

	for k, v := range files.Copy() {
		path, _ := getRealFileName(k)
		parts := strings.Split(path, "/")

		var entry *FileEntry
		for i, p := range parts {
			if entry == nil {
				panic("what")
			}
			if p == "" {
				entry = &resp
			} else {
				if entry.Children == nil {
					entry.Children = map[string]*FileEntry{}
				}
				newEntry := entry.Children[p]
				if entry == nil {
					newEntry = &FileEntry{
						Path: strings.Join(parts[:i+1], "/"),
					}
					entry.Children[p] = newEntry
				}
				entry = newEntry
			}
		}

		entry.Path = path
		entry.Title = v.title
		entry.Type = v.typ
		entry.Frontmatter = v.frontmatter
		entry.Level = v.makeLevel()
	}

	return c.JSON(200, resp)
}
