# Blocks and incremental updates

The document describes what the page is built of and how the change in data reaches only those parts of the site that show it.

## Decisions

- After the first complete assembly, the project is updated pointwise: only what is associated with the changed data changes.
- Unit of update - block: page, layout or component call.
- The block change does not spread up and to the sides.
- The same blocks are not repeated, no matter how many pages they show.
- A block is calculated only if the page reaches it: the unselected branch of the condition is not met.
- If the recalculation gave the previous result, the page does not touch.
- Everything is stored in a disassembled form. The page does not contain its components, but calls them: rantime collects it from layouts and components. The speed is not because the runtime is written in Go, but because the entire structure and all the data are at hand. The *Changed by the decision of the author of the project 22 November 2026 year - templates renders Svelte ("okay, take a swelt"):* page contains HTML components that are rendered on-site, and layouts and shared blocks still call.
- The general block for the entire site, such as a header, does not touch the page entries when changing.
- If the fifth item is removed from the list in 100,000, the remaining 99,995 is not considered changed.
- The layout is stored separately from the pages: the layout change does not touch the page entries, the page change is a layout. The layout that reads the page address is stored with its own entry on each page (the decision of the author of the project 22 November 2026, according to his opinion).

## Implemented, pending approval

### Page records

- **Block** is part of a page in the entry: HTML, which rendered one component along with everything it showed on site; a snippet that the display location gave to the shared block; or a link to the shared block.
- **HTML with holes** is what lies in the block: HTML, as it was rendered by Svelte, without hydration comments. In holes, it is what is substituted when the page is shown: a common block, a snippet that gave the place of display, an island.
- **Page Record** is a list of its blocks. The HTML block has HTML itself written down, what it gives `<head>` pages from `<svelte:head>`, and the last `<title>` among this. The link to the common block has - what snippets this place gave it.
- **Shared block** is a component that shows the same thing on any page. It has its own record under the name; pages link to this name. Layout is always a separate record: common to the entire site or its own to each page.
- **File not in HTML** is what gives a module among pages, for example `src/pages/sitemap.xml.ts`. Its record is one block with text that returned the function `GET`. What `GET` read from the database is recorded on this record like a page.

Product page in the store from [examples/catalog](../examples/catalog/README.md):

```
page entry /products/jug
└─ layout src/pages/+layout.svelte{|children} common: one for the whole site
   ├─ Header general: one for the whole site
   ├─ {@render children()}: HTML pages in page entries
   │  ├─ <title> and what is given in <head> in the page entry
   │  ├─ "Kuvshin", "$2,400", options  3 HTML, rendered on site
   │  ├─ Badge{"text":"from the oven"} hole: common, one on all such icons
   │  └─ ProductCard{"id":"p7"} hole: common, one to all pages with this product
   └─ Footer general: one for the whole site
```

The layout doesn’t read anything about the page, so it’s one-on-one for the entire site. The page gives him a snippet of `children` herself. Snippet names are in the common block name after `|`. The snippet itself is rendered in the page and lies in its entry; in the layout entry, a hole "snippet `children`" is in its place, which gave the location of the display.

The documents around — `<!doctype>`, `<html>`, `<head>`, `<body>` — are not in the records. This is the shell of `src/site.html`, one for the entire site; it lies in the project framework, cut into its places `%shugo.head%` and `%shugo.body%`.

### Shared components

A component is rendered as a shared block when it shows the same thing everywhere it stands with the same parameters:

- it does not depend on the page: does not import `page` and `paginate` from the `shugo` module - neither itself nor through what it imports;
- it does not depend on the place where it stands: does not take the context of Svelte (`getContext`, `setContext`, `hasContext`, `getAllContexts`, `createContext`), does not call `$props.id()` and does not show a snippet inside `<select>`, neither does it show what
- it is given simple values strings, finite numbers, `true` and `false`, `null` and snippets without parameters written inside its tag; `bind:` is not a simple value: through it, the component writes to the one who showed it.

