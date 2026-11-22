// Package render builds the HTML of a page from what is kept for it. It
// walks the operations of every block's template and puts the block's values
// into the markup. Nothing of the project runs here: no expression, no
// query. The values come from the stage that ran the project's code and are
// not trusted: each one is escaped for the place it goes to, and values that
// do not fit their template end the page with an error instead of broken HTML.
package render

import (
	"errors"
	"fmt"
	"strconv"
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
	maxDepth  = 256      // components inside components
	maxOutput = 64 << 20 // bytes of one page
)

// Page appends the HTML of a page to dst.
func Page(dst []byte, site *skeleton.Site, rec *record.Record, src Source) ([]byte, error) {
	r := renderer{site: site, src: src, out: dst, start: len(dst), headAt: -1}
	if err := r.show(rec, rec.Top, nil); err != nil {
		return dst, err
	}
	if err := r.putHead(); err != nil {
		return dst, err
	}
	return r.out, nil
}

// Check tells whether the values of one block fit its template: whether
// showing the block would read exactly its values and exactly as many
// components as it has. The blocks inside it are not looked at.
func Check(site *skeleton.Site, b *record.Block) error {
	ops, ok := site.Part(b.File, b.Part)
	if !ok {
		return fmt.Errorf("there is no template %d, part %d", b.File, b.Part)
	}
	r := renderer{site: site, dry: true}
	return r.run(ops, &cursor{block: b, rest: b.Values})
}

type renderer struct {
	site  *skeleton.Site
	src   Source
	out   []byte
	start int
	depth int
	// dry tells to read the values of one block without showing anything.
	dry bool
	// head is what the templates of the page give its <head>, in the order
	// they are shown, and title the last <title> among it. headAt is where
	// the <head> of the page ends in out; -1 until it is met.
	head   []given
	title  *given
	headAt int
}

// given is blocks of a record that a template gives the head of the page,
// with the template they are written in.
type given struct {
	rec    *record.Record
	blocks []int
	frame  *frame
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
	if c.block.Part == 0 {
		// What the template gives the head waits until the page is shown.
		if len(c.block.Head) > 0 {
			r.head = append(r.head, given{c.rec, c.block.Head, c.frame})
		}
		if n := len(c.block.Title); n > 0 {
			r.title = &given{c.rec, c.block.Title[n-1:], c.frame}
		}
	}
	c.rest = c.block.Values
	return r.run(ops, &c)
}

// putHead puts what the templates of the page give its head at the end of
// the <head>, as Svelte does: in the order the templates were shown, and of
// the titles the last one, after the rest. A page without a <head> has
// nowhere to put it.
func (r *renderer) putHead() error {
	if r.headAt < 0 {
		return nil
	}
	body := len(r.out)
	// A component inside <svelte:head> may give the head too: the list grows
	// while it is shown.
	for i := 0; i < len(r.head); i++ {
		for _, at := range r.head[i].blocks {
			if err := r.show(r.head[i].rec, at, r.head[i].frame); err != nil {
				return err
			}
		}
	}
	if t := r.title; t != nil {
		if err := r.show(t.rec, t.blocks[0], t.frame); err != nil {
			return err
		}
	}
	if len(r.out)-r.start > maxOutput {
		return errors.New("the page is too large")
	}
	// The head was written after the page: it moves to its place.
	head := append([]byte(nil), r.out[body:]...)
	copy(r.out[r.headAt+len(head):], r.out[r.headAt:body])
	copy(r.out[r.headAt:], head)
	return nil
}

// run goes through the operations of a block and checks that they took all
// of its values and all of its children.
func (r *renderer) run(ops []skeleton.Op, c *cursor) error {
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

func (r *renderer) ops(ops []skeleton.Op, c *cursor) error {
	for i := range ops {
		op := &ops[i]
		switch op.Kind {
		case skeleton.Markup:
			r.out = append(r.out, op.Text...)

		case skeleton.Text:
			value, err := c.next()
			if err != nil {
				return err
			}
			r.out = appendEscaped(r.out, value, false)

		case skeleton.HTML:
			value, err := c.next()
			if err != nil {
				return err
			}
			r.out = append(r.out, value...)

		case skeleton.Attr:
			value, err := c.next()
			if err != nil {
				return err
			}
			if err := r.attribute(op.Text, value); err != nil {
				return err
			}

		case skeleton.Spread:
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
				if err := r.attribute(string(name), value); err != nil {
					return err
				}
			}

		case skeleton.If:
			taken, err := c.number(len(op.Bodies))
			if err != nil {
				return err
			}
			if taken > 0 {
				if err := r.ops(op.Bodies[taken-1], c); err != nil {
					return err
				}
			}

		case skeleton.For:
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

		case skeleton.Call:
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

		case skeleton.Render:
			value, err := c.next()
			if err != nil {
				return err
			}
			if err := r.snippet(value, c); err != nil {
				return err
			}

		case skeleton.Element:
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
			if skeleton.IsVoid(string(tag)) {
				r.out = append(r.out, "/>"...)
				continue
			}
			r.out = append(r.out, '>')
			if err := r.ops(op.Bodies[1], c); err != nil {
				return err
			}
			r.out = append(append(append(r.out, "</"...), tag...), '>')

		case skeleton.Mount:
			if op.Text == "head" && r.headAt < 0 {
				r.headAt = len(r.out)
			}
			r.out = append(r.out, r.site.Mounts[op.Text]...)

		default:
			return fmt.Errorf("unknown operation %q", rune(op.Kind))
		}
	}
	return nil
}

// attribute writes one attribute. The value starts with a sign: "-" for no
// attribute, "+" for one that is there without a value, as disabled="",
// "=" before the text of the value.
func (r *renderer) attribute(name string, value []byte) error {
	if len(value) == 0 {
		return errors.New("an empty value stands where an attribute is expected")
	}
	switch value[0] {
	case '-':
	case '+':
		r.out = append(append(append(r.out, ' '), name...), `=""`...)
	case '=':
		r.out = append(append(append(r.out, ' '), name...), '=', '"')
		r.out = append(appendEscaped(r.out, value[1:], true), '"')
	default:
		return fmt.Errorf("the value %q stands where an attribute is expected", value)
	}
	return nil
}

// snippet shows what {@render} rendered. The value says what: "-" nothing;
// "c" and a number, the next child block, a snippet computed where it was
// rendered, which belongs to the component that many levels out; "f", a
// number, ":" and a name, what the caller of the component that many levels
// out gave under that name.
func (r *renderer) snippet(value []byte, c *cursor) error {
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
	for _, given := range f.block.Snippet(name) {
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
