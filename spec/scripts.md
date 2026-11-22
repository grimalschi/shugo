# Page scripts

The document describes the code that is executed in the visitor's browser.

## Decisions

- `<script>` inside the markup gets into the page as written; the curly brackets inside it are plain text (21 decision of November 2026 year).
- The basis for the code in the browser is Svelte (21 solution November 2026 year).
- The `*.island.svelte` component becomes a hydration point when it is shown by a pattern: it is an island. The file name is visible in the source and import (22 November 2026 solution). The islands are made before the data adapter, although the November 22 decided to take them after it.
- Svelte for islands is sewn into Shugo, the browser part of the same version as the compiler: the project with the islands does not need to put anything (the decision of the author of the project 22 November 2026 years).
- When a JavaScript page is requested, the HTML island for the filtered address collects Go on a program that the assembly prepares from the server output of Svelte (22 November 2026 solution).

## Implemented, pending approval

The top `<script>` of a regular `.svelte` file is performed when calculating the page. For small browser scripts, you can use `<script>` inside the element or in `<svelte:head>`:

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

- The contents of such `<script>` fall into the page as written. Expressions in it are not calculated: `{x}` remains the text of `{x}`.
- The attributes of the tag itself are common attributes: `defer` is displayed as `defer=""`, `src={…}` is calculated.
- The script files are put into `public/` and connected to the address.
- A component with a script that is shown ten times will give ten identical scripts.
- The second `<script>` at the top level of the file is the Svelte error: the top `<script>` of the component one, and that's its code.
- Event handlers in markup (`onclick={…}`) do not get into the page: there is no one to perform them on the static page.

The value from the data is transferred to the script via the attribute:

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

The page imports the island by name and passes it data as parameters. The file can contain standard runes, actions and Svelte handlers. The island compiles the same Svelte as the pages sewn into Shugo along with its browser part: you don’t need to put anything. The island, like any template, is written on runes. The island is a browser code: it can wait (`await`) in the functions and at the top of the modules that only it imports, and it runs the components of Svelte from the npm packages, which are an error on the pages.

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

The suffix selects the boundary of hydration. A subsidiary of `*.island.svelte` inside another island is included in its tree Svelte. The regular `ProductCard.svelte` can be imported into both the server template and the island if the component takes data through the parameters and uses only Svelte tools. `query` from the `shugo` module is asked in a normal server template that transmits the result to the island.

`request:initial={products}` binds the parameter `initial` to the output of URL. The `search` call registers the reading of the initial issuance during the calculation, including calls within components and `getStaticPaths`. The call when loading the module registers reading for the entire site. The normal parameter `initial={products}` transmits the result of the calculation step.

When assembling Shugo prepares the program SSR from the output of Svelte. On request Go reads the published index and applies the program, including branches, lists and hydration tags. In the browser Svelte connects handlers to the received markup. [How the Islands Are Made](../docs/islands.md)

### Tag island

What the assembly checks:

- The island is imported by default, right in the usual `.svelte` template: `import Catalog from "../components/Catalog.island.svelte"`. Named import and import not from the template is an error.
- The island tag only transmits data: values that are translated into JSON and `{...parameters}`. Content, snippets and directives in the tag are error: directives are written inside the island file. Function in parameters is error when calculating the page.
- `request:name={…}` stands only on the island tag, with one expression - the result of `search`. The same parameter is also a common attribute - an error.
- The island inside `<svelte:head>`, `<svg>`, `<select>`, `<option>`, `<optgroup>`, `<textarea>` or `<template>` is an error: its wrapper needs a place in `<body>`.
- The island is shown with its tag: `<svelte:component this={Catalog}>` is an error.
- `<svelte:head>` in the island itself is a mistake when calculating a page: what you need in `<head>`, writes the page.

### Search by table

The first argument of `search` calls the table in the `data` database from `shugo.json`. Shugo displays the index identity from the table, search columns and projection fields. Page size, default sorting and fixed `where` values refer to a specific call. `products` and `articles` searches read their tables, and two `products` searches with the same projection can use the same index.

| Parameter | Conduct |
| --- | --- |
| `text` | Full-text search columns by `q` |
| `fields` | Additional columns in `hits` |
| `filters` | `multi` reads repetitive values, e.g. `category=cups&category=plates`. `range` reads `price_from` and `price_to`. `boolean` reads `available=1` or `available=0` |
| `sorts`, `defaultSort` | Permitted variants of `sort` and the default option |
| `pageSize` | Number of results per page, default is 24. Page number is passed as `page` |
| `where` | Fixed equality, e.g. `{ published: true }`. The request from the browser is applied within this area |

By default, the URL filter name matches the column. You can set your own, for example, `filters: { topic: { column: "section", type: "multi" } }`. In addition to `fields`, the `hits` string includes the primary key and text columns, filters and sorting.

The result contains `hits`, `total`, `facets`, `query`, pagination fields and `endpoint`. The `/catalog?category=cups&sort=cheap` page immediately gives the HTML for these parameters. After hydration, the island sends the same set of parameters to `initial.endpoint` and updates the address via History API. The GET form at the page address allows you to open the same issue without JavaScript in the browser.

`shugo serve` serves both HTML and search endpoint. With `shugo export`, files get an issue prepared during assembly. In the example there are two pages with a common island and search for different tables. [Example of catalogue and articles](../examples/search/README.md)

## Open questions

- Do you need a separate way to collect browser scripts with imports for pages without islands?
