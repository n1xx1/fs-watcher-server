package main

import (
	"bytes"
	"dev-helper/utils"
	"dev-helper/watcher"
	"encoding/json"
	"fmt"
	"github.com/adrg/frontmatter"
	"github.com/fsnotify/fsnotify"
	"log"
	"net/http"
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
	name        string
	hasLevel    bool
	level       int
	typ         string
	contents    string
}

const baseDir = "C:\\Users\\n1xx1\\Development\\pfitdb-hugo\\content"

var files = utils.NewRWMap[string, FileData]()
var titleRegexp = regexp.MustCompile(`(?:^|\n)#+ (.+?)(?: - (.+?)(?: (\d+))?)?(?:$|\n)`)

func readFile(fileName string) {
	rel, _ := filepath.Rel(baseDir, fileName)
	file := filepath.ToSlash(rel)

	data, err := os.ReadFile(fileName)
	if err != nil {
		log.Fatal(err)
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
		name:        title,
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
		batcher := time.NewTimer(100 * time.Millisecond)
		batches := map[string][]fsnotify.Event{}

		for {
			select {
			case <-batcher.C:
				for _, es := range batches {
					for _, e := range es {
						switch e.Op {
						case watcher.Write:
							go readFile(e.Name)
						}
					}
				}
				batches = map[string][]fsnotify.Event{}
			case e := <-w.Events:
				batches[e.Name] = append(batches[e.Name], e)
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

	http.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		type FileEntry struct {
			Frontmatter map[string]any `json:"frontmatter"`
			Name        string         `json:"name"`
			Level       *int           `json:"level,omitempty"`
			Type        string         `json:"type,omitempty"`
			Contents    string         `json:"contents"`
		}

		k := r.URL.Query().Get("k")
		v, ok := files.TryGet(k)
		if !ok {
			w.WriteHeader(404)
			_, _ = w.Write([]byte("Not found!"))
			return
		}

		var level *int
		if v.hasLevel {
			level = new(int)
			*level = v.level
		}
		resp := FileEntry{
			Frontmatter: v.frontmatter,
			Name:        v.name,
			Level:       level,
			Type:        v.typ,
			Contents:    v.contents,
		}

		respBytes, err := json.Marshal(&resp)
		if err != nil {
			w.WriteHeader(500)
			_, _ = fmt.Fprintf(w, "%v", err)
			return
		}

		w.Header().Set("content-type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(respBytes)
	})

	http.HandleFunc("/all", func(w http.ResponseWriter, r *http.Request) {
		type FileEntry struct {
			Frontmatter map[string]any `json:"frontmatter"`
			Name        string         `json:"name"`
			Level       *int           `json:"level,omitempty"`
			Type        string         `json:"type,omitempty"`
		}
		resp := map[string]FileEntry{}

		for k, v := range files.Copy() {
			var level *int
			if v.hasLevel {
				level = new(int)
				*level = v.level
			}
			resp[k] = FileEntry{
				Frontmatter: v.frontmatter,
				Name:        v.name,
				Level:       level,
				Type:        v.typ,
			}
		}

		respBytes, err := json.Marshal(&resp)
		if err != nil {
			w.WriteHeader(500)
			_, _ = fmt.Fprintf(w, "%v", err)
			return
		}

		w.Header().Set("content-type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(respBytes)
	})

	log.Print("Listening on http://127.0.0.1:8090")
	_ = http.ListenAndServe(":8090", nil)
}
