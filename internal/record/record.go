// Package record is what the runtime keeps for one page or for one shared
// block: its HTML, as Svelte rendered it, cut into blocks. A record holds
// everything a page needs but the shared blocks it shows, so a page is read
// with one lookup and one more for every shared block on it.
package record

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
)

// Hash names a record by its content: two records that hash alike are the
// same, and an update writes only a record whose hash has changed.
type Hash [16]byte

// HashOf returns the hash of an encoded record.
func HashOf(data []byte) Hash {
	sum := sha256.Sum256(data)
	return Hash(sum[:16])
}

// Record is the blocks of one page or of one shared block.
type Record struct {
	Top    int // the block that everything starts from
	Blocks []Block
}

// Block is a part of a page: what one component rendered, or a snippet a
// template gave to a shared block; or a use, which stands for a shared block.
type Block struct {
	// Use tells that the block stands for a shared block. Shared names it,
	// and Snippets are what this place gave it: snippets without
	// parameters, rendered where they were written, blocks of this record.
	Use      bool
	Shared   int64
	Snippets []Snippet

	// Body is the HTML of the block. Head is what it gives the <head> of the
	// page in <svelte:head>, and Title the <title> among it that it rendered
	// last, as HTML. A snippet given to a shared block has them too: it is
	// rendered apart, and they go where the shared block shows it.
	Body, Head Fragment
	Title      []byte
}

// Fragment is HTML with holes in it: Texts[0], Holes[0], Texts[1], … The
// texts are one more than the holes.
type Fragment struct {
	Texts [][]byte
	Holes []Hole
}

// Hole is a place in HTML that something else fills when the page is shown.
type Hole struct {
	Kind  HoleKind
	Block int    // Child: the block of the record that stands here
	Text  []byte // Given: the name of the snippet; Island: what the island is, in JSON
}

// HoleKind says what fills a hole.
type HoleKind byte

const (
	Child  HoleKind = 'c' // a block of the same record: a use of a shared block
	Given  HoleKind = 'g' // the snippet of that name that the place a shared block is shown at gave it
	Island HoleKind = 'i' // a Svelte island
)

// Empty tells whether a fragment has nothing to show.
func (f *Fragment) Empty() bool {
	return len(f.Holes) == 0 && (len(f.Texts) == 0 || len(f.Texts[0]) == 0)
}

// Text returns a fragment without holes.
func Text(html []byte) Fragment {
	return Fragment{Texts: [][]byte{html}}
}

// Snippet is a snippet without parameters that a template gave a shared
// block, rendered where it was written: blocks of the same record.
type Snippet struct {
	Name   string
	Blocks []int
}

// Snippet returns the blocks of the snippet of the given name.
func (b *Block) Snippet(name []byte) []int {
	for i := range b.Snippets {
		if b.Snippets[i].Name == string(name) {
			return b.Snippets[i].Blocks
		}
	}
	return nil
}

// Encode writes a record compactly. Numbers are written as varints.
func (r *Record) Encode() []byte {
	out := make([]byte, 0, 64)
	out = binary.AppendUvarint(out, uint64(r.Top))
	out = binary.AppendUvarint(out, uint64(len(r.Blocks)))
	for i := range r.Blocks {
		b := &r.Blocks[i]
		if b.Use {
			out = append(out, 'u')
			out = binary.AppendUvarint(out, uint64(b.Shared))
			out = binary.AppendUvarint(out, uint64(len(b.Snippets)))
			for _, snippet := range b.Snippets {
				out = appendBytes(out, []byte(snippet.Name))
				out = binary.AppendUvarint(out, uint64(len(snippet.Blocks)))
				for _, n := range snippet.Blocks {
					out = binary.AppendUvarint(out, uint64(n))
				}
			}
			continue
		}
		out = append(out, 'b')
		out = b.Body.append(out)
		out = b.Head.append(out)
		out = appendBytes(out, b.Title)
	}
	return out
}

func (f *Fragment) append(out []byte) []byte {
	out = binary.AppendUvarint(out, uint64(len(f.Holes)))
	for i, hole := range f.Holes {
		out = appendBytes(out, f.Texts[i])
		out = append(out, byte(hole.Kind))
		if hole.Kind == Child {
			out = binary.AppendUvarint(out, uint64(hole.Block))
		} else {
			out = appendBytes(out, hole.Text)
		}
	}
	if len(f.Texts) > len(f.Holes) {
		return appendBytes(out, f.Texts[len(f.Holes)])
	}
	return appendBytes(out, nil)
}

func appendBytes(out, data []byte) []byte {
	out = binary.AppendUvarint(out, uint64(len(data)))
	return append(out, data...)
}

var errDamaged = errors.New("record: damaged data")

