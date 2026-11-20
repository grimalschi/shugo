# Variant 5: database change journal and row read tracking

On the `lab/query-signals` branch, this prototype extends variant 2 with row read tracking and keeps the original query model without a dependency tree.

- **Update — as in option 2** (`lab/query`) The data are in the SQLite database of the project, the database is maintained by a change log, and only the changes are calculated.
- **No addiction tree.** There are no two sides, "his own part" and three rules of option 1. One world JavaScript, one build, base compiler.
- **The page depends only on those columns of lines that are actually read by anyone.** The string that `Shugo.query` returned records the readings all the time while the person who requested it is being calculated: in the page code, in the template, in the functions, in the ordinary components that were handed over to the line. The `select` list is not needed for this.
- **Database indices are built by Shugo**, on request of the project. The author of the index site does not write.
- **The lines always come in the same order.** Where order request did not ask or order the lines, they follow the key. Without this index, the Shugo changed what the page had already shown.

This branch started from `lab/query`. The query module with row watchers, the `-check` option, URL-derived page parameters, and the Node row prototype came from `lab/query-tree`. See the earlier notes with `git show lab/query:VARIANT.md` and `git show lab/query-tree:VARIANT.md`. Their decisions carry forward except where this document says otherwise.

## How it works

1. **Data** is located in the SQLite database, named `shugo.json`: `{ "data": "data/site.db" }`. Shugo puts triggers on each table, and they write every string change together with the previous values.
2. **Request.** Page, mock-up or component asks the base via `Shugo.query`, route through `query` in `getStaticPaths`. When a request for this type is first made, Shugo builds an index for it.
3. **Reading.** Each line of answer is wrapped in a "guard." Whoever reads it, the row watcher records the name of the column on the account of the person who requested the line: pages, common block or list of pages of the route.
4. **Outcome.** When the page is ready, reading is down. Each request becomes "reading": rows of which tables with what values were sought and which columns they have important. Reading goes to Go with the record. Then the line cannot read itself.
5. **Update.** Go reads the journal and learns which lines and columns have changed. By reading, it finds pages, common blocks and lists of pages that have been touched and only calculates them. If no one has been touched, JS does not start.

```
data/site.db ── journal: which lines have changed and which columns
      │
      ▼
Go: Who read these lines in these columns ── pages, common blocks, lists of route pages
      │  "comply these"
      ▼
eval.mjs: page code and template ──► Shugo.query ──► line with guard
      │   the functions and components read the same lines; the watchman records each column
      ▼
recording and reading ──► Go: storage facility
```

It's all being re-calculated when the journal can't tell what changed, first time, after a change in the design or the base tables, and on `shugo build` command.

## What a site author writes

```
<!-- pages/products/[slug].shugo -->
---
import Variant from "../../components/Variant.shugo"
import { finalPrice, price } from "../../lib/catalog"

export function getStaticPaths({ query }) {
  return query("products").select("slug").all().map(({ slug }) => ({ params: { slug } }))
}

const item = Shugo.query("products").where({ slug: Shugo.params.slug }).first()
const variants = Shugo.query("variants").where({ product: item.id }).order("position").all()
---
<h1>{item.title}</h1>
<g-if cond={item.discount}>
  <strong>{price(finalPrice(item))}</strong> up to {item.discount_until}
</g-if>
<g-else><strong>{price(item.price)}</strong></g-else>
<g-for each={variants} as="variant"><Variant variant={variant} /></g-for>
```

Requests are all lined up. The function gets the product right in the template, the component gets the whole version. The page depends on `title`, `price` and `discount` - read by `finalPrice`, `discount_until` - only as long as the products have a discount, and `title` and `stock` - read by the `Variant` component. The change of `description`, `note` or the shelves of the option does not calculate this page.

There are no rules for option 1:

- the function can be declared in the page code and called in the template;
- the base can be asked directly in the template and in the function of the module that `Shugo` has been transferred to;
- any parameter that asks the base is given;
- the functions that trigger the template need not be transferred to a separate module without downloading data.

`select` is only left as a way to take less from the base: in `getStaticPaths` above it takes one column from 100,000 lines.

