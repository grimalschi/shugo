# Templates

The template — page, model or component — is a component Svelte 5 in file `.svelte`It's compiled for the server and render when assembled Svelte 5.59.1., curled in Shugo:The project does not need to be set up for the pages. SvelteSee also the "S" report, para. [documentation Svelte](https://svelte.dev/docs/svelte/overview). Shugo Adds blocks -- the general component is re-rated once on the entire site -- the module `shugo` with data and page address, shell `src/site.html` And his tests, and the hydration comments cut out. Shugo He's retreating from Svelte — [below](#how-shugo-differs-from-svelte)The net component may also be included in the section of the island where Svelte Combine it for the browser. The sub-assembly with the hydration entry point has a suffix. `.island.svelte`. [Islands Svelte](scripts.md#svelte-islands)

## Decisions

- The terms, cycles, expressions, snippets and styles of regular templates have the syntax Svelte 5.
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
- Templates are `.svelte` files, the island is `.island.svelte`; their official parser Svelte 5.59.1 (22 solution November 2026 year). Prior to that, the files were called `.shugo`, and Shugo parsed them with its parser.
- Templates compile and render Svelte itself, rather than its Shugo compiler; comments that Svelte writes for hydration are cut everywhere except islands (22 project author's decision November 2026 years).
- An error in expression shows a row and column in the `.svelte` file.
- Data and page address are taken by import from the module `shugo`: `import { query, paginate, page } from "shugo"`; `page` How to `page` the City of `$app/state` the City of London SvelteKitGlobal facility `Shugo` No, and the transition time with two records is also (decision 22 November 2026 years).
- `search` for islands is also from the `shugo` module; recording `shugo:search` is a hint error (22 solution November 2026 year).

## Implemented, pending approval

### File structure

```svelte
<script module>
  // Once per module. List of route pages is here.
  import { query, page } from "shugo"

  export function getStaticPaths() {
    return query("products").select("slug").all().map(({ slug }) => ({ params: { slug } }))
  }
</script>

<script lang="ts">
  // Every time the file is shown.
  import Card from "../../components/Card.svelte"
  import { price } from "@/lib/shop"

  let { limit = 10 } = $props()
  const item = query("products").where({ slug: page.params.slug }).first()
  const related = query("related").where({ product: item.id }).order("position").limit(limit).all()
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
- `<script>` is executed every time a file is shown: once for a page, once for each component call. Common blocks are an exception, see. [Components](components.md#common-components). `getStaticPaths` in it is an error with a prompt to transfer it to `<script module>`. `export let` - parameters Svelte 4 - error Svelte with a hint about `$props()`.
- The component parameters are `let { … } = $props()`.
- What framework knows is imported from the `shugo` module, in any file page, layout, component, module from `src/lib/`:

  | Name | What is it |
  | --- | --- |
  | `query(table)` | question to the project database, see [Data Download](data-loading.md) |
  | `paginate(…)` | for the list part of the page with the number, see [Dynamic pages](dynamic-pages.md#pagination) |
  | `page.url` | page address, `URL`, on the site from `shugo.json` |
  | `page.params` | address options: `{ slug: "jug" }` for `/products/[slug]` |
  | `page.route.id` | route: `/products/[slug]` |
  | `page.status` | 200, and the page `404.svelte` — 404 |
  | `search(table, { … })` | search by table for islands, see [Page scripts](scripts.md#search-by-table) |

  The query belongs to what is being calculated: a page or a common component. Outside of computing for example, at the top level of a module it is an error with an explanation. The component that takes `page` or `paginate`, either by itself or through imported modules, is calculated for each page separately, see. [Components](components.md#common-components). Writing `Shugo.query` and `Astro.params` is an error with a hint of what to import.
- `await` no: the page is calculated without waiting, because `query` responds immediately. `await` in `<script>` or in markup is an error with this explanation. What is expected and done once for example, the preparation of the code highlighting is expected at the top level of `<script module>` or the module from `src/lib/`.
- For TypeScript, write `<script lang="ts">` or `<script module lang="ts">`: Svelte recognizes TypeScript by this attribute and erases the types without checking them. `enum` and other designs that TypeScript turns into code are the error Svelte: instead of `enum`, it is the object `as const`. In `.ts` modules, the types are removed by esbuild, and there `enum` works. Type of module imported via `type`: `import type { Product } from "@/lib/shop"` or `import { price, type Product } from "@/lib/shop"`. Svelte does not know that `Product` is the type, and leaves the usual import in the code, and the assembly stops: `No matching export`.
- `@/` in import path means folder `src/`.
- Rune modules — `counter.svelte.ts`, `state.svelte.js` — compile Svelte, as in SvelteKit: imported as regular modules, `import { counter } from "@/lib/counter.svelte"`. On the server `$state` in them — just a value; on the island they work in the browser.
- From Svelte template and everything it imports take modules `svelte`, `svelte/store`, `svelte/reactivity`, `svelte/reactivity/window`, `svelte/easing`, `svelte/transition`, `svelte/animate`, `svelte/motion`, `svelte/events`, `svelte/attachments` and `svelte/server`The server part is sewn into Shugo. Another module, for example `svelte/legacy`Error with the list of those that are available.
- Runes on server: `$state(x)` is just a value of `x`, `$derived(…)` is considered to be once, `$effect` is not executed. The page is rendered once, when assembled, and its code does not work in the browser; what should work there is written by the island.
- `return` at the top level `<script>` and `export default` are Svelte errors.

### Text

```svelte
<h1>{item.title}</h1>
<p>{price(item.price * (100 - discount) / 100)}</p>
<p>{user ? user.name : "guest"}</p>
```

What brings up the value:

| Value | In text |
| --- | --- |
| string | as is, `&` and `<` are screened |
| `null`, `undefined` | nothing |
| `true`, `false` | `true`, `false` |
| number | as written by JS |
| array | comma elements: `{[1, 2]}`  `1,2` |
| object | `[object Object]` |

In the curly brackets are written expressions JavaScript or TypeScript. To repeat markup use `{#each}`.

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
- `class` accepts array and object: true elements and keys with true values fall into the class. There are `class:name={condition}` and `style:property={value}`.
- Event handlers (`onclick`), `use:`, `{@attach …}`, `transition:` and the like do not get to the page: there is no one to perform them on the static page. `bind:value` displays a value. The code for the browser is in [Scripts on page](scripts.md).

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
- The key in parentheses is accepted and does not affect anything on the static page.
- `{#each items}` without name and `{#each { length: 3 }, i}` also work.

### Other blocks and tags

| Recording | What does |
| --- | --- |
| `{@html post.body}` | inserts a value without shielding; comments remain in it |
| `{@const total = a + b}` | constant inside the block |
| `{#snippet name(params)}…{/snippet}`, `{@render name(…)}` | a piece of markup that is shown elsewhere; details are in [Components](components.md#snippets) |
| `{#key value}…{/key}` | shows the content; on a static page, the key does not affect anything |
| `{#await promise}…{:then value}…{/await}` | for promise shows a waiting line, and it remains on the page: the page does not expect anything; for the usual value - the branch `{:then}` |
| `{@debug …}` | page does not print anything; values print to the build output at each render |

### Special elements

| Element | What does |
| --- | --- |
| `<svelte:head>` | markup for `<head>` pages: title, meta tags. It is written in any component and gets to the place of `%shugo.head%` shell `src/site.html`, see [Layouts](layouts.md) |
| `<svelte:element this={tag}>` | element with a calculated name; empty name shows nothing |
| `<svelte:boundary>` | shows the contents, and with the snippet `pending` - only `pending`, and it remains on the page. The error inside does not catch, even with the snippet `failed`: assembly stops with this error |
| `<svelte:window>`, `<svelte:document>`, `<svelte:body>` | they do nothing on the server. |
| `<svelte:options>` | configures component compilation; `css="injected"` puts component styles in `<head>` pages, see [Styles](styles.md#one-stylesheet-per-site) |
| `<svelte:self>`, `<svelte:component this={…}>` | work, but Svelte warns that they are outdated: instead, a component imported into itself and a component from a variable named tag |

### Markup and gaps

- Spaces and row transfers between tags and within text are compressed into one space. At the edges of the element and block (`{#if}`, `{#each}`Snippets, they are removed completely: `{#if a} ${a} {/if}` Gn `$5` without spaces at the edges. The required space at the edge of the block is written as follows: `{" "}`.
- In `<pre>` and `<textarea>`, the text remains as written.
- Comments `<!-- -->` in the page do not fall. The necessary comment displays `{@html "<!-- … -->"}`.
- Entities like `&nbsp;` and `&copy;` are displayed by symbols.
- The element that the browser would rearrange (`<div>` inside `<p>`, `<tr>` directly into `<table>`) is an assembly error. If there is a block between them, it is a warning.
- Invalid attachment of closing tags is an error with the space in the file. Where HTML allows not to close the tag (`<li>`, `<p>`, `<td>`), it closes itself, with a warning.
- `</script>` anywhere `<script>`, even in a line or comment, closes it.
- The literal brace in the text is `{"{"}` or `&#123;`.
- Attached `<script>` and `<style>` inside the element or in `<svelte:head>` fall into the page as written.

### How Shugo differs from Svelte

| What | Svelte | Shugo |
| --- | --- | --- |
| Comments on hydration `<!--[-->`, `<!--]-->`, `<!---->` and others | sd | cuts: static page they are not needed. Remain at the islands and in HTML of `{@html}` |
| Layout | renders with the page | rendered separately: the context given in it through `setContext` does not reach either the nested layout or the page, and the names from `$props.id()` in it start with his file number, `b3-s1` |
| Warnings about what is changing in the browser: `state_referenced_locally`, `non_reactive_update`, `binding_property_non_reactive`, `perf_avoid_inline_class`, `perf_avoid_nested_class`, `store_rune_conflict` | I have | no: static page is rendered once. The islands have them |

### What is not and what the compiler answers

Errors and warnings Svelte are displayed in its text, with reference to the explanation of `https://svelte.dev/e/…`; Shugo adds its errors to them.

| Recording | Response |
| --- | --- |
| `---` at the beginning of the file (Astro) | error Shugo: component code — in `<script>` |
| `<g-if>`, `<g-else-if>`, `<g-else>`, `<g-for>`, `<g-html>`, `<g-slot>`, `<g-fragment>` — the first syntax of Shugo | error Shugo with Svelte record, which is needed instead of tag |
| `<slot>`, `slot="…"` right in the component tag, `let:`, `<svelte:fragment>` (Svelte 4) | error Shugo: snippets, `{@render children?.()}` and `{#snippet name()}…{/snippet}` are needed. Elsewhere, for example, in a custom element, `slot="…"` is a common attribute |
| `<Fragment>` (Astro) | error Shugo: need `{#snippet name()}…{/snippet}` in component tag |
| `export let title`, `$:`, `$$props` (Svelte 4) | error Svelte: need `let { title } = $props()`, instead of `$:` `$derived` or `$effect` |
| `getStaticPaths` in `<script>` | error Shugo: its place is `<script module>` |
| `await` in `<script>` and in markup | error with explanation Shugo: page is calculated without waiting |
| `<html>`, `<head>`, `<body>`, `<!doctype>` | error Shugo: this is written in the shell of `src/site.html`, see [Layouts](layouts.md#shell-srcsitehtml) |
| `{cond && <b>…</b>}`, `{list.map(x => <li>…</li>)}` | error Svelte `Unexpected token`: markup is not worth inside `{…}`, you need `{#if}` and `{#each}` |
| `set:html={…}` (Astro) | only Svelte warning about colon in attribute name, and in page - attribute `set:html`; need `{@html …}` |
| `class:list={[…]}` (Astro) | nothing: this is `class:` directive, it puts the class `list`; need `class={[…]}` |
| `Shugo.props`, `Shugo.slots`, `Astro.props` | page calculation error: `$props()` is needed; caller snippet is parameter |
| `Shugo.query`, `Shugo.params`, `Astro.params`, `Astro.url` | error in calculating the page: import from the module `shugo` — `query`, `page.params`, `page.url` is required |
| `import … from "shugo:search"` | error: `search` is taken from the `shugo` module |
| import File `.shugo` | error with prompt to replace `.shugo` with `.svelte` |
| `svelte/legacy` and other Svelte modules not listed in the list of [above](#file-structure) | error with the list of modules that are available |
| island with content or directives, inside `<svelte:head>` or `<select>`; `request:` at tag that is not island | errors Shugo, see [Tag island](scripts.md#tag-island) |

## Open questions

- The remaining deviations from Svelte are in the table of [above](#how-shugo-differs-from-svelte). There is no separate proposal for compatibility anymore: the basis is Svelte itself, it compiles and renders templates, and its checks are those that work (22 closed on November 2026 year).
- Type check: it is decided to make a separate command `shugo check` together with verification of component parameters ([Command line](cli.md#decisions)). While the types are only erased.
- Warning when an object is displayed in the text: it is almost always a mistake. In Svelte it is not.

What Svelte does in templates, and what Shugo does, and the test that was chosen for syntax, is in [Shugo and Svelte](../docs/svelte.md).
