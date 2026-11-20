# Variant 2: data in a database that keeps a journal of itself

A test implementation, to be compared with variant 1 (fingerprints by a dependency tree). Branch `lab/query`, on top of `lab/base`.

## The idea

All data of a site lives in one SQLite database that Shugo controls. Pages read it only through `Shugo.query`. Two things follow:

- **Shugo knows who read what.** Every query that runs leaves a note with the page or shared block that made it: which rows of which table, which columns.
- **The database knows what changed.** Triggers that Shugo puts on every table write each change of a row into a journal.

An update reads the journal, finds who read the changed rows, and computes those pages and shared blocks. Nothing is loaded and compared. The cost of an update follows the size of the change, not the size of the site.

## How an update goes

```
data/site.db ── journal: rows touched since the last update
      │
      ▼
Go: which rows are really other than they were        internal/data
Go: who read them — one lookup per kind of reading    internal/evaluate/track.go
      │   pages, shared blocks, lists of pages of routes
      ▼
JS: lists those routes again, computes those pages and blocks, with their new readings
      ▼
Go: writes the records that differ, replaces the readings, notes where the journal stands
```

If nothing was reached, the JavaScript process is not even started.

Everything is computed instead when the journal cannot tell what changed: the first time, after the project or the tables of the database changed, when another database stands in place of the old one, and on `shugo build`.

## What the author of a site writes

```
<!-- pages/products/[slug].shugo -->
---
export function getStaticPaths({ query }) {
  return query("products").select("slug").all().map(({ slug }) => ({ params: { slug } }))
}
const item = Shugo.query("products").select("title", "price").where({ slug: Shugo.params.slug }).first()
const variants = Shugo.query("variants").select("title", "stock").where({ product: item.id }).order("position").all()
---
<h1>{item.title}</h1>
```

