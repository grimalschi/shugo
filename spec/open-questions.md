# Open questions

Questions are numbered: answer the numbers. Where there is advice, yes - to accept it. Decisions taken without confirmation are collected in [Decisions awaiting approval](decisions.md); proposals that do not yet exist are in [docs/proposals](../docs/proposals/README.md).

### 1. Which component is common

A component that does not read the page address and is called without parameters or with simple values - rows, numbers, `true`, `false`, `null` - is calculated and stored once: `<Header />` - one entry on 100,000 pages, `<ProductCard id="p5" />` - one per product. The snippets that are passed inside do not interfere with it.

**The Council:** approve [3](decisions.md#3-which-blocks-are-shared) and warn when assembly is needed when one component has too many entries (`<Price value={n} />`).

### 2. Pages not in HTML

In SvelteKit, `sitemap.xml`, `robots.txt` and RSS writes a file on TypeScript, not a template: `src/routes/sitemap.xml/+server.ts` with `prerender = true` and the function of `GET`.

**The Council:** also: `src/pages/sitemap.xml.ts` exports `GET({ query })` and returns text; Shugo tracks what it has read from the database as from the page, and writes the `sitemap.xml` file.

### 3. Development mode: page on request

The entire site in development mode does not need to be calculated: the page is calculated when asked, and stored until the code or lines it read have changed. This is how SvelteKit and Astro work. Price — an error in the page that was not opened, will find only `shugo build`. See also [Live Reload](../docs/proposals/live-reload.md).

**The Council:** do development mode like this.

### 4. `query` and `page` from the module `shugo`

Instead of global `Shugo`, `import { query, paginate, page } from "shugo"`, where `page.url`, `page.params`, `page.route.id` and `page.status` are like `page` in SvelteKit. Details — [proposal](../docs/proposals/shugo-module.md).

**The Council:** yes, and at once, without two entries at the time of transition.

### 5. Page shell: `app.html` or `site.html`

`<html>`, `<head>` and `<body>` in a separate file with places `%shugo.head%` and `%shugo.body%`The root is a common component, such as `+layout.svelte` the City of London SvelteKitand is transferred between them without editing. Detailed — [proposal](../docs/proposals/app-html.md).

**The Council:** `src/app.html`: This agent name knows by SvelteKit and will write himself.

### 6. Image cache

Its project folder outside `_build/`, `.shugo-cache/images/`, with the setting where to transfer it; files are called by the image hash, parameters and compression method; cleaning command. Details - [proposal](../docs/proposals/image-cache.md).

**The Council:** yes.

### 7. Compatible with Svelte

`lang="ts"`, runes locations, availability alerts, `<svelte:head>` in any component, `$lib` is [proposal](../docs/proposals/svelte-compatibility.md). Style class has been postponed. The main question is whether to check files with a real Svelte compiler in `shugo check` or transfer it to checks in Go.

**The Council:** is short — in the Shugo compiler, Svelte checks — its own compiler in `shugo check`.
