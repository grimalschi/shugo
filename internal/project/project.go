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
	Source   = "src"        // the code of the site: pages, components, modules, styles
	Pages    = "src/pages"  // the pages and their layouts
	Public   = "public"     // files that go to the site as they are
	Output   = "_build"     // everything the framework writes
)

const (
	TemplateExtension = ".svelte"
	IslandSuffix      = ".island.svelte"
)

// IsIsland reports whether a source file declares a Svelte island.
func IsIsland(path string) bool {
	return strings.HasSuffix(path, IslandSuffix)
}

// Config is what shugo.json holds. Every field may be left out.
type Config struct {
	// Site is the address the site is published at: "https://example.com".
	Site string `json:"site"`
	// Runtime runs the JavaScript of the project: "bun" or "node".
	Runtime string `json:"runtime"`
	// Redirects sends one path to another: {"/old": "/new"}.
	Redirects map[string]string `json:"redirects"`
	// Data is the SQLite database the pages read through Shugo.query:
	// "data/site.db". An update computes only what a change in it reaches.
	Data string `json:"data"`
	// CatalogIslands is accepted for projects built with the previous experiment.
	// Search usage in the evaluation bundle now selects the projection.
	CatalogIslands bool `json:"catalogIslands,omitempty"`
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
	if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(Pages))); err != nil || !info.IsDir() {
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

// Data returns the path of the data database of the project, or nothing
// if it has none.
func (p *Project) Data() string {
	if p.Config.Data == "" {
		return ""
	}
	return p.Path(filepath.FromSlash(p.Config.Data))
}

// Templates returns the sorted regular .svelte files in src/. Islands enter
// the compiler through imports. The paths are relative to the root and use "/".
func (p *Project) Templates() ([]string, error) {
	var out []string
	source := p.Path(Source)
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := entry.Name()
		if entry.IsDir() {
			// Not sources: installed packages and hidden folders.
			if path != source && (name == "node_modules" || strings.HasPrefix(name, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		extension := filepath.Ext(name)
		const legacyExtension = ".shugo"
		if IsIsland(name) || (extension != TemplateExtension && extension != legacyExtension) {
			return nil
		}
		rel, err := filepath.Rel(p.Root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if extension == legacyExtension {
			renamed := strings.TrimSuffix(rel, legacyExtension) + TemplateExtension
			return fmt.Errorf("%s: rename this template to %s and update its imports", rel, renamed)
		}
		out = append(out, rel)
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
