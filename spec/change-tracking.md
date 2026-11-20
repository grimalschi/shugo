# Change tracking

The document describes how framework learns what has changed in the data, and recalculates only what this change shows.

## Decisions

The words of the author of the project, 20 November 2026 year.

- Data change analysis is required for any step device. The choice was only where it is performed: on Go between data loading and expression calculation or on TS without an intermediate step.
- Expressions are calculated only for changed areas.
- Purpose: changing the data of one product only counts the pages that show the data of this product, and blocks on other pages that show this product.
- Data should not be marked with special calls like `source(...)`. The mechanism does not spoil the project code and leaves no holes: regular code without markings should not give an outdated page.
- You need to analyze how patterns use variables and data: the dependency tree returns.
- The template can see data code variables not in pure form, but through serialization. The author named Seroval for this; what exactly to copy is discussed below.
- How data gets into pages and components can be changed for the sake of a simple and quick comparison. The main thing is the effectiveness of updates.
- In the second variant, changes are recognized from the log that triggers on tables, and not from the mandatory field `updated_at`: the log sees inserts and deletions, and it cannot be forgotten to update. Confirmed by 20 on November 2026 year.
- A copy of the data in SQLite is part of the architecture: [Data adapter and data copy](data-adapter.md).
- The first option, fingerprints without a database, is no longer considered: options with a database work much faster and better.

## Trial implementations

The author of the project asked to make two implementations for comparison, then “the second option with a dependency tree from the first” and “the second with lines that remember what you read, without a tree” (20 November 2026). All done, pass all tests on Bun and Node and stand on a common basis. Which one to take is not decided. The 3 number in the discussion is occupied by [data Adapter](data-adapter.md): it is not a substitute for these options, but what brings data to the database of variants 2 and 4.

| | Branch & Folder | Essence |
| --- | --- | --- |
| Base | `lab/base`, `../shugo-lab/base` | Go himself names which pages and common blocks to calculate, and injects a partial result into the repository. She calculates everything |
| Variant 1 | `lab/fingerprints`, `../shugo-lab/fingerprints` | proposal from this document: data code is executed entirely, the imprint on the dependency tree, templates - only for the changed |
| Variant 2 | `lab/query`, `../shugo-lab/query` | idea of the author: data is in SQLite under the framework control, read only through `Shugo.query`, dependencies are tied to base entries |
| Variant 4 | `lab/query-tree`, `../shugo-lab/query-tree` | update as in 2, compiler and two sides as in 1; the string itself remembers which columns it has read |
| Variant 5 | `lab/query-signals`, `../shugo-lab/query-signals` | version of 2, in which the string remembers what its columns are read by anyone; without a tree and without the rules of the version of 1. The base indexes are built by Shugo itself |

Each option in the root branch is `VARIANT.md`: how it works, rules for the author of the site, taken along the way of the decision with the reasons and what is not.

### Variant 2 in two words

- The framework puts triggers on each table of the database and maintains a log: inserting, changing and deleting the record leaves a string with the same value. Mandatory `updated_at` is not needed: the log sees and deletes, and you can not forget it.
- Each query through `Shugo.query` leaves a page or general block with a “read”: which rows of which table with what values in which columns it searched and which columns it took.
- The update reads the journal, finds by reading who was touched by the changed lines, and calculates only them. If no one is hurt, JS does not start at all.
- The route pages list (`getStaticPaths`) is the same reader: the route is re-listed only when what he read has changed.

### What is calculated

Both options calculate the same thing: the price of one product - its page and its card; the name of the product of the month - page, cap and card; phone in the footer - footer; field, which is not shown anywhere - nothing. Both implementations undergo twenty bench changes on 1,000, 10,000 and 100,000 products, and the site updated in steps coincides with the build from scratch, file by file.

On top of the stands each is checked separately: two dozen changes that are not in the stands (change of product address, reshuffle of categories, change of key of record, 300 prices together, rollback of transaction, record of the same value, re-creation of table). After each change, the site coincided with the build from scratch.

### Time

