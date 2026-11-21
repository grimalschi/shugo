package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/grimalschi/shugo/internal/compile"
	"github.com/grimalschi/shugo/internal/store"
)

// Exported tells what Export did.
type Exported struct {
	Pages   int `json:"pages"`   // pages of the site
	Built   int `json:"built"`   // pages built: those whose HTML may differ from the last export
	Files   int `json:"files"`   // files of the site, pages among them
	Written int `json:"written"` // files that were new or had changed
	Removed int `json:"removed"` // files of the last export that are gone
}

// listFile is where an export leaves its list for the next one.
const listFile = ".shugo-export.json"

// list tells the next export what this one wrote: from which state of which
// store its pages are, the files of the pages, and the files it copied, each
// with the size and the time of change its original had.
type list struct {
	store.Mark
	Pages  []string          `json:"pages"`
	Copied map[string]string `json:"copied"`
}

// Export writes the site into dir as files that any web server can give
// out: every page, the public files and the files the compiler made.
//
// It builds only the pages whose HTML may have changed since the export
// before, and copies only the files whose originals have: it learns what that
// export wrote from the list it left in dir. A file that is already there
// with the same content is left alone. The folder belongs to the export: a
// file changed there by hand stays so until its page or original changes.
// Without the list, every page is built and every file copied.
func Export(pages *Pages, public, assets, dir string) (Exported, error) {
	var done Exported
	last := readList(dir)
	next := list{Copied: map[string]string{}}
	write := func(name string, data []byte) error {
		target := filepath.Join(dir, filepath.FromSlash(name))
		if old, err := os.ReadFile(target); err == nil && bytes.Equal(old, data) {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		done.Written++
		return os.WriteFile(target, data, 0o644)
	}

	// All the pages come from one state of the store.
	err := pages.store.View(func(v *store.View) error {
		site, err := pages.program(v)
		if err != nil {
			return err
		}
		if next.Mark, err = v.Mark(); err != nil {
			return err
		}
		// The versions of another store, or of a state this store has not
		// come to, say nothing about the pages written.
		since := last.Version
		if last.Store != next.Store || last.Version > next.Version {
			since = 0
		}
		src := &source{view: v}
		var html []byte
		return v.Pages(since, func(path string, data []byte) error {
			name := pageFile(path)
			next.Pages = append(next.Pages, name)
			if data == nil {
				return nil
			}
			done.Built++
			if html, err = build(html[:0], site, data, src); err != nil {
				return fmt.Errorf("page %s: %w", path, err)
			}
			return write(name, html)
		})
	})
	if err != nil {
		return done, err
	}

	for prefix, folder := range map[string]string{"": public, strings.Trim(compile.AssetsURL, "/") + "/": assets} {
		err := filepath.WalkDir(folder, func(path string, entry fs.DirEntry, err error) error {
			if errors.Is(err, fs.ErrNotExist) {
				return fs.SkipAll
			}
			if err != nil || entry.IsDir() {
				return err
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(folder, path)
			name := prefix + filepath.ToSlash(rel)
			stamp := fmt.Sprintf("%d %d", info.Size(), info.ModTime().UnixNano())
			next.Copied[name] = stamp
			if last.Copied[name] == stamp {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return write(name, data)
		})
		if err != nil {
			return done, err
		}
	}

	// What the last export wrote and this one did not is not a part of the
	// site any more.
	now := make(map[string]struct{}, len(next.Pages)+len(next.Copied))
	for _, name := range next.Pages {
		now[name] = struct{}{}
	}
	for name := range next.Copied {
		now[name] = struct{}{}
	}
	gone := func(name string) {
		if _, kept := now[name]; !kept && os.Remove(filepath.Join(dir, filepath.FromSlash(name))) == nil {
			done.Removed++
		}
	}
	for _, name := range last.Pages {
		gone(name)
	}
	for name := range last.Copied {
		gone(name)
	}
	done.Pages, done.Files = len(next.Pages), len(now)
	data, err := json.Marshal(next)
	if err != nil {
		return done, err
	}
	return done, os.WriteFile(filepath.Join(dir, listFile), data, 0o644)
}

// readList returns the list the last export left in dir. A folder without
// one, or with one that cannot be read, is written as if for the first time.
func readList(dir string) list {
	var last list
	if data, err := os.ReadFile(filepath.Join(dir, listFile)); err == nil {
		if json.Unmarshal(data, &last) != nil {
			return list{}
		}
	}
	return last
}

// pageFile returns the file a page is written to: a folder with index.html
// for every address, which every web server understands. The page for
// missing addresses is 404.html, where hosts look for it.
func pageFile(address string) string {
	switch address {
	case "/":
		return "index.html"
	case notFound:
		return "404.html"
	}
	return strings.TrimPrefix(address, "/") + "/index.html"
}
