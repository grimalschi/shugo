package compile

import (
	"strings"
	"testing"

	"github.com/grimalschi/shugo/internal/program"
)

func TestLowerIsland(t *testing.T) {
	u, err := compileFile("src/pages/catalog.svelte", `<script module>
  import Counter from '@/components/Counter.island.svelte';
</script>
<script>
  import StaticCard from '@/components/StaticCard.svelte';
  const props = { start: 2 };
</script>
<Counter {...props} />
<StaticCard />`, 0, page)
	if err != nil {
		t.Fatal(err)
	}
	if u.islands["Counter"] != (islandBinding{path: "@/components/Counter.island.svelte"}) || len(u.islands) != 1 {
		t.Fatalf("island imports: %#v", u.islands)
	}
	var kinds []program.Kind
	for _, op := range u.parts[0] {
		if op.Kind != program.Markup {
			kinds = append(kinds, op.Kind)
		}
	}
	if len(kinds) != 2 || kinds[0] != program.Island || kinds[1] != program.Call {
		t.Fatalf("compiled %q; want one island followed by one ordinary component call", kinds)
	}
	if code := strings.Join(u.module.lines, "\n"); !strings.Contains(code, "$$.island((\nCounter\n), {") {
		t.Fatalf("missing island evaluation: %s", code)
	}
}

func TestRequestBindingSelectsIslandProgram(t *testing.T) {
	for _, test := range []struct {
		markup string
		want   bool
	}{
		{`<Island />`, false},
		{`<Island initial={search()} />`, false},
		{`<Island {...props} />`, false},
		{`<Island request:initial={search()} />`, true},
		{`<Island request:results={source} {...props} />`, true},
		{`<Island request:initial={source} /><Island initial={data} />`, true},
	} {
		u, err := compileFile("src/pages/catalog.svelte", `<script>import Island from './Island.island.svelte'</script>`+test.markup, 0, page)
		if err != nil {
			t.Fatal(err)
		}
		if got := u.islands["Island"].request; got != test.want {
			t.Errorf("%s: request program = %v, want %v", test.markup, got, test.want)
		}
	}
}

func TestIslandBoundaryDiagnostics(t *testing.T) {
	for _, test := range []struct{ source, message string }{
		{`<script>import { Counter } from './Counter.island.svelte';</script><Counter />`, "import an island as a default component"},
		{`<script>import Counter from './Counter.island.svelte';</script><Counter>child text</Counter>`, "cannot pass children or snippets"},
		{`<script>import Counter from './Counter.island.svelte';</script><Counter>{#snippet detail()}hi{/snippet}</Counter>`, "cannot pass children or snippets"},
		{`<script>import Counter from './Counter.island.svelte';</script><svelte:component this={Counter} />`, "must be rendered by its imported tag"},
		{`<script>import Counter from './Counter.island.svelte';</script><Counter bind:value />`, "directives belong inside"},
		{`<script>import Counter from './Counter.island.svelte';</script><Counter shugoSearch />`, "use request:initial"},
		{`<script>import Counter from './Counter.island.svelte';</script><Counter request:initial />`, "needs one source expression"},
		{`<script>import Counter from './Counter.island.svelte';</script><Counter request:initial={source} initial={value} />`, "already gets its value"},
		{`<script>import Counter from './Counter.island.svelte';</script><Counter request:initial={source} request:initial={other} />`, "already bound"},
		{`<div request:initial={source}></div>`, "binds a prop on an .island.svelte component"},
		{`<script>import Counter from './Counter.island.svelte';</script><svg><Counter /></svg>`, "its wrapper in the HTML body"},
	} {
		if _, err := compileFile("src/pages/index.svelte", test.source, 0, page); err == nil || !strings.Contains(err.Error(), test.message) {
			t.Errorf("compile %s: got %v, want %q", test.source, err, test.message)
		}
	}
}

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
