package compile

import "testing"

func TestIslandShugoReference(t *testing.T) {
	for _, test := range []struct {
		code string
		want bool
	}{
		{`const rows = Shugo.query('products').all();`, true},
		{`function render() { return Shugo?.query('products') }`, true},
		{"const text = `rows: ${Shugo.query('products')}`;", true},
		{`function render(Shugo) { return Shugo.query('products') }`, false},
		{`const data = { Shugo: 'Shugo.query' }; console.log(data.Shugo);`, false},
	} {
		if got := readsShugo(test.code); got != test.want {
			t.Errorf("readsShugo(%q) = %v, want %v", test.code, got, test.want)
		}
	}
}
