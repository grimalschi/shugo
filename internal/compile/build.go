package compile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/grimalschi/shugo/internal/program"
	"github.com/grimalschi/shugo/internal/project"
	"github.com/grimalschi/shugo/internal/route"
)

// The files a compiled project consists of, under the output folder.
const (
	LoadBundle   = "load.mjs"     // the scripts: they load the data, and lead the computation of pages
	RenderBundle = "render.cjs"   // the templates: they compute values from what they are given
	Site         = "site.json"    // the program of the site: templates and routes
	Assets       = "assets"       // style sheets, published under AssetsURL
	Sources      = "sources.json" // what the project was compiled from
)

// AssetsURL is where the files of the Assets folder are published.
const AssetsURL = "/_assets/"

// Result is a compiled project.
type Result struct {
	Site *program.Site
	// ID changes whenever the compiled project does.
	ID       string
	Warnings []string
}

// Build compiles the project and writes the result under its output folder.
func Build(p *project.Project) (*Result, error) {
	paths, err := p.Templates()
	if err != nil {
		return nil, err
	}
	b := &builder{project: p, paths: paths, ids: map[string]int{}}
	for id, path := range paths {
		b.ids[path] = id
	}
	if err := b.read(); err != nil {
		return nil, err
	}
	routes, err := b.routes()
	if err != nil {
		return nil, err
	}
	// The scripts are bundled first: what they import tells which files
	// load data, and that decides how every template is written.
	loaded, err := b.bundle(loadSide, routes)
	if err != nil {
		return nil, err
	}
	b.sort(loaded.meta)
	for _, u := range b.units {
		u.renderModule()
	}
	rendered, err := b.bundle(renderSide, routes)
	if err != nil {
		return nil, err
	}
	if err := b.sealed(rendered.meta); err != nil {
		return nil, err
	}
	shared, styles, err := b.graph(rendered.meta)
	if err != nil {
		return nil, err
	}

	site := &program.Site{Origin: p.Config.Site, Redirects: p.Config.Redirects, Routes: routes}
	if site.Origin == "" {
		site.Origin = "http://localhost"
	}
	for i, path := range paths {
		u := b.units[i]
		gives := make([]string, 0, len(u.provided))
		for name := range u.provided {
			gives = append(gives, name)
		}
		sort.Strings(gives)
		site.Files = append(site.Files, program.File{Path: path, Parts: u.parts, Slots: u.outlets, Shared: shared[path], Loads: u.loads, Why: u.why, Gives: gives})
	}

	output := p.Path(project.Output)
	assets := filepath.Join(output, Assets)
	if err := os.RemoveAll(assets); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(assets, 0o755); err != nil {
		return nil, err
	}
	sheet, err := b.styles(styles)
	if err != nil {
		return nil, err
	}
	if sheet != nil {
		name := "styles." + digest(sheet)[:10] + ".css"
		if err := os.WriteFile(filepath.Join(assets, name), sheet, 0o644); err != nil {
			return nil, err
		}
		site.Mounts = map[string]string{"head": `<link rel="stylesheet" href="` + AssetsURL + name + `">`}
	}

	data, err := site.Encode()
	if err != nil {
		return nil, err
	}
	sources, err := json.Marshal(b.sources(append(loaded.meta.files(), rendered.meta.files()...)))
	if err != nil {
		return nil, err
	}
	for name, content := range map[string][]byte{
		LoadBundle: loaded.code, LoadBundle + ".map": loaded.smap,
		RenderBundle: rendered.code, RenderBundle + ".map": rendered.smap,
		Site: data, Sources: sources,
	} {
		if err := os.WriteFile(filepath.Join(output, name), content, 0o644); err != nil {
			return nil, err
		}
	}
	return &Result{Site: site, ID: digest(data, loaded.code, rendered.code), Warnings: append(loaded.warnings, rendered.warnings...)}, nil
}

// Load reads a project that Build has compiled before.
func Load(p *project.Project) (*Result, error) {
	data, err := os.ReadFile(p.Path(project.Output, Site))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, errors.New("the project is not compiled yet: run `shugo build` first")
	} else if err != nil {
		return nil, err
	}
	load, err := os.ReadFile(p.Path(project.Output, LoadBundle))
	if err != nil {
		return nil, err
	}
	render, err := os.ReadFile(p.Path(project.Output, RenderBundle))
	if err != nil {
		return nil, err
	}
	site, err := program.Decode(data)
	if err != nil {
		return nil, err
	}
	return &Result{Site: site, ID: digest(data, load, render)}, nil
}

// sources is the list of what a project was compiled from: its templates,
// and every file that went into the bundle with a hash of its content.
type sources struct {
	Templates []string          `json:"templates"`
	Files     map[string]string `json:"files"`
}

// sources hashes the files of the project that the compiled result depends on.
func (b *builder) sources(bundled []string) sources {
	out := sources{Templates: b.paths, Files: map[string]string{}}
	for _, name := range append(bundled, project.Settings) {
		out.Files[name] = hashFile(b.project.Path(filepath.FromSlash(name)))
	}
	return out
}

// hashFile returns a hash of the content of a file, or nothing if it cannot
// be read: a file that is not there is a state of its own.
func hashFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return digest(data)
}

