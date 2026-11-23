# Writing a Shugo site

Shugo builds static sites with interactive islands. A page, a layout and a component are each a `.svelte` file, and a `.svelte` file is a **Svelte 5 component**: `<script>`, markup, `<style>`. Svelte 5.59.1, which comes inside Shugo, compiles it and renders every page on the server, when the page is computed, so what you know of Svelte holds: `$props()`, `{#if}`, `{#each}`, snippets, `{@render}`, `{@html}`, context, scoped styles; whitespace, errors and warnings are Svelte's own. Shugo takes out the comments Svelte writes for hydration: a static page has no use for them. What is Shugo's own: data comes from the site's SQLite database through `query`, imported from the module `shugo`, and a page with a parameter in its name lists its pages with `getStaticPaths`. A component named `*.island.svelte` also runs in the browser: see [Islands](#islands). The section [Coming from Svelte and SvelteKit](#coming-from-svelte-and-sveltekit) lists what differs.

## Project layout

```
shugo.json       settings (optional)
package.json     shugo, and the packages the code imports
tsconfig.json    { "extends": "shugo/tsconfig.json" }
src/
  site.html      the document every page stands in: <html>, <head>, <body>
  pages/         pages and layouts; the path of a file is its address;
                 sitemap.xml.ts and the like give files that are not HTML
  components/    components
  lib/           shared TypeScript: helpers
  styles/        style sheets
data/            the data database of the site
public/          files served as they are: /robots.txt, /img/logo.svg
_build/          written by shugo; never edit, never commit
```

Only `src/pages/` is required. Every `.svelte` file lives in `src/`. `@/` in an import is `src/`: `import { price } from "@/lib/shop"`.

Shugo is a package of the project: `bun add shugo`, then `bunx shugo build` (with Node, `npm install shugo` and `npx shugo build`). The types of the module `shugo` come with it, and those of Svelte: `tsconfig.json` is the one line above, and the editor knows `shugo`, `svelte` and the runes.

Svelte comes with the package shugo, of the one version Shugo is made for, and so do `svelte-check` and TypeScript: never install them yourself. A package from npm is installed with the package manager of the project, `bun add date-fns` or `npm install date-fns`, and imported by its name. It is bundled with the code of the site, and `svelte` in it is that Svelte. The build says at the import when a package is not installed, needs another version of Svelte or brings native code. A package that reads files next to its own code does not find them. A Svelte component from a package is an error in a page: copy it into `src/components/`.

`shugo.json`, every field optional:

```json
{
  "site": "https://example.com",
  "data": "data/site.db",
  "redirects": { "/old": "/new" },
  "runtime": "bun"
}
```

