# Data change tracking

Shugo tracks changes to the project's data and reevaluates only the parts of the site that depend on those changes.

## Decisions

The project author made these decisions on November 20, 2026.

- Data change tracking is required. The options were to track changes in Go between loading data and evaluating expressions, or in TypeScript without an intermediate stage.
- Evaluate expressions only in the parts of the site affected by a change.
- When one product changes, reevaluate only the pages that display its data and the blocks that display it elsewhere.
- Ordinary project code must produce fresh pages without requiring authors to tag inputs with calls such as `source(...)`.
- The original design called for a dependency tree that analyzed how templates use variables and data. The approved final version records actual reads on each row, as in variant 5.
- The first proposal allowed templates to receive copies of data variables through serialization. The author suggested Seroval. The final implementation dropped this requirement along with the first proposal.
- The way data reaches pages and components can change to make comparisons simpler and faster. The goal is efficient updates.
- The author approved a journal maintained by database triggers for variant 2 on November 20. The journal records inserts and deletes as well as updates, and authors do not need to maintain an `updated_at` field.
- The SQLite copy of the data is part of the architecture. See the [data adapter proposal](proposals/data-adapter.md).
- The team set aside variant 1, which used fingerprints without a database, after the database variants performed better.

The author made these additional decisions on November 21, 2026.

