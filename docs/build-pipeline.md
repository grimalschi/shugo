# Build pipeline

The project is being built in stages. Each stage has its own entrance and its own result. The next stage uses only the results of the previous ones.

## Decisions

- The work is divided into stages with a clear boundary: compilation of the project, work with data and expressions, assembly of pages in rantime.
- Between loading data and calculating expressions is the analysis of changes: expressions are calculated only for changed areas. Analysis is mandatory for any device stages. The choice is only in who compares: Go between two stages on TS or TS itself.
- When you change the project, the entire chain starts from scratch. When you change the data, the update is point-by-point. Confirmed by 21 on November 2026: code change calculates the whole site, and it is necessary. Development will accelerate a separate Live Reload - [proposal](proposals/live-reload.md).
- While every update process JS starts again and loads all the data; then the result is compared with the past. *Replaced by solutions from 20 November 2026 year:* update recognizes changes from the database log and calculates only what they touched - [Change tracking](change-tracking.md#decisions).
- Expressions and data code perform one JS process, not a page-to-page process: starting a process is more expensive than calculating a page.
- JS is performed by Bun and Node, both must work.
- Rantime always takes meanings from the saved and doesn't know how or when they were updated.
- The results are stored in SQLite.
- The entire update failed: the storage remains the same, nothing is applied by half, the data log is not lost, and the next update meets the same changes. Solved 21 November 2026 year; works as follows: [Failed Update](#failed-updates).
- Sandbox is not part of `shugo`. It describes only how to run the stage of JS in the sandbox. Solved by 21 on November 2026 year.

## Implemented, pending approval

Below is how the stages are arranged now: in the final version ([Change tracking](change-tracking.md#final-implementation)); templates compile and render Svelte 5.59.1 ([Shugo and Svelte](svelte.md)).

| Stage | Who performs | What gets | What gives |
| --- | --- | --- | --- |
| 1. Compilation | Go and Svelte on Bun or Node | project Sources from `src/` | `_build/eval.mjs` - the entire JS project in one file; `_build/site.json` - the project framework; `_build/assets/` - island styles and browser modules |
| 2. Calculation | Bun or Node running Go | `eval.mjs`, project database and its log | changes in `_build/store.db` and report on them |
| 3. Rantime | Go | `store.db`, `public/`, `assets/`, published search index if available | HTML pages on request or changed page files |

### Compilation

1. Finds normal files `.svelte` the City of London `src/` the Island `*.island.svelte` enter through imports and compile each compiler Svelte 5.59.1 for the server, `svelte/compiler.compile` S `generate: "server"`, `runes: true` and `css: "external"`, in the process Bun or Node; large project - in several at once. The compiler is sewn into Shugothe Custom Project Package `svelte` not needed. What Svelte does not accept, - error with the place in the file; what Svelte alerts warning, except for warnings about what changes in the browser: static page is rendered once. Template `.shugo` error with a hint on how to rename it. Your checks Shugo makes a tree that gives the same challenge: a template that writes `<html>`, `<head>`, `<body>` or `<!doctype>`, error, one per file: the document around the pages is written in the shell; errors and tags of the first syntax Shugo, slots Svelte 4, wrong island tags. By same tree Shugo Find out if they have a pattern. `page` or `paginate` from module `shugo` and context from `svelte`Does it call `$props.id()`Does the snippet show inside `<select>`It exports `getStaticPaths` Which islands are imported?
2. Finds `.ts` and `.js` modules in `src/pages/`, except for `.d.ts` and those whose name or folder starts with `_`. Each gives a file not in HTML; its type is in the name: `sitemap.xml.ts` gives `/sitemap.xml`. The module without the type in the name, `data.ts`, is an error. The route of such a module carries a file extension, it has no layouts. Module numbers come after template numbers.
3. The template module is the one that Svelte gave: imports, exports and the entire `<script module>` — there stands `getStaticPaths` — at the top, and the component is a function that performs `<script>` and writes HTML. The assembly template is two modules: a component Svelte, under the suffix `?svelte`, and a module Shugo, which imports the rest: it gives the same as `<script module>`, and the component is wrapped in `component(file number, component)`. Through this wrapper, each component display decides whether to render it in place or put a hole in the common block ([Blocks](blocks.md#shared-components)).
4. From the server part Svelte the Template Code `svelte/internal/server`The project itself can be imported. `svelte`, `svelte/store`, `svelte/reactivity` and other modules that make sense on the server. The server part of the same version is sewn into Shugo ([how it is made](../internal/svelte/vendor/README.md)). `{@html}` Shugo Replaces its function: it HTML between the labels so that the comments in it are not cut out as hydration comments ([Blocks](blocks.md#how-html-is-split-into-blocks)).
5. For imported `*.island.svelte`, Svelte compiles server and browser versions of their graph; for this, the project requires the `svelte@5.59.1` package the same version as sewn into Shugo. From the server version of Shugo builds the SSR program of the island, tied to the request data, and at the calculation stage checks it HTML with the output of Svelte itself. Browser modules are placed in `_build/assets/` under the hash names.
6. Collects all modules, their imported code and the framework library into one file. Types TypeScript the Templates Erases SvelteIn the modules `.ts` — assembler. Modules with runes, `.svelte.js` and `.svelte.ts`, Svelte Compile for Server (`compileModule`); in `.svelte.ts` First, remove the types esbuildand a map of the source Svelte is drawn through its map, and the error shows a line in `.svelte.ts`. `@/` Imports mean `src/`. `query`, `paginate`, `page` and `search` project code imported from module `shugo`; it is an integrated assembly module, not a package npm.
7. The import graph that the assembler gives decides which layouts and components depend on the page: those that import `page` or `paginate` from the `shugo` module, themselves or through other modules, for example from `src/lib/`. It also decides which components depend on where they stand: those that take the context of Svelte (`getContext` and neighboring functions), call `$props.id()`, or show a snippet inside `<select>`, themselves or through what they import. A layout that does not depend on the page is calculated once per entire site, and which depends on the layout once per page ([Blocks](blocks.md#layouts)). A component that does not depend on either the page or the location is noted in general: its display with simple parameters and snippets without parameters is calculated once for each set of parameters ([Blocks](blocks.md#shared-components)).
8. Reads the `src/site.html` shell — a document that contains each page — and cuts it in `%shugo.head%` and `%shugo.body%` places by three pieces: to the head, between places, and after the body. Each place should stand exactly once, the head - before the body. No file is available - the framework shell (`skeleton.DefaultShell`) is taken. Pieces are laid in the frame of the site.
9. Collects styles into one file with a hash in the name: `<style>` components, as rewritten by Svelte, so that they act only on elements of its component (class `svelte-…`), CSS files that import code, and Tailwind if installed in the project.

Svelte compiles templates, checks them and writes component styles. esbuild processes TypeScript modules, binds modules and builds a graph of dependencies for the computation stage.

### Evaluation

1. Go opens a transaction in the repository and reads the project database log: what lines have changed since the last update and what were. From the reading table, he finds who has read the changed columns of these lines: pages, common blocks, lists of route pages. If you can't find anyone, JS won't start.
2. Go launches `eval.mjs` in Bun or Node and one message says what is known from past launches: types of readings and questions to the database for which there is already an index.
3. If the route pages are affected, JS lists it again (`getStaticPaths`). New Go pages add to what needs to be calculated, and the missing ones delete.
4. Go asks you to calculate pages and common blocks. JS calculates them one by one, without waiting: Svelte renders a page with all the components that are rendered on-site, and in place of the common block is a label; the snippets that the page gave to the common block are rendered separately; the layouts around the page are common blocks. For a file not in HTML, the function of `GET` of its module is calculated. The resulting HTML JS cuts through the labels on the blocks and cuts out the Svelte ([Blocks](blocks.md#how-html-is-split-into-blocks)) hydration comments from it. Each line from the database records which columns are read. JS asks about the new type of request Go, and he builds an index in the project database.
5. JS sends Go along with readings via a separate channel. What prints the project code does not get into this channel.
6. Go checks each record and compares it to the saved one. Differs — writes along with the storage version, no — skips. Common blocks that records are called for the first time, Go asks to calculate in the next round.
7. At the end, Go removes shared blocks that no one else shows, finds pages where HTML has changed, and completes the transaction. If the entire site is calculated, it still weighs the general components: which 100 has common blocks or more, and each is shown on average in less than one and a half places, the assembly warns about this ([3](../spec/decisions.md#3-which-blocks-are-shared)).

The first build goes the same way: everything is calculated.

### Failed updates

Update is a single transaction of storage. If the project code has fallen, the blocks of record have not formed into one tree or the common block that someone shows has disappeared, the transaction is canceled. The storage remains the same: the server shows the previous site, export does not find changes. The error is printed with the space in the project file.

The database log is not lost. Go erases the strings from it only to the place that is already written in the repository by the last successful update. The next update will encounter the same changes and drop again until the error is corrected. This is how the `TestFailedUpdateChangesNothing` and `TestSharedBlockThatFails` tests are tested.

One subtlety. After changing the data, Go calculates the general blocks that read the changed lines. Such a block may fall: for example, a card of a remote product does not find its goods. If no one shows it at the end of the updatethe pages that showed it are re-computed and no longer call itit’s not an error: the block is deleted along with the rest, which no one shows. If it shows at least one page, the update did not take place.

### Runtime

Described in [Rantime](runtime.md).

### How long does it take

Shop from [examples/catalog](../examples/catalog/README.md) to 100,000 items, 104,202 pages, Bun, 8 cores, 22 November 2026 years:

| What | Time |
| --- | ---: |
| First build, with eight indexes | 14.6 s |
| Update: price of one product | 31 ms |
| Update: column that no one reads, 5,000 has products | 0.11 s |
| Update: header on all pages | 0.12–0.17 s |
| Update: product added, counters in the menu on all pages | 0.49 s |

The update time depends on what has changed, not the size of the site. Details are in [README booth](../examples/catalog/README.md#results).

## Open questions

- Several JS processes for the first build of large sites: pages are divided between processes, time falls almost in proportion to the number of cores.