`site` is where the site is published; `page.url` is built from it (`http://localhost` without it). `data` is the SQLite database the pages read their data from: see [Data](#data). `runtime` is `bun` or `node`.

## Commands

```
shugo build     compile the project and compute every page
shugo update    bring the site up to date after the data or the project changed
shugo dev       update, serve at http://localhost:4321, update again on every change
shugo serve     serve the site: every page is built on request from what build and update computed
shugo export    write the site as files into _build/dist; after the first time, only what changed
shugo check     check the types and the props of components with svelte-check and TypeScript
```

Run them through the package manager: `bunx shugo build`. Run `shugo build` after every change and read the errors and warnings: each names a file, a line and a column. Those of Svelte, accessibility warnings among them, end with a link to their page on svelte.dev.

`shugo update` is for data that changed: it computes the pages and components that read a row of the data database which is not what it was, and nothing else. It takes tens of milliseconds, whatever the size of the site. If a file of the project changed, it compiles the project first and computes everything, so it is always safe to run. An update that fails changes nothing: the site stays as it was, and the next update meets the same changes again.

`shugo update -check` computes every page and names the ones that came out different although no row they read has changed. Such a page reads data past the data database. Run it when a page looks stale.

`shugo serve` keeps nothing of the site in memory: every request reads its page from the store, so a page shows an update as soon as the update ends.

`shugo export` writes `/about` as `about.html` and `/` as `index.html`, as SvelteKit prerenders, the page for missing addresses as `404.html`, a file such as `/sitemap.xml` under its own name, and copies `public/` and the style sheet. It leaves a list in the folder, `.shugo-export.json`, and the next export follows it: it builds only the pages whose HTML changed since, removes the files of pages that are gone, and copies only the files that changed. After `shugo build`, or into the folder of another project, it builds every page and removes the page files the site no longer has. The folder belongs to the export: a file edited there by hand stays so until its page changes, and a folder whose name starts with a dot is left alone. `-out` names another folder.

## A page

```svelte
<script lang="ts">
  import { query } from "shugo"
  import Card from "../components/Card.svelte"
  import { price } from "@/lib/shop"

  const products = query("products").order("title").all()
  const total = products.reduce((sum, item) => sum + item.price, 0)
</script>

<svelte:head><title>Catalogue</title></svelte:head>

<h1>Catalogue</h1>
<p>{products.length} products, {price(total)} in all</p>

<ul>
  {#each products as product, i}
    <li><Card title={product.title} featured={i === 0} /></li>
  {:else}
    <li>Nothing yet.</li>
  {/each}
</ul>
```

- The `<script>` is optional: a file may be markup alone.
- `<script>` runs once for every page the file gives, and once for every place a component is shown; a component kept apart runs once for each set of its props, see [Where to ask](#where-to-ask). Everything it declares is visible in the markup.
- `<script module>` runs once, when the module loads. `getStaticPaths` goes there: in `<script>` it is an error. Names imported in `<script module>` are seen in `<script>` too; importing one again is an error of Svelte.
- There is no `await` in `<script>` or in the markup: a page is computed without waiting, since `query` answers at once. What takes time and needs doing once, such as making a highlighter, can be awaited at the top of `<script module>` or of a module in `src/lib/`: the code waits once, when it loads. Top-level `return` is an error.
- TypeScript: write `<script lang="ts">`. Svelte removes the types and does not check them: a type error does not stop the build, and `post.titel` shows nothing. `shugo check` finds such mistakes, and wrong props of components: run it before you are done. TypeScript that turns into code, such as `enum`, is an error: use an object `as const`. Import a type with `type`: `import type { Product } from "@/lib/shop"`, or `import { price, type Product } from "@/lib/shop"`. Svelte does not know that `Product` is a type and keeps a plain import of it, and the build stops with `No matching export`.
- Only runes: `export let`, `$:`, `<svelte:options runes={false} />` and the slots of Svelte 4 (`<slot>`, `slot="…"` right inside a component's tag, `let:`, `<svelte:fragment>`) are errors; take props with `$props()` and markup with snippets.
- Modules with runes, `counter.svelte.ts` or `state.svelte.js`, are compiled by Svelte too: import them like any module. In an island they run in the browser. In `src/pages/` they are code to import, not files of the site.
- A page is rendered on the server: `$state(x)` is just `x`, `$derived(…)` is computed once, `$effect` never runs. The page is static.
- A page may import these modules of Svelte: `svelte`, `svelte/store`, `svelte/reactivity`, `svelte/reactivity/window`, `svelte/easing`, `svelte/transition`, `svelte/animate`, `svelte/motion`, `svelte/events`, `svelte/attachments`, `svelte/server`. Any other, such as `svelte/legacy`, is an error.

## Text and attributes

How a value is written:

| Value | In text | In an attribute |
| --- | --- | --- |
| string, number | itself, escaped | itself, escaped |
| `null`, `undefined` | nothing | no attribute |
| `false`, `true` | `false`, `true` | `"false"`, `"true"` |
| array | items joined by commas: `1,2` | the same |
| object | `[object Object]` | the same |

- So `{inStock && "In stock"}` shows `false` when `inStock` is false: write `{inStock ? "In stock" : ""}` or `{#if inStock}In stock{/if}`.
- Boolean attributes (`disabled`, `checked`, `hidden`, `selected`, `required` and the rest of Svelte's list) are written `disabled=""` and dropped when false.
- `class` takes arrays and objects: `class={["card", { featured, sale: price < old }]}`. There are `class:name={cond}` and `style:color={value}`.
- Values inside quotes work: `title="Order {id}"`. `{src}` is `src={src}`; `{...rest}` spreads an object.
- Event handlers (`onclick`) and `bind:`, `use:`, `transition:` do nothing on a static page; `bind:value` writes the value.

**Markup cannot stand inside `{…}`**: no `{items.map(item => <li>…</li>)}`. Use `{#each}` and `{#if}`.

## Blocks

```svelte
{#if stock > 3}
  In stock
{:else if stock > 0}
  Only {stock} left
{:else}
  On order
{/if}

{#each rows as { id, title }, i (id)}…{/each}
{@html post.html}
{@const total = a + b}
```

- `{#each}` takes arrays and anything like them: `Set`, `Map`, a string; `null` and `undefined` are an empty list. A key in parentheses is accepted and changes nothing on a static page.
- `{@html value}` writes the value unescaped: rendered Markdown, content from a CMS.
- `{#await promise}` shows its pending branch: a page is rendered without waiting.
- `<svelte:element this={tag}>` is an element whose name is a value.

## Whitespace and markup

- **White space at the edges of an element or a block disappears**: `{#if price}${price} {/if}` loses the space before `{/if}`, and `<b> x </b>` is `<b>x</b>`. Write a space you need at the edge as `{" "}`.
- White space between tags and inside text collapses to one space; `<pre>` and `<textarea>` keep it.
- `<!-- comments -->` do not reach the page.
- `&nbsp;` and other entities become characters.
- An element the browser would move — a `<div>` inside a `<p>`, a `<tr>` right inside a `<table>` — is an error; through a block, a warning.
- `</script>` anywhere inside a `<script>`, even in a string or a comment, ends it.
- A literal brace: `{"{"}`.

## Components

A component is a `.svelte` file. Import it in the script and use it as a tag that starts with a capital letter. **Import every component you use**, in layouts too.

```svelte
<!-- src/components/Card.svelte -->
<script lang="ts">
  import type { Snippet } from "svelte"

  interface Props {
    title: string
    featured?: boolean
    children?: Snippet
    footer?: Snippet
  }
  let { title, featured = false, children, footer }: Props = $props()
</script>

<section class={["card", { featured }]}>
  <h2>{title}</h2>
  {#if children}{@render children()}{:else}shown when nothing was put inside{/if}
  {#if footer}<footer>{@render footer()}</footer>{/if}
</section>

<style>
  .card { border: 1px solid #ddd; }
</style>
```

```svelte
<Card title="Jug" featured>
  <p>this is children</p>
  {#snippet footer()}<small>this is footer</small>{/snippet}
</Card>
```

- Attributes are the props: `title="text"` is a string, `featured` is `true`, `count={3}` is a value, `{...props}` gives every field of an object.
- What stands inside the tag is the snippet `children`; `{#snippet name()}` inside the tag is the prop `name`. A snippet may take parameters: `{#snippet row(item)}…{/snippet}`, shown with `{@render row(item)}`.
- A snippet sees the variables of the template that wrote it.
- `<style>` applies to its own component only; `:global(…)` reaches the whole site.
- A component shows itself by importing itself: `import Tree from "./Tree.svelte"` inside `Tree.svelte`.
- A component in a variable is shown by its tag: `const View = compact ? List : Grid`, then `<View {items} />`.

### The module `shugo`

What the framework knows is imported from the module `shugo`, in any file: a page, a layout, a component, a module of `src/lib/`.

```ts
import { query, paginate, page, search } from "shugo"
```

| | |
| --- | --- |
| `query("table")` | a question to the data database: see [Data](#data) |
| `paginate(query, { pageSize })` | one page of a long list: see [Pages from data](#pages-from-data) |
| `page.params` | the parameters of the address of the page: `{ slug: "jug" }` |
| `page.url` | the address of the page, a `URL` |
| `page.route.id` | the route: `/products/[slug]` |
| `page.status` | 200, and 404 on `src/pages/404.svelte` |
| `search("table", { … })` | a search of a table, for islands: see [Islands](#islands) |

`page` is the `page` of `$app/state` in SvelteKit, with the fields a static page has. Props come from `$props()`. There is no global `Shugo`: writing it is an error that names the import.

## Layouts

`src/pages/+layout.svelte` wraps every page in its folder and below. A layout in a deeper folder goes inside the outer one. A layout gets the page as `children`, like `+layout.svelte` in SvelteKit.

```svelte
<!-- src/pages/+layout.svelte -->
<script>
  import Header from "../components/Header.svelte"
  import "../styles/main.css"

  let { children } = $props()
</script>

<svelte:head><title>My site</title></svelte:head>

<Header />
<main>{@render children()}</main>
```

- `{@render children()}` is where the page goes. A layout must show it.
- A layout starts with what goes into the `<body>`, like `+layout.svelte`. The document itself is `src/site.html`, as `src/app.html` in SvelteKit:

  ```html
  <!doctype html>
  <html lang="en">
    <head>
      <meta charset="utf-8">
      <meta name="viewport" content="width=device-width, initial-scale=1">
      %shugo.head%
    </head>
    <body>
      %shugo.body%
    </body>
  </html>
  ```

  `%shugo.body%` is the page in its layouts, `%shugo.head%` what the templates give in `<svelte:head>` and then the link to the style sheet. Each stands once. The file is plain HTML, copied as it is: no expressions. Without it Shugo uses the one above, without `lang`. `<html>`, `<head>`, `<body>` or `<!doctype>` in a template is an error.
- A page gives its title and meta tags with `<svelte:head>` at its top level; any component may. They go to `%shugo.head%`, in the order of the document: outer layouts first, the page last. A page has one `<title>`, the last one, so the layout above gives a default and a page replaces it.
- Pages do not import or call layouts: importing a `+layout.svelte` is an error. Markup that pages share goes into a component.
- A layout is kept apart from its pages: once for the whole site when it reads nothing of the page, once for every page when it imports `page` from `shugo`, for example to highlight the current menu item. A change of what the layout reads computes the layout and not the pages; a change of a page computes the page and not its layout.
- So a layout is rendered apart from the page in it: a context the layout sets with `setContext` reaches its own components, not the page. Pass what the page needs through `query` or a module of `src/lib/`.

## Addresses

| File | Address |
| --- | --- |
| `src/pages/index.svelte` | `/` |
| `src/pages/about.svelte` | `/about` |
| `src/pages/blog/[slug].svelte` | `/blog/…` |
| `src/pages/docs/[...path].svelte` | `/docs` and everything below |
| `src/pages/(shop)/cart.svelte` | `/cart` — a folder in parentheses is not a part of the address |
| `src/pages/_parts/x.svelte` | not a page |
| `src/pages/404.svelte` | shown for addresses that have no page |
| `src/pages/sitemap.xml.ts` | `/sitemap.xml`: a file that is not HTML, see [Files that are not HTML](#files-that-are-not-html) |

Names that start with `+` are kept for the framework; only `+layout.svelte` exists. Addresses have no slash at the end: the server sends `/about/` on to `/about`.

When two files could give one address, the one that names more of it wins: `blog/new.svelte` over `blog/[slug].svelte` over `blog/[...path].svelte`. So `blog/[...page].svelte` and `blog/posts/[slug].svelte` live side by side.

## Pages from data

A file with a parameter in its name must say which pages there are. `getStaticPaths`, exported from `<script module>`, takes nothing and asks the data database with `query`:

```svelte
<!-- src/pages/products/[slug].svelte -->
<script module>
  import { query, page } from "shugo"

  export function getStaticPaths() {
    return query("products").select("slug").all().map(({ slug }) => ({ params: { slug } }))
  }
</script>

<script>
  const product = query("products").where({ slug: page.params.slug }).first()
</script>

<h1>{product.title}</h1>
```

`getStaticPaths` runs once; `<script>` runs for every address it returned. `page.params` holds the parameters as they stand in the address of the page: always text, and only the ones the file name has. `getStaticPaths({ query })` is an error: import instead.

**`getStaticPaths` returns addresses alone, `{ params }`, and every page asks for its own data by them.** Then one product that changes has one page computed. An item with `props` is an error. An address returned twice is one page, as in SvelteKit: no need to remove repeats.

The list of addresses is the one place where `select` pays: of a hundred thousand rows it fetches one column instead of all.

For lists split into pages name the file `[...page].svelte`. `paginate` in `getStaticPaths` takes the number of items and says which pages there are; `paginate` in the page takes a query, or an array, and returns the items of that page:

```svelte
<!-- src/pages/blog/[...page].svelte gives /blog, /blog/2, /blog/3 … -->
<script module>
  import { query, paginate } from "shugo"

  export function getStaticPaths() {
    return paginate(query("posts").count(), { pageSize: 10 })
  }
</script>

<script>
  const list = paginate(query("posts").order("date", "desc"), { pageSize: 10 })
</script>

{#each list.data as post}…{/each}
{#if list.url.prev}<a href={list.url.prev}>Newer</a>{/if}
{#if list.url.next}<a href={list.url.next}>Older</a>{/if}
```

| | |
| --- | --- |
| `list.data` | the rows of this page |
| `list.currentPage`, `list.lastPage` | numbers from 1 |
| `list.total`, `list.size` | the number of all items, and of items on a page |
| `list.start`, `list.end` | the places of the first and the last item of this page, from 0 |
| `list.url.current`, `.prev`, `.next`, `.first`, `.last` | addresses; `undefined` where there is no such page |

With `[page].svelte` the first page is `/blog/1`. `paginate(count, { pageSize, params })` adds `params` to every page, for a list inside something else: `src/pages/tags/[tag]/[...page].svelte` with `params: { tag }`. The same `pageSize` goes to both calls. Do not name the result `page`: that is the page of the module `shugo`.

## Files that are not HTML

A module among the pages gives a file whose type stands in its name before `.ts`: `src/pages/sitemap.xml.ts` gives `/sitemap.xml`, `src/pages/robots.txt.ts` gives `/robots.txt`. It exports `GET`, which takes nothing and returns the text of the file:

```ts
// src/pages/sitemap.xml.ts
import { query, page } from "shugo"

export function GET() {
  const products = query("products").select("slug").order("slug").all()
  const urls = products.map((p) => `<url><loc>${new URL("/products/" + p.slug, page.url)}</loc></url>`)
  return `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">${urls.join("")}</urlset>`
}
```

- `GET` returns a string, at once: not `async`, not a `Response`. Data comes from `query`, the address from `page`, as in a page; `page.url` is the full address of the file on the site of `shugo.json`.
- What `GET` reads is tracked as for a page: the file is computed again when a row it read changes, here when a product is added, removed or gets another slug.
- A parameter works as in a page: `src/pages/api/[slug].json.ts` with `getStaticPaths` gives `/api/jug.json`; `GET` reads `page.params.slug`.
- The text reaches the file as it is: no shell, no layouts, no escaping. Escape what you put into XML yourself.
- The server answers with the type of the extension, as for `public/`; the export writes the file under its address.
- A module without a type in its name, `src/pages/data.ts`, is an error. Code for such modules to import goes to `src/lib/` or into a file whose name starts with `_`.

## Data

The data of a site lives in one SQLite database, named in `shugo.json`: `{ "data": "data/site.db" }`. Whatever brings data to the site — an import script, a sync with a CMS, a person with a SQL console — writes to that database. Pages read it with `query` from the module `shugo`, and only with it.

That is what makes updates exact. Shugo notes for every page and every shared component which rows it asked for and which columns of them were read while it was computed — in its script, in its template, in the functions they call, in the components the template gives a row to. The database keeps a journal of the rows that change. `shugo update` computes what read a changed column of a changed row. Nothing has to be marked, declared or invalidated, and a query need not name its columns.

### Rules for the database

- **Every table has a primary key of one column.** By it Shugo tells the rows of a table apart. A table of pairs gets a key of its own: `id INTEGER PRIMARY KEY`.
- Shugo adds a table `_shugo_journal`, a table `_shugo_meta`, and triggers and indexes named `_shugo_…` to the database and switches it to WAL mode. Leave them alone. Write to the tables with anything, in any way; `INSERT OR REPLACE` and upserts are fine.
- **Make no indexes for the pages.** Shugo makes the ones their queries need, the first time a query is asked: by the columns that `where` compares with values, then by the columns of `order`, then by the key. An index over those columns that is there already is used instead. A column that must not repeat a value is said in the table: `slug TEXT NOT NULL UNIQUE`.
- After a table is added, removed or altered, the next update computes everything once.
- Views are not queried: Shugo cannot tell which rows a view is made of. Query the tables.
- Data read from anywhere else — a file, `fetch`, a value that a module keeps between pages — is not tracked: `shugo update` does not see it change. `shugo build` computes everything and reads it again; `shugo update -check` names the pages that depend on such data.

### Queries

```ts
query("products")                             // a table
  .where({ category: "c9", available: true }) // equal values
  .where("price", "<", 1000)                  // =  !=  <  <=  >  >=  like  in
  .order("position")                          // .order("price", "desc")
  .limit(24).offset(48)
  .all()                                      // the rows, as objects
```

| | |
| --- | --- |
| `.all()` | the rows |
| `.first()` | the first row, or `null` |
| `.get(key)` | the row with this primary key, or `null` |
| `.count()` | how many rows the `where` finds; order, limit and offset do not matter |
| `.select("title", "price")` | take these columns only, to fetch less; without it a row has all its columns |
| `.join("categories", "category")` | puts into every row the row of `categories` whose key is the row's `category`, as `row.categories`, or `null`. Options: `{ select: ["title"], as: "group", on: "slug" }`; `on` must be the key or a unique column of the other table. For many rows, make a second query |

- Every step returns a new query, so a query can be kept and asked in several ways.
- The primary key is in every row, selected or not.
- **A page depends on what is read of its rows, not on what a query took.** A row notes every column that is read of it: `item.title` in a template, `item.price` inside a function that got the row, `variant.stock` in a component that got the row. A change of any other column does not reach the page. A branch of `{#if}` that is not taken reads nothing; when the data sends the page into it, the page is computed again.
- Taking a row as a whole reads all of it: `{ ...row }`, `JSON.stringify(row)`, `Object.keys(row)`, `{...row}` among attributes. Read the columns you show instead.
- **A row belongs to the script that asked for it.** Do not keep rows or a query in a variable of a module for other pages: read there, a row would leave that page depending on nothing, so it throws and says so. Let every script ask for its rows; the database is fast.
- `structuredClone(row)` does not work on a row; copy with `{ ...row }`.
- **Compare with values of the column's kind**: a text for a text column, a number for a number column. `where({ id: 5 })` on a text column is an error that says what to write.
- `where({ column: null })` finds rows without a value.
- Rows come in the order of `order()`. Rows that it does not tell apart, and rows of a query without it, come in the order of their keys. So the same data always gives the same page, however the rows got into the table. To show rows in an order that means something, keep it in a column and say `order("position")`.
- A query runs when it is asked, synchronously; there is nothing to `await`.

### Where to ask

- **In the page**, for what the page shows.
- **In a component called with an id**, for what many pages show: `<ProductCard id={row.id} />` with `let { id } = $props()` and `query("products").get(id)` inside. Such a component is computed once for every id and kept on its own, so when the product changes its card is computed, and none of the pages that show it. A component is computed once like this when neither it nor any module it imports takes `page` or `paginate` from `shugo` or the context of Svelte (`getContext`, `setContext` and the like), calls `$props.id()` or shows a snippet inside a `<select>`, and it is called with plain values (strings, numbers, booleans, `null`), snippets without parameters written inside its tag, or without props (`bind:` is not a plain value). Otherwise it is rendered where it stands, as a part of the page: the HTML is the same either way.
- **In a component without props**, for what every page shows: `<Header />` reads the menu itself.
- A build warns of a component kept apart for nothing: 100 shared blocks or more, each shown in fewer than 1.5 places on average, as `<Price value={n} />` with a number of its own on nearly every page. Give such a component an object, such as the row it shows, and it is computed as a part of the page.
- In a function of `src/lib/` that a page or a component calls: the query belongs to whoever is computed when it runs. Not at the top of a module: nothing is computed there, so it is an error. A module of `src/lib/` that imports `page` makes every component that imports it depend on the page.
- A component that got a row from its page (`<Variant variant={row} />`) needs no query: it shows what the page asked for, and the page depends on what the component reads of the row.
- Throw an error when a row that must be there is not: `if (!product) throw new Error(…)`.

Create `Intl.NumberFormat`, `Intl.DateTimeFormat` and the like once in a module, not on every call: they are slow to make.

## Styles and scripts

- `import "../styles/main.css"` in a script adds a style sheet to the site. All style sheets and the styles of all components become one file, linked from every page.
- `<style>` of a component applies to that component only; `:global(…)` and `:global { … }` reach the whole site. A selector that selects nothing in the markup of its own file is dropped, with a warning: elements of a child component, or of a snippet written elsewhere, are styled where they are written, or with `:global(…)`.
- Tailwind works when it is installed in the project (`npm install tailwindcss @tailwindcss/cli`) and a style sheet has `@import "tailwindcss";`.
- Code for the browser goes into a `<script>` inside an element or inside `<svelte:head>`; it reaches the page as written, braces included. Pass values through attributes: `<div id="map" data-places={JSON.stringify(places)}>`.
- A component with such a `<script>` shown ten times gives ten copies of it. Keep such a script in the page or in the layout.

## Islands

A component named `*.island.svelte` is an island: the page shows its HTML, and in the browser Svelte hydrates it, so runes, events and actions work there. A template imports it by name and gives it JSON props:

```svelte
<!-- src/pages/catalog.svelte -->
<script>
  import Catalog from "../components/Catalog.island.svelte"
  import { search } from "shugo"

  const products = search("products", {
    text: ["title", "description"],
    filters: { category: "multi", price: "range", available: "boolean" },
    sorts: { popular: { position: "asc" }, cheap: { price: "asc" } },
    defaultSort: "popular",
    pageSize: 24,
  })
</script>

<Catalog request:initial={products} action="/catalog" />
```

- Nothing else is installed for islands: the Svelte of the package shugo runs in the browser too. An island is written with runes, as every template is. It is code for the browser, and a Svelte component from a package works in it.
- Props are JSON data: no functions, no snippets, nothing inside the tag.
- An island may import regular components, such as `ProductCard.svelte`, when they take data through props and use Svelte alone: no `query`, no `page`.
- `search("table", { … })` describes a search of a table of the data database; Shugo builds and names its index. `text` are the columns of the full-text search by `q`; `filters` take `multi` (`category=cups&category=plates`), `range` (`price_from`, `price_to`) and `boolean` (`available=1`); `sorts` and `defaultSort` name the orders for `sort`; `where` fixes a part of the table: `{ published: true }`. The result has `hits`, `total`, `facets`, `query`, the fields of pages and `endpoint`.
- `request:initial={products}` binds the prop `initial` to the search of the address: `/catalog?category=cups&sort=cheap` comes from the server with those results already, built by Go from a program the build made of the island, without JavaScript. In the browser the island asks `initial.endpoint` with the same parameters and keeps the address in step through the History API. A plain prop, `initial={products}`, gets what the page computed.
- What a page reads through `search` is tracked: an update computes it again when the first page of results changes.

## Coming from Svelte and SvelteKit

| Svelte, SvelteKit | Shugo |
| --- | --- |
| `.svelte` | `.svelte` for static templates, `*.island.svelte` for a hydration entry |
| `+page.svelte` in a folder per address | a file per address in `src/pages/` |
| `+layout.svelte`, `+layout.ts` with `load` | `+layout.svelte`; data comes from `query` in whoever shows it |
| `+page.ts` with `load`, `entries` | `<script>` asks `query`; `getStaticPaths` in `<script module>` lists the pages |
| `page` from `$app/state` | `page` from `shugo`: `page.params`, `page.url`, `page.route.id`, `page.status` |
| `$lib/…` | `@/lib/…` |
| `src/app.html` with `%sveltekit.head%`, `%sveltekit.body%` | `src/site.html` with `%shugo.head%`, `%shugo.body%` |
| `+error.svelte` | `src/pages/404.svelte` |
| `+server.ts` with `GET` and `prerender` | `src/pages/sitemap.xml.ts` with `GET` that returns a string |
| hydration comments `<!--[-->`, `<!--]-->` in the HTML | none, but in islands; what `{@html}` gives stays as it is |
| a context set in `+layout.svelte` reaches the page | it does not: the layout is rendered apart |
| warnings about what changes in the browser, such as `state_referenced_locally` | not shown for pages, which do not change in the browser; islands get them |
| `svelte/legacy` | an error; the modules a page may import are listed in [A page](#a-page) |
| hydration, `$state` that changes, events | only in `*.island.svelte`; the rest of the page is static |
