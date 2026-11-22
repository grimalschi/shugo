package svelte

import (
	"errors"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

// compiler starts a compiler on Bun, or on Node without it.
func compiler(t *testing.T) *Compiler {
	t.Helper()
	runtime := "bun"
	if _, err := exec.LookPath(runtime); err != nil {
		runtime = "node"
	}
	c, err := Start(runtime, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	return c
}

func TestCompile(t *testing.T) {
	c := compiler(t)
	got, err := c.Compile("src/pages/index.svelte", `<script module>
  import Counter from '@/components/Counter.island.svelte'
  export function getStaticPaths() { return [] }
</script>
<script lang="ts">
  import { page, type Page } from "shugo"
  import { getContext } from "svelte"
  import Card from '@/components/Card.svelte'
  const id: string = $props.id()
  const theme = getContext("theme")
</script>
<Counter start={2} />
<Card {id} />
<p class="x">{page.url.pathname}</p>
<style>p { color: red }</style>`)
	if err != nil {
		t.Fatal(err)
	}
	want := Facts{Page: true, Context: true, ID: true, Paths: true, Islands: []Island{{Source: "@/components/Counter.island.svelte"}}}
	if !reflect.DeepEqual(got.Facts, want) {
		t.Errorf("facts: %+v, want %+v", got.Facts, want)
	}
	if !strings.Contains(got.Code, "export default function Pages($$renderer") || !strings.Contains(got.Map, `"sources":["index.svelte"]`) {
		t.Errorf("code: %s\nmap: %s", got.Code, got.Map)
	}
	if !strings.HasPrefix(got.CSS, "p.svelte-") {
		t.Errorf("css: %s", got.CSS)
	}
}

func TestRequestBindings(t *testing.T) {
	c := compiler(t)
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
		{`<Island request:initial={source} /><Same initial={data} />`, true},
	} {
		got, err := c.Compile("src/pages/catalog.svelte", `<script>import Island from './Island.island.svelte'; import Same from './Island.island.svelte'</script>`+test.markup)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Islands) != 1 || got.Islands[0].Request != test.want {
			t.Errorf("%s: islands %+v, want request %v", test.markup, got.Islands, test.want)
		}
		if len(got.Warnings) > 0 {
			t.Errorf("%s: warnings %v", test.markup, got.Warnings)
		}
	}
}

// The mistakes that are Shugo's own are told as Svelte tells its own.
func TestMistakes(t *testing.T) {
	c := compiler(t)
	for _, test := range []struct{ source, message string }{
		{`<script>import { Counter } from './Counter.island.svelte';</script><Counter />`, "1:18: import an island as a default component"},
		{`<script>import Counter from './Counter.island.svelte';</script><Counter>child text</Counter>`, "cannot pass children or snippets"},
		{`<script>import Counter from './Counter.island.svelte';</script><Counter>{#snippet detail()}hi{/snippet}</Counter>`, "cannot pass children or snippets"},
		{`<script>import Counter from './Counter.island.svelte';</script><svelte:component this={Counter} />`, "is rendered by its imported tag"},
		{`<script>import Counter from './Counter.island.svelte'; let value = $state(1)</script><Counter bind:value />`, "directives belong inside"},
		{`<script>import Counter from './Counter.island.svelte';</script><Counter request:initial />`, "needs one source expression"},
		{`<script>import Counter from './Counter.island.svelte';</script><Counter request:initial={source} initial={value} />`, "already gets its value"},
		{`<script>import Counter from './Counter.island.svelte';</script><Counter request:initial={source} request:initial={other} />`, "Attributes need to be unique"},
		{`<div request:initial={source}></div>`, "binds a prop on an .island.svelte component"},
		{`<script>import Counter from './Counter.island.svelte';</script><svg><Counter /></svg>`, "its wrapper in the HTML body"},
		{"<svelte:options runes={false} />\n<script>export let a</script>", "1:17: runes={false} turns on the syntax of Svelte 4"},
		{"<p>{cond && <b>x</b>}</p>", "1:13: markup cannot stand inside {…}"},
		{`<g-if cond={true}>x</g-if>`, "there is no <g-if>: write {#if condition}"},
		{`<div><slot /></div>`, "there is no <slot>: write {@render children?.()}"},
		{`<script>import C from "./c.svelte"</script><C><b slot="x">t</b></C>`, `slot="…" is a part of the slots of Svelte 4`},
		{`<html><body><p>x</p></body></html>`, "1:1: <html> is written in src/site.html"},
		{"<script>\nconst items = await Promise.resolve([1])\n</script>", "a page is computed without waiting"},
		{`<script>export function getStaticPaths() { return [] }</script>`, "getStaticPaths goes into <script module>"},
		{`<p>{#if x}</p>`, "1:11: `</p>` attempted to close an element that was not open"},
	} {
		_, err := c.Compile("src/pages/index.svelte", test.source)
		var found Mistakes
		if !errors.As(err, &found) || len(found) != 1 || !strings.Contains(found.Error(), test.message) {
			t.Errorf("compile %s: got %v, want %q", test.source, err, test.message)
		}
	}
}

// A static page is rendered once: the warnings of Svelte about what changes
// in a browser are not told.
func TestWarnings(t *testing.T) {
	c := compiler(t)
	got, err := c.Compile("src/components/Card.svelte", `<script>
  let { product } = $props()
  const price = product.price * 2
</script>
<img src="/a.png"><b>{price}</b>`)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Warnings) != 1 || !strings.Contains(got.Warnings[0].Text, "`<img>` element should have an alt attribute") {
		t.Errorf("warnings: %+v", got.Warnings)
	}
}

func TestModules(t *testing.T) {
	modules := Modules()
	for _, want := range []string{"svelte", "svelte/internal/server", "svelte/store", "svelte/reactivity/window"} {
		found := false
		for _, module := range modules {
			found = found || module == want
		}
		if !found {
			t.Errorf("no %s among %v", want, modules)
		}
	}
}
