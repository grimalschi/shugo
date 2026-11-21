# Shugo

Shugo - framework for static sites. Pages are written in `.shugo` files similar to Astro: code on TypeScript and markup. The data of the site is in the database of SQLite project, and pages read them only through `Shugo.query`. Project code is executed when pages are calculated; the page visitor is given a server on Go that collects HTML from ready values and does not calculate anything.

What is finally decided and what awaits approval is recorded in [specifications](spec/README.md).

## How it works

```
source files ── compilation (Go) ─┬─ eval.mjs: code pages and components
                              └─ site.json: markup templates with places for values

project Database ──► calculation (Bun or Node): eval.mjs reads lines via Shugo.query
  (Triggers are          │  records of pages and common blocks, and what each has read
   journal of Changes)       ▼
                       storage (SQLite) ──► server (Go): HTML for each request
                                         └──► export: pages files that have changed
```

1. **Compilation.** Two things are obtained from each template: JS, which counts the values of expressions, and the program is a markup with places for these values.
2. **Calculation.** JS executes the code of pages without waiting: the request to the database responds immediately. Each line that the request returned remembers which columns it read — in the page code, in the template, in functions, in components. Go compares the result to the storage and records only differences. A component called with simple parameters is calculated once and stored in a separate record: pages link to it.
3. **Update.** Triggers in the database write each changed line to the journal. `shugo update` reads the journal, finds who has read the changed columns of these lines, and calculates only them.
4. **Server.** There are no copies of the site in memory. For each request, the server reads from the storage the page record and the records of common blocks that it shows - one search per record, all from one state - and collects HTML.
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
| [spec/](spec/README.md) | specification: solutions, device, suggestions, open questions |
| [docs/authoring.md](docs/authoring.md) | a memo for someone who writes a site on Shugo (in English, for agents) |
| [examples/basic](examples/basic) | small site without a database: layout, components, slots, dynamic pages |
| [examples/catalog](examples/catalog/README.md) | store of any size with database and point update stand |
| [examples/inside](examples/inside/README.md) | Shugo booth inside: all intermediate formats and connections in a small store |
| [examples/tailwind](examples/tailwind) | styles through Tailwind |
| `cmd/shugo` | `shugo` |
| `cmd/inside` | booth program «Shugo inside» |
| `internal/` | framework, by package per stage: `syntax`, `compile`, `jsrt`, `data`, `evaluate`, `store`, `record`, `render`, `server` |
| `test/` | tests that run entire sites through all stages |

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
| First build, with eight indexes that Shugo builds itself | 13.7–14.1 s |
| Update after price change of one product: page and card calculated, HTML changed to 7 pages | 28–32 ms, the whole team of 36–41 ms |
| Update after changing column that no one reads, 5,000 products | 0.10–0.11 s, nothing calculated |
| Update after changing the cap that is on all pages | 0.11–0.12 s: One record of the common block |
| First export to files | 7.1–8.5 s |
| Export after the price change of one product | 7–8 ms: built by 7 pages |
| Export after changing the header | 8.1–8.7 s: all files rewritten |
| Assembly of the product page by the server from the storage | 58–67 µs |
| Request to server as a whole, 8 clients in the same process | about 17–18 thousand per second |

After 21 changes the data, the site updated and exported in steps, matches the file to the file with the build from scratch. Details are in [README booth](examples/catalog/README.md).
