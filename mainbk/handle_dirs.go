package main

import (
	"github.com/labstack/echo/v4"
	"regexp"
	"strings"
)

var matchSameDir = regexp.MustCompile(`^[^/]+(/_index.md)?$`)

func handleGetDirs(c echo.Context) error {
	slug := c.QueryParam("k")
	onlyDirs := c.QueryParam("d") == "1"

	if !strings.HasSuffix(slug, "/") {
		slug += "/"
	}

	type FileEntry struct {
		Path        string         `json:"path"`
		Frontmatter map[string]any `json:"frontmatter"`
		Title       string         `json:"title"`
		Level       *int           `json:"level,omitempty"`
		Type        string         `json:"type,omitempty"`
	}

	resp := make([]FileEntry, 0)

	for k, v := range files.Copy() {
		if sub := strings.TrimPrefix("/"+k, slug); len(sub) != len(k)+1 {
			if !matchSameDir.MatchString(sub) {
				continue
			}
			trimIndex := strings.TrimSuffix(sub, "/_index.md")
			index := len(trimIndex) != len(sub)
			if onlyDirs && !index {
				continue
			}
			if index {
				resp = append(resp, FileEntry{
					Path:        trimIndex + "/",
					Frontmatter: v.frontmatter,
					Title:       v.title,
					Level:       v.makeLevel(),
					Type:        v.typ,
				})
				continue
			}
			if trimMd := strings.TrimSuffix(sub, ".md"); len(trimMd) != len(sub) {
				if trimMd == "_index" {
					continue
				}
				resp = append(resp, FileEntry{
					Path:        trimMd,
					Frontmatter: v.frontmatter,
					Title:       v.title,
					Level:       v.makeLevel(),
					Type:        v.typ,
				})
			}
		}
	}
	return c.JSON(200, resp)
}
