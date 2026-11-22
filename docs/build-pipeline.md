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

Below is how the stages are arranged now: in the final version ([Change tracking](change-tracking.md#final-implementation)), with templates in the format of components Svelte 5.

| Stage | Who performs | What gets | What gives |
| --- | --- | --- | --- |
| 1. Compilation | Go and Svelte on Bun or Node | project Sources from `src/` | `_build/eval.mjs` - the entire JS project in one file; `_build/site.json` - the project framework; `_build/assets/` - island styles and browser modules |
| 2. Calculation | Bun or Node running Go | `eval.mjs`, project database and its log | changes in `_build/store.db` and report on them |
| 3. Rantime | Go | `store.db`, `public/`, `assets/`, published search index if available | HTML pages on request or changed page files |

### Compilation

1. Finds the usual `.svelte` files in `src/` — the `*.island.svelte` islands are entered through imports — and parses each with the official Svelte 5.59.1, `svelte/compiler.parse`, in one Bun or Node process for assembly. Parser is sewn into Shugo: the usual project package `svelte` is not needed. The adapter converts the tree Svelte to the nodes of Shugo and saves the places in the source. What Svelte does not accept is a bug with a place in the file; the `.shugo` template is an error with a hint on how to rename it; what Svelte warns about, for example, the tag that the browser would rearrange if a block was standing between tags is a warning. The template that `<html>`, `<head>`, `<body>` or `<!doctype>` write is a bug, one per file: the document around the pages is written in the shell.
2. Finds `.ts` and `.js` modules in `src/pages/`, except for `.d.ts` and those whose name or folder starts with `_`. Each gives a file not in HTML; its type is in the name: `sitemap.xml.ts` gives `/sitemap.xml`. The module without the type in the name, `data.ts`, is an error. The route of such a module carries a file extension, it has no layouts. Module numbers come after template numbers.
3. For each template, JS is written. Imports, exports and all `<script module>` — `getStaticPaths` — remain at the top of the module. `<script>` and template become one normal function, without `async`, which is performed for each display of the component. The template is a regular `if` and `for` and the calls that record the meaning of each expression. `$props()` gives the parameters and snippers that the caller gave.
4. For each template, a framework is written: markup of the template with places where values are substituted. Separate parts in it are the snippets of the template and its `<svelte:head>`. The spaces between tags are removed and compressed here, according to the rules of Svelte. There are no expressions or project code in the frame. The `src/pages/` module framework is one operation: write as there is text that will return it to `GET`.
5. For imported `*.island.svelte`, Svelte compiles server and browser versions of their graph; for this, the project needs a package of `svelte@5.59.1`. From the server version of Shugo builds the SSR program of the island, tied to the request data, and at the calculation stage checks it HTML with the output of Svelte itself. Browser modules are placed in `_build/assets/` under the hash names.
6. Collects all modules, their imported code and the framework library in one file. TypeScript types are erased here. `@/` in imports means `src/`. `query`, `paginate` and `page` project code is imported from the `shugo` module, `search` from `shugo:search`; these are built-in assembly modules, not npm packages.
7. The import graph that the assembler gives decides which layouts and components depend on the page: those that import `page` or `paginate` from the `shugo` module, themselves or through other modules, for example from `src/lib/`. The rest show the same thing on any page and are stored in common blocks: the layout is calculated once for the entire site, a component with simple parameters - once for each set of them ([Blocks](blocks.md#layouts)).
8. Reads the `src/site.html` shell — a document that contains each page — and cuts it in `%shugo.head%` and `%shugo.body%` places by three pieces: to the head, between places, and after the body. Each place should stand exactly once, the head - before the body. No file is available - the framework shell (`skeleton.DefaultShell`) is taken. Pieces are laid in the frame of the site.
9. Collects styles into one file with a hash in the name: `<style>` components, rewritten to act only on elements of its component (class `svelte-…`, as in Svelte), CSS files that import code, and Tailwind if it is installed in the project.

The Svelte parser defines expression boundaries and checks their syntax. esbuild processes TypeScript, binds modules, and builds a graph of dependencies for the computation stage.

### Evaluation

1. Go opens a transaction in the repository and reads the project database log: what lines have changed since the last update and what were. From the reading table, he finds who has read the changed columns of these lines: pages, common blocks, lists of route pages. If you can't find anyone, JS won't start.
2. Go launches `eval.mjs` in Bun or Node and one message says what is known from past launches: types of readings and questions to the database for which there is already an index.
3. If the route pages are affected, JS lists it again (`getStaticPaths`). New Go pages add to what needs to be calculated, and the missing ones delete.
4. Go asks you to calculate pages and common blocks. JS computes them one by one, without waiting: code from `<script>`, template, components in it, layouts around the page; for a file not in HTML, the function of `GET` is its module. Each line from the database records which columns are read. JS asks about the new type of request Go, and he builds an index in the project database.
5. JS sends Go along with readings via a separate channel. What prints the project code does not get into this channel.
6. Go checks each record and compares it to the saved one. Differs — writes along with the storage version, no — skips. Common blocks that records are called for the first time, Go asks to calculate in the next round.
7. At the end, Go removes shared blocks that no one else shows, finds pages where HTML has changed, and completes the transaction. If the entire site is calculated, it still weighs the general components: which 100 has common blocks or more, and each is shown on average in less than one and a half places, the assembly warns about this ([3](../spec/decisions.md#3-which-blocks-are-shared)).

The first build goes the same way: everything is calculated.

### Failed updates

Update is a single transaction of storage. If the project code has fallen, the record has not matched the template frame or a shared block that someone is showing has disappeared, the transaction is canceled. The storage remains the same: the server shows the previous site, export does not find changes. The error is printed with the space in the project file.

The database log is not lost. Go erases the strings from it only to the place that is already written in the repository by the last successful update. The next update will encounter the same changes and drop again until the error is corrected. This is how the `TestFailedUpdateChangesNothing` and `TestSharedBlockThatFails` tests are tested.

One subtlety. After changing the data, Go calculates the general blocks that read the changed lines. Such a block may fall: for example, a card of a remote product does not find its goods. If no one shows it at the end of the updatethe pages that showed it are re-computed and no longer call itit’s not an error: the block is deleted along with the rest, which no one shows. If it shows at least one page, the update did not take place.

### Runtime

Described in [Rantime](runtime.md).

### How long does it take

Shop from [examples/catalog](../examples/catalog/README.md) to 100,000 items, 104,202 pages, Bun, 8 cores:

| What | Time |
| --- | ---: |
| First build, with eight indexes | 14.1 s |
| Update: price of one product | 28 ms |
| Update: column that no one reads, 5,000 has products | 0.10 s |
| Update: header on all pages | 0.11–0.12 s |
| Update: product added, counters in the menu on all pages | 0.48 s |

The update time depends on what has changed, not the size of the site. Details are in [README booth](../examples/catalog/README.md#results).

## Open questions

- Several JS processes for the first build of large sites: pages are divided between processes, time falls almost in proportion to the number of cores.
