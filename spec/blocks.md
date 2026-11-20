# Blocks and incremental updates

The document describes what the page is built of and how the change in data reaches only those parts of the site that show it.

## Decisions

- After the first complete assembly, the project is updated pointwise: only what is associated with the changed data changes.
- Unit of update - block: page, layout or component call.
- The block change does not spread up and to the sides.
- The same blocks are not repeated, no matter how many pages they show.
- A block is calculated only if the page reaches it: the unselected branch of the condition is not met.
- If the recalculation gave the previous result, the page does not touch.
- Everything is stored in disassembled form. The page does not contain its components, but calls them: the rantime collects it from layouts and components. The speed is obtained not because the rantime is written in Go, but because the whole structure and all the data are at hand.
- The general block for the entire site, such as a header, does not touch the page entries when changing.
- If the fifth item is removed from the list in 100,000, the remaining 99,995 is not considered changed.

## Made in the first version, waiting for approval

### Page records

- **Block** is one display of a single template: a page, layout, component or markup transferred to the slot.
- **Tape** is the value of block expressions in the order of the template. There is no markup in the tape: it lies in the project program, one for all template impressions.
- **Page Record** is a list of its blocks. The block has its template, a link to the tape, the blocks inside it and what is transferred to its slots.
- **Shared block** is a component that shows the same thing on any page. It has its own record under the name; pages link to that name.

```
page /products/jug
├─ layout pages/+layout.shugo general: one for all pages
│  ├─ Header general: one for the whole site
│  └─ slot: Page
│     ├─ feed page "Kuvshin", "$2,400", ...
│     ├─ Variant  3 own tapes
│     └─ ProductCard{id: p7} common: one to all pages with this product
└─ slot head: <title>
```

Tapes are stored according to their content: the same tapes of different blocks are one line in the storage. The reasons are in [2](decisions.md#2-values-are-stored-by-content-and-shared-blocks-by-name) and [3](decisions.md#3-which-blocks-are-shared).

### Updates

1. JS re-counts all pages and shared blocks.
2. Go for each record counts the hash and compares it to the saved one. Match - the record is not touched.
3. Different entries are checked and recorded. New tapes are added, tapes that no one refers to are deleted.
4. Pages that no longer exist are deleted.

Everything is calculated, only the difference ([1](decisions.md#1-data-loading-and-expression-are-counted-together-go-compares-the-result)) is recorded.

### Effects of data changes

Shop on 100,000 products: 104,202 pages, 100,009 common blocks, 1.2 million blocks, 304,300 different tapes. Each update took 3.3–3.7 s. Full table and run in [README booth](../examples/catalog/README.md).

| What changed in the data | Pages have changed | The blocks have changed |
| --- | ---: | ---: |
| Nothing. | 0 | 0 |
| Field that is not shown anywhere, 5,000 has products | 0 | 0 |
| Price of one product | 1 | 1: his card |
| Prices 20 products | 20 | 20 |
| Product Name of the Month | 1 | 2: cap and card |
| The goods of the month became another product | 0 | 1: cap |
| Phone in the footer | 0 | 1: footer |
| The product was moved to another category | 168: lists of two categories | 1: cap with counters |
| Text of the approved review | 1 | 0 |
| Text of the hidden review | 0 | 0 |
| The first product was removed | 1 | 0 |
| Two recommended products changed places | 1 | 0 |
| Product removed | 89, of which 1 has been removed | 1 |
| Product added | 85, of which 1 added | 2, of which 1 added |

The price of the product is shown on its page, in the category lists and in the recommendations on other pages. One page and one card are changed: the rest of the pages link to a card by name.

At the end, the stand collects the same data from scratch and checks the result with the site updated in steps: all 104,202 pages coincided.

### Why the prototype header change touched all pages

There were two reasons.

1. These caps were in the data of each page. The cap changed — the data of all pages changed, and each had to be re-read.
2. The page record kept the hash of each block, including the cap. The cap has a new hash - the record of each page must be rewritten.

Now the header loads its data itself and is calculated once. The page stores its name, not content. The header has changed - one record has been rewritten.

### What is no longer a prototype

| It was in the prototype. | Why removed |
| --- | --- |
| Dependency tree: what data fields each block reads | compare results, not input |
| Accuracy rules: `a.b.c` - one field, method call - the whole object | same; filter by common list in template is no longer a problem |
| List Item Key | the elements are not dependent on neighbors. |
| “The Significance of the Entry Hash” | the tape is stored under the hash of itself |
| Banning a component from calling itself | the limitation of the prototype |

### Update report

After each update, we know how many pages and common blocks are added, changed and deleted, with the names of the first ones; how many tapes are added and removed; how long each step took. An example is in [Command Line](cli.md).

## Open questions

- Approve comparison of results instead of comparing input data.
- The key in `<g-for>`: is it needed for something else.
- Update on signals, without recalculating the entire site: [proposal](partial-reload.md).
- What pages show the changed common block. Now it is not written: the server does not need, and export to files collects all the pages and writes different ones.
- Memory for a common block with an object in parameters: now only calls with simple parameters are considered common.