// Changed tells whether the sources of the project are not what it was
// compiled from: a template was added or removed, or a file that went into
// the bundle has other content. Then the compiled project is out of date, and
// computing it again would show the old code.
func Changed(p *project.Project) (bool, error) {
	data, err := os.ReadFile(p.Path(project.Output, Sources))
	if errors.Is(err, fs.ErrNotExist) {
		return true, nil
	} else if err != nil {
		return false, err
	}
	var compiled sources
	if err := json.Unmarshal(data, &compiled); err != nil {
		return true, nil
	}
	templates, err := p.Templates()
	if err != nil {
		return false, err
	}
	if !slices.Equal(templates, compiled.Templates) {
		return true, nil
	}
	for name, hash := range compiled.Files {
		if hashFile(p.Path(filepath.FromSlash(name))) != hash {
			return true, nil
		}
	}
	return false, nil
}

func digest(parts ...[]byte) string {
	sum := sha256.New()
	for _, part := range parts {
		sum.Write(part)
	}
	return hex.EncodeToString(sum.Sum(nil))[:32]
}

type builder struct {
	project *project.Project
	paths   []string       // the .shugo files, sorted
	ids     map[string]int // the number of every file: its place in paths
	units   []*unit
}

// roleOf tells what a file is by where it lies.
func roleOf(path string) role {
	rel, inPages := strings.CutPrefix(path, project.Pages+"/")
	switch {
	case !inPages:
		return component
	case pathBase(rel) == route.Layout:
		return layout
	}
	if _, routed, err := route.Parse(rel); err == nil && !routed {
		return component
	}
	return page
}

// read reads every file, several at a time.
func (b *builder) read() error {
	b.units = make([]*unit, len(b.paths))
	failures := make([]error, len(b.paths))
	next := make(chan int)
	var group sync.WaitGroup
	for range runtime.NumCPU() {
		group.Go(func() {
			for id := range next {
				path := b.paths[id]
				source, err := os.ReadFile(b.project.Path(filepath.FromSlash(path)))
				if err == nil {
					b.units[id], err = readFile(path, string(source), id, roleOf(path))
				}
				failures[id] = err
			}
		})
	}
	for id := range b.paths {
		next <- id
	}
	close(next)
	group.Wait()

	var errs Errors
	for _, err := range failures {
		switch err := err.(type) {
		case nil:
		case Errors:
			errs = append(errs, err...)
		default:
			return err
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// routes makes a route of every page and finds the layouts around it.
func (b *builder) routes() ([]program.Route, error) {
	var errs Errors
	var out []program.Route
	segments := map[int][]route.Segment{} // by file
	shapes := map[string]string{}
	for id, path := range b.paths {
		u := b.units[id]
		if u.role != page {
			continue
		}
		parsed, _, err := route.Parse(strings.TrimPrefix(path, project.Pages+"/"))
		if err != nil {
			errs = append(errs, u.file.Errorf(0, "%s", err))
			continue
		}
		if other, taken := shapes[route.Shape(parsed)]; taken {
			errs = append(errs, u.file.Errorf(0, "this page and %s give the same addresses", other))
			continue
		}
		shapes[route.Shape(parsed)] = path

		r := program.Route{File: id, Pattern: route.Pattern(parsed), Segments: []program.Segment{}, Layouts: []int{}}
		dynamic := false
		for _, s := range parsed {
			switch s.Kind {
			case route.Param:
				r.Segments = append(r.Segments, program.Segment{Param: s.Name})
				dynamic = true
			case route.Rest:
				r.Segments = append(r.Segments, program.Segment{Rest: s.Name})
				dynamic = true
			default:
				r.Segments = append(r.Segments, program.Segment{Text: s.Name})
			}
		}
		if dynamic && !u.script.paths {
			errs = append(errs, u.file.Errorf(0, "the name of this page has a parameter, so its script must say which pages there are: export async function getStaticPaths() { return [{ params: { … } }] }"))
		}
		// Every folder from pages/ down to the page may hold a layout.
		for dir := project.Pages; ; {
			if layout, found := b.ids[dir+"/"+route.Layout]; found {
				r.Layouts = append(r.Layouts, layout)
			}
			rest := strings.TrimPrefix(path, dir+"/")
			name, _, deeper := strings.Cut(rest, "/")
			if !deeper {
				break
			}
			dir += "/" + name
		}
		errs = append(errs, b.provided(u, r.Layouts)...)
		segments[id] = parsed
		out = append(out, r)
	}
	if len(errs) > 0 {
		return nil, errs
	}
	route.Sort(out, func(r program.Route) []route.Segment { return segments[r.File] })
	return out, nil
}

// provided checks that every slot a page fills for its layouts is shown by
// one of them, and the same for every layout and the layouts around it. The
// nearest layout that has the slot shows it.
func (b *builder) provided(page *unit, layouts []int) Errors {
	var errs Errors
	check := func(u *unit, around []int) {
		for name, at := range u.provided {
			shown := false
			for _, layout := range around {
				shown = shown || slices.Contains(b.units[layout].outlets, name)
			}
			if !shown {
				errs = append(errs, u.file.Errorf(at, `slot=%q gives markup to a layout, and no layout around this file has <g-slot name=%q />`, name, name))
			}
		}
	}
	check(page, layouts)
	for i, layout := range layouts {
		check(b.units[layout], layouts[:i])
	}
	return errs
}
