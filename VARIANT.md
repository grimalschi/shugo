# Variant 4: database change journal and dependency tree

On the `lab/query-tree` branch, this prototype combines variant 2 with the dependency tree from variant 1.

- **Update — as in option 2** (`lab/query`) The data are in the SQLite database of the project, the database is running a change log, and only the changes are calculated. The update time depends on the size of the change, not the site.
- **Compilation and two sides as option 1** (`lab/fingerprints`) Data code is performed on the loading side, templates are performed on its side and copies of what they read. Three rules for the author and their errors remain.
- **New: The page depends only on those columns of lines that actually read.** List `select` is not necessary for this.

This branch started from `lab/fingerprints` and brought in the database, journal, queries, and read tracking from `lab/query`. Decision 1 gives the rationale. See the earlier notes with `git show lab/fingerprints:VARIANT.md` and `git show lab/query:VARIANT.md`. Their decisions carry forward except where this document says otherwise.

## How it works

1. **Compilation.** As option 1: each `.shugo` file produces a data code, a template and a tree of relationships -- which the template reads from the variable data code. The data code is collected in `_build/load.mjs`, the templates are in `_build/render.cjs`.
2. **Data.** is located in the SQLite database, called `shugo.json`: `{ "data": "data/site.db" }`. Shugo puts triggers on each table, and they write every string change together with the previous values.
3. **Request.** The data code asks the database via `Shugo.query`. Each line received remembers which columns it reads.
4. **My own part.** When the data code has been used, the loading side goes through the addiction tree and copys the template what it reads. This passage reads lines as well as the data code, and the lines remember it.
5. **Reading.** When a copy is ready, no one will read the lines: The template works for the copy. Each request becomes "read": the lines of which tables with what values were sought and what columns they read. Reading goes to Go with a page or block.
6. **Update.** Go reads the journal and learns which lines and columns have changed. He finds pages, common blocks and lists of pages of routes to which this has touched, and he calls the side of the load only. If no one is touched, JS does not start.

```
data/site.db ── journal: which lines have changed and which columns
      │
      ▼
Go: Who read these lines in these columns ── pages, common blocks, lists of route pages
      │  "comply these"
      ▼
load.mjs: Data code ──► Shugo.query ──► lines that remember what they read
      │  part of it: copy of what the template reads; copying is also reading
      ▼
render.cjs: template ──► recording and reading ──► Go: storage facility
```

It's all being re-calculated when the journal can't tell what changed, first time, after a change in the design or the base tables, and on `shugo build` command.

The project without `data` in `shugo.json` works as option 1: The update uses the data code of all pages, and prints tell us which templates to calculate. The test of option 1 sites is not changed.

## What a site author writes

```
<!-- pages/products/[slug].shugo -->
---
import { finalPrice, price } from "../../lib/money"

export function getStaticPaths({ query }) {
  return query("products").select("slug").all().map(({ slug }) => ({ params: { slug } }))
}

const item = Shugo.query("products").where({ slug: Shugo.params.slug }).first()
const variants = Shugo.query("variants").where({ product: item.id }).order("position").all()
const cost = finalPrice(item)
---
<h1>{item.title}</h1>
<strong>{price(cost)}</strong>
<g-for each={variants} as="variant"><li>{variant.title}: {variant.stock}</li></g-for>
```

Requests are all lined up. The page depends on `title` – it reads the template, `price` and `discount` – it reads `finalPrice` in the data code, and `position`, `title`, `stock`. The change of `description`, `note` or any other column of the product does not calculate this page.

`select` is only left as a way to take less from the base: in `getStaticPaths` above it takes one column from 100,000 lines.

## How column reads are recorded

The string that returned the request was a normal item wrapped in `Proxy`. The wrapping (in `Watcher` code, "guard") records the name of each column that is read at the line. The row reads two: the data code and the passage that copys the template part of it. The shuttle does not see the lines.

