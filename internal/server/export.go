package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/grimalschi/shugo/internal/compile"
)

// Exported tells what Export did.
type Exported struct {
	Pages   int `json:"pages"`   // pages of the site
	Files   int `json:"files"`   // files of the site, pages among them
	Written int `json:"written"` // files that were new or had changed
	Removed int `json:"removed"` // files of the last export that are gone
}

// The list of files of the last export, kept in the folder of the export
// so that the next one can remove what is gone.
const manifest = ".shugo-export.json"

// Export writes the whole site into dir as files that any web server can
// give out: every page, the public files and the files the compiler made.
// A file that is already there with the same content is left alone.
func Export(pages *Pages, public, assets, dir string) (Exported, error) {
	var done Exported
	var files []string
	write := func(name string, data []byte) error {
		files = append(files, name)
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

	err := pages.Each(func(address string, html []byte) error {
		done.Pages++
		return write(pageFile(address), html)
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
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(folder, path)
			return write(prefix+filepath.ToSlash(rel), data)
		})
		if err != nil {
			return done, err
		}
	}

	// What the last export wrote and this one did not is not a part of the
	// site any more.
	var last []string
	if data, err := os.ReadFile(filepath.Join(dir, manifest)); err == nil {
		json.Unmarshal(data, &last)
	}
	sort.Strings(files)
	for _, name := range last {
		if i := sort.SearchStrings(files, name); i < len(files) && files[i] == name {
			continue
		}
		if os.Remove(filepath.Join(dir, filepath.FromSlash(name))) == nil {
			done.Removed++
		}
	}
	done.Files = len(files)
	data, _ := json.Marshal(files)
	return done, os.WriteFile(filepath.Join(dir, manifest), data, 0o644)
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
