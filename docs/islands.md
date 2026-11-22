# Svelte islands

The island is a component of `*.island.svelte` inside a static page, which also works in the browser. HTML island comes from the server ready, and in the browser it is hydrated by the real Svelte: connects handlers to the already built markup. The main case is a directory with filters that live in the address: a link with filters immediately opens the filtered list, and the filter change in the browser works without rebooting. How to use it in [Scripts on page](../spec/scripts.md#svelte-islands).

## Decisions

- The island is marked with the suffix `.island.svelte` (22 solution of November 2026 year).
- JavaScript is not available on the server. HTML collects Go for the filtered address according to the program that the assembly prepares from the output of Svelte (the decision of the author of the project 22 November 2026 year: "in the morning should not be JavaScript").
- Search — `search("table", { … })` from the module `shugo`; the name of the Shugo index displays itself (22 solutions November 2026 year). The first version of the experiment, `search()` from `shugo:catalog` with one index according to the store scheme, was removed on the same day.

## How it works

1. **Compilation.** Svelte 5.59.1 compiles the island graph twice: for the browser and for the server. The browser module and its styles are placed in `_build/assets/` under the hash names. If the island tag links the parameter to the query data - `request:initial={…}`, - Shugo parses the server output of Svelte and makes it the SSR program of the island: ready HTML, hydration tags, text and attributes inserting, branches, repetitions and calls of nested components. The inserts and branches have only the paths to JSON fields and constants. The software is located in the framework of the site, `islandPrograms`.
2. **Calculation.** `eval.mjs` calculates the page as usual, and in place of the island calls the server render of Svelte project with page parameters. In HTML, the island remains a hole with JSON island: its module, styles, parameters, HTML and binding parameters to query data, for example `{"initial":"search:<search name>"}`. The hydration marks in HTML island remain: on them the island comes to life in the browser. Before recording, Go checks that the island program gives the same HTML as Svelte, byte byte; divergence stops publication.
3. **Request.** The usual request is received by HTML island from record. If the island is linked to a search and the address has the search parameters, Go takes the index version from the same transaction of the storage as the page, reads the issue and collects the HTML of the island again on the program. The page gets the same parameters that the program worked with: the browser does not need to ask for a return. This page responds with `X-Robots-Tag: noindex` and a link `rel="canonical"` to the address without parameters.
4. **Browser.** `hydrate` the City of Svelte Connect to Ready DOM. When changing the filter, the island changes address via History API and takes the next issue of JSON-om `/_islands/search/<search name>`. Without JavaScript form sends a regular GET, and the server responds with a filtered HTML.

A regular component, such as a product card, can enter both the page and the island graph: in the page it compiles it for the Svelte server sewn into Shugo, and in the island - Svelte of the project of the same version, for the browser and for the server. To do this, he receives data parameters and uses only means Svelte - without `query` and `page`.

## What does the SSR program of the island

The program takes JSON fields, Boolean conditions and arrays: inserts values with shielding in place, selects branches, repeats lines, calls nested components. There are no expressions in it. Therefore, the markup that the server collects on request cannot count: `{price.toFixed(2)}` or `{new Intl.NumberFormat(…).format(price)}` in it is an assembly error. What you need to show should come with data: a table column or a ready value in the parameters. The browser part of the island is a regular Svelte without restrictions.

The program compilation is based on the server output of Svelte 5.59.1. When switching to another version of Svelte, you need to re-run the comparison of HTML program with HTML Svelte.

## Search

`search("products", { … })` describes a search for the project database table. The index is a SQLite file with FTS5, `_build/search-<hash>.sqlite`. The index name is derived from the table, search columns and fields of issue, so two searches for the same table with the same fields share the same index, even with different `where` and `pageSize`.

- The index is built and published together with page records, one transaction of storage. Files of previous versions are not deleted: they can be read by the server, which still shows the previous state.
- The first search results are read by the person who showed it: a page, a common block or a list of route pages. The update calculates them again when the first issue has changed. Search at the top level of the module belongs to all calculations.
- Index and pages read the database by different connections. If the database has changed between these readings, the update starts anew, up to three times, and publishes one state of the data.
- Changing the column that the index does not hold does not touch it. Changing the search column, filter or sorting while building the table index anew: on 100,000, the goods are about 1.2 s.

## Methods considered

How to assemble HTML islands on demand without JavaScript on server, compared 22 November 2026 year.

| Method | What the test showed |
| --- | --- |
| The markup program from the server output Svelte | Selected. Svelte solves the spaces, branches and hydration tags itself; Shugo translates its output into a small set of operations on data. The directory program with the card is 123 operations, near 8 KB. |
| Cut real SSR into finished pieces | Coincident with Svelte in all 88 checks and took about 7 KB, but the fragments had to be described by the structure of the catalog manually: a new branch in the markup requires changing the rules of slicing. |
| Prepare HTML of each card in advance | 100,000 cards are 39.4 MB HTML and around 1.1 s. It is suitable as an acceleration of expensive cards, but facets, blank issuance and pages still require a program; you can not override all combinations of filters: 50 marks give up to 250 combinations. |
| Generate Go-code from server output Svelte | So does [SvelteGo](https://github.com/binsarjr/sveltego/issues/421): parses output through Acorn and releases Go-functions that are compiled into the application. In Shugo, the server is common to any sites, so the program is stored with data, not code. |
| Your template compiler right in Go | Like [templ](https://templ.guide/). You would have to repeat the rules of labels, spaces and attributes of Svelte yourself. |
| JavaScript engine inside Go | [Golte](https://github.com/nichady/golte) launches SSR in Goja, others in QuickJS or Wasm. This is JavaScript on a request that was rejected. |
| Only the first issue, filters - after hydration | Suitable for files on CDN, but direct link first shows the wrong, and GET form without JavaScript does not filter. |

## Measurements

Search for two tables from [examples/search](../examples/search/README.md), 100,000 goods, 22 November 2026 Year: Site assembly 1.76 s; index of goods 11.6 MB; island browser module 74 KB; HTML by default - 0.25 ms p50, with rare word and filters - 8.35 msWith a frequent word 43 ms; JSON The same request 7 ms; column change outside the index - about 12 ms.

The experiment catalog on 1,000 goods, Chromium with a slow network and processor: the contents are visible through 0.46 s with both a static page and an island; the island is ready to work through 1.13–1.20 s; filter change in the island is 297–386 ms against 446–486 ms in the GET form. The island adds to the first load around 30 KB JavaScript after gzip.

## History

- **The first version, 21 November 2026 year,** renders the island with filters by a separate process Bun or Node during the query. The author of the project refused it: the server should not be JavaScript. The transition to the Go program reduced the first request with filters for 100,000 products from 678 to 654 ms - the main time there is the opening of the index - and eliminated the launch of the JavaScript process on the first appeal.
- **First search, `search()` of `shugo:catalog`,** built one index according to the store scheme: goods, categories, sizes from options, ready-made price signatures and availability, and updated the price directly in the index by magazine. 22 November 2026 replaced it with a total `search("table", { … })`, and the `examples/islands` stand on the old search is removed; simple islands without search moved to `examples/search`.
