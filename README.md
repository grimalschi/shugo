# Shugo

Shugo is a framework for static sites with interactive islands. Pages, mock-ups and components are `.svelte` files, each of which is a component of Svelte 5: `<script>`, Markup, `<style>`. It's taking out their official Svelte parser, and Shugo releases the same HTML that Svelte 559 server. The `*.island.svelte` component is an island: it is hydrated by Svelte in a browser, and HTML is collected by Go without JavaScript on the server for the search filter address. The document around the pages — `<html>`, `<head>`, `<body>` — is written once in the `src/site.html` shell; the files are not in HTML, such as `sitemap.xml`, and provide modules on TypeScript among the pages. The site's data are in the SQLite database of the project, and only read through `query` from the `shugo` module. The project code is executed when the pages are released; the visitor is given a server on Go, which collects HTML from the finished values and does not compute anything.

How to collect and run projects that are finally decided, and what awaits approval, in [specifications](spec/README.md). As Shugo arranged inside - in [docs](docs/README.md), offers - in [docs/proposals](docs/proposals/README.md).

## How it works

```
source files ── compilation (Svelte + Go) ─┬─ eval.mjs: code pages and components
                                      └─ site.json: markup templates with places for values

project Database ──► calculation (Bun or Node): eval.mjs reads strings via query
  (Triggers are          │  records of pages and common blocks, and what each has read
   journal of Changes)       ▼
                       storage (SQLite) ──► server (Go): HTML for each request
                                         └──► export: pages files that have changed
```

1. **Compilation.** The official parser of Svelte parses the templates; it is sewn into Shugo and runs on Bun or Node without the `svelte` package in the project. From each template, two things are obtained: JS, which counts the values of expressions, and the framework - markup with places for these values, according to the rules of Svelte: spaces, nesting elements, component styles. Svelte compiles the islands for the browser and server, and from the server version of Shugo makes the program SSR, which then performs Go.
2. **Calculation.** JS executes the code of pages without waiting: the request to the database responds immediately. Each line that the request returned remembers which columns it read — in the page code, in the template, in functions, in components. Go compares the result to the storage and records only differences. A component called with simple parameters is calculated once and stored in a separate record: pages link to it.
3. **Update.** Triggers in the database write each changed line to the journal. `shugo update` reads the journal, finds who has read the changed columns of these lines, and calculates only them.
4. **Server.** For each query, Go reads a page record and records its shared blocks from a single storage state and collects HTML. For an island that is tied to a search through `request:`, it takes a version of the search index from the same state and substitutes the issue at the request address into the pre-assembled SSR program.
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
| `cmd/inside` | booth program «Shugo inside» |
| `internal/` | framework, by package per stage: `syntax`, `compile`, `jsrt`, `data`, `evaluate`, `store`, `record`, `render`, `server` |
| `test/` | tests that run entire sites through all stages; `test/svelte` are standards from the real Svelte |

## Tests

```sh
go test ./...                                   # all tests; the calculation stage is on Bun
SHUGO_RUNTIME=node go test ./test/              # the Node
go test ./test/ -run xxx -bench .               # speed of page assembly and server response
node examples/catalog/stand.mjs --size 10000    # stand: 21 data change, update and export after each
```

## Results

Store on 100,000 products, 104,202 pages and 100,009 common blocks, Bun, 8 cores:

| What | Time |
| --- | ---: |
| First build, with eight indexes that Shugo builds itself | 14.5 s |
| Update after price change of one product: page and card calculated, HTML changed to 7 pages | 29 ms, the whole team of 36 ms |
| Update after changing column that no one reads, 5,000 products | 0.11 s, nothing calculated |
| Update after changing the cap that is on all pages | 0.12–0.13 s: One record of the common block |
| First export to files | 5.0 s |
| Export after the price change of one product | 8 ms: built by 7 pages |
| Export after changing the header | 8.2–8.5 s: all files rewritten |
| Assembly of the product page by the server from the storage | 58–67 µs |
| Request to server as a whole, 8 clients in the same process | 16–18 thousand per second |

After 21 changes the data, the site updated and exported in steps, matches the file to the file with the build from scratch. Details are in [README booth](examples/catalog/README.md).
