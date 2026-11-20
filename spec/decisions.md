# First version solutions awaiting approval

Here is written what was decided when assembling the first version without direct confirmation of the author of the project: where the answer was “don’t know”, “decision” or the question was not before. Every decision has a reason. After approval, the decision is transferred to the “Decision” section of its document, after refusal redone.

The solutions are from the most important to the small.

## 1. Data loading and expression are counted together, Go compares the result

**Rejected by the author of the project 20 November 2026 year.** Data change analysis is mandatory: expressions are calculated only for the changed one. The author’s doubt was only about who was comparing the dataGo or TSrather than whether a comparison was needed. The argument for “saving almost nothing” is not complete below: it takes into account expressions, but not the code of components and templates, which can be expensive. In return, [proposal](change-tracking.md). It is described in the first version.

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

**What's done.** Each block has a value tape. It is stored under the hash of itself, so the same tapes are in storage once. A page entry is a list of its blocks with links to the feeds. The shared block is stored as a separate entry under a name from the component file and parameters, and the page links to it by that name.

**Why not.** This is the answer to the question about the header. In the prototype, the page stored the hash of each of its blocks, including the cap. The header has changed its hash has changed the record of each of the 100,000 pages needs to be rewritten. In addition, these caps were in the data of each page, and all pages had to be re-read. Now the page stores the name of the shared block, not its content. The header has changed one recording has been rewritten, the pages have not been touched.

**“The Significance of the Entry Hash”** from the prototype meant: the block result was stored under the hash of the data that the block reads. In the first version, this is not: the hash is taken from the result.

## 3. Which blocks are shared

**What's done.** The component is calculated and stored once if both conditions are met:
- he himself and everything he imports does not read `Shugo.url` and `Shugo.params` and does not pass the markup to layouts;
- it is called with simple parameters: lines, numbers, `true`, `false`, `null` or without parameters.

The name of the shared block is composed of the file path and parameters: `components/ProductCard.shugo{"id":"p7"}`.

**Why not.** The agent does not need to mark anything. The header without parameters is general by itself. The card to which the product number was transferred is common for all pages with this product; the card to which the object of the goods was transferred is considered on each page anew.

**Price.** Components like `<Price value={n} />` It works, but storage is growing.

## 4. The key in `<g-for>` is not entered

**What's done.** `<g-for>` does not have the `key` attribute.

**Why not.** Key is needed when the input data is compared: the block recognizes its “own” element. Results are now being compared. If the fifth item is removed from the list, the tapes of the others do not change or are not copied without keys. An obligatory attribute that does not affect anything would only add errors to the agents. If you need a key (partial data reboot, updating the page in the browser), you can add it.

## 5. The program of the whole project is one file, the rantime is common

**What's done.** The compiler writes `_build/site.json`: templates of all files and routes. When calculating, this program is put in the storage next to the values. Rantime only needs a storage file and static files. Go code is not generated.

**Why not.** Answer to the question "page code or the whole project": the entire project, one file that the rantime reads once. The program and values change in a single transaction, so the server never sees new patterns with old values. The program is data, not code, so anything that runs Go is safe in itself: the worst thing that can happen is the wrong HTML.

**That's postponed.** Go-code generation as optional acceleration. The page assembly now takes microseconds, the win would be 12–15% from them.

## 6. Components and functions are imported in data code, as in Astro

**What's done.**
```
---
import Card from "../components/Card.shugo"
import { price } from "@/lib/catalog"
---
<Card title="x" />
```
A capital or dot tag (`<Card>`, `<ui.Button>`) is a component; the name is taken from the import. `@/` stands for project root.

**Why not.** This is a departure from the first wish of "importing components not in the data code". Reasons:
- The data code has become the component code, as in Astro: it reads parameters and functions are announced. Imports are also a familiar record to agents.
- The question of `components/shop/Cart` is decided by itself: the path is specified in the import, the name is chosen by the one who imports.
- No magic: by import, you can see where it came from.

**Nuxt and Nitro experience.** Nitro 3 removed autoimports altogether: "Nitro v3 is centered around explicat typed imports". In Nuxt 5, server-side autoimports are disabled by default. On the side of Vue-applications, Nuxt left them. The reasons given: the code is not clear without knowing the sources, the editor needs a separate step of preparation, the transition to the definition leads to an intermediate file, the names conflict.

## 7. Object `Shugo`

**What's done.** In the data code and in expressions, the object `Shugo` is available:

| Field | What is it |
| --- | --- |
| `Shugo.props` | component parameters; page has `props` from `getStaticPaths` |
| `Shugo.params` | page Address Settings |
| `Shugo.url` | page address, object `URL` |
| `Shugo.slots.has("name")` | is something transferred to the slot? |
| `Shugo.self` | component itself, to call itself |

Turning to `Astro.props` gives a bug with a hint of what to write instead.

