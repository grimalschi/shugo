# Svelte islands

An island is a `*.island.svelte` component embedded in a static page. The server sends rendered HTML, and Svelte hydrates it in the browser by attaching event handlers to the existing markup. In the main example, a catalog stores its filters in the URL. A link opens the prefiltered catalog, and changing filters in the browser updates the results in place. [Svelte islands usage](../spec/scripts.md#svelte-islands)

## Decisions

- The `.island.svelte` suffix marks an island entry point. This was approved on November 22, 2026.
- On November 22, 2026, Shugo bundled the Svelte compiler and its server and browser runtimes at the same version. The project author changed this on November 23, 2026. The `shugo` package now installs the pinned `svelte` dependency.
- Go renders the island HTML for filtered URLs. During the build, Shugo prepares a rendering program from Svelte's SSR output. The project author approved this on November 22, 2026, to keep JavaScript out of request handling.
- On November 22, 2026, `search("table", { … })` from `shugo` replaced the experimental `search()` API from `shugo:catalog`, which used a single index tied to the shop schema. Shugo derives index names from the search declaration.

## How it works

1. **Compilation.** The pinned Svelte compiler compiles the island's dependency graph twice in runes mode, once for the browser and once for the server. The build bundles and hashes the browser module and its styles and writes them to `_build/assets/`. Both builds include imported npm packages. Shugo compiles Svelte components from those packages with the same version of Svelte and resolves their `svelte` imports to that version. If the call site binds a prop to request data with `request:initial={…}`, Shugo compiles Svelte's server output into an island SSR program. That program contains HTML, hydration markers, text and attribute substitutions, branches, loops, and nested components. Substitutions and conditions use JSON field paths and constants. The site skeleton holds the programs in `islandPrograms`.
2. **Evaluation.** `eval.mjs` evaluates the page and calls the island's Svelte server renderer with its props. It leaves a placeholder in the page record with the island's module, styles, props, rendered HTML, and request bindings, such as `{"initial":"search:<search-id>"}`. The rendered island keeps its hydration markers. The page supplies `<head>` content. `<svelte:head>` in an island fails evaluation. Before it writes the site records, Go checks that the island SSR program renders the same bytes as Svelte. A mismatch stops publication.
3. **Request.** An ordinary request uses the island HTML from the page record. If the URL supplies parameters for a search bound to the island, Go gets the search index version from the same store transaction as the page. It queries that index and runs the island SSR program with the results. The same props go into the response, so hydration can use them directly. The response includes `X-Robots-Tag: noindex` and a `rel="canonical"` link to the URL without search parameters.
4. **Browser.** Svelte's `hydrate` attaches to the existing DOM. After a filter changes, the island updates the URL with the History API and fetches JSON from `/_islands/search/<search-id>`. With JavaScript disabled, the form submits a regular GET request and the server returns filtered HTML.

A regular component, such as a product card, can appear in both the page and an island's dependency graph. Svelte compiles it for the server as part of the page, and for both the browser and the server as part of the island. The component takes its data through props and uses Svelte APIs. Its caller can pass query results or page metadata as props.

## Island SSR program

The program reads JSON fields, evaluates boolean conditions, iterates over arrays, escapes values for their HTML context, and renders nested components. Expressions such as `{price.toFixed(2)}` or `{new Intl.NumberFormat(…).format(price)}` fail the build when they appear in request-rendered markup. Pass the display value in a projected column or as a precomputed prop. The browser portion uses standard Svelte.

Shugo compiles the program from Svelte 5.59.1 server output. When upgrading Svelte, rerun the HTML comparison between the program and Svelte itself.

## Search

`search("products", { … })` declares a search over a table in the project's database. Shugo writes the SQLite FTS5 projection to `_build/search-<hash>.sqlite` and applies English stemming to search terms. The index name comes from the table, the full-text columns, and the columns used by the projection and its lookups. Two bindings with the same physical specification can share an index with different `where` values or `pageSize` settings.

- The store publishes the new index generation and page records in one transaction. Existing index files stay available to a server reading an older generation.
- The first result set is a tracked read by the page, shared block, or route page list that uses it. An update reevaluates those readers when the first result set changes. A search run at module scope is a dependency of every evaluation.
- The index and pages read the source database through different connections. If the database changes between those reads, the update retries up to three times so it can publish a consistent state.
- Changes to columns outside the projection leave the index alone. Changing a full-text, filter, or sort column currently rebuilds the whole table's index. With 100,000 products, the rebuild takes about 1.2 s.

## Approaches considered

The November 22, 2026 comparison looked at ways to render islands for filtered requests without running JavaScript on the server.

| Approach | Result |
| --- | --- |
| Compile a markup program from Svelte's SSR output | This is the approach Shugo uses. Svelte handles whitespace, branching, and hydration markers. Shugo translates the output into a small set of operations on data. The catalog program, including the product card, has 123 operations and is about 8 KB. |
| Split native SSR output into fragments | All 88 checks matched Svelte, and the fragments took about 7 KB. The splitting rules were specific to the catalog structure and would need to change when the markup acquired a new branch. |
| Pre-render every product card | Rendering 100,000 cards took about 1.1 s and produced 39.4 MB of HTML. Pre-rendering could help with expensive cards. Facets, empty states, and pagination still require a program, and 50 filter checkboxes allow up to 2⁵⁰ combinations. |
| Generate Go code from Svelte's SSR output | [SvelteGo](https://github.com/binsarjr/sveltego/issues/421) parses the output with Acorn and generates Go functions compiled into the application. The same Shugo server serves different sites, so the rendering program is stored as data. |
| Implement a template compiler in Go | A compiler like [templ](https://templ.guide/) would need to reproduce Svelte's rules for markers, whitespace, and attributes. |
| Embed JavaScript in Go | [Golte](https://github.com/nichady/golte) runs SSR in Goja. Other implementations use QuickJS or Wasm. These approaches run JavaScript for each request. |
| Ship initial results and apply filters after hydration | This works with files on a CDN. A filtered URL initially shows the default results, and a GET form needs the server to filter the response. |

## Benchmarks

The November 22, 2026 [two-table search example](../examples/search/README.md) used 100,000 products. The site built in 1.76 s, the product index was 11.6 MB, and the browser module was 74 KB. Median HTML response times were 0.25 ms for default results, 8.35 ms for a rare term with filters, and 43 ms for a common term with filters. JSON for the rare term and the same filters took about 7 ms. Updating a column outside the index took about 12 ms.

In the experimental catalog with 1,000 products, Chromium with network and CPU throttling displayed content after 0.46 s for both the static page and the island. The island became interactive after 1.13–1.20 s. An island filter change took 297–386 ms. Submitting the GET form took 446–486 ms. The island added about 30 KB of gzip-compressed JavaScript to the initial load.

## History

- **First island implementation, November 21, 2026.** A separate Bun or Node process ran during the request to render filtered results. The project author chose to keep JavaScript out of server request handling. Switching to the Go rendering program removed the process startup and reduced the first filtered request for 100,000 products from 678 ms to 654 ms. Opening the index was the main cost.
- **First search implementation, `search()` from `shugo:catalog`.** It indexed the shop schema, including products, categories, variant sizes, formatted prices, and stock labels. It applied price changes directly to the index from the journal. On November 22, 2026, `search("table", { … })` replaced it. The old `examples/islands` demo was removed, and the standalone counter islands moved to `examples/search`.
