# Shugo

Shugo is a framework for static sites with interactive islands. Pages, layouts, and components are Svelte 5 components with a `<script>`, markup, and a `<style>` block. Svelte 5.59.1 compiles and renders them as a dependency of the `shugo` package. Shugo strips hydration markers from the HTML outside islands.

A `*.island.svelte` component is an island. Svelte hydrates it in the browser. For URLs with search filters, Go renders the island's HTML without running JavaScript on the server. Write the document shell once in `src/site.html`. For non-HTML files such as `sitemap.xml`, put TypeScript modules alongside your pages.

Site data lives in a project SQLite database. Pages read it through `query` from the `shugo` module during evaluation. A Go server serves each request by assembling HTML that Shugo has already generated.

The [specification](spec/README.md) covers project authoring, accepted decisions, and open questions. The [architecture docs](docs/README.md) describe the internals, and [proposals](docs/proposals/README.md) track potential changes.

## How it works

```text
Source files ── Compilation (Svelte + Go) ─┬─ eval.mjs: compiled pages and components
                                         └─ site.json: routes, document shell, island programs

Project database ──► Evaluation (Bun or Node): eval.mjs reads rows through query
  (triggers log              │  page and shared-block HTML, plus dependency reads
   changes)                  ▼
                          Store (SQLite) ──► Server (Go): HTML for each request
                                         └──► Export: files for changed pages
```

1. **Compilation.** The Svelte compiler compiles templates for Bun or Node. The `shugo` package pins the Svelte version. Shugo wraps each component to decide whether to render it inline or as a shared block. It compiles islands for both browser and server. Shugo turns the server build of a request-bound island into an SSR program that Go can execute.
2. **Evaluation.** Page database queries return synchronously. Shugo tracks which columns the page, its template, and its components or functions read from each result row. A component with simple props that depends on neither the page nor its placement is rendered once per prop combination and stored in a separate record. Layouts are stored separately too. Page records contain placeholders where those blocks belong. JavaScript splits the HTML at these boundaries and strips hydration markers. Go compares the results with the store and writes the changes.
3. **Updates.** Database triggers log changed rows. `shugo update` reads the log, looks up which pages and shared blocks read the changed columns of those rows, and reevaluates them.
4. **Serving.** Go reads a page and its shared blocks from the same store snapshot and assembles their HTML. For an island bound to search with `request:`, Go reads the search index from that snapshot and feeds the results for the request URL into the precompiled SSR program.
5. **Export.** `shugo export` writes the site to disk. Subsequent exports write pages whose HTML changed and remove files for deleted pages.

## Try it

Install Go 1.27 and Bun or Node 24. Run `bun install` from the repository root to install the same Svelte, `svelte-check`, and TypeScript versions the `shugo` package uses.

```sh
bun install
go build -o bin/shugo ./cmd/shugo
bin/shugo dev examples/basic          # http://localhost:4321
```

```sh
bin/shugo build examples/basic        # compile and evaluate all pages
bin/shugo update examples/basic       # reevaluate pages and blocks affected by changes
bin/shugo serve examples/basic        # serve the site
bin/shugo export examples/basic       # write files to examples/basic/_build/dist
```

Once the npm packages are published, a site project can run `bun add shugo` and then `bunx shugo build`. The package includes types for the `shugo` module. The project's `tsconfig.json` can be as short as `{ "extends": "shugo/tsconfig.json" }`. Build the packages with `go run npm/make.go -version 0.1.0`. The [command-line reference](spec/cli.md) covers the package layout and installation.

## Repository layout

| Path | Contents |
| --- | --- |
| [spec/](spec/README.md) | Project setup, decisions, and open questions |
| [spec/authoring.md](spec/authoring.md) | Quick reference for building a Shugo site |
| [docs/](docs/README.md) | Architecture documentation and [proposals](docs/proposals/README.md) |
| [examples/basic](examples/basic) | A small site without a database, with a document shell, a layout, components, snippets, dynamic pages, and modules for `sitemap.xml` and `robots.txt` |
| [examples/catalog](examples/catalog/README.md) | A database-backed shop and benchmark for incremental updates |
| [examples/inside](examples/inside/README.md) | A small shop that demonstrates the intermediate formats and how they fit together |
| [examples/search](examples/search/README.md) | Svelte islands with URL-based filters for searches on two tables, plus two non-search islands on the same page |
| [examples/tailwind](examples/tailwind) | Styling with Tailwind |
| `cmd/shugo` | The `shugo` CLI |
| [npm/](npm/README.md) | The npm package, TypeScript types and configuration, and per-platform binary packaging |
| `cmd/inside` | The program that generates the internals report |
| `internal/` | Framework packages, including `svelte`, `compile`, `jsrt`, `data`, `evaluate`, `store`, `record`, `render`, and `server` |
| `test/` | End-to-end tests and the Svelte fixtures and expected HTML in `test/svelte` |

## Tests and benchmarks

```sh
bun install                                     # install Svelte, svelte-check, and TypeScript
go test ./...                                   # run all tests with Bun for page evaluation
SHUGO_RUNTIME=node go test ./test/              # run the integration tests with Node
go test ./test/ -run xxx -bench .               # benchmark page assembly and serving
node examples/catalog/stand.mjs --size 10000    # run 21 data changes, updating and exporting after each
```

## Benchmark results

The catalog benchmark ran with Bun on eight cores on November 22, 2026. The shop had 100,000 products, 104,202 pages, and 100,009 shared blocks.

| Operation | Time |
| --- | ---: |
| First build, including eight automatically created indexes | 14.6 s |
| Change one product price | 31 ms for evaluation, 39 ms end to end. One page and one product card reevaluated, and seven pages changed. |
| Change an unread column for 5,000 products | 0.11 s, with no reevaluation |
| Update the header shown on every page | 0.12–0.17 s, with one shared-block record written |
| First export | 3.5 s |
| Export after a product price change | 7 ms for seven pages |
| Export after a header change | 4.4–4.6 s to rewrite all pages |
| Assemble a product page from the store | 64 µs |
| Serve HTTP requests with eight in-process clients | About 16,000 requests per second |

After 21 data changes, the incrementally updated export matched a clean build byte for byte. The [catalog benchmark](examples/catalog/README.md) has the full results.
