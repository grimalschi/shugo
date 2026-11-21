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
	r := renderer{site: site, src: src, out: dst, start: len(dst)}
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
}

// frame is a component being shown, as seen from inside its template: what
// was put into its slots, and who put it there.
type frame struct {
	rec   *record.Record // where the blocks put into the slots are
	block *record.Block  // the block whose Slots name them
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
		// The shared block is shown from its own record; what fills its slots
		// stays with the place that uses it.
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
	// Otherwise the block is markup put into a component: it belongs to the
	// template it was written in, and sees that template's slots.

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
			r.out = appendEscaped(r.out, value)

		case program.HTML:
			value, err := c.next()
			if err != nil {
				return err
			}
			r.out = append(r.out, value...)

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

		case program.Slot:
			filled, err := c.number(1)
			if err != nil {
				return err
			}
			if filled == 0 {
				if err := r.ops(op.Bodies[0], c); err != nil {
					return err
				}
				continue
			}
			if r.dry || c.frame == nil {
				continue
			}
			// What fills the slot was written in the template of the caller.
			for _, fill := range c.frame.block.Slot(op.Text) {
				if err := r.show(c.frame.rec, fill, c.frame.outer); err != nil {
					return err
				}
			}

		case program.Mount:
			r.out = append(r.out, r.site.Mounts[op.Text]...)

		default:
			return fmt.Errorf("unknown operation %q", rune(op.Kind))
		}
	}
	return nil
}

// attribute writes one attribute. The value starts with a sign: "-" for no
// attribute, "+" for the name alone, "=" before the text of the value.
func (r *renderer) attribute(name string, value []byte, address bool) error {
	if len(value) == 0 {
		return errors.New("an empty value stands where an attribute is expected")
	}
	switch value[0] {
	case '-':
	case '+':
		r.out = append(append(r.out, ' '), name...)
	case '=':
		text := value[1:]
		if address && !safeAddress(text) {
			text = blockedAddress
		}
		r.out = append(append(append(r.out, ' '), name...), '=', '"')
		r.out = append(appendEscaped(r.out, text), '"')
	default:
		return fmt.Errorf("the value %q stands where an attribute is expected", value)
	}
	return nil
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
