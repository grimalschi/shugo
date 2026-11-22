# First experiment with Svelte islands in Shugo

This report covers the first experiment, which ran a JavaScript worker for requests. The current example assembles filtered HTML in Go. [Current experiment report](GO-SSR.md)

A catalog link can include a search query, filters, sort order, and page number. The initial HTML already contains the corresponding product cards. After hydration, Svelte updates the results in place. Users can share the URL or use Back and Forward to restore the form and cards. A standard GET form handles the same parameters with JavaScript disabled.

SSR and hydration run on Shugo after it adopted Svelte 5 template syntax. The experiment starts from the merge that introduced Svelte templates. The design uses filters in GET parameters, facets, a separate query index, and a shared product-card template. The experiment renders each card in Svelte from the data returned by the index. The proposed Shugo design selects cards prepared ahead of time for each product.

[Run and verify the example](README.md)

## How it works

| Stage | What happens |
| --- | --- |
| Compilation | A direct `.svelte` import in a `.shugo` template creates an island. Svelte 5.59.1 compiles it for the server and browser. Shugo writes browser code and styles to content-hashed files and packages the SSR code in a private bundle for build-time evaluation and request handling. |
| Build and update | Go builds a private projection from the product, category, and variant tables. Evaluation reads the initial results from the index and runs SSR for the island. The store holds markup with Svelte's hydration markers and props. It records the index version in the same transaction as the page records. |
| Open `/catalog` | Shugo serves the prepared markup and props. The browser hydrates the island with them and makes zero extra requests for the initial results. |
| Open `/catalog?stock=1&sort=cheap` | Go parses the URL, queries the published index, and passes the results to a separate Bun or Node process that renders the island. Shugo keeps the page shell in the store. The first filtered request starts the JavaScript process, and subsequent requests reuse it. |
| Change filters | The island updates the URL through the History API and fetches JSON from `/_islands/catalog`. Current cards stay visible until the next results arrive. A new request cancels an older pending request. After an error, users can retry with the current URL. |

The example supports text search, multiple categories and sizes, availability, price range, four sort orders, 24 products per page, and a reset. Values in one facet match with OR, and separate facets match with AND. Each facet count shows how many products match that value and the remaining filters. The search matches each term as a case-insensitive substring of the name or description. When sort keys tie, position and ID determine the order.

Same [products card](src/components/ProductCard.shugo) imported into `.shugo` The comparison page and [Catalog.svelte](src/components/Catalog.svelte)I'm on the island. Svelte compiles clean `.shugo` The following are the components, including inputs, that are to be considered: Shugoincluding: `Shugo.query` In the input component, the build is mistaken. JSON props or APIThe island can also be placed in a regular Shugo component.

The example uses Svelte's standard `render` and `hydrate`. [Svelte documentation](https://svelte.dev/docs/svelte/imperative-component-api)

## Build and update

Each cell gives the median of five runs in milliseconds. Both versions have two pages, the same data and catalog at `/catalog`, identical initial results, and the same styles. [The static baseline](src/pages/baseline.shugo) also builds the projection, so the build time difference measures the cost of adding the island. Timing excludes database setup and dependency installation. Build and update times cover the full command in a separate process.

The environment is Linux x86-64, Intel Xeon E3-1270 v6, 8 logical CPUs, Go 1.27.4, Bun 1.4.4 and Node 24.22.1. for the launch of the bench. HTTP is a loopback network. The sequence of options was alternated by a common machine. [Build script and HTTP](scripts/bench.mjs)

| Products | Shugo / Island | Shugo compilation / Island | Shugo/island projection | Update of the price of the first item Shugo / Island | Renewal unchanged Shugo / Island |
| ---: | ---: | ---: | ---: | ---: | ---: |
| 1,000 | 64 / 404 | 18 / 349 | 14 / 13 | 43 / 50 | 9 / 14 |
| 10,000 | 173 / 588 | 18 / 405 | 121 / 124 | 155 / 162 | 10 / 14 |
| 100,000 | 1,395 / 1,768 | 19 / 394 | 1,324 / 1,324 | 1,419 / 1,337 | 11 / 19 |

Cold Svelte compilation, including bundling browser and server code, added **0.33–0.39 s**. Evaluation after compilation took 5–6 ms for the static version and 7–8 ms for the island. A price change in the initial results makes both versions rebuild the projection and reevaluate the pages. Variation between update runs at 100,000 products is comparable to the observed difference.

