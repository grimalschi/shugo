// Package evaluate is the second stage: it has the compiled JavaScript of a
// project compute pages and shared blocks, and keeps the result in the
// store.
//
// Go leads: it asks the JavaScript side which pages there are, names the
// pages and the shared blocks to compute, and takes their records. What it
// takes is compared with what the store has, and only what differs is
// written. An update may compute any part of the site: everything it does
// not name stays in the store as it is.
package evaluate

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/grimalschi/shugo/internal/compile"
	"github.com/grimalschi/shugo/internal/project"
	"github.com/grimalschi/shugo/internal/store"
)

// Report tells what one update did.
type Report struct {
	// Build names the compiled project. Rebuilt tells that the store held
	// the result of another one, so everything was computed anew.
	Build   string `json:"build"`
	Rebuilt bool   `json:"rebuilt"`

	// Computed counts the pages and the shared blocks the update computed,
	// Blocks the blocks they consist of. Pages and Shared tell what became
	// of the records: computing a record again often gives the same one.
	Computed struct {
		Pages  int `json:"pages"`
		Shared int `json:"shared"`
	} `json:"computed"`
	Blocks int     `json:"blocks"`
	Pages  Changes `json:"pages"`
	Shared Changes `json:"shared"`
	// Values counts the distinct values of blocks that are kept.
	Values struct {
		Total   int `json:"total"`
		Added   int `json:"added"`
		Removed int `json:"removed"`
	} `json:"values"`
	// Render counts the pages whose HTML is not what it was: the pages that
	// are new or have changed themselves, and the pages that show a shared
	// block that has. RenderNames are the first of them.
	Render      int      `json:"render"`
	RenderNames []string `json:"renderNames,omitempty"`
	// Shadowed counts addresses that a dynamic page gave and a more exact
	// page had already taken.
	Shadowed int `json:"shadowed"`
	// Time is how long every step took, in milliseconds.
	Time map[string]float64 `json:"time"`
}

// Changes tells what happened to the pages or to the shared blocks.
type Changes struct {
	Total   int `json:"total"`
	Added   int `json:"added"`
	Changed int `json:"changed"`
	Removed int `json:"removed"`
	// The names of the first of them, sorted; Names limits how many.
	AddedNames   []string `json:"addedNames,omitempty"`
	ChangedNames []string `json:"changedNames,omitempty"`
	RemovedNames []string `json:"removedNames,omitempty"`
}

// note counts one page or shared block under what happened to it.
func (c *Changes) note(count *int, names *[]string, name string, limit int) {
	if *count++; len(*names) < limit {
		*names = append(*names, name)
	}
}

// Options tune a run.
type Options struct {
	// Log receives what the project's code prints.
	Log io.Writer
	// Names is how many names of pages a report lists at most.
	Names int
}

// Run brings the store up to date with the project. If it fails, the store
// stays as it was.
func Run(p *project.Project, compiled *compile.Result, st *store.Store, options Options) (*Report, error) {
	started := time.Now()
	lap := started
	report := &Report{Build: compiled.ID, Time: map[string]float64{}}
	took := func(step string) {
		report.Time[step] += float64(time.Since(lap).Microseconds()) / 1000
		lap = time.Now()
	}
	if options.Log == nil {
		options.Log = os.Stderr
	}
	if options.Names == 0 {
		options.Names = 1000
	}

	u, err := st.Update()
	if err != nil {
		return nil, err
	}
	defer u.Rollback()
	site, err := compiled.Site.Encode()
	if err != nil {
		return nil, err
	}
	k, err := newKeeper(compiled.Site, u, report, options.Names)
	if err != nil {
		return nil, err
	}
	if kept, _, err := u.Build(); err != nil {
		return nil, err
	} else if kept != compiled.ID && kept != "" {
		// Records name templates by their numbers in the compiled project,
		// so those of another one say nothing here.
		report.Rebuilt = true
		if err := k.clear(); err != nil {
			return nil, err
		}
	}
	took("open")

	s, err := start(p, options.Log)
	if err != nil {
		return nil, err
	}
	defer s.stop()
	pages, err := s.list(report)
	if err != nil {
		return nil, err
	}
	took("list")

	if err := everything(s, k, pages); err != nil {
		return nil, err
	}
	if err := s.end(); err != nil {
		return nil, err
	}
	took("compute")

	if err := k.finish(); err != nil {
		return nil, err
	}
	if err := u.SetBuild(compiled.ID, site); err != nil {
		return nil, err
	}
	if err := u.Commit(); err != nil {
		return nil, err
	}
	took("save")
	report.Time["total"] = float64(time.Since(started).Microseconds()) / 1000
	return report, nil
}

// everything computes all pages and every shared block they show, and drops
// the pages the project no longer has.
func everything(s *session, k *keeper, pages []listed) error {
	// The whole site is compared, so the whole of what the store has is
	// read at once.
	if err := k.preload(); err != nil {
		return err
	}
	k.again = func(string, bool) bool { return true }

	units := make([]unit, len(pages))
	there := make(map[string]struct{}, len(pages))
	for i, page := range pages {
		units[i] = unit{name: page.path}
		there[page.path] = struct{}{}
	}
	if err := s.compute(units, k); err != nil {
		return err
	}
	// A shared block is computed once it is known that something shows it;
	// what it shows itself comes with the next round.
	for names := k.take(); len(names) > 0; names = k.take() {
		units = units[:0]
		for _, name := range names {
			units = append(units, unit{shared: true, name: name})
		}
		if err := s.compute(units, k); err != nil {
			return err
		}
	}
	for path := range k.pages {
		if _, still := there[path]; !still {
			if err := k.remove(path); err != nil {
				return err
			}
		}
	}
	return nil
}

// unit names what is computed as one: a page, by its address, or a shared
// block, by its name.
type unit struct {
	shared bool
	name   string
}

func (u unit) String() string {
	if u.shared {
		return "the shared block " + u.name
	}
	return "the page " + u.name
}

// errorf makes an error about what the project's code sent.
func errorf(format string, arguments ...any) error {
	return fmt.Errorf("the project's code sent "+format, arguments...)
}
