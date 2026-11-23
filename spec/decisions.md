# Decisions pending approval

Here is written what is decided without direct confirmation of the author of the project: where the answer was “don’t know”, “decision” or question before. Every decision has a reason. After approval, the decision is transferred to the “Decision” section of its document, after refusal redone. The numbers don’t change so that the links don’t break; every decision above tells you what happened to it.

Decisions 1–27 taken during the assembly of the first version, 28–35 when switching to syntax Svelte 21 November 2026 years, 36–38 — 22 November 2026 The year when the shell appeared `src/site.html` Files are not HTML, 39 When there is only one search, 40–48 — the same time when the templates began to compile and render themselves Svelte, 49–53 — then, according to the author's answers to questions about documents, 54–55 then, according to the answers about the packages npm and the waiting, 56 — 23 November 2026 The year when Shugo the Slippery Pack npm.

## 1. Data loading and expression are counted together, Go compares the result

**Rejected by the author of the project 20 November 2026 year.** Data change analysis is mandatory: expressions are calculated only for the changed one. The author’s doubt was only about who was comparing the dataGo or TSrather than whether a comparison was needed. The argument for “saving almost nothing” is not complete below: it takes into account expressions, but not the code of components and templates, which can be expensive. In return, [proposal](../docs/change-tracking.md). It is described in the first version.

**What's done.** Stages three, not four: compilation, computation, runtime. At the computation stage, one JS process executes the data code of the pages and components and immediately counts the expression of the templates. Go gets the values ready, compares them with the saved ones and records only the differences.

**Why not.**
- Astro-style data code makes variables visible to the template. Among them are functions, dates, `Map`. If there is Go between loading and expressions, all this needs to be serialized, and functions cannot be serialized.
- There was no benefit from the split. Data is born in JS as objects. To give them to Go, all the data would have to be turned into JSON, and then the changed parts would be disassembled into JS back. The values of the expressions in 2 are times more compact than the input data, and they still need to be transmitted.
- The very calculation of expressions in the prototype took 2–5% step time. There is almost nothing to save it by skipping unchanged blocks.
- The compiler has become much easier: it no longer needs to parse the expressions and output which fields each block reads.

**What's lost.** Each update counts the expressions of all pages, not just those that have changed. The update time depends on the size of the site, not the size of the change: on the stand it is 0.4 s on 10,000 pages and 3.2–3.7 s on 100,000. Exact figures are in [README booth](../examples/catalog/README.md).

**About WASM for comparison.** It is not needed. Comparison is a hash of each entry on the side of Go. On 100,000 pages it takes fractions of a second and goes at the same time as the work of JS.

**What's left.** Record only changed. The header is stored once. After changing the price of one product, one page and one shared card are changed. Rantime collects pages from the saved and does not execute the project code.

## 2. Values are stored by content and shared blocks by name

