// Package render builds the HTML of a page from what is kept for it. It
// walks the operations of every block's template and puts the block's values
// into the markup. Nothing of the project runs here: no expression, no
// query. The values come from the stage that ran the project's code and are
// not trusted: each one is escaped for the place it goes to, and values that
// do not fit their template end the page with an error instead of broken HTML.
package render

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/grimalschi/shugo/internal/program"
	"github.com/grimalschi/shugo/internal/record"
)

// Source gives the renderer the shared blocks that records name.
type Source interface {
	// Shared returns the record of a shared block.
	Shared(id int64) (*record.Record, error)
}

// Limits that keep a damaged or hostile record from taking the server down.
const (
	maxDepth  = 256      // components inside components
	maxOutput = 64 << 20 // bytes of one page
)

// Page appends the HTML of a page to dst.
func Page(dst []byte, site *program.Site, rec *record.Record, src Source) ([]byte, error) {
	return PageWithIslands(dst, site, rec, src, nil)
}

// Island holds the SSR output and inputs for hydrating one component.
type Island struct {
	ID      string            `json:"id"`
	URL     string            `json:"url"`
	CSS     string            `json:"css"`
	Props   json.RawMessage   `json:"props"`
	Body    string            `json:"body"`
	Request map[string]string `json:"request,omitempty"` // prop name to data source
	Search  bool              `json:"search,omitempty"`  // version 4 records
}

// PageWithIslands lets the Go server replace the precomputed SSR output of
// selected islands. Export and ordinary requests use the stored output.
func PageWithIslands(dst []byte, site *program.Site, rec *record.Record, src Source, replace func(*Island) error) ([]byte, error) {
	r := renderer{site: site, src: src, out: dst, start: len(dst), replace: replace}
	if err := r.show(rec, rec.Top, nil); err != nil {
		return dst, err
	}
	return r.out, nil
}

// Check tells whether the values of one block fit its template: whether
// showing the block would read exactly its values and exactly as many
// components as it has. The blocks inside it are not looked at.
func Check(site *program.Site, b *record.Block) error {
	ops, ok := site.Part(b.File, b.Part)
	if !ok {
		return fmt.Errorf("there is no template %d, part %d", b.File, b.Part)
	}
	r := renderer{site: site, dry: true}
	return r.run(ops, &cursor{block: b, rest: b.Values})
}

type renderer struct {
	site  *program.Site
	src   Source
	out   []byte
	start int
	depth int
	// dry tells to read the values of one block without showing anything.
	dry bool
	// replace is set only for requests that can change an island's input.
	replace func(*Island) error
	scripts map[string]bool
	styles  map[string]bool
}

// frame is a component being shown, as seen from inside its template: the
// snippets its caller gave it, and who the caller is.
type frame struct {
	rec   *record.Record // where the blocks of the snippets are
	block *record.Block  // the block whose Snippets name them
	outer *frame         // the component whose template they were written in
}

// cursor is the place reached in one block.
type cursor struct {
	rec   *record.Record
	block *record.Block
	rest  record.Tape // values not read yet
	child int         // how many of the block's children have been shown
	frame *frame
}

// show shows the block of rec at the given place. from is the component in
// whose template that place is written.
func (r *renderer) show(rec *record.Record, at int, from *frame) error {
	if at < 0 || at >= len(rec.Blocks) {
		return fmt.Errorf("the record has no block %d", at)
	}
	if r.depth++; r.depth > maxDepth {
		return errors.New("components are nested too deep")
	}
	defer func() { r.depth-- }()

	b := &rec.Blocks[at]
	c := cursor{rec: rec, block: b, frame: from}
	switch {
	case b.Use:
		// The shared block is shown from its own record; the snippets this
		// place gave it stay with the place.
		shared, err := r.src.Shared(b.Shared)
		if err != nil {
			return err
		}
		if shared.Top >= len(shared.Blocks) || shared.Blocks[shared.Top].Use {
			return errors.New("a shared block does not start with a template")
		}
		c = cursor{rec: shared, block: &shared.Blocks[shared.Top], frame: &frame{rec, b, from}}
	case b.Part == 0:
		c.frame = &frame{rec, b, from}
	}
	// Otherwise the block is a snippet: it belongs to the template it was
	// written in, and sees the snippets that template was given.

	ops, ok := r.site.Part(c.block.File, c.block.Part)
	if !ok {
		return fmt.Errorf("there is no template %d, part %d", c.block.File, c.block.Part)
	}
	c.rest = c.block.Values
	return r.run(ops, &c)
}

