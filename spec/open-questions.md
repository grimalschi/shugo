# Open questions

Questions are numbered: answer the numbers. Where there is advice, yes - to accept it. Decisions taken without confirmation are collected in [Decisions awaiting approval](decisions.md); proposals that do not yet exist are in [docs/proposals](../docs/proposals/README.md).

### 1. `props` in `getStaticPaths`

The addresses in both ways are the same - one page per product: `/products/jug`, `/products/cup`, `/products/tray`. The only difference is that the route returns along with the addresses.

The first way is only addresses:

```js
[{ params: { slug: "jug" } }, { params: { slug: "cup" } }, { params: { slug: "tray" } }]
```

The page itself takes its product: `Shugo.query("products").where({ slug: Shugo.params.slug }).first()`. The price of the jug has changed - one page of `/products/jug` is calculated.

The second way is addresses and data:

```js
[
  { params: { slug: "jug" }, props: { product: { id: "p1", slug: "jug", title: "Pitcher", price: 2400, … } } },
  { params: { slug: "cup" }, props: { product: { id: "p2", slug: "cup", title: "Cup", price: 900, … } } },
  …
]
```

The page takes the goods from `$props()`. Its data is only in this list, so to calculate one page, you need the entire list: after changing the price of the jug, the route reads all the goods anew and calculates all its pages.

**The Council:** remove `props` and `paginate(list)`: route gives only addresses. In SvelteKit also: `entries` returns only parameters.

### 2. Which component is common

A component that does not read the page address and is called without parameters or with simple values is calculated and stored once: `<Header />` is one entry on 100,000 pages, `<ProductCard id="p5" />` is one entry per product. Snippets passed inside do not interfere with it.

