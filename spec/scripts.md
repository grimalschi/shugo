# Scripts on a page

Browser code can run in a script embedded in the HTML or in a Svelte island.

## Decisions

- A `<script>` nested in markup is copied into the page. Braces inside it are ordinary text (November 21, 2026).
- Svelte provides the foundation for browser code (November 21, 2026).
- When a template renders an imported `*.island.svelte` component, it becomes a hydration entry point. The suffix appears both on disk and in imports (November 22, 2026). Islands shipped before the data adapter, reversing the order planned on November 22.
- Shugo initially bundled the browser runtime at the compiler's Svelte version (November 22, 2026). Since November 23, 2026, the `shugo` package has installed `svelte` as a dependency that supplies both the compiler and the browser runtime.
- Page requests execute only Go on the server. To render an island for a URL with filters, Go applies a program produced at build time from Svelte's server output (November 22, 2026).

## Implemented, pending approval

The top-level `<script>` in a regular `.svelte` file runs during page computation. For short browser scripts, put a `<script>` inside an element or in `<svelte:head>`.

```svelte
<button id="menu">Menu</button>
<div>
  <script>
    document.getElementById("menu").addEventListener("click", () => { … })
  </script>
</div>

<svelte:head>
  <script src="/js/slider.js" defer></script>
</svelte:head>
```

- Shugo copies the script body verbatim. For example, `{x}` remains the literal text `{x}`.
- Script attributes follow normal template rules. `defer` renders as `defer=""` and `src={…}` is evaluated.
- Put standalone scripts in `public/` and reference them by URL.
- Rendering a component with an embedded script ten times emits ten copies of the script.
- Svelte accepts one top-level instance `<script>` per component. A second one is an error.
- Static markup omits event bindings such as `onclick={…}`. Use an island for Svelte event handling in the browser.

Pass page data to an embedded script through attributes.

```svelte
<div id="map" data-lat={shop.lat} data-lon={shop.lon} data-places={JSON.stringify(places)}></div>
<div>
  <script>
    const map = document.getElementById("map")
    const places = JSON.parse(map.dataset.places)
  </script>
</div>
```

## Svelte islands

Import an island into a page and pass data as props. Use Svelte's runes, actions, and event handlers inside the island. The `svelte` dependency installed by `shugo` compiles both pages and islands. Islands also accept Svelte components from npm packages.

```svelte
<script>
  import Catalog from "../components/Catalog.island.svelte"
  import { search } from "shugo"

  const products = search("products", {
    text: ["title", "description"],
    fields: ["slug"],
    filters: { category: "multi", price: "range", available: "boolean" },
    sorts: { popular: { position: "asc" }, cheap: { price: "asc" } },
    defaultSort: "popular",
    pageSize: 24,
  })
</script>

<Catalog request:initial={products} action="/catalog" />
```

The `.island.svelte` suffix marks the hydration boundary for a component rendered by a server template. An island imported by another island is part of that island's Svelte component tree. A regular component such as `ProductCard.svelte` can work in both contexts if it takes data through props and uses only Svelte APIs. Call `query` from a server template and pass the result to the island.

`request:initial={products}` binds the `initial` prop to search results for the request URL. A `search` call during computation records a read of the initial results, including calls inside components and `getStaticPaths`. A call at module load records a site-wide read. A regular `initial={products}` prop passes the result from build-time computation.

At build time, Shugo derives a server-rendering program from Svelte's output. When a request arrives, Go reads the published index and runs that program, including its branches, loops, and hydration markers. Svelte attaches behavior to the resulting HTML in the browser. See [Island internals](../docs/islands.md).

### Island tag

The build enforces these rules.

- Import an island directly in a regular `.svelte` template using its default export, as in `import Catalog from "../components/Catalog.island.svelte"`. Named imports and imports from standalone modules are errors.
- Island tags accept JSON-serializable props, including `{...props}`. Put directives inside the island file. Child content, snippets, and directives on the island tag are build errors. A function prop fails during page computation.
- `request:name={…}` belongs on an island tag and takes one expression that returns a `search` result. Supplying the same prop as a regular attribute is an error.
- An island's wrapper needs a place in the document body. An island inside `<svelte:head>`, `<svg>`, `<select>`, `<option>`, `<optgroup>`, `<textarea>`, or `<template>` is an error.
- Render the island with its own tag, such as `<Catalog />`. The build rejects `<svelte:component this={Catalog}>` for an island.
- Put head markup in the page. `<svelte:head>` inside an island reports an error during page computation.

### Table search

The first argument to `search` names a table in the database configured as `data` in `shugo.json`. Shugo identifies an index by table, search columns, and projected fields. Page size, default sorting, and fixed `where` values apply to individual calls. Searches of `products` and `articles` use separate tables. Two searches of `products` with the same projection can share an index.

| Option | Behavior |
| --- | --- |
| `text` | Columns searched by the full-text query `q` |
| `fields` | Additional columns to return in `hits` |
| `filters` | `multi` accepts repeated values such as `category=cups&category=plates`. `range` accepts `price_from` and `price_to`. `boolean` accepts `available=1` or `available=0`. |
| `sorts`, `defaultSort` | Allowed values for `sort` and the default |
| `pageSize` | Results per page, 24 by default. The page number comes from `page`. |
| `where` | Fixed equality constraints, for example `{ published: true }`. The browser query searches within those constraints. |

A URL filter uses the column name by default. To name a filter differently, use a declaration such as `filters: { topic: { column: "section", type: "multi" } }`. Each row in `hits` includes the primary key and the columns used for text search, filters, sorting, and `fields`.

The result contains `hits`, `total`, `facets`, `query`, pagination fields, and `endpoint`. Visiting `/catalog?category=cups&sort=cheap` returns HTML for those parameters. After hydration, the island sends the parameters to `initial.endpoint` and updates the URL with the History API. A GET form submitted to the page URL can show the same results with browser JavaScript disabled.

`shugo serve` serves both the HTML and the search endpoint. `shugo export` writes the results prepared at build time. The [catalog and articles example](../examples/search/README.md) has two pages that use the same island to search different tables.

## Open questions

- Whether to offer a separate way to bundle browser scripts with imports on pages that use no islands.
