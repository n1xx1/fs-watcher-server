package main

import (
	"github.com/labstack/echo/v4"
)

func handleGetAll(c echo.Context) error {
	type FileEntry struct {
		Frontmatter map[string]any `json:"frontmatter"`
		Title       string         `json:"title"`
		Level       *int           `json:"level,omitempty"`
		Type        string         `json:"type,omitempty"`
	}

	resp := map[string]FileEntry{}

	for k, v := range files.Copy() {
		// path, index := getRealFileName(k)
		resp[k] = FileEntry{
			Frontmatter: v.frontmatter,
			Title:       v.title,
			Level:       v.makeLevel(),
			Type:        v.typ,
		}
	}

	return c.JSON(200, resp)
}