**Changed by the decision of the author of the project 21 November 2026 year:** value tapes lie inside page records and shared blocks, rather than a separate table under its hash, [Storage](../docs/storage.md#decisions). Without a copy of the site in memory, the server reads a page from SQLite for each request, and time determines the number of searches: with tapes inside they are half as large ([](../docs/runtime.md#sqlite-read-performance)). Shared blocks are still stored by name.

*With 22 November 2026 year tape values are not: templates render Svelte, and the records are ready HTML with holes — [Storage](../docs/storage.md).*

**What was done.** Each block had a value tape. It was stored under the hash of itself, so the same tapes were lying in storage once. Page entry - a list of its blocks with links to the tapes. The shared block is stored as a separate entry under a name from the component file and parameters, and the page links to it by that name.

**Why do we have common blocks by name?** This is the answer to the question about the header. In the prototype, the page stored the hash of each of its blocks, including the cap. The header has changed its hash has changed the record of each of the 100,000 pages needs to be rewritten. In addition, these caps were in the data of each page, and all pages had to be re-read. Now the page stores the block number, not its content. The header has changed one recording has been rewritten, the pages have not been touched.

**“The Significance of the Entry Hash”** from the prototype meant: the block result was stored under the hash of the data that the block reads. This is not: the hash is taken from the result.

## 3. Which blocks are shared

**Approved by the author of the project 22 November 2026 year** along with a warning when the common component shares almost nothing is [Components](components.md#decisions). Warning thresholds are selected without the author; the author left them as is and postponed the statement (22 November 2026 years). They are below, in "Warning".

**What's done.** The component is calculated and stored once if both conditions are met:
- neither he nor the modules he imports take `page` or `paginate` from the `shugo` module (before 22 November 2026 years — did not read `Shugo.url` and `Shugo.params`);
- it is called with simple parameters: lines, numbers, `true`, `false`, `null` or without parameters.

The name of the shared block is composed of the file path and parameters: `components/ProductCard.svelte{"id":"p7"}`.

*C 22 November 2026 years when templates render Svelte, conditions are greater: component does not take context SvelteIt doesn't call `$props.id()`It doesn’t show a snippet inside. `<select>` and does not receive `bind:` ([42](#42-shared-components-do-not-use-context-or-propsid)). Snippets without parameters, written inside its tag, are common do not interfere ([43](#43-render-shared-block-snippets-separately)). The path in the name is from the root of the project, and the names of snippets are after `|`: `src/components/ProductCard.svelte{"id":"p7"|children}`.*

**Why not.** The agent does not need to mark anything. The header without parameters is general by itself. The card to which the product number was transferred is common for all pages with this product; the card to which the object of the goods was transferred is considered on each page anew.

**Price.** Components like `<Price value={n} />` It works, but storage is growing.

**Warning.** When the entire site is computed, the build counts for each component of its shared blocks and the places where they are shown. If 100 blocks or more, and on average each is shown in less than one and a half places, a warning is printed with the advice to transfer an object to the component then it is calculated as part of the page. Layouts are not weighted: A layout that reads a page is stored for each page on purpose. With a regular data update, nothing counts: counting goes on all common blocks, about 14 ms on 10,000.

**Why such thresholds?** Less 100 blocks are almost worthless, and on the developer's small data, the warning would be noise. One and a half places on average means that almost every set of parameters is different. In the store from the example, the product card is shown on average in 5 places, the icon is in 1375, so there is no warning there.

## 4. The key in `<g-for>` is not entered

**Approved 21 November 2026 year:** is not required key, cycle is `{#each}` from Svelte. Key in parentheses, `{#each items as item (item.id)}`, is accepted as in Svelte, and on static page does not affect anything.

**Why is the key not needed?** Key is needed when you compare the input data: the block recognizes the "own" element. Shugo compares the results. If you remove the fifth product from the list, the tapes of the others do not change and are not rewritten - without keys.

*22 November 2026 year tapes are not: in the page entry is HTML, and the page with the list is rewritten as a whole. Common blocks of other products are not calculated and not rewritten - still without keys.*

## 5. The framework of the entire project is one file, the rantime is common

**The approach is approved by 21 November 2026 year:** frame instead of Go-code project. The name "frame" instead of "pattern program" approved 22 November 2026 year; in the code is a package `skeleton`.

**What's done.** The compiler writes `_build/site.json`: templates of all files and routes. When calculating this frame is put in the storage next to the values. Rantime only needs a storage file and static files. Go code is not generated.

**Why not.** Answer to the question "page code or the whole project": the entire project, one file that the rantime reads once. The framework and values change with one transaction, so the server never sees new patterns with old values. The framework is data, not code, so anything that runs Go is safe in itself: the worst thing that can happen is the wrong HTML.

**That's postponed.** Go-code generation as optional acceleration. The page assembly now takes microseconds, the win would be 12–15% from them.

*With 22 November 2026 year templates in the frame no: HTML writes Svelte when calculating, and records keep it ready. In the frame remained files, routes, `src/site.html` shell, a link to island styles and programs, 8 format. In storage, it still lies with one transaction with records. Generate Go-code is no longer a thing: the server sews ready pieces of HTML.*

## 6. Components and functions are imported in data code, as in Astro

**Replaced by 21 November 2026 year:** files `.svelte` components Svelte 5The components are imported into `<script>`As in Svelte: `import Card from "../components/Card.svelte"`. The reasons for the previous decision remained in force: the path is visible in imports, the name is chosen by the one who imports, there are no autoimports (Nitro 3 They refused, in Nuxt 5 They are off the server.

## 7. Object `Shugo`

**Replaced by the decision of the author of the project 22 November 2026 year:** global object is not, `query`, `paginate` and `page` imported from the module `shugo`, immediately, without a transition time — [Templates](templates.md#file-structure). `Shugo.query` and `Astro.params` give an error with a hint of what to import.

**It was.** C 21 November 2026 the Year in the Facility `Shugo` remained `Shugo.query`, `Shugo.params`, `Shugo.url` and `Shugo.paginate`The parameters of the component were already taken from `$props()`.

## 8. Slots

**Replaced by 21 November 2026 year:** The marking is transmitted to the component by snippets, as in Svelte 5: `children` and `{#snippet name()}…{/snippet}` inside component tag, `{@render …}` in the component. `<g-slot>`, `<g-fragment>`, `<slot>` and `slot="…"` give an error with a hint. Detailed in [Components](components.md#snippets).

## 9. Layouts

**Changed to 21 on November 2026 of the year, pending approval.** Layout is a component of Svelte: the page it shows `{@render children()}`, the title and meta tags the page writes in `<svelte:head>`; this is the author's decision. The rest awaits approval:

- `+layout.svelte` in the folder wraps all pages of this folder and nested; the nested layout turns out to be **inside** external, as in SvelteKit, Next.js and PocketPages. Replaces only Nuxt, where the page has one named layout.
- The root layout was written by `<html>`, `<head>` and `<body>`. Replaced by the decision of the author 22 November 2026: the document is written in the shell `src/site.html`, the root layout is a normal component ([Layouts](layouts.md#site-shell), [36](#36-the-srcsitehtml-shell)).

Reset Chain`+page@.svelte` the City of London SvelteKit) postponed. Resolved 22 November 2026 Year: The official name remains `+layout`, now `+layout.svelte`; data from layout to page, how `load` the City of London `+layout.ts` SvelteKit, no and will not; the layout is stored separately from the pages - one entry for the entire site or, if reading the address, with its entry for each page ([Blocks](../docs/blocks.md#layouts)).

## 10. Routes

**Waiting for approval.** The author asked to compare with SvelteKit — a comparison in [questions](open-questions.md).

**What's done.**

| File | Address |
| --- | --- |
| `src/pages/index.svelte` | `/` |
| `src/pages/about.svelte` | `/about` |
| `src/pages/blog/[slug].svelte` | `/blog/…`, one parameter |
| `src/pages/docs/[...path].svelte` | `/docs` and deeper |
| `src/pages/(shop)/cart.svelte` | `/cart`: folder in parentheses is not included |
| `src/pages/_parts/x.svelte`, `pages/_x.svelte` | no Page |
| `src/pages/404.svelte` | shown for addresses without page |

- Names with `+` are occupied by the framework. Now there is only `+layout.svelte`; any other is an assembly error.
- Part of the path is either text or one parameter. `post-[id]` is an error.
- When two routes give one address, the one that calls it more accurate wins: the text is stronger than the parameter, the parameter is stronger than the remainder of the path. Two files with the same type of address are assembly errors.
- The page has one address, without a slash at the end. `/about/` is redirected to `/about`.
- The redirects are given in `shugo.json`: `"redirects": {"/old": "/new"}`.

**Why not.** This is the common place Astro, SvelteKit, Next.js and Nuxt. There is no consent about slash: SvelteKit and Next.js remove it, Astro and Nuxt accept both options. One address per page is selected to avoid duplicates.

## 11. `getStaticPaths`

**Changed and approved.** Deleted page approved by 21 on November 2026 year. 22 November 2026 is decided: the route gives only addresses, `{ params }`, and `props` and `paginate(list)` are removed; the address that the list returned twice is one page, as in SvelteKit. `getStaticPaths` is in `<script module>` ([29](#29-script-module-and-export)).

**It was.** As in Astro: the function returned a list of `{ params, props }`, and all pages data could be retrieved with one request. But then it was impossible to calculate one page: its data was only in the list, and changing any read route of the line calculated all its pages.

**Like now.** Page takes its data by itself, by `page.params`: a line change calculates the page of this line. `paginate` in `getStaticPaths` receives the number of elements, and in the page cuts the query or array. With the 22 November 2026 year `getStaticPaths` gets nothing by argument: `query` and `paginate` are imported from the `shugo` module.

## 12. What outputs `{expression}` and `attribute={expression}`

**Replaced by 21 November 2026 year:** values are displayed as in Svelte - [Templates](templates.md#text). The previous rules (`false` - nothing, the list - in a row without a separator, attributes as in React) are removed.

## 13. Ready HTML — tag `<g-html>`

**Replaced by 21 November 2026 year:** `{@html …}` as in Svelte.

## 14. Addresses in links are checked

**22 November 2026 Year Cancelled:** addresses are displayed as is, as in Svelte. The statement from 21 November is withdrawn; below - as it was.

**What's done.** In `href`, `src`, `action`, `formaction`, `poster`, `cite`, the value from the expression is replaced by `about:invalid#blocked` if its circuit can execute the code (`javascript:` and the like). Relating addresses are allowed, `http`, `https`, `mailto`, `tel`, `sms`, `ftp` and pictures in `data:`.

**Why not.** The address often comes from data that was entered by visitors. So it was in the prototype.

## 15. Styles

**Approved 21 November 2026 year.**

**What's done.**
- `import "../styles/main.css"` in `<script>` connects the style file. All the connected files and the contents of all `<style>` tags are collected into one file with a hash in the name. A link to it is added before `</head>`.
- If styles have Tailwind directives, the file collects Tailwind from the `node_modules` project.
- `<style>` is only valid for it, as in Svelte: this is the author's decision from 21 on November 2026, in detail - in [Styles](styles.md).

## 16. `<script>` remains as written

**Approved 21 November 2026 year.** The basis for the code in the browser is Svelte.

**What's done.** The contents of `<script>` inside the markup are included in the page unchanged. The expressions in the parentheses inside it are not calculated. With the transition to Svelte, the top `<script>` file is component code, and the browser script is written inside the element or in `<svelte:head>`, as in Svelte.

**That's postponed.** Scripting Assembly: TypeScript, Imports, One File to Site. This is a separate document of ["Scripts on the Page"](scripts.md).

## 17. Strict analysis of markup

**Replaced by 21 November 2026 year:** The label is as well defined as Svelte: Unclosed or confused tag - error with place; `<li>`, `<p>`, `<td>` and others who HTML allows you not to close, close themselves, with a warning; the element that the browser would rearrange is an error or, through the block, a warning; comments are removed, spaces are compressed by the rules Svelte. Details — in [Templates](templates.md#markup-and-gaps).

*With 22 November 2026 year markup parses and checks the Svelte itself. `<div><span>text</div>` is not an error for it, but a warning `element_implicitly_closed`: `</div>` closes and `<span>`. Error is a closing tag that has nothing to close, and the file is not closed.*

## 18. Project structure

**Replaced by 21 November 2026 year:** site code is in `src/` — the author's decision. Details that are waiting for approval — in [35](#35-folder-src) and in [Project Structure](project-structure.md).

## 19. Commands

**Waiting for approval.** The author asked how `dev` differs from other commands, the answer is in [Command Line](cli.md#how-teams-are-different).

| Command | What does |
| --- | --- |
| `shugo build` | compilation and calculation of all pages |
| `shugo update` | re-write, write only changed |
| `shugo serve` | server: pages are collected on the fly from the storage |
| `shugo export` | site to files; only changed ones are written |
| `shugo dev` | assembly, server, reassembly when changing files, updating the page in the browser |

Report `build`, `update` and `export` can be obtained in JSON (`-json`). JS-runtime is selected as follows: `-runtime`, then `"runtime"` the City of London `shugo.json`, then BunIf it is installed, otherwise Node.

## 20. Storage

**21 November 2026 Year Approved**, along with a strict failed update: the update that fell is considered completely failed. The driver of SQLite remains on the pure Go.

**What's done.** One file SQLite: records pages and shared blocks with ribbons inside, who is what common block shows who read what from the project database, missing pages, service data with the project framework - [Storage](../docs/storage.md#tables). The server reads the page entirely from one state. Update is one transaction.

*With 22 November 2026 year records lies HTML with holes, not value tapes: render templates Svelte. Table `page` is with rovid, storage layout is `8` ([45](#45-storage-page-table--with-rowid)).*

The driver is `modernc.org/sqlite`, without cgo. It is hidden behind the `store` package, you can replace it in one place.

**Failed update.** If the project code has fallen, the transaction is canceled, the storage remains the same, and the data log is not cleared: the next update will meet the same changes.

## 21. Export to files

**21 November 2026 Year Approved**, with file names like SvelteKit: `/about` is `about.html`.

**What's done.** Page `/about` is written in `about.html`, as in SvelteKit, page 404 is in `404.html`. File with previous content is not overwritten.

Export follows the previous one: in the folder, it leaves a list of which storage and which version the pages are written from and which files are copied. The next export from the same repository builds only pages that have changed the record or the record of the shared block they show, deletes missing page files and copies the changed files. Without a list, exports build everything and remove nothing. With a list from another repository or from the version before the full reassembly - builds everything and removes page files that the site does not have, without touching other files and folders whose name starts with a point. See also [Rantime](../docs/runtime.md#export).

**Why not.** Export, which after a change in one price collects all 104,202 pages, cost 7 s; by changes — 8 ms. To update nothing noted for export, the version is written along with the record, and the pages that show the changed common block, export finds itself: it builds them anyway. The export folder is considered to be an export folder: a file that has been manually corrected remains until its page changes.

## 22. Message Language

**Approved 21 November 2026 year.**

**Implementation.** The compiler, CLI, and specification use English.

**Why not.** Messages are read primarily by agents and searched for. Habits from Astro (`set:html`, `<slot>`, `Astro.props`, markup inside `{}`) the compiler recognizes and says what to write instead.

*22 November 2026 The author of the project decided: everything that was for Astro has been removed. Tips for his records — `---`, `<Fragment>`, `set:html`, `class:list`, `Astro.props` No more: they are answered. Svelte and JavaScriptlike any other markup and code. Your tips Shugo Stay where the message doesn’t say what to fix: slots Svelte 4The first syntax Shugo, `runes={false}`, marking inside `{…}`Type imported without `type`, import layout, component from package npm A component that shows itself infinitely.*

## 23. Small

**Waiting for approval**, except for calling himself.

- **The component can call itself.** *Replaced by 21 November 2026 year:* as in Svelte 5 — the component imports itself; `<svelte:self>` works with a warning. An endless call with the same parameters is caught with an understandable error. *With the 22 of November, the 2026 is year and the component that renders on-site ([51](#51-a-component-that-endlessly-shows-itself)).*
- **Module path Go** — `shugo`. Before publication, it must be replaced with the present one.
- **Author of commits** is the name of `itareo` and the owner's mail, specified only in this repository.
- **Page queue.** Pages are calculated one by one, in a row: there is nothing to wait for ([Data loading](data-loading.md#decisions)). The JS process is one.

## 24. Verification on agent

**Re-checking later**, the solution of 21 November 2026 year: we will check more later.

**What's done.** The individual agent was given only [memo](authoring.md) and the `shugo` commandno source, examples, or specificationand the task: a blog with a breakdown into pages, tag pages, nested layout, three components given in JSON, styles, and script.

**What happened.**
- The site from 25 pages was collected on the first try. There were no build errors, HTML is checked for shielding, links and nesting layouts.
- One real trap: the agent imported JSON with data like in Vite and Astro. The imported file becomes part of the code, and `shugo update` has not seen any changes to it. Fixed: `update` now notices himself that the project has changed, and compiles it again.
- The update report named three changed pages, and the export recorded six files: another three pages showed a changed card. Fixed: the report appeared the number of pages on which HTML changed.
- Spaces in the memo: addresses when broken down into pages, `page` fields, priority routes where to keep data files that types are not checked. Added.

**What it does not show.** The agent did not need error messages, so they were not checked on the agent. The task was one, the model one.

## 25. Service symbols in values are shielded

**Waiting for approval.**

**What's done.** JS transmits entries in Go messages in which characters with 0–3 codes serve as separators. The first version silently cut them out of meaning. Now JS shields them inside the value, and with them the 4 symbol, which it shields: the 4 symbol, then the symbol code plus 0x40. Go removes the shielding, and the corrupted shielding stops the upgrade. The attribute name from `{...attributes}` with this symbol is an error, as are other names that cannot be the attribute name.

*With 22 November 2026 year attributes writes Svelte: name from `{...attributes}` with space, quote, `/`, `=` or `>` it silently skips, and the control characters leave in the name. Screening on the way from JS to Go remains: now it is HTML blocks, not values.*

**Why not.** The value should reach the page as the project code gave it. The characters cut out changed the text imperceptibly.

## 26. Project without database

**Approved 21 November 2026 year.**

**What's done.** A project without a database `shugo update` does not calculate anything if the project itself has not changed. If it has changed, the project is compiled anew and everything is calculated. Previously, such an update always calculated everything.

**Why not.** The data of such a project is in its code, and the code changes only with the project. Everything that the page reads from other places is not tracked when working with the base: `shugo build` calculates everything and reads it again.

## 27. Commands

**Waiting for approval.**

- `shugo dev` updates the site in the same way as `shugo update`: after changing the source code compiles and calculates everything, after changing the database - what the changed lines read, after changing other files - nothing. Previously, any changed file calculated the entire site anew.
- The team refuses the option it does not use: `shugo export -check` is an error, not an option that was silently missed. `-runtime` is only taken by commands that execute the project code: `build`, `update`, `dev`.

**Why not.** The agent who transmitted the option is waiting for action from it. Silence would hide the error.

## 28. HTML as in the server Svelte 5.59.1

**Waiting for approval.** Author's solution is to copy the behavior of Svelte templates. How it is done:

- Shugo is fixed on one version of Svelte, 5.59.1. The page matches what `render()` gives this version, bytes per byte, except for hydration tags and style class. Match check benchmarks from the real Svelte: 30 tasks in `test/svelte/`, template tests and random pages — [Shugo and Svelte](../docs/svelte.md).
- Svelte version update new benchmarks and correction of discrepancies.
- Derogations: hydration labels are not written, style class is placed on all elements ([32](#32-style-class-on-all-component-elements)), `<svelte:head>` - only in the page and layout ([30](#30-sveltehead-and-title)), there are no availability warnings, runes are not checked. Derogations are listed in [Templates](templates.md#how-shugo-differs-from-svelte).

*C 22 November 2026 the Year Templates Compiles and Renders Himself Svelte 5.59.1, sewn into Shugo (author's decision): page coincides with the conclusion Svelte Building and updating Svelte - means to replace the sewn compiler and server part and run tests ([40](#40-server-part-of-svelte-is-sewn-into-shugo)). There are three differences: comments of hydration are cut ([44](#44-comments-of-hydration-are-cut-by-sight)The layout context does not reach the page ([42](#42-shared-components-do-not-use-context-or-propsid)), warnings about what is changing in the browser, pages do not have ([34](#34-checks-svelte)).*

**Why so strict.** The template should be transferred to Svelte And then there will be islands. Svelte. If only Shugo Repetitive Svelte "about", the agent could not count on any of the rules: gaps at the edges of the blocks, `false` in the text, `class` from the array.

## 29. `<script module>` and export

**Waiting for approval.** `getStaticPaths` is exported from `<script module>`. Export from regular `<script>` is a prompt error, `export let` is a prompt error about `$props()`.

**Why not.** In Svelte, `<script module>` is performed once per module, and `<script>` is executed every time a component is displayed. `getStaticPaths` is a route function, not a display function: in `<script>` it would see variables that are not present when it is called. `export let` in Svelte 5 is also an error. In SvelteKit, the list of pages is written in a separate file `+page.ts` (`entries`); in Shugo, it remains in the page file as before.

*C 22 November 2026 the Year of Compile Svelte. Error Shugo — only `getStaticPaths` the City of London `<script>`Other exports from `<script>` Svelte 5 Accept. `export let` The mistake itself Svelte, `legacy_export_invalid`: its text and prompts `$props()`.*

## 30. `<svelte:head>` and title

**Waiting for approval.**

- Contents `<svelte:head>` falls into place `%shugo.head%` shell `src/site.html`, in the order of the document: external layouts, internal, page. `<title>` is one the last, and stands after the rest. This is what Svelte does. Since November 22 2026, the head assembles the server when it shows the page: `<svelte:head>` blocks lie in the record of the template that wrote them ([Blocks](../docs/blocks.md#page-head)).
- `<svelte:head>` works only in the page and in the layout. In the normal component, this is an error: the common component, which is calculated once per entire site, does not have a page in which `<head>` to write. Allow it in components that are not common, you can: the head of the page collects the server from the blocks of all the templates that he showed.

*With 22 November 2026 year `<svelte:head>` works in any component: its renderer Svelte. What gave the common component lies in its record, and the server puts this in the head of each page where this block is shown.*

## 31. Snippets and common blocks

The **Waiting for approval.** Snippet without parameters, written inside the component tag (`children` too), is calculated where written and stored in the caller's record. The component to which such snippets were transferred remains a common block. Therefore, the layout is stored by its own record, although each page passes it its `children`. The snippet with parameters is calculated where it is shown by `{@render}`, - [Blocks](../docs/blocks.md).

*22 November 2026 refined [43](#43-render-shared-block-snippets-separately): such a snippet Svelte renders separately, with its head and `<title>`, and a snippet that does not render so leaves the component in place.*

## 32. Style class on all component elements

**Deferred** is the author of the project 22 November 2026 year. *Closed the same day by the decision of the author of the project that the templates compiles Svelte: class puts itself Svelte, only on selected elements, and throws out extra selectors.* As it was: Svelte puts the class `svelte-…` only on the elements that selector selects, and throws selectors that do not choose anything. Shugo put a class on all components and left all selectors.

**Why not.** To select elements like Svelte, you need to parse selectors and match with a template, including blocks and dynamic classes, a noticeable part of the Svelte compiler. The page view does not change from this. The question is whether to do exactly in [Styles](styles.md#open-questions).

## 33. Extensions `.svelte` and `.island.svelte`

**Approved 22 November 2026 year.** Common templates have the extension `.svelte`. The suffix `.island.svelte` marks components that are hydrated when called from Shugo; the filename shows this in the editor, project tree and imports. Linking data from the address is specified in the call. [Svelte islands](scripts.md#svelte-islands)

The analysis of the usual templates is translated into the official parser Svelte 5.59.1. The assembly of Shugo still writes the framework and code that marks the reading of the data. TypeScript is written in `<script lang="ts">`, as in Svelte. *The same day, the author of the project decided that the templates and compiles Svelte: the template frame is no longer, data readings are marked the same way, and HTML Sd Svelte. TypeScript the City of London `<script lang="ts">` He also handles Sveltenot esbuild ([46](#46-typescript-in-templates-is-the-one-that-accepts-svelte)).*

**It was.** Before the 22 of November, 2026 files were called `.shugo`, parsed by their parser Shugo, and TypeScript was accepted in any `<script>`. The file extension says that the `Shugo.query` file is not a component of the Svelte file. With the `shugo` module and the official parser, this argument has disappeared. The `.shugo` template is now an assembly error with a hint on how to rename it.

## 34. Checks Svelte

**Waiting for approval.** C 22 November 2026 the Year of Compile Svelte, so errors and warnings are his own, everything: nesting elements, confused tags, runes places, outdated `<svelte:self>` and `<svelte:component>`Availability (`a11y_…`), unused selectors. Message Svelte As it stands, citing his explanation: `https://svelte.dev/e/…`. Not only warnings about what is changing in the browser are displayed: `state_referenced_locally`, `non_reactive_update`, `binding_property_non_reactive`, `perf_avoid_inline_class`, `perf_avoid_nested_class`, `store_rune_conflict`. Static page is rendered once, and the island to which they are important, Svelte compiles for the browser and warns there. Its errors Shugo Adds to Errors Svelte: shell markup in template, island tags, first syntax tags Shugo, slots Svelte 4 (`<slot>`, `slot="…"`, `let:`, `<svelte:fragment>`), `getStaticPaths` the City of London `<script>` and `---` at the beginning of the file.

The *As it was before this day:* parsing errors were Svelte, and the rest was repeated by Shugo, without warnings about availability and rune site checks.

## 35. Folder `src`

**Waiting for approval.** Folder `src/` is the author's solution. Details:

- `src/` has `pages/`, `components/`, `lib/`, `styles/`; `shugo.json`, `data/` and `public/` at the root;
- `.svelte` files are only searched in `src/`; importing a `.svelte` file from another location is a prompt error;
- `@/` in import means `src/`, as is customary in Vite projects;
- the store stand scripts moved to `examples/catalog/scripts/`: only the site code remained in `src/lib/`.

## 36. The `src/site.html` shell

**Left as is, the approval is delayed** by the author of the project 22 November 2026. Separate file with places `%shugo.head%` and `%shugo.body%` and the name `site.html` - the author's decision. Details:

- No file - takes the shell of the framework: `<!doctype html>`, `<meta charset="utf-8">`, `<meta name="viewport" …>` and both places; `lang` is not in it.
- Each place stands exactly once, the head is before the body. Otherwise, the assembly stops with an error that says what is wrong.
- The shell is copied as written: there are no expressions in it, essences and gaps are not touched.
- The template that `<html>`, `<head>`, `<body>` or `<!doctype>` write is an assembly error, one per file, with a hint about the shell and `<svelte:head>`.
- In place of `%shugo.head%`, what the templates gave in `<svelte:head>` is followed by a link to styles; in place of `%shugo.body%`, a page is followed by, then, in development mode, a script reboot.
- Modify shell - change project: the next build calculates the site again, as after editing the template.

**Why not.** This is how `src/app.html` is arranged in SvelteKit, and the agent knows this technique. Error instead of silent `<html>` inside `<body>` says where to move the markup. Compute the site again after the shell is not necessary - pages do not depend on it - but any project revision is also arranged; a separate path for the shell can be done if this becomes noticeable.

## 37. Files not in HTML: module with `GET`

**Left as is, the approval is delayed** is the author of the project 22 November 2026 year. Module `src/pages/sitemap.xml.ts` with `GET` — the author's decision. Details:

- `GET` gets nothing and returns a string. `async`, promise, `Response` and object are a prompt error: for an object, `JSON.stringify`, for `Response`, return the text itself. `GET({ params })` is a prompt error about `page.params`.
- File type is an extension in the name before `.ts` or `.js`. Module without it (`data.ts`) - error: so in the page folder does not hide the code that does not give anything. A module whose name or folder starts with `_` is an import code; `.d.ts` is a type declaration, not a module.
- `index` in the module name is a common name: `blog/index.xml.ts` gives `/blog/index.xml`, not `/blog.xml`.
- Options and `getStaticPaths` are like a page; an address that has no name left before the extension is an error.
- There are no layouts and shell in the file: the text is written as is.
- The server responds with an extension type, just like for files from `public/`; the unknown extension is `application/octet-stream`.
- When two routes name the same parts of an address, the first one is the one with the extension: `/[id].json` before `/[slug]`.

**Why not.** This is how Astro files are written.`src/pages/rss.xml.js` S `GET`) and SvelteKit (`+server.ts` S `GET`A line instead of `Response` Because the pages in Shugo It is not written without waiting, but text. `Response` is read only by waiting. The file is the same page for storage and analysis of changes: one entry from one block, so the update calculates it on the same lines read.

## 38. Exporting module files

**Left as is, the approval is delayed** is the author of the project 22 November 2026 year.

- The module file is written under its address: `/sitemap.xml` - `sitemap.xml`, `/api/jug.json` - `api/jug.json`.
- The repository remembers the route of the page that is missing, and the export deletes the file with the correct name: `api/jug.json`, not `api/jug.json.html`.
- Export list `.shugo-export.json` lists the files of the modules. Export, which builds all pages again, removes from them those that the site no longer has; it finds pages in HTML by extension `.html`, as before.

**Why not.** The page file name in HTML is output from the address, and the module file name is not: `/v1.2` can be both a `v1.2.html` page and a file. Without the route, exports would not know what to delete, and would leave an outdated file.

## 39. Stand `examples/islands` removed

**Waiting for approval.** The author agreed to leave one search, `search("table", { … })`, and transfer the stand `examples/islands` to it. It is impossible to translate it into one, so the stand is removed:

- Price and availability signatures, discounts and filter by size built an old search on Go according to the store scheme, with options from the second table. General search gives columns of one table, and the SSR program on the island server does not calculate the expression: signatures would have to be put in the table ready. After the translation, the stand would become a copy of the product page from `examples/search`.
- The islands without search — a counter and two islands on one page — went to `examples/search`, the `/counters` page.
- The experimental reportsthe methods, checks, measurements, and history examinedare compressed into [Svelte islands](../docs/islands.md); the full text and measurement scripts remain in the git history, in the `islands` branch.
- Old search tests that checked for common propertiesreader accuracy, route enumeration, module level search, write during calculationwere rewritten for general search.

**What's lost.** Old search changed price right in the index by journal; general builds the table index anew when its column changes: on 100,000 products near 1.2 s.

## 40. Server part of Svelte is sewn into Shugo

**Approved by the author of the project 22 November 2026 years:** “Svelt built-in version”, and the browser part too. Details below — how it is done.

Compiler Svelte 5.59.1 was sewn into Shugo and before. Now its server part is also nearby: modules `svelte`, `svelte/internal/server`, `svelte/server`, `svelte/store`, `svelte/reactivity`, `svelte/reactivity/window`, `svelte/easing`, `svelte/transition`, `svelte/animate`, `svelte/motion`, `svelte/events`, `svelte/attachments`. Each - one file, the common code - in the common pieces, 265 KB. They collect esbuild from package npm the same version of the script `internal/svelte/vendor/make.go`for the server and without code, which works only during development. The server part is sewn into `_build/eval.mjs`. Importing another module Sveltefor example `svelte/legacy`Error with the list of those that are available.

**Why not.** The code that the compiler issued only works with the server part of the same version. Taking it from the `node_modules` project would mean requiring `npm install svelte@5.59.1` to each project, even without islands, and check the version. *C 22 November 2026 the Year of Stitching and Browser Part - Modules `svelte`, `svelte/internal/client` and the same ones that the server has, which make sense in the browser, 403 KB, - and the islands compiles the same sewn compiler: the project with the islands does not need to put anything. Nearby lies acorn 8.19.0Parser JavaScriptwhich he works SvelteNo change: them Shugo Read the server code of the island linked to the request.*

## 41. Rune-only templates

**Waiting for approval.** Templates compile with `runes: true`: Svelte syntax 4 is `export let`, `$:`, `$$props` is a Svelte error. `<slot>` Svelte still accepts with a warning in this mode, and `slot="…"`, `let:` and `<svelte:fragment>` are silent; Shugo considers all this a mistake with a hint about snippets. `slot="…"` is a slot of Svelte 4 only when the item is directly in the component tag; elsewhere it is a common attribute, for example for a custom item.

*22 November 2026 year author of the project decided: `<svelte:options runes={false} />`, which includes the syntax Svelte 4, is also a bug, with a hint. Islands compile on runes as well ([49](#49-islands-only-on-runes)).*

**Why not.** One entry for all - fewer errors in the agent. Slots Svelte 4 Svelte transmits differently than the snippets, and the common block to which they were given could not show what was given to it.

## 42. Shared components do not use context or $props.id()

**Waiting for approval.** The rule of the common component ([3](#3-which-blocks-are-shared)) is added: renders on-site a component that itself or through what imports and shows,

- takes the context of Svelte — `getContext`, `setContext`, `hasContext`, `getAllContexts`, `createContext`;
- calls `$props.id()`;
- shows the snippet inside `<select>`;

and another component, which was transferred to `bind:`. The layout is rendered separately from the page, so its context does not reach the page, and the names of `$props.id()` in it begin with the number of his file: `b3-s1`.

**Why not.** The shared block is rendered once, separately from the places where it stands. The context would give it components around — they are not with a separate render. `$props.id()` counts components that were previously rendered on the page there is no such account in the common block, and one name would appear on the page twice. The `<select>` value marks the selected `<option>`, and the `<option>` from the snippet is written elsewhere. Through `bind:`, the component writes to the one who showed it, and this will not happen with a separate render. Before Svelte, these funds were not in Shugo.

## 43. Render shared-block snippets separately

**Waiting for approval.** The snippet without parameters given to the common block is rendered separatelyby its render of Svelte, where written, with the context of this placeand stored in a place record along with what it gave to `<head>`, and by its `<title>`. The server shows it where the shared block shows it: the head and title go in the same order as Svelte, and the snippet, which the shared block did not show, does not give anything to the head. Snippet renders even if the shared block does not show it now: then the place record is ready when the shared block after changing its data begins to show it.

If the snippet cannot be rendered this way it falls or calls `$props.id()` the component is rendered on-site, along with its snippets, by the usual Svelte render. A snippet with parameters, even with default values for everyone, always leaves the component in place: the component shows it with its arguments. This is recognized by the source of the snippet function: such parameters are not included in the length of the JavaScript function.

The only difference from Svelte is that the layout context does not reach the page ([42](#42-shared-components-do-not-use-context-or-propsid)).

**Why not.** So the common block is calculated once, and the HTML pages are the same as the Svelte. The first version of the renderer snippet inside the render space, between labels: then the snippet under a condition that is in the common block false, dropped the assembly, its `<title>` played the title of the common block, and the `$props.id()` in the snippet shown twice gave two identical names. Each case is checked by `TestSvelteSemanticsAcrossBlocks`.

## 44. Comments of hydration are cut by sight

**Waiting for approval.** The author of the project decided to cut out comments Svelte everywhere except islands. Comments of the type that writes Svelte 5.59.1: `<!---->`, `<!--[-->`, `<!--[!-->`, `<!--[1-->`, `<!--[-1-->`, `<!--]-->`, `<!--$s1-->`hash components like `<!--1f5b1r6-->` and `<!>` At the end of “The Rich” `<select>`. `{@html}` output value by string, as y Sveltebut its function Shugo Replaces his: HTML Stands between the tags ShugoNot between comments. SvelteAnd the comments inside it remain, even the same kind.

**Why not.** Svelte can not help but write these comments: Svelte 5 does not have such a parameter. There are no other comments in the Svelte output — he does not display the template comments, except those brought by `{@html}`, and their Shugo does not touch. There is one possibility of error: the text `<!--[-->` inside `<script>` or `<style>`, written directly in the markup; this was not the case.

## 45. Storage page table — with rowid

**Waiting for approval.** Table `page` storage declared without `WITHOUT ROWID`; storage layout is `8`.

**Why not.** In a table without a rovid SQLite holds about a quarter of its volume on a file page, about 1,000 bytes, and takes the rest to a separate page, almost empty. Records with HTML — about 850 bytes on average and more on product pages — are often longer: on 10,000 of goods, the file was 32 MB, of which 13 MB is empty; with rovid, 20 MB. Price is another search for the index on the page query, the return from this has not slowed down.

## 46. TypeScript in templates is the one that accepts Svelte

**Waiting for approval.** `<script lang="ts">` compiles Svelte: it erases types, and `enum`, namespaces with values and other constructs that TypeScript converts into code, is a Svelte error. Previously, Shugo had been running the script through esbuild and `enum` was running. Instead of `enum`, it is a `as const` object and type from it. Import type without `type` Svelte leaves in the code, and assembly stops: `No matching export in "src/lib/types.ts" for import "Product"`. Therefore, type is imported via `import type`: `import type { Product } from "@/lib/types"`. Previously, such imports were removed by esbuild. The `.ts` modules are still being processed by the esbuild.

*With 22 November 2026 year to this message Shugo adds a hint if the module declared a type or interface with the following name: `Product is a type, and a type is imported with type, import type { Product } from …`.*

**Why not.** So in Svelte: SvelteKit by default also does not pass scripts through TypeScript. The agent who knows Svelte 5 knows this rule as well.

## 47. Posts by Svelte with reference

**Waiting for approval.** Errors and warnings Svelte are displayed in its text, in one line, with reference to the explanation: `` `</div>` attempted to close an element that was not open https://svelte.dev/e/element_invalid_closing_tag ``. Shugo messages are still in English, without references ([22](#22-message-language)). Two Svelte messages have been replaced by their own: about `await` in the Shugo component explains that the page is calculated without expectation, and about the markup inside `{…}` - what `{#if}` and `{#each}` are needed. The second is with the 22 of November 2026; it is recognized by the error of parsing JavaScript before `<` and by letter. Svelte's warning about the colon in the attribute name of `request:…` is not displayed: this is a Shugo entry for islands, and Shugo itself speaks about it.

**Why not.** The agent finds a rule and example of correction in Svelte.

## 48. Modules with runes

**Waiting for approval.** Modules `.svelte.js` and `.svelte.ts` compile Svelte, `compileModule`, as in SvelteKit: for pages for server, on the island for browser and for server. Svelte only reads JavaScript, so `.svelte.ts` first takes off the esbuild types; the Svelte source card is routed through the esbuild card, and the error shows a string in `.svelte.ts`.

**Why not.** So in Svelte 5 write the state, which is divided by components, and the islands need it in the browser. Before that, `$state` in such a module fell when calculating a page with an unknown error.

*22 November 2026 year the author of the project decided: such modules in `src/pages/` - code for import, not site files like `.d.ts` and files with `_`.*

## 49. Islands only on runes

**Waiting for approval.** Islands compiles sewn Svelte with `runes: true` as templates: `export let`, `$:` and `<svelte:options runes={false} />` in the island — errors. Previously, the island compiled the package of `svelte` project, and Svelte decided on file, on ru.

**Why not.** One entry for all `.svelte` files is less error for the agent. A normal component that stands in both the page and the island, otherwise it would compile in them differently.

## 50. Styles `css="injected"` — once per page

**Waiting for approval.** Svelte gives `<style>` a component with `<svelte:options css="injected" />` in the head of the page. Shugo puts it in a block record with a hole of its kind, `s`, and the server writes each such `<style>` to the page head once, no matter how many blocks page, layouts, shared blocks it did not give.

**Why not.** In one render, Svelte writes such a `<style>` once. The common block and layout are rendered separately from the page, and without this, the component shown on both the page and in the common block would give the head two identical `<style>`.

## 51. A component that endlessly shows itself

**Waiting for approval.** Components that are rendered on-site are considered: more 512 embedded in each other is a calculation error with a file chain, `src/components/Tree.svelte shows itself without end: src/components/Tree.svelte → src/components/Tree.svelte`. If there is no repetition in the chain, the error calls the deepest file.

**Why not.** Otherwise, JavaScript falls off `Maximum call stack size exceeded`, without a file and string, although [23](#23-small) promises a clear error. 512 is deeper than any real tree and smaller than the stack of Bun and Node holds.

## 52. How the assembly recognizes package components

**Waiting for approval.** The author of the project decided that the component Svelte from the package npm is still a bug with a hint. The assembly recognizes it as follows: the import path ends on `.svelte` or `package.json` packet in `node_modules` calls the components of Svelte a `"svelte"` field or a `"svelte"` condition in `"exports"`. So the component packages declare themselves for SvelteKit and Vite. The package that gives both components and ordinary code is also a mistake: what exactly is taken from it, the assembly does not disassemble.

**Why not.** The build package code does not compile, and such a component fell when calculating: on Bun - page, on Node - the whole process, without a hint.

*22 November 2026 year verification moved to where the assembler allows the import of the package: component is the path it found, on `.svelte`, or the package whose `package.json` names components. In the islands, the components of the packages work: the island is compiled entirely by the sewn Svelte.*

## 53. Files of the package manager — source of the project

**Waiting for approval.** `package.json`, `bun.lock`, `package-lock.json`, `pnpm-lock.yaml` and `yarn.lock` are included in the source code of each project: after their change, `shugo update` collects the project again and calculates the site. They used to only enter when the project had islands. The build revision is 5, so the project assembled earlier is reassembling once.

**Why not.** The page code takes packages from `node_modules` when calculating, and the collector does not see them. Updating the package in the project without islands left the pages old, and no one talked about it. Editing files inside `node_modules` hands does not catch it.

*22 November 2026 Year-round packages are being assembled in `eval.mjs` ([Project structure](project-structure.md#decisions)), and each of their files in the assembly is a source with a hash: you can see and edit inside `node_modules`. The files of the package manager remained sources for what is taken from `node_modules` by the assembly, Tailwind.*

## 54. Site modules are not waiting at the top level

**Waiting for approval.** The author of the project decided to intentionally ban `await` at the top level of `<script module>` to see where the wait is needed and track the data past the database. The ban applies to all site modules that calculate pages: `src/lib/`, modules with runes `.svelte.ts` and `.svelte.js`, modules among the pages. `await` and `for await` at their top level is an assembly error in its place: Shugo asks esbuild to assemble a module for the environment without such an expectation, and he calls the place. You can wait inside the function. Islands and modules that import only islands are code for the browser: they can wait. The memo no longer advises to prepare at the top of the module what is waiting, for example, code highlighting.

**Why not.** Without this prohibition in `<script module>` Transferring the waiting to `src/lib/`where he was advised to carry the memo, and the ban would not show anything.

## 55. Packages in `eval.mjs`: how made

**Waiting for approval.** The author of the project decided that the packages are put by the package manager of the project, and the page code is collected with them. Details:

- Importing a package first allows the assembler to itself, in the same way as it will then collect, and checks the package for its `package.json`. The error occurs at the point of import, including import from another package.
- Package is not in any `node_modules` around the file — error with the installation command. The command is on the project lock file: `bun.lock` — `bun add`, `package-lock.json` — `npm install`, `pnpm-lock.yaml` — `pnpm add`, `yarn.lock` — `yarn add`without it, in the form of an early time that calculates pages.
- Svelte versions, which the package calls in `peerDependencies` or `dependencies`, are checked against the `github.com/Masterminds/semver` sewn library: it understands the npm ranges. A record that does not range, such as `workspace:*`, says nothing.
- The package written for CommonJS, the framework library gives `require`, `__filename` and `__dirname` assembly through `globalThis`: it is performed first, before the site code. Their ads of these names in the modules do not face them.
- An error while the site code is loaded goes to Go just like a page error: it is caught by the `uncaughtException` handler, and Bun, and Node calls it for an error at the top of the module. Previously, the process was just completed, and Go wrote that it ended before the site is computed.
- The error `ENOENT`, whose innermost place is in the package, gets a hint: the package reads the file next to its code, and the code is now in `_build/`.
- esbuild's message about a file it can't assemble, `No loader is configured for ".node" files`, gets a prompt to take a packet on one JavaScript.
- In the islands, the same checks are made except for the components: the islands compile them.
- Build revision - 6: A project assembled when packages were taken from `node_modules` when computed is reassembled once.

**Why not.** So there's only one page. Svelte: `getContext` the package sees the context of the page, and the package `svelte/store` works without package `svelte` in the project. The missing package is at the assembly, at the place of import, and not at calculation. Changes to packages are visible as accurately as changes to the project code. Checked on Bun and NodeCommonJS Package `require` and `__dirname`Package with `getContext` and `svelte/store` Next to a stranger `svelte` the City of London `node_modules`, missing package, package under Svelte 4, native code, reading your file.

## 56. Package npm `shugo`: how made

**Waiting for approval.** The author of the project decided that Shugo is always put into the project through `bun add shugo` to types come into the project themselves. Details:

- It is arranged like esbuild: the `shugo` package gives a command, and the ready program under the system lies in the `@shugo/<system>-<cPU>` package - an optional dependence with the `os` and `cpu` fields. System six: Linux, macOS and Windows, each on x64 and arm64. The program is assembled without cgo, so all six are assembled on the same machine, about one and a half minutes; each weighs 17–18 MB, in the archive - about 7 MB. There are no installation scripts: Bun defaults and does not run them.
- The package command is `bin/shugo.js` on Node or Bun: it finds the system program and runs it with the same arguments, and ends up just like the program. Without the system package, it says what to put.
- The `shugo` module types are `index.d.ts` packages. The `TestTypesOfTheModuleShugo` test checks them against what the `shugo` module exports.
- Svelte types are from the same version of the `svelte` package as the program: the `shugo` package puts it dependent. Assembly does not take it.
- `tsconfig.json` package settings with which Shugo collects: `moduleResolution: "bundler"`, `verbatimModuleSyntax`, `allowImportingTsExtensions`, strict verification, `@/` is `src/`. He writes the project paths through `${configDir}`, so `tsconfig.json` project is one line of `{ "extends": "shugo/tsconfig.json" }`; so can TypeScript with the version of 5.5. `ambient.d.ts` connects Svelte types with runes: `.svelte.ts` modules know `$state`.
- The version is put `npm/make.go`: in the package `shugo` it is written as `0.0.0` - and his own, and packages for the system.
- Each package under the system are the license Svelte and information about the code from which it is made: the program carries Svelte inside. The license of the packages themselves is still `UNLICENSED`.

**Why not.** So make the tools on Go and Rust, which are put from npm, - esbuild, Biome, Turborepo: the package manager himself chooses the program for the machine, and nothing is downloaded past it. 23 checked November 2026 on Linux: `bun add` put a command, the program of this system and Svelte, `bunx shugo build` put together a site, and TypeScript 5.9 and 7.0 and `svelte-check` found no other errors.