| What's done with the row | What is considered read |
| --- | --- |
| `item.price`, `const { title } = item` in the data code | this column |
| `"price" in item`, `item.hasOwnProperty("price")` | this column |
| `{item.title}`, `href={item.slug}` in the template | this column: read the passageway |
| `cond={item}` in the template | nothing: only asked if there was a line |
| `{ ...item }`, `JSON.stringify(item)`, `Object.keys(item)`, `for … in` | line whole |
| `{shout(item)}` in the template; `<Row row={item} />` to the conventional component | line in whole: tree says "the whole" |
| line given to pages via `props` of `getStaticPaths` | line whole |

What comes from the request:

| Request | Reading |
| --- | --- |
| `query("products").get("p7")`, read `title` and `price` | products on `id` = p7; important `title`, `price` |
| `query("variants").where({ product: "p7" }).order("position").all()`, read `title`, `stock` | variants on `product` = p7; `position`, `title`, `stock` are important. The added and remote version of this product always hits |
| `query("reviews").where({ product: "p7", approved: true }).all()` | reviews on `approved, product` = 1, p7. Unapproved withdrawal does not affect |
| `query("products").where({ category: "c9" }).count()` | products on `category` = c9. The price does not affect |
| `query("products").where({ category: "c9" }).order("position").limit(24).all()` | products on `category` = c9; it's important `position`. And separate reading on each returned line: products on `id`; columns read at this line are important. |
| `….join("categories", "category")` | column `category` of the product read by the compound itself; cells on `id`; columns read by the category are important |
| `query("products").select("slug").all()` in `getStaticPaths`, pages given | products without conditions; `slug` is important. It affects every item added and removed. |

Rules:

- **The filter columns and order and columns that are read are important.** The first decides which lines are found and in what order, the second lines are what is shown.
- **The lines of the request that returned everything found are counted together.** Reading One, Columns - all that's been read in one line.
- **The lines of the request, which are cut by `limit`, `offset` or `first()`, are considered to be one.** is the same line behind the trim; its changes should not be hit by a page over columns that it only shows. Therefore, each returned line is a separate key reading.
- **The result is when no one reads the line:** at the page and the general block after the templates were given a copy; at the list of pages when `getStaticPaths` returned the list.
- **What was read in this calculation is read.** Data code that is `available = 1` Doesn't look in `arrival`from `arrival` It's not gonna be. `available = 0` - The page will be counted and read `arrival`The shuttle reads differently: the passage takes whatever pattern can read, whatever branch `g-if` He didn't go.
- **The accuracy ends where the tree says "spinned."** The string transmitted by the function in the template or the conventional component is read out in its entirety. The line that the route distributed to the pages via `props`.

### Performance cost

The readings of 200,000 lines in 12 columns, Bun 1.4.4 and Node 24.22.1; the script `rows.mjs` is in a research folder near `bulk.mjs`. The range is on several launches.

| | Bun | Node |
| --- | ---: | ---: |
| Read the column of a conventional object | 3–4 ns | 4–5 ns |
| Read column through row watcher | 30–60 ns | 60–80 ns |
| Set the row watcher to `Set`, `Watcher`, `Proxy` | 35–60 ns | 45–80 ns |
| Copy the whole row, ordinary object | 0.13–0.19 μs | 0.26–0.28 μs |
| Copy the whole row through the row watcher, column by column | 4.4–4.8 μs | 7.7–8.8 μs |
| Same step (decision 15) | 0.18–0.31 μs | 0.33–0.47 μs |

The first build of 100,000 products, the Bun loading side, is 7.3 s. Of these, 2.6 c are the SQLite (of 620,000), 0.25 s is the row watcher and the record of the reading, 0.25 s is the recapture of readings (which was also in option 2), 0.36 s is the copy for templates. First build under the same conditions:

| First build, 100,000 products, in turn, six laps, in `/tmp` | Option 2 | Option 4 |
| --- | ---: | ---: |
| Bun | 13.1–13.5 s, average 13.3 | 13.3–13.7 s, average 13.5 |
| Node | 16.7–18.1 s, average 17.2 | 16.2–16.7 s, average 16.5 |

The `/tmp` folder is stored in memory, so the storage log is faster here than the booth. On Bun option 4 is on average 0.2 s longer, which is 1% to 2% less than the difference between circles. On Node, it is faster because the lines are no longer copied (decision 14). Option 2 can take the same correction; here it gave 1.9 s, so it's going to make option 2 faster by about a second. The text is not affected by the update: it lists the pages.

