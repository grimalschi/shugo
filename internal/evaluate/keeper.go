package evaluate

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/grimalschi/shugo/internal/record"
	"github.com/grimalschi/shugo/internal/render"
	"github.com/grimalschi/shugo/internal/skeleton"
	"github.com/grimalschi/shugo/internal/store"
)

// keeper takes the records of the units a run computes and writes to the
// store those that differ from what it has. It also keeps the store's
// account of who shows which shared block and of who read what from the
// data, so that at the end it can drop what nothing shows any more and tell
// on which pages the HTML differs.
type keeper struct {
	site   *skeleton.Site
	update *store.Update
	report *Report
	limit  int // how many names a report lists

	// all tells that the run computes the whole site: every shared block a
	// record shows is computed too, one that fails ends the run at once, and
	// what the store has of every page and shared block is read at the start
	// into pages and shared, since all of it is compared anyway. Otherwise a
	// shared block the store has is left as it is, and the store is asked
	// one record at a time.
	all    bool
	pages  map[string]store.Kept
	shared map[string]store.Kept
	totals store.Totals

	// check tells that the run computes everything to see whether an
	// update that follows the journal would have been right: a record that
	// differs although nothing told so is noted in the report. reached are
	// those that the journal told to compute: pages and shared blocks as the
	// users they are, and the lists of pages of routes.
	check   bool
	reached map[string]struct{}

	// What the run has computed.
	seenPages  map[string]struct{}
	seenShared map[int64]struct{}
	// failed are shared blocks whose computation failed. That is an error
	// only if something still shows them when the run is over.
	failed map[int64]*ScriptError

	// The shared blocks the run has met, and those of them that are still
	// to be computed, in the order they were met.
	met   map[string]met
	named map[int64]string
	asked map[int64]struct{}
	next  []string
	// blank are shared blocks that got a number in this run and no record yet.
	blank map[int64]struct{}

	// Which records are new or not what they were.
	touched map[string]bool // pages
	fresh   map[int64]bool  // shared blocks
	// uses is which shared blocks a shared block shows, for those the run
	// has looked at.
	uses map[int64][]int64
	// loose are shared blocks that someone stopped showing.
	loose map[int64]struct{}

	// shapes are the shapes of what the project's code reads from its data
	// database; nil if it has none. plans are the questions it has asked
	// indexes for, as the store keeps them, that this run has met.
	shapes *shapes
	plans  map[string]bool
	reads  readings

	shows   edges
	was, is content
}

// met is a shared block as the run found it in the store, or made it.
type met struct {
	id    int64
	hash  record.Hash
	reads []byte
	known bool // the store had its record before the run
}

func newKeeper(site *skeleton.Site, u *store.Update, report *Report, limit int) (*keeper, error) {
	totals, err := u.Totals()
	if err != nil {
		return nil, err
	}
	return &keeper{
		plans: map[string]bool{},
		site:  site, update: u, report: report, limit: limit, totals: totals,
		seenPages: map[string]struct{}{}, seenShared: map[int64]struct{}{}, failed: map[int64]*ScriptError{},
		met: map[string]met{}, named: map[int64]string{}, asked: map[int64]struct{}{}, blank: map[int64]struct{}{},
		touched: map[string]bool{}, fresh: map[int64]bool{}, uses: map[int64][]int64{}, loose: map[int64]struct{}{},
		reads: readings{update: u},
		shows: edges{update: u},
		was:   content{},
		is:    content{},
	}, nil
}

// clear drops the records of another compiled project.
func (k *keeper) clear() error {
	k.totals = store.Totals{}
	return k.update.ClearRecords()
}

// computeAll makes the run one that computes the whole site: see all.
func (k *keeper) computeAll() (err error) {
	if k.pages, err = k.update.KeptPages(); err != nil {
		return err
	}
	k.shared, err = k.update.KeptShared()
	k.all = err == nil
	return err
}