## What counts as a read

The string is a regular item wrapped in `Proxy`. The wrapping (in `Watcher` code, "guard") records the name of each column that the line reads.

| What's done with the row | What is considered read |
| --- | --- |
| `item.price`, `const { title } = item` | this column |
| `"price" in item`, `item.hasOwnProperty("price")` | this column |
| `{item.title}`, `href={item.slug}` in the template | this column |
| `{price(finalPrice(item))}` - Function from a module or page code | that the function read: `price` and `discount` |
| `<Variant variant={variant} />` — A Conventional Component | what the component read: `title` and `stock` |
| `<Frame>{item.title}</Frame>` — in a component slot or common block | this column, on the page: the contents of the lote is calculated by it |
| `cond={item}`, `item ? … : …` | nothing: only asked if there was a line |
| `{item}`, `title={item}` — All line printed | nothing: it's `[object Object]` like any object. |
| `item.toString`, `item.then`, `item.length`, Symbols | nothing: it's not columns. |
| `{ ...item }`, `JSON.stringify(item)`, `Object.keys(item)`, `for … in`, `<p {...item}>` | line whole |
| line given to pages via `props` of `getStaticPaths` | complete line, on the page list |

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
- **The lines of the request, which are cut by `limit`, `offset` or `first()`, are considered to be one.** is the same line behind the trim; its changes shall not be hit by a page over columns that it just shows.
- **The result is when the record is ready:** at the page -- after its template and model, at the general block -- after its template, at the list of pages -- when `getStaticPaths` returned the list.
- **What was read in this calculation is read.** The Shablon that did not enter the `g-else` branch does not read what the branch shows.

## Why tracking actual reads works

The statement: If no amended line has affected any page reading, then there is no reason to calculate a page — the same entry would have been made on new data.

The way the page is done is in two ways only: it asks a question and reads a line column. We run it on new data and we go over steps from the previous calculation.

- **Question.** His species and its values are the same: they are derived from previous steps. What lines are found and in what order, determine the search values, filter column, order columns and key. All of this is recorded on reading. The line added, deleted or modified in such a column would hurt reading. The answer is the same: the same lines are in the same order.
- **Reading column.** The column is recorded in the reading. It would have changed, it would have hurt. So the value is the same.

Each step sees what it saw, so the calculation goes the same way: in the same branches, in the same cycle circles, in the same function challenges. The only way to get it to the other branch is to send it the value it read, and it's the same. Columns that were not read do not affect the result: they will not be read now. The same is true for the SolidJS signals.

Three things must be true, and each one is provided separately:

1. **Calculation repeated**: depends only on the project code and the reading from the database. File, network, clock, variable module is just past; this is what `shugo update -check` finds.
2. **Every reading is recorded on the person who's computing.** cannot be read after the summation and cannot be read from another's computation: it will refuse (the section "Whose Line").
3. **The order of rows is defined by the data**, not how the base found them (the section "Ordinance of the row").

Test edges checked, `TestWaysThatAreTaken` test:

| Edge | What's checked |
| --- | --- |
| A branch that didn't fit the pattern. | `arrival` of the product in stock changes -- nothing. The product has ceased to exist -- the page is calculated, went into another branch and read `arrival`. Now `arrival` hits it. The product again is available -- `arrival` again does not hurt it. |
| `a \|\| b` | `b` reads only as long as `a` is empty |
| A function that reads different | `label(row)` reads `title` at the high-value item and `note` at the cheap one. Which of them knows which one is from `price` she read. |
| A cycle that's longer. | The new line is the string that the request was looking for: it hurts reading. What she reads is read in her new computation. |
| Line for `limit` | Each is considered separately in what she read: cheap - `price`♪ at the darling's ♪ `price` and `title` |
| Value read only inside function | `finalPrice(item)` in the template: `price` and `discount` are hit, the rest is not. |

Above this is a check-in on the contact point, `adversarial-query.mjs … reads`: 13 changes along these creeks by example of the store, followed by each site being checked against the build from zero. Everybody's passing. Two of them, the "date of arrival of the products available" and the "date of end of the discount on the products without the adjustment", do not calculate anything; options 2 and 4 are calculated.

