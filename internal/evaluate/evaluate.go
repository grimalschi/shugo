// Package evaluate is the second stage: it runs the compiled JavaScript of a
// project, which loads the data and computes the values of every page, and
// keeps the result in the store. It compares what was computed with what the
// store has and writes only what differs, so that after a change in the data
// only the pages and the shared blocks that show it are touched.
package evaluate

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/grimalschi/shugo/internal/compile"
	"github.com/grimalschi/shugo/internal/program"
	"github.com/grimalschi/shugo/internal/project"
	"github.com/grimalschi/shugo/internal/record"
	"github.com/grimalschi/shugo/internal/render"
	"github.com/grimalschi/shugo/internal/store"
)

// Report tells what one update did.
type Report struct {
	// Build names the compiled project. Rebuilt tells that the store held
	// the result of another one, so everything was computed anew.
	Build   string `json:"build"`
	Rebuilt bool   `json:"rebuilt"`

	Pages  Changes `json:"pages"`
	Shared Changes `json:"shared"`
	// Blocks is how many blocks all pages and shared blocks consist of.
	// Values counts the distinct values of blocks that are kept.
	Blocks int `json:"blocks"`
	Values struct {
		Total   int `json:"total"`
		Added   int `json:"added"`
		Removed int `json:"removed"`
	} `json:"values"`
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

// Options tune a run.
type Options struct {
	// Log receives what the project's code prints.
	Log io.Writer
	// Names is how many names of pages a report lists at most.
	Names int
}

// Limits on what the project's code may send. It runs outside of our
// control, so nothing it sends is taken on trust.
const (
	maxFrame  = 1 << 30 // bytes of one message
	maxBlocks = 1 << 22 // blocks of one page
)

// Run computes the site and brings the store up to date with it. If it
// fails, the store stays as it was.
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
	if kept, _, err := u.Build(); err != nil {
		return nil, err
	} else if kept != compiled.ID {
		// Records name templates by their numbers in the compiled project,
		// so those of another one say nothing here. Values are kept: most of
		// them will be computed again as they were.
		report.Rebuilt = kept != ""
		if err := u.ClearRecords(); err != nil {
			return nil, err
		}
	}
	k := &keeper{site: compiled.Site, update: u, report: report, names: options.Names, numbers: map[int64]int64{}, live: map[record.Hash]struct{}{}}
	if k.pages, err = u.KeptPages(); err != nil {
		return nil, err
	}
	if k.shared, err = u.KeptShared(); err != nil {
		return nil, err
	}
	if k.values, err = u.KeptValues(); err != nil {
		return nil, err
	}
	k.seenPages, k.seenShared = make(map[string]struct{}, len(k.pages)), make(map[string]struct{}, len(k.shared))
	took("open")

	if err := compute(p, options.Log, k); err != nil {
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

// compute runs the JavaScript of the project and hands every record it
// sends to the keeper.
func compute(p *project.Project, log io.Writer, k *keeper) error {
	reader, writer, err := os.Pipe()
	if err != nil {
		return err
	}
	defer reader.Close()
	command := exec.Command(p.Runtime(), p.Path(project.Output, compile.Bundle))
	command.Dir = p.Root
	command.Stdout, command.Stderr = log, log
	// The results come through a pipe of their own, so that whatever the
	// project prints cannot get mixed into them.
	command.ExtraFiles = []*os.File{writer}
	command.Env = append(os.Environ(), "SHUGO_OUT=3")
	if err := command.Start(); err != nil {
		writer.Close()
		return fmt.Errorf("cannot start %s to run the project's code: %w", p.Runtime(), err)
	}
	writer.Close()

	failure := receive(bufio.NewReaderSize(reader, 1<<20), k)
	if failure != nil {
		// What follows is of no use; let the process end instead of blocking on the pipe.
		command.Process.Kill()
	}
	exit := command.Wait()
	var script *ScriptError
	switch {
	case errors.As(failure, &script):
		script.locate(p)
		return script
	case failure != nil:
		return failure
	case exit != nil:
		return fmt.Errorf("%s ended before the site was computed: %w", p.Runtime(), exit)
	case !k.done:
		return fmt.Errorf("%s ended without saying that the site was computed", p.Runtime())
	}
	return nil
}

// receive reads the messages of the project's code: a length in four bytes,
// then a letter that says what the message is, then the message.
func receive(in *bufio.Reader, k *keeper) error {
	var head [4]byte
	var message []byte
	for {
		if _, err := io.ReadFull(in, head[:]); err == io.EOF {
			return nil
		} else if err != nil {
			return fmt.Errorf("the results of the project's code are cut short: %w", err)
		}
		size := int(binary.LittleEndian.Uint32(head[:]))
		if size < 1 || size > maxFrame {
			return fmt.Errorf("the project's code sent a message of %d bytes", size)
		}
		if cap(message) < size {
			message = make([]byte, size)
		}
		message = message[:size]
		if _, err := io.ReadFull(in, message); err != nil {
			return fmt.Errorf("the results of the project's code are cut short: %w", err)
		}
		switch body := message[1:]; message[0] {
		case 'R':
			for len(body) > 0 {
				var one []byte
				one, body, _ = bytes.Cut(body, []byte{3})
				if err := k.keep(one); err != nil {
					return err
				}
			}
		case 'E':
			failure := &ScriptError{}
			if err := json.Unmarshal(body, failure); err != nil {
				return fmt.Errorf("the project's code failed, and its report cannot be read: %w", err)
			}
			return failure
		case 'D':
			var counts struct{ Shadowed int }
			if err := json.Unmarshal(body, &counts); err != nil {
				return fmt.Errorf("the project's code sent a damaged summary: %w", err)
			}
			k.report.Shadowed = counts.Shadowed
			k.done = true
		default:
			return fmt.Errorf("the project's code sent a message of an unknown kind %q", message[0])
		}
	}
}

// keeper takes the records of one run and writes to the store those that
// differ from what it has.
type keeper struct {
	site   *program.Site
	update *store.Update
	report *Report
	names  int
	done   bool

	// What the store had before the run.
	pages  map[string]record.Hash
	shared map[string]store.Kept
	values map[record.Hash]struct{}

	// What the run has shown so far.
	seenPages, seenShared map[string]struct{}
	numbers               map[int64]int64 // the number a shared block has in this run → its number in the store
	live                  map[record.Hash]struct{}
}

// keep takes one record as the project's code sent it: a header and blocks
// split by \x01, their fields split by \x02.
func (k *keeper) keep(data []byte) error {
	header, rest, _ := bytes.Cut(data, []byte{1})
	fields := bytes.Split(header, []byte{2})
	if len(fields) != 4 || len(fields[0]) != 1 {
		return errors.New("the project's code sent a record with a damaged header")
	}
	kind, name := fields[0][0], string(fields[1])
	origin, err1 := strconv.ParseInt(string(fields[2]), 10, 64)
	top, err2 := strconv.Atoi(string(fields[3]))
	if err1 != nil || err2 != nil {
		return errors.New("the project's code sent a record with a damaged header")
	}
	where := "the shared block " + name
	if kind == 'p' {
		where = "the page " + name
	}

	rec := &record.Record{Top: top}
	var values [][]byte
	for len(rest) > 0 {
		if len(rec.Blocks) == maxBlocks {
			return fmt.Errorf("%s has more than %d blocks", where, maxBlocks)
		}
		var one []byte
		one, rest, _ = bytes.Cut(rest, []byte{1})
		block, data, err := k.block(one)
		if err != nil {
			return fmt.Errorf("%s: %w", where, err)
		}
		rec.Blocks = append(rec.Blocks, block)
		values = append(values, data)
	}
	k.report.Blocks += len(rec.Blocks)
	encoded := rec.Encode()
	hash := record.HashOf(encoded)

	// A record that is what it was needs nothing: its values are in the
	// store already, and it was checked when it got there.
	changed := true
	switch kind {
	case 'p':
		if !validPath(name) {
			return fmt.Errorf("the project's code sent a page with the address %q", name)
		}
		if _, twice := k.seenPages[name]; twice {
			return fmt.Errorf("the project's code sent %s twice", where)
		}
		k.seenPages[name] = struct{}{}
		kept, known := k.pages[name]
		if changed = !known || kept != hash; changed {
			if err := k.check(rec, values); err != nil {
				return fmt.Errorf("%s: %w", where, err)
			}
			if err := k.update.PutPage(name, hash, encoded); err != nil {
				return err
			}
			k.count(&k.report.Pages, name, known)
		}
	case 's':
		if _, twice := k.seenShared[name]; twice {
			return fmt.Errorf("the project's code sent %s twice", where)
		}
		k.seenShared[name] = struct{}{}
		kept, known := k.shared[name]
		if !known {
			if kept.ID, err1 = k.update.NewShared(name); err1 != nil {
				return err1
			}
		}
		k.numbers[origin] = kept.ID
		if changed = !known || kept.Hash != hash; changed {
			if rec.Blocks[rec.Top].Use {
				return fmt.Errorf("%s does not start with a template", where)
			}
			if err := k.check(rec, values); err != nil {
				return fmt.Errorf("%s: %w", where, err)
			}
			if err := k.update.PutShared(kept.ID, hash, encoded); err != nil {
				return err
			}
			k.count(&k.report.Shared, name, known)
		}
	default:
		return fmt.Errorf("the project's code sent a record of an unknown kind %q", kind)
	}

	for i := range rec.Blocks {
		b := &rec.Blocks[i]
		if b.Use {
			continue
		}
		if _, known := k.live[b.Values]; known {
			continue
		}
		k.live[b.Values] = struct{}{}
		if _, kept := k.values[b.Values]; !kept {
			if err := k.update.PutValues(b.Values, values[i]); err != nil {
				return err
			}
			k.report.Values.Added++
		}
	}
	return nil
}

// block reads one block: "b", its file, its part, its children, its slots
// and its values; or "u", a shared block and what fills its slots.
func (k *keeper) block(data []byte) (record.Block, []byte, error) {
	damaged := errors.New("the project's code sent a damaged block")
	fields := bytes.SplitN(data, []byte{2}, 6)
	var b record.Block
	var slots []byte
	var values []byte
	switch {
	case len(fields) == 3 && string(fields[0]) == "u":
		number, err := strconv.ParseInt(string(fields[1]), 10, 64)
		if err != nil {
			return b, nil, damaged
		}
		id, known := k.numbers[number]
		if !known {
			return b, nil, errors.New("the project's code used a shared block before sending it")
		}
		b.Use, b.Shared, slots = true, id, fields[2]
	case len(fields) == 6 && string(fields[0]) == "b":
		var err1, err2, err3 error
		b.File, err1 = strconv.Atoi(string(fields[1]))
		b.Part, err2 = strconv.Atoi(string(fields[2]))
		b.Children, err3 = numbers(fields[3])
		if err1 != nil || err2 != nil || err3 != nil {
			return b, nil, damaged
		}
		if _, ok := k.site.Part(b.File, b.Part); !ok {
			return b, nil, fmt.Errorf("the project's code sent a block of template %d, part %d, which the site does not have", b.File, b.Part)
		}
		slots, values = fields[4], fields[5]
		b.Values = record.HashOf(values)
	default:
		return b, nil, damaged
	}
	// Slots are written as name=1,2;other=3.
	for len(slots) > 0 {
		var one []byte
		one, slots, _ = bytes.Cut(slots, []byte{';'})
		name, list, found := bytes.Cut(one, []byte{'='})
		fills, err := numbers(list)
		if !found || len(name) == 0 || err != nil {
			return b, nil, damaged
		}
		b.Slots = append(b.Slots, record.Slot{Name: string(name), Blocks: fills})
	}
	return b, values, nil
}

// numbers reads "1,2,3".
func numbers(list []byte) ([]int, error) {
	if len(list) == 0 {
		return nil, nil
	}
	parts := bytes.Split(list, []byte{','})
	out := make([]int, len(parts))
	for i, part := range parts {
		n, err := strconv.Atoi(string(part))
		if err != nil || n < 0 {
			return nil, errors.New("not a number")
		}
		out[i] = n
	}
	return out, nil
}

// check tells whether a record can be shown: its blocks make one tree, and
// the values of every block fit its template.
func (k *keeper) check(rec *record.Record, values [][]byte) error {
	if err := rec.Check(); err != nil {
		return err
	}
	for i := range rec.Blocks {
		if b := &rec.Blocks[i]; !b.Use {
			if err := render.Check(k.site, b, values[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

// count notes one page or shared block that is new or not what it was.
func (k *keeper) count(changes *Changes, name string, known bool) {
	if known {
		changes.Changed++
		if len(changes.ChangedNames) < k.names {
			changes.ChangedNames = append(changes.ChangedNames, name)
		}
		return
	}
	changes.Added++
	if len(changes.AddedNames) < k.names {
		changes.AddedNames = append(changes.AddedNames, name)
	}
}

// finish drops from the store what the run did not show: pages that are not
// there any more, shared blocks nothing uses, values no block has.
func (k *keeper) finish() error {
	for path := range k.pages {
		if _, seen := k.seenPages[path]; !seen {
			if err := k.update.DeletePage(path); err != nil {
				return err
			}
			k.report.Pages.Removed++
			if len(k.report.Pages.RemovedNames) < k.names {
				k.report.Pages.RemovedNames = append(k.report.Pages.RemovedNames, path)
			}
		}
	}
	for name, kept := range k.shared {
		if _, seen := k.seenShared[name]; !seen {
			if err := k.update.DeleteShared(kept.ID); err != nil {
				return err
			}
			k.report.Shared.Removed++
			if len(k.report.Shared.RemovedNames) < k.names {
				k.report.Shared.RemovedNames = append(k.report.Shared.RemovedNames, name)
			}
		}
	}
	for hash := range k.values {
		if _, live := k.live[hash]; !live {
			if err := k.update.DeleteValues(hash); err != nil {
				return err
			}
			k.report.Values.Removed++
		}
	}
	k.report.Pages.Total, k.report.Shared.Total, k.report.Values.Total = len(k.seenPages), len(k.seenShared), len(k.live)
	for _, names := range []*[]string{
		&k.report.Pages.AddedNames, &k.report.Pages.ChangedNames, &k.report.Pages.RemovedNames,
		&k.report.Shared.AddedNames, &k.report.Shared.ChangedNames, &k.report.Shared.RemovedNames,
	} {
		sort.Strings(*names)
	}
	return nil
}

// validPath tells whether text can be the address of a page.
func validPath(path string) bool {
	if !strings.HasPrefix(path, "/") || len(path) > 4096 {
		return false
	}
	for _, part := range strings.Split(path[1:], "/") {
		if part == "." || part == ".." || (part == "" && path != "/") {
			return false
		}
	}
	return !strings.ContainsAny(path, "\x00\\?#")
}
