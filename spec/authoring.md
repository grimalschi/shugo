# Writing a Shugo site

Shugo builds static sites. A page, a layout and a component are each a `.shugo` file, and a `.shugo` file is a **Svelte 5 component**: `<script>`, markup, `<style>`. Shugo renders it exactly as the server renderer of Svelte 5.59 does, so what you know of Svelte holds: `$props()`, `{#if}`, `{#each}`, snippets, `{@render}`, `{@html}`, scoped styles, the same whitespace rules. What is Shugo's own: data comes from the site's SQLite database through `Shugo.query`, and a page with a parameter in its name lists its pages with `getStaticPaths`. The section [Coming from Svelte and SvelteKit](#coming-from-svelte-and-sveltekit) lists what differs.

## Project layout

```
shugo.json       settings (optional)
src/
  pages/         pages and layouts; the path of a file is its address
  components/    components
  lib/           shared TypeScript: helpers
  styles/        style sheets
data/            the data database of the site
public/          files served as they are: /robots.txt, /img/logo.svg
_build/          written by shugo; never edit, never commit
```

Only `src/pages/` is required. Every `.shugo` file lives in `src/`. `@/` in an import is `src/`: `import { price } from "@/lib/shop"`.

`shugo.json`, every field optional:

```json
{
  "site": "https://example.com",
  "data": "data/site.db",
  "redirects": { "/old": "/new" },
  "runtime": "bun"
}
```

`site` is where the site is published; `Shugo.url` is built from it (`http://localhost` without it). `data` is the SQLite database the pages read their data from: see [Data](#data). `runtime` is `bun` or `node`.

## Commands

```
shugo build     compile the project and compute every page
shugo update    bring the site up to date after the data or the project changed
shugo dev       update, serve at http://localhost:4321, update again on every change
shugo serve     serve the site: every page is built on request from what build and update computed
shugo export    write the site as files into _build/dist; after the first time, only what changed
```

Run `shugo build` after every change and read the errors and warnings: each names a file, a line and a column.

`shugo update` is for data that changed: it computes the pages and components that read a row of the data database which is not what it was, and nothing else. It takes tens of milliseconds, whatever the size of the site. If a file of the project changed, it compiles the project first and computes everything, so it is always safe to run. An update that fails changes nothing: the site stays as it was, and the next update meets the same changes again.

`shugo update -check` computes every page and names the ones that came out different although no row they read has changed. Such a page reads data past the data database. Run it when a page looks stale.

`shugo serve` keeps nothing of the site in memory: every request reads its page from the store, so a page shows an update as soon as the update ends.

`shugo export` writes `/about` as `about.html` and `/` as `index.html`, as SvelteKit prerenders, the page for missing addresses as `404.html`, and copies `public/` and the style sheet. It leaves a list in the folder, `.shugo-export.json`, and the next export follows it: it builds only the pages whose HTML changed since, removes the files of pages that are gone, and copies only the files that changed. After `shugo build`, or into the folder of another project, it builds every page and removes the page files the site no longer has. The folder belongs to the export: a file edited there by hand stays so until its page changes, and a folder whose name starts with a dot is left alone. `-out` names another folder.

## A page

```svelte
<script lang="ts">
  import Card from "../components/Card.shugo"
  import { price } from "@/lib/shop"

  const products = Shugo.query("products").order("title").all()
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
- `<script>` runs once for every page the file gives, and once for every place a component is shown. Everything it declares is visible in the markup.
- `<script module>` runs once, when the module loads. `getStaticPaths` goes there, and so does any other export: an export from `<script>` is an error.
- There is no `await`: a page is computed without waiting, since `Shugo.query` answers at once. What takes time and needs doing once, such as making a highlighter, can be awaited at the top of a module in `src/lib/`. Top-level `return` is an error.
- TypeScript: write `<script lang="ts">`, as in Svelte. Types are removed, not checked: a type error does not stop the build, and `post.titel` shows nothing.
- Runes behave as on Svelte's server: `$state(x)` is just `x`, `$derived(…)` is computed once, `$effect` never runs. The page is static.

## Text and attributes

Values are written as Svelte writes them:

| Value | In text | In an attribute |
| --- | --- | --- |
| string, number | itself, escaped | itself, escaped |
| `null`, `undefined` | nothing | no attribute |
| `false`, `true` | `false`, `true` | `"false"`, `"true"` |
| array | items joined by commas: `1,2` | the same |
| object | `[object Object]` | the same |

- So `{user && user.name}` shows `false` when there is no user: write `{user ? user.name : ""}` or `{#if user}{user.name}{/if}`.
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
- `{#await promise}` shows its pending branch, as Svelte's server does; there is nothing to wait for in Shugo.
- `<svelte:element this={tag}>` is an element whose name is a value.

## Whitespace and markup

The rules of Svelte:

- **White space at the edges of an element or a block disappears**: `{#if price}${price} {/if}` loses the space before `{/if}`, and `<b> x </b>` is `<b>x</b>`. Write a space you need at the edge as `{" "}`.
- White space between tags and inside text collapses to one space; `<pre>` and `<textarea>` keep it.
- `<!-- comments -->` do not reach the page.
- `&nbsp;` and other entities become characters.
- An element the browser would move — a `<div>` inside a `<p>`, a `<tr>` right inside a `<table>` — is an error; through a block, a warning.
- `</script>` anywhere inside a `<script>`, even in a string or a comment, ends it.
- A literal brace: `{"{"}`.

## Components

A component is a `.shugo` file. Import it in the script and use it as a tag that starts with a capital letter. **Import every component you use**, in layouts too.

```svelte
<!-- src/components/Card.shugo -->
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
- `<style>` applies to its own component only, as in Svelte; `:global(…)` reaches the whole site.
- A component shows itself by importing itself: `import Tree from "./Tree.shugo"` inside `Tree.shugo`.
- A component in a variable is shown by its tag: `const View = compact ? List : Grid`, then `<View {items} />`.

### `Shugo`

| | |
| --- | --- |
| `Shugo.query("table")` | a question to the data database: see [Data](#data) |
| `Shugo.paginate(query, { pageSize })` | one page of a long list: see [Pages from data](#pages-from-data) |
| `Shugo.params` | the parameters of the address of the page |
| `Shugo.url` | the address of the page, a `URL` |

Props come from `$props()`, not from `Shugo`.

## Layouts

`src/pages/+layout.shugo` wraps every page in its folder and below. A layout in a deeper folder goes inside the outer one. A layout gets the page as `children`, like `+layout.svelte` in SvelteKit.

```svelte
<!-- src/pages/+layout.shugo -->
<script>
  import Header from "../components/Header.shugo"
  import "../styles/main.css"

  let { children } = $props()
</script>

<svelte:head><title>My site</title></svelte:head>

<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
  </head>
  <body>
    <Header />
    <main>{@render children()}</main>
  </body>
</html>
```

- `{@render children()}` is where the page goes. A layout must show it.
- The root layout writes `<html>`, `<head>` and `<body>` itself; there is no `app.html`.
- A page gives its title and meta tags with `<svelte:head>` at its top level. They go to the end of the `<head>`, in the order of the document: outer layouts first, the page last. A page has one `<title>`, the last one, as in Svelte, so the layout above gives a default and a page replaces it.
- `<svelte:head>` works in pages and layouts, not in other components.
- Pages do not import or call layouts.
- A layout that reads neither `Shugo.url` nor `Shugo.params` is computed once for the whole site. A layout whose menu highlights the current page reads the address, so it is computed for every page, and a change of what it reads computes every page. Keep what is the same on every page in a component without props, such as `<Menu />` that reads the menu itself, and highlight the current item in the browser.

## Addresses

| File | Address |
| --- | --- |
| `src/pages/index.shugo` | `/` |
| `src/pages/about.shugo` | `/about` |
| `src/pages/blog/[slug].shugo` | `/blog/…` |
| `src/pages/docs/[...path].shugo` | `/docs` and everything below |
| `src/pages/(shop)/cart.shugo` | `/cart` — a folder in parentheses is not a part of the address |
| `src/pages/_parts/x.shugo` | not a page |
| `src/pages/404.shugo` | shown for addresses that have no page |

Names that start with `+` are kept for the framework; only `+layout.shugo` exists. Addresses have no slash at the end: the server sends `/about/` on to `/about`.

When two files could give one address, the one that names more of it wins: `blog/new.shugo` over `blog/[slug].shugo` over `blog/[...path].shugo`. So `blog/[...page].shugo` and `blog/posts/[slug].shugo` live side by side.

## Pages from data

A file with a parameter in its name must say which pages there are. `getStaticPaths`, exported from `<script module>`, gets `query` to ask the data database:

```svelte
<!-- src/pages/products/[slug].shugo -->
<script module>
  export function getStaticPaths({ query }) {
    return query("products").select("slug").all().map(({ slug }) => ({ params: { slug } }))
  }
</script>

<script>
  const product = Shugo.query("products").where({ slug: Shugo.params.slug }).first()
</script>

<h1>{product.title}</h1>
```

`getStaticPaths` runs once; `<script>` runs for every item it returned. `Shugo.params` holds the parameters as they stand in the address of the page: always text, and only the ones the file name has.

**Return `params` alone and let every page ask for its own data.** Then one product that changes has one page computed. `getStaticPaths` may also hand rows to its pages, `{ params, props: { product } }`, read with `let { product } = $props()`; but then only the route knows what a page shows, and when anything of a row it read changes, all its pages are computed again.

The list of addresses is the one place where `select` pays: of a hundred thousand rows it fetches one column instead of all.

For lists split into pages name the file `[...page].shugo`. `paginate` in `getStaticPaths` takes the number of items and says which pages there are; `Shugo.paginate` in the page takes a query and returns the items of that page:

```svelte
<!-- src/pages/blog/[...page].shugo gives /blog, /blog/2, /blog/3 … -->
<script module>
  export function getStaticPaths({ query, paginate }) {
    return paginate(query("posts").count(), { pageSize: 10 })
  }
</script>

<script>
  const page = Shugo.paginate(Shugo.query("posts").order("date", "desc"), { pageSize: 10 })
</script>

{#each page.data as post}…{/each}
{#if page.url.prev}<a href={page.url.prev}>Newer</a>{/if}
{#if page.url.next}<a href={page.url.next}>Older</a>{/if}
```

| | |
| --- | --- |
| `page.data` | the rows of this page |
| `page.currentPage`, `page.lastPage` | numbers from 1 |
| `page.total`, `page.size` | the number of all items, and of items on a page |
| `page.start`, `page.end` | the places of the first and the last item of this page, from 0 |
| `page.url.current`, `.prev`, `.next`, `.first`, `.last` | addresses; `undefined` where there is no such page |

With `[page].shugo` the first page is `/blog/1`. `paginate(count, { pageSize, params })` adds `params` to every page, for a list inside something else: `src/pages/tags/[tag]/[...page].shugo` with `params: { tag }`. The same `pageSize` goes to both calls.

## Data

The data of a site lives in one SQLite database, named in `shugo.json`: `{ "data": "data/site.db" }`. Whatever brings data to the site — an import script, a sync with a CMS, a person with a SQL console — writes to that database. Pages read it with `Shugo.query`, and only with it.

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
Shugo.query("products")                       // a table
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
- **A row belongs to the script that asked for it.** Do not keep rows, `Shugo` or a query in a variable of a module for other pages: read there, a row would leave that page depending on nothing, so it throws and says so. Let every script ask for its rows; the database is fast.
- `structuredClone(row)` does not work on a row; copy with `{ ...row }`.
- **Compare with values of the column's kind**: a text for a text column, a number for a number column. `where({ id: 5 })` on a text column is an error that says what to write.
- `where({ column: null })` finds rows without a value.
- Rows come in the order of `order()`. Rows that it does not tell apart, and rows of a query without it, come in the order of their keys. So the same data always gives the same page, however the rows got into the table. To show rows in an order that means something, keep it in a column and say `order("position")`.
- A query runs when it is asked, synchronously; there is nothing to `await`.

### Where to ask

- **In the page**, for what the page shows.
- **In a component called with an id**, for what many pages show: `<ProductCard id={row.id} />` with `let { id } = $props()` and `Shugo.query("products").get(id)` inside. Such a component is computed once for every id and kept on its own, so when the product changes its card is computed, and none of the pages that show it. A component is computed once like this when neither it nor any component it imports reads `Shugo.url` or `Shugo.params`, and it is called with plain values (strings, numbers, booleans) or without props.
- **In a component without props**, for what every page shows: `<Header />` reads the menu itself.
- Not in a module of `src/lib/` when it is imported: a query belongs to the page or component that asks, so only `Shugo.query` and the `query` of `getStaticPaths` can ask. A helper takes the query function or the rows as an argument.
- A component that got a row from its page (`<Variant variant={row} />`) needs no query: it shows what the page asked for, and the page depends on what the component reads of the row.
- Throw an error when a row that must be there is not: `if (!product) throw new Error(…)`.

Create `Intl.NumberFormat`, `Intl.DateTimeFormat` and the like once in a module, not on every call: they are slow to make.

## Styles and scripts

- `import "../styles/main.css"` in a script adds a style sheet to the site. All style sheets and the styles of all components become one file, linked from every page.
- `<style>` of a component applies to that component only; `:global(…)` and `:global { … }` reach the whole site.
- Tailwind works when it is installed in the project (`npm install tailwindcss @tailwindcss/cli`) and a style sheet has `@import "tailwindcss";`.
- Code for the browser goes into a `<script>` inside an element or inside `<svelte:head>`; it reaches the page as written, braces included. Pass values through attributes: `<div id="map" data-places={JSON.stringify(places)}>`.
- A component with such a `<script>` shown ten times gives ten copies of it. Keep such a script in the page or in the layout.

## Coming from Svelte and SvelteKit

| Svelte, SvelteKit | Shugo |
| --- | --- |
| `.svelte` | `.shugo`; the same component inside |
| `+page.svelte` in a folder per address | a file per address in `src/pages/` |
| `+layout.svelte`, `+layout.ts` with `load` | `+layout.shugo`; data comes from `Shugo.query` in whoever shows it |
| `+page.ts` with `load`, `entries` | `<script>` asks `Shugo.query`; `getStaticPaths` in `<script module>` lists the pages |
| `page` from `$app/state` | `Shugo.params`, `Shugo.url` |
| `$lib/…` | `@/lib/…` |
| `src/app.html` | the root layout writes `<html>`, `<head>`, `<body>` |
| `+error.svelte` | `src/pages/404.shugo` |
| `+server.ts` | not there yet |
| `<svelte:head>` in any component | in pages and layouts |
| TypeScript only with `lang="ts"` | also without it, but write `lang="ts"` so the file moves to Svelte as it is |
| hydration, `$state` that changes, events | none: pages are static |
| `<svelte:self>`, `<svelte:component>` | work, with the warnings of Svelte 5 |
