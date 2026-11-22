package test

import "testing"

// A template means what it means in Svelte, whatever Shugo keeps apart: the
// HTML of every case is what Svelte 5.59.1 renders for the same files.
func TestSvelteSemanticsAcrossBlocks(t *testing.T) {
	for _, c := range []struct{ name, why, files, head, body string }{
		{"args", "a snippet without parameters ignores what the shared block passes it", `
-- src/pages/index.svelte --
<script>import Dropdown from "../components/Dropdown.svelte"</script>
<Dropdown label="Menu">Static</Dropdown>
-- src/components/Dropdown.svelte --
<script>let { label, children } = $props(); const close = () => {}</script>
<details><summary>{label}</summary>{@render children?.({ close })}</details>
`, "", "<details><summary>Menu</summary>Static</details>"},
		{"bind", "a component given bind: is rendered in place, and the render Svelte does again leaves no block behind", `
-- src/pages/index.svelte --
<script>
import Badge from "../components/Badge.svelte"
import Picker from "../components/Picker.svelte"
let value
</script>
<Badge label="x" /><Picker bind:value items={[1, 2]} /><p>{value}</p>
-- src/components/Badge.svelte --
<script>let { label } = $props()</script>
<i>{label}</i>
-- src/components/Picker.svelte --
<script>let { value = $bindable("a"), items } = $props()</script>
<b>{items.length}</b>
`, "", "<i>x</i><b>2</b><p>a</p>"},
		{"custom", "slot=\"…\" of a custom element is not a slot of Svelte 4", `
-- src/pages/index.svelte --
<sl-button><sl-icon slot="prefix" name="gear"></sl-icon>Settings</sl-button>
`, "", "<sl-button><sl-icon slot=\"prefix\" name=\"gear\"></sl-icon>Settings</sl-button>"},
		{"ids", "a snippet that calls $props.id() is rendered in place, so every copy has an id of its own", `
-- src/pages/index.svelte --
<script>
import Twice from "../components/Twice.svelte"
import Field from "../components/Field.svelte"
</script>
<Twice><Field /></Twice><Field />
-- src/components/Field.svelte --
<script>const id = $props.id()</script>
<input {id}>
-- src/components/Twice.svelte --
<script>let { children } = $props()</script>
<div>{@render children()}{@render children()}</div>
`, "", "<div><input id=\"s1\"/><input id=\"s2\"/></div><input id=\"s3\"/>"},
		{"select", "a component that shows a snippet inside a <select> is rendered in place, so its value chooses the option", `
-- src/pages/index.svelte --
<script>import Select from "../components/Select.svelte"</script>
<Select value="b"><option value="a">A</option><option value="b">B</option></Select>
-- src/components/Select.svelte --
<script>let { value, children } = $props()</script>
<select {value}>{@render children()}</select>
`, "", "<select><option value=\"a\">A</option><option value=\"b\" selected=\"\">B</option></select>"},
		{"show", "a snippet that fails where the shared block would not show it sends the component back in place", `
-- src/pages/index.svelte --
<script>
import Show from "../components/Show.svelte"
const p = { title: "Cup", discount: null }
const q = { title: "Jug", discount: { percent: 10 } }
</script>
<Show when={p.discount !== null}>{p.discount.percent}%</Show><Show when={q.discount !== null}>{q.discount.percent}%</Show>
-- src/components/Show.svelte --
<script>let { when, children } = $props()</script>
<span>{#if when}{@render children()}{/if}</span>
`, "", "<span></span><span>10%</span>"},
		{"title", "what a snippet gives the head goes with the snippet, and its <title> comes after the shared block's", `
-- src/pages/index.svelte --
<script>
import Shell from "../components/Shell.svelte"
import ProductHead from "../components/ProductHead.svelte"
</script>
<Shell title="Products"><ProductHead name="Cup" /></Shell>
-- src/components/ProductHead.svelte --
<script>let { name } = $props()</script>
<svelte:head><title>{name}</title><meta name="product" content={name}></svelte:head>
<h1>{name}</h1>
-- src/components/Shell.svelte --
<script>let { title, children } = $props()</script>
<svelte:head><title>{title} | Shop</title></svelte:head>
<main>{@render children()}</main>
`, "<meta name=\"product\" content=\"Cup\"/><title>Cup</title>", "<main><h1>Cup</h1></main>"},
		{"unicode", "snippet names are names of JavaScript, not only of ASCII", `
-- src/pages/index.svelte --
<script>import Box from "../components/Box.svelte"</script>
<Box>{#snippet títle()}<b>Hello</b>{/snippet}<p>body</p></Box>
-- src/components/Box.svelte --
<script>let { títle, children } = $props()</script>
<section>{@render títle()}{@render children()}</section>
`, "", "<section><b>Hello</b><p>body</p></section>"},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := newSite(t, c.files)
			s.build()
			if got, want := s.page("/"), c.head+c.body; got != want {
				t.Errorf("%s\n got: %s\nwant: %s", c.why, got, want)
			}
		})
	}
}
