package main

import (
	"github.com/labstack/echo/v4"
	"strings"
)

func handleGetGet(c echo.Context) error {
	type FileEntry struct {
		Frontmatter map[string]any `json:"frontmatter"`
		Title       string         `json:"title"`
		Level       *int           `json:"level,omitempty"`
		Type        string         `json:"type,omitempty"`
		Contents    string         `json:"contents"`
	}

	k := c.QueryParam("k")
	k = strings.TrimPrefix(k, "/")

	v, ok := files.TryGet(k + ".md")
	if !ok {
		if k == "" {
			k += "_index.md"
		} else {
			k += "/_index.md"
		}

		v, ok = files.TryGet(k)
		if !ok {
			return echo.ErrNotFound
		}
	}
	resp := FileEntry{
		Frontmatter: v.frontmatter,
		Title:       v.title,
		Level:       v.makeLevel(),
		Type:        v.typ,
		Contents:    v.contents,
	}

	return c.JSON(200, resp)
}