The index is the same in both options, 0.38 / 3.80 / 38.53 MB on disc respectively. For 100,000 products, the open index occupied about 42.6 MB Go heap in a separate post-crash inspection. The index contains a projection and a normalized text. The opening order is four sorting orders. The request is made in all the order chosen to calculate the results and the facets. The log update, if there are new entries, is still building the whole projection. If the initial results was a match, you can keep the pages. If it is different, the current implementation calculates all the pages of the project. In an experiment with the change of product on page 2 of JSON, it was updated and Shugo calculated 0 pages. The failure test for the index build checked that the previous results and HTML remained available.

## HTTP

The query uses `handmade`, two categories, an availability filter, a maximum price, and ascending price order. For each warm scenario, the benchmark made 5 warm-up requests and 60 sequential requests over one keep-alive connection. The table gives the median p50 and p95 full response time across five runs. Cold results are medians of the first request to fresh server processes. The benchmark served uncompressed responses.

| Products | Statistical p50 / p95, ms | Island without filters p50 / p95, ms | SSR with p50 / p95 filters, ms | JSON p50 / p95, ms | First filter opening, ms |
| ---: | ---: | ---: | ---: | ---: | ---: |
| 1,000 | 0.31 / 0.49 | 0.55 / 0.83 | 1.39 / 1.81 | 0.41 / 0.52 | 32 |
| 10,000 | 0.28 / 0.39 | 0.49 / 0.66 | 3.54 / 4.02 | 1.83 / 2.22 | 81 |
| 100,000 | 0.28 / 0.37 | 0.47 / 0.56 | 26.01 / 28.10 | 24.62 / 28.74 | 695 |

The first JSON requests for individual processes were 6, 54 and 670 ms. They open the index and don't run the JS-worker. The next filtered SSR in these processes, with a warm index and a cold crow, was 27, 28 and 51 ms respectively. These are different series of measurements. For big data, the main delay in the first filter visit is reading and preparing the projection.

A single run per 100,000 products compared 1 and 8 customers, with three series of 120 route requests and the number of clients after warming up, all 1,440 measured responses, HTTP 200.

| Route | Clients | p50 / p95, ms | Replies/s |
| --- | ---: | ---: | ---: |
| SSR with filters | 1 | 26.0 / 29.9 | 37.6 |
| SSR with filters | 8 | 32.1 / 42.4 | 233.7 |
| JSON | 1 | 24.7 / 28.5 | 39.5 |
| JSON | 8 | 29.1 / 35.9 | 264.1 |

Both routes perform the same search, so this load does not define the limit of the JS-worker separately from Go. For a filtered SSR with 8 p95 clients 6.5 ms above JSON.

## Page size

Dimensions in bytes for 1,000 products. Gzip is calculated for each file separately, level 6. No image or external fonts are available on the stand.

| Resources | Shugo, Byte / gzip | Island byte / gzip |
| --- | ---: | ---: |
| HTML initial results | 19,261 / 2,640 | 30,058 / 4,163 |
| CSS | 4,983 / 1,634 | 4,983 / 1,634 |
| JS | 0 / 0 | 82,499 / 30,842 |
| Total | 24,244 / 4,274 | 117,540 / 36,639 |

HTML is 10.8 KB heavier before compression and 1.5 KB after compression. These include hydration markers and JSON for 24 cards and facets. Heshed scripts and styles are re-used at the next discoveries. On the page Shugo without islands, the browser receives 0 JS. The filter change only downloads a new release. In HTTP, this is 1.0–1.6 KB JSON after gzip versus rebooting HTML. The full private index is still on the server.

## Loading and responsiveness in the browser

Chromium 158.0.8127.18, 1,000 products, and seven runs of each scenario with a fresh browser context. One profile uses a local connection. The other uses DevTools with 150 ms of latency, 1.6 Mbit/s down, 0.75 Mbit/s up, and a 4× CPU slowdown. The server index is warm for interaction tests. The catalog and static page use a cold browser cache. [Chromium script](scripts/browser-timing.mjs)

| Median | Local | Under deceleration |
| --- | ---: | ---: |
| First content diagram, Shugo / Island | 68 / 56 ms | 456 / 464 ms |
| Island is ready to interact. | 85 ms | 1,197 ms |
| Action : new results, island / GET form | 70 / 126 ms | 297 / 446 ms |
| Change of filter on the island, from input to frame with new release | 45 ms | 237 ms |

The time until the result count changes includes Playwright's wait. For the GET form, timing starts on pressing Show after selecting a checkbox with JavaScript disabled. For the island, it starts on toggling the checkbox. A separate in-page measurement tracks the count and the frame after the DOM update. With `requestAnimationFrame`, the longest frame interval in that segment was 16.8 ms. The observed layout shift on initial load was zero in every run.

