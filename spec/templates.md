# Templates

The document describes the `.shugo` files: what parts they are and how to write the markup. The `.shugo` file is a component of Svelte 5. Pages, layouts and components are written in the same way.

## Decisions

- **Templates — components of Svelte 5** (the decision of the author of the project 21 November 2026 year). The syntax and behavior of the templates copy Svelte: conditions, cycles, output values in text and attributes, snippets, styles. A piece of the template can be moved from Shugo to Svelte and back, and it will show the same. The meaning is a single syntax throughout the project: later on this basis, the islands of Svelte will appear, and the agent will not be confused between the two template languages. This replaces previous decisions about the Astro-like format and its `g-if`, `g-for`, `<g-slot>`, `<g-html>` tags.
- Cycle - `{#each}` of Svelte; no mandatory key (21 solution November 2026 year).
- The terms `{:else if}` and `{:else}` are similar to Svelte; `{#each}` also has `{:else}` for an empty list (21 solution November 2026 year).
- What output expressions in text and in attributes — as in Svelte (21 solution November 2026 year).
- The HTML is `{@html …}`, as in Svelte (21 solution November 2026 year).
- The markup is dealt with as strictly as in Svelte (21 solution of November 2026 year).
- The component passes the markup to another component in the same way as in Svelte: snippets (21 solution November 2026 year). Details - in [Components](components.md).
- The component shows itself in the same way as in Svelte (21 solution November 2026 year).
- The name tag is only as it is in Svelte: `<svelte:element this={…}>` (21 November 2026 solution).
- Expressions are written in JS, in source code - TypeScript. Functions and components are connected by explicit imports.
- The values are screened when inserted into the page.
- Addresses in `href`, `src` and similar attributes are displayed as is, as in Svelte: `javascript:` address replacement is canceled (22 November 2026 year solution).
- The file extension remains `.shugo` (22 solution of November 2026 year).
- An error in expression shows a row and column in the `.shugo` file.

## Implemented, pending approval

### File structure

```svelte
<script module>
  // Once per module. List of route pages is here.
  export function getStaticPaths({ query }) {
    return query("products").select("slug").all().map(({ slug }) => ({ params: { slug } }))
  }
</script>

<script lang="ts">
  // Every time the file is shown.
  import Card from "../../components/Card.shugo"
  import { price } from "@/lib/shop"

  let { limit = 10 } = $props()
  const item = Shugo.query("products").where({ slug: Shugo.params.slug }).first()
  const related = Shugo.query("related").where({ product: item.id }).order("position").limit(limit).all()
</script>

<svelte:head><title>{item.title}</title></svelte:head>
<h1>{item.title}</h1>
<p class="price">{price(item.price)}</p>
{#each related as one}<Card id={one.other} />{/each}

<style>
  h1 { font-size: 2rem }
</style>
```