**The Council:** approve [3](decisions.md#3-which-blocks-are-shared) and warn when assembly is needed when one component has too many entries (`<Price value={n} />`).

### 3. Title instead of "pattern program"

This is a markup of all the patterns with places where values stand.

| Title | Image | In code |
| --- | --- | --- |
| frame | markup without values that are filled | `skeleton` |
| stencil | A list of the things that matter. | `stencil` |
| matrix | printing form, which is cast pages | `matrix` |
| printable form | in the words of the printer | `plate` |
| , treated | they cover every page. | `pattern` |
| workpiece, twat | unfinished page | `blank` |
| drawing | page Plan | `blueprint` |
| form | markup with fields | `form` — confused with `<form>` |

**The Council:** "frame" - short and clear without explanation; the second option is a "stencil".

### 4. Object `Shugo` or import as `page` to SvelteKit

In SvelteKit address and page parameters give `page` of `$app/state` ([documentation](https://svelte.dev/docs/kit/$app-state#page)):

```svelte
<script>
  import { page } from '$app/state';
</script>

<p>Currently at {page.url.pathname}</p>
```

| Field `page` | What is it |
| --- | --- |
| `url` | address of the page, `URL` |
| `params` | route parameters: `{ slug: "jug" }` for `/blog/[slug]` |
| `route.id` | route: `/blog/[slug]` |
| `status` | response Code |
| `error` | error if it is an error page |
| `data` | that returned the functions of `load` pages and layouts |
| `form` | data after sending the form |
| `state` | the state that the page passed when you clicked |

The offer for Shugo is the same name and the same fields where they make sense; `data`, `form` and `state` are not needed, they are not there:

```svelte
<script>
  import { query, page } from "shugo"

  const product = query("products").where({ slug: page.params.slug }).first()
</script>

<p>{page.url.pathname}: {product.title}</p>
```

**The Council:** is like `query` and `paginate` are functions, `page` is like SvelteKit. Explicit import instead of a name that is taken from nowhere; an agent who knows SvelteKit writes `page.params` out of habit.

### 5. Service files: `+layout` as in SvelteKit, or `_layout` as in Fresh

| | SvelteKit | Fresh | Shugo |
| --- | --- | --- | --- |
| Page | folder to the address with `+page.svelte` | file to the address: `routes/about.tsx` | file to the address: `src/pages/about.shugo` |
| Layout | `+layout.svelte` | `_layout.tsx` | `+layout.shugo` |
| `<html>`, `<head>`, `<body>` | `src/app.html` | `_app.tsx` | root Layout |
| A file that is not a page | any without `+` | folder of the form `(_components)` | name with `_` |
| Reset external layouts | `+page@.svelte` | `skipInheritedLayouts` in page or layout settings | none |
| Optional parameter | `[[lang]]` | `[[lang]]` | none |

In Fresh, the layout also wraps the pages of its folder below, and nested layouts are nested in each other ([layouts](https://usefresh.dev/docs/concepts/layouts), [routes](https://usefresh.dev/docs/concepts/file-routing), Fresh 2). On the routes Shugo is closer to Fresh: file to address.

- `_layout` is good because Fresh also has routes: file to address and `_` to service files. But in Shugo `_` already means "not page" - `_parts/card.shugo`, `_draft.shugo` - and it would have to be divided into service names and all the rest.
- `+layout` is a name that everyone who wrote on SvelteKit knows, and the templates we have are Svelte. Risk is a habit agent will write `+page.shugo`; to this build now responds that the page in Shugo is a file named at the address.

**The Council:** to keep `+layout.shugo`. The root layout is still written by `<html>` himself: a separate `app.html` or `_app` is another special file without winning.

### 6. Pages not in HTML

In SvelteKit, `sitemap.xml`, `robots.txt` and RSS writes a file on TypeScript, not a template: `src/routes/sitemap.xml/+server.ts` with `prerender = true` and the function of `GET`.

**The Council:** also: `src/pages/sitemap.xml.ts` exports `GET({ query })` and returns text; Shugo tracks what it has read from the database as from the page, and writes the `sitemap.xml` file.

### 7. One address from two list items

Example: tag pages are collected from posts, and two posts have one tag.

```svelte
<script module>
  export function getStaticPaths({ query }) {
    return query("post_tags").all().map((row) => ({ params: { tag: row.tag } }))
  }
</script>
```

If 1 and 2 have a `glina` tag, `{ params: { tag: "glina" } }` is twice listed, and both elements want a `/tags/glina` address.

- Shugo now: build error "getStaticPaths of /tags/[tag] returned the address /tags/glina twice". The agent must remove the repetitions itself: `[...new Set(tags)]`.
- SvelteKit: No errors, no warnings; the `/tags/glina` page is written once.

While elements can have `props`, two repetitions can have different data, and it is not clear whose data to show - so the error. Without `props`, repetitions are no different: the page still takes the data to the address.

**The Council:** if `props` is removed (1 question), do as SvelteKit: repeat silently give one page.

### 8. The layout that reads the address is a separate entry

Your thought is correct. The layout that the address does not read is already stored separately: one entry for the entire site, pages link to it. It is only bad when the layout reads the address - for example, highlights the current menu item: now it is calculated inside the record of each page, and the change in the menu calculates all the pages again along with their code. The proposal is to store such a layout with your entry on each page: changing the menu calculates only the layout entries, changing the page is only her. See also [Layout of a separate record](../docs/proposals/layout-records.md).

**The Council:** do, starting with the layouts.

### 9. Development mode: page on request

Yes, the entire site in development mode is not necessary to calculate: the page is calculated when asked, and stored until the code or lines it has read have changed. This is how SvelteKit and Astro work. Price — an error in the page that was not opened, will find only `shugo build`. See also [Live Reload](../docs/proposals/live-reload.md).

**The Council:** do development mode like this.

### 10. Pictures: format and cache

Format: only WebP is one address in the usual `<img>`, equally on the server and on the hosting; `<picture>` with AVIF and WebP - pictures on 15–20% are easier, but AVIF is only with libvips; the choice of the header `Accept` - does not work on static hosting. Cache: in `_build/` - disappears with it; its project folder outside `_build/` - is undergoing cleaning; the user's shared folder is common to all projects, but grows without a host. See also [Pictures](../docs/proposals/images.md).

**The Council:** is the default WebP only, `<picture>` with AVIF is for libvips; cache is a custom project folder outside of `_build/` with a setting where to move it.

### 11. Compatible with Svelte

`lang="ts"`, rune locations, accessibility alerts, style class like u Svelte, `<svelte:head>` In the components, `$lib` — [proposal](../docs/proposals/svelte-compatibility.md). The main question in it is whether to check files with a real compiler Svelte the City of London `shugo check` or to carry out its verifications in Go.

**The Council:** is short — in the Shugo compiler, Svelte checks — its own compiler in `shugo check`.

### 12. Islands of Svelte

When to start and how to mark a component that works in the browser.

**The Council:** after data adapter.
