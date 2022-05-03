package main

import (
	"bytes"
	"fmt"
	"fs-watcher-server/utils"
	"github.com/adrg/frontmatter"
	"github.com/fsnotify/fsnotify"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type FsServer struct {
	Base string
	Port int

	done chan bool

	watcher     *fsnotify.Watcher
	loadedFiles *utils.RWMap[string, fsFileData]
}

type fsFileData struct {
	Path     string
	Rel      string
	Contents []byte
	Meta     any
}

func NewFsServer(dir string, port int) *FsServer {
	return &FsServer{
		Base: dir,
		Port: port,

		done:        make(chan bool),
		loadedFiles: utils.NewRWMap[string, fsFileData](),
	}
}

func (s *FsServer) updateFiles(files []string) {
	for _, f := range files {
		log.Infof("[CHANGE] %s", f)
		s.updateFile(f)
	}
}

func (s *FsServer) updateFile(fileName string) {
	rel, _ := filepath.Rel(s.Base, fileName)
	file := filepath.ToSlash(rel)

	stat, err := os.Stat(fileName)
	if stat.IsDir() {
		return
	}

	data, err := os.ReadFile(fileName)
	if err != nil {
		return
	}

	entry := fsFileData{
		Path:     fileName,
		Rel:      file,
		Contents: data,
	}
	if strings.HasSuffix(fileName, ".md") || strings.HasSuffix(fileName, ".mdx") {
		var matter map[any]any
		_, err := frontmatter.Parse(bytes.NewReader(data), &matter)
		if err != nil {
			log.Warnf("frontmatter: %v", err)
		} else {
			entry.Meta = utils.YamlToJson(matter)
		}
	}

	s.loadedFiles.Set(file, entry)
}

func (s *FsServer) startWatcher() {
	ticker := time.NewTicker(100 * time.Millisecond)
	var updateQueue []string

	for {
		select {
		case <-ticker.C:
			go s.updateFiles(updateQueue)
			updateQueue = nil

		case e := <-s.watcher.Events:
			stat, err := os.Stat(e.Name)
			if err == nil && stat.IsDir() {
				if e.Op&fsnotify.Create != 0 {
					err = s.watchRecursive(e.Name, false)
					if err != nil {
						log.Warnf("watch recursive: %v", err)
					}
				}
			}
			if e.Op&fsnotify.Remove != 0 {
				_ = s.watcher.Remove(e.Name)
				if _, ok := s.loadedFiles.TryGet(e.Name); ok {
					s.loadedFiles.Delete(e.Name)
				}
			}
			if err == nil && !stat.IsDir() && e.Op&fsnotify.Write != 0 {
				updateQueue = append(updateQueue, e.Name)
			}

		case e := <-s.watcher.Errors:
			log.Warnf("watcher: %v", e)

		case <-s.done:
			_ = s.watcher.Close()
			ticker.Stop()
			return
		}
	}
}

func (s *FsServer) isIgnoredDir(walkPath string, d os.DirEntry) bool {
	n := d.Name()
	if strings.HasSuffix(n, "~") || strings.HasPrefix(n, ".") {
		return true
	}
	return false
}

func (s *FsServer) watchRecursive(path string, remove bool) error {
	err := filepath.WalkDir(path, func(walkPath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if remove {
				if err = s.watcher.Remove(walkPath); err != nil {
					return err
				}
			} else if !s.isIgnoredDir(walkPath, d) {
				if err = s.watcher.Add(walkPath); err != nil {
					return err
				}
			} else {
				return filepath.SkipDir
			}
		}
		return nil
	})
	return err
}

func (s *FsServer) Start() (err error) {
	s.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("watcher: %w", err)
	}

	go s.startWatcher()

	e := echo.New()
	e.Match([]string{"GET", "POST"}, "/readFile", s.handleReadFile)
	e.Match([]string{"GET", "POST"}, "/readdir", s.handleReadDir)

	return e.Start(fmt.Sprintf(":%d", s.Port))
}

func (s *FsServer) handleReadDir(c echo.Context) (err error) {
	var data struct {
		Dir         string `query:"d" json:"dir"`
		IncludeMeta bool   `query:"m" json:"include_meta"`
	}

	err = c.Bind(&data)
	if err != nil {
		return
	}

	if data.Dir == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing dir")
	}

	data.Dir = strings.TrimLeft(data.Dir, "/")
	data.Dir = strings.TrimRight(data.Dir, "/")

	dirRegex, err := regexp.Compile(`^` + regexp.QuoteMeta(data.Dir) + `/([^/]+)(.*?)$`)
	if err != nil {
		return err
	}

	if data.IncludeMeta {
		type respEntry struct {
			Dir      bool   `json:"dir"`
			Path     string `json:"path"`
			Contents string `json:"contents"`
			Meta     any    `json:"meta,omitempty"`
		}

		set := map[string]*respEntry{}
		for k, v := range s.loadedFiles.Copy() {
			if match := dirRegex.FindStringSubmatch(k); match != nil {
				if _, ok := set[match[1]]; !ok {
					dir := strings.HasPrefix(match[2], "/")
					entry := &respEntry{
						Dir:  dir,
						Path: match[1],
					}
					if !dir {
						entry.Contents = string(v.Contents)
						entry.Meta = v.Meta
					}
					set[match[1]] = entry
				}
			}
		}

		return c.JSON(200, set)
	}
	set := utils.NewSet[string]()
	for k := range s.loadedFiles.Copy() {
		if match := dirRegex.FindStringSubmatch(k); match != nil {
			set.Add(match[1])
		}
	}

	return c.JSON(200, set)
}

func (s *FsServer) handleReadFile(c echo.Context) (err error) {
	var data struct {
		File  string   `query:"f" json:"file"`
		Files []string `json:"files"`
	}

	err = c.Bind(&data)
	if err != nil {
		return
	}

	if data.File != "" {
		data.Files = append(data.Files, data.File)
	}

	if len(data.Files) == 0 {
		return fmt.Errorf("no files requested")
	}

	type respEntry struct {
		Path     string `json:"path"`
		Contents string `json:"contents"`
		Meta     any    `json:"meta,omitempty"`
	}

	resp := make([]respEntry, 0, len(data.Files))
	for _, f := range data.Files {
		if file, ok := s.loadedFiles.TryGet(f); ok {
			resp = append(resp, respEntry{
				Path:     file.Rel,
				Contents: string(file.Contents),
				Meta:     file.Meta,
			})
		}
	}

	return c.JSON(200, resp)
}
