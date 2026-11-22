# Svelte island with server-side HTML rendering in Go

A direct link with filters returns HTML showing the matching products. `shugo serve` reads the index and executes the island's markup program, which Shugo generates at build time. Svelte hydrates the resulting DOM in the browser and updates the results when filters change. The GET form works with JavaScript disabled.

This experiment builds on the [first version with a JavaScript worker](REPORT.md). The new server handles the requests in Go. [Run the example](README.md)

## The boundary between Go and Svelte

```mermaid
flowchart LR
    A["Catalog.island.svelte and ProductCard.svelte"] --> B["Svelte compiler at build time"]
    B --> C["Browser bundle"]
    B --> D["Svelte server output"]
    D --> E["Markup program in the Shugo store"]
    D --> F["Initial results and reference HTML during evaluation"]
    G["URL → index → result data in Go"] --> H["Go renders the island"]
    E --> H
    H --> I["HTML + the same data"]
    C --> J["Browser hydration"]
    I --> J
```

1. **Compilation.** Svelte 5.59.1 makes a browser and server build of the island. The `request:initial={search()}` link on the island tag means that Shugo can be used to frame the request in prop `initial`. For these calls, Shugo reads the server output tree and extracts HTML, hydration tags, text and attributes, branches, repeats and challenges of the input components. The boxes and branches have links to JSON fields and constants. The programs are recorded in `site.json` and then in the page repository. [Compilation of Svelte](https://svelte.dev/docs/svelte/svelte-compiler) · [Version 5.59.1 labels](https://github.com/sveltejs/svelte/blob/svelte%405.59.1/packages/svelte/src/internal/server/hydration.js)
2. **Page evaluation.** When the evaluation bundle calls `search()` from `shugo:catalog`, Shugo prepares the index from SQLite. The same function renders results in a static `.svelte` page such as `/baseline`. The standard `eval.mjs` on Bun or Node computes the props, and Go compares the markup program's output with Svelte's native SSR. A mismatch prevents publication. The store holds the parameter-free HTML, props, and bindings such as `{"initial":"catalog"}`.
3. **Filtered request.** Go reads the page and index version from the same store snapshot and runs the search. Go uses the results to fill the bound prop. The markup program selects branches and loops and escapes inserted values for their context. The response contains the HTML with hydration markers and the same props that the program used.
4. **Browser.** Svelte's `hydrate` attaches to the existing DOM. When users change filters, the island updates the URL and fetches the next results from the JSON API. The browser handles events, history, and requests. [Svelte hydration](https://svelte.dev/docs/svelte/imperative-component-api#hydrate)

The render program for the catalog and card contains 123 operations in a compact representation of about 8 KB. One program covers any number of pages and filter combinations. Both island bundles use `ProductCard.svelte`, and ordinary Shugo pages can use it too. The live route shares this card template.

## Approaches considered

| Method | What the test showed |
| --- | --- |
| Compiling Svelte server output into the markup program | This option is implemented. Svelte compiler already solves the details of the gaps, branches and markers for the browser. Shugo converts its output into a limited set of data transactions. It works with the storage facility and published pages. |
| Cut the real SSR on finished fragments | The test catalog with fragments for the marked facets, cards and pagination was similar to SSR Svelte in all 88 checks. The cluster of fragments took about 7 kB. They had to be collected by manually described catalog structure. The new branch in the markings requires a change of extraction rules. |
| Prepare HTML each card in advance | In the sample, 100,000 cards were taken by 39.4 MB HTML The generator took about 1.1 with the process warmed up. SvelteThe way you optimize expensive cards is to optimize the cost of your cards, and the facets, empty issuances and pagination still require their program. It's impossible to overwrite the pages that are finished, with only 50 independent marks being given to the full. \(2^{50}\) The combination before the text and price. |
| Generate Go Sources from Svelte server output | The closest analogy is SvelteGo. He also takes the official SSR output, he reviews JavaScript through Acorn and recognizes `$$renderer.push`, `$.escape` and Svelte cycles. Its JSON AST is used for build, then the generator releases Go functions to compile into an application. In Shugo, the build retains a markup program in the storage facility and the general Go server reads it when publishing. SvelteGo has a clear Node-render for selected routes. [RFC SvelteGo](https://github.com/binsarjr/sveltego/issues/421) · [Implementation](https://github.com/binsarjr/sveltego/pull/444) |
| Compie your own templates in Go | the tablet compiles Go-expression original templates in Go-function. Svelte for Shugo would require self-repeating of tags, gaps and attributes. [templ](https://templ.guide/) |
| Save the template & intermediate program | Glimmer compiles Handlebars-stables into serilyized `WireFormat`This is the same as the format of the program, which is kept separate from data. [Glimmer Precompiler](https://github.com/glimmerjs/glimmer-vm/blob/main/guides/02-precompiler-overview.md) |
| Start SSR inside one Go-binarnic | Golte uses Goja, projects with QuickJS/Wasm also retain the JS engine during HTTP request. This option is not available for a given boundary. API - Yes. Svelte It also involves the launch of a compiled component. [Golte and Goja](https://github.com/nichady/golte/blob/master/go.mod) · [Example SSR via QuickJS/Wasm](https://github.com/bennypowers/lit-ssr-wasm) · [Svelte custom renderers](https://svelte.dev/docs/svelte/custom-renderers) |
| Save only the initial results and apply URL after hydration | This option is suitable for the publication of files on CDN. The direct link first shows the initial results, and the GET form on the file hosting cannot filter the answer. From existing projects, Planti uses Go and Svelte to pre-build static pages. [Plenti](https://github.com/plentico/plenti) |

The selected program takes almost as much as the sample fragments and is generated from existing markings automatically. The difference is particularly important when changing the template: the author changes the Svelte component, and the build checks that Go can show it with the same tags.

## Verification

- **Server isolation.** The verification script deploys the Go binary, published store, index, and browser assets. It uses an empty `PATH` and points the configuration at a missing JavaScript runtime. The deployed files omit the source database, templates, `eval.mjs`, and `node_modules`. Shugo serves every request without child processes.
- **Byte coincidence.** is checked against each of the 12 sets of parameters by HTML of the catalog and a separate test island with `if → each → if → each → if / else if / else` and a sub-card call. All branches, including empty results, products without size and different conditions of discount and availability, have been passed on to the test island. Among the queries are long page, combined facets, price 0 and text with HTML symbols. Props in HTML are the same as JSON API. The verification is performed after build and with Bun and Node. The `.toFixed()` call immediately stops the build at the request mark.
- **Hydration.** Nine Chromium catalog scenarios passed against the isolated Go server with Svelte's automatic recovery disabled. With the bundle deliberately delayed, hydration preserved the title and first card DOM nodes that the browser had created from server HTML. Loading made no duplicate request for the results. A separate test island also retained the DOM node in an inner loop.
- **Interactivity.** Changing categories, sizes, price range, sort order, pages, or browser history preserves the current document. A late response cannot replace a newer one, and users can retry a failed request. With JavaScript disabled, the form submits a GET and the first response shows the filtered list and the control state.
- **Publication with a server running.**, a new binarium with an empty `PATH`, opened the vault from the previous version. With the failed upgrades, he continued to give the catalog. After renaming the templates and changing the product, `shugo update` published a new index, HTML and a client module. All 224 parallel requests were fully old or new during the update. The old hash resource is still available.
- **Precision updates.** Two routes that show the common block with `search()` receive a new HTML by counting the single block. The page reading another field via `Shugo.query` is updated separately. The change of the product outside the initial results changes the published index and filter answers. In the 100,000-product stand, 21 consecutive updates and checks were performed on all 104,203 exported non-construction files. [Dependency tests](../../test/island_migration_test.go) · [Table script](../catalog/stand.mjs)

## Limitations

The Go program reads JSON fields, booleans, and arrays. The Go projection prepares values such as formatted prices, selected facets, page bounds, and pagination URLs. Arbitrary calculations in server markup cause a build error. Browser handlers still use Svelte's features. For example, server HTML sets `selected` on sort options, and a browser effect restores the field on history navigation.

The compilation depends on the fixed server output of Svelte 5.59.1. When updating Svelte, it is useful to run byte comparison and browser test before changing version. The format of the old worker experiment requires a re-build of `shugo build`.

The counter uses a store layout and one search source. `request:` links the island's parameter to this source. The page can be placed under another URL and transferred to prop `action`. When changing the price or discount Shugo applies the log to a copy of SQLite image, while saving FTS5. The changes in the text, the facets and the catalog composition recapture the images. The cost of creating the index and the cost of implementing the SSR programme can be measured separately. [How changes are monitored](../../docs/change-tracking.md#initial-search-results)

## Benchmarks after the switch to `.svelte` and `.island.svelte`

We compared the first SQLite FTS5 catalog build with the version that uses the new extensions. Both used the same products, SQLite FTS5, and a Go server. Each size used three fresh builds and runs of 24 loopback requests. The table gives medians for the previous and new versions in that order. Build time includes index creation. [Benchmark script](scripts/bench.mjs)

| Products | Collection of the island site, ms | Compilation, ms | Price update, ms | Warmed TTFB on p50 / p95 filter, ms |
| ---: | ---: | ---: | ---: | ---: |
| 1,000 | 464 → 466 | 381 → 379 | 90 → 81 | 2.47 / 3.43 → 2.54 / 3.33 |
| 100,000 | 5,007 → 5,029 | 375 → 375 | 4,809 → 4,563 | 59.4 / 65.8 → 59.0 / 83.9 |

The initial results of the price update recalculated one page. The two pages and the block were previously re-readed. This is a measurement made before the journal's price update. With 100,000 products, 4.52 s out of 4.56 seconds, the creation of a new FTS5 image was then taken. The eight simultaneous requests for this size p95 increased from 376 to 408 ms. After the p95 was passed, the three runs were between 64 and 88 ms.

After the change to the journal price update, separate measurements were repeated on the same data. The median of the three `shugo update` teams for the price of one item was 37 ms instead of 101 ms per 1,000 products and 221 ms instead of 4.73 s per 100,000. Of the 221 ms, the catalog took 189 ms. The change in line uses the finished FTS5, and the publication continues to copy and store the 35.6 MB index file.

The initial dimensions of HTML and the resources were in the same order. The catalog weighs 4,784 bytes per 100,000 items after gzip HTML, while the client JS weighs 29,735 bytes. In a static catalog with the same HTML cards, 2,651 bytes after gzip, the browser JS is missing.

## Comparison with the worker version using the previous JSON index

Two versions are collected from separate copies of the originals with the same determinized base. Each size consists of five fresh assemblies and five series of 60 warm consecutive requests for the route. Requests were made for loopback. Go 1.27.4, Bun 1.4.4 in the build phase, Intel Xeon E3-1270 v6, eight logical CPUs. Time in milliseconds. Each pair first has a worker version, then Go. [Measurement scenario](scripts/bench.mjs)

| Products | Build whole | First filter transition | First transition after the JSON index is warmed up | HTML p50 | HTML p95 |
| ---: | ---: | ---: | ---: | ---: | ---: |
| 1,000 | 399 → 444 | 30.6 → 6.6 | 25.7 → 2.0 | 1.39 → 1.32 | 1.71 → 1.71 |
| 10,000 | 516 → 565 | 85.0 → 54.9 | 27.5 → 3.4 | 3.55 → 3.31 | 4.26 → 4.35 |
| 100,000 | 1,735 → 1,807 | 678.0 → 654.3 | 50.0 → 25.6 | 25.92 → 25.62 | 28.50 → 27.63 |

Some 32–52 ms of additional build was produced. In an already operational server, about 24 ms was lost, which the first HTML request spent on launching a worker. For 100,000 products, the first reading of the 38.5 MB index still takes about 656 ms, as seen in the first JSON request. After warming up JSON in the same scenario, it is 24.62 ms mid-point. For this data, the transition to Go reduces the pivot dependency and cold start of the render. Searching takes most of the time of the HTML response.

The first page per 1,000 items grew from 30,058 to 38,275 byte HTML, after gzip from 4,163 to 4,767 bytes. Go began to give props ready signatures, links and flags. JS has decreased from 30,842 to 29,754 byte gzip. Together with the total CSS, the first load in total decreased from 36,639 to 36,155 bytes gzip. JSON has the same filtered request, and after gzip, the weight has increased from 1,010 to 1,390 bytes per 1,000 products and from 1,538 to 2,258 bytes per 100,000.

For 100,000 products with eight customers, the filtered HTML changed from 35.65 to 31.30 ms p50, from 61.23 to 40.90 ms p95 and from 194 to 239 responses/s. JSON API, without a render p95, changed from 37.38 to 45.83 ms in the same series. Due to the dispersion and the new format of JSON, capacity increases should be considered as an observation for this load. The steady win is the absence of a JS-tracker launch in the first round.

In Chromium, the first drawing for 1,000 commodities was 68 ms for the static page and 56 ms for the island locally, 452 ms and 472 ms for deceleration. The island is ready to interact through 64 ms and 1.13 cc, respectively. The filter change in the slow profile took 386 ms, and the GET form took 486 ms. The first version of the separate browser race was 297 ms and 446 ms. The network was transmitting uncompressed answers. The growth of the body size of JSON is measured, and the old and new browser runs are also different from the updated counter-ol model. Seven script launches, 150 ms delay, 1.6 Mbit/s down and 0.75 Mbit/s up, CPU slows four times. [Chromium script](scripts/browser-timing.mjs)
