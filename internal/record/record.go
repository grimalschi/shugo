// Package record is what the runtime keeps for one page or for one shared
// block: its blocks, each with its template, the values of its expressions
// and the blocks shown inside it. A record holds everything a page needs but
// the shared blocks it shows, so a page is read with one lookup and one more
// for every shared block on it.
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

// Block is one template being shown: a page, a layout, a component, or a
// snippet a template gave to a component.
type Block struct {
	// Use tells that the block stands for a shared block. Shared names it,
	// and only Snippets says more: the snippets this place gave it.
	Use    bool
	Shared int64

	File, Part int   // the template: a file and a part of it
	Values     Tape  // the values of its expressions, in the order of the template
	Children   []int // the components and snippets it shows, in the order it reaches them
	Snippets   []Snippet
}

// Tape is the values of a block one after another: each is its length in
// bytes, written as a varint, and then the bytes. A value may hold any byte.
type Tape []byte

// Append adds a value to the end of the tape.
func (t Tape) Append(value []byte) Tape {
	t = binary.AppendUvarint(t, uint64(len(value)))
	return append(t, value...)
}

// Next returns the first value of the tape and the tape after it; ok is
// false when the tape is empty or damaged.
func (t Tape) Next() (value []byte, rest Tape, ok bool) {
	size, n := binary.Uvarint(t)
	if n <= 0 || uint64(len(t)-n) < size {
		return nil, t, false
	}
	return t[n : n+int(size)], t[n+int(size):], true
}

// Strings returns the values of the tape as texts.
func (t Tape) Strings() []string {
	var out []string
	for len(t) > 0 {
		value, rest, ok := t.Next()
		if !ok {
			break
		}
		out = append(out, string(value))
		t = rest
	}
	return out
}

// Snippet is a snippet without parameters that the caller of a component
// gave it, computed where it was written: blocks of the same record.
type Snippet struct {
	Name   string
	Blocks []int
}

// Snippet returns the blocks of the snippet of the given name.
func (b *Block) Snippet(name string) []int {
	for i := range b.Snippets {
		if b.Snippets[i].Name == name {
			return b.Snippets[i].Blocks
		}
	}
	return nil
}

// Encode writes a record compactly. Numbers are written as varints.
func (r *Record) Encode() []byte {
	out := make([]byte, 0, 8+len(r.Blocks)*24)
	out = binary.AppendUvarint(out, uint64(r.Top))
	out = binary.AppendUvarint(out, uint64(len(r.Blocks)))
	for i := range r.Blocks {
		b := &r.Blocks[i]
		if b.Use {
			// A block of a template writes its file plus one, so zero is free.
			out = binary.AppendUvarint(out, 0)
			out = binary.AppendUvarint(out, uint64(b.Shared))
		} else {
			out = binary.AppendUvarint(out, uint64(b.File)+1)
			out = binary.AppendUvarint(out, uint64(b.Part))
			out = binary.AppendUvarint(out, uint64(len(b.Values)))
			out = append(out, b.Values...)
			out = appendList(out, b.Children)
		}
		out = binary.AppendUvarint(out, uint64(len(b.Snippets)))
		for _, snippet := range b.Snippets {
			out = binary.AppendUvarint(out, uint64(len(snippet.Name)))
			out = append(out, snippet.Name...)
			out = appendList(out, snippet.Blocks)
		}
	}
	return out
}

func appendList(out []byte, list []int) []byte {
	out = binary.AppendUvarint(out, uint64(len(list)))
	for _, n := range list {
		out = binary.AppendUvarint(out, uint64(n))
	}
	return out
}

var errDamaged = errors.New("record: damaged data")

// Decode reads what Encode wrote. It never trusts a length it reads. The
// values of the blocks stay in data: data must not change while the record
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
		if file := d.number(); file == 0 {
			b.Use, b.Shared = true, int64(d.number())
		} else {
			b.File, b.Part = file-1, d.number()
			if values := d.bytes(d.number()); len(values) > 0 {
				b.Values = values
			}
			b.Children = d.list(count)
		}
		if snippets := d.number(); snippets > 0 && snippets <= len(d.data) {
			b.Snippets = make([]Snippet, snippets)
			for j := range b.Snippets {
				b.Snippets[j].Name = string(d.bytes(d.number()))
				b.Snippets[j].Blocks = d.list(count)
			}
		} else if snippets != 0 {
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

func (d *decoder) bytes(n int) []byte {
	if d.failed || n > len(d.data) {
		d.failed = true
		return nil
	}
	out := d.data[:n]
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
		for _, child := range r.Blocks[at].Children {
			if err := visit(child); err != nil {
				return err
			}
		}
		for _, snippet := range r.Blocks[at].Snippets {
			for _, given := range snippet.Blocks {
				if err := visit(given); err != nil {
					return err
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
