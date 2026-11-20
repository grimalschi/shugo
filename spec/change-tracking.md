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

- Approve the mechanism and three rules for authors of the site.
- Approve a copy of your part instead of Seroval.
- Comparison of prints on Go: he owns the storage and knows what common blocks someone else needs. You can compare in JS, the price is the same.
- Accuracy through functions: `{finalPrice(item)}` now means "goods as a whole". The compiler can look into the function of the project and find out that it reads `price` and `discount`. Do it right away.
- Functions declared in the data code: later, you can allow those that read only the data.
- How to call a full check command and when to run it.
- What hash function to take for a print: the model is homemade, the code needs a well-known and tested.