## Rules for site authors

Three rules of option 1 remain, each violation is a mistake with a location in the file and a clue:

1. **Data downloads pages, models and common blocks.** The component whose data code is downloading something only gets simple parameters and is calculated once on the entire site. The request to the database is also a download.
2. **The shuttle reads data and causes imported functions.** The function declared in the data code does not see the template.
3. **The module whose function causes the template does not load the data itself.**

The database rules were added to them:

4. **Data are read from the project base.** read past it -- file, network -- does not track: `shugo update` does not see it, `shugo build` reads it over again, `shugo update -check` will name pages that depend on it.
5. **The database asks for the data code** pages, layout or common block through `Shugo.query` and `Shugo.paginate`, and route through `query` in `getStaticPaths`. The shuttle and what it calls are not asked about the base.
6. **The string belongs to the person who asked it.** cannot be left in the variable of the module for other pages.
7. **The table has a single column key; the request goes to the table, not to the presentation; the value is the same as the column.** These are the rules of option 2.

| What's done? | What's it say? |
| --- | --- |
| request in template: `{Shugo.query("products").count()}` | `a template cannot ask the data database: it shows what its script gives it. Ask in the script — const rows = Shugo.query(…).all() — and show the variable` |
| the function that triggers the template makes the request | `Shugo.query is called in a template, or in code that a template calls…` — When computing page |
| the request component was transferred to the object | `components/More.shugo loads data: its script asks the data database (Shugo.query). So it is computed once for the whole site and takes only plain values… Here item is an object` |
| the component makes a request and reads the page on which it stands | `this component loads data — its script asks the data database (Shugo.query) — so it is computed once for the whole site; and it reads the page it is shown on…` |
| the line was left in the module and read by another page | `a row of the table products is read when the page, shared block or list of pages that asked for it is already computed, so whoever reads it now would not be computed again when the row changes. A row is for the script that asked for it…` |
| request in project without base | `Shugo.query reads the data database of the project, and this project names none. Name it in shugo.json: { "data": "data/site.db" }` |

The messages of options 1 and 2 — the functions in the template, `fetch` and `Date.now()` in the template code, submissions, type of value, unknown table or column — remained as they were. The complete manual: [docs/authoring.md](docs/authoring.md), sections "Pages from Data" and "Data".

## Decisions made during implementation

We can review each one.

