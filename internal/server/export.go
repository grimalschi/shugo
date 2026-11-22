package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/grimalschi/shugo/internal/compile"
	"github.com/grimalschi/shugo/internal/skeleton"
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
// store its pages are, the files it copied, each with the size and the time
// of change its original had, and the files that modules among the pages
// gave. An HTML file is known for a page by its name; those are not, so an
// export that builds every page knows from the list which of them to remove.
type list struct {
	Store   string            `json:"store"`
	Version int64             `json:"version"`
	Copied  map[string]string `json:"copied"`
	Files   []string          `json:"files,omitempty"`
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

	// Pages and the list of their assets come from one state of the store.
	err := pages.store.View(func(v *store.View) error {
		site, err := pages.skeleton(v)
		if err != nil {
			return err
		}
		// The copied files go first: where a page has the same address as a
		// copied file, the page is the one that stays.
		if err := copyFiles(public, "", nil, last.Copied, next.Copied, write); err != nil {
			return err
		}
		prefix := strings.Trim(compile.AssetsURL, "/") + "/"
		if err := copyFiles(assets, prefix, site.Assets, last.Copied, next.Copied, write); err != nil {
			return err
		}
		for name := range last.Copied {
			if _, kept := next.Copied[name]; !kept {
				if err := remove(name); err != nil {
					return err
				}
			}
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
		files := map[string]struct{}{} // the files of modules the folder has
		var body []byte
		writePage := func(path string, route int, data []byte) error {
			done.Built++
			if body, err = build(body[:0], site, route, data, src); err != nil {
				return fmt.Errorf("page %s: %w", path, err)
			}
			name := pageFile(site, route, path)
			if site.FileRoute(route) {
				files[name] = struct{}{}
			}
			return write(name, body)
		}
		defer func() { next.Files = slices.Sorted(maps.Keys(files)) }()

		if last.Store == mark.Store && mark.Origin <= last.Version && last.Version <= mark.Version {
			for _, name := range last.Files {
				files[name] = struct{}{}
			}
			return v.Changes(last.Version, func(path string, route int, data []byte) error {
				if data != nil {
					return writePage(path, route, data)
				}
				name := pageFile(site, route, path)
				delete(files, name)
				if _, copied := next.Copied[name]; copied {
					return nil
				}
				return remove(name)
			})
		}
		kept := map[string]struct{}{}
		err = v.Pages(func(path string, route int, data []byte) error {
			kept[pageFile(site, route, path)] = struct{}{}
			return writePage(path, route, data)
		})
		if err != nil || !followed {
			return err
		}
		for _, name := range last.Files {
			_, page := kept[name]
			_, copied := next.Copied[name]
			if !page && !copied {
				if err := remove(name); err != nil {
					return err
				}
			}
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

// copyFiles uses the manifest when supplied, or all files in the folder.
// A file with the same size and time of change is not read again.
func copyFiles(folder, prefix string, names []string, last, copied map[string]string, write func(name string, data []byte) error) error {
	copyFile := func(path, name string, info fs.FileInfo) error {
		name = prefix + name
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
	}
	if names != nil {
		for _, name := range names {
			if !filepath.IsLocal(name) || name != filepath.Base(name) || name == "." {
				return fmt.Errorf("invalid compiled asset name %q", name)
			}
			path := filepath.Join(folder, name)
			info, err := os.Stat(path)
			if err != nil {
				return err
			}
			if err := copyFile(path, name, info); err != nil {
				return err
			}
		}
		return nil
	}
	return filepath.WalkDir(folder, func(path string, entry fs.DirEntry, err error) error {
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
		return copyFile(path, filepath.ToSlash(rel), info)
	})
}

// removeOtherPages removes from dir the files of pages that are neither
// pages nor copied files of the site now: what an export of another state
// left. Only HTML files are looked at, the kind an export writes for pages,
// and no folder whose name starts with a dot.
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
		if !strings.HasSuffix(name, ".html") {
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

// pageFile returns the file a page of a route is written to. A page of
// templates is named the way SvelteKit names the pages it prerenders:
// /about is about.html, and static hosts serve /about from it; / is
// index.html. The page for missing addresses is 404.html, where hosts look
// for it. The file a module gives is named by its address: /sitemap.xml is
// sitemap.xml.
func pageFile(site *skeleton.Site, route int, address string) string {
	if site.FileRoute(route) {
		return strings.TrimPrefix(address, "/")
	}
	switch address {
	case "/":
		return "index.html"
	case notFound:
		return "404.html"
	}
	return strings.TrimPrefix(address, "/") + ".html"
}
