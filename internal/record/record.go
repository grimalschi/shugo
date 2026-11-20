// Package record is what the runtime keeps for one page or for one shared
// block: its blocks, each naming its template, its values and the blocks
// shown inside it. A record holds no values itself: values are kept by their
// content, so that equal ones are kept once however many blocks have them.
package record

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
)

// Hash names the values of a block by their content.
type Hash [16]byte

// HashOf returns the name of values.
func HashOf(values []byte) Hash {
	sum := sha256.Sum256(values)
	return Hash(sum[:16])
}

// Record is the blocks of one page or of one shared block.
type Record struct {
	Top    int // the block that everything starts from
	Blocks []Block
}

// Block is one template being shown: a page, a layout, a component, or
// markup that a template put into a component.
type Block struct {
	// Use tells that the block stands for a shared block. Shared names it,
	// and only Slots says more: what this place put into its slots.
	Use    bool
	Shared int64

	File, Part int   // the template: a file and a part of it
	Values     Hash  // the values of its expressions, in the order of the template
	Children   []int // the components it shows, in the order it reaches them
	Slots      []Slot
}

// Slot is what was put into one slot of a component: blocks of the same record.
type Slot struct {
	Name   string
	Blocks []int
}

// Slot returns the blocks put into the slot of the given name.
func (b *Block) Slot(name string) []int {
	for i := range b.Slots {
		if b.Slots[i].Name == name {
			return b.Slots[i].Blocks
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
			out = append(out, b.Values[:]...)
			out = appendList(out, b.Children)
		}
		out = binary.AppendUvarint(out, uint64(len(b.Slots)))
		for _, slot := range b.Slots {
			out = binary.AppendUvarint(out, uint64(len(slot.Name)))
			out = append(out, slot.Name...)
			out = appendList(out, slot.Blocks)
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

// Decode reads what Encode wrote. It never trusts a length it reads.
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
			copy(b.Values[:], d.bytes(len(b.Values)))
			b.Children = d.list(count)
		}
		if slots := d.number(); slots > 0 && slots <= len(d.data) {
			b.Slots = make([]Slot, slots)
			for j := range b.Slots {
				b.Slots[j].Name = string(d.bytes(d.number()))
				b.Slots[j].Blocks = d.list(count)
			}
		} else if slots != 0 {
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
		for _, slot := range r.Blocks[at].Slots {
			for _, fill := range slot.Blocks {
				if err := visit(fill); err != nil {
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