1. **Started with option 1, option 2 is moved to it.** Option 1 is primarily a compiler: expression review, tree, two assemblies. He's big and connected. Option 2 of the compiler did not touch: parts of it — `internal/data`, `query.js`, `track.go`, and the storage tables — are moved by the entire files. The reverse would mean moving the compiler to pieces.
2. **The reading is a string itself, not a compiler.** The compiler only sees the template. The data code is arbitrary TypeScript: what `finalPrice(item)` reads from the product, it is not possible to produce it using the text. The line in `Proxy` sees both readings the same way. The passage of "its part" from option 1 was a model reading itself: no line was added to the compiler.
3. **What is read in the calculation is considered to be what is read, not what is read.** That's true because the calculation is repeated: while the columns are old, the data code will follow the same path and give the same message. The same is true of the SolidJS signals.
4. **Uncut request — one reading per line, one cut per line.** The reason behind the cut is that there are lines behind the clipping that the page does not show. The reason for the first is one `Set` on a request instead of `Set` on a line, and one entry in the vault instead of hundreds.
5. **The column by which `join` searches the string is considered read.** reads it itself by framework, past the row watcher. Without this, the product was moved to another category, and the page with the name of the category was not calculated; the `a joined row` test caught it.
6. **The route that gives lines to pages through `props` reads them all.** The pages read these lines after the list has given its reading. You can move the reading from page to page, but this is the second accounting system; not done. The Council of the previous: the route gives addresses, the page asks for it.
7. **The page's parameters are always taken from her address.** The page is calculated from the list of routes, then from one address. In the first case, `Shugo.params` was given back `getStaticPaths`, a number instead of text, extra keys, and one page could have come out different. Now both ways give one.
8. **The prints are not considered to be from the database.** What to compute is the journal's decision. The print would have to be considered the same passage that reads the lines, that is, read twice the same for the answer that already exists. The storage space in this mode does not hold any fingerprints.
9. **One protocol and one record of both ways.** Go asks three things: list pages, say prints, compute. The record carries both prints and reading; the empty is empty. Where options 1 and 2 did the same thing in different ways — a session, the recording that fell into the common block's data code, left one implementation.
10. **The database name is included in the build mark.** The Warehouse collected by prints does not contain readings, and vice versa. The `data` was changed to `shugo.json` -- everything is re-compiled.
11. **`shugo update -check` also works with base.** He calculates everything and calls the pages and the general blocks that have changed their recording, although the journal has not reached them: they read something past the base. If they do, the team ends up with a mistake; the vault is after it is correct. The check takes 7 s per 100,000 products.
12. **`select` is left only for speed.** is not a precise one: the column taken and not read does not hurt the page; the `TestSelectTakesLess` test.
13. **The line read after the person who asked for it was already counted is a mistake.** The string records what it read on the account of who asked for it. If it's kept in the variable module and read another page, that page isn't dependent on anything and it's not gonna be good. Option 2 didn't notice that. Here, the line says no and says what to do. The lines that are given through `props` can be read: they are already counted in their entirety.
14. **Node is a regular prototype, not a copy.** `node:sqlite` gives objects without a prototype: they have no `hasOwnProperty` and `toString`. Option 2 copied each line. Deliver the prototype cheaper: the first build on Node was shorter by 1.9 s out of 20.
15. **The whole line takes one step.** lists columns through `Proxy` 25–35 times more expensive than the normal object. The passage that needs the entire line asks the row watcher for the row itself (`unwatched` in `query.js`); the guard notes "read everything." Asking a regular object if it's not a line is less than 20 ns. For 100,000 items, this is 0.5 to 0.7 s on Bun and 0.8 s on Node.
16. **Request on the template side is closed in three places.** Compier refuses to request in the template. The component whose data code makes the request becomes downloadable and it has rule 1. The function that the template has called and that has reached `Shugo.query` is made an error in computing.
17. **In the stand, instead of "articula of option", "a shelf of option".** in the articular database is the key to a line: change it means delete one line and add another. For the "column that no one shows" scenario, a column of `shelf` is added to the table of options.
18. **The discount price is considered in the data code** cards and pages of products. This is how `price` and `discount` are read, and the template gets the number. Sending the item to `finalPrice` directly in the template would mean reading it in its entirety.

## Limitations

- **Accuracy does not go through functions and ordinary components.** The Products Page gives each version of the `Variant` component in its entirety, so it depends on all columns of the Option. In the stand, a change of shelf calculates the page, and the record is the same. In the example, it is deliberately left to show the boundary; it is treated simply — to pass `title` and `stock` instead of the line. In the frame, it is not treated indiscriminately by components across a border that was not quite in option 1.
- **`props` route** reads the entire lines and calculates all its pages when it changes.
- **`select` is needed for speed** where there are many lines and few readings, without which the list of 100,000 items is read 112 ms instead of 34 and the addition of 0.63 s instead of 0.46.
- **First build** is 1.5 times longer than option 1: this is the price of requests and readings as in option 2. Over the second version 2, the record adds 1- 2% to Bun and is estimated to be about a second out of seventeen on Node.
- **By the base.** File or network in the data code, simple values (not rows) left in the variable module, a package that reads something itself is not monitored. Only `shugo update -check` is found.
- **The line in the module isn't always on.** Error occurs when the applicant has already been calculated. The data code without `await` is run in a row, and there it is always so. If two pages wait for data and are done in a mix, reading the second can go into the first undetected.
- **`structuredClone(row)` is not operational:** line – `Proxy`. `{ ...row }` works and reads the whole line.
- **What was read depends on the data.** The product without feedback does not read `text` reviews, and the types of reading in the vault are increasing: per 100,000 items of their 18s, reading 613 thousand.
- **The weak option 2 is:** is small; the route is listed in its entirety when the line has been added or deleted - 0.25 s per 100,000 pages; as external sources are entered into the database, it is not done.
- **The weak version 1 compiler remains:** `using` and decorators in the data code, the location of the analysis error is the beginning of the expression, and `Shugo.self` is not available in the downloader file data code.
- **`shugo dev` with base** checked with his hands: the price changed -- the page and card are calculated; the column that nobody reads has changed -- nothing. There is no automatic test, just like option 2.
- **Documents in `spec/` and root `README.md` in this branch were not updated** except `docs/authoring.md`.

