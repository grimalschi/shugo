# Templates

Pages, layouts, and reusable components are Svelte 5 components in `.svelte` files. Shugo uses Svelte 5.59.1 to compile them for the server and render them during the build. The `shugo` package installs that version. See the [Svelte documentation](https://svelte.dev/docs/svelte/overview) for syntax, errors, and warnings.

Shugo can render a shared component once for the whole site. It provides data and page information through the `shugo` module, assembles pages in `src/site.html`, and runs additional template checks. It removes hydration comments from static output. See [How Shugo differs from Svelte](#how-shugo-differs-from-svelte) for details.

A component can also be part of an island's dependency graph, where Svelte compiles it for the browser. An island's hydration entry point uses the suffix `.island.svelte`. See [Svelte islands](scripts.md#svelte-islands).

## Decisions

- Use Svelte 5 for conditions, loops, expressions, snippets, and styles.
- Use Svelte's `{#each}` for loops. Keys are optional (November 21, 2026).
- Use `{:else if}` and `{:else}` for conditions, and `{:else}` on an `{#each}` to handle an empty list (November 21, 2026).
- Svelte renders expressions in text and attributes (November 21, 2026).
- Insert HTML with `{@html …}` (November 21, 2026).
- Svelte applies its own strict markup parsing (November 21, 2026).
- Pass markup between components with snippets. See [Components](components.md) (November 21, 2026).
- A recursive component imports its own file (November 21, 2026).
- Use `<svelte:element this={…}>` for a dynamic tag name (November 21, 2026).
- Write expressions in JavaScript or TypeScript, and import functions and components explicitly.
- The renderer escapes interpolated values.
- The renderer preserves URLs in `href`, `src`, and similar attributes, including `javascript:` URLs (November 22, 2026).
- Use `.svelte` for templates and `.island.svelte` for islands. Svelte 5.59.1 parses both. Earlier templates used `.shugo` template files and Shugo's own parser (November 22, 2026).
- Svelte compiles and renders templates. Shugo removes the hydration comments outside islands and raw HTML (November 22, 2026).
- A template is any `.svelte` file for a page, layout, or component. A component is a reusable part of a template (November 22, 2026).
- Errors in expressions identify the line and column in the `.svelte` file.
- Import `query`, `paginate`, and `page` from `shugo`. The `page` object follows SvelteKit's `$app/state` API. The former `Shugo` global was removed without a compatibility period (November 22, 2026).
- Import island `search` from `shugo`. The build reports an error with migration advice for `shugo:search` (November 22, 2026).
- Templates use runes. The compiler reports an error and a hint for `<svelte:options runes={false} />` (November 22, 2026).
- Top-level `await` in `<script module>` was briefly disabled to identify waits and data sources outside the database (November 22, 2026). The following day, the project author restored top-level `await` in `<script module>` and site modules.

## Implemented, pending approval

### File structure

```svelte
<script module>
  // Runs once per module. Put the route's page list here.
  import { query, page } from "shugo"

  export function getStaticPaths() {
    return query("products").select("slug").all().map(({ slug }) => ({ params: { slug } }))
  }
</script>

<script lang="ts">
  // Runs each time the component renders.
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

- `<script module>` runs when the module loads. Define `getStaticPaths`, constants, functions, and module-level imports there. Its bindings are available throughout the file.
- `<script>` runs for each render of a page or component. [Shared components](components.md#shared-components) reuse their output. If you put `getStaticPaths` here, the compiler tells you to move it to `<script module>`. Svelte also rejects the Svelte 4 prop declaration `export let` and points to `$props()`.
- Read component props with `let { … } = $props()`.
- Import framework values from `shugo` in a page, layout, component, or module under `src/lib/`.

  | API | Value or behavior |
  | --- | --- |
  | `query(table)` | Query the project database. See [Loading data](data-loading.md). |
  | `paginate(…)` | Select one page of a list. See [Pagination](dynamic-pages.md#pagination). |
  | `page.url` | A `URL` resolved against the site address in `shugo.json`. |
  | `page.params` | Route parameters, such as `{ slug: "jug" }` for `/products/[slug]`. |
  | `page.route.id` | The route pattern, such as `/products/[slug]`. |
  | `page.status` | 200 by default and 404 for `404.svelte`. |
  | `search(table, { … })` | Search a table from an island. See [Table search](scripts.md#table-search). |

  Shugo associates a query with the page or shared component it is computing. A call outside that computation, such as at the top level of a module, reports an error. A component that uses `page` or `paginate` directly or through an import renders separately for each page. See [Shared components](components.md#shared-components). Using `Shugo.query` reports an error with the required import.
- `query` is synchronous. While rendering a page, using `await` in `<script>` or the markup reports an error. Put one-time asynchronous work, such as preparing a syntax highlighter, at the top level of `<script module>` or a module under `src/lib/`. Module loading waits for it before the first render.
- Use `<script lang="ts">` or `<script module lang="ts">` for TypeScript. Svelte strips types without checking them. For constructs that generate JavaScript, such as `enum`, use a plain object with `as const`. In separate `.ts` modules, esbuild handles TypeScript and supports `enum`. Import types explicitly with `import type { Product } from "@/lib/shop"` or `import { price, type Product } from "@/lib/shop"`. A regular import of a type is emitted as a JavaScript import and fails with `No matching export`. Run `shugo check` to type-check against the project's `tsconfig.json` using the packaged `svelte-check` and TypeScript.
- The import prefix `@/` points to `src/`.
- Svelte compiles rune modules such as `counter.svelte.ts` and `state.svelte.js`, as in SvelteKit. Import them as ordinary modules, for example `import { counter } from "@/lib/counter.svelte"`. On the server, `$state` holds a value. In an island, the rune module also runs in the browser.
- Templates and their imports may use `svelte`, `svelte/store`, `svelte/reactivity`, `svelte/reactivity/window`, `svelte/easing`, `svelte/transition`, `svelte/animate`, `svelte/motion`, `svelte/events`, `svelte/attachments`, and `svelte/server` from Shugo's Svelte dependency. An import such as `svelte/legacy` reports an error listing the supported modules.
- On the server, `$state(x)` yields `x`, `$derived(…)` is evaluated once, and `$effect` is skipped. Static pages render at build time. Put code that should run in the browser in an island.
- Svelte rejects top-level `return` and `export default` in `<script>`.

### Text

```svelte
<h1>{item.title}</h1>
<p>{price(item.price * (100 - discount) / 100)}</p>
<p>{user ? user.name : "guest"}</p>
```

| Value | Text output |
| --- | --- |
| String | The string with `&` and `<` escaped |
| `null`, `undefined` | No output |
| `true`, `false` | `true`, `false` |
| Number | The JavaScript string representation |
| Array | Comma-separated items, for example `{[1, 2]}` renders `1,2` |
| Object | `[object Object]` |

Write JavaScript or TypeScript expressions inside braces. Use `{#each}` to repeat markup.

### Attributes

```svelte
<a href={url} class="link {active ? 'active' : ''}" title="Plain text">…</a>
<input disabled={!ready} value={query} {...rest}>
<img {src} alt="">
<p class={["card", { featured, sale: item.discount > 0 }]} class:wide style:color={accent}>…</p>
```

- Use `attribute={expression}` or interpolate inside a quoted value, as in `title="Order {id}"`. The shorthand `{src}` means `src={src}`. Spread an object's fields with `{...rest}`.
- `null` and `undefined` omit the attribute. Other values render as they do in text. For example, `title={false}` produces `title="false"`.
- HTML boolean attributes such as `disabled`, `checked`, `hidden`, and `selected` render by presence, for example `disabled=""`. A false value omits the attribute. An explicit empty string keeps it.
- `class` accepts arrays and objects. Truthy array entries and object keys with truthy values contribute class names. You can also write `class:name={condition}` and `style:property={value}`.
- Static output omits event handlers such as `onclick`, as well as `use:`, `{@attach …}`, and `transition:`. `bind:value` still writes the value. See [Scripts on a page](scripts.md) for browser code.

### Conditions

```svelte
{#if item.stock > 3}
  In stock
{:else if item.stock > 0}
  {item.stock} left
{:else}
  Available to order
{/if}
```

### Loops

```svelte
{#each products as product, i (product.id)}
  <li>{i + 1}. {product.title}</li>
{:else}
  <li>No products yet</li>
{/each}
```

- Loop over an array, a `Set`, a `Map`, or a string. `null` and `undefined` count as empty lists.
- You can destructure each item with `{#each rows as { id, title }}` or `{#each pairs as [key, value]}`.
- A key in parentheses is accepted and has no effect on static output.
- Loops such as `{#each items}` and `{#each { length: 3 }, i}` can omit the item binding.

### Other blocks and tags

| Syntax | Behavior |
| --- | --- |
| `{@html post.body}` | Inserts the value without escaping and keeps its comments. |
| `{@const total = a + b}` | Declares a constant in the current block. |
| `{#snippet name(params)}…{/snippet}`, `{@render name(…)}` | Define and render reusable markup. See [Snippets](components.md#snippets). |
| `{#key value}…{/key}` | Renders its contents once. The key has no effect on static output. |
| `{#await promise}…{:then value}…{/await}` | Renders the pending branch for a promise. A plain value renders the `{:then}` branch. |
| `{@debug …}` | Logs values during each render and emits no page content. |

### Special elements

| Element | Behavior |
| --- | --- |
| `<svelte:head>` | Contributes a title, metadata, or other head markup from any component. Shugo inserts it at `%shugo.head%` in `src/site.html`. See [Layouts](layouts.md). |
| `<svelte:element this={tag}>` | Renders a dynamic element. An empty tag name produces no output. |
| `<svelte:boundary>` | Renders the contents or, when supplied, the `pending` snippet. Errors inside the boundary stop the build even if there is a `failed` snippet. |
| `<svelte:window>`, `<svelte:document>`, `<svelte:body>` | Have no server-side effect. |
| `<svelte:options>` | Configures compilation. `css="injected"` puts component CSS in the page's `<head>`, and `runes={false}` reports an error. See [One stylesheet per site](styles.md#one-stylesheet-per-site). |
| `<svelte:self>`, `<svelte:component this={…}>` | Work with Svelte deprecation warnings. Use a self-import for recursion and a variable as the tag name for dynamic components. |

### Markup and whitespace

- Whitespace between tags and within text collapses to one space. Svelte removes it at the start and end of an element or a block such as `{#if}`, `{#each}`, or a snippet. For example, `{#if a} {a} USD {/if}` renders `5 USD` without surrounding spaces. Use `{" "}` to preserve a space at a boundary.
- Svelte keeps the text in `<pre>` and `<textarea>` as written.
- Svelte removes `<!-- -->` comments. To keep one, write `{@html "<!-- … -->"}`.
- Svelte renders entities such as `&nbsp;` and `&copy;` as their characters.
- Invalid nesting that a browser would rearrange, such as `<div>` inside `<p>` or `<tr>` directly inside `<table>`, is a build error. A block between the elements produces a warning.
- Misnested closing tags report a source location. When HTML allows the end tag to be omitted, as for `<li>`, `<p>`, or `<td>`, Svelte closes the element and issues a warning.
- Within `<script>`, `</script>` closes the element even inside a string or comment.
- Write a literal opening brace as `{"{"}` or `&#123;`.
- Nested `<script>` and `<style>` elements, including those in `<svelte:head>`, pass through as written.

### How Shugo differs from Svelte

| Feature | Svelte | Shugo |
| --- | --- | --- |
| Hydration comments such as `<!--[-->`, `<!--]-->`, and `<!---->` | Writes the comments. | Removes them from static output. Keeps them in islands and in markup passed to `{@html}`. |
| Layout | Renders with the page. | Renders separately. A layout's `setContext` applies within the layout, and nested layouts and pages have their own contexts. Layout IDs from `$props.id()` start with the layout's file number, as in `b3-s1`. |
| Browser and reactivity warnings, including `state_referenced_locally`, `non_reactive_update`, `binding_property_non_reactive`, `perf_avoid_inline_class`, `perf_avoid_nested_class`, and `store_rune_conflict` | Reports these warnings. | Reports them for islands. Static pages render once at build time. |

### Compiler diagnostics

Shugo shows Svelte's messages and help links at `https://svelte.dev/e/…`, and adds its own checks.

| Syntax or situation | Diagnostic |
| --- | --- |
| Shugo's original tags `<g-if>`, `<g-else-if>`, `<g-else>`, `<g-for>`, `<g-html>`, `<g-slot>`, and `<g-fragment>` | Shugo reports an error and suggests the equivalent Svelte syntax. |
| Svelte 4 `<slot>`, `slot="…"` directly on a component, `let:`, and `<svelte:fragment>` | Shugo requires snippets such as `{@render children?.()}` and `{#snippet name()}…{/snippet}`. Elsewhere, such as on a custom element, `slot="…"` is a normal attribute. |
| Svelte 4 `export let title`, `$:`, and `$$props` | Svelte reports an error. Use `let { title } = $props()` for props and `$derived` or `$effect` for reactivity. |
| `<svelte:options runes={false} />` | Shugo requires runes. |
| `getStaticPaths` in `<script>` | Shugo asks you to move it to `<script module>`. |
| `await` in `<script>` or markup | Shugo explains that page rendering is synchronous. Top-level `await` works in `<script module>` and site modules. |
| `<html>`, `<head>`, `<body>`, and `<!doctype>` | Shugo requires these in `src/site.html`. See [Site shell](layouts.md#site-shell). |
| Importing `+layout.svelte` | Shugo applies a layout automatically. Move shared markup into a component if you need to import it. |
| `{cond && <b>…</b>}` or `{list.map(x => <li>…</li>)}` | Shugo rejects markup inside expressions. Use `{#if}` and `{#each}`. |
| `Shugo.props` or `Shugo.slots` | Rendering reports an error. Read props with `$props()` and receive a caller's snippet as a prop. |
| `Shugo.query` or `Shugo.params` | Rendering reports an error. Import `query` and `page` from `shugo` to access the database, `page.params`, and `page.url`. |
| `import … from "shugo:search"` | Import `search` from `shugo`. |
| Importing a `.shugo` template | Shugo tells you to use `.svelte`. |
| `svelte/legacy` and other modules outside the [supported list](#file-structure) | Shugo lists the supported modules. |
| A `.svelte` component from an npm package | Shugo cannot compile packaged components. Place the component under `src/components/`. |
| Package missing from `node_modules` | Shugo reports the import location and an install command, such as `date-fns is not installed: bun add date-fns`. |
| Package requires another Svelte version | Shugo reports both versions. |
| Package contains a native `.node` binary or another non-JavaScript file | Shugo reports a build error and suggests a JavaScript-only package. |
| Package reads a file relative to its own code | Rendering reports an error and explains that the package code was bundled into `_build/eval.mjs`. |
| Importing a type as a value, such as `import { Product } from "@/lib/shop"` | The build reports `No matching export` and suggests `import type`. |
| An in-place component renders itself indefinitely | Shugo reports the component chain, such as `src/components/Tree.svelte shows itself without end: … → …`. |
| An island with child content or directives, an island in `<svelte:head>` or a `<select>`, or `request:` on another tag | Shugo reports an error. See [Island tag](scripts.md#island-tag). |

## Open questions

- The remaining compatibility differences are [listed above](#how-shugo-differs-from-svelte). The separate compatibility proposal was closed after Shugo adopted Svelte's compiler, renderer, and checks (November 22, 2026).
- Interpolating an object as text is usually accidental. Consider a warning for it. Svelte does not currently warn.

[Shugo and Svelte](../docs/svelte.md) describes the division of responsibilities and the tests that informed the template syntax.
