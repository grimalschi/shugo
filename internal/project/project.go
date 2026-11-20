// Package project finds the files of a Shugo project and reads its settings.
package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// The folders and files every project has by the same names.
const (
	Settings = "shugo.json" // the settings of the project
	Pages    = "pages"      // the pages and their layouts
	Public   = "public"     // files that go to the site as they are
	Output   = "_build"     // everything the framework writes
)

// Config is what shugo.json holds. Every field may be left out.
type Config struct {
	// Site is the address the site is published at: "https://example.com".
	Site string `json:"site"`
	// Runtime runs the JavaScript of the project: "bun" or "node".
	Runtime string `json:"runtime"`
	// Redirects sends one path to another: {"/old": "/new"}.
	Redirects map[string]string `json:"redirects"`
}

// Project is a folder with a site in it.
type Project struct {
	Root   string // an absolute path
	Config Config
}

// Open reads the settings of the project in dir.
func Open(dir string) (*Project, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if info, err := os.Stat(filepath.Join(root, Pages)); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("%s is not a Shugo project: it has no %s folder", root, Pages)
	}
	p := &Project{Root: root}
	data, err := os.ReadFile(filepath.Join(root, Settings))
	if errors.Is(err, fs.ErrNotExist) {
		return p, nil
	} else if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&p.Config); err != nil {
		return nil, fmt.Errorf("%s: %w", Settings, err)
	}
	return p, nil
}

// Path returns the absolute path of a file of the project.
func (p *Project) Path(parts ...string) string {
	return filepath.Join(append([]string{p.Root}, parts...)...)
}

// Templates returns the .shugo files of the project, sorted. The paths are
// relative to the root and use "/".
func (p *Project) Templates() ([]string, error) {
	var out []string
	err := filepath.WalkDir(p.Root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := entry.Name()
		if entry.IsDir() {
			// Not sources: what the framework wrote, what is published as it
			// is, installed packages and hidden folders.
			if path != p.Root && (name == Output || name == Public || name == "node_modules" || strings.HasPrefix(name, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(name, ".shugo") {
			rel, _ := filepath.Rel(p.Root, path)
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(out)
	return out, err
}

// Runtime returns the program that runs the JavaScript of the project: the
// one its settings name, or Bun if it is installed, or Node.
func (p *Project) Runtime() string {
	if p.Config.Runtime != "" {
		return p.Config.Runtime
	}
	if _, err := exec.LookPath("bun"); err == nil {
		return "bun"
	}
	return "node"
}