## Row ownership

The string records what it read about whoever asked it. So only he can read it and only so far is he able to figure it out. Now, who is being calculated is known at any moment, including after `await` and in a function caused by any source: the calculation goes within `AsyncLocalStorage`. The test is worth 3 ns on Bun and 13 ns on Node per column reading.

| What's done? | What's it say? |
| --- | --- |
| the line was left in the variable module and read by another page when the first one is already calculated. | `a row of the table products is read when the page, shared block or list of pages it belongs to is already computed, so whoever reads it now would not be computed again when the data changes. A row is for the script that asked for it: do not keep it in a variable of a module for others, let every script ask for the rows it uses` |
| same, but the first page is still being calculated: waiting for `await` | `a row of the table products is read by another page, shared block or list of pages than the one it belongs to, …` |
| the route left the lines in its pages module | first message |
| `Shugo` or request was left in the module and asked for another page | `the table products is asked when the page … it belongs to is already computed, … Shugo.query belongs to the script it is called in: do not keep Shugo, or a query made of it, in a variable of a module for others` |

The `TestRowKeptForAnotherScript` test, five times, the lines that the route distributed to the pages via `props` can be read: they are counted entirely on the route.

## Indexes

Without indices, each page oversells the whole table, and the time of the first build grows as a square of the number of entries: by measuring the coordinator on option 2 to 11 s instead of 0.7 s per 5,000 products. So far, the indices have created an example generator. Now Shugo is building them.

**What a index.** When this type of request is first given, JS releases its index:

1. columns for which the rows are searched with a given value, including the column of the `in` list, in the order of their names;
2. then the columns of order in their order, to the first line, which is distinguished by any two lines;
3. then the key, if you don't know the order of the lines.

The index does not need a request that is searchable by key or by column `UNIQUE`: it finds one line. It does not need a request that neither searches for nor streamlines anything. The comparisons of `<`, `like` and similar index add nothing.

**Who's building.** JS only reads. He requests the Go index and waits for the answer; Go creates the `_shugo_<table>_<columns>` index if it is not appropriate. JS then starts reading again: otherwise he will not see the new index.

**Which index is good for yours.** Key of the Table, column `UNIQUE`, index made by hands, its previous.

- Any index that starts with search columns is good for the account, in any order.
- For rows, the index should be exactly the same as the column of the request: first, the search columns in any order, then the order columns, then the key -- and that ends. The index that goes further or doesn't reach the key, the strings will not in the order: SQLite would have to sort them and read all the lines found, including those that cut `limit`. Option 2 already measured this: 21 s instead of 13.
- The table that holds the key lines itself -- the whole number key or the `WITHOUT ROWID` created -- each index ends with the key. There's no name for the key in the index, and the index is good without it.
- The index for the lines, by expression or other comparison is not appropriate.

**Journal and `shugo dev`.** Index is not a change in tables or a change in data. The journal's name is being taken out of the triggers and the index is not changed, so the next update goes on the log. `shugo dev`, after recording the database file, asks the log if it has a new one and updates the site only then. He used to start an update on any file entry, and the index would have been too much.

**Why can you start reading again in the middle of computation?** Go remembers the space in the journal before JS read the first line. Starting a new reading means reading the base's condition later, no earlier. Everything that's changed in the database between two readings is in the journal after the memory, and the next update will figure out who read it. The same is true of the change that occurred during the build without any indices. Before the next update, the page can show a mixture of two states; the `TestChangeWhileAnIndexIsMade` test shows this mixture and that the update removes it. The way to get through this reasoning was not the way that SQLite could not read the index created after it started, and all queries could not be found before the calculation began.

**Which is built on the example of a store.** Eight indices:

| Index | What for? |
| --- | --- |
| `_shugo_products_category` | category metering in hat |
| `_shugo_products_position_id` | first 24 items on main |
| `_shugo_products_category_position_id` | pages of categories |
| `_shugo_variants_product_position_sku` | options |
| `_shugo_badges_product_position` | products badges |
| `_shugo_related_product_position` | recommendations |
| `_shugo_reviews_approved_product` | approved product reviews |
| `_shugo_categories_position_id` | menu |

