# Blocks and incremental updates

Shugo stores each page as a set of blocks. A data change reevaluates the blocks whose output depends on that data.

## Decisions

- After the first full build, updates reevaluate the affected parts of the project.
- The update unit is a page, layout, or shared component call. On November 22, 2026, the project adopted Svelte rendering. A component rendered inline contributes HTML to the caller's block.
- Updating one block does not force its parents or siblings to be reevaluated.
- Pages can reference the same shared block or layout. Each page record contains its own inline HTML.
- Shugo evaluates the branches a page actually reaches. It prepares caller-provided snippets separately for a shared block even if the block does not display them yet. A subsequent update to the block can then display an already prepared snippet. See [decision 43](../spec/decisions.md#43-render-shared-block-snippets-separately).
- If a reevaluation produces the same record, Shugo keeps the existing one.
- The runtime assembles inline HTML and references to layouts and shared blocks using records whose data is already available.
- Updating a site-wide block such as the header leaves page records intact.
- Removing the fifth item from a list of 100,000 does not mark the other records as changed.
- Layouts have their own records. A layout that reads the page URL is stored once per page. The project author confirmed this on November 22, 2026.

## Implemented, pending approval

### Page records

- A **block** is a record entry containing the HTML a component rendered inline, a snippet a caller supplied to a shared block, or a reference to a shared block.
- An **HTML fragment** contains HTML from Svelte with its hydration markers removed. Placeholders can refer to shared blocks, caller-provided snippets, or islands.
- A **page record** lists its blocks. An HTML block stores its markup, its `<svelte:head>` contribution, and the last `<title>` in that contribution. A shared-block reference lists the snippets supplied by the caller.
- A **shared block** is a component whose output is the same across pages for the same props. Shugo stores it under its own name and pages reference that name. A layout always has a separate record, either site-wide or per page.
- A **non-HTML endpoint** such as `src/pages/sitemap.xml.ts` has a one-block record with the text returned by its `GET` function. Shugo tracks the data reads of that function against the record.

A product page in [the catalog example](../examples/catalog/README.md) has this structure.

```text
Page record /products/jug
└─ Layout src/pages/+layout.svelte{|children}   shared across the site
   ├─ Header                                    shared across the site
   ├─ {@render children()}                       HTML in the page record
   │  ├─ <title> and other head content         in the page record
   │  ├─ Pitcher, $2,400, three variants         inline HTML
   │  ├─ Badge{"text":"Kiln fired"}              reference shared by matching badges
   │  └─ ProductCard{"id":"p7"}                  reference shared by pages with this product
   └─ Footer                                    shared across the site
```

The layout does not read page data, so all pages use one layout record. The page supplies itself as the layout's `children` snippet. Snippet names follow `|` in the shared block name. The page record holds the rendered snippet, and the layout record has a placeholder for the caller's `children`.

The surrounding `<!doctype>`, `<html>`, `<head>`, and `<body>` come from the site shell, `src/site.html`. The project skeleton stores the shell split at `%shugo.head%` and `%shugo.body%`.

### Shared components

A component can render as a shared block when the same props produce the same output everywhere it appears.

- Neither it nor its imports read `page` or `paginate` from the `shugo` module.
- Neither it nor the code it renders or imports depends on its placement. It does not use Svelte context functions, call `$props.id()`, or render a snippet inside `<select>`.
- Its props are simple values such as strings, finite numbers, booleans, or `null`. Its snippets take no parameters and are declared inside the component tag. A `bind:` writes back to the caller and is not a simple prop.

Shugo evaluates it once per set of props and references it by a name such as `src/components/Card.svelte{"id":"p7"|children}`. Other components render inline with their caller. Svelte context comes from ancestors, `$props.id()` depends on render order, and a `<select>` can choose which option a snippet marks as selected. A separately evaluated block would lack those inputs.

A parameterized snippet keeps the entire component inline. JavaScript's function length excludes parameters with defaults, so Shugo inspects the snippet source. A parameterless snippet ignores arguments passed by the shared component.

### How HTML is split into blocks

Svelte renders the page to a string. Shugo puts markers at shared-block references, supplied snippets, islands, `{@html}` content, and the end of each head contribution. Markers consist of a NUL byte, a random token per run, the marker data, and a final NUL byte.

1. Shugo wraps each template with `component(fileNumber, svelteComponent)`. Each call checks the compile-time sharing flag and the supplied props. For a shared call, Shugo renders and records each parameterless snippet separately with the caller's `getAllContexts()`. The resulting HTML, head content, and title belong to the caller. The wrapper emits a shared-block marker.
2. If a snippet errors or calls `$props.id()`, Shugo renders the component inline with its snippets. For example, a snippet might read `p.discount.percent` and depend on a surrounding `{#if p.discount}`. Shugo detects the ID call from Svelte's `<!--$…-->` marker.
3. Shugo evaluates a shared block from its name. It deserializes the props and gives it stubs for each supplied snippet that emit placeholders such as a reference to the caller's `children`.
4. After the render, JavaScript splits the final string at the markers. A shared-block marker becomes a placeholder and a reference entry. The entry points to the previously rendered snippets. Only markers in the final output create records. A Svelte rerender for `bind:` does not leave entries from the first pass.
5. Shugo strips Svelte hydration comments outside `{@html}`. Those include `<!---->`, `<!--[-->`, `<!--[!-->`, `<!--[1-->`, `<!--[-1-->`, `<!--]-->`, the `<!--$s1-->` from `$props.id()`, head hashes, and the trailing `<!>` for a rich `<select>`. The comments accounted for about a quarter of the page bytes. Shugo wraps `{@html}` with its own markers to preserve internal comments. Svelte renders islands separately, and the browser needs the islands' original markers.
6. Svelte writes head content separately and appends its last `<title>`. Shugo marks the end of the head content so it can split out the title in page, shared-block, and snippet renders.

### Layouts

The page stores a reference to its layout and holds the snippets it supplies. The layout has its own record.

- A layout independent of the page has one site-wide record, such as `src/pages/+layout.svelte{|children}`.
- A layout that reads `page.url` or `page.params` from `shugo` gets a record for each page. Its key includes the route and URL, for example `src/pages/+layout.svelte{"@path":"/c1","@route":0|children}`. Shugo uses that key to reconstruct the layout's `page` value.

Shugo detects page dependencies from imports of `page` and `paginate`, including transitive imports. Such a layout is evaluated for each page and such a component is rendered inline. `query` can be used in either case. Its reads belong to the record currently being evaluated.

A layout renders separately from its page and nested layouts. Its own components can see its `setContext` values. A layout's `$props.id()` includes the template's file number, as in `b3-s1`, so it does not collide with page IDs.

| Data change | Reevaluation |
| --- | --- |
| A site-wide layout dependency, such as the menu | One layout record |
| A page-specific layout dependency | Each affected layout record |
| A page dependency | The affected page record |

Removing a page also removes its page-specific layout record. `TestLayoutKeptForEveryPage` covers this behavior.

### Page head

An HTML block stores its head contribution and last title separately. The page, layouts, shared blocks, and snippets each own their contributions.

As it assembles the page, the server collects head fragments in render order. Outer layouts precede inner layouts, layouts precede the page, and a shared block precedes the snippets it renders. A snippet contributes once for each time it is shown. The server appends the last title and the stylesheet link, and it substitutes the result for `%shugo.head%` in the shell. The skeleton stores the segments before, between, and after the two shell markers.

See [decision 2](../spec/decisions.md#2-values-are-stored-by-content-and-shared-blocks-by-name) and [decision 3](../spec/decisions.md#3-which-blocks-are-shared) for the record design.

### Snippets

A component can accept markup from its caller in two ways.

- A parameterless snippet declared inside a shared component's tag, including its implicit `children`, renders separately in the caller's scope. Shugo renders it eagerly so a later shared-block update can display it. The caller's record contains its HTML, head content, and title. The shared-block reference matches the snippet name to that entry. If the snippet can't render separately, the component renders inline. See [how HTML is split into blocks](#how-html-is-split-into-blocks).
- Svelte renders parameterized snippets and snippets declared elsewhere in the template at `{@render}`. A component that receives one of these snippets renders inline because its function can't be serialized as part of a shared-block name.

Svelte gives inline components their original snippets.

### Updates

1. Go reads the database change log and finds pages, shared blocks, and route lists that read a changed column in a changed row.
2. JavaScript reevaluates those readers.
3. Go hashes each result and compares it with the stored record. A matching hash keeps the existing record.
4. Go validates and writes changed records. It deletes pages that a route no longer returns and shared blocks with no remaining references.

One transaction covers all writes. A failure leaves the store unchanged so the next update sees the same inputs. See [change tracking](change-tracking.md#final-implementation) and [evaluation](build-pipeline.md#evaluation).

### Effects of data changes

The benchmark had 100,000 products, 104,202 pages, 100,009 shared blocks, and about 900,000 blocks. Changing one product price took 31 ms. Updating the header took 0.12–0.17 s. See the [catalog results](../examples/catalog/README.md#results).

| Data change | Page records written | Shared-block records written | Pages with changed HTML |
| --- | ---: | ---: | ---: |
| None | 0 | 0 | 0 |
| Unread field on 5,000 products | 0 | 0 | 0 |
| One product price | 1 | 1, the product card | 7 |
| Twenty product prices | 20 | 20 | 70 |
| Featured product name | 1 | 2, the header and card | 104,202 |
| Switch the featured product | 0 | 1, the header | 104,202 |
| Footer phone number | 0 | 1, the footer | 104,202 |
| Move a product to a different category | 168 across two category listings | 1, the header with counts | 104,202 |
| Approved review text | 1 | 0 | 1 |
| Hidden review text | 0 | 0 | 0 |
| Remove a product's first variant | 1 | 0 | 1 |
| Swap two recommended products | 1 | 0 | 1 |
| Remove a product | 89, including 1 deletion | 1 deletion | 88 |
| Add a product | 86, including 1 new record | 2, including 1 new record | 104,202 |

The product price appears on its page, on the home and category pages, and in recommendations on four other pages. One page record and one card record change, while seven pages gain new HTML because they reference the same card.

The header appears on every page. A header update changes the HTML of 104,202 pages by writing one record.

The benchmark finishes by building and exporting the same data from scratch. Its output matched the incremental output byte for byte.

### Header updates in the prototype

The prototype stored header data in each page and kept each block's content hash in every referencing page record. A header change therefore required rereading data and rewriting each page. The current header reads its own data, and pages reference its name. The update writes only the header record.

### Changes since the prototype

| Prototype | Current implementation |
| --- | --- |
| Dependency tree of fields read by each block | Database rows track column reads at runtime across branches, functions, and components. See [change tracking](change-tracking.md). |
| Read precision such as `a.b.c` for one field and method calls for an entire object | The same read rules apply. |
| List item keys | Svelte accepts `{#each items as item (item.id)}`. Server rendering does not need keys because an item's HTML is independent of its neighbors. |
| Values hashed from inputs | Shugo hashes the resulting record. |
| No recursive component calls | That limit belonged to the prototype. |

### Update report

Each update reports changed data rows, evaluated pages and shared blocks, and the counts and initial names of added, changed, and removed records. It also reports how many pages have different HTML and the time per stage. See the [CLI report examples](../spec/cli.md#reports).

## Open questions

- Sharing a component with object props would require a key derived from object contents. When that object is a database row, the caller would then depend on every column.
