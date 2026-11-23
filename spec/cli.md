# Command line and development mode

## Decisions

- Development mode rebuilds after file changes and reloads open pages.
- A project source change reevaluates the site. True Live Reload is a separate [proposal](../docs/proposals/live-reload.md), as decided on November 21, 2026.
- CLI messages and the specification use English so agents and readers can search for the same terms. This was decided on November 21, 2026.
- The proposed `shugo init` command creates a sample project with `tsconfig.json`, editor types, and an authoring guide. It has been planned since November 21, 2026.
- The `shugo check` command checks types and component props. The November 22 decision used bundled tooling. The November 23 decision made `svelte-check` and TypeScript dependencies of the `shugo` npm package.
- The November 22 distribution plan covered GitHub releases, npm packages with platform binaries, and Homebrew. The November 23 decision standardized on an npm project dependency that ships with the module types. The packages are built locally. See [the Shugo npm package](#the-shugo-npm-package).

## Implemented, pending approval

```text
shugo build  [directory]    compile and evaluate all pages
shugo update [directory]    reevaluate content affected by data or project changes
shugo serve  [directory]    serve pages on request
shugo export [directory]    write site files, keeping later exports incremental
shugo dev    [directory]    update, serve, and rebuild when files change
```

The directory is the site project. It defaults to the current directory.

| Option | Behavior |
| --- | --- |
| `-json` | Print a JSON report for `build`, `update`, or `export`. |
| `-check` | With `update`, reevaluate everything and report which pages changed despite reading no changed rows. |
| `-runtime bun` or `-runtime node` | Choose the JavaScript runtime for Svelte and project code in `build`, `update`, and `dev`. The fallback order is the `runtime` setting in `shugo.json`, then Bun if installed, then Node. |
| `-addr address` | Set the listen address for `serve` and `dev`. The default is `localhost:4321`. |
| `-out directory` | Set the export directory. The default is `_build/dist`. |

A command reports options it cannot use as errors. See [decision 27](decisions.md#27-commands).

### Command behavior

| Command | Compiles | Evaluates | Output | Watches files |
| --- | --- | --- | --- | --- |
| `build` | Always | All pages | Store records | No |
| `update` | On project source changes | Affected pages and blocks | Store records | No |
| `serve` | No | No | Pages from the store | No |
| `export` | No | No | Files | No |
| `dev` | On project source changes | Affected pages and blocks | Pages from the store | Yes, with browser reload |

`dev` combines `update` and `serve` with file watching. It runs an update after a change and asks open browser tabs to reload. The [Live Reload proposal](../docs/proposals/live-reload.md) explores reevaluating individual pages on request.

### Choosing a command

- Run `shugo update` when site data changes. It reuses the compiled JavaScript and evaluates the pages and blocks that read the changed rows.
- When project sources change, `shugo update` recompiles and reevaluates the site. The build records hashes of the input files and the Svelte version to identify stale output.
- `shugo build` reevaluates the entire site. Use it when pages depend on data outside the project database.
- `shugo serve` sees each completed update on the next request.

### Reports

This November 22, 2026 benchmark changed one product price in a shop with 100,000 products.

```text
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

One page record changed. Seven pages have different HTML because the product card is a shared block. The six other pages that display it keep their own records.

JSON includes the same counts and step durations in milliseconds.

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

- `data` reports changed rows, route lists reread, and the change log position processed. A true `whole` means the change log could not identify individual changes and the site was reevaluated.
- `render` counts the pages whose final HTML changed. It includes pages displaying a changed shared block. The corresponding names can be used to invalidate an upstream cache.
- Each name list is limited to one thousand entries.
- `rebuilt` means Shugo recompiled and reevaluated the project from scratch.
- `shadowed` counts dynamic addresses taken by pages with literal filenames.

### Errors

Shugo reports project errors at the source location and includes the relevant line. Svelte compiler errors and warnings include Svelte's message and documentation link.

```text
error: src/pages/about.svelte:4:15: `</div>` attempted to close an element that was not open https://svelte.dev/e/element_invalid_closing_tag
     3 | <section>
     4 |   <p>words</p></div>
       |               ^
```

Warnings are printed on one line during compilation and the build continues.

```text
warning: src/pages/about.svelte:3:1: `<img>` element should have an alt attribute https://svelte.dev/e/a11y_missing_attribute
```

Evaluation errors name the page and point back to the project source.

```text
error: Cannot read properties of undefined (reading 'title')
  while computing the page /about (src/pages/about.svelte)
  at src/pages/about.svelte:5:21
     4 | const first = products[10]
     5 | const title = first.title.toUpperCase()
       |                     ^
```

Node reports the precise location. Bun sometimes reports a neighboring line because it transforms the code before running it.

### Development mode

`shugo dev` watches the project except for `_build`, `node_modules`, and hidden files. It applies the same rules as `shugo update`. Source changes recompile and reevaluate the site. Database changes reevaluate the readers of changed rows. Other file changes reload open pages without reevaluation. See the [Live Reload proposal](../docs/proposals/live-reload.md) for potential improvements.

### The Shugo npm package

Shugo is installed as a project dependency, like Vite or SvelteKit. After publication, install and run it using the package manager.

```sh
bun add shugo                  # or npm install shugo
bunx shugo build               # or npx shugo build
```

The `shugo` package provides the CLI, module types, and TypeScript configuration. It depends on Svelte 5.59.1, `svelte-check` 4.7.8, and TypeScript 5.9.3. Packages such as `@shugo/linux-x64` and `@shugo/darwin-arm64` carry the platform executables. The package manager installs the suitable one, and the CLI forwards arguments to it.

A project's `tsconfig.json` can extend the package configuration.

```json
{ "extends": "shugo/tsconfig.json" }
```

The configuration maps `@/` to `src/` and checks `src/` using the same conventions as the build. It excludes `_build/` and `node_modules/`. The package supplies `shugo` types, and its pinned Svelte dependency supplies types for `svelte`, `svelte/store`, and runes. The CLI passes the dependency paths to the executable through `SHUGO_SVELTE` and `SHUGO_SVELTE_CHECK`. Tests on November 23, 2026 checked TypeScript 5.9 and 7.0 and `svelte-check` against valid and invalid uses of `query`, `page`, and `svelte/store`.

See [npm packages](../npm/README.md) for packaging and publication.

## Open questions

- Publishing requires an npm organization for the `@shugo` scope and a project license in place of `UNLICENSED`.