| | 1,000 products | 10,000 | 100,000 |
| --- | ---: | ---: | ---: |
| Build eight indices | 8–9 ms | 57–80 ms | 0.78–0.90 s |
| The first build in which they are built, Bun | 0.25 s | 1.66–1.73 s | 15.1–15.4 s |
| The first build, when they already exist, Bun | — | 1.55–1.69 s | 14.4–14.7 s |
| Option 2 on generator indices, under the same conditions | — | 1.50–1.62 s | 13.6–14.1 s |

With Shugo indices, the first build of 2,000 items is 0.45 s, 5,000- 0.88 s. Without indices, the coordinator measured 2 1.9 s and 11 s.

The bench database per 100,000 products: 60 MB without indices, 83 MB with eight Shugo indices; with seven generator indices there were 83 MB. The update is as much as it cost: the table in the "Stand" section.

## Row order

Option 2 decided that "the strings come as the base found them if the order request is not made". With Shugo indices, it's no longer harmless. Checked: The page takes out the products in the category without order, and for it an index is built in the category, and the products are as set out in the table. Then another page asks the same products by name, builds a second index, and SQLite starts giving the first page by name. The first page is already calculated and shows the old order; the build from zero on the same base shows a new one.

So every request that takes a line ends with the key: `ORDER BY position, id`. What the page shows now is only from the lines themselves:

- doesn't depend on the index that the lines found;
- does not depend on where the line is in the table. The line was deleted and inserted, or recorded on top of `INSERT OR REPLACE`, which ends with no change, and the page with lines had previously been in order to be removed from zero. `TestRowsComeInTheOrderOfTheirKeys` test;
- the same data, which are downloaded back to the database, gives the same website.

Price per 100,000 products:

- key-text tables index: 3 MB of 83 databases;
- there is nothing to sort SQLite: checked by sample request plans;
- the time of the first build and updates has not changed on Bun;
- as at Node The first build was about 0.5 s of 17.5. The list of items now follows the key, the pages are listed in this order and the lines in the database are in another line. `order("position")` The 0.5 c is back on the address list.

Another way to be checked and discarded is to organize the row in the table (`rowid`). It's free, but it doesn't protect the same line from being removed and inserted. The third is the key-text tables to create `WITHOUT ROWID`: The stand passes, but the first build of 100,000 products takes 23.5 s instead of 15.1, because `OFFSET` on such a table goes into every missing line.

## Row watcher cost

The first build of 100,000 items, in turn, is the same example and data. "No row watcher" is the same program that doesn't turn up; "without checking the owner" turns around, but doesn't ask who reads.

| Calculation of pages and blocks | Bun | Node |
| --- | ---: | ---: |
| As it is. | 7.5–7.9 s | 10.6 s |
| Without owner checking | 7.6–8.5 s | 10.5–10.6 s |
| Without a row watcher | 7.2–7.5 s | 10.6–10.7 s |

The build was wrapped in 1.2 million lines and read through the row watcher 3.3 million columns. On Bun, this is 0.3–0.35 s, which is 2%–3% of the first build and about 0.1 μms for reading with the creation of the row watcher. Node does not show the difference. The owner's check is not visible. The row watcher does not affect the time of the update: there are pages.

## Rules for site authors

1. **Data are read from the project base.** read past it -- file, network -- does not track: `shugo update` does not see it, `shugo build` reads it over again, `shugo update -check` will name pages that depend on it.
2. **The string and `Shugo` belong to the person who is being calculated.** cannot be left in the variable module for other pages.
3. **The table has a single column key; the request goes to the table, not to the presentation; the value is the same as the column.** These are the rules of option 2.
4. **The author does not write page indices.**
5. **Order that means something is stored in the column**, and the request calls it `order("position")`. Without this line, the key is followed.

Full management: [docs/authoring.md](docs/authoring.md), sections "Pages from Data" and "Data".

