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
	Removed int `json:"removed"` // files of pages and copied files that are gone
}

// listFile is where an export leaves its list for the next one.
const listFile = ".shugo-export.json"

// list tells the next export what this one wrote: from which state of which
// store its pages are, and the files it copied, each with the size and the
// time of change its original had.
type list struct {
	Store   string            `json:"store"`
	Version int64             `json:"version"`
	Copied  map[string]string `json:"copied"`
}

// Export writes the site into dir as files that any web server can give
// out: every page, the public files and the files the compiler made.
//
// An export follows the one before it, from the list that export left in
// dir. It copies only the files whose originals have changed since. From
// the same store it builds only the pages whose HTML has changed since, and
// removes the files of the pages that went away; so its work grows with the
// change, not with the site. From another store, or from a state the store
// no longer knows the changes after, it builds every page and removes the
// files of pages the site no longer has. A file that is already there with
// the same content is left alone.
//
// The folder belongs to the export: a file changed there by hand stays so
// until its page or original changes. Into a folder without a list the
// export writes every file and removes nothing.
func Export(pages *Pages, public, assets, dir string) (Exported, error) {
	var done Exported
	last, followed := readList(dir)
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
	// remove removes a file of the site that is gone, and the folders it
	// leaves empty.
	remove := func(name string) error {
		target := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.Remove(target); errors.Is(err, fs.ErrNotExist) {
			return nil
		} else if err != nil {
			return err
		}
		done.Removed++
		for folder := filepath.Dir(target); folder != dir && os.Remove(folder) == nil; folder = filepath.Dir(folder) {
		}
		return nil
	}

	// The copied files go first: where a page has the same address as a
	// copied file, the page is the one that stays.
	if err := copyFiles(map[string]string{"": public, strings.Trim(compile.AssetsURL, "/") + "/": assets}, last.Copied, next.Copied, write); err != nil {
		return done, err
	}
	for name := range last.Copied {
		if _, kept := next.Copied[name]; !kept {
			if err := remove(name); err != nil {
				return done, err
			}
		}
	}

	// All the pages come from one state of the store.
	err := pages.store.View(func(v *store.View) error {
		site, err := pages.program(v)
		if err != nil {
			return err
		}
		mark, err := v.Mark()
		if err != nil {
			return err
		}
		totals, err := v.Totals()
		if err != nil {
			return err
		}
		next.Store, next.Version, done.Pages = mark.Store, mark.Version, totals.Pages
		src := &source{view: v}
		var html []byte
		writePage := func(path string, data []byte) error {
			done.Built++
			if html, err = build(html[:0], site, data, src); err != nil {
				return fmt.Errorf("page %s: %w", path, err)
			}
			return write(pageFile(path), html)
		}

		if last.Store == mark.Store && mark.Origin <= last.Version && last.Version <= mark.Version {
			return v.Changes(last.Version, func(path string, data []byte) error {
				if data != nil {
					return writePage(path, data)
				}
				if _, copied := next.Copied[pageFile(path)]; copied {
					return nil
				}
				return remove(pageFile(path))
			})
		}
		kept := map[string]struct{}{}
		err = v.Pages(func(path string, data []byte) error {
			kept[pageFile(path)] = struct{}{}
			return writePage(path, data)
		})
		if err != nil || !followed {
			return err
		}
		return removeOtherPages(dir, kept, next.Copied, remove)
	})
	if err != nil {
		return done, err
	}
	done.Files = done.Pages + len(next.Copied)
	data, err := json.Marshal(next)
	if err != nil {
		return done, err
	}
	return done, os.WriteFile(filepath.Join(dir, listFile), data, 0o644)
}

// copyFiles copies the files of folders, each under its prefix, and notes
// them in copied. A file whose original has the same size and time of
// change as when it was copied last is not read.
func copyFiles(folders map[string]string, last, copied map[string]string, write func(name string, data []byte) error) error {
	for prefix, folder := range folders {
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
			copied[name] = stamp
			if last[name] == stamp {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return write(name, data)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// removeOtherPages removes from dir the files of pages that are neither
// pages nor copied files of the site now: what an export of another state
// left. Only the files an export writes for pages are looked at, and no
// folder whose name starts with a dot.
func removeOtherPages(dir string, pages map[string]struct{}, copied map[string]string, remove func(name string) error) error {
	var other []string
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != dir && strings.HasPrefix(entry.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		name := filepath.ToSlash(rel)
		if entry.Name() != "index.html" && name != "404.html" {
			return nil
		}
		_, page := pages[name]
		_, file := copied[name]
		if !page && !file {
			other = append(other, name)
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, name := range other {
		if err := remove(name); err != nil {
			return err
		}
	}
	return nil
}

// readList returns the list the last export left in dir, and whether there
// was one. A list that cannot be read counts as one from another store.
func readList(dir string) (list, bool) {
	data, err := os.ReadFile(filepath.Join(dir, listFile))
	if err != nil {
		return list{}, false
	}
	var last list
	if json.Unmarshal(data, &last) != nil {
		return list{}, true
	}
	return last, true
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