func (k *keeper) page(path string) (kept store.Kept, known bool, err error) {
	if k.all {
		kept, known = k.pages[path]
		return kept, known, nil
	}
	return k.update.PageKept(path)
}

// meet finds a shared block by its name, giving it a number if the store
// has none for it.
func (k *keeper) meet(name string) (met, error) {
	if m, already := k.met[name]; already {
		return m, nil
	}
	kept, known := k.shared[name]
	if !k.all {
		var err error
		if kept, known, err = k.update.SharedNamed(name); err != nil {
			return met{}, err
		}
	}
	if !known {
		id, err := k.update.NewShared(name)
		if err != nil {
			return met{}, err
		}
		kept.ID = id
		k.blank[id] = struct{}{}
	}
	m := met{kept.ID, kept.Hash, kept.Reads, known}
	k.met[name], k.named[kept.ID] = m, name
	return m, nil
}

// take returns the shared blocks that records of this run show and that
// are to be computed next.
func (k *keeper) take() []string {
	names := k.next
	k.next = nil
	return names
}

// count tells how many units the run has an answer for, and has, whether a
// unit is among them.
func (k *keeper) count() int { return len(k.seenPages) + len(k.seenShared) + len(k.failed) }

func (k *keeper) has(u unit) bool {
	if !u.shared {
		_, seen := k.seenPages[u.name]
		return seen
	}
	m, known := k.met[u.name]
	_, seen := k.seenShared[m.id]
	_, failed := k.failed[m.id]
	return known && (seen || failed)
}

// fail takes the failure of a shared block that was asked for. A block the
// store has a record of may fail: after a change of data a block is computed
// that perhaps nothing shows any more, and that is known only at the end.
func (k *keeper) fail(name string, failure *ScriptError) error {
	m, err := k.meet(name)
	if err != nil {
		return err
	}
	if _, blank := k.blank[m.id]; blank || k.all {
		return failure
	}
	k.failed[m.id] = failure
	return nil
}

