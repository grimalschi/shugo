# Open questions

The questions are numbered: answer the numbers. Where there is advice, yes - take it. Decisions taken without confirmation are collected in [Decisions awaiting approval](decisions.md); everyone there says what happened to him after the answers of 21 November 2026 year.

## Answers to the author's questions and what to solve for them

### 1. `props` in `getStaticPaths`

Two ways. The first one is the route gives only addresses, the page takes data itself:

```svelte
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

The price of the product has changed - one page is calculated: she read `price` of her line. The route has read only `slug` and does not calculate anything.

The second route gives the data:

```svelte
<script module>
  export function getStaticPaths({ query }) {
    return query("products").all().map((product) => ({ params: { slug: product.slug }, props: { product } }))
  }
</script>

<script>
  let { product } = $props()
</script>

<h1>{product.title}</h1>
```

The page receives a line from the route hands. You can not calculate one page separately: its data is only in the list, so after changing any read route of the line, the route reads all 100,000 products again and calculates all its pages. You can make this path accuratethe lines passed to the page would write down what you read on itbut the list would still have to be read in its entirety to get a single page of `props`.

**The Council:** remove `props` and `paginate(list)`: the route gives only addresses. In SvelteKit also: `entries` returns only parameters, the page receives data itself.

### 2. Which component is common

Yes, the component without dependencies is stored once. A component that does not read the page address and is called without parameters or with simple ones strings, numbers, `true`, `false`, `null` is calculated once and stored by one record under the name “file + parameters”: `src/components/ProductCard.svelte{"id":"p5"}`. Pages refer to the entry number. `<Header />` without parameters is one entry on 100,000 pages. Snippets transmitted inside do not interfere with it: therefore, the layout, to which each page passes itself as `children`, is stored once.

**The Council:** approve [3](decisions.md#3-which-blocks-are-shared) and warn when one component has too many entries (`<Price value={n} />`).

### 3. Title instead of "pattern program"

This is a markup of all templates with places where values stand. Options:

- **frame** — “page frame”, “frame templates”; in the code `skeleton`;
- **stencil** — markup with slots for values; in the code `stencil`;
- **workpiece** is in the code `blank`.

**The Council:** "framework".

### 4. Object `Shugo`: are there any alternatives

Now it has `query`, `params`, `url`, `paginate`. Options:

| Option | Recording | For | Against |
| --- | --- | --- | --- |
| Object as now | `Shugo.query("products")`, `Shugo.params.slug` | one name for everything, no imports | a global name that is not in Svelte |
| Import from `shugo` | `import { query, page } from "shugo"`, `page.params.slug` | clearly like everything else; the editor knows the types | one more line in each file |
| As in SvelteKit | `import { page } from "$app/state"` for address and parameters | agents know the record | data in SvelteKit come from `load`, not request; half of API is still its own |

**The Council:** imports from `shugo`: `query` and `paginate` - features, `page.params` and `page.url` - both `page` in SvelteKit. Explicit imports are better combined with no magic solution.

### 5. Layouts in SvelteKit

V SvelteKit `+layout.svelte` in the folder `src/routes/…` wraps all pages of this folder and nested; nested layout stands inside the external; page layout shows `{@render children()}`. `<html>`, `<head>` and `<body>` Not in the script, but in `src/app.html` with places `%sveltekit.head%` and `%sveltekit.body%`. Chain reset `+page@.svelte` and `+layout@(group).svelte`. The data gives the layout `+layout.ts`B. Shugo It is the same, except `app.html`The Root Layout itself `<html>`.

**Question:** to start `src/app.html`, as in SvelteKit. **The Council:** yet — this is another special file, and the layout with `<html>` is understandable and so.

### 6. Routes in SvelteKit

| | SvelteKit | Shugo |
| --- | --- | --- |
| Page | folder to address, file `+page.svelte`: `src/routes/blog/[slug]/+page.svelte` | file to the address: `src/pages/blog/[slug].svelte` |
| List of pages | `entries` in `+page.ts` | `getStaticPaths` in `<script module>` |
| Not Page | any file without `+` in the route folder | name with `_` |
| 404 | `+error.svelte` | `404.svelte` |
| Parameters | `[slug]`, `[...rest]`, `[[optional]]`, `[x=matcher]`, `(group)` | `[slug]`, `[...rest]`, `(group)` |
| Files not in HTML | `+server.ts` | none |

**Question:** Go to folders with `+page`. **The Council:** No. The template is already transferred to SvelteKit without editing; the file is shorter, and the agent is less likely to be wrong when the file name is the address.

### 7. Pages not in HTML

Yes, in SvelteKit `sitemap.xml`, `robots.txt` and RSS writes a file on TypeScript, not a template: `src/routes/sitemap.xml/+server.ts` with `export const prerender = true` and the function of `GET` that returns text.

**The Council:** also: `src/pages/sitemap.xml.ts` exports `GET({ query })` and returns text; Shugo tracks what it has read from the database as from the page, and writes the `sitemap.xml` file.

### 8. Same address from two list items

SvelteKit simply skips this address: each address is written once, without error and without warning. In Shugo, an assembly error is encountered.

**The Council:** if `props` remove (1 question), do as SvelteKit: two identical elements give one page. While `props` has — leave a mistake: two elements can have different data.

### 9. Component with object in parameter

`<Variant variant={row} />` is calculated on each page again. You can remember it from the content of the parameter, but this breaks the accuracy: the name of the common block would be built from all columns of the row, and the page that calls it would depend on all columns. Now `Variant` reads `title` and `stock`, and changing the shelf of the option page does not touch; with memorization on the content would touch.

**The Council:** leave as is; who wants a common block - transfers the key like a product card.

### 10. Data from layout to page

In SvelteKit, the layout has `+layout.ts` with the function of `load`: what it returned gets both the layout and all the pages below it. For example, a list of categories is loaded once in the layout, and pages use it. There is no separate mechanism in Shugo: the page itself asks for the base (`Shugo.query` is responsible for microseconds on the index) or calls a function from `src/lib/`, and the common one for all pages shows a common component.

**The Council:** mechanism is not needed.

### 11. Menu in the layout: why pages are calculated again

A layout that does not read the address is a common block: it is calculated once. The menu has changed - one record is calculated, and the pages are not touched; HTML they have another, because the server collects it from records.

But the menu often highlights the current item: `class:active={Shugo.url.pathname === item.href}`. This layout reads the address, it is different on each page and is no longer general: it is calculated inside each page. The menu has changed - all 100,000 pages are calculated.

How to avoid: do the backlight in the browser - by `location.pathname` or CSS selector by `aria-current`, which puts a small script - and keep the menu in a common component.

**The Council:** warn when assembling about a layout that reads the address and asks the base itself.

### 12. Compression of answers, `ETag`, 304

This is about `shugo serve` when he himself gives the site to visitors.

- **Compression.** The browser writes in the request what compression it understands (gzip, brotli). The server can give HTML compressed: the page in 5–10 is smaller, loads faster.
- **`ETag` and 304.** The server puts a version label on the page. A browser that already has a page asks "has it changed since version X?". If not, the server responds with a short "304 Not Modified" without the page itself.

The storage has versions, so the label is cheap. If a CDN is in front of the server or the site is laid out with files for hosting, they do both themselves.

**The Council:** do in `shugo serve`: compression and `ETag` on the recording version.

### 13. Time and random numbers in the page code

Yes, `Date.now()` itself does not calculate anything again: a page is calculated only when the read it in the database has changed. But when it is calculated for another reason, the time is different and the page is considered changed; and the neighboring pages show the time of their calculation. Recorded in [Data Download](data-loading.md#what-is-worth-knowing).

**The Council:** approve: time is from data.

### 14. Installation like Hugo

Hugo is also a single file on Go. It is distributed as follows: ready files for all systems in releases on GitHub, Homebrew, Snap, Winget, Scoop, distribution packages, Docker image and `go install`. Through npm Hugo put other people's wrapping packages. esbuild is distributed by the npm package, which puts the finished file under the system.

**The Council:** releases on GitHub, npm package like esbuild — agents work in projects with `package.json` — and Homebrew.

### 15. Optional parameter `[[lang]]`

Details are in [proposal for internationalization](../docs/proposals/i18n.md). In short: `src/pages/[[lang]]/about.svelte` gives both `/about` and `/en/about`; without parameter, the main language.

## New questions after switching to Svelte

16. **Types for the editor.** Extensions `.svelte` and `.island.svelte` are selected. How to describe `Shugo`, `shugo:catalog` and `request:` for review in editor.
17. **`lang="ts"`.** Svelte understands TypeScript only in `<script lang="ts">`, Shugo — any. **The Council:** require `lang="ts"`, like Svelte, with a clear error: then the file is transferred to Svelte without editing.
18. **Accessibility alerts.** U Svelte There are dozens of them (`alt` in pictures, roles, signatures). **The Council:** The agent is useful, and Svelte He's waiting for them.
19. **Rune places.** Svelte forbids `$props()` in markup, second call `$props()` and the like; Shugo does not check yet. **The Council:** check as well as Svelte.
20. **Style class.** Put the component class only on items that selectors select, and throw out extra selectors like Svelte ([32](decisions.md#32-style-class-on-all-component-elements)). **The Council:** yes, but after the rest: the page view does not change from this.
21. **`<svelte:head>` in a conventional component.** In Svelte it works everywhere, in Shugo it works only in page and layout. **The Council:** allow in components that are not common; in general - an error with explanation.
22. **`$lib`.** In SvelteKit `$lib` is `src/lib`, and the code from SvelteKit writes `import X from "$lib/…"`. **The Council:** understand `$lib` along with `@/`.
23. **Data from the islands.** Do I need a general binding mechanism for `request:` besides the published catalog?

## From the proposals

24. **Pictures:** format selects `<picture>` in the page, not the title `Accept`, and is the last extension in the address (`/_img/w800/photos/jug.jpg.webp`): so the server and static hosting work the same way. Image cache is a shared folder on the machine or its own project folder outside of `_build/`. — [Pictures](../docs/proposals/images.md)
25. **Internationalization:** keep lines in project files (changing a line calculates the entire site) or in the database (only pages with this line). — [Internationalization](../docs/proposals/i18n.md)
26. **Live Reload:** show the open page as soon as it is calculated, without waiting for the whole site. [Live Reload](../docs/proposals/live-reload.md)
27. **Data adapter:** is the next big step; schema is any library with Standard Schema; the site updates itself after downloading. — [Data adapter](../docs/proposals/data-adapter.md)

## No answer from the last time.

28. **Breakdown into pages.** `paginate(number)` in `getStaticPaths` says what pages there are, `Shugo.paginate(request)` in the page takes part of it; addresses `/blog`, `/blog/2`. **The Council:** approve. — [Dynamic pages](dynamic-pages.md)
29. **Service symbols in meanings.** Symbols with 0–4 codes inside values are screened when transmitting from JS to Go, rather than cut out. **The Council:** approve. — [25](decisions.md#25-service-symbols-in-values-are-shielded)
30. **Multiple JS processes for first build.** The first build time falls almost in proportion to the number of cores; now 14 s is on 100,000 products. **The Council:** do when the first build will be a bottleneck.
31. **Libraries without a synchronous option.** Page is not waiting; this is created once at the top of the module or makes the adapter. **The Council:** it is enough. — [Data loading](data-loading.md)

The question of spaces between tags is removed: spaces are now processed according to the rules of Svelte.
