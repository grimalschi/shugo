# Writing a Shugo site

Shugo builds static sites. A page is a `.shugo` file: TypeScript between two `---` lines, then markup. If you know Astro, most of this is familiar; the section [Coming from Astro](#coming-from-astro) lists what differs.

## Project layout

```
shugo.json       settings (optional)
pages/           pages and layouts; the path of a file is its address
components/      components
lib/             shared TypeScript: data, helpers
data/            data files, if the site has any
styles/          style sheets
public/          files served as they are: /robots.txt, /img/logo.svg
_build/          written by shugo; never edit, never commit
```

Only `pages/` is required. `@/` in an import is the root of the project.

`shugo.json`, every field optional:

```json
{
  "site": "https://example.com",
  "redirects": { "/old": "/new" },
  "runtime": "bun"
}
```

`site` is where the site is published; `Shugo.url` is built from it (`http://localhost` without it). `runtime` is `bun` or `node`.

## Commands

```
shugo build     compile the project and compute every page
shugo update    compute the pages again after the data changed
shugo dev       build, serve at http://localhost:4321, rebuild on every change
shugo serve     serve the site: every page is built on request from what build and update computed
shugo export    write the site as files into _build/dist
```

Run `shugo build` after every change and read the errors: each names a file, a line and a column.

`shugo update` is for data that changed outside of the project's code: it runs the compiled code again and writes only what differs. If a file of the project changed, it compiles the project first, so it is always safe to run.

`shugo export` writes `/about` as `about/index.html`, the page for missing addresses as `404.html`, and copies `public/` and the style sheet.

## A page

```
---
import Card from "../components/Card.shugo"
import { getProducts, price, type Product } from "@/lib/shop"

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
- The script runs once for every page the file gives. Everything it declares is visible in the markup: values, functions, imports.
- Top-level `await` is fine. Top-level `return` is an error.
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

### `Shugo`

| | |
| --- | --- |
| `Shugo.props` | the props of the component; for a page, the `props` from `getStaticPaths` |
| `Shugo.params` | the parameters of the address of the page |
| `Shugo.url` | the address of the page, a `URL` |
| `Shugo.slots.has("name")` | whether something was put into the slot |
| `Shugo.self` | the component itself |

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

A file with a parameter in its name must say which pages there are:

```
<!-- pages/products/[slug].shugo -->
---
import { getProducts, type Product } from "@/lib/shop"

export async function getStaticPaths() {
  const products = await getProducts()          // one query for all pages
  return products.map((product) => ({
    params: { slug: product.slug },
    props: { product },
  }))
}

const { product } = Shugo.props as { product: Product }
---
<h1>{product.title}</h1>
```

`getStaticPaths` sees the imports and nothing else of the script. It runs once; the rest of the script runs for every item it returned, with that item's `params` in `Shugo.params` and its `props` in `Shugo.props`.

For lists split into pages name the file `[...page].shugo` and use `paginate`, which works as in Astro:

```
<!-- pages/blog/[...page].shugo gives /blog, /blog/2, /blog/3 … -->
---
export async function getStaticPaths({ paginate }: { paginate: Function }) {
  return paginate(await getPosts(), { pageSize: 10 })
}
const { page } = Shugo.props
---
<g-for each={page.data} as="post">…</g-for>
<g-if cond={page.url.prev}><a href={page.url.prev}>Newer</a></g-if>
<g-if cond={page.url.next}><a href={page.url.next}>Older</a></g-if>
```

| | |
| --- | --- |
| `page.data` | the items of this page |
| `page.currentPage`, `page.lastPage` | numbers from 1 |
| `page.total`, `page.size` | the number of all items, and of items on a page |
| `page.start`, `page.end` | the places of the first and the last item of this page, from 0 |
| `page.url.current`, `.prev`, `.next`, `.first`, `.last` | addresses; `undefined` where there is no such page |

With `[page].shugo` the first page is `/blog/1`. `paginate(items, { pageSize, params, props })` adds `params` and `props` to every page, for a list inside something else: `pages/tags/[tag]/[...page].shugo`.

## Data

There is no special data layer: load data with ordinary TypeScript.

Read data files when the code runs, with a path from the root of the project:

```ts
// lib/posts.ts
import { readFileSync } from "node:fs"

export interface Post { slug: string; title: string; body: string }
export const posts: Post[] = JSON.parse(readFileSync("data/posts.json", "utf8"))
```

`import posts from "../data/posts.json"` works too, but then the data is a part of the compiled code: every change of it means compiling the project again.

The props of a page come from `getStaticPaths` and may be anything. The props of a component decide whether it is computed once:

- **For many pages of one kind**, load once in `getStaticPaths` and hand every page its part through `props`.
- **For what every page shows** (a menu, the settings of the site), load in a component without props, such as `<Header />`. A component is computed once for the whole site when neither it nor any component it imports reads `Shugo.url` or `Shugo.params`, and it is called with plain values (strings, numbers, booleans) or without props.
- **Pass an id, not an object**, to a component that many pages show: `<ProductCard id={product.id} />`. Each card is then computed once, and when the product changes only the card changes, not the pages that show it.
- **Put shared loading into a module** in `lib/`: a module runs once per build.
- Create `Intl.NumberFormat`, `Intl.DateTimeFormat` and the like once in a module, not on every call: they are slow to make.
- The script of a component runs every time the component is shown, unless it is computed once as described above. Do not query a database there for a component shown thousands of times.

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
| `<Layout title="…">` around the page | `pages/+layout.shugo`, applied by folder; `<title slot="head">` in the page |
| `client:load`, islands | not there: pages are static; write a `<script>` |
| scoped `<style>` | `<style>` is global |
| `src/pages`, `src/components` | `pages`, `components` at the root |
| `.astro` | `.shugo` |

Every tag must be closed (`<br>`, `<img>` and other void elements aside; `<div />` is an empty `div`).
