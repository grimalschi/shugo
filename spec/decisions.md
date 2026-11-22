# Decisions pending approval

Here is written what is decided without direct confirmation of the author of the project: where the answer was “don’t know”, “decision” or question before. Every decision has a reason. After approval, the decision is transferred to the “Decision” section of its document, after refusal redone. The numbers don’t change so that the links don’t break; every decision above tells you what happened to it.

Decisions 1–27 taken during the assembly of the first version, 28–35 when switching to syntax Svelte 21 November 2026 years, 36–38 — 22 November 2026 The year when the shell appeared `src/site.html` Files are not HTML.

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

**Approved by the author of the project 22 November 2026 year** along with a warning when the common component shares almost nothing is [Components](components.md#decisions). Warning thresholds are selected without the author and wait for the statement: they are below, in the "Warning".

**What's done.** The component is calculated and stored once if both conditions are met:
- neither he nor the modules he imports take `page` or `paginate` from the `shugo` module (before 22 November 2026 years — did not read `Shugo.url` and `Shugo.params`);
- it is called with simple parameters: lines, numbers, `true`, `false`, `null` or without parameters.

The name of the shared block is composed of the file path and parameters: `components/ProductCard.shugo{"id":"p7"}`.

**Why not.** The agent does not need to mark anything. The header without parameters is general by itself. The card to which the product number was transferred is common for all pages with this product; the card to which the object of the goods was transferred is considered on each page anew.

**Price.** Components like `<Price value={n} />` It works, but storage is growing.

**Warning.** When the entire site is computed, the build counts for each component of its shared blocks and the places where they are shown. If 100 blocks or more, and on average each is shown in less than one and a half places, a warning is printed with the advice to transfer an object to the component then it is calculated as part of the page. Layouts are not weighted: A layout that reads a page is stored for each page on purpose. With a regular data update, nothing counts: counting goes on all common blocks, about 14 ms on 10,000.

**Why such thresholds?** Less 100 blocks are almost worthless, and on the developer's small data, the warning would be noise. One and a half places on average means that almost every set of parameters is different. In the store from the example, the product card is shown on average in 5 places, the icon is in 1375, so there is no warning there.

## 4. The key in `<g-for>` is not entered

**Approved 21 November 2026 year:** is not required key, cycle is `{#each}` from Svelte. Key in parentheses, `{#each items as item (item.id)}`, is accepted as in Svelte, and on static page does not affect anything.

**Why is the key not needed?** Key is needed when you compare the input data: the block recognizes the "own" element. Shugo compares the results. If you remove the fifth product from the list, the tapes of the others do not change and are not rewritten - without keys.

## 5. The framework of the entire project is one file, the rantime is common

**The approach is approved by 21 November 2026 year:** frame instead of Go-code project. The name "frame" instead of "pattern program" approved 22 November 2026 year; in the code is a package `skeleton`.

**What's done.** The compiler writes `_build/site.json`: templates of all files and routes. When calculating this frame is put in the storage next to the values. Rantime only needs a storage file and static files. Go code is not generated.

**Why not.** Answer to the question "page code or the whole project": the entire project, one file that the rantime reads once. The framework and values change with one transaction, so the server never sees new patterns with old values. The framework is data, not code, so anything that runs Go is safe in itself: the worst thing that can happen is the wrong HTML.

**That's postponed.** Go-code generation as optional acceleration. The page assembly now takes microseconds, the win would be 12–15% from them.

## 6. Components and functions are imported in data code, as in Astro

**Replaced by 21 November 2026 year:** files `.shugo` components Svelte 5The components are imported into `<script>`As in Svelte: `import Card from "../components/Card.shugo"`. The reasons for the previous decision remained in force: the path is visible in imports, the name is chosen by the one who imports, there are no autoimports (Nitro 3 They refused, in Nuxt 5 They are off the server.

## 7. Object `Shugo`

**Replaced by the decision of the author of the project 22 November 2026 year:** global object is not, `query`, `paginate` and `page` imported from the module `shugo`, immediately, without a transition time — [Templates](templates.md#file-structure). `Shugo.query` and `Astro.params` give an error with a hint of what to import.

**It was.** C 21 November 2026 the Year in the Facility `Shugo` remained `Shugo.query`, `Shugo.params`, `Shugo.url` and `Shugo.paginate`The parameters of the component were already taken from `$props()`.

## 8. Slots

**Replaced by 21 November 2026 year:** The marking is transmitted to the component by snippets, as in Svelte 5: `children` and `{#snippet name()}…{/snippet}` inside component tag, `{@render …}` in the component. `<g-slot>`, `<g-fragment>`, `<slot>` and `slot="…"` give an error with a hint. Detailed in [Components](components.md#snippets).

## 9. Layouts

**Changed to 21 on November 2026 of the year, pending approval.** Layout is a component of Svelte: the page it shows `{@render children()}`, the title and meta tags the page writes in `<svelte:head>`; this is the author's decision. The rest awaits approval:

- `+layout.shugo` in the folder wraps all pages of this folder and nested; the nested layout turns out to be **inside** external, as in SvelteKit, Next.js and PocketPages. Replaces only Nuxt, where the page has one named layout.
- The root layout was written by `<html>`, `<head>` and `<body>`. Replaced by the decision of the author 22 November 2026: the document is written in the shell `src/site.html`, the root layout is a normal component ([Layouts](layouts.md#shell-srcsitehtml), [36](#36-the-srcsitehtml-shell)).

Reset Chain`+page@.svelte` the City of London SvelteKit) postponed. Resolved 22 November 2026 Year: The official name remains `+layout.shugo`; data from layout to page, how `load` the City of London `+layout.ts` SvelteKit, no and will not; the layout is stored separately from the pages - one entry for the entire site or, if reading the address, with its entry for each page ([Blocks](../docs/blocks.md#layouts)).

## 10. Routes

**Waiting for approval.** The author asked to compare with SvelteKit — a comparison in [questions](open-questions.md).

**What's done.**

| File | Address |
| --- | --- |
| `src/pages/index.shugo` | `/` |
| `src/pages/about.shugo` | `/about` |
| `src/pages/blog/[slug].shugo` | `/blog/…`, one parameter |
| `src/pages/docs/[...path].shugo` | `/docs` and deeper |
| `src/pages/(shop)/cart.shugo` | `/cart`: folder in parentheses is not included |
| `src/pages/_parts/x.shugo`, `pages/_x.shugo` | no Page |
| `src/pages/404.shugo` | shown for addresses without page |

- Names with `+` are occupied by the framework. Now there is only `+layout.shugo`; any other is an assembly error.
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
- Derogations: hydration labels are not written, style class is placed on all elements ([32](#32-style-class-on-all-component-elements)), `<svelte:head>` - only in the page and layout ([30](#30-sveltehead-and-title)), there are no availability warnings, runes are not checked. How to remove these deviations - [Svelte compatibility offer](../docs/proposals/svelte-compatibility.md).

**Why so strict.** The template should be transferred to Svelte And then there will be islands. Svelte. If only Shugo Repetitive Svelte "about", the agent could not count on any of the rules: gaps at the edges of the blocks, `false` in the text, `class` from the array.

## 29. `<script module>` and export

**Waiting for approval.** `getStaticPaths` is exported from `<script module>`. Export from regular `<script>` is a prompt error, `export let` is a prompt error about `$props()`.

**Why not.** In Svelte, `<script module>` is performed once per module, and `<script>` is executed every time a component is displayed. `getStaticPaths` is a route function, not a display function: in `<script>` it would see variables that are not present when it is called. `export let` in Svelte 5 is also an error. In SvelteKit, the list of pages is written in a separate file `+page.ts` (`entries`); in Shugo, it remains in the page file as before.

## 30. `<svelte:head>` and title

**Waiting for approval.**

- Contents `<svelte:head>` falls into place `%shugo.head%` shell `src/site.html`, in the order of the document: external layouts, internal, page. `<title>` is one the last, and stands after the rest. This is what Svelte does. Since November 22 2026, the head assembles the server when it shows the page: `<svelte:head>` blocks lie in the record of the template that wrote them ([Blocks](../docs/blocks.md#page-head)).
- `<svelte:head>` works only in the page and in the layout. In the usual component, this is an error: the common component, which is calculated once per entire site, does not have a page in which `<head>` to write. Allow it in components that are not common, in [the Compatibility Proposal](../docs/proposals/svelte-compatibility.md).

## 31. Snippets and common blocks

The **Waiting for approval.** Snippet without parameters, written inside the component tag (`children` too), is calculated where written and stored in the caller's record. The component to which such snippets were transferred remains a common block. Therefore, the layout is stored by its own record, although each page passes it its `children`. The snippet with parameters is calculated where it is shown by `{@render}`, - [Blocks](../docs/blocks.md).

## 32. Style class on all component elements

**Deferred** is the author of the project 22 November 2026; how to do as Svelte — in [the Compatibility Proposal](../docs/proposals/svelte-compatibility.md). Svelte puts the `svelte-…` class only on items that some selector chooses, and throws selectors that do not choose anything. Shugo puts the class on all components and leaves all selectors.

**Why not.** To select elements like Svelte, you need to parse selectors and match with a template, including blocks and dynamic classes, a noticeable part of the Svelte compiler. The page view does not change from this. The question is whether to do exactly in [Styles](styles.md#open-questions).

## 33. Extension `.shugo` and TypeScript

**`.shugo` extension is still left** is the 22 solution of November 2026 year. TypeScript is accepted in any `<script>`, and Svelte requires `lang="ts"`: whether to require it — in [the Compatibility Proposal](../docs/proposals/svelte-compatibility.md).

**Why `.shugo`.** File with `Shugo.query` and `getStaticPaths` was not a component of Svelte: its expansion was honest about it. Since November 22, 2026, `query` and `page` have been imported from the `shugo` module, and for the Svelte compiler, the file is a regular import component; the argument has been weakened but the extension has been left by the author's decision. Price: Editor doesn't recognize file without setting.

## 34. What is checked the same as in Svelte

**Waiting for approval**What to repeat again in [the Compatibility Proposal](../docs/proposals/svelte-compatibility.md). Errors and warnings Sveltewhich change the appearance of the page or break it, Shugo Repeats: nesting elements, confused tags, outdated `<svelte:self>` and `<svelte:component>`, `<slot>` and `export let` the City of Svelte 4. Accessibility Warnings (`a11y_…`, them in Svelte A few dozen) not.

## 35. Folder `src`

**Waiting for approval.** Folder `src/` is the author's solution. Details:

- `src/` has `pages/`, `components/`, `lib/`, `styles/`; `shugo.json`, `data/` and `public/` at the root;
- `.shugo` files are only searched in `src/`; importing a `.shugo` file from another location is a prompt error;
- `@/` in import means `src/`, as is customary in Vite projects;
- the store stand scripts moved to `examples/catalog/scripts/`: only the site code remained in `src/lib/`.

## 36. The `src/site.html` shell

**Waiting for approval.** Separate file with places `%shugo.head%` and `%shugo.body%` and name `site.html` — the author's decision. Details:

- No file - takes the shell of the framework: `<!doctype html>`, `<meta charset="utf-8">`, `<meta name="viewport" …>` and both places; `lang` is not in it.
- Each place stands exactly once, the head is before the body. Otherwise, the assembly stops with an error that says what is wrong.
- The shell is copied as written: there are no expressions in it, essences and gaps are not touched.
- The template that `<html>`, `<head>`, `<body>` or `<!doctype>` write is an assembly error, one per file, with a hint about the shell and `<svelte:head>`.
- In place of `%shugo.head%`, what the templates gave in `<svelte:head>` is followed by a link to styles; in place of `%shugo.body%`, a page is followed by, then, in development mode, a script reboot.
- Modify shell - change project: the next build calculates the site again, as after editing the template.

**Why not.** This is how `src/app.html` is arranged in SvelteKit, and the agent knows this technique. Error instead of silent `<html>` inside `<body>` says where to move the markup. Compute the site again after the shell is not necessary - pages do not depend on it - but any project revision is also arranged; a separate path for the shell can be done if this becomes noticeable.

## 37. Files not in HTML: module with `GET`

**Waiting for approval.** Module `src/pages/sitemap.xml.ts` with `GET` — Author's decision. Details:

- `GET` nothing gets and returns the line. `async`The promise, `Response` and the object is a bug with a hint: for the object - `JSON.stringify`, for `Response` — return the text itself. Astro recording `GET({ params })` Error with a hint about `page.params`.
- File type is an extension in the name before `.ts` or `.js`. Module without it (`data.ts`) - error: so in the page folder does not hide the code that does not give anything. A module whose name or folder starts with `_` is an import code; `.d.ts` is a type declaration, not a module.
- `index` in the module name is a common name: `blog/index.xml.ts` gives `/blog/index.xml`, not `/blog.xml`.
- Options and `getStaticPaths` are like a page; an address that has no name left before the extension is an error.
- There are no layouts and shell in the file: the text is written as is.
- The server responds with an extension type, just like for files from `public/`; the unknown extension is `application/octet-stream`.
- When two routes name the same parts of an address, the first one is the one with the extension: `/[id].json` before `/[slug]`.

**Why not.** This is how Astro files are written.`src/pages/rss.xml.js` S `GET`) and SvelteKit (`+server.ts` S `GET`A line instead of `Response` Because the pages in Shugo It is not written without waiting, but text. `Response` is read only by waiting. The file is the same page for storage and analysis of changes: one entry from one block, so the update calculates it on the same lines read.

## 38. Exporting module files

**Waiting for approval.**

- The module file is written under its address: `/sitemap.xml` - `sitemap.xml`, `/api/jug.json` - `api/jug.json`.
- The repository remembers the route of the page that is missing, and the export deletes the file with the correct name: `api/jug.json`, not `api/jug.json.html`.
- Export list `.shugo-export.json` lists the files of the modules. Export, which builds all pages again, removes from them those that the site no longer has; it finds pages in HTML by extension `.html`, as before.

**Why not.** The page file name in HTML is output from the address, and the module file name is not: `/v1.2` can be both a `v1.2.html` page and a file. Without the route, exports would not know what to delete, and would leave an outdated file.
