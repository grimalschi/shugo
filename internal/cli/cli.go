// Package cli is the shugo command: it reads the arguments and runs the
// stages of the framework in the order a command asks for.
package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/grimalschi/shugo/internal/compile"
	"github.com/grimalschi/shugo/internal/evaluate"
	"github.com/grimalschi/shugo/internal/project"
	"github.com/grimalschi/shugo/internal/server"
	"github.com/grimalschi/shugo/internal/store"
)

const usage = `shugo builds a static site and shows it.

  shugo build  [folder]   compile the project and compute every page
  shugo update [folder]   bring the site up to date: after a change of the data,
                          only what read a changed row is computed; after a
                          change of the project, it is compiled again and
                          every page is computed
  shugo serve  [folder]   show the site, building every page on request
  shugo export [folder]   write the site as files for any web server; after the
                          first time, only what has changed since
  shugo dev    [folder]   update, show, and update again whenever a file changes

The folder is the project; without it, the current one. Options:

  -json          print the report of build, update and export as JSON
  -check         with update: compute every page, whatever has changed, and
                 tell which pages came out different although no row of the
                 data database that they read has changed
  -runtime NAME  what runs the JavaScript of the project: bun or node
                 (default: the "runtime" of shugo.json, else bun if installed)
  -addr ADDRESS  where serve and dev listen (default localhost:4321)
  -out FOLDER    where export writes (default _build/dist in the project)
`

// Store is the file inside the output folder where the computed site is kept.
const Store = "store.db"

// options are the options every command takes.
var options = map[string][]string{
	"build":  {"json", "runtime"},
	"update": {"json", "check", "runtime"},
	"serve":  {"addr"},
	"export": {"json", "out"},
	"dev":    {"addr", "runtime"},
}