## Benchmark

```sh
go build -o bin/shugo ./cmd/shugo
node examples/catalog/stand.mjs --size 10000        # --runtime node -- compute on Node
```

The bar changes the data 21 times by the usual SQL: 20 variations to option 2 and "the shelf of the option". After each, it checks how many pages and common blocks are calculated and at the end compares the step-up website with the zero-size build, the file behind the file. For 1,000, 10,000 and 100,000 items, Bun and Node, all checks are passed, sites are consistent. 23 changes to the top of the bench have been made from the version 2 test (change of address and key of the product, reset of categories, back-up of transaction, `INSERT OR REPLACE`, `VACUUM`, re-forming of the table): the site has been matched with a zero build each time.

The example of a store is an example of option 2 given by the rules of option 1. `select` lists are removed anywhere except for the list of product addresses. Price functions are in `lib/money.ts`. The discount price is considered in the data code.

### 100,000 products, 104,202 pages, and 100,009 shared blocks

Xeon E3-1270 v6, Go 1.27.4, Bun 1.4.4, Node 24.22.1, 20 November 2026, the walls were running in turn, the vehicle was loaded from 1.5 to 3.3, time is the whole team. `shugo update` I'm going to go with the launch.

| What changed the data | The string changed | Pages calculated | Common blocks calculated | Pages with new HTML | Bun, ms | Node, ms |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Nothing. | 0 | 0 | 0 | 0 | 132 | 148 |
| A column that nobody reads has 5,000 items. | 5,000 | 0 | 0 | 0 | 115 | 143 |
| Price of one commodity | 1 | 1 | 1 | 7 | 37 | 88 |
| Prices of 20 products | 20 | 20 | 20 | 70 | 45 | 104 |
| Name of item of month | 1 | 1 | 2 | all | 121 | 192 |
| The products of the month were another commodity | 1 | 0 | 1 | all | 119 | 153 |
| Products of the month are no longer available | 1 | 0 | 1 | all | 117 | 156 |
| Products moved to another category | 1 | 168 | 1 | all | 275 | 339 |
| The phone's in the basement. | 1 | 0 | 1 | all | 117 | 148 |
| The products have a discount. | 1 | 1 | 1 | 7 | 31 | 69 |
| Text of the approved withdrawal | 1 | 1 | 0 | 1 | 31 | 77 |
| A hidden withdrawal added | 1 | 0 | 0 | 0 | 14 | 15 |
| Balance of one option | 1 | 1 | 0 | 1 | 34 | 78 |
| A shelf of options that no one shows | 1 | 1, previous entry | 0 | 0 | 38 | 91 |
| Option added first | 1 | 1 | 0 | 1 | 36 | 91 |
| The first option was deleted | 1 | 1 | 0 | 1 | 38 | 103 |
| One commodity values | 1 | 1 | 1, new | 1 | 42 | 75 |
| Two recommended items were replaced | 4 | 1 | 0 | 1 | 47 | 69 |
| The recommended products were removed | 1 | 1 | 0 | 1 | 42 | 77 |
| The last item was removed. | 15 | 89 | 0; card removed | 88 | 408 | 580 |
| Products added | 5 | 86 | 2 | all | 456 | 695 |

- **Column `note` for 5,000 items does not compute anything**, although no request other than the address list is called columns. In option 2, it required `select` in each request.
- "Nothing" comes first after the build and costs 132 ms. Later, the upgrade that didn't hurt anyone takes 14 ms: hidden withdrawal.
- Where the changed block shows all the pages (a hat, a basement), about 90 ms is counted at 104,202 pages with a new HTML.
- The same as in option 2, all 20 of its modifications.