**Why not.** The form is the same as that of `Astro.*`, the name is its own: semantics is different in places, and someone else's name would be misleading.

## 8. Slots

**What's done.**
```
<!-- component -->
<section>
  <g-slot> is shown if nothing was passed inside</g-slot>
  <footer><g-slot name="footer" /></footer>
</section>

<!-- usage -->
<Card>
  <p>in regular slot</p>
  <small slot="footer">v Footer</small> Slot
  <g-fragment slot="footer">Also there, without your tag</g-fragment>
</Card>
```
`slot="name"` works on a tag that stands right inside the component. Inside `<g-if>` and `<g-for>` is an assembly error.

**Options considered.**

| Option | Recording | Why not selected |
| --- | --- | --- |
| As in Astro | `<slot />`, `<Fragment slot>` | `<slot>` is a real HTML tag, and all Shugo tags start with `g-`. On `<slot>` and `<Fragment>`, the compiler responds with a prompt |
| As in Vue | `<template #footer>` | new entry with `#`, Astro agents not familiar |
| As in Svelte 5 | `{#snippet}` and `{@render}` | requires its blocks in braces |

The contents of the slot are counted in the template that wrote it, and are counted always, even if the component did not show it.

## 9. Layouts

**What's done.**
- `+layout.shugo` in the folder wraps all pages of this folder and nested.
- The nested layout turns out to be **inside** external, rather than replacing it.
- The layout must contain `<g-slot />`: the page gets there.
- Page passes markup to the layout via the slot: `<title slot="head">…</title>` at the top level of the page. It gets the nearest layout, which has `<g-slot name="head" />`. If such a slot is not in any layout - build error.

**Why not.** On the question "does nested replace external - do others have the same?": no. In SvelteKit, Next.js and PocketPages nested layouts wrap around each other. Replaces only Nuxt, where the page has one named layout. When replaced, each partition layout would repeat `<html>` and `<head>`.

**That's postponed.** Reset the chain for a separate page (in SvelteKit it is `+page@.svelte`).

## 10. Routes

**What's done.**

| File | Address |
| --- | --- |
| `pages/index.shugo` | `/` |
| `pages/about.shugo` | `/about` |
| `pages/blog/[slug].shugo` | `/blog/…`, one parameter |
| `pages/docs/[...path].shugo` | `/docs` and deeper |
| `pages/(shop)/cart.shugo` | `/cart`: folder in parentheses is not included |
| `pages/_parts/x.shugo`, `pages/_x.shugo` | no Page |
| `pages/404.shugo` | shown for addresses without page |

- Names with `+` are occupied by the framework. Now there is only `+layout.shugo`; any other is an assembly error.
- Part of the path is either text or one parameter. `post-[id]` is an error.
- When two routes give one address, the one that calls it more accurate wins: the text is stronger than the parameter, the parameter is stronger than the remainder of the path. Two files with the same type of address are assembly errors.
- The page has one address, without a slash at the end. `/about/` is redirected to `/about`.
- The redirects are given in `shugo.json`: `"redirects": {"/old": "/new"}`.

**Why not.** This is the common place Astro, SvelteKit, Next.js and Nuxt. There is no consent about slash: SvelteKit and Next.js remove it, Astro and Nuxt accept both options. One address per page is selected to avoid duplicates.

## 11. `getStaticPaths`

**What's done.** As in Astro: function returns list of `{ params, props }`. `props` gets into `Shugo.props` pages, so all pages can be retrieved with one request. `paginate` comes to the function, it cuts the list into pages in the same way as in Astro. There is no separate step “one page data”: it is the data code of the page itself.

The page that is missing from the list is removed from the storage, and the server responds to its 404 address.

## 12. What outputs `{expression}` and `attribute={expression}`

**What's done.**

| Value | In text | In attribute |
| --- | --- | --- |
| `null`, `undefined` | nothing | attribute no |
| `false` | nothing | attribute not available; `aria-*`, `data-*`, `draggable`, `spellcheck`, `contenteditable` — `"false"` |
| `true` | `true` | `disabled`, `hidden` and the like have a name without meaning; the rest have `"true"` |
| string, number | as is, with the shielding | as is, with the shielding |
| list | elements in a row, without separator | `String(value)` |
| object | `String(value)` | `String(value)` |

**Why not.** `false` The text is not displayed to work the usual recording `{condition && "text"}`. Otherwise, this is the rule "scalars as is". The text behaves like in Astro. Attributes behave like in React: `class={condition && "x"}` does not give `class="false"`, and `aria-expanded={false}` Gn `"false"`.

## 13. Ready HTML — tag `<g-html>`

**What's done.** `<article><g-html value={post.body} /></article>`. On `set:html`, the compiler responds with a prompt.

**Why not.** All designs of Shugo are tags with `g-`. A separate layer of modifiers like `set:html` is not needed for one opportunity.

## 14. Addresses in links are checked

