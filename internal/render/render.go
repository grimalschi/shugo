// Package render builds the HTML of a page from what is kept for it. The
// HTML itself is Svelte's, rendered when the page was computed; here the
// blocks of the page are put together: the shared blocks it shows, the
// snippets it gave them, its islands and its head. Nothing of the project
// runs here: no expression, no query. A record that does not fit together
// ends the page with an error instead of broken HTML.
package render

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/grimalschi/shugo/internal/record"
	"github.com/grimalschi/shugo/internal/skeleton"
)

// Source gives the renderer the shared blocks that records name.
type Source interface {
	// Shared returns the record of a shared block.
	Shared(id int64) (*record.Record, error)
}

// Limits that keep a damaged or hostile record from taking the server down.
const (
	maxDepth  = 256      // shared blocks inside shared blocks
	maxOutput = 64 << 20 // bytes of one page
)

// Page appends a page of a route to dst. A page of templates is HTML: the
// page in its layouts, inside the shell of the site. A module gives a file
// of its own type, written as it is.
func Page(dst []byte, site *skeleton.Site, route int, rec *record.Record, src Source) ([]byte, error) {
	return PageWithIslands(dst, site, route, rec, src, nil)
}

// Island holds the SSR output and inputs for hydrating one component.
type Island struct {
	ID      string            `json:"id"`
	URL     string            `json:"url"`
	CSS     string            `json:"css"`
	Props   json.RawMessage   `json:"props"`
	Body    string            `json:"body"`
	Request map[string]string `json:"request,omitempty"` // prop name to data source
}

// PageWithIslands is Page for a request that can change the input of an
// island: replace gives such an island its markup for the request before it
// is written. Export and ordinary requests use the stored output.
func PageWithIslands(dst []byte, site *skeleton.Site, route int, rec *record.Record, src Source, replace func(*Island) error) ([]byte, error) {
	if route < 0 || route >= len(site.Routes) {
		return dst, fmt.Errorf("the page is of route %d, which the site does not have", route)
	}
	r := renderer{site: site, src: src, out: dst, start: len(dst), replace: replace}
	if site.FileRoute(route) {
		if err := r.show(rec, rec.Top, nil); err != nil {
			return dst, err
		}
		return r.out, nil
	}
	r.out = append(r.out, site.Shell.Before...)
	r.headAt = len(r.out)
	r.out = append(r.out, site.Shell.Middle...)
	if err := r.show(rec, rec.Top, nil); err != nil {
		return dst, err
	}
	r.out = append(r.out, site.Mounts["body"]...)
	r.out = append(r.out, site.Shell.After...)
	if err := r.putHead(); err != nil {
		return dst, err
	}
	return r.out, nil
}