Then it is calculated once for each set of parameters, and pages refer to it by name: `src/components/Card.svelte{"id":"p7"|children}`. Otherwise, it will render on the spot, in the record of who showed it, exactly like in Svelte. Context, `$props.id()` and `<select>` depend on the location: context gives the components around, the number in `$props.id()` counts the components rendered earlier, and the value of `<select>` chooses which `<option>` from the snippet is marked. The common block, rendered separately, would not see either one or the other.

A snippet with parameters even if everyone has default values, `{#snippet row(n = 1)}` the component shows with its arguments, so such a component is rendered in place. The arguments that the common component transmits to the snippet without parameters do not change anything, as in Svelte.

### How HTML is split into blocks

Svelte renders a page with one line. Where the common block is located and what the place of display gave it, mark the tags that no page can write: byte 0, a random word selected to run, which means the label and again byte 0.

1. Each template in the build is shown through the Shugo function of `component(file number, component Svelte)`. For the overall component, it does not render it. Each snippet without the parameters it was given a place, it renders separatelywith its Svelte render, with the context of the placeand remembers it HTML, what it gave `<head>`, and its `<title>`. Then put the label "here is the common block N".
2. If the snippet is not rendered separately falling or calling `$props.id()` the component is rendered on-site, along with its snippets, as in Svelte. Falling is where the common component of the snippet would not show: `{#if p.discount}…{p.discount.percent}…{/if}` inside the component, and the snippet is written outside. The number `$props.id()` in the snippet depends on how many times and where it will be shown.
3. The common block is calculated separately, by its name: parameters are taken from the name, and instead of each snippet that was given to it, a plug that puts a label "snippet `children`, which gave the place of display".
4. When Svelte is finished, the runtime reads a string by tags: the common block label becomes a hole and the record block becomes a reference to the common block, and the remembered snippets become the blocks it refers to. The block appears only for a mark that has reached the end: if Svelte renders part of the page once again, as it does for `bind:`, the first attempt leaves no extra blocks.
5. From text outside `{@html}` Comments are cut out which Svelte Writes for hydration in the browser: `<!---->`, `<!--[-->`, `<!--[!-->`, `<!--[1-->`, `<!--[-1-->`, `<!--]-->`, `<!--$s1-->` U -  `$props.id()`hash component around it `<svelte:head>` and `<!>` At the end of “The Rich” `<select>`. Static page they are not needed: it is a quarter byte of the page. `{@html}` in templates Shugo The same function Sveltebut his HTML stands between the labels, and comments remain in it. The islands render Svelte The project, and their hydration marks remain: on them the island comes to life in the browser.
6. What the components gave `<head>`, Svelte writes separately, and the last `<title>` writes after it. Shugo puts a label at the end of what is given in `<head>`, and so separates `<title>` - in each render: pages, common block, snippet.

### Layouts

The layout never lies in the page entry. The page stores a link to it and its snippets, and the layout is its own entry:

- **A model that does not read a page**, — one entry for the entire site: `src/pages/+layout.svelte{|children}`.
- **A model that reads the address** `page.url` or `page.params` from the `shugo` module, for example, to highlight the current menu item own entry to each page. Its name is the route and address of the page: `src/pages/+layout.svelte{"@path":"/c1","@route":0|children}`. The name of the entry is calculated by itself: from the address you get `page` - `page.url` and `page.params`, like a page.

Whether the template reads a page, the compiler learns by imports. The template or module that imports `page` or `paginate` from the `shugo` module depends on the page, and with it everyone who imports it: for example, a layout that imports the module from `src/lib/`, where the function reads `page.url`. Such a layout is a record for each page, such a component is not a common block: it is rendered on each page anew, in the record of the one who showed it. `query` does not depend on the page: what it read is written on who is currently being calculated.

The layout is rendered separately from the page, so the context it sets through `setContext` sees its components but not the page inside it. `$props.id()` in the layout gives names with the file number of the layout, `b3-s1`, so that they do not match the page names.

Readings of the layout are recorded on his record, readings of the page are recorded on it. Therefore:

| What has changed | What is calculated |
| --- | --- |
| what reads the layout without an address, for example, a menu | one recording layout |
| what to read with address | layout entry for each page; the pages themselves are not |
| what the page reads | page entry; layout - no |

A page that is no longer there takes its own layout record with it: no one shows it anymore. Checks this `TestLayoutKeptForEveryPage`.

### Page head

What the components write in `<svelte:head>` is in the HTML block they render: separately what is given in `<head>`, and separately the last `<title>`. The layout, which is calculated once on the entire site, keeps its own page, its own general block, the snippet given to the common block - at itself.

The head of the page is collected by the server. While it shows a page, it remembers what each block gave in `<head>`, in order of display: the outer layout is before the inner one, the layout is before the page, the common block is before the snippet it shows like the order of a document in Svelte, which writes a `<svelte:head>` component before it is marked. Snippet, which the common block did not show, does not give anything to the head, and shown twice - gives twice, too, as in Svelte. When the page is shown, he writes the remembered one, after him, the last `<title>`, then a link to styles, and puts all this in place of the `%shugo.head%` shell of `src/site.html`. Where this place is, the server knows from the frame: the shell lies there cut into three pieces - up to `%shugo.head%`, between places and after `%shugo.body%`.

HTML lies within records, shared blocks are separate records under their own name. The reasons are in [2](../spec/decisions.md#2-values-are-stored-by-content-and-shared-blocks-by-name) and [3](../spec/decisions.md#3-which-blocks-are-shared).

### Snippets

The component receives markup from the caller in the same way as in Svelte: snippets. They are stored in two ways.

- **Snippet without parameters, written inside the tag of the common component**, including `children`, anything in the tag except `{#snippet}`. It is rendered separately, where it is written, when the component call is rendered, whether its component shows or not: so the place record is ready, even if the common block starts to show it later, after changing its data. Its block - HTML, head and `<title>` - lies in the record of the caller, and the link to the common block is written: a snippet with this name - this block. Therefore, a component with snippets without parameters remains a common block: from snippets in its record only names. Snippet, which can not be rendered, leaves the component in place, see. [How HTML is cut into blocks](#how-html-is-split-into-blocks).
- **Other snippets** with parameters and those that are declared in the template itself, and not in the component tag Svelte renders where `{@render}` shows them, as always. The component to which such a snippet was transmitted is not a common block: the function cannot be named in the name.

The component that renders on-site gets all of its snippets as in Svelte: from the snippets, Shugo stores nothing separately.

### Updates

1. Go reads the database log and finds those who have read the changed columns of the changed rows: pages, common blocks, lists of route pages.
2. JS only calculates them.
3. Go for each record counts the hash and compares it to the saved one. Match - the record is not touched.
4. Different entries are checked and recorded. Common blocks that no one else shows, and pages that the route no longer gives, are deleted.

All this is one transaction. If something falls, nothing is recorded: the site remains the same, and the next update will meet the same changes. Details - in [Analysis of data changes](change-tracking.md#final-implementation) and [Assembly Sequences](build-pipeline.md#evaluation).

### Effects of data changes

Shop on 100,000 products: 104,202 pages, 100,009 common blocks, 0.9 million blocks. Update after price change of one product — 31 ms, after changing the cap — 0.12–0.17 s. Full table and launch — in [README booth](../examples/catalog/README.md#results).

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
| List Item Key | HTML element is already independent of neighbors; the key `{#each items as item (item.id)}` Svelte allows and on the server does not use |
| “The Significance of the Entry Hash” | hash is taken from the record, not from the input |
| Banning a component from calling itself | the limitation of the prototype |

### Update report

After each update, we know how many data lines have changed; how many pages and common blocks have been calculated, added, modified and deleted, with the names of the first ones; how many pages have HTML changed; how long each step took. An example is in [Command Line](../spec/cli.md).

## Open questions

- Common block with object in parameters: now only calls with simple parameters are considered common. If you remember a block by the content of an object, the page that transmits it would depend on all column lines, not read ones.
