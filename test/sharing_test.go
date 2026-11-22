package test

import (
	"regexp"
	"strings"
	"testing"
)

// When the whole site is computed, a component that is kept apart for
// nothing is told of: it has many shared blocks, and each is shown in about
// one place.
func TestSharingThatGainsNothing(t *testing.T) {
	s := newSite(t, `
-- src/pages/+layout.svelte --
<script>
import { page } from "shugo"
let { children } = $props()
</script>
<nav>{page.url.pathname}</nav>{@render children()}
-- src/pages/[n].svelte --
<script module>
import { page } from "shugo"
export function getStaticPaths() {
  return Array.from({ length: 120 }, (_, i) => ({ params: { n: String(i + 1) } }))
}
</script>
<script>
import Price from "../components/Price.svelte"
import Badge from "../components/Badge.svelte"
</script>
<Price value={Number(page.params.n) * 10} /><Badge kind="new" />
-- src/components/Price.svelte --
<script>
let { value } = $props()
</script>
<b>${value}</b>
-- src/components/Badge.svelte --
<script>
let { kind } = $props()
</script>
<i>{kind}</i>
`)
	report := s.build()
	// The price of every page is its own, the badge is one for all pages,
	// and the layout reads the page: it is kept for every page by design.
	if len(report.Warnings) != 1 || !strings.HasPrefix(report.Warnings[0], "src/components/Price.svelte is kept apart as 120 shared blocks, each shown in 1.0 places on average.") {
		t.Errorf("warnings: %q", report.Warnings)
	}
	expect(t, s.page("/7"), "<nav>/7</nav><b>$70</b><i>new</i>")
	// An update computes a part of the site, and weighs nothing.
	if report := s.update(); len(report.Warnings) != 0 {
		t.Errorf("an update warns: %q", report.Warnings)
	}
}

// A snippet given to a shared block may go on into another one: the place
// renders it, wherever the shared blocks show it.
func TestSnippetsThroughSharedBlocks(t *testing.T) {
	s := newSite(t, `
-- src/components/Frame.svelte --
<script>
import Box from "./Box.svelte"
let { title, children } = $props()
</script>
<section><h2>{title}</h2><Box kind="inner">{@render children()}</Box></section>
-- src/components/Box.svelte --
<script>
let { kind, children, footer } = $props()
</script>
<div class={kind}>{@render children()}{@render footer?.()}</div>
-- src/pages/[n].svelte --
<script module>
export const getStaticPaths = () => [{ params: { n: "1" } }, { params: { n: "2" } }]
</script>
<script>
import { page } from "shugo"
import Frame from "../components/Frame.svelte"
import Box from "../components/Box.svelte"
</script>
<Frame title="Same"><p>page {page.params.n}</p></Frame>
<Box kind="outer">x{#snippet footer()}<i>{page.params.n}</i>{/snippet}</Box>
`)
	report := s.build()
	// Frame and the Box inside it are one block each for both pages; the
	// Box with a footer of its own is one more, as both pages give it the
	// same props.
	if report.Shared.Total != 3 {
		t.Errorf("shared blocks: %+v", report.Shared)
	}
	expect(t, s.page("/1"), `<section><h2>Same</h2><div class="inner"><p>page 1</p></div></section> <div class="outer">x<i>1</i></div>`)
	expect(t, s.page("/2"), `<section><h2>Same</h2><div class="inner"><p>page 2</p></div></section> <div class="outer">x<i>2</i></div>`)
}

// What depends on the place a component stands at keeps it from being
// shared: the context of Svelte, which the components around give, and the
// ids of $props.id(), which count what was rendered before.
func TestContextAndIdsAreRenderedInPlace(t *testing.T) {
	s := newSite(t, `
-- src/components/Theme.svelte --
<script>
import { setContext } from "svelte"
let { name, children } = $props()
setContext("theme", name)
</script>
<div>{@render children()}</div>
-- src/components/Label.svelte --
<script>
import { getContext } from "svelte"
let { text } = $props()
const theme = getContext("theme")
</script>
<b class={theme}>{text}</b>
-- src/components/Field.svelte --
<script>
let { text } = $props()
const id = $props.id()
</script>
<label for={id}>{text}</label><input {id}>
-- src/pages/+layout.svelte --
<script>
import Field from "../components/Field.svelte"
let { children } = $props()
</script>
<Field text="search" />{@render children()}
-- src/pages/index.svelte --
<script>
import Theme from "../components/Theme.svelte"
import Label from "../components/Label.svelte"
import Field from "../components/Field.svelte"
</script>
<Theme name="dark"><Label text="a" /></Theme><Theme name="light"><Label text="a" /></Theme>
<Field text="one" /><Field text="two" />
`)
	report := s.build()
	if report.Shared.Total != 1 {
		t.Errorf("only the layout is kept apart: %+v", report.Shared)
	}
	// The ids of the layout, computed apart, are named after its file.
	got := s.page("/")
	want := `<div><b class="dark">a</b></div><div><b class="light">a</b></div> <label for="s1">one</label><input id="s1"/><label for="s2">two</label><input id="s2"/>`
	if !strings.Contains(got, want) || !regexp.MustCompile(`^<label for="b\d+-s1">search</label><input id="b\d+-s1"/>`).MatchString(got) {
		t.Errorf("got %s", got)
	}
}

// A snippet whose parameters all have defaults looks like one without
// parameters: a shared block that shows it with arguments is told so.
func TestSnippetWithDefaultsGivenToASharedBlock(t *testing.T) {
	s := newSite(t, `
-- src/components/List.svelte --
<script>
let { row } = $props()
</script>
<ul>{@render row("x")}</ul>
-- src/pages/index.svelte --
<script>
import List from "../components/List.svelte"
</script>
<List>{#snippet row(item = "none")}<li>{item}</li>{/snippet}</List>
`)
	if _, err := s.tryBuild(); err == nil || !strings.Contains(err.Error(), "src/components/List.svelte shows the snippet row with arguments") {
		t.Errorf("got %v", err)
	}
}