// Check tells whether the islands of a block can be shown: each is what an
// island needs, and one that is rendered again on request gives what Svelte
// gave. The blocks the block shows are not looked at.
func Check(site *skeleton.Site, b *record.Block) error {
	r := renderer{site: site, dry: true}
	for _, f := range []*record.Fragment{&b.Body, &b.Head} {
		for _, hole := range f.Holes {
			if hole.Kind == record.Island {
				if err := r.island(hole.Text); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

type renderer struct {
	site  *skeleton.Site
	src   Source
	out   []byte
	start int
	depth int
	// dry tells to check islands without showing anything.
	dry bool
	// head is what the blocks of the page give its <head>, in the order
	// they are shown, and title the last <title> among it. headAt is where
	// it goes in out: the place of the head in the shell.
	head   []given
	title  []byte
	headAt int
	// replace is set only for requests that can change an island's input.
	// scripts and styles are the files of islands the page links already.
	replace func(*Island) error
	scripts map[string]bool
	styles  map[string]bool
	// injected are the <style> elements of components compiled with
	// css="injected" that the page has already.
	injected map[string]bool
}

// given is HTML of a record that a block gives the head of the page, with
// the place the block is shown at.
type given struct {
	rec   *record.Record
	head  *record.Fragment
	frame *frame
}

// frame is a shared block being shown, as seen from inside it: the place
// that shows it, which gave it its snippets, and where that place is shown.
type frame struct {
	rec   *record.Record // the record of the place
	use   *record.Block  // the use that names the snippets
	outer *frame
}

// show shows the block of rec at the given place, inside the shared block of
// the frame from.
func (r *renderer) show(rec *record.Record, at int, from *frame) error {
	if at < 0 || at >= len(rec.Blocks) {
		return fmt.Errorf("the record has no block %d", at)
	}
	if r.depth++; r.depth > maxDepth {
		return errors.New("shared blocks are nested too deep")
	}
	defer func() { r.depth-- }()

	b := &rec.Blocks[at]
	if b.Use {
		// The shared block is shown from its own record; the snippets this
		// place gave it stay with the place.
		shared, err := r.src.Shared(b.Shared)
		if err != nil {
			return err
		}
		if shared.Top >= len(shared.Blocks) || shared.Blocks[shared.Top].Use {
			return errors.New("a shared block does not start with what a component rendered")
		}
		rec, b, from = shared, &shared.Blocks[shared.Top], &frame{rec, b, from}
	}
	// What the block gives the head waits until the page is shown.
	if !b.Head.Empty() {
		r.head = append(r.head, given{rec, &b.Head, from})
	}
	if len(b.Title) > 0 {
		r.title = b.Title
	}
	return r.fragment(rec, &b.Body, from)
}

// fragment writes HTML with holes, filling every hole.
func (r *renderer) fragment(rec *record.Record, f *record.Fragment, from *frame) error {
	for i, text := range f.Texts {
		r.out = append(r.out, text...)
		if len(r.out)-r.start > maxOutput {
			return errors.New("the page is too large")
		}
		if i == len(f.Holes) {
			break
		}
		switch hole := &f.Holes[i]; hole.Kind {
		case record.Child:
			if err := r.show(rec, hole.Block, from); err != nil {
				return err
			}
		case record.Given:
			// What the place gave was written in the template of the place.
			if from == nil {
				return fmt.Errorf("the snippet %s is shown where no place gave it", hole.Text)
			}
			for _, at := range from.use.Snippet(hole.Text) {
				if err := r.show(from.rec, at, from.outer); err != nil {
					return err
				}
			}
		case record.Island:
			if err := r.island(hole.Text); err != nil {
				return err
			}
		case record.Style:
			if !r.injected[string(hole.Text)] {
				if r.injected == nil {
					r.injected = make(map[string]bool)
				}
				r.injected[string(hole.Text)] = true
				r.out = append(r.out, hole.Text...)
			}
		default:
			return fmt.Errorf("a hole of an unknown kind %q", rune(hole.Kind))
		}
	}
	return nil
}

// putHead puts what the blocks of the page give its head into the place of
// the head in the shell, as Svelte does: in the order the blocks were shown,
// and of the titles the last one, after the rest. The framework's own markup
// for the head follows.
func (r *renderer) putHead() error {
	body := len(r.out)
	// A shared block inside <svelte:head> may give the head too: the list
	// grows while it is shown.
	for i := 0; i < len(r.head); i++ {
		if err := r.fragment(r.head[i].rec, r.head[i].head, r.head[i].frame); err != nil {
			return err
		}
	}
	r.out = append(r.out, r.title...)
	r.out = append(r.out, r.site.Mounts["head"]...)
	if len(r.out)-r.start > maxOutput {
		return errors.New("the page is too large")
	}
	// The head was written after the page: it moves to its place.
	head := append([]byte(nil), r.out[body:]...)
	copy(r.out[r.headAt+len(head):], r.out[r.headAt:body])
	copy(r.out[r.headAt:], head)
	return nil
}

func (r *renderer) island(value []byte) error {
	var island Island
	if err := json.Unmarshal(value, &island); err != nil {
		return fmt.Errorf("invalid island record: %w", err)
	}
	if island.ID == "" || !strings.HasPrefix(island.URL, "/_assets/") ||
		(island.CSS != "" && !strings.HasPrefix(island.CSS, "/_assets/")) ||
		len(island.Props) == 0 || island.Props[0] != '{' {
		return errors.New("invalid island entry or props")
	}
	if r.dry {
		// During evaluation, use the native Svelte render as the reference
		// for the build-produced Go program before publishing the record.
		if len(island.Request) > 0 {
			body, err := IslandBody(r.site, island.ID, island.Props)
			if err != nil {
				return err
			}
			if body != island.Body {
				at := 0
				for at < min(len(body), len(island.Body)) && body[at] == island.Body[at] {
					at++
				}
				return fmt.Errorf("island %s: Go markup differs from the Svelte server render at byte %d", island.ID, at)
			}
		}
		return nil
	}
	if r.replace != nil && len(island.Request) > 0 {
		if err := r.replace(&island); err != nil {
			return err
		}
	}
	// Marshal escapes HTML metacharacters in the JSON, including </script>.
	payload, err := json.Marshal(island.Props)
	if err != nil {
		return err
	}
	if len(r.out)-r.start+len(island.Body)+len(payload) > maxOutput {
		return errors.New("the page is too large")
	}
	if island.CSS != "" && !r.styles[island.CSS] {
		if r.styles == nil {
			r.styles = make(map[string]bool)
		}
		r.styles[island.CSS] = true
		r.out = append(r.out, `<link rel="stylesheet" href="`...)
		r.out = appendEscaped(r.out, []byte(island.CSS), true)
		r.out = append(r.out, `">`...)
	}
	r.out = append(r.out, `<div data-shugo-island="`...)
	r.out = appendEscaped(r.out, []byte(island.URL), true)
	r.out = append(r.out, `"><div data-shugo-target>`...)
	r.out = append(r.out, island.Body...)
	r.out = append(r.out, `</div><script type="application/json" data-shugo-props>`...)
	r.out = append(r.out, payload...)
	r.out = append(r.out, `</script></div>`...)
	if !r.scripts[island.URL] {
		if r.scripts == nil {
			r.scripts = make(map[string]bool)
		}
		r.scripts[island.URL] = true
		r.out = append(r.out, `<script type="module" src="`...)
		r.out = appendEscaped(r.out, []byte(island.URL), true)
		r.out = append(r.out, `"></script>`...)
	}
	return nil
}