// Main runs the command and returns its exit status.
func Main(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 || arguments[0] == "help" || arguments[0] == "-h" || arguments[0] == "--help" {
		fmt.Fprint(stdout, usage)
		return 0
	}
	c := &command{stdout: stdout, stderr: stderr}
	flags := flag.NewFlagSet(arguments[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.BoolVar(&c.json, "json", false, "")
	flags.BoolVar(&c.check, "check", false, "")
	flags.StringVar(&c.addr, "addr", "localhost:4321", "")
	flags.StringVar(&c.out, "out", "", "")
	flags.StringVar(&c.runtime, "runtime", "", "")
	flags.Usage = func() { fmt.Fprint(stderr, usage) }
	// The folder may stand before or after the options.
	var rest []string
	for remaining := arguments[1:]; ; {
		if err := flags.Parse(remaining); err != nil {
			return 2
		}
		if flags.NArg() == 0 {
			break
		}
		rest = append(rest, flags.Arg(0))
		remaining = flags.Args()[1:]
	}
	dir := "."
	switch len(rest) {
	case 0:
	case 1:
		dir = rest[0]
	default:
		fmt.Fprintf(stderr, "shugo %s takes one folder, got %d\n", arguments[0], len(rest))
		return 2
	}

	run := map[string]func() error{"build": c.build, "update": c.update, "serve": c.serve, "export": c.export, "dev": c.dev}[arguments[0]]
	if run == nil {
		fmt.Fprintf(stderr, "shugo: there is no command %q\n\n%s", arguments[0], usage)
		return 2
	}
	// An option the command does not use is a mistake, not a wish to ignore.
	var stray []string
	flags.Visit(func(f *flag.Flag) {
		if !slices.Contains(options[arguments[0]], f.Name) {
			stray = append(stray, "-"+f.Name)
		}
	})
	if len(stray) > 0 {
		fmt.Fprintf(stderr, "shugo %s does not take %s\n", arguments[0], strings.Join(stray, ", "))
		return 2
	}
	var err error
	if c.project, err = project.Open(dir); err == nil {
		if c.runtime != "" {
			c.project.Config.Runtime = c.runtime
		}
		err = run()
	}
	if err != nil {
		c.report(err)
		return 1
	}
	return 0
}

type command struct {
	stdout, stderr io.Writer
	project        *project.Project
	json, check    bool
	position       int64 // of the journal of the data database, at the latest build or update
	addr, out      string
	runtime        string
}

// open opens the store of the project.
func (c *command) open() (*store.Store, error) {
	if err := os.MkdirAll(c.project.Path(project.Output), 0o755); err != nil {
		return nil, err
	}
	return store.Open(c.project.Path(project.Output, Store))
}

// build compiles the project and computes every page.
func (c *command) build() error {
	st, err := c.open()
	if err != nil {
		return err
	}
	defer st.Close()
	return c.rebuild(st)
}

func (c *command) rebuild(st *store.Store) error {
	started := time.Now()
	compiled, err := compile.Build(c.project)
	if err != nil {
		return err
	}
	for _, warning := range compiled.Warnings {
		fmt.Fprintln(c.stderr, "warning:", warning)
	}
	compiledIn := time.Since(started)
	// A build computes everything: what the pages read from elsewhere than
	// the data database is read again.
	report, err := evaluate.Run(c.project, compiled, st, evaluate.Options{Log: c.stderr, Whole: true})
	if err != nil {
		return err
	}
	report.Time["compile"] = float64(compiledIn.Microseconds()) / 1000
	report.Time["total"] += report.Time["compile"]
	c.noted(report)
	return c.print(report, func() string { return summary("built", report) })
}

// noted keeps where the journal of the data database stood when the site
// was brought up to date: development mode tells by it whether the data has
// changed since.
func (c *command) noted(report *evaluate.Report) {
	if report.Data != nil {
		c.position = report.Data.Position
	}
}

// update computes the pages of a compiled project again. If the project
// itself has changed since it was compiled, it is compiled again first.
func (c *command) update() error {
	st, err := c.open()
	if err != nil {
		return err
	}
	defer st.Close()
	return c.renew(st)
}

func (c *command) renew(st *store.Store) error {
	if changed, err := compile.Changed(c.project); err != nil {
		return err
	} else if changed {
		fmt.Fprintln(c.stderr, "the project has changed since it was compiled: compiling it again")
		return c.rebuild(st)
	}
	compiled, err := compile.Load(c.project)
	if err != nil {
		return err
	}
	report, err := evaluate.Run(c.project, compiled, st, evaluate.Options{Log: c.stderr, Check: c.check})
	if err != nil {
		return err
	}
	c.noted(report)
	if err := c.print(report, func() string { return summary("updated", report) }); err != nil {
		return err
	}
	if len(report.Unseen) > 0 {
		// The store is right now: everything was computed. But the next
		// plain update would again leave such a page as it is.
		return fmt.Errorf("%d pages or shared blocks came out different although no row they read has changed; the first: %s.\n"+
			"They read data past the data database — a file, the network, something another page left in a module — or a package does so for them. "+
			"Put that data into the database and ask for it with query from \"shugo\"; until then only `shugo build` reads it again", len(report.Unseen), report.Unseen[0])
	}
	return nil
}

// print writes a report: as JSON for programs, in words for people.
func (c *command) print(report any, words func() string) error {
	if c.json {
		encoder := json.NewEncoder(c.stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(report)
	}
	_, err := fmt.Fprintln(c.stdout, words())
	return err
}

// summary says in a few lines what an update did.
func summary(verb string, r *evaluate.Report) string {
	var out strings.Builder
	fmt.Fprintf(&out, "%s in %.0f ms: %d pages, %d shared blocks", verb, r.Time["total"], r.Pages.Total, r.Shared.Total)
	fmt.Fprintf(&out, "\n  computed %d pages and %d shared blocks, %d blocks in all", r.Computed.Pages, r.Computed.Shared, r.Blocks)
	if r.Data != nil && !r.Data.Whole {
		fmt.Fprintf(&out, "\n  %d rows of the data have changed", r.Data.Rows)
	}
	if r.Data != nil && len(r.Data.Indexes) > 0 {
		fmt.Fprintf(&out, "\n  made %d indexes in the data database in %.0f ms: %s", len(r.Data.Indexes), r.Time["indexes"], strings.Join(r.Data.Indexes, ", "))
	}
	if r.Rebuilt {
		out.WriteString("\n  the project was compiled anew, so everything was computed from scratch")
	}
	line := func(what string, c evaluate.Changes) {
		if c.Added+c.Changed+c.Removed == 0 {
			return
		}
		fmt.Fprintf(&out, "\n  %s: %d added, %d changed, %d removed", what, c.Added, c.Changed, c.Removed)
		for label, names := range map[string][]string{"changed": c.ChangedNames, "removed": c.RemovedNames} {
			if len(names) > 0 && len(names) <= 10 {
				fmt.Fprintf(&out, "\n    %s: %s", label, strings.Join(names, ", "))
			}
		}
	}
	line("pages", r.Pages)
	line("shared blocks", r.Shared)
	if r.Render > r.Pages.Added+r.Pages.Changed {
		fmt.Fprintf(&out, "\n  the HTML differs on %d pages: they show a shared block that has changed", r.Render)
	}
	if r.Checked && r.Data != nil && !r.Data.Whole && len(r.Unseen) == 0 {
		out.WriteString("\n  checked: everything was computed, and every page that came out different had read a row that has changed")
	}
	if r.Pages.Added+r.Pages.Changed+r.Pages.Removed+r.Shared.Added+r.Shared.Changed+r.Shared.Removed == 0 {
		out.WriteString("\n  nothing has changed")
	}
	return out.String()
}

// serve shows the site that the store holds.
func (c *command) serve() error {
	st, err := c.open()
	if err != nil {
		return err
	}
	defer st.Close()
	pages := server.NewPages(st)
	if err := pages.Ready(); err != nil {
		return err
	}
	return c.listen(&server.Server{Pages: pages})
}

// listen answers requests until the process is stopped.
func (c *command) listen(s *server.Server) error {
	s.Public = c.project.Path(project.Public)
	s.Assets = c.project.Path(project.Output, compile.Assets)
	s.Failed = func(err error) { fmt.Fprintln(c.stderr, "error:", err) }
	listener, err := net.Listen("tcp", c.addr)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.stdout, "the site is at http://%s\n", listener.Addr())
	return http.Serve(listener, s)
}

// export writes the site as files.
func (c *command) export() error {
	st, err := c.open()
	if err != nil {
		return err
	}
	defer st.Close()
	out := c.out
	if out == "" {
		out = c.project.Path(project.Output, "dist")
	}
	started := time.Now()
	done, err := server.Export(server.NewPages(st), c.project.Path(project.Public), c.project.Path(project.Output, compile.Assets), out)
	if err != nil {
		return err
	}
	return c.print(done, func() string {
		return fmt.Sprintf("exported to %s in %.0f ms: %d pages, %d of them built; %d files, %d written, %d removed",
			out, float64(time.Since(started).Microseconds())/1000, done.Pages, done.Built, done.Files, done.Written, done.Removed)
	})
}

// report prints an error the way its kind deserves: a mistake in the
// project with the lines around it, anything else as it is.
func (c *command) report(err error) {
	var mistakes compile.Errors
	var script *evaluate.ScriptError
	switch {
	case errors.As(err, &mistakes):
		for _, mistake := range mistakes {
			fmt.Fprintf(c.stderr, "error: %s\n%s", mistake, excerpt(c.project, mistake.Path, mistake.Line, mistake.Column))
		}
	case errors.As(err, &script):
		fmt.Fprintf(c.stderr, "error: %s\n", script)
		if len(script.Frames) > 0 {
			at := script.Frames[0]
			fmt.Fprint(c.stderr, excerpt(c.project, at.File, at.Line, at.Column))
		}
	default:
		fmt.Fprintln(c.stderr, "error:", err)
	}
}

// excerpt returns the line of a file of the project that a message points
// at, with the line before it and a mark under the column.
func excerpt(p *project.Project, file string, line, column int) string {
	if p == nil {
		return ""
	}
	data, err := os.ReadFile(p.Path(file))
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	if line < 1 || line > len(lines) {
		return ""
	}
	var out strings.Builder
	for n := max(1, line-1); n <= line; n++ {
		fmt.Fprintf(&out, "%6d | %s\n", n, strings.TrimRight(lines[n-1], "\r"))
	}
	fmt.Fprintf(&out, "       | %s^\n", strings.Repeat(" ", max(0, column-1)))
	return out.String()
}
