# Decisions pending approval

Here is written what is decided without direct confirmation of the author of the project: where the answer was “don’t know”, “decision” or question before. Every decision has a reason. After approval, the decision is transferred to the “Decision” section of its document, after refusal redone. The numbers don’t change so that the links don’t break; every decision above tells you what happened to it.

1–27 solutions are made when the first version is assembled, 28–35 - when Svelte 21 2026 is switched to syntax on November.

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

**What was done.** Each block had a value tape. It was stored under the hash of itself, so the same tapes were lying in storage once. Page entry - a list of its blocks with links to the tapes. The shared block is stored as a separate entry under a name from the component file and parameters, and the page links to it by that name.

**Why do we have common blocks by name?** This is the answer to the question about the header. In the prototype, the page stored the hash of each of its blocks, including the cap. The header has changed its hash has changed the record of each of the 100,000 pages needs to be rewritten. In addition, these caps were in the data of each page, and all pages had to be re-read. Now the page stores the block number, not its content. The header has changed one recording has been rewritten, the pages have not been touched.

**“The Significance of the Entry Hash”** from the prototype meant: the block result was stored under the hash of the data that the block reads. This is not: the hash is taken from the result.

## 3. Which blocks are shared

**Waiting for approval.** The author asked if the component is stored without dependencies once: yes — the answer is in [questions](open-questions.md).

**What's done.** The component is calculated and stored once if both conditions are met:
- he himself and everything he imports does not read `Shugo.url` and `Shugo.params` and does not pass the markup to layouts;
- it is called with simple parameters: lines, numbers, `true`, `false`, `null` or without parameters.

The name of the shared block is composed of the file path and parameters: `components/ProductCard.svelte{"id":"p7"}`.

**Why not.** The agent does not need to mark anything. The header without parameters is general by itself. The card to which the product number was transferred is common for all pages with this product; the card to which the object of the goods was transferred is considered on each page anew.

**Price.** Components like `<Price value={n} />` It works, but storage is growing.

## 4. The key in `<g-for>` is not entered

**Approved 21 November 2026 year:** is not required key, cycle is `{#each}` from Svelte. Key in parentheses, `{#each items as item (item.id)}`, is accepted as in Svelte, and on static page does not affect anything.

**Why is the key not needed?** Key is needed when you compare the input data: the block recognizes the "own" element. Shugo compares the results. If you remove the fifth product from the list, the tapes of the others do not change and are not rewritten - without keys.

## 5. The program of the whole project is one file, the rantime is common

**The approach is approved by 21 November 2026 year:** program instead of Go-code project. The author asks for another name for the "pattern program" - options in [questions](open-questions.md).

**What's done.** The compiler writes `_build/site.json`: templates of all files and routes. When calculating, this program is put in the storage next to the values. Rantime only needs a storage file and static files. Go code is not generated.

**Why not.** Answer to the question "page code or the whole project": the entire project, one file that the rantime reads once. The program and values change in a single transaction, so the server never sees new patterns with old values. The program is data, not code, so anything that runs Go is safe in itself: the worst thing that can happen is the wrong HTML.

**That's postponed.** Go-code generation as optional acceleration. The page assembly now takes microseconds, the win would be 12–15% from them.

## 6. Components and functions are imported in data code, as in Astro