Xeon E3-1270 v6, Go 1.27.4, Bun 1.4.4, Node 24.22.1. 100,000 Products: 104,202 Pages and 100,009 common blocks. Stands were in turn. Time — the whole team `shugo update`.

| Update after change, Bun | First version | Variant 1 | Variant 2 |
| --- | ---: | ---: | ---: |
| Price of one product | 3.2–3.7 s | 1.28 s | 0.03 s |
| Prices 20 products | 3.2–3.7 s | 1.21 s | 0.04 s |
| Review, option, icon, recommendation | 3.2–3.7 s | 1.2–1.3 s | 0.03 s |
| Product name of the month, phone in the footer | 3.2–3.7 s | 1.27–1.29 s | 0.12 s |
| This product has been moved to another category, 168 pages | 3.2–3.7 s | 1.38 s | 0.26 s |
| Product added or deleted | 3.2–3.7 s | 1.31–1.38 s | 0.41–0.42 s |
| What has changed is what no page shows. | 3.2–3.7 s | 1.16–1.28 s | 0.01–0.16 s |

| Many records together, Bun | Variant 1 | Variant 2 |
| --- | ---: | ---: |
| Prices 100 products, 0.1% | 1.4 s | 0.17 s |
| Prices 1,000 products, 1% | 1.7 s | 0.5 s |
| Prices 5,000 products, 5% | 2.5 s | 1.6 s |
| Prices 20,000 products, 20% | 4.3 s | 4.9 s |
| Prices of all 100,000 products | 14 s | 22 s |

| The rest | Base | Variant 1 | Variant 2 |
| --- | ---: | ---: | ---: |
| First Assembly, Bun | 8.8–9.0 s | 9.1 s | 13.8–13.9 s |
| First Assembly, Node | — | 9.8 s | 17.4 s |
| Update on Node | — | 1.9–2.2 s | 0.07–0.75 s |
| Storage | 103 MB | 112 MB | 140 MB |
| Stand Data | JSON, 62 MB | JSON, 62 MB | base, 80 MB |
| 10,000 products: first assembly and update, Bun | — | 1.0 s and 0.14–0.17 s | 1.5 s and 0.02–0.08 s |

- The time of the 1 variant is almost independent of what has changed: 0.62–0.77 s takes the data loading with the example code, 0.47–0.56 s takes the data code and 204 thousand fingerprints of pages and blocks. It grows with the site.
- The time of the 2 variant depends on the size of the change, not the site. When a fifth of the entries change and more, it is slower than the 1 variant: each calculated page makes its requests.
- The data code and fingerprints in the 1 variant were three times more expensive than in the model below: 2.5 µs per page or block instead of 0.6–0.9 µs.

### What is changing for the author

| | Variant 1 | Variant 2 |
| --- | --- | --- |
| Where data comes from | from anywhere: file, database, API | only from the project database |
| How they are read | normal TypeScript | `Shugo.query("products").select(…).where(…)` |
| What is rewritten in the example of the store | price functions are rendered in `lib/money.ts`, one call is transferred from template to data code | data transferred to database, the code of pages and components data rewritten to requests |
| Limitations | template does not see functions from the data code; component with object in parameters does not load anything; module with functions for templates does not load data | table has one key; request to one table, conditions through "and", of aggregates only `count`; `join` finds one line; views not supported |
| What depends on accuracy | from what reads the template: displays the compiler | from column list in `select`: writes the author |
| What happens in violation | build error with prompt | data past the database and the memory of the module between pages is not tracked, and no one will say |

### What is not and what is weak

Variant 1:

- The data code of all pages is executed at each update. The request for each page will be executed for each page.
- Accuracy stops at the function call and on the usual component: the object transmitted to them is read in its entirety.
- A package from npm that reads the file in synchronous form inside the template is not visible; only finds this full check of `shugo update -check`.
- The compiler is much more complicated: parsing expressions, two builds, a new dependence of `tdewolff/parse`. Changed 4,300 lines.

Variant 2:

- The main thing around it is not done: how external sources get into the base. A generator writes the base on the stand. The device is discussed in [Data Adapter](data-adapter.md).
- The query constructor is small: see the restrictions above.
- The first build is one and a half times longer: 620 thousand queries and 610 thousand reads in storage.
- The route is listed as a whole when the entry has been added or deleted: 0.25 s on 100,000 pages.
- The route that distributes `props` pages calculates all its pages when it changes.
- Changed 3,900 lines; compiler not touched, new dependencies in Go no.

### 4 variant: second with dependency tree from first

The update is in the database log, as in the 2 version. The compiler, two sides and three rules for the author are taken from the 1 variant. New one: the page depends only on those columns of the entry that are actually read, and the list of `select` is not needed for this.

How it's done. The dependency tree only knows what the pattern reads. What the data code reads like `finalPrice(item)` it doesn’t know, so narrowing the dependency to one tree would be wrong. Each row returned by the query itself records which columns it reads: the data code and the pass that copies the pattern of its part. Nothing is added to the compiler for this.

| 100,000 products, Bun | Variant 2 | Variant 4 |
| --- | ---: | ---: |
| Price of one product | 0.03 s | 0.04 s |
| Product name of the month, phone in the footer | 0.12 s | 0.12 s |
| Product added | 0.41 s | 0.46–0.58 s |
| The field that no one reads has 5,000 products | 0.12 s, nothing calculated; columns listed in `select` | 0.12 s, nothing calculated; `select` not needed |
| Prices 20,000 items together | 4.9 s | 4.9 s |
| First Assembly, Bun and Node | 13.9 s and 17.4 s | 13.9 s and 17.0 s |
| Storage | 140 MB | 140 MB |
| Changed lines regarding the base | 3 900 | 8 600 |

What was added to the 2 variant:

- Accuracy without `select`. `select` remained a way to take less from the database: in the example, it is needed in one place, in the list of addresses of one hundred thousand goods.
- Templates and regular components run on copy and cannot depend on what they are not handed; the rules of the 1 variant and their assembly errors.
- The line left in the module variable and read from another page is an error. The 2 version was silently missing this.
- `shugo update -check` works with the database: calculates everything and names pages that read something past it. On 100,000, products are 7 s.

What it does not give and what is worth:

- Speed does not add or subtract: the read entry costs 1–2% first build.
- Accuracy ends where the tree says "whole": a string passed to a function in a template or a normal component reads everything.
- The template is considered to read everything it can read, no matter what branch `g-if` goes into. Verified: the date of receipt of the product that is in stock and does not show it, still calculates the page; the record goes the same.
- The compiler of the version 1 with all its complexity and three rules for the author.

Test output. The accuracy was not given by the tree, but by the lines that remember what was read. The tree gives another: the pattern cannot depend on what is not passed to it. Lines with memory can be put on the 2 version without a tree: then the template reads them itself, the read is recorded in fact - with accuracy to the branch conditions and through functions and components - and the rules of the 1 variant are not needed. This combination is made as a variant of 5.

### 5: lines that remember what you read without a tree

A variant of 2 in which each line from `Shugo.query` records which columns it reads while the one who requested it is calculated: page code, template, functions, the usual components to which the line was passed. There is no dependency tree, two sides and three rules of the 1 variant. The compiler is not affected.

Why do you think that's true. The calculation learns about the data in two ways: it sets a request and reads a column of a line. Both are recorded. While all the previous readings, the same code goes the same way in the same branches and the same calls and gives the same. In another branch, it can only send the value it has read. Also, signals are arranged in SolidJS.

What it gave compared to the 4 variant:

| What changed | Variant 4 | Variant 5 |
| --- | --- | --- |
| Product option column that no one shows; the option is passed to the component as a whole | page calculated, record old | nothing calculated |
| The date of receipt of the goods that are available and does not show it | page calculated, record old | nothing calculated |
| Price function in template: `{price(finalPrice(item))}` | the product is read in its entirety; in the example, the calculation is made in the data code | `price` and `discount` |
| Rules for the author | three rules of the 1 variant and base rules | base rules only |

Time, 100,000 products, my runs in turn:

| | Variant 2 | Variant 4 | Variant 5 |
| --- | ---: | ---: | ---: |
| Price per item, Bun | 0.03 s | 0.04 s | 0.04 s |
| Product name of the month, phone in the footer | 0.12 s | 0.12 s | 0.12 s |
| Product added | 0.41 s | 0.46–0.58 s | 0.44 s |
| Prices 20,000 items together | 4.9 s | 4.9 s | 5.5 s |
| Prices of all 100,000 products | 22 s | 22 s | 24 s |
| First Assembly, Bun | 13.9 s | 13.9 s | 15.4 s with index construction |
| First Assembly, Node | 17.4 s | 17.0 s | 19.0 s with index construction |
| Price per item, Node | 0.07 s | 0.08 s | 0.09 s |
| Storage | 140 MB | 140 MB | 141 MB |
| Changed lines regarding the base | 3 900 | 8 600 | 6 000 |

2 and 4 variants measured on indexes that the data generator created with its hands; the 5 variant builds the indices itself, and this is included in its first build: 0.8 s from 15.4. According to the measurement of the branch, when the indexes are already there, the first build is 14.4–14.7 s.

What else is done in the 5 variant:

- **You can't read another line.** The string records what is read to the account of the person who requested it, and only he can read it and only while it is being calculated. A string left in a variable module for another page is an error, including when pages are waiting for data interspersed. The 2 variant did not notice this, the 4 variant was not always caught.
- **`shugo update -check`**, as in the 4 variant: calculates everything and names pages that read something past the base. 7.6 s on 100,000 products.
- **Shugo**: [Data adapter](data-adapter.md#indices).
- **The strings come in the order of the query and then by key.** Variant 2 gave the strings without order as they found the base. With indexes that appear themselves, this has ceased to be harmless: the new index changed the order of the lines on the already calculated page, and it diverged from the build from scratch.

What is weak:

- Massive changes are a tenth more expensive: branch reading gives more types of reading, and for every line that has changed, Go looks for readers for each type.
- The first build on Bun is longer than the 2 variant on 0.5–0.7 s from 14, not counting the construction of indices.
- The template can read something past the database again file, network, time, module variable. Finds it only `-check`.
- The weak one from the 2 option remains: a small query constructor and a route that is listed as a whole when the entry is added or deleted.

## Proposal, pending approval

### Essence

The code of these pages and common blocks is executed with each update in its entirety: this is the loading of all data. It is not necessary to monitor what it reads.

The compiler knows in advance what exactly the template reads from the data code variables. This is **dependency Tree**. For each page and each common block, **fingerprint** is considered to be a hash of exactly those values that the template reads.

Go compares prints to saved ones. The template is only run where the print has changed and gets not the variables themselves, but **part of**: a copy of what it reads.

### Stages

| Stage | Who performs | What does |
| --- | --- | --- |
| 1. Compilation | Go | three things are done from each `.shugo` file: data code for `load.mjs`, template for `render.mjs` and tree of pattern dependencies; the program for the runtime remains as is |
| 2. Downloading data | JS, `load.mjs` | executes the data code of all pages and common blocks and counts the fingerprint of each |
| 3. Comparison | Go | compares the prints to saved ones and names pages and shared blocks that have changed or appeared |
| 4. Calculating Expressions | JS, `render.mjs` | executes templates of only named pages and blocks |
| 5. Rantime | Go | no change |

These are steps from the author's first plan: the comparison is between loading and calculating and is performed on Go. The 2 and 4 stages go in the same process of JS, so data between them is not transmitted anywhere. In Go, only prints go: 8 bytes per page or block.

### Dependency Tree

Product page template, with abbreviations:

```html
<h1>{item.title}</h1>
<g-if cond={item.discount}><span>−{item.discount.percent}%</span></g-if>
<p>{price(item.price)}</p>
<g-for each={item.variants} as="variant"><Variant variant={variant} /></g-for>
<h2>Reviews: {reviews.length}</h2>
<g-for each={reviews} as="review"><p>{review.text}</p></g-for>
<g-for each={item.related} as="id"><ProductCard id={id} /></g-for>
```

Its dependency tree:

```
item
├─ title
├─ discount is or not
│  └─ percent
├─ price
├─ variants of each element
│  ├─ title this reads the Variant component
│  └─ stock
└─ related each item: these are product numbers
reviews
├─ length
└─ each element
   └─ text
```

There are no fields `item.note`, `item.category` and `item.variants[].sku` in the tree: the template does not read them, and their change to the page does not reach.

What goes into the tree:

| In the template | What is read |
| --- | --- |
| `{item.title}`, `href={item.a.b}` | one value |
| `<g-if cond={item.discount}>` | value or not; of an object only what is read next |
| `<g-for each={item.variants} as="v">` and `{v.title}` in body | the length of the list and each element has what the body reads |
| `{reviews.length}` | length |
| `{price(item.price)}` | `item.price`; `price` is an imported function, it's code |
| `{finalPrice(item)}`, `{item.tags.join(", ")}`, `{items[0].title}`, `{...attrs}` | object as a whole: what will read the function, the compiler does not know |
| `<Variant variant={variant} />` | what the `Variant` component reads; if the compiler failed to output this, `variant` whole |
| `<ProductCard id={id} />` | value `id`; card — separate common block with its own fingerprint |

“It is always safe: from less accuracy there is an unnecessary calculation, but not an outdated page.

Trees are built by compilation. esbuild erases types, the `tdewolff/parse` library parses the expression and knows what name is taken from. This bundle is tested by a separate breakdown: field chains, method calls, arrow functions and conditions it distinguishes. The sample parsed individual expressions; the output through components is not yet written.

### The imprint and its part

- **Fingerprint** is a 64 bit. JS runs through the dependency tree, reads values directly from objects and mixes them into a hash. Nothing serializes.
- **My part** is a copy of the same values collected by the same passage. It is only made for pages and blocks that need to be calculated.

The pattern gets its part, not the variables. So he can’t depend on what’s not in the print: he just didn’t get it. An error in the dependency tree is visible immediately, at the first assembly: the template will not be value. The old page won't turn in a month.

The fingerprint of a page or block is only compared to its own past fingerprint, so the 64 bit is enough.

### List Keys

In the prototype, the key gave the name to the block inside the list. Here the name is only for the page - its address - and the common block - its parameters. The list item belongs to its page or shared block and is calculated with it. Therefore, `key` in `<g-for>` is still not needed.

If the list item must be updated separately from the page, it is shown in a common block: `<ProductCard id={item.id} />`. The parameters serve as its key.

### Who uploads the data

Files remain the same as now: data code at top, template at bottom. There are three rules for the author of the site.

1. **Data loads pages, layouts and shared blocks.** Common block - as it is now: a component called with simple parameters. The usual component to which the object was transferred only shows the transmitted: its data code does not wait for anything (`await`) and does not read files and the network.
2. **The template reads data and calls imported functions.** Data is data code variables and imported values: numbers, strings, objects, lists, dates, `Map`, `Set`. The function declared in the data code will not see the template: it needs to be taken out into the module.
3. **The module whose function calls the template does not load the data itself.** Functions for templates and data loading lie in different modules.

Violation of any rule is a mistake when assembling with a place and a clue, and not an outdated page silently.

What you will need to change in the example of the store:

```ts
// lib/money.ts — new module: template functions moved from lib/catalog.ts
export function price(value: number): string { … }
export function finalPrice(product: Product): number { … }
```

```html
<!-- components/Header.shugo: usage product() moved from template to data code -->
---
import { categories, inCategory, product, shop } from "../lib/catalog"
const menu = …
const featured = shop.featured ? product(shop.featured) : null
---
<a class="brand" href="/">{shop.name}</a>
…
<g-if cond={featured}>
  <p>Product of the month: {featured.title}</p>
  <ProductCard id={featured.id} />
</g-if>
```

The product page, category list, card, option and footer remain as written, only the import string changes.

### Why an Outdated Page Won't Work

1. The data code is executed anew with each update and sees the latest data. What exactly it reads and where it comes from, it does not matter.
2. The template and the usual components receive only their part. They cannot depend on the rest of the data.
3. The template code is collected separately from the data code. These are separate instances of modules: there is no downloaded data in them. When assembling, it is checked that this code does not reach files, network, time and random numbers.

Third point checks tested on esbuild separately from the framework:

| What's in the code that triggers the pattern | What the test showed |
| --- | --- |
| module reads file when importing | `node:fs` remains in the collected code, and esbuild names the module that imports it: the compiler will report an error |
| file reads a function that the template does not call | esbuild removes it along with import; no error |
| module waiting network at import (`await fetch`) | error esbuild: template code is collected in a format without `await` top level |
| `fetch`, `Date.now()`, `Math.random()`, `process.env` | esbuild replaces the name with a stub; the stub will give a clear error |

`new Date()` does not see without arguments esbuild; it will find the analysis of expressions that is needed for the dependency tree.

One case remains: a package from npm that reads the file or database in synchronous fashion inside a normal component. The compiler does not look into the packages. This remains a complete check: the team calculates the entire site, as the first version does, and names the block whose result changed with the previous print.

### What is calculated after each change

Model on the stand data: 100,000 products, 204 thousands of pages and common blocks. Dependency trees are written by hand as the compiler would have them; the compiler itself is not yet written. The first version for any of these changes calculates all 204 thousands.

| What changed in the data | Pages to calculate | Total blocks to calculate | Pages with new HTML |
| --- | ---: | ---: | ---: |
| Nothing. | 0 | 0 | 0 |
| Field that is not shown anywhere, 5,000 has products | 0 | 0 | 0 |
| Price of one product | 1 | 1: his card | 7 |
| Prices 20 products | 20 | 20 | 109 |
| Product Name of the Month | 1 | 2: cap and card | all |
| The goods of the month became another product | 0 | 1: cap | all |
| Phone in the footer | 0 | 1: footer | all |
| The product was moved to another category | 168: lists of two categories | 0 | 168 |
| Text of the approved review | 1 | 0 | 1 |
| One option balance | 1 | 0 | 1 |
| An option that is not shown anywhere | 0 | 0 | 0 |
| The first option is removed | 1 | 0 | 1 |
| Two recommended products changed places | 1 | 0 | 1 |
| Last Product Removed | 88 | 0; product page and card removed | 88 |
| Product added | 85 | 2: his card and header | all |

The price of the product is shown on seven pages. Its page and its card are calculated; the remaining six pages refer to the card by name and are not touched.

### How much is it worth

Xeon E3-1270 v6, Bun 1.4.4, Node 24.22.1, Go 1.27.4. 100,000 products, data file 65 MB.

Everything JS does in the update before calculating the changed:

| Step | Bun | Node |
| --- | ---: | ---: |
| Project code: read data file | 0.31–0.37 s | 0.61–0.71 s |
| Project code: data module and `getStaticPaths` | 0.15–0.27 s | 0.26–0.35 s |
| Data code 100,000 pages of goods and 100,000 cards | 0.05–0.07 s | 0.07–0.16 s |
| Prints of all pages and common blocks | 0.08–0.10 s | 0.10–0.15 s |
| Keys and fingerprints for Go, 6 MB | 0.02–0.04 s | 0.04–0.05 s |
| **Total** | **0.61–0.78 s** | **1.19–1.36 s** |

Go side: read 204 thousands of saved prints and compare - 0.16–0.18 s if the print is in its line of entry, and 0.01 s if all prints are in one value.

The whole upgrade is based on the sum of 0.7–1.0 s parts on Bun and 1.3–1.6 s parts on Node. This is an estimate: the whole mechanism has not yet been assembled. Now 3.2–3.7 s and 4.3–4.7 s occupy the update.

From this time framework itself occupies 0.2–0.4 s. The rest is the project code that 65 MB and 58 read once passes through the list of products.

The part for the template costs 0.2–0.5 µs on the product page. A full copy of the variables through `structuredClone` costs 7–11 µs.

### What is tested and not suitable

All measurements are prints of 100,000 product pages.

| Ways to find out what has changed | Bun | Node | Why not him |
| --- | ---: | ---: | --- |
| **Dependency tree print** | **0.05 s** | **0.08 s** | |
| `JSON.stringify` and hash text | 0.17 s | 0.55 s | loses `Map`, dates and `undefined`: different data give the same text |
| Hash variables entirely, bypassing JS | 0.41 s | 0.44 s | reads everything, not just what is shown; remains a backup path for the "whole" |
| Built-in serialization of Bun and Node and hash | 0.64 s | 0.80 s | read everything and more expensive |
| Seroval and hash | 0.74–0.80 s | 1.3–2.3 s | read everything and more expensive |
| Go in WebAssembly within the JS process | 0.32–0.38 s | 0.72–0.95 s | see below |
| Comparison of results as in the first version | 3.2–3.7 s for all updates | 4.3–4.7 s | calculates everything |

**Seroval.** is not needed for comparison: fingerprints are compared, not serialized data. To transfer data to the template too: its part is a normal copy of values, and it is in 15–70 times cheaper than a full copy of variables. A copy through Seroval costs 7 µs to the page as JSON and 48–57 µs as text. The rule that Seroval had to provide remains: the template sees data, not live variables.

**Go in WebAssembly.** Go itself works quickly: 78 MB hash for 25–50 ms, comparison for 1 ms. But the data lies in JS objects, and the WebAssembly module only sees bytes. To give it data, you need to turn it into text, then into bytes and copy: 0.3–0.9 s. Bypassing the objects on the JS itself does not require this step. The module weighs 1.9 MB and runs behind 20 ms; the built-in WASI in Bun 1.4 only runs it by-pass.

**Source Marking, `source(...)`.** Requires that the author does not forget to mark the data. Forgot — the page is silently outdated.

**Record readings during work.** Data is wrapped in `Proxy`, and framework records what was read. Not measured. Reasons for failure: every reading in the project code goes through the interceptor, `Map` and dates for `Proxy` break, and the data root still needs to be called somehow.

### What depends on the size of the site

- The data code of all pages and common blocks is executed at each update. On the stand is 0.05–0.07 s on 200 thousand. An expensive data code, such as a request for each page, will be executed for each page. Many pages still need to be loaded with a single query in `getStaticPaths` or in the module.
- Prints are considered for all pages and blocks: 0.1 s on 200 thousand.
- The data itself is loaded entirely. You can not download them only if the source himself reports what has changed: [proposal](partial-reload.md).

Calculating expressions, writing and rantime depends only on the size of the change.

### What is changing in the framework

| Package | What is changing |
| --- | --- |
| `internal/compile` | parsing expressions and dependency tree; new dependency `tdewolff/parse`; instead of one collected file, two: `load.mjs` and `render.mjs`; check the template code |
| `internal/jsrt` | two parts: loading (data code, fingerprint, own part) and calculation (templates, without `await`) |
| `internal/evaluate` | Go is updating: gets fingerprints, compares, calls what to calculate, accepts records |
| `internal/store` | the recording has an imprint |
| `program`, `render`, `server`, `record` | do not change |

Order of works:

1. The dependency tree in the compiler, with tests for accuracy rules.
2. Splitting into `load.mjs` and `render.mjs`, its part, checks on assembly.
3. Comparison on Go, prints in storage, report "calculated N from M".
4. Stand: the same twenty changes, reconciliation with assembly from scratch, time.
5. Random checks: Random data edit, then the site updated in steps should match the build from scratch.

## Open questions

- Choose the option: second, fourth or fifth.
- Approve the decision of the fifth option on the order of lines: in the order of the request, then by the key.
- For the 1 variant: approve the mechanism and three rules for site authors.
- For the 2 variant: what the query designer should be able to do. How external sources get into the database is in [Data Adapter](data-adapter.md).
- Approve a copy of your part instead of Seroval.
- Comparison of prints on Go: he owns the storage and knows what common blocks someone else needs. You can compare in JS, the price is the same.
- Accuracy through functions: `{finalPrice(item)}` now means "goods as a whole". The compiler can look into the function of the project and find out that it reads `price` and `discount`. Do it right away.
- Functions declared in the data code: later, you can allow those that read only the data.
- How to call a full check command and when to run it.
- What hash function to take for a print: the model is homemade, the code needs a well-known and tested.