// Decode reads what Encode wrote. It never trusts a length it reads. The
// texts of the blocks stay in data: data must not change while the record
// is in use.
func Decode(data []byte) (*Record, error) {
	d := decoder{data: data}
	r := &Record{Top: d.number()}
	count := d.number()
	if d.failed || count > len(d.data) {
		return nil, errDamaged
	}
	r.Blocks = make([]Block, count)
	for i := range r.Blocks {
		b := &r.Blocks[i]
		switch d.byte() {
		case 'u':
			b.Use, b.Shared = true, int64(d.number())
			if n := d.number(); n > 0 && n <= len(d.data) {
				b.Snippets = make([]Snippet, n)
				for j := range b.Snippets {
					b.Snippets[j].Name = string(d.bytes())
					b.Snippets[j].Blocks = d.list(count)
				}
			} else if n != 0 {
				d.failed = true
			}
		case 'b':
			b.Body = d.fragment(count)
			b.Head = d.fragment(count)
			b.Title = d.bytes()
		default:
			d.failed = true
		}
		if d.failed {
			return nil, errDamaged
		}
	}
	if len(d.data) != 0 || r.Top >= count {
		return nil, errDamaged
	}
	return r, nil
}

type decoder struct {
	data   []byte
	failed bool
}

func (d *decoder) number() int {
	n, size := binary.Uvarint(d.data)
	if size <= 0 || n > 1<<40 {
		d.failed = true
		return 0
	}
	d.data = d.data[size:]
	return int(n)
}

func (d *decoder) byte() byte {
	if d.failed || len(d.data) == 0 {
		d.failed = true
		return 0
	}
	c := d.data[0]
	d.data = d.data[1:]
	return c
}

// bytes reads a length and that many bytes; nil for none.
func (d *decoder) bytes() []byte {
	n := d.number()
	if d.failed || n > len(d.data) {
		d.failed = true
		return nil
	}
	if n == 0 {
		return nil
	}
	out := d.data[:n:n]
	d.data = d.data[n:]
	return out
}

// list reads numbers of blocks, each below limit.
func (d *decoder) list(limit int) []int {
	n := d.number()
	if d.failed || n > len(d.data) {
		d.failed = true
		return nil
	}
	if n == 0 {
		return nil
	}
	out := make([]int, n)
	for i := range out {
		if out[i] = d.number(); out[i] >= limit {
			d.failed = true
		}
	}
	return out
}

// fragment reads a fragment whose holes name blocks below limit.
func (d *decoder) fragment(limit int) Fragment {
	holes := d.number()
	if d.failed || holes > len(d.data) {
		d.failed = true
		return Fragment{}
	}
	f := Fragment{Texts: make([][]byte, holes+1)}
	if holes > 0 {
		f.Holes = make([]Hole, holes)
	}
	for i := range f.Holes {
		f.Texts[i] = d.bytes()
		h := &f.Holes[i]
		switch h.Kind = HoleKind(d.byte()); h.Kind {
		case Child:
			if h.Block = d.number(); h.Block >= limit {
				d.failed = true
			}
		case Given, Island:
			h.Text = d.bytes()
		default:
			d.failed = true
		}
	}
	f.Texts[holes] = d.bytes()
	return f
}

// Check tells whether the blocks of a record make one tree under Top: every
// block is shown at exactly one place. A record that came from outside is
// checked before it is kept, so that showing it can never go in circles.
func (r *Record) Check() error {
	if r.Top < 0 || r.Top >= len(r.Blocks) {
		return fmt.Errorf("record: no block %d to start from", r.Top)
	}
	seen := make([]bool, len(r.Blocks))
	left := len(r.Blocks)
	var visit func(at int) error
	visit = func(at int) error {
		if at < 0 || at >= len(r.Blocks) {
			return fmt.Errorf("record: no block %d", at)
		}
		if seen[at] {
			return fmt.Errorf("record: block %d is shown at two places", at)
		}
		seen[at] = true
		left--
		b := &r.Blocks[at]
		for _, snippet := range b.Snippets {
			for _, given := range snippet.Blocks {
				if err := visit(given); err != nil {
					return err
				}
			}
		}
		for _, f := range []*Fragment{&b.Body, &b.Head} {
			if len(f.Texts) != len(f.Holes)+1 && !(len(f.Texts) == 0 && len(f.Holes) == 0) {
				return fmt.Errorf("record: block %d has %d texts around %d holes", at, len(f.Texts), len(f.Holes))
			}
			for _, hole := range f.Holes {
				if hole.Kind == Child {
					if err := visit(hole.Block); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
	if err := visit(r.Top); err != nil {
		return err
	}
	if left != 0 {
		return fmt.Errorf("record: %d blocks are shown nowhere", left)
	}
	return nil
}
