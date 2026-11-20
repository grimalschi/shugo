# Writing a Shugo site

Shugo builds static sites. A page is a `.shugo` file: TypeScript between two `---` lines, then markup. If you know Astro, most of this is familiar; the section [Coming from Astro](#coming-from-astro) lists what differs.

## Project layout

```
shugo.json       settings (optional)
pages/           pages and layouts; the path of a file is its address
components/      components
lib/             shared TypeScript: helpers
data/            the data database of the site, or its data files
styles/          style sheets
public/          files served as they are: /robots.txt, /img/logo.svg
_build/          written by shugo; never edit, never commit
```

Only `pages/` is required. `@/` in an import is the root of the project.

`shugo.json`, every field optional:

```json
{
  "site": "https://example.com",
  "data": "data/site.db",
  "redirects": { "/old": "/new" },
  "runtime": "bun"
}
```

`site` is where the site is published; `Shugo.url` is built from it (`http://localhost` without it). `data` is the SQLite database the pages read: see [Data](#data). `runtime` is `bun` or `node`.

## Commands

```
shugo build     compile the project and compute every page
shugo update    compute the pages again after the data changed
shugo dev       build, serve at http://localhost:4321, rebuild on every change
shugo serve     serve the site: every page is built on request from what build and update computed
shugo export    write the site as files into _build/dist
```

Run `shugo build` after every change and read the errors: each names a file, a line and a column.

`shugo update` is for data that changed outside of the project's code. With a data database it reads which rows have changed and computes only the pages and shared blocks that read them; its time follows the size of the change, not of the site. Without one it runs every script again, which loads the data, and computes only what shows something that has changed. If a file of the project changed, it compiles the project first, so it is always safe to run.

`shugo update -check` computes everything, whatever has changed, and names the pages that came out different although nothing they are seen to read did. Run it when a page looks stale: see [Data](#data).

`shugo export` writes `/about` as `about/index.html`, the page for missing addresses as `404.html`, and copies `public/` and the style sheet.

## A page

```
---
import Card from "../components/Card.shugo"
import { getProducts, type Product } from "@/lib/shop"
import { price } from "@/lib/format"

const products: Product[] = await getProducts()
const total = products.reduce((sum, item) => sum + item.price, 0)
---
<title slot="head">Catalogue</title>

<h1>Catalogue</h1>
<p>{products.length} products, {price(total)} in all</p>

<ul>
  <g-for each={products} as="product" index="i">
    <li><Card title={product.title} featured={i === 0} /></li>
  </g-for>
</ul>

<g-if cond={products.length === 0}>
  <p>Nothing yet.</p>
</g-if>
```

- The script is optional: a file may be markup alone.
- The script runs once for every page the file gives, on every build and update: it is where data is loaded.
- The markup sees the **values** the script declares and everything the script **imports**. It does not see a function the script declares: import the function from a module, or call it in the script and show the result.
- A function that the markup calls comes from a module that loads nothing: `lib/format.ts` here, apart from `lib/shop.ts` that reads the data. See [Data](#data).
- Top-level `await` is fine in the script, never in the markup. Top-level `return` is an error.
- TypeScript everywhere: in the script and inside `{…}`. Types are removed, not checked: a type error does not stop the build, and `post.titel` shows nothing.
- Markup goes to the page as written, HTML entities included: `&nbsp;`, `&lt;`.

## Expressions

`{expression}` writes a value as text, escaped. An expression is plain TypeScript.

| Value | Shown as |
| --- | --- |
| string, number | itself |
| `null`, `undefined`, `false` | nothing |
| `true` | `true` |
| array | its items one after another, no separator |
| object | `String(value)` |

So `{user && user.name}` works. **Markup cannot stand inside `{…}`**: no `{items.map(item => <li>…</li>)}`, no `{cond && <b>…</b>}`. Use `<g-for>` and `<g-if>`.

A comment that does not reach the page: `{/* note */}`. A literal brace: `{"{"}`.

## Attributes

```
<a href={url} class={active ? "link active" : "link"} title="plain text">…</a>
<input disabled={!ready} value={query} {...rest}>
<img {src} alt="">
```

| Value | Result |
| --- | --- |
| `null`, `undefined` | no attribute |
| `false` | no attribute; `"false"` for `aria-*`, `data-*`, `draggable`, `spellcheck`, `contenteditable` |
| `true` | the bare name for `disabled`, `hidden`, `checked` and the like; `"true"` otherwise |
| anything else | the value, escaped |

Braces inside quotes are text: `title="{x}"` shows `{x}`. For a string with a value in it write ``class={`card ${kind}`}``. There is no `class:list`: build the string, for example `class={["card", big && "big"].filter(Boolean).join(" ")}`.

Addresses in `href`, `src`, `action` and the like are checked: `javascript:` and other schemes that run code are replaced.

## Conditions

```
<g-if cond={stock > 3}>In stock</g-if>
<g-else-if cond={stock > 0}>Only {stock} left</g-else-if>
<g-else>On order</g-else>
```

The branches stand next to each other with nothing but white space between them. `cond` is any expression; it is taken the way `if` takes it, so `cond={user}` and `cond={items.length}` work.

## Loops

```
<g-for each={products} as="product" index="i">
  <li>{i + 1}. {product.title}</li>
</g-for>
```

- `each` takes anything iterable; `null` and `undefined` are an empty list.
- `as` names one item. It may take the item apart: `as="{ id, title }"`, `as="[key, value]"`.
- `index` is optional and counts from 0.
- There is no `key`.

## Ready-made HTML

```
<article><g-html value={post.html} /></article>
```

The value goes to the page unescaped. Use it for HTML you trust: rendered Markdown, content from a CMS.

## Components

A component is a `.shugo` file. Import it in the script and use it as a tag that starts with a capital letter.

```
<!-- components/Card.shugo -->
---
interface Props {
  title: string
  featured?: boolean
}
const { title, featured = false } = Shugo.props as Props
---
<section class={featured ? "card featured" : "card"}>
  <h2>{title}</h2>
  <g-slot>shown when nothing was put inside</g-slot>
  <g-if cond={Shugo.slots.has("footer")}>
    <footer><g-slot name="footer" /></footer>
  </g-if>
</section>
```

```
<Card title="Jug" featured>
  <p>goes to the default slot</p>
  <small slot="footer">goes to the slot "footer"</small>
</Card>
```

- Attributes are the props: `title="text"` is a string, `featured` is `true`, `count={3}` is a value, `{...props}` gives every field of an object.
- `slot="name"` works on a tag that stands right inside the component tag, not inside `<g-if>` or `<g-for>`. To pass several nodes without a tag of their own, wrap them in `<g-fragment slot="name">`.
- Several tags may go to one slot; they are shown in their order. A tag without content can carry `slot` too: `<meta slot="head" …>`.
- What stands inside `<g-slot>…</g-slot>` is shown only when nothing at all was put into the slot.
- A component may show itself: `<Shugo.self node={child} />`.
- A component may be picked at run time: `const View = compact ? List : Grid`, then `<View />`.

A component is one of two kinds, by what its script does:

- **It shows what it is given.** The script works on `Shugo.props` alone: no `await`, no module that reads files or the network. Give it anything, objects too. This is what most components are.
- **It loads data.** The script asks the data database (`Shugo.query`), has `await`, or imports a module that reads files or the network. Such a component takes only plain values — text, numbers, `true`, `false`, `null` — or nothing, and it cannot read `Shugo.url` or `Shugo.params`. It is computed once for the whole site for every set of values, however many pages show it: `<ProductCard id={product.id} />`, `<Header />`.

Giving an object to a component that loads data is an error that says so.

### `Shugo`

| | |
| --- | --- |
| `Shugo.props` | the props of the component; for a page, the `props` from `getStaticPaths` |
| `Shugo.params` | the parameters of the address of the page |
| `Shugo.url` | the address of the page, a `URL` |
| `Shugo.slots.has("name")` | whether something was put into the slot |
| `Shugo.self` | the component itself |
| `Shugo.query("table")` | a question to the data database, in a script: see [Data](#data) |
| `Shugo.paginate(query, { pageSize })` | one page of a long list, in the script of a page |

## Layouts

`pages/+layout.shugo` wraps every page in its folder and below. A layout in a deeper folder goes inside the outer one.

```
<!-- pages/+layout.shugo -->
---
import Header from "../components/Header.shugo"
import "../styles/main.css"
---
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <g-slot name="head"><title>My site</title></g-slot>
  </head>
  <body>
    <Header />
    <main><g-slot /></main>
  </body>
</html>
```

- `<g-slot />` is where the page goes. A layout must have it.
- A page gives markup to its layout with `slot="name"` on a tag at the top level of the page: `<title slot="head">About</title>`. The nearest layout that has `<g-slot name="head">` shows it. Giving markup for a slot that no layout around the page has is an error.
- Pages do not import or call layouts.
- The script of a layout runs for every page it wraps, unless neither the layout nor any component it imports reads `Shugo.url` or `Shugo.params`: then it runs once. A layout whose header highlights the current page runs for every page, so do not load data in its script; put the loading into a module in `lib/`.

## Addresses

| File | Address |
| --- | --- |
| `pages/index.shugo` | `/` |
| `pages/about.shugo` | `/about` |
| `pages/blog/[slug].shugo` | `/blog/…` |
| `pages/docs/[...path].shugo` | `/docs` and everything below |
| `pages/(shop)/cart.shugo` | `/cart` — a folder in parentheses is not a part of the address |
| `pages/_parts/x.shugo` | not a page |
| `pages/404.shugo` | shown for addresses that have no page |

Names that start with `+` are kept for the framework; only `+layout.shugo` exists. Addresses have no slash at the end: the server sends `/about/` on to `/about`.

When two files could give one address, the one that names more of it wins: `blog/new.shugo` over `blog/[slug].shugo` over `blog/[...path].shugo`. So `blog/[...page].shugo` and `blog/posts/[slug].shugo` live side by side.

## Pages from data

A file with a parameter in its name must say which pages there are. `getStaticPaths` gets `query` to ask the data database:

```
<!-- pages/products/[slug].shugo -->
---
export function getStaticPaths({ query }) {
  return query("products").select("slug").all().map(({ slug }) => ({ params: { slug } }))
}

const product = Shugo.query("products").where({ slug: Shugo.params.slug }).first()
---
<h1>{product.title}</h1>
```

`getStaticPaths` sees the imports and nothing else of the script. It runs once; the rest of the script runs for every item it returned, with that item's `params` in `Shugo.params`. The parameters are the ones in the name of the file, as text.

**Return `params` alone and let every page ask for its own data.** Then one product that changes has one page computed. `getStaticPaths` may also hand data to its pages, `{ params, props: { product } }`, read as `Shugo.props`; but then only the route knows what a page shows, and when a row it read changes, all its pages are computed again.

For lists split into pages name the file `[...page].shugo`. `paginate` in `getStaticPaths` takes the number of items and says which pages there are; `Shugo.paginate` in the page takes a query and returns the items of that page:

```
<!-- pages/blog/[...page].shugo gives /blog, /blog/2, /blog/3 … -->
---
export function getStaticPaths({ query, paginate }) {
  return paginate(query("posts").count(), { pageSize: 10 })
}
const page = Shugo.paginate(Shugo.query("posts").order("date", "desc"), { pageSize: 10 })
---
<g-for each={page.data} as="post">…</g-for>
<g-if cond={page.url.prev}><a href={page.url.prev}>Newer</a></g-if>
<g-if cond={page.url.next}><a href={page.url.next}>Older</a></g-if>
```

| | |
| --- | --- |
| `page.data` | the rows of this page |
| `page.currentPage`, `page.lastPage` | numbers from 1 |
| `page.total`, `page.size` | the number of all items, and of items on a page |
| `page.start`, `page.end` | the places of the first and the last item of this page, from 0 |
| `page.url.current`, `.prev`, `.next`, `.first`, `.last` | addresses; `undefined` where there is no such page |

With `[page].shugo` the first page is `/blog/1`. `paginate(count, { pageSize, params })` adds `params` to every page, for a list inside something else: `pages/tags/[tag]/[...page].shugo` with `params: { tag }`. The same `pageSize` goes to both calls.

A site without a data database gives `paginate` the list itself, as in Astro — `paginate(await getPosts(), { pageSize: 10 })` — and reads the page from `Shugo.props.page`.

## Data

The data of a site lives in one SQLite database, named in `shugo.json`: `{ "data": "data/site.db" }`. Whatever brings data to the site — an import script, a sync with a CMS, a person with a SQL console — writes to that database. Scripts read it with `Shugo.query`.

That is what makes updates exact and quick. Shugo notes for every page and every shared component which rows it read and which columns of them were read, and the database keeps a journal of the rows that change. `shugo update` computes what read a changed row in a column it read. Nothing has to be marked, declared or invalidated, and a query need not name its columns.

### Rules for the database

- **Every table has a primary key of one column.** By it Shugo tells the rows of a table apart. A table of pairs gets a key of its own: `id INTEGER PRIMARY KEY`.
- Shugo adds a table `_shugo_journal`, a table `_shugo_meta` and triggers named `_shugo_…` to the database and switches it to WAL mode. Leave them alone. Write to the tables with anything, in any way; `INSERT OR REPLACE` and upserts are fine.
- After a table is added, removed or altered, the next update computes everything once.
- Views are not queried: Shugo cannot tell which rows a view is made of. Query the tables.
- **Data read from anywhere else** — a file, `fetch`, a module that holds state — is not tracked: `shugo update` does not see it change. `shugo build` computes everything and reads it again, and `shugo update -check` names the pages that depend on such data.

### Queries

```ts
Shugo.query("products")                       // a table
  .where({ category: "c9", available: true }) // equal values
  .where("price", "<", 1000)                  // =  !=  <  <=  >  >=  like  in
  .order("position")                          // .order("price", "desc")
  .limit(24).offset(48)
  .all()                                      // the rows
```

| | |
| --- | --- |
| `.all()` | the rows |
| `.first()` | the first row, or `null` |
| `.get(key)` | the row with this primary key, or `null` |
| `.count()` | how many rows the `where` finds; order, limit and offset do not matter |
| `.select("title", "price")` | take only these columns, and the key. For speed alone: see below |
| `.join("categories", "category")` | puts into every row the row of `categories` whose key is the row's `category`, as `row.categories`, or `null`. Options: `{ select: ["title"], as: "group", on: "slug" }`; `on` must be the key or a unique column of the other table. For many rows, make a second query |

- Every step returns a new query, so a query can be kept and asked in several ways.
- **A page depends on the columns that are read, not on the columns that are taken.** A row notes what is read of it: by the script (`const cost = item.price * 2`) and by the markup (`{item.title}`). A change of any other column of the row does not reach the page. So `select` is not needed to keep updates small; use it where a query returns many rows of which little is read, to fetch less: the list of the addresses of a hundred thousand pages.
- **Taking a row as a whole reads all of it**: `{ ...row }`, `JSON.stringify(row)`, `Object.keys(row)`, handing the row to a function in the markup, or to a component that only shows what it is given. Read the members you need in the script instead — `const cost = finalPrice(item)`, then `{price(cost)}` — or pass the id to a component that asks for the row itself.
- **Compare with values of the column's kind**: a text for a text column, a number for a number column. `where({ id: 5 })` on a text column is an error that says what to write.
- `where({ column: null })` finds rows without a value.
- Rows come in the order the database finds them; say `order()` where the order matters, and add an index for it on big tables: `CREATE INDEX … ON products (category, position)` for `where({ category }).order("position")`.
- A query runs when it is asked, synchronously; there is nothing to `await`.

### Where to ask

- **In the script of a page or a layout**, for what it shows.
- **In a component called with an id**, for what many pages show: `<ProductCard id={row.id} />` with `Shugo.query("products").get(id)` inside. Such a component is computed once for every id and kept on its own, so when the product changes its card is computed, and none of the pages that show it.
- **In a component without props**, for what every page shows: `<Header />` reads the menu itself.
- **Not in the markup**, and not in a component that was given an object: those only show. A query in `{…}` is an error that says so.
- Not in a module of `lib/` when it is imported: a query belongs to the page or component that asks, so only `Shugo.query` and the `query` of `getStaticPaths` can ask. A helper takes the query function or the rows as an argument.
- Throw an error when a row that must be there is not: `if (!product) throw new Error(…)`.

### Without a database

A site may also load its data with ordinary TypeScript, from anywhere: leave `data` out of `shugo.json`.

```ts
// lib/posts.ts
import { readFileSync } from "node:fs"

export interface Post { slug: string; title: string; body: string }
export const posts: Post[] = JSON.parse(readFileSync("data/posts.json", "utf8"))
```

Then nothing tells Shugo what has changed, so on every update all scripts run: that loads the data. Shugo compares, for every page and shared block, exactly what its markup reads of the script, and computes the markup only where that has changed. The time of an update grows with the site. `getStaticPaths` hands every page its part through `props`; a request in the script of a page with a thousand pages is a thousand requests on every update, so load once in a module or in `getStaticPaths`.

### What the markup may do

The script loads; the markup shows. Three things follow, with or without a database, and the build says so when one of them is broken:

1. **The markup works on data.** It gets a copy of what it reads: text, numbers, `true` and `false`, lists, plain objects, dates, `Map`, `Set`. A function or an instance of a class in a variable that the markup reads is an error; reading a member of such an instance, `{money.amount}`, is fine. A function that the script declares is not seen by the markup: import it from a module.
2. **The markup calls only what loads nothing.** A function for the markup lives in a module that reads no file and asks no network, also not at import. Keep `lib/shop.ts` (data) apart from `lib/format.ts` (helpers). The same holds for what a component that only shows what it is given imports.
3. **No clock and no chance in the markup.** `new Date()`, `Date.now()` and `Math.random()` belong to a script: `const year = new Date().getFullYear()`, then `{year}`.

To keep updates small:

- **Read members in the markup, not whole objects.** `{item.title}` reads one value. `{format(item)}`, `{item.tags.join(", ")}` and `{items[0]}` read the whole of `item`, `item.tags` and `items`. Where a function needs the whole object, call it in the script.
- **Pass an id, not an object**, to a component that many pages show.
- Create `Intl.NumberFormat`, `Intl.DateTimeFormat` and the like once in a module, not on every call: they are slow to make.

One thing Shugo cannot see: a package from `node_modules` that reads a file or a database by itself, called from the markup or from a component that only shows what it is given. Such a page stays as it is when that file changes. `shugo update -check` finds it; the cure is to do that reading in a script.

## Styles and scripts

- `import "../styles/main.css"` in a script adds a style sheet to the site. All style sheets and all `<style>` tags become one file, linked from every page.
- `<style>` applies to the whole site, not only to its component. `<style is:inline>` stays in the page as written.
- Tailwind works when it is installed in the project (`npm install tailwindcss @tailwindcss/cli`) and a style sheet has `@import "tailwindcss";`.
- `<script>` goes to the page as written; braces inside `<script>` and `<style>` are text. Pass values through attributes: `<div id="map" data-places={JSON.stringify(places)}>`.
- A component with a `<script>` shown ten times gives ten copies of the script. Keep such a script in the page or in the layout.

## Coming from Astro

| Astro | Shugo |
| --- | --- |
| `Astro.props`, `Astro.params`, `Astro.url` | `Shugo.props`, `Shugo.params`, `Shugo.url` |
| `{items.map(item => <li>{item}</li>)}` | `<g-for each={items} as="item"><li>{item}</li></g-for>` |
| `{cond && <b>text</b>}`, `{cond ? <a/> : <b/>}` | `<g-if cond={cond}>…</g-if>` with `<g-else>` |
| `<slot />`, `<slot name="x" />` | `<g-slot />`, `<g-slot name="x" />` |
| `<Fragment slot="x">` | `<g-fragment slot="x">` |
| `set:html={html}` | `<g-html value={html} />` |
| `class:list={[…]}` | a string built in an expression |
| a helper declared in the frontmatter and called in the markup | import it from a module in `lib/` that loads nothing, or call it in the script |
| a component that fetches and takes an object | pass an id and let it fetch, or fetch in the page and pass the data |
| `<Layout title="…">` around the page | `pages/+layout.shugo`, applied by folder; `<title slot="head">` in the page |
| `client:load`, islands | not there: pages are static; write a `<script>` |
| scoped `<style>` | `<style>` is global |
| `src/pages`, `src/components` | `pages`, `components` at the root |
| `.astro` | `.shugo` |

Every tag must be closed (`<br>`, `<img>` and other void elements aside; `<div />` is an empty `div`).