## Decisions made during implementation

We can review each one.

1. **Started with option 2; option 4 took the request module.** The compiler, build and protocol of option 2 are intact. `query.js` option 4 removed `unwatched` - take the whole line one step: this required passage to "its own part" that is not here.
2. **What is read in the calculation is counted.** Why is this correct is the section above. Accuracy goes through functions and components itself: the row watcher doesn't care who reads.
3. **Only columns are counted.** `toString`, `then`, `length`, symbols ask many of the lines -- `await`, seal, library checks -- that says nothing about data, and "read everything" would be a false addiction because of this.
4. **A line printed in its entirety, reads nothing** and printed as `[object Object]`. This is how any object behaves; printing all columns for the author would mean showing what he did not ask for.
5. **Line among attributes, `<p {...item}>` read whole**: All its columns are marked.
6. **The contents of the common block slot read the pages of the page.** The Common Unit is counted once and does not know about the page; what the page has placed in its slot is the page. The Test is the page `fill`.
7. **The line owner checks `AsyncLocalStorage`.** Option 4 only caught reading after the applicant had been counted; two pages that were waiting for data in a mix could read the other's things undetected. Here it is caught and this: only the person who is being counted can read.
8. **`Shugo` also belongs to its calculus.** `Shugo` or the request left in the module would have finished the readings in the sum of the text already given. The request is not valid with a clue.
9. **The route that gives lines to pages through `props` reads them all.** The pages read these lines after the list has given its reading. The old Council: the route gives addresses, the page asks for it.
10. **Page parameters always come from her address.**, as in option 4.
11. **`shugo update -check` works with base**, as in option 4: calculates everything and calls it pages whose recording has changed, although the journal has not reached them. For 100,000 items, the check takes 7.6 s.
12. **Node is a regular prototype, not a copy.**, as in option 4.
13. **The index is requested in two parts: search columns and order columns.** Index search columns can be in any order, order columns can not. The index made by the hands on `(product, approved)` is the same as the one that Shugo would build `(approved, product)`.
14. **Whether the request has an index, it decides Go.** He knows which indices are unique and that the whole key table is on the key. JS says only that a request is needed.
15. **The lines come in the order of request, then the key.** Line Order Section. This overrides option 12 ("the order is not supplemented"), because the reason for it, sorting each request, is that the index ends with a key.
16. **The line index should end where the order of request ends.** otherwise SQLite sorts out. Therefore, "starts with the same columns" from the assignment is left only for the count.
17. **The indices are not removed**, including the short one, when the long line with the same start appeared: the lines found in the short columns are coming out of it in their order. First the removal was made and removed for this reason.
18. **The index is constructed in the same calculation where the first request is made**, including update. `TestIndexMadeByAnUpdate` test.
19. **The column `UNIQUE` is considered to be the only one, only if the index covers all lines.** Partially unique index option 2 was taken as complete.
20. **The column distinguishes between any two lines if it is `UNIQUE` and `NOT NULL`.** after this order has nothing to add to, and the index is not going further.
21. **In the stand, instead of "articula of option", "pole of option"**, as in option 4.

## Limitations