### Other results

| | Option 1 | Option 2 | Option 4 |
| --- | ---: | ---: | ---: |
| Price of one commodity, Bun | 1.28 s | 0.03 s | 0.04 s |
| Name of item of month, cell phone in basement, Bun | 1.27–1.29 s | 0.12 s | 0.11–0.15 s |
| Products moved to another category, Bun | 1.38 s | 0.26 s | 0.28–0.35 s |
| Products added or deleted, Bun | 1.31–1.38 s | 0.41–0.42 s | 0.36–0.48 s |
| Node update | 1.9–2.2 s | 0.07–0.75 s | 0.02–0.72 s |
| 100 prices at once, 0.1% | 1.4 s | 0.17 s | 0.17–0.19 s |
| Prices of 1,000 products, 1% | 1.7 s | 0.5 s | 0.51–0.62 s |
| Prices 5,000 products, 5% | 2.5 s | 1.6 s | 1.6–1.7 s |
| Prices of 20,000 products, 20% | 4.3 s | 4.9 s | 4.9–5.0 s |
| Prices of all 100,000 products | 14 s | 22 s | 22 s |
| First build, Bun | 9.1 s | 13.8–13.9 s | 14.0–14.5 s |
| First build, Node | 9.8 s | 17.4 s | 17.0–17.3 s |
| Storage | 112 MB | 140 MB | 140 MB |
| Platform database | JSON, 62 MB | 80 MB | 80 MB |
| 10,000 items: first build and update, Bun | 1.0 s and 0.14- 0.17 s | 1.5 s and 0.02- 0.08 s | 1.5–1.8 s and 0.01–0.09 s |
| 1,000 products: first build and update, Bun | — | 0.26 s and 0.01-0.04 s | 0.24 s and 0.01-0.06 s |

The numbers of options 1 and 2 are taken from the `spec/change-tracking.md` comparison of the main branch: they were removed earlier that day. Option 4 numbers were removed after; other tasks were also being carried on the car. The first compilation of options 2 and 4 under the same conditions is in the table under "How much is it worth?".

This is the first build per 100,000 items, Bun: compilation of 0.04 s, list of pages 0.2 s, calculation of 7.3 to 7.6 s, recording in storage 6.4 to 6.7 seconds.

## Where to look in the code

| File | What's in it? |
| --- | --- |
| `internal/jsrt/query.js` | `Watcher` – The line records the reading; `unwatched` – the string is one step in its entirety; `Reads` and `Asked.settle` – the result of reading; `shaper` – the columns of reading; `#rows`, `#plan` – the request and its lines |
| `internal/jsrt/load.js` | `copy` and `reader` are part of the same text as the template reading; `Run.compute` is the summation; `Run.list` is the list of pages as a reader |
| `internal/evaluate/evaluate.go` | `Run`; `fingerprinted` - without base; `tracked`, `part`, `relist` - by journal; `checked` - Full check |
| `internal/evaluate/session.go` | one protocol for both modes |
| `internal/evaluate/keeper.go` | `put`: Record, print, reading; who was not foreseen |
| `internal/evaluate/track.go`, `internal/data/` | from option 2 unchanged: types of readings that were hit by a line, log and triggers |
| `internal/compile/bind.go`, `reads.go`, `bundle.go` | who asks the base: request in template, component with request |
| `examples/catalog/` | `select` stand shop |

Tests: `test/reads_test.go` - columns read by data code, template and no one; reading the whole (`...`, `JSON.stringify`, function, ordinary component); cut requests; connections; route from `props`; `select`; line left in the module; verification of `-check`; who can ask the base. `test/data_test.go` is an option 2 test that is not changed. `test/fingerprints_test.go`, `test/random_test.go`, is an option 1 test that is not changed. One test is recorded: the protocol between Go and JS, `internal/evaluate/evaluate_test.go`.

```sh
go test ./...                          # on Bun
SHUGO_RUNTIME=node go test ./test/     # on Node
```
