package compile

import (
	"sort"
	"strings"
)

// shape is what a template reads of one value. It is one of four:
//
//	whole    the value as it is, with everything in it
//	fields   some members of it, each with a shape of its own
//	items    what stands at every place of a list
//	nothing  only whether the value is there
//
// A shape only grows: reading more of a value never takes away what was
// read before, and reading the value whole covers everything else.
//
// The shapes of the variables a template reads are its dependency tree. The
// JavaScript side walks them twice: once to make a fingerprint of exactly
// what the template reads, and once to copy that for the template to work on.
type shape struct {
	whole  bool
	fields map[string]*shape
	items  *shape
}

// all notes that the value is read whole.
func (s *shape) all() {
	s.whole, s.fields, s.items = true, nil, nil
}

// field returns the shape of a member. A member of a value that is read
// whole is read with it.
func (s *shape) field(name string) *shape {
	if s.whole {
		return s
	}
	if member, known := s.fields[name]; known {
		return member
	}
	if s.fields == nil {
		s.fields = map[string]*shape{}
	}
	member := &shape{}
	s.fields[name] = member
	return member
}

// item returns the shape of every item of a list.
func (s *shape) item() *shape {
	if s.whole {
		return s
	}
	if s.items == nil {
		s.items = &shape{}
	}
	return s.items
}

// settle makes a shape one of the four. A list whose items are read gives
// its length with them; a list that is asked for anything else beside its
// items is read whole.
func (s *shape) settle() {
	if s.items != nil && len(s.fields) > 0 {
		if _, onlyLength := s.fields["length"]; !onlyLength || len(s.fields) > 1 {
			s.all()
			return
		}
		s.fields = nil
	}
	for _, member := range s.fields {
		member.settle()
	}
	if s.items != nil {
		s.items.settle()
	}
}

// write writes a settled shape as JavaScript: 1 for whole, 0 for nothing,
// an object of the members, or a list of one for the items.
func (s *shape) write(out *strings.Builder) {
	switch {
	case s.whole:
		out.WriteString("1")
	case s.items != nil:
		out.WriteString("[")
		s.items.write(out)
		out.WriteString("]")
	case len(s.fields) > 0:
		names := make([]string, 0, len(s.fields))
		for name := range s.fields {
			names = append(names, name)
		}
		sort.Strings(names)
		out.WriteString("{")
		for i, name := range names {
			if i > 0 {
				out.WriteString(",")
			}
			out.WriteString(quote(name) + ":")
			s.fields[name].write(out)
		}
		out.WriteString("}")
	default:
		out.WriteString("0")
	}
}

func (s *shape) String() string {
	var out strings.Builder
	s.write(&out)
	return out.String()
}
