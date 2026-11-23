# Command line and development mode

## Decisions

- Development mode is needed: reassembly when changing files and updating the page in the browser.
- Changing the project code calculates the entire site anew, and this is correct. The development mode will receive a real Live Reload separately: [proposal](../docs/proposals/live-reload.md) (21 solution November 2026 year).
- CLI messages and the specification use English so agents can read them and search for their wording. Confirmed November 21, 2026.
- `shugo init` is needed: a new project with an example, `tsconfig.json`, types for the editor and a memo for the agent (21 solution November 2026 years). Not yet done.
- Type checking is a separate `shugo check` command, along with component parameter checking (21 November 2026 solution). It is made by `svelte-check` with TypeScript sewn into Shugo as Svelte: the project does not need to put them, nor npm (the decision of the author of the project 22 November 2026 year). Not yet. *With the 23 November Shugo is put by the npm package, and sew them maybe not needed: [question 3](open-questions.md#3-shugo-check-sewing-or-delivering-with-package).*
- How `shugo` gets on the machine: ready files for all systems in releases on GitHub, npm package, which puts the finished file under the system, like esbuild, and Homebrew (22 solution November 2026 year). *Changed by the decision of the author of the project 23 November 2026 year:* Shugo is always put in the project package npm, `bun add shugo`, and with it the types of `shugo` module come into the project. Packages are made, not yet published, see. [below](#the-shugo-npm-package).

## Implemented, pending approval

```
shugo build  [directory]   compilation and calculation of all pages
shugo update [directory]   calculate what the data or project changes have touched; only the difference is written
shugo serve  [directory]   server: pages are collected on request
shugo export [directory]   site to files for any web server; after the first time - only changed
shugo dev    [directory]   update, server and update every time you change files
```

A folder is a project; without it, the current one is taken.

| Parameter | What does |
| --- | --- |
| `-json` | `build`, `update` and `export` report in JSON |
| `-check` | with `update`: calculate everything and name pages that have changed, although no line of the database they read has changed |
| `-runtime bun` or `node` | than to execute JS - the Svelte compiler and project code - in `build`, `update` and `dev`; otherwise `"runtime"` from `shugo.json`, otherwise Bun, if it is installed, otherwise Node |
| `-addr address` | where they listen to `serve` and `dev`; by default `localhost:4321` |
| `-out folder` | where `export` writes; default `_build/dist` |

The option that the command does not use is an error ([27](decisions.md#27-commands)).

### How Teams are Different

| Command | Compiles | Calculates | Gives pages | Follows files |
| --- | --- | --- | --- | --- |
| `build` | always | all | none | none |
| `update` | if the project has changed | what changes have occurred | none | none |
| `serve` | none | none | from storage | none |
| `export` | none | none | write to Files | none |
| `dev` | if the project has changed | what changes have occurred | from storage | yes, and resets open pages |

`dev` is `update` and `serve` together, plus tracking: after each file change, it makes `update` and tells the pages to reboot open in the browser. It does not calculate anything of its own: the pages are the same as after `shugo update`. You need it to work on the site: one command that is run once, and the site in the browser itself has time for code and data changes. The suggestion to make it truly different is to compute the page on demand, not the entire site, in [Live Reload](../docs/proposals/live-reload.md).

### When a team

- The data has changed — `shugo update`. It does not compile the project, but runs the already collected JS and calculates only what the changed lines read.
- If you run `shugo update` after changing the project itself, it will notice this, compile the project and calculate everything. To do this, the build records hashes of all files from which the project is collected, and a version of Svelte. So the site can not show the result of the old code.
- `shugo build` calculates everything always. It is needed if pages read something past the database.
- `shugo serve` can not be restarted: it sees each update from the next request.

### Reports

Shop on 100,000 products, changed the price of one product; run 22 November 2026 years:

```
$ shugo update
updated in 31 ms: 104202 pages, 100009 shared blocks
  computed 1 pages and 1 shared blocks, 9 blocks in all
  1 rows of the data have changed
  pages: 0 added, 1 changed, 0 removed
    changed: /products/product-5
  shared blocks: 0 added, 1 changed, 0 removed
    changed: src/components/ProductCard.svelte{"id":"p5"}
  the HTML differs on 7 pages: they show a shared block that has changed
```

The report has two different numbers. “Changed pages” are page entries in the repository: one. “HTML is different” are the pages that a visitor will see others: the product page itself and the six pages where his card stands. Their records are not touched because the card is kept separately.

In JSON, the same time of each step is in milliseconds:

```json
{
  "build": "7e2a9db7a114efe7bddf2f8b9ea84dbe",
  "rebuilt": false,
  "data": { "whole": false, "rows": 1, "routes": 0, "position": 5060 },
  "computed": { "pages": 1, "shared": 1 },
  "blocks": 9,
  "pages": { "total": 104202, "added": 0, "changed": 1, "removed": 0, "changedNames": ["/products/product-5"] },
  "shared": { "total": 100009, "added": 0, "changed": 1, "removed": 0, "changedNames": ["src/components/ProductCard.svelte{\"id\":\"p5\"}"] },
  "render": 7,
  "renderNames": ["/", "/categories/c5", "/products/product-5", "…"],
  "shadowed": 0,
  "time": { "open": 0.3, "data": 3.7, "list": 0.1, "compute": 24.0, "save": 0.2, "total": 28 }
}
```

- `data` - what the database log said: how many lines changed, how many lists of route pages were read again and to which log line the data was taken into account. `whole` means that the log could not say what had changed, and everything was calculated.
- `render` — how many pages of HTML have changed: these are the changed pages and pages that show the changed common block. You can reset the cache in front of the site from this list.
- The names in the report are no more than a thousand of each species.
- `rebuilt` means that the project has been compiled anew and everything is calculated from scratch.
- `shadowed` - how many addresses did the dynamic page give way to a page with an exact name.

### Errors

An error in the project is shown with a place in the project file and a line of code. Errors and warnings Svelte displayed by its text, with reference to the explanation:

```
error: src/pages/about.svelte:4:15: `</div>` attempted to close an element that was not open https://svelte.dev/e/element_invalid_closing_tag
     3 | <section>
     4  <p>text</p></div>
       |               ^
```

The warning is written in one line and the assembly does not stop. It is printed when the project is compiled:

```
warning: src/pages/about.svelte:3:1: `<img>` element should have an alt attribute https://svelte.dev/e/a11y_missing_attribute
```

The error during the calculation shows the page that was calculated and the place in the original file, not in the JS collected:

```
error: Cannot read properties of undefined (reading 'title')
  while computing the page /about (src/pages/about.svelte)
  at src/pages/about.svelte:5:21
     4 | const first = products[10]
     5 | const title = first.title.toUpperCase()
       |                     ^
```

Node indicates the location accurately. Bun sometimes points to a nearby string: it rearranges the code itself before executing.

### Development Mode

`shugo dev` monitors all project files except `_build`, `node_modules` and hidden files. After the change, it does what `shugo update` did: the source code has changed compiles the project and calculates everything; the database has changed calculates what the changed lines have read; other files have changed calculates nothing. The pages are reloaded anyway. What to improve in it - in [the Live Reload](../docs/proposals/live-reload.md).

### The Shugo npm package

Shugo — project package, like Vite or SvelteKit:

```sh
bun add shugo                  # or npm install shugo
bunx shugo build               # p  b  b                           
```

The `shugo` package gives the `shugo` command, `shugo` module types, and TypeScript settings for the project. The ready program for the system lies in its package - `@shugo/linux-x64`, `@shugo/darwin-arm64` and so on - and the package manager puts only the one that fits the machine, like esbuild. The `shugo` command from the package runs it with the same arguments.

`tsconfig.json` project — one line:

```json
{ "extends": "shugo/tsconfig.json" }
```

Package settings tell TypeScript that `@/` is `src/`, what to check (`src/` but not `_build/` and not `node_modules/`) and how: as Shugo collects. `shugo` module types are from the `shugo` package, Svelte types are from the `svelte` package of the same version as the one sewn into the program: the `shugo` package puts it dependent. Assembly does not take it, it uses its own, and the editor and `svelte-check` it gives the types `svelte`, `svelte/store` and rune. 23 is verified on November 2026 year: TypeScript 5.9 and 7.0 and `svelte-check` find bugs in the code with `query`, `page` and `svelte/store` and do not find them in the correct one.

How packages are made and published [npm/README.md](../npm/README.md).

## Open questions

- Packages publication: `shugo` name in npm is free; `@shugo/…` packages need `shugo` organization in npm, and for all packages, Shugo license, while `UNLICENSED` costs.
