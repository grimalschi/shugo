# Shugo

Shugo is a framework for static site generation. Pages are written in `.shugo` files similar to Astro: code on TypeScript and markup. The project code is executed once, when assembled; the page visitor is given a server on Go, which collects HTML from ready values and does not calculate anything.

This is the first working version. What is finally decided in it, and what awaits approval, is written in [specifications](spec/README.md).

## How it works

```
source files ── compilation (Go) ─┬─ eval.mjs ── calculation (Bun or Node) ── values ─┐
                              └─ site.json: markup with places for values ────────┴─ rantime (Go) ── HTML
```

1. **Compilation.** Two things are obtained from each template: JS, which counts the values of expressions, and the program is a markup with places for these values.
2. **Calculation.** JS loads data and counts all pages. Go compares the result with the saved one and writes only differences in SQLite.
3. **Rantime.** The server collects a page from the program and values: about 7 µs to the store page.

After changing the data, only what shows them changes. On the stand in 100,000 goods, a change in the price of one product rewrites one page and one card, and a change in the cap is one record, although the cap is on each page.

## Try it

You need Go 1.27 and Bun or Node 24.

```sh
go build -o bin/shugo ./cmd/shugo
bin/shugo dev examples/basic          # http://localhost:4321
```

```sh
bin/shugo build examples/basic        # compilation and Calculation
bin/shugo update examples/basic       # calculation again after changing the data
bin/shugo serve examples/basic        # server
bin/shugo export examples/basic       # site to files: examples/basic/_build/dist
```

## Repository layout

| Folder | What's up? |
| --- | --- |
| [spec/](spec/README.md) | specification: solutions, device, suggestions, open questions |
| [docs/authoring.md](docs/authoring.md) | a memo for someone who writes a site on Shugo (in English, for agents) |
| [examples/basic](examples/basic) | small site: layout, components, slots, dynamic pages |
| [examples/catalog](examples/catalog/README.md) | shop of any size and point update stand |
| [examples/tailwind](examples/tailwind) | styles through Tailwind |
| `cmd/shugo` | `shugo` |
| `internal/` | framework, by package per stage: `syntax`, `compile`, `jsrt`, `evaluate`, `store`, `render`, `server` |
| `test/` | tests that run entire sites through all stages |

## Tests

```sh
go test ./...                             # all tests; the calculation stage is on Bun
SHUGO_RUNTIME=node go test ./test/        # the Node
go test ./test/ -run xxx -bench .         # speed of page assembly and server response
node examples/catalog/stand.mjs --size 10000    # stand: 20 kinds of data changes
```