- **Mass changes cost 7-8% more.** Reading in branches gives more readings: four instead of one, at a discount and without, available and not available. For every changed line Go, readers look for each type of reading of its table. This is 15 μms per line: 5.3–5.8 s instead of 5.0–5.3 per 20,000 items. He treats himself in a vault: to look for readers once on a table and search columns, and to keep important columns when reading. Not done.
- **First build** on Bun is more than option 2 by 0.7 s of 14 and option 4 by 0.4 s. Of these, 0.3–0.35 c is a row watcher, the rest is full. On Node, it is more than 0.5 s of 17.4 option 2, and the row watcher is not visible and so much is worth the key order on the list of items.
- **First build in which the indices are built**, longer by 0.8–0.9 s per 100,000 items.
- **Indexes that no longer need any request are removed.** Not now: You need to know all the project requests, and they are only known after full calculation.
- **Which indices are built depends a little on the order of requests.** The account set earlier on the list receives its short index; later, it uses a long one. In the example, `_shugo_products_category`. The result is not affected.
- **Orders in different directions**, `order("a").order("b", "desc")`: The index is built, but SQLite still sorts.
- **Reading starts again when the index is built.** If written in the database at this time, the page can show two data states before the next update.
- **By the base.** File or web in page code, simple values (not rows) left in the variable module, a copy of the `{ ...row }` line left in the same location are not monitored. Only `shugo update -check` is found.
- **A timer or a promise no one expects,** and reading the line after the page is calculated: build drops with the same message but no page name.
- **`structuredClone(row)` is not operational:** line – `Proxy`. `{ ...row }` works and reads the whole line.
- **`select` is needed for speed** where there are many lines and few readings, without which the list of 100,000 items is read 0.1 with longer time, at first build and at each addition and disposal.
- **`props` route** reads the entire lines and calculates all its pages when it changes.
- **The weak option 2 is:** is small; the route is listed in its entirety when the line has been added or deleted - 0.26 s per 100,000 pages; as external sources are entering the base, it is not done.
- **`shugo dev` with base** is checked by hand: first build, one calculation and eight indices; nothing recorded, value recorded on top of itself, index made with hands, none; price, phone, new table, one each; automatic test not available.
- **Documents in `spec/` and root `README.md` in this branch were not updated** except `docs/authoring.md`.

## Benchmark

```sh
go build -o bin/shugo ./cmd/shugo
node examples/catalog/stand.mjs --size 10000        # --runtime node -- compute on Node
```

The bar changes the data 21 times with the usual SQL requests, the same changes as in option 4. After each, it checks how many pages and common blocks are calculated and at the end compares the step-up website with the zero-size build, the file behind the file. For 1,000, 10,000 and 100,000 items, Bun and Node, all checks are passed, sites are consistent. The top of the bench is covered by two checks by the coordinator: 23 changes from the version 2 test and 13 changes along the red-reading test; the site was each matched with a zero-tongue build.

The example of a store is an example of option 2 without `select` lists, except for the list of addresses of products. It is different from the example of option 4. `finalPrice(item)` is called directly in the card template and page of the product, not in the page code. And the data generator doesn't create indices. The price function file is called `lib/catalog.ts`, as in option 2: there is no separate `lib/money.ts` because there is no rule that required it.

### 100,000 products, 104,202 pages, and 100,009 shared blocks

Xeon E3-1270 v6, Go 1.27.4, Bun 1.4.4, Node 24.22.1, 20 November 2026. Vehicle load 1.5–3 Time is the entire `shugo update` command together with launch.

| What changed the data | The string changed | Pages calculated | Common blocks calculated | Pages with new HTML | Bun, ms | Node, ms |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Nothing. | 0 | 0 | 0 | 0 | 185 | 172 |
| A column that nobody reads has 5,000 items. | 5,000 | 0 | 0 | 0 | 115 | 111 |
| Price of one commodity | 1 | 1 | 1 | 7 | 38 | 74 |
| Prices of 20 products | 20 | 20 | 20 | 70 | 49 | 86 |
| Name of item of month | 1 | 1 | 2 | all | 123 | 163 |
| The products of the month were another commodity | 1 | 0 | 1 | all | 123 | 155 |
| Products of the month are no longer available | 1 | 0 | 1 | all | 134 | 157 |
| Products moved to another category | 1 | 168 | 1 | all | 323 | 327 |
| The phone's in the basement. | 1 | 0 | 1 | all | 153 | 161 |
| The products have a discount. | 1 | 1 | 1 | 7 | 41 | 66 |
| Text of the approved withdrawal | 1 | 1 | 0 | 1 | 34 | 74 |
| A hidden withdrawal added | 1 | 0 | 0 | 0 | 12 | 13 |
| Balance of one option | 1 | 1 | 0 | 1 | 33 | 73 |
| **A shelf of options that no one shows** | 1 | **0** | 0 | 0 | **11** | **12** |
| Option added first | 1 | 1 | 0 | 1 | 32 | 64 |
| The first option was deleted | 1 | 1 | 0 | 1 | 33 | 68 |
| One commodity values | 1 | 1 | 1, new | 1 | 35 | 66 |
| Two recommended items were replaced | 4 | 1 | 0 | 1 | 32 | 67 |
| The recommended products were removed | 1 | 1 | 0 | 1 | 32 | 66 |
| The last item was removed. | 15 | 89 | 0; card removed | 88 | 377 | 563 |
| Products added | 5 | 86 | 2 | all | 440–552 | 650–765 |

