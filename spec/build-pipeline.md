# Build pipeline

The project is being built in stages. Each stage has its own entrance and its own result. The next stage uses only the results of the previous ones.

## Decisions

- The work is divided into stages with a clear boundary: compilation of the project, work with data and expressions, assembly of pages in rantime.
- Between loading data and calculating expressions is the analysis of changes: expressions are calculated only for changed areas. Analysis is mandatory for any device stages. The choice is only in who compares: Go between two stages on TS or TS itself.
- When the project is changed, the entire chain starts from scratch. When data is changed, the update is point-by-point.
- For now, with each update, the JS process starts again and loads all the data; after that, the result is compared to the past.
- Expressions and data code perform one JS process, not a page-to-page process: starting a process is more expensive than calculating a page.
- JS is performed by Bun and Node, both must work.
- Rantime always takes meanings from the saved and doesn't know how or when they were updated.
- The results are stored in SQLite.

## Made in the first version, waiting for approval

**The author of the project rejected this device 20 November 2026 year:** first version calculates everything and compares the result, but you need to compare the data and calculate only the changed. In return - [proposal](change-tracking.md). Below is described how the first version works now.

Stage three ([1](decisions.md#1-data-loading-and-expression-are-counted-together-go-compares-the-result)).

| Stage | Who performs | What gets | What gives |
| --- | --- | --- | --- |
| 1. Compilation | Go | project Sources | `_build/eval.mjs` — the entire JS project in one file; `_build/site.json` — project program; `_build/assets/` — styles |
| 2. Calculation | Bun or Node running Go | `eval.mjs`, data from project sources | changes in `_build/store.db` and report on them |
| 3. Rantime | Go | `store.db`, `public/`, `assets/` | HTML pages on request or the whole site in files |

### Compilation

1. Finds all `.shugo` files and parses each: data code and template.
2. For each file, the JS module writes: imports and `getStaticPaths` remain at the top, the rest of the data code and template become one function. The template in it is the usual `if` and `for` and calls that record the value of each expression.
3. For each file, a program writes: markup of the template with places where values are substituted. The program has no expressions or project code.
4. Collects all modules, their imported code and the framework library in one file. TypeScript types are erased here.
5. Collects styles.

The compiler does not parse JavaScript itself: it does esbuild, which is connected as a library of Go. The compiler only needs to distinguish code from lines and comments to find the end of an expression.

### Evaluation

1. Go runs `eval.mjs` in Bun or Node and opens one transaction in the storage.
2. JS will find out the list of pages (`getStaticPaths`) for each route, then count the pages. The page is executed by its data code, template, all components and layouts.
3. JS sends Go through a separate channel. What prints the project code does not fall into this channel.
4. Go checks each entry and compares it to the saved one. Differs - records, no - skips.
5. At the end, Go removes what no longer exists: pages, shared blocks, values. Then completes the transaction.

If the project code has fallen, the transaction is canceled and the error is shown with the space in the project file.

### Runtime

Described in [Rantime](runtime.md).

### How long does it take

The stand is a store of [examples/catalog](../examples/catalog/README.md), Bun 1.4, a machine with 4 cores.

| Products | Pages | First Assembly | Update |
| ---: | ---: | ---: | ---: |
| 1 000 | 1 052 | 0.11–0.15 s | 0.11–0.13 s |
| 10 000 | 10 452 | 0.64–0.69 s | 0.40–0.45 s |
| 100 000 | 104 202 | 8.5–8.6 s | 3.2–3.7 s |

Update time does not depend on what has changed: everything is calculated, only the difference is recorded. Of 3.5 s on 100,000, about 2.7 s products are occupied by JS itself, including 0.4 s - read the stand data file. Node update is occupied by 4.3–4.7 s.

## Open questions

- Approve [proposal for analysis of changes](change-tracking.md): Dependency tree prints, comparison on Go, calculation only changed.
- Several JS processes for large sites: pages are divided between processes, time falls almost in proportion to the number of cores.
- Update on signals from sources, without counting everything: [proposal](partial-reload.md).
- Sandbox for stage JS: who sets it up and what it is allowed to do.
- Reassembly of only changed files in development mode. Now compiles everything: examples are 15–50 ms.
