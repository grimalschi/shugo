# Build pipeline

The project is built in stages. Each stage has defined inputs and produces files or records that later stages consume.

## Decisions

- Compilation, data loading and expression evaluation, and runtime page assembly have separate inputs and outputs.
- Change tracking sits between data loading and expression evaluation. Only affected portions are evaluated again. This is required whether Go compares the results between two TypeScript stages or TypeScript does the comparison directly.
- A project change reruns the entire pipeline. A data change triggers an incremental update. On November 21, 2026, the project author confirmed that a code change should reevaluate the whole site. The [Live Reload proposal](proposals/live-reload.md) describes ways to shorten development feedback.
- The original update started a new JavaScript process, loaded all data, and compared the result with the previous build. The November 20, 2026 decisions replaced that design. An update now reads the database journal and evaluates the readers of the changed data. [Change tracking decisions](change-tracking.md#decisions)
- One JavaScript process runs expressions and data-loading code for the whole evaluation. Process startup costs more than rendering an individual page.
- Both Bun and Node must work as JavaScript runtimes.
- The runtime reads persisted results regardless of when or how they were updated.
- SQLite stores the results.
- The November 21, 2026 decision requires a failed update to roll back completely. The store and the outstanding journal entries remain available for the next attempt. [Failed updates](#failed-updates)
- On November 21, 2026, sandboxing was left outside `shugo`. The documented interface is how to run the JavaScript stage in a sandbox.

## Current implementation, awaiting approval

The current pipeline uses the [final change tracking implementation](change-tracking.md#final-implementation). Svelte 5.59.1 compiles and renders the templates. [Shugo and Svelte](svelte.md)

| Stage | Runs in | Inputs | Outputs |
| --- | --- | --- | --- |
| 1. Compilation | Go and Svelte on Bun or Node | Project sources under `src/` | `_build/eval.mjs` for the complete JavaScript bundle, `_build/site.json` for the site skeleton, and `_build/assets/` for stylesheets and island browser modules |
| 2. Evaluation | Bun or Node under Go's control | `eval.mjs`, project database, and database journal | Updates to `_build/store.db` and a report of the changes |
| 3. Runtime | Go | `store.db`, `public/`, `assets/`, and any published search index | HTML on request or files for the pages that changed |

### Compilation

1. Shugo finds `.svelte` files under `src/`. Islands use `*.island.svelte` and enter the build through imports. Shugo calls Svelte 5.59.1's `compile` from `svelte/compiler` with `generate: "server"`, `runes: true`, and `css: "external"`. Compilation runs in Bun or Node, and large projects use multiple worker processes. The `shugo` package installs the pinned `svelte` dependency and sets `SHUGO_SVELTE` to point to it. Otherwise, Shugo searches `node_modules` in the project directory and its ancestors. A different Svelte version fails the build. Shugo reports Svelte's errors and warnings with source locations and suppresses warnings about browser updates in static templates, which render once. A `.shugo` template produces an error explaining how to rename it. Shugo also checks Svelte's AST. `<html>`, `<head>`, `<body>`, and `<!doctype>` belong in the document shell and trigger one diagnostic per template file. Legacy Shugo tags, Svelte 4 slots, `<svelte:options runes={false} />`, and invalid island tags are errors. A parse error for markup inside `{…}` suggests `{#if}` or `{#each}`. The AST also identifies imports of `page` and `paginate`, use of Svelte context and `$props.id()`, snippets rendered within `<select>`, exports of `getStaticPaths`, and island imports.
2. Shugo finds `.ts` and `.js` modules under `src/pages/`. It excludes `.d.ts`, runes modules (`.svelte.ts` and `.svelte.js`), and files or directories whose names start with `_`. Each remaining module produces a file, with its extension in the module name. For example, `sitemap.xml.ts` serves `/sitemap.xml`. A name like `data.ts` lacks an output extension and fails the build. File routes carry the extension and have no layouts. They receive IDs after the templates.
3. The Svelte-generated module puts imports, exports, and the contents of `<script module>` at module scope, including `getStaticPaths`. The component is a function that runs `<script>` and writes HTML. The bundle represents a template as two modules. The module with the `?svelte` suffix is the Svelte-generated component. Other modules import the Shugo wrapper, which re-exports `<script module>` exports and wraps the component in `component(fileNumber, component)`. At each call, that wrapper either renders inline or inserts a shared block reference. [Shared components](blocks.md#shared-components)
4. Compiled templates use `svelte/internal/server`. Project code can import `svelte`, `svelte/store`, `svelte/reactivity`, and other modules that work on the server. The bundler resolves the server runtime from the pinned `svelte` package under the `production` condition, which excludes development-only code. Svelte's own sources can resolve their internal imports by package name without using the project import allowlist. Shugo wraps `{@html}` output in markers to preserve its comments when removing Svelte's hydration comments. [How HTML is split into blocks](blocks.md#how-html-is-split-into-blocks)
5. The same Svelte compiler builds the imported islands' dependency graphs for the server and browser in runes mode. The browser bundle uses the browser runtime from the same package. For islands bound to request data, Shugo compiles an SSR program from Svelte's server output. Evaluation verifies that the program produces the same HTML as Svelte. Browser modules go into `_build/assets/` with hashed filenames.
6. Shugo bundles the modules, their project and npm dependencies, and the framework runtime into one file. A package's `svelte` imports resolve to the version installed by `shugo`. The bundler resolves each package import first so it can report missing packages with the appropriate install command for the project's lockfile. It reports both versions when a package's `peerDependencies` or `dependencies` require an incompatible Svelte version. It rejects imports of package Svelte components into page code, with a diagnostic at the import. It also adds context to errors for unsupported package files, such as native `.node` modules. Every bundled package file is a hashed source input. The project `package.json` and package manager lockfiles are inputs too, so a change such as a Tailwind upgrade triggers recompilation. Importing `+layout.svelte` is an error because layouts are applied by the framework. If the bundler cannot find a type imported as a value, Shugo suggests adding `type` to the import. Svelte erases types from templates, and esbuild erases types from `.ts` modules. Svelte compiles `.svelte.js` and `.svelte.ts` for the server with `compileModule`. For `.svelte.ts`, esbuild erases types first. Shugo composes the source maps so diagnostics point to the original `.svelte.ts` lines. The `@/` alias points to `src/`. The project imports `query`, `paginate`, `page`, and `search` from `shugo`, which the bundler supplies as a built-in module.
7. Shugo uses the bundler's import graph to identify layouts and components that depend on the current page. This includes direct and transitive imports of `page` or `paginate`, such as through a module under `src/lib/`. A component depends on its call site if it or one of its imports reads Svelte context with `getContext` or related functions, calls `$props.id()`, or renders a snippet inside `<select>`. A layout independent of the page is evaluated once for the whole site. A page-dependent layout is evaluated once per page. Components independent of both the page and call site are eligible for sharing. With simple props and snippets that take no arguments, they are evaluated once per set of props. [Layouts](blocks.md#layouts) · [Shared components](blocks.md#shared-components)
8. Shugo reads `src/site.html` and splits it at `%shugo.head%` and `%shugo.body%`. The three fragments before, between, and after the placeholders go into the site skeleton. Each placeholder must appear once, with the head first. If the file is absent, Shugo uses `skeleton.DefaultShell`.
9. Shugo bundles CSS into one stylesheet with a hashed filename. It includes component `<style>` blocks rewritten by Svelte to scope them with `svelte-…` classes, CSS imported by project code, and Tailwind when the project installs it.

Svelte compiles and validates the templates and produces component CSS. esbuild handles TypeScript modules, bundles imports, and produces the dependency graph used during evaluation.

### Evaluation

1. Go opens a store transaction and reads the project database journal for rows changed since the last update, including their previous values. The recorded reads identify the pages, shared blocks, and route page lists that used changed columns from those rows. If there are no affected readers, Go skips JavaScript evaluation.
2. Go starts `eval.mjs` in Bun or Node. In one message it sends the read shapes and query indexes known from earlier runs. The framework library runs first and supplies CommonJS packages with `require`, `__filename`, and `__dirname` for the bundle, which a Node ES module otherwise lacks. An exception during module or package initialization is sent to Go like a page error, with locations in project or package source files.
3. If a route's page list is affected, JavaScript runs `getStaticPaths` again. Go schedules new pages for evaluation and removes pages that disappeared from the list.
4. Go requests pages and shared blocks. JavaScript evaluates them one at a time without waiting. Svelte renders the page and its inline components, writes markers for shared blocks, and renders any snippets supplied by the page to shared blocks separately. Surrounding layouts are also shared blocks. For a file route, JavaScript evaluates the module's `GET` function. JavaScript splits the generated HTML into blocks at the markers and strips Svelte's hydration comments from static markup. Each returned database row records which of its columns were read. When JavaScript encounters a new query shape that needs an index, Go builds the index in the project database. [How HTML is split into blocks](blocks.md#how-html-is-split-into-blocks)
5. JavaScript sends completed records and their reads to Go over a dedicated channel. Output printed by project code stays separate.
6. Go validates each record and compares it with the stored version. It writes changed records with the new store version and skips identical ones. It requests any newly referenced shared blocks in the next round.
7. Go removes unreferenced shared blocks, identifies pages whose HTML changed, and commits the transaction. After a full-site evaluation, the build also checks whether components benefit from sharing. It warns if a component produces at least 100 shared blocks and each appears in fewer than 1.5 places on average. [Decision 3](../spec/decisions.md#3-which-blocks-are-shared)

The first build follows the same path and evaluates everything.

### Failed updates

The update uses a single store transaction. It rolls back if project code fails, a record's blocks do not form one tree, or a referenced shared block is missing. The server continues serving the previous site, and export sees no changes. The diagnostic includes the location in the project source.

The database journal is retained. Go removes entries only up to the position recorded by the last successful update. The next run sees the same changes again and fails until the error is fixed. `TestFailedUpdateChangesNothing` and `TestSharedBlockThatFails` exercise this behavior.

A shared block can fail after a data change, such as a card for a deleted product. If reevaluation removes every reference to that block, the update can commit and delete the block along with other unused blocks. If any page still references it, the update rolls back.

### Runtime

[Serving and exporting pages](runtime.md)

### Benchmarks

The November 22, 2026 benchmark ran with Bun on eight cores and used a shop with 100,000 products and 104,202 pages. [Catalog example](../examples/catalog/README.md)

| Operation | Time |
| --- | ---: |
| Initial build, including eight indexes | 14.6 s |
| Update one product price | 31 ms |
| Update an unread column on 5,000 products | 0.11 s |
| Update the header shown on every page | 0.12–0.17 s |
| Add a product and update the navigation counts shown on every page | 0.49 s |

Update time follows the work caused by each change. [Detailed benchmark results](../examples/catalog/README.md#results)

## Open questions

- The first build of a large site could split pages among several JavaScript processes. The proposed speedup is approximately proportional to the number of cores.
