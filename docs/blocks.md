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
- The layout is stored separately from the pages: the layout change does not touch the page entries, the page change is a layout. The layout that reads the page address is stored with its own entry on each page (the decision of the author of the project 22 November 2026, according to his opinion).

## Implemented, pending approval

### Page records

- **Block** is one display of one template: a page, layout, component or snippet.
- **Tape** — block expression values in the order of the template. There is no markup in the tape: it lies in the project framework, one for all template impressions.
- **Page Record** is a list of its blocks. The block has its template, its tape, the blocks inside it, the snippets that the caller gave it, and the blocks that its template gave `<head>` pages in `<svelte:head>`.
- **Shared block** is a component that shows the same thing on any page. It has its own record under the name; pages link to this name. Layout is always a separate record: common to the entire site or its own to each page.
- **File not in HTML** is what gives a module among pages, for example `src/pages/sitemap.xml.ts`. Its record is one block with one value: the text that returned the function `GET`. What `GET` read from the database is written to this record like a page.

Product page in the store from [examples/catalog](../examples/catalog/README.md):

```
page entry /products/jug
└─ layout src/pages/+layout.svelte{|children} common: one for the whole site
   ├─ Header general: one for the whole site
   ├─ {@render children()}: page block in page entry
   │  ├─ <svelte:head>: block <title> in page entry
   │  ├─ feed page "Kuvshin", "$2,400", ...
   │  ├─ Badge{text: "from the oven"} common: one for all such icons
   │  ├─ Variant  3 own tapes, in page record
   │  └─ ProductCard{id: p7} General: one to all pages with this product
   └─ Footer general: one for the whole site
```

The layout doesn’t read anything about the page, so it’s one-on-one for the entire site. The page gives him a snippet of `children` herself. Snippet names are in the common block name after `|`. The snippet itself is calculated in the page and lies in its entry; in the layout entry, in its place, there is a link "snippet `children` of the one who showed me".

The documents around — `<!doctype>`, `<html>`, `<head>`, `<body>` — are not in the records. This is the shell of `src/site.html`, one for the entire site; it lies in the project framework, cut into its places `%shugo.head%` and `%shugo.body%`.

### Layouts

The layout never lies in the page entry. The page stores a link to it and its snippets, and the layout is its own entry:

- **A model that does not read a page**, — one entry for the entire site: `src/pages/+layout.svelte{|children}`.
- **A model that reads the address** `page.url` or `page.params` from the `shugo` module, for example, to highlight the current menu item own entry to each page. Its name is the route and address of the page: `src/pages/+layout.svelte{"@path":"/c1","@route":0|children}`. The name of the entry is calculated by itself: from the address you get `page` - `page.url` and `page.params`, like a page.

Whether the template reads a page, the compiler learns by imports. The template or module that imports `page` or `paginate` from the `shugo` module depends on the page, and with it everyone who imports it: for example, a layout that imports the module from `src/lib/`, where the function reads `page.url`. Such a layout is a record for each page, such a component is not a common block: it is calculated on each page anew, in the record of the one who showed it. `query` does not depend on the page: what it read is written on who is currently being calculated.

Readings of the layout are recorded on his record, readings of the page are recorded on it. Therefore:

| What has changed | What is calculated |
| --- | --- |
| what reads the layout without an address, for example, a menu | one recording layout |
| what to read with address | layout entry for each page; the pages themselves are not |
| what the page reads | page entry; layout - no |

A page that is no longer there takes its own layout record with it: no one shows it anymore. Checks this `TestLayoutKeptForEveryPage`.

### Page head

What the template writes in `<svelte:head>` is its own writing blocks: the template block has a head block list and a separate list of `<title>` blocks. The layout, which is calculated once on the entire site, stores its head blocks at home, the page - at home.

The head of the page is collected by the server. As it shows a page, it remembers the head blocks of each template it met, in order: the outer layout is before the inner one, the layout before the page is the document order, as in Svelte. When the page is shown, it shows the blocks that are remembered, and of `<title>`, only the last one, after the rest, then a link to the styles, and puts all this in place of `%shugo.head%`'s shell `src/site.html`. Where this place is, the server knows from the frame: the shell lies there cut into three pieces - up to `%shugo.head%`, between places and after `%shugo.body%`.

Tapes lie inside records, common blocks are separate records under their own name. The reasons are in [2](../spec/decisions.md#2-values-are-stored-by-content-and-shared-blocks-by-name) and [3](../spec/decisions.md#3-which-blocks-are-shared).

### Snippets

The component receives markup from the caller in the same way as in Svelte: snippets. They are stored in two ways.

- **Snippet without parameters, written inside the component tag**, including `children`, anything in the tag except `{#snippet}`. It is calculated where it is written, when the component call is calculated, whether its component shows or not. Its block is in the record of the caller, and the block component is written: a snippet with that name - these blocks. Therefore, a component with snippets without parameters remains a common block: from snippets in its record only names.
- **Other snippets**with parameters and those declared in the template itself, rather than in the component tagare calculated where `{@render}` shows them: the values of snippet expressions with parameters depend on the arguments. Its block is the nested block of the one who showed it. The component to which such a snippet was passed is not a common block: the function cannot be named in the name.

In place of `{@render}` in the tape stands, what is the case and how many components between it and the one who wrote the snippet. Details - in [Storage](storage.md#tape).

### Updates

1. Go reads the database log and finds those who have read the changed columns of the changed rows: pages, common blocks, lists of route pages.
2. JS only calculates them.
3. Go for each record counts the hash and compares it to the saved one. Match - the record is not touched.
4. Different entries are checked and recorded. Common blocks that no one else shows, and pages that the route no longer gives, are deleted.

All this is one transaction. If something falls, nothing is recorded: the site remains the same, and the next update will meet the same changes. Details - in [Analysis of data changes](change-tracking.md#final-implementation) and [Assembly Sequences](build-pipeline.md#evaluation).

### Effects of data changes

Shop on 100,000 products: 104,202 pages, 100,009 common blocks, 1.3 million blocks. Update after price change of one product — 29 ms, after changing the cap — 0.12–0.13 s. Full table and launch — in [README booth](../examples/catalog/README.md#results).

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
| List Item Key | the tape elements are not dependent on neighbors; key `{#each items as item (item.id)}` Svelte allows, Shugo takes it and does not use for anything |
| “The Significance of the Entry Hash” | hash is taken from the record, not from the input |
| Banning a component from calling itself | the limitation of the prototype |

### Update report

After each update, we know how many data lines have changed; how many pages and common blocks have been calculated, added, modified and deleted, with the names of the first ones; how many pages have HTML changed; how long each step took. An example is in [Command Line](../spec/cli.md).

## Open questions

- Common block with object in parameters: now only calls with simple parameters are considered common. If you remember a block by the content of an object, the page that transmits it would depend on all column lines, not read ones.
