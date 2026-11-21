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

## Implemented, pending approval

### Page records

- **Block** is one display of a single template: a page, layout, component or markup transferred to the slot.
- **Tape** is the value of block expressions in the order of the template. There is no markup in the tape: it lies in the project program, one for all template impressions.
- **Page Record** is a list of its blocks. The block has its template, its tape, the blocks inside it and what is transferred to its slots.
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

Tapes lie inside records, common blocks are separate records under their own name. The reasons are in [2](decisions.md#2-values-are-stored-by-content-and-shared-blocks-by-name) and [3](decisions.md#3-which-blocks-are-shared).

### Updates

1. Go reads the database log and finds those who have read the changed columns of the changed rows: pages, common blocks, lists of route pages.
2. JS only calculates them.
3. Go for each record counts the hash and compares it to the saved one. Match - the record is not touched.
4. Different entries are checked and recorded. Common blocks that no one else shows, and pages that the route no longer gives, are deleted.

Details are in [Analysis of data changes](change-tracking.md#final-version-waiting-for-approval) and [Assembly Sequences](build-pipeline.md#evaluation).

### Effects of data changes

Shop on 100,000 products: 104,202 pages, 100,009 common blocks, 1.2 million blocks. Update after price change of one product — 28 ms, after changing the cap — 0.12 s. Full table and launch — in [README booth](../examples/catalog/README.md#results).

| What changed in the data | Page entries rewritten | Common blocks rewritten | Pages with new HTML |
| --- | ---: | ---: | ---: |
| Nothing. | 0 | 0 | 0 |
| Field that is not shown anywhere, 5,000 has products | 0 | 0 | 0 |
| Price of one product | 1 | 1: his card | 7 |
| Prices 20 products | 20 | 20 | 70 |
| Product Name of the Month | 1 | 2: cap and card | 104 202 |
| The goods of the month became another product | 0 | 1: cap | 104 202 |
| Phone in the footer | 0 | 1: footer | 104 202 |
| The product was moved to another category | 168: lists of two categories | 1: cap with counters | 104 202 |
| Text of the approved review | 1 | 0 | 1 |
| Text of the hidden review | 0 | 0 | 0 |
| The first product was removed | 1 | 0 | 1 |
| Two recommended products changed places | 1 | 0 | 1 |
| Product removed | 89, of which 1 has been removed | 1 | 88 |
| Product added | 86, of which 1 is new | 2, of which 1 is new | 104 202 |

The price of the product is shown on its page, on the main page, in the category list and in the recommendations on four other pages. HTML changes on seven pages, and one page record and one card are rewritten: the remaining pages link to a card by name.

The header is on every page. When it changes, HTML changes on all 104,202 pages, and one entry is rewritten in the repository.

At the end, the stand collects and exports the same data from scratch and checks the result with the site updated and exported in steps: all files are matched.

### Why the prototype header change touched all pages

There were two reasons.

1. These caps were in the data of each page. The cap changed — the data of all pages changed, and each had to be re-read.
2. The page record kept the hash of each block, including the cap. The cap has a new hash - the record of each page must be rewritten.

Now the header loads its data itself and is calculated once. The page stores its name, not content. The header has changed - one record has been rewritten.

### What is no longer a prototype

| It was in the prototype. | What is replaced |
| --- | --- |
| Dependency tree: what data fields each block reads | lines from the database themselves remember which columns they have read, more precisely the tree: with precision to the branch conditions and through functions and components ([Change tracking](change-tracking.md)) |
| Accuracy rules: `a.b.c` - one field, method call - the whole object | the same |
| List Item Key | the elements are not dependent on neighbors. |
| “The Significance of the Entry Hash” | hash is taken from the record, not from the input |
| Banning a component from calling itself | the limitation of the prototype |

### Update report

After each update, we know how many data lines have changed; how many pages and common blocks have been calculated, added, modified and deleted, with the names of the first ones; how many pages have HTML changed; how long each step took. An example is in [Command Line](cli.md).

## Open questions

- Approve [final Version](change-tracking.md#final-version-waiting-for-approval).
- The key in `<g-for>`: is it needed for something else.
- Memory for a common block with an object in parameters: now only calls with simple parameters are considered common.
