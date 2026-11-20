// Package evaluate is the second stage: it has the compiled JavaScript of a
// project compute pages and shared blocks, and keeps the result in the
// store.
//
// Go leads: it asks the JavaScript side which pages there are, names the
// pages and the shared blocks to compute, and takes their records. What it
// takes is compared with what the store has, and only what differs is
// written. An update may compute any part of the site: everything it does
// not name stays in the store as it is.
//
// Which part to compute, the data database of the project tells. Every page
// and shared block left a note of the rows it read from the database, and
// the database keeps a journal of the rows that change. An update computes
// what read a row that has changed, and nothing else: its cost follows the
// size of the change, not of the site. A project without a data database
// has nothing to tell what changed, and is computed whole every time.
package evaluate

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/grimalschi/shugo/internal/compile"
	"github.com/grimalschi/shugo/internal/data"
	"github.com/grimalschi/shugo/internal/project"
	"github.com/grimalschi/shugo/internal/store"
)

// Report tells what one update did.
type Report struct {
	// Build names the compiled project. Rebuilt tells that the store held
	// the result of another one, so everything was computed anew.
	Build   string `json:"build"`
	Rebuilt bool   `json:"rebuilt"`

	// Data tells what the data database of the project said, if it has one.
	Data *DataReport `json:"data,omitempty"`

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
	// Checked tells that everything was computed to check what a plain
	// update would have done. Unseen are the pages and shared blocks whose
	// record came out different although no row they read has changed: they
	// show something that Shugo does not see them read.
	Checked bool     `json:"checked,omitempty"`
	Unseen  []string `json:"unseen,omitempty"`
	// Time is how long every step took, in milliseconds.
	Time map[string]float64 `json:"time"`
}