// run goes through the operations of a block and checks that they took all
// of its values and all of its children.
func (r *renderer) run(ops []program.Op, c *cursor) error {
	if err := r.ops(ops, c); err != nil {
		return fmt.Errorf("%s: %w", r.site.Files[c.block.File].Path, err)
	}
	switch {
	case len(c.rest) != 0:
		return fmt.Errorf("%s: the block has more values than its template reads", r.site.Files[c.block.File].Path)
	case c.child != len(c.block.Children):
		return fmt.Errorf("%s: the block has more components than its template shows", r.site.Files[c.block.File].Path)
	}
	return nil
}

var errShort = errors.New("the block has fewer values than its template reads")

// next takes the next value of the block.
func (c *cursor) next() ([]byte, error) {
	if len(c.rest) == 0 {
		return nil, errShort
	}
	value, rest, ok := c.rest.Next()
	if !ok {
		return nil, errors.New("the values of the block are damaged")
	}
	c.rest = rest
	return value, nil
}

// number takes the next value as a count that is not above limit.
func (c *cursor) number(limit int) (int, error) {
	value, err := c.next()
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(string(value))
	if err != nil || n < 0 || n > limit {
		return 0, fmt.Errorf("the value %q stands where a number up to %d is expected", value, limit)
	}
	return n, nil
}

func (r *renderer) ops(ops []program.Op, c *cursor) error {
	for i := range ops {
		op := &ops[i]
		switch op.Kind {
		case program.Markup:
			r.out = append(r.out, op.Text...)

		case program.Text:
			value, err := c.next()
			if err != nil {
				return err
			}
			r.out = appendEscaped(r.out, value, false)

		case program.HTML:
			value, err := c.next()
			if err != nil {
				return err
			}
			r.out = append(r.out, value...)

		case program.Island:
			value, err := c.next()
			if err != nil {
				return err
			}
			if err := r.island(value); err != nil {
				return err
			}

		case program.Attr, program.URL:
			value, err := c.next()
			if err != nil {
				return err
			}
			if err := r.attribute(op.Text, value, op.Kind == program.URL); err != nil {
				return err
			}

		case program.Spread:
			count, err := c.number(len(c.rest))
			if err != nil {
				return err
			}
			for range count {
				name, err := c.next()
				if err != nil {
					return err
				}
				value, err := c.next()
				if err != nil {
					return err
				}
				if !attributeName(name) {
					return fmt.Errorf("%q cannot be the name of an attribute", name)
				}
				if err := r.attribute(string(name), value, program.IsAddress(string(name))); err != nil {
					return err
				}
			}

		case program.If:
			taken, err := c.number(len(op.Bodies))
			if err != nil {
				return err
			}
			if taken > 0 {
				if err := r.ops(op.Bodies[taken-1], c); err != nil {
					return err
				}
			}

		case program.For:
			count, err := c.number(maxOutput)
			if err != nil {
				return err
			}
			for range count {
				if err := r.ops(op.Bodies[0], c); err != nil {
					return err
				}
				if len(r.out)-r.start > maxOutput {
					return errors.New("the page is too large")
				}
			}
			if count == 0 && len(op.Bodies) == 2 {
				if err := r.ops(op.Bodies[1], c); err != nil {
					return err
				}
			}

		case program.Call:
			if c.child >= len(c.block.Children) {
				return errors.New("the block has fewer components than its template shows")
			}
			c.child++
			if r.dry {
				continue
			}
			if err := r.show(c.rec, c.block.Children[c.child-1], c.frame); err != nil {
				return err
			}

		case program.Render:
			value, err := c.next()
			if err != nil {
				return err
			}
			if err := r.snippet(value, c, op.Text == program.Last); err != nil {
				return err
			}

		case program.Element:
			tag, err := c.next()
			if err != nil {
				return err
			}
			if len(tag) == 0 {
				continue
			}
			if !tagName(tag) {
				return fmt.Errorf("%q cannot be the name of a tag", tag)
			}
			r.out = append(append(r.out, '<'), tag...)
			if err := r.ops(op.Bodies[0], c); err != nil {
				return err
			}
			if program.IsVoid(string(tag)) {
				r.out = append(r.out, "/>"...)
				continue
			}
			r.out = append(r.out, '>')
			if err := r.ops(op.Bodies[1], c); err != nil {
				return err
			}
			r.out = append(append(append(r.out, "</"...), tag...), '>')

		case program.Mount:
			r.out = append(r.out, r.site.Mounts[op.Text]...)

		default:
			return fmt.Errorf("unknown operation %q", rune(op.Kind))
		}
	}
	return nil
}