// put takes one record as the project's code sent it: a header and blocks
// split by \x01, their fields split by \x02. The header is p or s, the
// address or the name, the route of a page, the block to start from, and
// what was read from the data to compute the record.
func (k *keeper) put(data []byte) error {
	header, rest, _ := bytes.Cut(data, []byte{1})
	fields := bytes.Split(header, []byte{2})
	if len(fields) != 5 || len(fields[0]) != 1 {
		return errorf("a record with a damaged header")
	}
	route, err := strconv.Atoi(string(fields[2]))
	top, err2 := strconv.Atoi(string(fields[3]))
	if err != nil || err2 != nil {
		return errorf("a record with a damaged header")
	}
	u := unit{shared: fields[0][0] == 's', name: string(fields[1])}

	// Who the record is of comes first, so that a shared block that shows
	// itself is not asked for again.
	var self met
	var user string
	var kept store.Kept
	var known bool
	switch fields[0][0] {
	case 'p':
		if !validPath(u.name) {
			return errorf("a page with the address %q", u.name)
		}
		if route < 0 || route >= len(k.site.Routes) {
			return errorf("%s of route %d, which the site does not have", u, route)
		}
		if _, twice := k.seenPages[u.name]; twice {
			return errorf("%s twice", u)
		}
		k.seenPages[u.name] = struct{}{}
		k.report.Computed.Pages++
		user = store.PageUser(u.name)
		if kept, known, err = k.page(u.name); err != nil {
			return err
		}
	case 's':
		if self, err = k.meet(u.name); err != nil {
			return err
		}
		if _, twice := k.seenShared[self.id]; twice {
			return errorf("%s twice", u)
		}
		k.seenShared[self.id] = struct{}{}
		k.report.Computed.Shared++
		user = store.SharedUser(self.id)
		kept, known = store.Kept{ID: self.id, Hash: self.hash, Reads: self.reads}, self.known
	default:
		return errorf("a record of an unknown kind %q", fields[0][0])
	}

	rec := &record.Record{Top: top}
	var shows []int64
	for len(rest) > 0 {
		if len(rec.Blocks) == maxBlocks {
			return fmt.Errorf("%s has more than %d blocks", u, maxBlocks)
		}
		var one []byte
		one, rest, _ = bytes.Cut(rest, []byte{1})
		block, err := k.block(one)
		if err != nil {
			return fmt.Errorf("%s: %w", u, err)
		}
		if block.Use {
			shows = append(shows, block.Shared)
		}
		rec.Blocks = append(rec.Blocks, block)
	}
	k.report.Blocks += len(rec.Blocks)
	encoded := rec.Encode()
	hash := record.HashOf(encoded)
	if u.shared {
		k.uses[self.id] = shows
	}

	// What the record read is kept whether or not the record has changed:
	// the same page may come of other rows.
	reads, read, err := k.read(user, kept.Reads, fields[4])
	if err != nil {
		return fmt.Errorf("%s: %w", u, err)
	}
	// A record that is what it was needs nothing more: what it shows is in
	// the store already, and it was checked when it got there.
	if known && kept.Hash == hash {
		switch {
		case u.shared && read:
			return k.update.SetSharedReads(self.id, reads)
		case !u.shared && (read || kept.Route != route):
			return k.update.SetPageReads(u.name, route, reads)
		}
		return nil
	}
	if known && k.check && !k.foreseen(user) {
		k.report.Unseen = append(k.report.Unseen, u.String())
	}

	var old []byte
	if u.shared {
		if top < len(rec.Blocks) && rec.Blocks[top].Use {
			return fmt.Errorf("%s does not start with a template", u)
		}
		if known {
			if old, _, err = k.update.Shared(self.id); err != nil {
				return err
			}
		}
	} else if known {
		if old, _, _, err = k.update.Page(u.name); err != nil {
			return err
		}
	}
	if err := k.fits(rec); err != nil {
		return fmt.Errorf("%s: %w", u, err)
	}
	if err := k.change(user, old, rec); err != nil {
		return fmt.Errorf("%s: %w", u, err)
	}
	if u.shared {
		delete(k.blank, self.id)
		k.fresh[self.id] = true
		k.noted(&k.report.Shared, u.name, known)
		return k.update.PutShared(self.id, hash, encoded, reads)
	}
	k.touched[u.name] = true
	k.noted(&k.report.Pages, u.name, known)
	return k.update.PutPage(u.name, hash, encoded, route, reads)
}

// foreseen tells whether a plain update would have computed a page or a
// shared block: the journal reached it.
func (k *keeper) foreseen(user string) bool {
	if k.reached == nil {
		return true
	}
	_, reached := k.reached[user]
	return reached
}

// noted counts one page or shared block that is new or not what it was.
func (k *keeper) noted(changes *Changes, name string, known bool) {
	if known {
		changes.note(&changes.Changed, &changes.ChangedNames, name, k.limit)
	} else {
		changes.note(&changes.Added, &changes.AddedNames, name, k.limit)
	}
}

