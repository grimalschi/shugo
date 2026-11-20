package record

import (
	"reflect"
	"testing"
)

func sample() *Record {
	return &Record{Top: 3, Blocks: []Block{
		{File: 5, Part: 0, Values: HashOf([]byte("page")), Children: []int{1}},
		{File: 0, Part: 0, Values: HashOf([]byte("card")), Slots: []Slot{{"default", []int{2}}}},
		{File: 5, Part: 2, Values: HashOf(nil)},
		{Use: true, Shared: 7, Slots: []Slot{{"default", []int{0}}, {"head", []int{4}}}},
		{File: 5, Part: 1, Values: HashOf([]byte("title"))},
	}}
}

func TestRoundTrip(t *testing.T) {
	r := sample()
	back, err := Decode(r.Encode())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back, r) {
		t.Errorf("\n got %+v\nwant %+v", back, r)
	}
	if err := back.Check(); err != nil {
		t.Error(err)
	}
	if got := back.Blocks[3].Slot("head"); !reflect.DeepEqual(got, []int{4}) {
		t.Errorf("slot head: %v", got)
	}
}

// Whatever the bytes are, decoding them gives a record or an error.
func TestDecodeDamaged(t *testing.T) {
	data := sample().Encode()
	for cut := 0; cut < len(data); cut++ {
		if _, err := Decode(data[:cut]); err == nil {
			t.Errorf("cut at %d: no error", cut)
		}
	}
	for i := range data {
		broken := append([]byte(nil), data...)
		broken[i] ^= 0xff
		if r, err := Decode(broken); err == nil {
			r.Check() // must not panic
		}
	}
	if _, err := Decode(append(data, 0)); err == nil {
		t.Error("bytes after the end: no error")
	}
}

func TestCheck(t *testing.T) {
	cases := map[string]func(r *Record){
		"shown at two places": func(r *Record) { r.Blocks[0].Children = []int{1, 1} },
		"shown nowhere":       func(r *Record) { r.Blocks[3].Slots = r.Blocks[3].Slots[:1] },
		"a circle":            func(r *Record) { r.Blocks[2].Children = []int{0} },
		"no such block":       func(r *Record) { r.Blocks[0].Children = []int{9} },
		"no top":              func(r *Record) { r.Top = 9 },
	}
	for name, damage := range cases {
		r := sample()
		damage(r)
		if err := r.Check(); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}
