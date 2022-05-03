package main

import (
	"bytes"
	"fs-watcher-server/utils"
	"fs-watcher-server/watcher"
	"github.com/adrg/frontmatter"
	"github.com/fsnotify/fsnotify"
	"github.com/labstack/echo/v4"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type FileData struct {
	path        string
	frontmatter map[string]any
	title       string
	hasLevel    bool
	level       int
	typ         string
	contents    string
}

func (v *FileData) makeLevel() *int {
	var level *int
	if v.hasLevel {
		level = new(int)
		*level = v.level
	}
	return level
}

const baseDir = "C:\\Users\\n1xx1\\Development\\n1xx1\\pfitdb\\pfitdb"

var files = utils.NewRWMap[string, FileData]()
var titleRegexp = regexp.MustCompile(`(?:^|\n)#+ (.+?)(?: - (.+?)(?: (\d+))?)?(?:$|\n)`)

func readFile(fileName string) {
	rel, _ := filepath.Rel(baseDir, fileName)
	file := filepath.ToSlash(rel)

	s, err := os.Stat(fileName)
	if s.IsDir() {
		return
	}

	data, err := os.ReadFile(fileName)
	if err != nil {
		return
	}

	var matter map[any]any
	rest, err := frontmatter.Parse(bytes.NewReader(data), &matter)

	var title string
	var typ string
	var level int

	if match := titleRegexp.FindStringSubmatch(string(rest)); match != nil {
		title = match[1]
		typ = match[2]
		if match[3] != "" {
			if x, err := strconv.ParseInt(match[3], 10, 32); err == nil {
				level = int(x)
			}
		}
	}

	files.Set(file, FileData{
		path:        fileName,
		frontmatter: utils.YamlToJson(matter),
		title:       title,
		typ:         typ,
		level:       level,
		contents:    string(data),
	})

	// log.Printf("%#v", files.Get(file))
}

func main() {
	ignoredDirs := func(walkPath string, d os.DirEntry) bool {
		return !strings.HasPrefix(d.Name(), ".")
	}
	w, err := watcher.NewWatcherWithFilter(ignoredDirs)

	if err != nil {
		log.Fatalf("watcher: %v", err)
	}

	go func() {
		batcher := time.NewTicker(100 * time.Millisecond)
		var batches []fsnotify.Event

		for {
			select {
			case <-batcher.C:
				for _, e := range batches {
					switch {
					case e.Op&fsnotify.Write != 0:
						log.Printf("%s [%v]", e.Name, e.Op)
						go readFile(e.Name)
					}
				}
				batches = nil
			case e := <-w.Events:
				for _, b := range batches {
					if b.Name == e.Name && b.Op == e.Op {
						goto dontadd
					}
				}
				batches = append(batches, e)
			dontadd:
				;
			case e := <-w.Errors:
				log.Printf("error: %v", e)
			}
		}
	}()

	err = filepath.WalkDir(baseDir, func(walkPath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && !ignoredDirs(walkPath, d) {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".md") {
			readFile(walkPath)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("walk: %v", err)
	}

	log.Print("Added all files!")

	err = w.AddRecursive(baseDir)
	if err != nil {
		log.Fatalf("watcher: %v", err)
	}

	e := echo.New()

	e.GET("/dirs", handleGetDirs)
	e.GET("/get", handleGetGet)
	e.GET("/all", handleGetAll)
	e.GET("/tree", handleGetTree)

	log.Print("Listening on http://127.0.0.1:8090")
	e.Logger.Fatal(e.Start(":8090"))
}

func getRealFileName(f string) (name string, index bool) {
	name = "/" + f
	if name1 := strings.TrimSuffix(name, ".md"); len(name) != len(name1) {
		name = name1
	}
	if name1 := strings.TrimSuffix(name, "/_index"); len(name) != len(name1) {
		name = name1
		if name == "" {
			name = "/"
		}
		index = true
	}
	return
}
