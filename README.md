# Shugo

Shugo - framework for static sites with interactive islands. Pages, layouts and components are `.svelte` files, and each is a component of Svelte 5: `<script>`, markup, `<style>`. Compiles and renders their Svelte 5.59.1 sewn into Shugo; comments hydration Shugo of HTML cuts: they are only needed by the islands. The `*.island.svelte` component is an island: Svelte hydrates it in the browser, and HTML collects Go for the search filter address, without JavaScript on the server. The document around the pages — `<html>`, `<head>`, `<body>` — is written once, in the `src/site.html` shell; files not in HTML, such as `sitemap.xml`, give modules on TypeScript among the pages. The data of the site is in the database of the SQLite project, and the pages read them only through `query` from the `shugo` module. Project code is executed when pages are calculated; the visitor page is given a server on Go, which collects it from ready-made pieces of HTML and does not calculate anything.

How to collect and run projects that are finally decided, and what awaits approval, in [specifications](spec/README.md). As Shugo arranged inside - in [docs](docs/README.md), offers - in [docs/proposals](docs/proposals/README.md).

## How it works

```
source files ── compilation (Svelte + Go) ─┬─ eval.mjs: pages and components compiled by Svelte
                                      └─ site.json: routes, page wrappers, island programs

project Database ──► calculation (Bun or Node): eval.mjs reads strings via query
  (Triggers are          │  HTML pages and shared blocks, and what everyone has read
   journal of Changes)       ▼
                       storage (SQLite) ──► server (Go): HTML for each request
                                         └──► export: pages files that have changed
```

1. **Compilation.** Compiler Svelte compiles templates for the server; it is sewn into Shugo along with the server and browser parts of Svelte and runs on Bun or Node without the `svelte` package in the project. Shugo wraps each component with its own function, which decides whether to render it on-site or as a shared block. It compiles islands for the browser and server; from the server version of the island, which is linked to the query, Shugo makes the program SSR, which then performs Go.
2. **Calculation.** Svelte renders pages without waiting: the database request responds immediately. Each line that the request returned remembers which columns it read — in the page code, in the template, in functions, in components. A component that is independent of the page, or the place where it stands, and called with simple parameters, is rendered once per set and stored separately; layouts are also stored separately. The page stores holes in their place. JS cuts HTML pages into such blocks and removes hydration comments; Go compares the result to storage and records only differences.
3. **Update.** Triggers in the database write each changed line to the journal. `shugo update` reads the journal, finds who has read the changed columns of these lines, and calculates only them.
4. **Server.** For each query, Go reads the page record and records its shared blocks from a single storage state and sews them together with HTML. For an island that is tied to a search through `request:`, it takes a version of the search index from the same state and substitutes the issue at the request address into the pre-assembled SSR program.
5. **Exports.** `shugo export` writes the site to files. After the first time, only pages whose HTML has changed are deleted.

## Try it

You need Go 1.27 and Bun or Node 24.

```sh
go build -o bin/shugo ./cmd/shugo
bin/shugo dev examples/basic          # http://localhost:4321
```

```sh
bin/shugo build examples/basic        # compilation and calculation of all pages
bin/shugo update examples/basic       # find out what changes have occurred.
bin/shugo serve examples/basic        # server
bin/shugo export examples/basic       # site to files: examples/basic/_build/dist
```

The Site Project Shugo the Package npm: `bun add shugo`, then `bunx shugo build`. With the package come and module types `shugo`: `tsconfig.json` Project one line `{ "extends": "shugo/tsconfig.json" }`. The packages collect `go run npm/make.go -version 0.1.0`They have not been published yet ([Command line](spec/cli.md#the-shugo-npm-package)).

## Repository layout

| Folder | What's up? |
| --- | --- |
| [spec/](spec/README.md) | specification: how to collect and run projects, solutions, open questions |
| [spec/authoring.md](spec/authoring.md) | a memo for someone who writes a site on Shugo (in English, for agents) |
| [docs/](docs/README.md) | how Shugo is arranged inside; [docs/proposals](docs/proposals/README.md) offers |
| [examples/basic](examples/basic) | small site without database: `src/site.html` shell, layout, components, snippets, dynamic pages, `sitemap.xml` and `robots.txt` from modules |
| [examples/catalog](examples/catalog/README.md) | store of any size with database and point update stand |
| [examples/inside](examples/inside/README.md) | Shugo booth inside: all intermediate formats and connections in a small store |
| [examples/search](examples/search/README.md) | Svelte islands: search by two tables with filters in the address and two islands without searching on one page |
| [examples/tailwind](examples/tailwind) | styles through Tailwind |
| `cmd/shugo` | `shugo` |
| [npm/](npm/README.md) | npm `shugo` package, through which Shugo is put into the project: command, `shugo` module types, TypeScript settings; and assembly of packages with a program for each system |
| `cmd/inside` | booth program «Shugo inside» |
| `internal/` | framework, by package per stage: `svelte` (Sweet) Svelte), `compile`, `jsrt`, `data`, `evaluate`, `store`, `record`, `render`, `server` |
| `test/` | tests that run entire sites through all stages; `test/svelte` is the test task and HTML gives them Svelte |

## Tests

```sh
go test ./...                                   # all tests; the calculation stage is on Bun
SHUGO_RUNTIME=node go test ./test/              # the Node
go test ./test/ -run xxx -bench .               # speed of page assembly and server response
node examples/catalog/stand.mjs --size 10000    # stand: 21 data change, update and export after each
```

## Results

Store on 100,000 products, 104,202 pages and 100,009 common blocks, Bun, 8 cores, 22 November 2026 year:

| What | Time |
| --- | ---: |
| First build, with eight indexes that Shugo builds itself | 14.6 s |
| Update after price change of one product: page and card calculated, HTML changed to 7 pages | 31 ms, the whole team of 39 ms |
| Update after changing column that no one reads, 5,000 products | 0.11 s, nothing calculated |
| Update after changing the cap that is on all pages | 0.12–0.17 s: One record of the common block |
| First export to files | 3.5 s |
| Export after the price change of one product | 7 ms: built by 7 pages |
| Export after changing the header | 4.4–4.6 s: all files rewritten |
| Assembly of the product page by the server from the storage | 64 µs |
| Request to server as a whole, 8 clients in the same process | about 16 thousand per second |

After 21 changes the data, the site updated and exported in steps, matches the file to the file with the build from scratch. Details are in [README booth](examples/catalog/README.md).