**Replaced by 21 November 2026 year:** files `.svelte` components Svelte 5The components are imported into `<script>`As in Svelte: `import Card from "../components/Card.svelte"`. The reasons for the previous decision remained in force: the path is visible in imports, the name is chosen by the one who imports, there are no autoimports (Nitro 3 They refused, in Nuxt 5 They are off the server.

## 7. Object `Shugo`

**Changed to 21 on November 2026 of the year, pending approval.** With the transition to Svelte, the parameters of the component are taken from `$props()`, the snippets of the calling are also parameters, and the component calls itself import. The following remained in the `Shugo` facility:

| Field | What is it |
| --- | --- |
| `Shugo.query(table)` | request to the project database |
| `Shugo.params` | page Address Settings |
| `Shugo.url` | page address, object `URL` |
| `Shugo.paginate(request, ...)` | part of the list for the page with the number |

On `Shugo.props`, `Shugo.slots` and `Astro.props` — error with a hint. The author asked if there are alternatives to the object — they are in [questions](open-questions.md).

## 8. Slots

**Replaced by 21 November 2026 year:** The marking is transmitted to the component by snippets, as in Svelte 5: `children` and `{#snippet name()}…{/snippet}` inside component tag, `{@render …}` in the component. `<g-slot>`, `<g-fragment>`, `<slot>` and `slot="…"` give an error with a hint. Detailed in [Components](components.md#snippets).

## 9. Layouts

**Changed to 21 on November 2026 of the year, pending approval.** Layout is a component of Svelte: the page it shows `{@render children()}`, the title and meta tags the page writes in `<svelte:head>`; this is the author's decision. The rest awaits approval:

- `+layout.svelte` in the folder wraps all pages of this folder and nested; the nested layout turns out to be **inside** external, as in SvelteKit, Next.js and PocketPages. Replaces only Nuxt, where the page has one named layout.
- The root layout itself is written by `<html>`, `<head>` and `<body>`; in SvelteKit, they are written by `src/app.html`.

The author asked how it works in Svelte, the answer is in [questions](open-questions.md). Chain reset (`+page@.svelte` in SvelteKit) has been postponed.

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

**Waiting for approval.** Delete Missing Page Approved 21 November 2026 years. `props` the City of `getStaticPaths` Open question: examples of both ways [questions](open-questions.md). `getStaticPaths` Now stands in `<script module>` ([29](#29-script-module-and-export)), and `props` the Page takes from `$props()`.

**What's done.** As in Astro: function returns list of `{ params, props }`. `props` page receives through `$props()`, so the data of all pages can be obtained with one request. `paginate` comes to the function, it cuts the list into pages in the same way as in Astro. There is no separate step “one page data”: it is the data code of the page itself.

The page that is missing from the list is removed from the storage, and the server responds to its 404 address.

## 12. What outputs `{expression}` and `attribute={expression}`

**Replaced by 21 November 2026 year:** values are displayed as in Svelte - [Templates](templates.md#text). The previous rules (`false` - nothing, the list - in a row without a separator, attributes as in React) are removed.

## 13. Ready HTML — tag `<g-html>`

**Replaced by 21 November 2026 year:** `{@html …}` as in Svelte.

## 14. Addresses in links are checked

**Approved 21 November 2026 year.** Svelte does not do this: it is a conscious retreat, see [Templates](templates.md#how-shugo-differs-from-svelte).

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

**What's done.** One file SQLite: records pages and shared blocks with feeds inside, who is what common block shows who read what from the project database, missing pages, service data with the project program - [Storage](../docs/storage.md#tables). The server reads the page entirely from one state. Update is one transaction.

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

## 23. Small

**Waiting for approval**, except for calling himself.

- **The component can call itself.** *Replaced by 21 November 2026 year:* as in Svelte 5 — the component imports itself; `<svelte:self>` works with a warning. An endless call with the same parameters is caught with an understandable error.
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
- Indents: `javascript:` addresses are blocked (14 solution), hydration labels are not written, style class is put on all elements ([32](#32-style-class-on-all-component-elements)), `<svelte:head>` - only in the page and layout ([30](#30-sveltehead-and-title)), there are no availability warnings, runes are not checked.

**Why so strict.** The template should be transferred to Svelte And then there will be islands. Svelte. If only Shugo Repetitive Svelte "about", the agent could not count on any of the rules: gaps at the edges of the blocks, `false` in the text, `class` from the array.

## 29. `<script module>` and export

**Waiting for approval.** `getStaticPaths` is exported from `<script module>`. Export from regular `<script>` is a prompt error, `export let` is a prompt error about `$props()`.

**Why not.** In Svelte, `<script module>` is performed once per module, and `<script>` is executed every time a component is displayed. `getStaticPaths` is a route function, not a display function: in `<script>` it would see variables that are not present when it is called. `export let` in Svelte 5 is also an error. In SvelteKit, the list of pages is written in a separate file `+page.ts` (`entries`); in Shugo, it remains in the page file as before.

## 30. `<svelte:head>` and title

**Waiting for approval.**

- The contents of `<svelte:head>` are at the end of `<head>`, which writes the page or the nearest layout, in the order of the document: external layouts, internal, page. `<title>` remains one - the last, and stands after the rest. So does Svelte.
- `<svelte:head>` works only in the page and in the layout. In the usual component, this is an error: the general component, which is calculated once per entire site, does not have a page in which `<head>` to write.

## 31. Snippets and common blocks

The **Waiting for approval.** Snippet without parameters, written inside the component tag (`children` too), is calculated where written and stored in the caller's record. The component to which such snippets were transferred remains a common block. Therefore, a layout that does not read the address is calculated once per entire site, although each page passes its `children` to it. The snippet with parameters is calculated where it is shown by `{@render}`, - [Blocks](../docs/blocks.md).

## 32. Style class on all component elements

**Waiting for approval.** Svelte puts the `svelte-…` class only on elements that some selector chooses and throws out selectors that choose nothing. Shugo puts the class on all elements of the component and leaves all selectors.

**Why not.** To select elements like Svelte, you need to parse selectors and match with a template, including blocks and dynamic classes, a noticeable part of the Svelte compiler. The page view does not change from this. The question is whether to do exactly in [Styles](styles.md#open-questions).

## 33. Extensions `.svelte` and `.island.svelte`

**Approved 22 November 2026 year.** Common templates have the extension `.svelte`. The suffix `.island.svelte` marks components that are hydrated when called from Shugo. The filename shows this in the editor, project tree and imports. Linking data from URL is specified in the call. [Svelte islands](scripts.md#svelte-islands)

The question of whether to require `lang="ts"` for TypeScript in normal templates remains [open](open-questions.md).

## 34. What is checked the same as in Svelte

**Waiting for approval.** Errors and warnings Sveltewhich change the appearance of the page or break it, Shugo Repeats: nesting elements, confused tags, outdated `<svelte:self>` and `<svelte:component>`, `<slot>` and `export let` the City of Svelte 4. Accessibility Warnings (`a11y_…`, them in Svelte A few dozen) not.

## 35. Folder `src`

**Waiting for approval.** Folder `src/` is the author's solution. Details:

- `src/` has `pages/`, `components/`, `lib/`, `styles/`; `shugo.json`, `data/` and `public/` at the root;
- `.svelte` files are only searched in `src/`; importing a `.svelte` file from another location is a prompt error;
- `@/` in import means `src/`, as is customary in Vite projects;
- the store stand scripts moved to `examples/catalog/scripts/`: only the site code remained in `src/lib/`.