// DataReport tells what an update learned from the data database.
type DataReport struct {
	// Whole tells that everything was computed: the first time, after the
	// project or the tables of the database changed, on request, or when the
	// journal of the database could not tell what had changed.
	Whole bool `json:"whole"`
	// Rows counts the rows that are not what they were at the previous update.
	Rows int `json:"rows"`
	// Routes counts the routes whose list of pages was read again.
	Routes int `json:"routes"`
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
	// Whole has everything computed, whatever the data database says has
	// changed: what a page reads from elsewhere is read again.
	Whole bool
	// Check computes every page and shared block too, and reports those
	// that came out different although no row they read has changed.
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
	kept, _, err := u.Build()
	if err != nil {
		return nil, err
	}
	if kept != compiled.ID && kept != "" {
		// Records name templates by their numbers in the compiled project,
		// so those of another one say nothing here.
		report.Rebuilt = true
		if err := k.clear(); err != nil {
			return nil, err
		}
	}
	took("open")

	k.check, report.Checked = options.Check, options.Check
	w := &worker{project: p, log: options.Log}
	defer w.stop()
	if path := p.Data(); path == "" {
		err = everything(w, k, took)
	} else {
		err = tracked(w, k, path, options.Whole || kept != compiled.ID, took)
	}
	if err == nil {
		err = w.end()
	}
	if err != nil {
		return nil, err
	}
	took("compute")

	if err := k.finish(); err != nil {
		// A shared block that failed earlier is an error only now.
		var script *ScriptError
		if errors.As(err, &script) {
			script.locate(p)
		}
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
func everything(w *worker, k *keeper, took func(string)) error {
	pages, err := w.list(nil, k)
	if err != nil {
		return err
	}
	took("list")
	// The whole site is compared, so the whole of what the store has is
	// read at once.
	if err := k.preload(); err != nil {
		return err
	}
	k.again = func(string, bool) bool { return true }
	k.strict = true

	units := make([]unit, len(pages))
	there := make(map[string]struct{}, len(pages))
	for i, page := range pages {
		units[i] = unit{name: page.path, route: page.route}
		there[page.path] = struct{}{}
	}
	if err := w.compute(units, k); err != nil {
		return err
	}
	if err := shared(w, k); err != nil {
		return err
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

// shared computes the shared blocks that the records of this run show and
// that are still to be computed. A shared block is computed once it is
// known that something shows it; what it shows itself comes with the next
// round.
func shared(w *worker, k *keeper) error {
	for names := k.take(); len(names) > 0; names = k.take() {
		units := make([]unit, len(names))
		for i, name := range names {
			units[i] = unit{shared: true, name: name}
		}
		if err := w.compute(units, k); err != nil {
			return err
		}
	}
	return nil
}

// tracked brings a project with a data database up to date: it computes
// what read the rows that the journal of the database tells have changed.
// whole has it compute everything instead; so has a journal that cannot
// tell what has changed since the records were computed.
func tracked(w *worker, k *keeper, path string, whole bool, took func(string)) error {
	source, err := data.Open(path)
	if err != nil {
		return err
	}
	defer source.Close()
	if k.shapes, err = newShapes(k.update, source); err != nil {
		return err
	}
	state, err := k.update.Data()
	if err != nil {
		return err
	}
	var journal string
	var at int64
	fmt.Sscanf(state, "%s %d", &journal, &at)
	k.report.Data = &DataReport{}

	var changes []data.Change
	now := at
	if whole = whole || journal != source.ID; !whole {
		// Where the journal stands is read before the data is: a row that
		// changes while the pages are computed is computed again next time.
		var lost bool
		if changes, now, lost, err = source.Changes(at); err != nil {
			return err
		}
		whole = lost
	}
	if whole {
		if now, err = source.Position(); err != nil {
			return err
		}
		k.report.Data.Whole = true
		took("data")
		err = everything(w, k, took)
	} else {
		k.report.Data.Rows = len(changes)
		var readers map[string]struct{}
		if readers, err = k.shapes.reached(changes); err == nil {
			took("data")
			if k.check {
				err = checked(w, k, readers, took)
			} else {
				err = part(w, k, readers, took)
			}
		}
	}
	if err != nil {
		return err
	}
	return k.update.SetData(fmt.Sprintf("%s %d", source.ID, now))
}

// checked computes everything where the journal told to compute only the
// readers given, and has the keeper note what came out different without
// having been told: a page that reads its data past the database, or takes
// it from what another page left in a module.
func checked(w *worker, k *keeper, readers map[string]struct{}, took func(string)) error {
	k.reached = map[string]struct{}{}
	for reader := range readers {
		route, is := store.ReaderRoute(reader)
		if !is {
			k.reached[reader] = struct{}{}
			continue
		}
		// A list of pages that is read again has its new pages computed,
		// and all of them if the route gives its pages more than addresses.
		props, _, _, err := k.update.Route(route)
		if err != nil {
			return err
		}
		if props {
			k.reached[reader] = struct{}{}
		}
	}
	return everything(w, k, took)
}

// part computes the readers that a change of the data has reached: pages,
// shared blocks, and lists of pages of routes.
func part(w *worker, k *keeper, readers map[string]struct{}, took func(string)) error {
	var routes []int
	var blocks []string
	pages := map[string]int{} // the route of every page to compute
	for reader := range readers {
		if route, is := store.ReaderRoute(reader); is {
			routes = append(routes, route)
		} else if id, is := store.UserShared(reader); is {
			name, err := k.update.SharedName(id)
			if err != nil {
				return err
			}
			blocks = append(blocks, name)
		} else if kept, found, err := k.page(reader); err != nil {
			return err
		} else if found {
			pages[reader] = kept.Route
		}
	}
	// A page of a route that gives its pages more than their addresses is
	// computed from what the route gives it, so the route is listed again.
	for _, route := range pages {
		props, _, _, err := k.update.Route(route)
		if err != nil {
			return err
		}
		if props {
			routes = append(routes, route)
		}
	}
	if err := relist(w, k, routes, pages); err != nil {
		return err
	}
	took("list")

	units := make([]unit, 0, len(pages)+len(blocks))
	for path, route := range pages {
		units = append(units, unit{name: path, route: route})
	}
	for _, name := range blocks {
		units = append(units, unit{shared: true, name: name})
	}
	// The order is of no matter to the result; a fixed one makes runs alike.
	sort.Slice(units, func(a, b int) bool { return units[a].name < units[b].name })
	if err := w.compute(units, k); err != nil {
		return err
	}
	return shared(w, k)
}

// relist reads again which pages routes have. A page a route no longer
// gives is dropped, and a page it gives for the first time is added to
// pages, to be computed. So is a page it takes over from a less exact
// route, and every page of a route that gives its pages more than their
// addresses.
func relist(w *worker, k *keeper, routes []int, pages map[string]int) error {
	done := map[int]bool{}
	for len(routes) > 0 {
		sort.Ints(routes)
		queue := routes[:0]
		for _, route := range routes {
			if !done[route] {
				done[route] = true
				queue = append(queue, route)
			}
		}
		if len(queue) == 0 {
			return nil
		}
		k.report.Data.Routes += len(queue)
		found, err := w.list(queue, k)
		if err != nil {
			return err
		}
		given := map[int]map[string]struct{}{}
		for _, route := range queue {
			given[route] = map[string]struct{}{}
		}
		for _, page := range found {
			given[page.route][page.path] = struct{}{}
		}

		var next []int
		for _, route := range queue {
			before := map[string]struct{}{}
			if err := k.update.RoutePages(route, func(path string) { before[path] = struct{}{} }); err != nil {
				return err
			}
			for path := range before {
				if _, still := given[route][path]; still {
					continue
				}
				if err := k.remove(path); err != nil {
					return err
				}
				delete(pages, path)
				// A less exact route may give the same address: it was
				// hidden behind this one so far.
				for other := route + 1; other < len(k.site.Routes); other++ {
					if !done[other] && fits(&k.site.Routes[other], path) {
						next = append(next, other)
					}
				}
			}
			// What the route gives its pages beside their addresses may be
			// other than it was, and only the route knows: all its pages are
			// computed again.
			props, _, _, err := k.update.Route(route)
			if err != nil {
				return err
			}
			for path := range given[route] {
				if _, had := before[path]; had {
					if props {
						pages[path] = route
					}
					continue
				}
				kept, known, err := k.update.PageKept(path)
				switch {
				case err != nil:
					return err
				case !known || kept.Route > route:
					pages[path] = route
				case kept.Route < route:
					// A more exact route has the address.
					k.report.Shadowed++
				}
			}
		}
		routes = next
	}
	return nil
}

// unit names what is computed as one: a page, by its address and the route
// that gives it, or a shared block, by its name.
type unit struct {
	shared bool
	name   string
	route  int
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