// block reads one block: "b", its file, its part, its children, the blocks
// of its <svelte:head> and of its <title>, the snippets its caller gave it
// and its values; or "u", the name of a shared block and the snippets this
// place gave it.
func (k *keeper) block(data []byte) (record.Block, error) {
	fields := bytes.SplitN(data, []byte{2}, 8)
	var b record.Block
	var snippets []byte
	switch {
	case len(fields) == 3 && string(fields[0]) == "u":
		name := string(fields[1])
		shown, err := k.meet(name)
		if err != nil {
			return b, err
		}
		// The run computes a shared block it has met at most once, and one
		// the store has no record of in any case.
		_, seen := k.seenShared[shown.id]
		if _, waits := k.asked[shown.id]; !seen && !waits && (!shown.known || k.all) {
			k.asked[shown.id] = struct{}{}
			k.next = append(k.next, name)
		}
		b.Use, b.Shared, snippets = true, shown.id, fields[2]
	case len(fields) == 8 && string(fields[0]) == "b":
		var err1, err2, err3, err4, err5 error
		b.File, err1 = strconv.Atoi(string(fields[1]))
		b.Part, err2 = strconv.Atoi(string(fields[2]))
		b.Children, err3 = numbers(fields[3])
		b.Head, err4 = numbers(fields[4])
		b.Title, err5 = numbers(fields[5])
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil {
			return b, errDamaged
		}
		if _, ok := k.site.Part(b.File, b.Part); !ok {
			return b, errorf("a block of template %d, part %d, which the site does not have", b.File, b.Part)
		}
		values, err := tape(fields[7])
		if err != nil {
			return b, err
		}
		snippets, b.Values = fields[6], values
	default:
		return b, errDamaged
	}
	// Snippets are written as name=1,2;other=3.
	for len(snippets) > 0 {
		var one []byte
		one, snippets, _ = bytes.Cut(snippets, []byte{';'})
		name, list, found := bytes.Cut(one, []byte{'='})
		blocks, err := numbers(list)
		if !found || len(name) == 0 || err != nil {
			return b, errDamaged
		}
		b.Snippets = append(b.Snippets, record.Snippet{Name: string(name), Blocks: blocks})
	}
	return b, nil
}