With the deceleration on, the static and SSR versions are shown in about 0.46 s. Then the island has about 0.73 before the processors are ready. The scripts start when the HTML tag is discovered and are downloaded by two requests in a live stand. These are candidates for optimizing the first discovery. During filtering, cards remain visible and the network is largely waiting for this profile. The local values of the first figure are closed to account for the dispersion. The numbers refer to the synthetic stand and the emulation of DevTools.

## Functional checks

- Opening page 2 directly with `cup`, an availability filter, and a sort order produces identical card IDs and order in the server HTML, the browser, and the API. The static page and island return the same initial cards.
- Hydration passes in Chromium with Svelte's automatic mismatch recovery disabled and no errors or warnings. With JavaScript disabled, the initial HTML is readable and submitting the GET form returns filtered HTML.
- The change of categories and dimensions, price ranges, sorting and pages, dumping, back and forward. For all interactive actions, the main document has never been requested. Pagination moves the focus to the heading of the list.
- The delayed response, interrupted by the next search, is 503, repeat, empty results. Previous cards remain on the screen while waiting and error. The HTML search line passes SSR and hydrated as text.
- Two copies of the counter on one page are changed independently and use one tag module. The normal Shugo page does not receive a browser code. 320 and 390 pixels do not have horizontal rolling.
- The change of `.svelte` results in a re-build over `shugo update`. After style changes, export takes files from the active build manifest, and the launch server can still give old hashed resources.

The browser scenario was run from SSR on Bun and Node. Firefox and Webkit were missing from the car, so the browser conclusions are for Chromium. [Functional audit scenario](scripts/browser.mjs)

## Future work

1. **Save the narrow border of the island.** Form, status of request, results and transitions are in the same component. Clean cards can be used on both sides. In the experiment, the border sets direct imports and JSON props. `search()` of `shugo:catalog` declares a business index dependency. `request:initial={search()}` is tied to a specific Prop of the island's UrL request.
2. **Connect the index to the log at the product level.** This will allow the main promise of the scheme to be verified from the discussion. The price change affects the item and data of one product. The expensive full passage, when updating and calculating all pages when changing the initial results, should be replaced by an update of the projection of the products and an exact dependence of the records on the initial issue. For text, sorting and faset, it is useful to compare SQLite FTS with the index by field with the set of inverted lists. The cost of measuring them is 100,000 products, with a change in the line and parallel requests.
3. **Remove the delay of the first discovery.** Warming the index to server readiness or format from which only the right parts can be read is more important than microoptimizations of Svelte for a catalog of this size. If the request SSR is retained, check the separate JS warm-up and several simultaneous references. Now one process serves SSR requests in sequence. The index and bandages are stored as long as they can be read by old pages or processes. A long-lived server needs a way to remove the versions after the links have been completed.
4. **Optimize first visit to browser measurements.** has a single cost of about 31 KB gzip at the total Svelte rebound. The next steps are to compress the publication, check the early upload of the module and compare separately the single band and the common cup. Check the dimensions of JSON on real cards with images and attributes.

## Limitations and open questions

At `shugo export`, the directory receives files, HTML initial results and scripts for hydration. When publishing only on file hosting, the different URL settings open the same file. For the filtered HTML in the first response and for JSON on the stand, `shugo serve` is required. Filtered responses are marked by HTTP with `noindex` and `canonical` to the catalog address. The indexing settings of the listings and the selection pages remain a product solution.

The island has one wrapping at HTML and a simultaneous SSR. External snips and functions are not transmitted via props, metadata are requested on the Shugo page. Several copies of the `$props.id()` component will require a unique SSR-prefix to be used, including when the complete common block is repeated. The meter on the stand is without this rune.

Open issues for next step

1. Should the reference directory give the filtered HTML and operate without JS in each location? If there is sufficient browser filtering at the static hosting, API can be removed separately and applied URL after hydrating. This removes the JS-worker from the public Shugo server, and direct opening of the link first shows the first results.
2. How does the author announce the list and projection? Options are the description of the search list in Shugo and the reference to an island by name, or a separate base adapter. To move the pilot idea into the main project, it is important to maintain several independent catalogues with different schemes.
3. Do you have a card on the island from JSON, like now, or are you keeping the cards ready separately, as suggested in the discussion? The first version provides one Svelte-sable for SSR and updates. The second one should be compared separately by the size of the response, the cost of updating and the ability to save the keys of DOM when switching the filters.
4. What are the parameters that should have their own URL indexed and canonical address? Do you want to decide whether the catalog pages, categories and editorial sets are considered as separate landing pages?
5. Do you need a search for each input, late and `replaceState`, or do we keep a separate confirmation for text and price? The chosen experiment mode creates history record for the application only.
6. Without this, it is easy to optimize the launch speed of a small SSR island when users of a larger store are poking themselves into reading a search projection.