- `<script module>` is executed once when the module is loaded. It contains `getStaticPaths` and everything you need once: constants, functions, imports. Its names are visible to the rest of the file.
- `<script>` is executed every time a file is shown: once for a page, once for each component call. Common blocks are an exception, see. [Components](components.md#common-components). Export from it can not: the compiler will prompt you to transfer exports to `<script module>`. `export let` — parameters Svelte 4 — also an error with a hint about `$props()`.
- The component parameters are `let { … } = $props()`, as in Svelte 5.
- The `Shugo` object is visible in `<script>` and in markup: `Shugo.query(…)` asks the project database ([Data loading](data-loading.md)), `Shugo.params` is the address parameters, `Shugo.url` is the page address, `Shugo.paginate(…)` is part of the list for the page with the number ([Dynamic pages](dynamic-pages.md)). `<script module>` does not have it: `getStaticPaths` gets `query` and `paginate` by argument.
- `await` no: the page is calculated without waiting, because `Shugo.query` responds immediately. What is expected and done once - for example, code highlighting - is expected at the top level of the module from `src/lib/`.
- TypeScript is accepted in any `<script>`. Svelte requires `lang="ts"`; for the file to be transferred to Svelte, write `<script lang="ts">`.
- `@/` in import path means folder `src/`.
- Runes Svelte work as on the server Svelte: `$state(x)` - just the value of `x`, `$derived(…)` is considered once, `$effect` is not executed. Page is static, in the browser its code does not work.
- `return` at the top level `<script>` and `export default` are build errors.

### Text

```svelte
<h1>{item.title}</h1>
<p>{price(item.price * (100 - discount) / 100)}</p>
<p>{user ? user.name : "guest"}</p>
```

The value is displayed as it is displayed by Svelte:

| Value | In text |
| --- | --- |
| string | as is, `&` and `<` are screened |
| `null`, `undefined` | nothing |
| `true`, `false` | `true`, `false` |
| number | as written by JS |
| array | comma elements: `{[1, 2]}`  `1,2` |
| object | `[object Object]` |

The expression is a regular TypeScript, and only it: you can not write the markup inside the braces. On `{items.map(item => <li>{item}</li>)}`, the compiler answers that you need `{#each}`.

### Attributes

```svelte
<a href={url} class="link {active ? 'active' : ''}" title="plain text">…</a>
<input disabled={!ready} value={query} {...rest}>
<img {src} alt="">
<p class={["card", { featured, sale: item.discount > 0 }]} class:wide style:color={accent}>…</p>
```

- `attribute={expression}`, a substitution inside the quotation marks of `title="Order {id}"`, a short entry of `{src}` is the same as `src={src}`, and `{...rest}` are all object fields.
- `null` and `undefined` remove the attribute. The rest is displayed as in the text: `title={false}` gives `title="false"`.
- Attributes that either are or are not (`disabled`, `checked`, `hidden`, `selected` and others from the Svelte list) are displayed as `disabled=""`; the false value removes them. An empty string, as in the markup, leaves.
- `class` accepts array and object as in Svelte: true elements and keys with true values fall into the class. There are `class:name={condition}` and `style:property={value}`.
- Event handlers (`onclick`, `on:click`), `bind:`, `use:`, `transition:` and the like work as on the Svelte server: handlers and actions do not fall into the page, `bind:value` outputs a value.

### Condition

```svelte
{#if item.stock > 3}
  Available
{:else if item.stock > 0}
  {item.stock}
{:else}
  Custom Made
{/if}
```

### Cycle

```svelte
{#each products as product, i (product.id)}
  <li>{i + 1}. {product.title}</li>
{:else}
  <li>While empty</li>
{/each}
```

- An array or something similar to it is moved: `Set`, `Map`, a string. `null` and `undefined` are an empty list.
- The element can be disassembled immediately: `{#each rows as { id, title }}`, `{#each pairs as [key, value]}`.
- The key in parentheses is accepted as in Svelte, and on the static page does not affect anything.
- `{#each items}` without name and `{#each { length: 3 }, i}` also work.

### Other blocks and tags

| Recording | What does |
| --- | --- |
| `{@html post.body}` | inserts a value without shielding |
| `{@const total = a + b}` | constant inside the block |
| `{#snippet name(params)}…{/snippet}`, `{@render name(…)}` | a piece of markup that is shown elsewhere; details are in [Components](components.md#snippets) |
| `{#key value}…{/key}` | shows the content; on a static page, the key does not affect anything |
| `{#await promise}…{:then value}…{/await}` | for promise shows a waiting branch like the Svelte server; for normal value, the `{:then}` branch |
| `{@debug …}` | nothing displays |

### Special elements

| Element | What does |
| --- | --- |
| `<svelte:head>` | markup for `<head>` pages: title, meta tags. Written in the page and in the layout, see [Layouts](layouts.md) |
| `<svelte:element this={tag}>` | element with a calculated name; empty name shows nothing |
| `<svelte:boundary>` | shows its content |
| `<svelte:window>`, `<svelte:document>`, `<svelte:body>`, `<svelte:options>` | they do nothing on the server. |
| `<svelte:self>`, `<svelte:component this={…}>` | work with a warning: in Svelte 5 they are outdated, instead of them - a component imported into itself and a component from a variable named tag |

### Markup and gaps

Svelte Rules:

- Comments `<!-- -->` in the page do not fall.
- Spaces and row transfers between tags and within text are compressed into one space. At the edges of the element and block (`{#if}`, `{#each}`Snippets, they are removed completely: `{#if a} ${a} {/if}` Gn `$5` without spaces at the edges. The required space at the edge of the block is written as follows: `{" "}`.
- In `<pre>` and `<textarea>`, the text remains as written.
- Entities like `&nbsp;` and `&copy;` are transformed into symbols and are re-emerged: `&` and `<` are screened.
- Empty elements are written with a slash: `<br>` is displayed as `<br/>`.
- The element that the browser would rearrange (`<div>` inside `<p>`, `<tr>` directly into `<table>`) is an assembly error. If there is a block between them, it is a warning, as in Svelte.
- Invalid attachment of closing tags is an error with the space in the file. Where HTML allows not to close the tag (`<li>`, `<p>`, `<td>`), it closes itself, with a warning.
- `</script>` anywhere `<script>` closes it as in Svelte.
- The literal brace in the text is `{"{"}` or `&#123;`.
- Attached `<script>` and `<style>` inside the element or in `<svelte:head>` fall into the page as written.

### How Shugo differs from Svelte

| What | Svelte | Shugo |
| --- | --- | --- |
| hydration tags `<!--[-->`, `<!--]-->` | sd | it does not say: static page they are not needed |
| A class that limits styles to a component | only bet on items that selectors choose | is placed on all components; page view is the same |
| TypeScript without `lang="ts"` | error | adopted |
| `<svelte:head>` | in any component | only in page and layout |
| Availability Warnings (`a11y_…`) | I have | none |
| `$props()` in markup and other rune place errors | error | not checked |

### What is not and what the compiler answers

| Recording | Response |
| --- | --- |
| `---` at the beginning of the file (Astro) | component code — `<script>` |
| `g-if`, `g-for`, `<g-slot>`, `<g-html>`, `<g-fragment>` | prompt which Svelte record is needed |
| `<slot>`, `slot="…"`, `<svelte:fragment>` (Svelte 4) | snippets needed: `{@render children?.()}` and `{#snippet name()}…{/snippet}` |
| `export let title` (Svelte 4) | `let { title } = $props()` |
| `Shugo.props`, `Shugo.slots` | `$props()` is needed; caller snippet is parameter |
| `{cond && <b>…</b>}`, `{list.map(x => <li>…</li>)}` | markup can not stand inside `{…}`: you need `{#if}` and `{#each}` |

## Open questions

- How to remove remaining deviations from Svelte — `lang="ts"`, runes, availability warnings, style class, `<svelte:head>` in components, `$lib` — [Svelte compatibility offer](../docs/proposals/svelte-compatibility.md).
- Type check: it is decided to make a separate command `shugo check` together with verification of component parameters ([Command line](cli.md#decisions)). While the types are only erased.
- Warning when an object is displayed in the text: it is almost always a mistake. In Svelte it is not.

As Shugo achieves the same HTML as Svelte, and the test by which syntax was chosen is in [Shugo and Svelte](../docs/svelte.md).
