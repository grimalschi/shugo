// Package evaluate is the second stage: it has the compiled JavaScript of a
// project compute pages and shared blocks, and keeps the result in the
// store.
//
// Go leads. It asks the JavaScript side which pages there are; then for the
// fingerprint of every page and of every shared block the store knows. The
// JavaScript side runs their scripts to answer: that loads all the data. A
// fingerprint covers exactly what the template of a page reads of its
// script, so a page whose fingerprint is what the store has would come out
// the same, and is left alone. Go names the rest, and only their templates
// are computed. What comes back is compared with the store once more, and
// only what differs is written.
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

	// Loaded counts the pages and the shared blocks whose scripts ran and
	// whose fingerprints were compared. Computed counts those whose
	// templates were computed after that: the ones that are new or whose
	// fingerprint differs. Blocks is how many blocks those consist of. Pages
	// and Shared tell what became of the records: computing a record again
	// may give the same one.
	Loaded   Counts  `json:"loaded"`
	Computed Counts  `json:"computed"`
	Blocks   int     `json:"blocks"`
	Pages    Changes `json:"pages"`
	Shared   Changes `json:"shared"`
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
	// Checked tells that everything was computed to check the fingerprints.
	// Unseen are the pages and shared blocks whose record came out
	// different although their fingerprint did not: they show something
	// that Shugo does not see them read.
	Checked bool     `json:"checked,omitempty"`
	Unseen  []string `json:"unseen,omitempty"`
	// Time is how long every step took, in milliseconds.
	Time map[string]float64 `json:"time"`
}

// Counts is a number of pages and a number of shared blocks.
type Counts struct {
	Pages  int `json:"pages"`
	Shared int `json:"shared"`
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
	// Check computes every page and shared block, whatever the fingerprints
	// say, and reports those that came out different although their
	// fingerprint did not.
	Check bool
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

	k.check, report.Checked = options.Check, options.Check
	if err := changed(s, k, pages); err != nil {
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

// changed computes the pages and shared blocks whose fingerprint is not
// what the store has, and drops the pages the project no longer has.
func changed(s *session, k *keeper, pages []listed) error {
	// Every page is compared, so the whole of what the store has is read
	// at once.
	if err := k.preload(); err != nil {
		return err
	}
	units := make([]unit, 0, len(pages)+len(k.shared))
	there := make(map[string]struct{}, len(pages))
	for _, page := range pages {
		units = append(units, unit{name: page.path})
		there[page.path] = struct{}{}
	}
	// A shared block the store knows may show something else by now, if its
	// script loads data. One that only shows what it is given cannot: its
	// name says all that it shows.
	for name := range k.shared {
		if k.loads(name) {
			units = append(units, unit{shared: true, name: name})
		}
	}

	var compute []unit
	err := s.prints(units, func(u unit, print uint64, failure *ScriptError) error {
		if failure != nil {
			k.failed[u.name] = failure
			return nil
		}
		kept, known := k.shared[u.name]
		if u.shared {
			k.report.Loaded.Shared++
		} else {
			k.report.Loaded.Pages++
			kept, known = k.pages[u.name]
		}
		if !known || kept.Print != print || k.check {
			compute = append(compute, u)
		}
		return nil
	})
	if err != nil {
		return err
	}
	// The shared blocks of the store have just been compared, so one that a
	// computed record shows is computed only if the store does not have it.
	// A check computes everything it meets.
	k.again = func(string, bool) bool { return k.check }
	if err := s.compute(compute, k); err != nil {
		return err
	}
	// What a shared block shows itself comes with the next round.
	for names := k.take(); len(names) > 0; names = k.take() {
		compute = compute[:0]
		for _, name := range names {
			compute = append(compute, unit{shared: true, name: name})
		}
		if err := s.compute(compute, k); err != nil {
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