func (r *renderer) island(value []byte) error {
	var island Island
	if err := json.Unmarshal(value, &island); err != nil {
		return fmt.Errorf("invalid island record: %w", err)
	}
	if island.ID == "" || !safeAddress([]byte(island.URL)) || !strings.HasPrefix(island.URL, "/_assets/") ||
		(island.CSS != "" && (!safeAddress([]byte(island.CSS)) || !strings.HasPrefix(island.CSS, "/_assets/"))) ||
		len(island.Props) == 0 || island.Props[0] != '{' {
		return errors.New("invalid island entry or props")
	}
	if r.dry {
		// During evaluation, use the native Svelte render as the reference
		// for the build-produced Go program before publishing the record.
		if island.Search || len(island.Request) > 0 {
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
	if r.replace != nil && (island.Search || len(island.Request) > 0) {
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

// attribute writes one attribute. The value starts with a sign: "-" for no
// attribute, "+" for one that is there without a value, as disabled="",
// "=" before the text of the value.
func (r *renderer) attribute(name string, value []byte, address bool) error {
	if len(value) == 0 {
		return errors.New("an empty value stands where an attribute is expected")
	}
	switch value[0] {
	case '-':
	case '+':
		r.out = append(append(append(r.out, ' '), name...), `=""`...)
	case '=':
		text := value[1:]
		if address && !safeAddress(text) {
			text = blockedAddress
		}
		r.out = append(append(append(r.out, ' '), name...), '=', '"')
		r.out = append(appendEscaped(r.out, text, true), '"')
	default:
		return fmt.Errorf("the value %q stands where an attribute is expected", value)
	}
	return nil
}

// snippet shows what {@render} rendered. The value says what: "-" nothing;
// "c" and a number, the next child block, a snippet computed where it was
// rendered, which belongs to the component that many levels out; "f", a
// number, ":" and a name, what the caller of the component that many levels
// out gave under that name; with last, only the last block of it.
func (r *renderer) snippet(value []byte, c *cursor, last bool) error {
	if len(value) == 0 {
		return errors.New("an empty value stands where a snippet is expected")
	}
	if value[0] == '-' && len(value) == 1 {
		return nil
	}
	kind := value[0]
	text := string(value[1:])
	depthText, name, _ := strings.Cut(text, ":")
	depth, err := strconv.Atoi(depthText)
	if err != nil || depth < 0 || depth > maxDepth || (kind != 'c' && kind != 'f') {
		return fmt.Errorf("the value %q stands where a snippet is expected", value)
	}
	if kind == 'c' {
		if c.child >= len(c.block.Children) {
			return errors.New("the block has fewer snippets than its template shows")
		}
		c.child++
		if r.dry {
			return nil
		}
		from, err := up(c.frame, depth)
		if err != nil {
			return err
		}
		return r.show(c.rec, c.block.Children[c.child-1], from)
	}
	if r.dry {
		return nil
	}
	f, err := up(c.frame, depth)
	if err != nil {
		return err
	}
	if f == nil {
		return fmt.Errorf("the snippet %s is shown where no component gave it", name)
	}
	// What the caller gave was written in the template of the caller.
	blocks := f.block.Snippet(name)
	if last && len(blocks) > 0 {
		blocks = blocks[len(blocks)-1:]
	}
	for _, given := range blocks {
		if err := r.show(f.rec, given, f.outer); err != nil {
			return err
		}
	}
	return nil
}

// up returns the frame that many levels out.
func up(f *frame, depth int) (*frame, error) {
	for range depth {
		if f == nil {
			return nil, errors.New("a snippet names a component further out than there are")
		}
		f = f.outer
	}
	return f, nil
}

// tagName tells whether a value can be the name of a tag.
func tagName(name []byte) bool {
	if len(name) == 0 || !(name[0] >= 'a' && name[0] <= 'z' || name[0] >= 'A' && name[0] <= 'Z') {
		return false
	}
	for _, c := range name {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == ':' || c == '.' || c == '_') {
			return false
		}
	}
	return true
}

// attributeName tells whether a name that came with the values can be
// written as the name of an attribute.
func attributeName(name []byte) bool {
	if len(name) == 0 {
		return false
	}
	for _, c := range name {
		if c <= ' ' || c == '"' || c == '\'' || c == '<' || c == '>' || c == '/' || c == '=' || c == 0x7f {
			return false
		}
	}
	return true
}