The whole API is in [docs/authoring.md](docs/authoring.md#data): `select`, `where` (equal values, `= != < <= > >= like in`, `null`), `order`, `limit`, `offset`, `join`, and `all`, `first`, `get`, `count`. `Shugo.paginate(query, { pageSize })` gives one page of a long list.

Rules and restrictions:

1. Data is read from the database, with `Shugo.query` in a page or component and with `query` in `getStaticPaths`. What is read from anywhere else is not tracked: `shugo update` does not see it change, `shugo build` reads it again.
2. Every table has a primary key of one column.
3. A query names a table, not a view.
4. A value compared with a column is of the kind of the column: a text for text, a number for numbers. Otherwise it is an error that says what to write.
5. `join` finds one row by its key or by a unique column. Many rows are a second query.
6. Routes return `params` alone; pages ask for their own rows. A route that hands `props` to its pages works, but all its pages are computed again whenever a row it read changes.
7. A query cannot be asked when a module is imported: it belongs to the page or component that asks.

## What a page depends on

A *reading* is "the rows of table T that hold these values in these columns". Readings of one *shape* differ only in the values, so a shape is stored once (`shape` table) and a reading is a shape number and a key (`reads` table, indexed by shape and key).

A changed row reaches a reading if it holds, or held, the values of the key, and if it was added, removed, or changed in a column that matters to the reading.

| Query | Readings |
| --- | --- |
| `query("products").select("title", "price").get("p7")` | products by `id` = p7; matters: title, price |
| `query("variants").select("title", "stock").where({ product: "p7" }).order("position").all()` | variants by `product` = p7; matters: product, position, title, stock |
| `query("reviews").where({ product: "p7", approved: true }).all()` | reviews by `approved, product` = 1, p7. A review that is not approved does not reach it |
| `query("products").where({ category: "c9" }).count()` | products by `category` = c9; matters: category. A price does not reach it |
| `query("products").select("id").where({ category: "c9" }).order("position").limit(24).offset(48).all()` | products by `category` = c9; matters: category, position. And, when it takes columns that do not decide the rows: every returned row by its key |
| `query("products").select("title").where("id", "in", ["p1", "p3"]).all()` | one reading for each value of the list |
| `….join("categories", "category", { select: ["title"] })` | for every row: categories by `id` = its category; matters: title |
| `query("products").select("slug").all()` in `getStaticPaths` | products by nothing; matters: slug. Every added or removed product reaches it |

A page keeps the list of its readings (`page.reads`), so computing it again writes only the readings that differ — usually none.

In the catalog example a product page makes five readings and a card one. The whole shop of 100 000 products has 17 shapes and about 610 thousand readings.

## Decisions taken, and why

1. **A journal kept by triggers, not `updated_at`.** `updated_at` cannot tell that a row was removed, has to be scanned for on every table, and whoever writes the row can forget it — and then a page is stale without a word. Triggers see inserts, updates and deletes alike and cannot be forgotten. The journal line holds what the row was, so an update knows which columns changed and which readings the row used to match.
2. **`INSERT OR REPLACE` is covered.** SQLite removes the replaced row without firing delete triggers. A `BEFORE` trigger notes the row that is about to be pushed out — by its key or by any unique index. The same for `UPDATE OR REPLACE`. Tested.
3. **Only real changes count.** The update trigger fires only when a value differs; and of every row the first journal line is compared with what the row is now. A row written with what it holds, changed and changed back, or removed and put back is no change. A sync that upserts everything costs journal lines to read (1.8 s for 100 000 rows) but computes nothing.
4. **The journal has a name; a changed set of tables gives it a new one.** A table added later had no triggers for a while, so the journal cannot vouch for what happened: Shugo installs the triggers, renames the journal, and the next update computes everything. Same when a trigger was dropped by hand, and when another database file stands in place of the old one.
5. **The journal is trimmed** at the start of an update, up to the position the previous update saved in the store: those lines are dealt with for good. The database keeps a mark of how far it was trimmed, so a second site reading the same database notices that lines are gone and computes everything rather than miss them. Without trimming the database would grow with every sync.
6. **A table needs a primary key of one column.** It is the identity of a row in the journal and in the readings. `rowid` is not stable across `VACUUM`; composite keys would complicate every lookup. A table of pairs gets `id INTEGER PRIMARY KEY`.
7. **Queries run inside the JavaScript process** (`bun:sqlite`, `node:sqlite`), in one read transaction: a run sees one state of the data. Asking Go for every row would cost a round trip per query. Where the journal stands is read before that transaction starts, so a row that changes during a run is computed again by the next update.
8. **`Shugo.query` is a method of the per-component `Shugo`**, and `getStaticPaths` gets `query` as an argument. So the framework always knows who asks, without hidden state, and a query at the top of a module is impossible by construction.
9. **A query that returns all rows its values find is one reading**, with the selected columns in it. A query cut by a limit reads the rows it returned one by one (by key), because rows behind the limit must not reach it through columns it only displays. Looking for a unique value is always one reading.
10. **Soundness over precision.** A query that is not narrowed by equal values (`where("price", "<", 8)`, or no `where`) is reached by every added and removed row of the table and by every change of the columns it uses. A filter beside equal values does not narrow the reading either.
11. **Values must be of the kind of the column.** SQLite compares `'5'` with a number column as 5; the key of a reading would not. An error at the place is better than a missed update. Equality is exact (`COLLATE BINARY`) for the same reason.
12. **No tie-break is added to `order`.** An earlier version appended the key to every order; SQLite then sorted a whole category for every page of a list (the first build took 21 s instead of 13). Rows come as the database finds them unless the query orders them.
13. **`join` is a lookup of one row per row**, done as a second prepared statement, not as SQL `JOIN`: each joined row is then a reading by its key, exact for inserts, deletes and changes. A joined row that is not there yet is a reading too: it reaches the page when it appears.
14. **The list of pages of a route is a reader like any other.** Only routes whose readings were reached are listed again; new pages are computed, missing ones removed. A page is computed from its address alone (Go knows its route, the JS side reads the parameters from the address). A more exact page still hides a dynamic one when its address appears later.
15. **A shared block that fails after a change of data is an error only if something still shows it** when the update is over. A product is deleted: its card is reached, cannot be computed, and is dropped because no page shows it any more. If a page does still show it, the update fails and the store stays as it was — and the next update meets the change again.
16. **A project without `data` in shugo.json is computed whole on every update**, as before: there is nothing to tell what changed. All earlier tests run this way.
17. **`shugo build` computes everything**; `shugo update` follows the journal; `shugo dev` follows the journal when only the database file changed. In dev mode one connection to the database stays open: SQLite removes and re-creates its side files when the last connection closes, and every update would look like a change of the data and set off the next.
18. **The store keeps what every reader read** (`reads` column of pages, shared blocks and routes) and the same turned around (`reads` table). The first is what makes replacing a reader's readings cheap; the second is what a changed row is looked up in.

## Results

Xeon E3-1270 v6 (4 cores), Go 1.27.4, Bun 1.4.4, Node 24.22.1, 20 November 2026. Another task was running benchmarks on the machine (load 2–3), so the times are preliminary. `update` is what the command reports; `command` is the whole `shugo update` process, start included.

Every scenario at every size: the computed pages and shared blocks are the expected ones, and the site updated step by step equals a build from scratch, file for file.

### 100 000 products: 104 202 pages, 100 009 shared blocks

| Change | Rows changed | Pages computed | Shared blocks computed | Pages with new HTML | update, ms | command, ms |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Nothing | 0 | 0 | 0 | 0 | 4–127 | 10–136 |
| A field no page shows, on 5 000 products | 5 000 | 0 | 0 | 0 | 119 | 128 |
| The price of one product | 1 | 1 | 1 | 7 | 30 | 38 |
| The prices of 20 products | 20 | 20 | 20 | 70 | 46 | 56 |
| The title of the product of the month | 1 | 1 | 2 | 104 202 | 132 | 141 |
| Another product of the month | 1 | 0 | 1 | 104 202 | 130 | 139 |
| No product of the month | 1 | 0 | 1 | 104 202 | 126 | 136 |
| A product moves to another category | 1 | 168 | 1 | 104 202 | 301 | 319 |
| The phone in the footer | 1 | 0 | 1 | 104 202 | 129 | 139 |
| A discount appears | 1 | 1 | 1 | 7 | 25 | 34 |
| The text of an approved review | 1 | 1 | 0 | 1 | 23 | 31 |
| A hidden review is added | 1 | 0 | 0 | 0 | 6 | 14 |
| The stock of one variant | 1 | 1 | 0 | 1 | 24 | 32 |
| A variant is added first | 1 | 1 | 0 | 1 | 23 | 32 |
| The first variant is removed | 1 | 1 | 0 | 1 | 29 | 38 |
| The badges of one product | 1 | 1 | 1 | 1 | 29 | 37 |
| Two related products change places | 4 | 1 | 0 | 1 | 26 | 34 |
| A related product is removed | 1 | 1 | 0 | 1 | 27 | 35 |
| The last product is deleted | 15 | 89 | 0 | 88 | 398 | 410 |
| A product is added | 5 | 86 | 2 | 104 202 | 500 | 511 |

- First build: 13.0–14.8 s (the base without tracking: 8.8–9.0 s). Queries cost about 3.4 s of JavaScript for 620 thousand of them; keeping 610 thousand readings costs the rest.
- The store: 140–147 MB (the base: 103 MB). The data database: 80 MB.
- "Nothing" measured 4 ms (10 ms for the command) on a quiet moment and 127 ms once right after the build.
- Where every page shows the changed block (header, footer), 90 ms of the update is counting the 104 202 pages whose HTML differs.
- Adding or removing a product costs 0.25 s for listing the route again: 100 000 addresses are asked for and compared with the store.
- Outside the stand: 20 000 prices changed at once — 5.0 s, 20 000 pages and 20 000 cards computed. All 100 001 products changed in a column no page takes — 1.8 s, nothing computed.

### 10 000 products: 10 452 pages, 10 009 shared blocks

First build 1.7–2.0 s. The price of one product: update 27 ms, command 34 ms on Bun; 60 and 67 ms on Node. A product is added: 82 and 91 ms on Bun; 129 and 137 ms on Node. All twenty changes: 15–130 ms.

### 1 000 products

First build 0.26 s; updates 20–41 ms, 5 ms when nothing is computed.

Run it: `go build -o bin/shugo ./cmd/shugo && node examples/catalog/stand.mjs --size 10000` (`--runtime node` for Node).

## What is not done, or is weak

- **Precision stops at the select list.** A page that selects a column and does not show it is still computed when the column changes. Narrowing by what the template really reads — the dependency tree of variant 1 — can be layered on top: the readings already carry the set of columns that matter; the compiler would shrink it. Not built.
- **Listing a route again is proportional to the route**: 0.25 s for 100 000 pages when a product is added or removed. The fix is to ask the route only about the rows that changed; not designed.
- **A route that hands out `props`** has all its pages computed again when a row it read changes. Comparing a fingerprint of the props of each page would make it exact.
- **The query builder is small**: one table; `AND` only; no `OR`, no grouping, no aggregates but `count`, no raw SQL (raw SQL cannot be tracked). `join` is one row per row; no filter or order by a joined column.
- **Unique columns** are recognised only as single-column unique indexes.
- **Views, virtual tables** (FTS) and tables with more than 127 columns are not supported. A view is refused with an explanation; a virtual table makes installing the journal fail.
- **Keys above 2^53** and blobs as keys or values of comparisons are not supported.
- **State kept in a module** between pages (a cache of rows filled by one page and read by another) defeats the tracking: the second page made no query. Nothing detects it.
- **The writer's SQLite must have the JSON functions** (built in since 3.38): the triggers use `json_array`.
- **The journal grows between updates** by a line for every touch of a row, also for upserts that change nothing.
- **Pages with new HTML** are counted by walking from the changed shared blocks to their pages: 90 ms when that is all 104 202 of them.
- **Dev mode** follows the journal on a change of the database file; tried by hand, this path has no automatic test. After an update that trimmed the journal one more empty update runs (8 ms).
- The first build is 1.5 times slower than without tracking.

## Where to look first

1. [internal/jsrt/query.js](internal/jsrt/query.js) — the query builder and what a query leaves as readings (`#plan`).
2. [internal/data/data.go](internal/data/data.go), [internal/data/changes.go](internal/data/changes.go) — the journal, its triggers, the rows that changed.
3. [internal/evaluate/track.go](internal/evaluate/track.go) — shapes, `reached`, keeping the readings of a reader.
4. [internal/evaluate/evaluate.go](internal/evaluate/evaluate.go) — `tracked`, `part`, `relist`: what an update does.
5. [test/data_test.go](test/data_test.go) — whole sites updated row by row, with the exact set of pages computed after each change.
6. [examples/catalog](examples/catalog) — the shop of the stand, ported.