- **One line is different from option 4, the shelf of the option.** There was a page and the record was the same, 38 ms on Bun and 91 ms on Node. There is nothing here: the `Variant` component does not read the shelf.
- The rest is calculated in the same way as options 2 and 4, and it's worth the same amount.
- Nothing goes first after the build and costs 185 ms. Later, the upgrade that didn't hit anyone takes 11–12 ms.
- The "products added" was repeated four times: 440–450 ms on Bun, 650–765 ms on Node; of these, 0.26 s and 0.45 s re-listed 100,000 addresses of the route.

### Other results

The first build and mass changes were removed for three versions in turn, under the same conditions: each version was its own program on its own example and data, in copies on the same disk.

| 100,000 products | Option 2 | Option 4 | Option 5 |
| --- | ---: | ---: | ---: |
| First build, Bun | 13.6–14.1 s | 14.0–14.2 s | 14.4–14.7 s |
| First build, Node | 17.3–17.4 s | 16.9–17.4 s | 17.4–18.1 s |
| 100 prices at once, 0.1% | 0.20 s | 0.25 s | 0.22 s |
| Prices of 1,000 products, 1% | 0.55 s | 0.52 s | 0.58 s |
| Prices 5,000 products, 5% | 1.7 s | 1.6 s | 1.9 s |
| Prices of 20,000 products, 20% | 5.0–5.3 s | 5.0–5.2 s | 5.3–5.8 s |
| Prices of all 100,000 products | 21.8 s | 21.9 s | 23.5–23.7 s |
| Storage | 139.8 MiB | 140.0 MiB | 140.7 MiB |
| Reading Types | 17 | 18 | 21 |
| Reading | 612,673 | 612,673 | 612,673 |

The first build of option 5 here is with the indices already built; with their construction 15.1 to 15.4 s on Bun and 18.4 to 18.9 seconds on Node. 10,000 items, Bun: 1.50 to 1.62 seconds, 1.45 to 1.59 seconds and 1.55 to 1.69 seconds.

This is the first build per 100,000 items, Bun: 0.02 s, 0.16 s list, 7.5- 7.8 s, 6.6- 6.7 s storage.

## Where to look in the code

| File | What's in it? |
| --- | --- |
| `internal/jsrt/query.js` | `Watcher` – The line records and checks who reads; `Reads`, `during` whose calculation is ongoing; `Asked.settle` – the result of reading; `#plan` – the request, order and index; `Data.serve` – the request for index and reading again |
| `internal/jsrt/runtime.js` | `Run.page`, `Run.block`, `Run.list` — Where Calculation Starts and where The summation is taken |
| `internal/data/index.go` | `Index` and `serves` — What Index is appropriate and what is built |
| `internal/data/changes.go` | `Moved` — has moved the log; `shugo dev` decides on it |
| `internal/evaluate/session.go` | `I` - Request for index in the middle of calculation |
| `internal/evaluate/evaluate.go`, `keeper.go` | `checked`, `foreseen` - `-check` check |
| `internal/evaluate/track.go` | option 2: types of readings that were affected by the line |
| `examples/catalog/` | stand shop |

Tests: `test/reads_test.go` — what is considered to be read; the actual reading edges; the string and `Shugo` left in the module; `-check`; the parameters from the address. `test/index_test.go` — What indices are being built, hand-made indices, order of lines, index of renewal, data changes during index construction. `internal/data/index_test.go` is the code of indices on the base; `Moved`. `test/data_test.go` and the rest of option 2 tests are not changed.

```sh
go test ./...                          # on Bun
SHUGO_RUNTIME=node go test ./test/     # on Node
```