**What's done.** In `href`, `src`, `action`, `formaction`, `poster`, `cite`, the value from the expression is replaced by `about:invalid#blocked` if its circuit can execute the code (`javascript:` and the like). Relating addresses are allowed, `http`, `https`, `mailto`, `tel`, `sms`, `ftp` and pictures in `data:`.

**Why not.** The address often comes from data that was entered by visitors. So it was in the prototype.

## 15. Styles

**What's done.**
- `import "../styles/main.css"` in the data code connects the style file. All the connected files and the contents of all `<style>` tags are collected into one file with a hash in the name. A link to it is added before `</head>`.
- If styles have Tailwind directives, the file collects Tailwind from the `node_modules` project.
- `<style>` is valid for the entire site. `<style is:inline>` remains in place as written.

**That's postponed.** Style limit with its component. In Astro, an attribute is added to each component tag with a hash, and selectors are rewritten.

## 16. `<script>` remains as written

**What's done.** The contents of `<script>` are not changed. The expressions in the parentheses inside it are not calculated.

**That's postponed.** Scripting Assembly: TypeScript, Imports, One File to Site. This is a separate document of ["Scripts on the Page"](scripts.md).

## 17. Strict analysis of markup

**What's done.** Each tag must be closed. Exceptions are tags without content (`<br>`, `<img>`) and recording `<div />`. An unclosed or confused tag is an assembly error with place. HTML comments and spaces fall into the page as written; spaces around the edges of the entire template are cropped.

**Why not.** Assembly error with exact location for agent is more useful than silently "fixed" markup.

## 18. Project structure

**What's done.**
```
shugo.json settings
pages/ pages and layouts
components/ components (by agreement: component can be imported from anywhere)
lib/ shared code on TypeScript
public/ files that are given as is
styles/ styles
_build/ everything framework writes
```
There is no `src` folder: the path is shorter, and the sources are already separated from `_build` and `node_modules`.

`_build` contains: `eval.mjs` (JS for the computation stage), `site.json` (project program), `assets/` (styles), `store.db` (storage), `dist/` (site in files after `shugo export`).

## 19. Commands

| Command | What does |
| --- | --- |
| `shugo build` | compilation and calculation of all pages |
| `shugo update` | re-write, write only changed |
| `shugo serve` | server: pages are collected on the fly from the storage |
| `shugo export` | site to files; only changed ones are written |
| `shugo dev` | assembly, server, reassembly when changing files, updating the page in the browser |

Report `build`, `update` and `export` can be obtained in JSON (`-json`). JS-runtime is selected as follows: `-runtime`, then `"runtime"` the City of London `shugo.json`, then BunIf it is installed, otherwise Node.

## 20. Storage

**What's done.** One SQLite file with four tables: page entries, shared block entries, value feeds, utility data with the project program. The server reads the page entirely from one state. Update — one transaction.

The driver is `modernc.org/sqlite`, without cgo. It is hidden behind the `store` package, you can replace it in one place.

**Failed update.** The question is postponed. While the simplest is done: if the project code fell, the transaction is canceled and the storage remains the same.

## 21. Export to files

**What's done.** Page `/about` is written in `about/index.html`, page 404 is written in `404.html`. The file with the former content is not copied, the file of the missing page is deleted.

**Why not.** `about/index.html` gives any web server without setting up.

## 22. Message Language

**Implementation.** The compiler, CLI, and specification use English.

**Why not.** Messages are read primarily by agents and searched for. Habits from Astro (`set:html`, `<slot>`, `Astro.props`, markup inside `{}`) the compiler recognizes and says what to write instead.

## 23. Small

- **The component can call itself**: `<Shugo.self node={child} />`. The ban was a prototype limitation. Infinite call without parameters is caught with an understandable error.
- **Module path Go** — `shugo`. Before publication, it must be replaced with the present one.
- **Author of commits** is the name of `itareo` and the owner's mail, specified only in this repository.
- **Page queue.** Simultaneously calculates the 32 page to wait for the data to go in parallel. The process of JS is one.

## 24. Verification on agent

**What's done.** The individual agent was given only [memo](../docs/authoring.md) and the `shugo` commandno source, examples, or specificationand the task: a blog with a breakdown into pages, tag pages, nested layout, three components given in JSON, styles, and script.

**What happened.**
- The site from 25 pages was collected on the first try. There were no build errors, HTML is checked for shielding, links and nesting layouts.
- One real trap: the agent imported JSON with data like in Vite and Astro. The imported file becomes part of the code, and `shugo update` has not seen any changes to it. Fixed: `update` now notices himself that the project has changed, and compiles it again.
- The update report named three changed pages, and the export recorded six files: another three pages showed a changed card. Fixed: the report appeared the number of pages on which HTML changed.
- Spaces in the memo: addresses when broken down into pages, `page` fields, priority routes where to keep data files that types are not checked. Added.

**What it does not show.** The agent did not need error messages, so they were not checked on the agent. The task was one, the model one.