- The [final version](#final-implementation) based on variant 5 was approved.
- Rows follow `order()`. Rows with equal sort keys and rows from queries without `order()` are ordered by key. This keeps Shugo's automatically created indexes from changing the row order on existing pages.

## Final implementation

The project author asked to remove the remnants of the original design on November 21, 2026. The final version was assembled from variant 5 on the `final` branch, merged into `main`, and approved that day.

- **Data.** The project's SQLite database is read through `query` from the `shugo` module. Each row records the columns any caller reads while the query's owner is being evaluated. Triggers maintain the journal, and Shugo builds the indexes.
- **Synchronous page evaluation.** Pages are evaluated one at a time, in order. See [Data loading](../spec/data-loading.md#decisions).
- **Communication between Go and JavaScript.** Go sends one initial message with the read shapes and query indexes it already knows. JavaScript asks only about new ones. A price change needs one request from Go and one response with the updated records.
- **Projects without a data database.** An update does no evaluation until the project itself changes. See [Decision 26](../spec/decisions.md#26-project-without-database).
- **Storage and serving.** Records originally contained value tapes. Since November 22, 2026, they contain HTML. The server reads from the store for each request. See [Storage](storage.md#decisions) and [Runtime](runtime.md#decisions).
- **Incremental export.** Each page and shared-block record stores the version in which it last changed. The update writes the version with the record. Export writes pages whose records changed after its last run, plus pages that include a shared block that changed during that period. See [Runtime](runtime.md#decisions).
- **Atomic updates.** A failed update leaves the published site intact. See [Build pipeline](build-pipeline.md#failed-updates).

Since November 22, 2026, Shugo also tracks reads by non-HTML files produced by page modules such as `src/pages/sitemap.xml.ts`. A module's `GET` function is evaluated like a page. Its `query` reads are stored with its file record, and Shugo evaluates it again when those inputs change. See [Routing](../spec/routing.md#files-not-in-html).

Svelte began compiling and rendering templates on November 22, 2026. The change tracking rules stayed the same. A row records reads by page code, functions in `src/lib/`, and code compiled by Svelte. Pages, shared blocks, layouts, and route address lists still track the same inputs. All 21 changes in the catalog benchmark affected the same records as before. Page records now contain HTML rendered by Svelte with placeholders for shared blocks. See [Storage](storage.md#why-html-not-values).

The [catalog benchmark](../examples/catalog/README.md#results) uses 100,000 products. Changing one price took 31 ms to update and 7 ms to export. Updating the header on every page took 0.12–0.17 s, and the initial build took 14.6 s. After 21 changes, the incrementally updated and exported site matches a clean build file for file.

Run `go run ./cmd/inside examples/inside` for a small shop example that shows the internal formats and dependencies. It generates `examples/inside/report/index.html`.

### Initial search results

Calling `search("table", { … })` records a dependency on the initial results for the page, shared block, or route address list being evaluated. If the initial results change, Shugo adds those readers to the readers affected by ordinary data changes. Each evaluation replaces its set of reads. A conditional search stops being a dependency when that branch stops running, and Shugo removes a search when no page or block uses it. Updating a shared block affects every page that displays it. A call during module initialization is treated as a site-wide dependency because its result can be used by any template.

Changing a column used in search, filtering, or sorting rebuilds the table's index. If the initial results are unchanged, pages keep their records while filtered requests use the new published index. Column changes outside the index follow the ordinary `query` dependencies. A change to a column used to compute an indexed column also invalidates the index. See [Svelte islands](islands.md#search).

Shugo checks the journal after preparing the index and evaluating the pages. If a concurrent write changed the source, Shugo discards the prepared pages and index snapshot and tries again. After three attempts interrupted by writes, the command fails and the server continues serving the last published state.

## Prototypes

The author requested two implementations for comparison on November 20, 2026. Two follow-up prototypes applied the first design's dependency tree and actual row reads to the second design, respectively. They share a base and passed the tests on Bun and Node. The approved final version came from variant 5. The discussion used number 3 for the [data adapter proposal](proposals/data-adapter.md), which would bring data into the database used by variants 2 and 4.

These prototypes predate the switch to Svelte's template compiler on November 22, 2026. References to the compiler below refer to Shugo's earlier implementation. The switch did not affect [change tracking](#final-implementation).

| Prototype | Branch | Approach |
| --- | --- | --- |
| Base | `lab/base` | Go tells JavaScript which pages and shared blocks to evaluate and merges the partial result into the store. The base still evaluates everything. |
| Variant 1 | `lab/fingerprints` | Run all data code, fingerprint inputs identified by the dependency tree, and run templates only when their fingerprints change. |
| Variant 2 | `lab/query` | Read data from a managed SQLite database through `query` and track dependencies on its rows. |
| Variant 4 | `lab/query-tree` | Pair variant 2's updates with variant 1's compiler and separate data and template phases. Rows record which columns are read. |
| Variant 5 | `lab/query-signals` | Extend variant 2 with tracking of the columns each reader actually accesses, omit the dependency tree and authoring rules from variant 1, and let Shugo create indexes. |

The variant branches describe their implementation, authoring rules, decisions, and limitations in `VARIANT.md`.

### Variant 2 at a glance

- Shugo installs triggers on every database table. Inserts, updates, and deletes create journal entries with the previous values when available.
- Each `query` stores a read on the owning page or shared block. It identifies the table, the columns and values used to select rows, and the columns returned.
- An update reads the journal, finds the readers affected by changed rows, and evaluates only those readers. Shugo skips starting JavaScript when no readers are affected.
- A route's `getStaticPaths` is also a reader. Shugo enumerates its addresses again only when its inputs change.

### What gets evaluated

Both variants evaluate the same records. Changing a product price evaluates its page and card. Renaming the featured product evaluates its page, the header, and its card. Changing the footer's phone number evaluates the footer. Changing a field that no page displays triggers no evaluation. Both variants passed 20 benchmark changes at 1,000, 10,000, and 100,000 products. The incrementally updated site matched a clean build file for file.

Each variant also passed about 20 changes outside the benchmark, including changes to a product URL or row key, reordering categories, changing 300 prices together, rolling back a transaction, writing the same value, and recreating a table. The result matched a clean build after each change.

### Timings

Xeon E3-1270 v6, Go 1.27.4, Bun 1.4.4, and Node 24.22.1. The 100,000-product example contains 104,202 pages and 100,009 shared blocks. The benchmarks ran sequentially and measure the full `shugo update` command.

| Update after a data change on Bun | Original release | Variant 1 | Variant 2 |
| --- | ---: | ---: | ---: |
| One product price | 3.2–3.7 s | 1.28 s | 0.03 s |
| Prices of 20 products | 3.2–3.7 s | 1.21 s | 0.04 s |
| Review, variant, badge, or recommendation | 3.2–3.7 s | 1.2–1.3 s | 0.03 s |
| Featured product name or footer phone number | 3.2–3.7 s | 1.27–1.29 s | 0.12 s |
| Product moved to another category, affecting 168 pages | 3.2–3.7 s | 1.38 s | 0.26 s |
| Product added or deleted | 3.2–3.7 s | 1.31–1.38 s | 0.41–0.42 s |
| Field that no page displays | 3.2–3.7 s | 1.16–1.28 s | 0.01–0.16 s |

| Bulk update on Bun | Variant 1 | Variant 2 |
| --- | ---: | ---: |
| Prices of 100 products, 0.1% | 1.4 s | 0.17 s |
| Prices of 1,000 products, 1% | 1.7 s | 0.5 s |
| Prices of 5,000 products, 5% | 2.5 s | 1.6 s |
| Prices of 20,000 products, 20% | 4.3 s | 4.9 s |
| Prices of all 100,000 products | 14 s | 22 s |

| Other measurements | Base | Variant 1 | Variant 2 |
| --- | ---: | ---: | ---: |
| Initial build on Bun | 8.8–9.0 s | 9.1 s | 13.8–13.9 s |
| Initial build on Node | — | 9.8 s | 17.4 s |
| Update on Node | — | 1.9–2.2 s | 0.07–0.75 s |
| Store | 103 MB | 112 MB | 140 MB |
| Benchmark data | 62 MB JSON | 62 MB JSON | 80 MB database |
| 10,000 products, initial build and update on Bun | — | 1.0 s and 0.14–0.17 s | 1.5 s and 0.02–0.08 s |

- Variant 1's cost grew with the size of the site and changed little between updates. The example's data-loading code took 0.62–0.77 s, and data code and fingerprints for 204,000 pages and blocks took 0.47–0.56 s.
- Variant 2's cost grew with the size of the change. When at least a fifth of all rows changed, variant 2 was slower than variant 1 because each reevaluated page ran its own queries.
- In variant 1, data code and fingerprints took about three times as long as in the model below, with 2.5 μs per page or block versus 0.6–0.9 μs.

### What changes for site authors

| | Variant 1 | Variant 2 |
| --- | --- | --- |
| Data sources | Files, databases, and APIs | The project's database |
| Reading data | Ordinary TypeScript | `query("products").select(…).where(…)` |
| Changes to the shop example | Price functions moved to `lib/money.ts`, and one call moved from the template to the data code | Data moved into the database, and page and component data code switched to queries |
| Restrictions | Templates call functions from modules that do not load data. Components that take object props cannot load data. | Tables have one key. Queries target one table and combine conditions with AND. The only aggregate is `count`, a `join` returns one row, and views are unsupported. |
| Precision | The compiler infers what the template reads. | The author supplies the column list in `select`. |
| Errors | The build reports the violation with guidance. | Reads outside the database and module state accessed across pages can cause stale output without a warning. |

### Limitations

Variant 1 runs the data code for every page on each update, including any per-page queries. Function calls and ordinary components read an entire object passed to them. An npm package that reads a file synchronously from a template can escape change tracking. The full `shugo update -check` detects that case. Expression parsing, separate bundles, and a new `tdewolff/parse` dependency made the compiler more complex. This prototype changed 4,300 lines.

Variant 2 leaves importing outside data to the [data adapter](proposals/data-adapter.md). The benchmark generator writes the database directly. The query builder has the limits described above. An initial build was 1.5× slower, with 620,000 database queries and 610,000 store reads. A row insert or delete re-enumerated the route, which took 0.25 s for 100,000 pages. If a route supplied page `props`, a change to the route reevaluated every page on it. Routes switched to returning only addresses on November 22, 2026. This prototype changed 3,900 lines and kept the existing compiler and Go dependencies.

### Variant 4 with a dependency tree and row tracking

Updates use the database journal from variant 2. The compiler, separate data and template phases, and three authoring rules come from variant 1. Pages depend on the columns they actually read without needing a `select` list.

The dependency tree knows what the template reads, but a function such as `finalPrice(item)` can also read columns from the data code. Query rows record both kinds of reads. Copying values to the template through the dependency tree also counts as reading them. The compiler needed no changes for row tracking.

| 100,000 products on Bun unless specified | Variant 2 | Variant 4 |
| --- | ---: | ---: |
| One product price | 0.03 s | 0.04 s |
| Featured product name or footer phone number | 0.12 s | 0.12 s |
| Product added | 0.41 s | 0.46–0.58 s |
| Unread column changed on 5,000 products | 0.12 s, no reevaluation, columns listed in `select` | 0.12 s, no reevaluation, no `select` needed |
| Prices of 20,000 products changed together | 4.9 s | 4.9 s |
| Initial build on Bun and Node | 13.9 s and 17.4 s | 13.9 s and 17.0 s |
| Store | 140 MB | 140 MB |
| Lines changed from the base | 3,900 | 8,600 |

Compared with variant 2, `select` becomes a way to fetch less data. The example uses it once in the query listing 100,000 product URLs. Templates and ordinary components work on copies, so they can only depend on data they receive. The build enforces variant 1's authoring rules. Reading a row from another page through a stored module variable is an error. The full `shugo update -check` works with the database and reports pages whose output changes because they read data outside it. It took 7 s on 100,000 products.

Recording reads took 1–2% of initial build time. When the tree marks a row as a whole value, passing it to a function or ordinary component still counts as reading every column. The dependency tree includes every branch a template could take. For example, changing the arrival date of an in-stock product reevaluates its page even if the template only displays the date for out-of-stock products. The output record stays the same. This variant keeps the compiler complexity and authoring rules from variant 1.

The experiment showed that recording row reads gives column-level precision. The tree limits the data available to a template. Letting templates read tracked rows directly can also capture the branch taken and reads within functions and components. Variant 5 implements that approach with the simpler compiler.

### Variant 5 with tracking of actual row reads

Variant 5 extends variant 2. While evaluating the page, block, or route that issued a query, each returned row records every column accessed by page code, templates, functions, and ordinary components. It uses variant 2's compiler without the dependency tree, separate template phase, or additional authoring rules.

An evaluation can observe data only by issuing a query or reading a column from a returned row. Shugo records both. If those observations stay the same, the same code takes the same branches and returns the same result. A change in a value the code reads can make it take a different branch. This is the same principle as signals in SolidJS.

| Change | Variant 4 | Variant 5 |
| --- | --- | --- |
| Updated an unread column on a product variant passed whole to a component | Page reevaluated with unchanged output | No reevaluation |
| Updated the arrival date for an in-stock product that does not display it | Page reevaluated with unchanged output | No reevaluation |
| Price function in a template, `{price(finalPrice(item))}` | Reads the entire product. The example moved the calculation into data code. | Reads only `price` and `discount`. |
| Authoring rules | Variant 1's three rules and the database rules | The database rules |

Sequential runs on 100,000 products gave the following results.

| Measurement | Variant 2 | Variant 4 | Variant 5 |
| --- | ---: | ---: | ---: |
| One product price on Bun | 0.03 s | 0.04 s | 0.04 s |
| Featured product name or footer phone number | 0.12 s | 0.12 s | 0.12 s |
| Product added | 0.41 s | 0.46–0.58 s | 0.44 s |
| Prices of 20,000 products changed together | 4.9 s | 4.9 s | 5.5 s |
| Prices of all 100,000 products changed | 22 s | 22 s | 24 s |
| Initial build on Bun | 13.9 s | 13.9 s | 15.4 s including index creation |
| Initial build on Node | 17.4 s | 17.0 s | 19.0 s including index creation |
| One product price on Node | 0.07 s | 0.08 s | 0.09 s |
| Store | 140 MB | 140 MB | 141 MB |
| Lines changed from the base | 3,900 | 8,600 | 6,000 |

For variants 2 and 4, the data generator created indexes ahead of time. Variant 5 built its indexes during the initial build, using 0.8 s out of 15.4 s. With those indexes already present, the initial build took 14.4–14.7 s.

- **Row ownership.** A row records reads for the evaluation that requested it. That owner can read it only while its evaluation is running. Storing a row in a module and reading it from another page is an error, including with interleaved awaits. Variant 2 allowed that read and variant 4 caught only some cases.
- **Full verification.** As in variant 4, `shugo update -check` reevaluates the site and reports pages whose output changes because they read data outside the database. It took 7.6 s for 100,000 products.
- **Indexes.** Shugo builds the [indexes](proposals/data-adapter.md#indices) needed by the project's queries.
- **Stable row order.** Rows follow the query order and break ties by key. Variant 2 returned unordered rows in the order SQLite found them. A new index could then cause an existing page to differ from a clean build.

Bulk updates cost about 10% more. Tracking actual branches results in more read shapes, and Go checks each shape for readers of every changed row. The approximately 14 s initial build on Bun is 0.5–0.7 s slower than variant 2 before index creation. Templates can read files, the network, time, or module variables outside the database, which only `-check` can detect. The query builder still has variant 2's limits, and adding or removing a record makes Shugo enumerate the whole route.

### Small changes and one record used on many pages

A separate benchmark built 100,000 product pages showing each product's brand and the site's currency. Brand sizes ranged across 1, 10, 100, 1,000, 5,000, and 20,000 products. The page either read the brand and currency itself or used keyed components as shared blocks. Each change ran 3–5 times and timed the full `shugo update` command on Bun. Variants 2, 4, and 5 were within a few milliseconds of each other. The table shows variant 5. After each run, the incrementally updated site matched a clean build. The benchmark was in `research/change-tracking/fanout`.

| Change | Page reads data directly | Page uses a shared block |
| --- | ---: | ---: |
| No change, or a column that nobody reads | 6–8 ms | 6–8 ms |
| Product name on one page | 22 ms | 23 ms |
| Brand name on 100 pages | 53 ms | 21 ms |
| Brand name on 1,000 pages | 0.15 s | 24 ms |
| Brand name on 5,000 pages | 0.50 s | 25 ms |
| Brand name on 20,000 pages | 1.65 s | 37–49 ms |
| Currency on all 100,000 pages | 7.5 s | 0.11 s |
| Names of 20,000 products, each on its own page | 1.9 s | 1.9 s |
| Product added or removed | 0.21–0.24 s | 0.22–0.23 s |

- The number of pages reevaluated determines the time for a small change. Updating one record read directly by 20,000 pages took 1.65 s, and changing 20,000 records read by those pages took 1.9 s. Reading the journal cost about 16 μs per changed row.
- A shared block makes the same change cheaper. Shugo reevaluates the block once, and the server assembles each page with that block. Counting pages with changed HTML costs about 0.8 μs per page.
- Updating one page on Bun takes 20–24 ms, mostly to start JavaScript. Evaluating the page takes tens of microseconds. Startup on Node takes longer, with a one-page update taking 80–90 ms in the catalog benchmark.
- Adding or removing a product takes about 0.2 s because Shugo enumerates all 100,000 addresses on the route again.

The benchmark exposed two additional costs that the final version fixed.

- **Repeated store reads.** Go read the route and its settings separately for each affected page. Reading the settings once per route reduced the update from 1.65 s to 1.4 s for 20,000 pages and from 7.5 s to 6.2 s for 100,000 pages. All three database variants had inherited the repeated lookups from variant 2.
- **Server reload after publishing.** The server cached every page and shared-block record. On a new store version, it reloaded the entire site before responding to the first request. The delay was consistent across updates on the same site. In the final version the server reads each requested page from the store, avoiding the reload. See [Runtime](runtime.md#page-assembly).

  | Products | Pages | First request after each of 10 consecutive updates | Following requests |
  | ---: | ---: | ---: | ---: |
  | 1,000 | 1,042 | 4–10 ms | 0.4 ms |
  | 10,000 | 10,420 | 24–49 ms | 0.3 ms |
  | 100,000 | 104,200 | 213–288 ms | 0.3 ms |

## Original fingerprinting proposal

The project set aside variant 1 on November 20, 2026. This is the record of the proposal and the experiments that informed it. Compiler references in this section are to Shugo's template compiler, which Svelte replaced on November 22, 2026.

### Overview

Every update runs all data code for pages and shared blocks and loads all their data. The framework does not need to track reads while this code runs.

The compiler works out which data variables a template reads. Its **dependency tree** describes those reads. For each page and shared block, Shugo computes a **fingerprint** from the values in the tree.

Go compares these fingerprints with the saved ones. Shugo evaluates a template only when its fingerprint changes. The template receives a copy of the values it reads.

### Stages

| Stage | Runs in | Work |
| --- | --- | --- |
| 1. Compilation | Go | Produce data code for `load.mjs`, a template for `render.mjs`, and a dependency tree from each `.svelte` file. Keep the existing runtime skeleton. |
| 2. Data loading | JavaScript, `load.mjs` | Run the data code for all pages and shared blocks and compute their fingerprints. |
| 3. Comparison | Go | Compare fingerprints and identify new or changed pages and shared blocks. |
| 4. Expression evaluation | JavaScript, `render.mjs` | Evaluate only the templates Go selected. |
| 5. Runtime | Go | Use the existing runtime. |

This follows the author's original plan, with Go comparing results between loading and evaluation. Both JavaScript stages run in the same process and can share data without copying across processes. Only fingerprints go to Go, at 8 bytes per page or block.

### Dependency tree

A shortened product page template, expressed in the current Svelte syntax.

```svelte
<h1>{item.title}</h1>
{#if item.discount}<span>−{item.discount.percent}%</span>{/if}
<p>{price(item.price)}</p>
{#each item.variants as variant}<Variant {variant} />{/each}
<h2>Reviews: {reviews.length}</h2>
{#each reviews as review}<p>{review.text}</p>{/each}
{#each item.related as id}<ProductCard {id} />{/each}
```

Its dependency tree is:

```
item
├─ title
├─ discount         whether it exists
│  └─ percent
├─ price
├─ variants         each element
│  ├─ title         read by the Variant component
│  └─ stock
└─ related          each element, a product ID
reviews
├─ length
└─ each element
   └─ text
```

The template never reads `item.note`, `item.category`, or `item.variants[].sku`. These fields are absent from the tree, and changes to them do not affect the page.

| Template expression | What the tree records |
| --- | --- |
| `{item.title}`, `href={item.a.b}` | The individual value |
| `{#if item.discount}` | Whether the value exists, plus any fields read inside the branch |
| `{#each item.variants as v}` with `{v.title}` in the body | The list length and the fields the body reads from each element |
| `{reviews.length}` | The list length |
| `{price(item.price)}` | `item.price`. The imported `price` function is code. |
| `{finalPrice(item)}`, `{item.tags.join(", ")}`, `{items[0].title}`, `{...attrs}` | The whole object, since the compiler cannot infer which parts the operation reads |
| `<Variant variant={variant} />` | The fields `Variant` reads, or the whole `variant` when the compiler cannot infer them |
| `<ProductCard id={id} />` | The value of `id`. The card is a separate shared block with its own fingerprint. |

Recording the whole object may cause extra evaluation and keeps pages fresh.

The compiler builds the tree during compilation. esbuild removes types, and `tdewolff/parse` parses expressions and identifies where each name comes from. A separate prototype verified that this combination recognizes property chains, method calls, arrow functions, and conditions. That prototype analyzed individual expressions and did not yet infer reads across components.

### Fingerprints and template inputs

- **Fingerprint.** JavaScript walks the dependency tree and hashes values read directly from their objects. The 64-bit fingerprint requires no serialization.
- **Template inputs.** The same traversal copies the required values for pages and blocks that will be evaluated.

A template can depend only on the copy it receives, so every available value is covered by its fingerprint. A missing dependency makes a value unavailable at the first build, making the error visible immediately.

Each page or block compares its fingerprint only with its own previous fingerprint, so 64 bits is sufficient here.

### List keys

The original prototype used a key to name a block within a list. This design names pages by address and shared blocks by their parameters. A list item belongs to the enclosing page or shared block and is evaluated with it. As a result, `{#each}` does not require a key.

To update a list item independently, render it as a shared block such as `<ProductCard id={item.id} />`. Its parameters serve as the key.

### Data loading rules

Data code stays at the top of each file and the template at the bottom. Site authors follow three rules.

1. **Pages, layouts, and shared blocks load data.** A shared block is still a component called with simple parameters. An ordinary component receiving an object only displays the supplied value. Its data code cannot use `await`, read files, or access the network.
2. **Templates read data and call imported functions.** They can read data variables and imported values such as numbers, strings, objects, lists, dates, `Map`, and `Set`. Functions declared in data code must move to a module before a template can call them.
3. **Modules used by template functions cannot load data.** Keep template functions and data-loading code in separate modules.

The build reports any violation with its location and a suggestion.

The shop example would add a module for price functions that templates use.

```ts
// src/lib/money.ts
// Functions used by templates moved here from src/lib/catalog.ts.
export function price(value: number): string { … }
export function finalPrice(product: Product): number { … }
```

The header would look up the featured product in data code instead of the template.

```svelte
<!-- src/components/Header.svelte -->
<script>
  import { categories, inCategory, product, shop } from "../lib/catalog"
  const menu = …
  const featured = shop.featured ? product(shop.featured) : null
</script>

<a class="brand" href="/">{shop.name}</a>
…
{#if featured}
  <p>Featured product: {featured.title}</p>
  <ProductCard id={featured.id} />
{/if}
```

The product page, category listing, product card, variant, and footer would only need their import line updated.

### How the proposal keeps pages fresh

1. Data code runs again for every update and sees the latest data regardless of what it reads or where it comes from.
2. Templates and ordinary components can depend only on the inputs passed to them.
3. Template code is bundled separately from data code and gets separate module instances without loaded data. The build checks that it cannot reach files, the network, the clock, or random number sources.

A separate esbuild prototype tested the third rule.

| Code a template can reach | Prototype result |
| --- | --- |
| Module reads a file during import | The bundle retains `node:fs`, and esbuild identifies the importing module so the compiler can report it. |
| An unused function reads a file | esbuild removes the function and the import. |
| Module awaits the network during import using `await fetch` | esbuild reports an error because the template bundle format lacks top-level `await`. |
| `fetch`, `Date.now()`, `Math.random()`, `process.env` | esbuild substitutes a stub that reports a clear error. |

esbuild does not detect `new Date()` without arguments. The expression parser used for the dependency tree can detect it.

A package from npm can still read a file or database synchronously from inside an ordinary component. Package internals are outside the compiler's checks. The full verification command evaluates the entire site and reports any block whose output changed with the same fingerprint.

### What gets evaluated after each change

The model uses the benchmark's 100,000 products, with 204,000 pages and shared blocks. The dependency trees were written by hand to model the expected compiler output. The original release evaluates all 204,000 records for any change listed below.

| Data change | Pages to evaluate | Shared blocks to evaluate | Pages with changed HTML |
| --- | ---: | ---: | ---: |
| None | 0 | 0 | 0 |
| Unused field on 5,000 products | 0 | 0 | 0 |
| One product price | 1 | 1, its card | 7 |
| Prices of 20 products | 20 | 20 | 109 |
| Featured product renamed | 1 | 2, the header and card | All |
| Different featured product selected | 0 | 1, the header | All |
| Footer phone number | 0 | 1, the footer | All |
| Product moved to another category | 168, listings in both categories | 0 | 168 |
| Text of an approved review | 1 | 0 | 1 |
| Stock for one variant | 1 | 0 | 1 |
| Unused variant SKU | 0 | 0 | 0 |
| First variant removed | 1 | 0 | 1 |
| Two related products swapped | 1 | 0 | 1 |
| Last product removed | 88 | 0, the product page and card are removed | 88 |
| Product added | 85 | 2, its card and the header | All |

Seven pages show the product price. Shugo evaluates its page and card. The other six pages refer to the card by name and need no separate evaluation.

### Cost

Xeon E3-1270 v6, Bun 1.4.4, Node 24.22.1, and Go 1.27.4. The data file for 100,000 products occupies 65 MB.

JavaScript work before evaluating changed records:

| Step | Bun | Node |
| --- | ---: | ---: |
| Project code reads the data file | 0.31–0.37 s | 0.61–0.71 s |
| Project data module and `getStaticPaths` | 0.15–0.27 s | 0.26–0.35 s |
| Data code for 100,000 product pages and 100,000 cards | 0.05–0.07 s | 0.07–0.16 s |
| Fingerprints for all pages and shared blocks | 0.08–0.10 s | 0.10–0.15 s |
| Transfer 6 MB of keys and fingerprints to Go | 0.02–0.04 s | 0.04–0.05 s |
| **Total** | **0.61–0.78 s** | **1.19–1.36 s** |

Reading and comparing 204,000 stored fingerprints takes Go 0.16–0.18 s when the store puts each fingerprint in its own record and 0.01 s when it stores all fingerprints together.

Adding up the parts gives an estimated update time of 0.7–1.0 s on Bun and 1.3–1.6 s on Node. The full mechanism had yet to be assembled at the time of this estimate. The original release updated the site in 3.2–3.7 s on Bun and 4.3–4.7 s on Node.

The framework takes 0.2–0.4 s. The rest is project code, which reads the 65 MB file and loops through the product list 58 times.

Copying the template's inputs costs 0.2–0.5 μs per product page. Copying all variables with `structuredClone` costs 7–11 μs.

### Approaches tested

All fingerprint measurements cover 100,000 product pages.

| How to detect changes | Bun | Node | Finding |
| --- | ---: | ---: | --- |
| **Fingerprint from the dependency tree** | **0.05 s** | **0.08 s** | |
| `JSON.stringify` and hash the text | 0.17 s | 0.55 s | Different data can produce the same text if it contains a `Map`, dates, or `undefined`. |
| Walk and hash all variables in JavaScript | 0.41 s | 0.44 s | Reads all values. The dependency tree can use this approach for entries marked as whole objects. |
| Hash the output of Bun or Node's built-in serializer | 0.64 s | 0.80 s | Reads all values and takes longer. |
| Seroval and hashing | 0.74–0.80 s | 1.3–2.3 s | Reads all values and takes longer still. |
| Go in WebAssembly within the JavaScript process | 0.32–0.38 s | 0.72–0.95 s | Requires encoding and copying the data. |
| Compare outputs as in the original release | 3.2–3.7 s for the whole update | 4.3–4.7 s | Evaluates every page and block. |

**Seroval.** The comparison uses fingerprints. A plain copy gives the template its inputs and costs 15–70 times less than copying every variable. A Seroval copy takes 7 μs as JSON or 48–57 μs as text per page. A template still sees copied values instead of live variables.

**Go in WebAssembly.** Go hashes 78 MB in 25–50 ms and compares the hashes in 1 ms. The WebAssembly module sees bytes, so passing JavaScript objects to it requires conversion to text, encoding, and copying. That step takes 0.3–0.9 s and is unnecessary when JavaScript walks its own objects. The module is 1.9 MB and starts in 20 ms. Bun 1.4's built-in WASI requires a workaround to run it.

**Marking sources with `source(...)`.** Authors would have to remember to annotate every data source. An omitted annotation could silently leave a page stale.

**Recording reads during evaluation.** A `Proxy` can capture each read at runtime. This approach was not measured. It would intercept every data access in project code, break ordinary use of `Map` and dates, and still require naming the root of each data source.

### What still scales with site size

- Data code runs for every page and shared block during each update. In the benchmark, that takes 0.05–0.07 s for 200,000 records. Expensive per-page queries would still run for every page. Loading many pages' inputs in one query in `getStaticPaths` or a module can reduce that cost.
- The framework fingerprints every page and block. That takes 0.1 s for 200,000 records.
- Each update loads the entire dataset. To avoid doing so, the source must report its changes. See [Data adapter](proposals/data-adapter.md).

Expression evaluation, writing records, and serving pages depend only on the size of the change.

### Changes to Shugo

| Package | Change |
| --- | --- |
| `internal/compile` | Parse expressions and build the dependency tree with `tdewolff/parse`. Build two bundles, `load.mjs` and `render.mjs`, and check template code. |
| `internal/jsrt` | Split data loading, fingerprinting, and input copying from template evaluation. Evaluation runs without `await`. |
| `internal/evaluate` | Drive updates in Go. Read and compare fingerprints, specify what to evaluate, and receive records. |
| `internal/store` | Store a fingerprint with each record. |
| `program`, `render`, `server`, `record` | Use the existing implementation. |

The proposed implementation order was:

1. Add a dependency tree to the compiler and test when it records specific reads.
2. Split the JavaScript bundles, copy template inputs, and add the build-time checks.
3. Compare fingerprints in Go, store them, and report how many of the total records were evaluated.
4. Run the same 20 benchmark changes, compare against a clean build, and record the timings.
5. Apply randomized data changes and compare each incremental result against a clean build.

## Open questions

- Which operations should the query builder support? See [Data loading](../spec/data-loading.md). The [data adapter proposal](proposals/data-adapter.md) discusses importing external data into the database.
- Read tracking records only values checked for equality. A comparison or `limit` window can therefore be invalidated by any row change in a column the query uses. Support was deferred while pages used equality conditions. See the [range query proposal](proposals/range-queries.md).