// tape reads the values of a block as the project's code sends them: each
// after a byte 0. The bytes 0 to 4 inside a value come escaped, as 4 and the
// byte plus 0x40, for the bytes 0 to 3 split the message.
func tape(data []byte) (record.Tape, error) {
	if len(data) == 0 {
		return nil, nil
	}
	if data[0] != 0 {
		return nil, errDamaged
	}
	var out record.Tape
	var value []byte
	for _, one := range bytes.Split(data[1:], []byte{0}) {
		value = value[:0]
		for i := 0; i < len(one); i++ {
			if one[i] != 4 {
				value = append(value, one[i])
				continue
			}
			if i++; i == len(one) || one[i] < 0x40 || one[i] > 0x44 {
				return nil, errDamaged
			}
			value = append(value, one[i]-0x40)
		}
		out = out.Append(value)
	}
	return out, nil
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

// fits tells whether a record can be shown: its blocks make one tree, and
// the values of every block fit its template.
func (k *keeper) fits(rec *record.Record) error {
	if err := rec.Check(); err != nil {
		return err
	}
	for i := range rec.Blocks {
		if b := &rec.Blocks[i]; !b.Use {
			if err := render.Check(k.site, b); err != nil {
				return err
			}
		}
	}
	return nil
}

// content is the shared blocks a record shows.
type content map[int64]struct{}

func (c content) of(rec *record.Record) {
	clear(c)
	if rec == nil {
		return
	}
	for i := range rec.Blocks {
		if b := &rec.Blocks[i]; b.Use {
			c[b.Shared] = struct{}{}
		}
	}
}

// change notes what a page or a shared block shows now in place of what it
// showed. old is its record as the store has it, now the one that replaces
// it; either may be missing.
func (k *keeper) change(user string, old []byte, now *record.Record) error {
	var before *record.Record
	if len(old) > 0 {
		var err error
		if before, err = record.Decode(old); err != nil {
			return fmt.Errorf("the record the store has: %w", err)
		}
	}
	k.was.of(before)
	k.is.of(now)
	for id := range k.is {
		if _, kept := k.was[id]; !kept {
			k.shows.list = append(k.shows.list, edge{id, user, true})
		}
	}
	for id := range k.was {
		if _, kept := k.is[id]; !kept {
			k.shows.list = append(k.shows.list, edge{id, user, false})
			k.loose[id] = struct{}{}
		}
	}
	return nil
}

// remove drops a page the project no longer has.
func (k *keeper) remove(path string) error {
	kept, found, err := k.page(path)
	if err != nil || !found {
		return err
	}
	old, _, _, err := k.update.Page(path)
	if err != nil {
		return err
	}
	if err := k.change(store.PageUser(path), old, nil); err != nil {
		return fmt.Errorf("the page %s: %w", path, err)
	}
	if _, _, err := k.read(store.PageUser(path), kept.Reads, nil); err != nil {
		return err
	}
	k.report.Pages.note(&k.report.Pages.Removed, &k.report.Pages.RemovedNames, path, k.limit)
	return k.update.DeletePage(path)
}

// finish settles the accounts of the run: it drops the shared blocks that
// nothing shows any more, and tells what the update came to.
func (k *keeper) finish() error {
	if err := k.loops(); err != nil {
		return err
	}
	if err := k.sweep(); err != nil {
		return err
	}
	// A shared block that failed and is still there is shown by someone.
	for id, failure := range k.failed {
		if _, still, err := k.update.Shared(id); err != nil {
			return err
		} else if still {
			return failure
		}
	}
	for id := range k.blank {
		return errorf("no record of the shared block %s, which a record shows", k.named[id])
	}
	if err := k.reads.settle(); err != nil {
		return err
	}
	r := k.report
	k.totals.Pages += r.Pages.Added - r.Pages.Removed
	k.totals.Shared += r.Shared.Added - r.Shared.Removed
	r.Pages.Total, r.Shared.Total = k.totals.Pages, k.totals.Shared
	if err := k.update.SetTotals(k.totals); err != nil {
		return err
	}
	if err := k.rendered(); err != nil {
		return err
	}
	for _, names := range []*[]string{
		&r.RenderNames,
		&r.Pages.AddedNames, &r.Pages.ChangedNames, &r.Pages.RemovedNames,
		&r.Shared.AddedNames, &r.Shared.ChangedNames, &r.Shared.RemovedNames,
	} {
		sort.Strings(*names)
	}
	return nil
}

// shown returns the shared blocks that a shared block shows.
func (k *keeper) shown(id int64) ([]int64, error) {
	if uses, looked := k.uses[id]; looked {
		return uses, nil
	}
	data, _, err := k.update.Shared(id)
	if err != nil {
		return nil, err
	}
	var uses []int64
	if len(data) > 0 {
		rec, err := record.Decode(data)
		if err != nil {
			return nil, fmt.Errorf("the shared block %s: %w", k.name(id), err)
		}
		for i := range rec.Blocks {
			if rec.Blocks[i].Use {
				uses = append(uses, rec.Blocks[i].Shared)
			}
		}
	}
	k.uses[id] = uses
	return uses, nil
}

func (k *keeper) name(id int64) string {
	if name, met := k.named[id]; met {
		return name
	}
	name, err := k.update.SharedName(id)
	if err != nil {
		return "number " + strconv.FormatInt(id, 10)
	}
	k.named[id] = name
	return name
}

// loops makes sure that no shared block shows itself, directly or through
// others: showing it would never end. Only a record that is new or has
// changed can close such a circle.
func (k *keeper) loops() error {
	const open, closed = 1, 2
	state := map[int64]int8{}
	var trail []int64
	var visit func(id int64) error
	visit = func(id int64) error {
		switch state[id] {
		case closed:
			return nil
		case open:
			var way []string
			for at := len(trail) - 1; at >= 0; at-- {
				way = append([]string{k.name(trail[at])}, way...)
				if trail[at] == id {
					break
				}
			}
			if len(way) == 1 {
				return fmt.Errorf("%s shows itself again with the same parameters, which would never end", way[0])
			}
			return fmt.Errorf("%s shows itself again with the same parameters through %s, which would never end", way[0], strings.Join(way[1:], ", "))
		}
		state[id] = open
		trail = append(trail, id)
		uses, err := k.shown(id)
		if err != nil {
			return err
		}
		for _, next := range uses {
			if err := visit(next); err != nil {
				return err
			}
		}
		trail = trail[:len(trail)-1]
		state[id] = closed
		return nil
	}
	for id := range k.fresh {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

// sweep drops the shared blocks that nothing shows any more. What such a
// block showed may go with it.
func (k *keeper) sweep() error {
	for {
		// Who shows what is told to the store first: it is asked right below.
		if err := k.shows.settle(); err != nil {
			return err
		}
		if len(k.loose) == 0 {
			return nil
		}
		ids := make([]int64, 0, len(k.loose))
		for id := range k.loose {
			ids = append(ids, id)
		}
		clear(k.loose)
		for _, id := range ids {
			if shown, err := k.update.Shown(id); err != nil {
				return err
			} else if shown {
				continue
			}
			old, found, err := k.update.Shared(id)
			if err != nil {
				return err
			} else if !found {
				continue
			}
			name := k.name(id)
			if err := k.change(store.SharedUser(id), old, nil); err != nil {
				return fmt.Errorf("the shared block %s: %w", name, err)
			}
			reads, err := k.update.SharedReads(id)
			if err != nil {
				return err
			}
			if _, _, err := k.read(store.SharedUser(id), reads, nil); err != nil {
				return err
			}
			if err := k.update.DeleteShared(id); err != nil {
				return err
			}
			delete(k.fresh, id)
			k.report.Shared.note(&k.report.Shared.Removed, &k.report.Shared.RemovedNames, name, k.limit)
		}
	}
}

// rendered finds the pages whose HTML differs from what it was: those whose
// record is new or has changed, and those that show a shared block whose
// record has, directly or through other shared blocks.
func (k *keeper) rendered() error {
	pages := make(map[string]struct{}, len(k.touched))
	for path := range k.touched {
		pages[path] = struct{}{}
	}
	// When every page has changed itself, there is nobody else to find.
	if len(pages) < k.totals.Pages {
		seen := make(map[int64]struct{}, len(k.fresh))
		queue := make([]int64, 0, len(k.fresh))
		for id := range k.fresh {
			seen[id] = struct{}{}
			queue = append(queue, id)
		}
		for len(queue) > 0 {
			id := queue[len(queue)-1]
			queue = queue[:len(queue)-1]
			err := k.update.Users(id, func(user string) {
				if shared, is := store.UserShared(user); !is {
					pages[user] = struct{}{}
				} else if _, already := seen[shared]; !already {
					seen[shared] = struct{}{}
					queue = append(queue, shared)
				}
			})
			if err != nil {
				return err
			}
		}
	}
	k.report.Render = len(pages)
	for path := range pages {
		k.report.RenderNames = append(k.report.RenderNames, path)
	}
	sort.Strings(k.report.RenderNames)
	if len(k.report.RenderNames) > k.limit {
		k.report.RenderNames = k.report.RenderNames[:k.limit]
	}
	return nil
}

// edges are the changes of who shows which shared block that a run has
// found and not told the store yet. Told in order, a hundred thousand pages
// that show the same layout are written several times faster.
type edges struct {
	update *store.Update
	list   []edge
}

// edge says that a user shows a shared block, or no longer does.
type edge struct {
	shared int64
	user   string
	shows  bool
}

// settle tells the store. A page may be dropped and come back within one
// run, so of what was said about one user and one block the last word
// counts: the order of the changes is kept where they meet.
func (e *edges) settle() error {
	sort.SliceStable(e.list, func(a, b int) bool {
		if e.list[a].shared != e.list[b].shared {
			return e.list[a].shared < e.list[b].shared
		}
		return e.list[a].user < e.list[b].user
	})
	for _, one := range e.list {
		tell := e.update.Unshow
		if one.shows {
			tell = e.update.Show
		}
		if err := tell(one.shared, one.user); err != nil {
			return err
		}
	}
	e.list = e.list[:0]
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
