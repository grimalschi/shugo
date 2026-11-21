# Build pipeline

The project is being built in stages. Each stage has its own entrance and its own result. The next stage uses only the results of the previous ones.

## Decisions

- The work is divided into stages with a clear boundary: compilation of the project, work with data and expressions, assembly of pages in rantime.
- Between loading data and calculating expressions is the analysis of changes: expressions are calculated only for changed areas. Analysis is mandatory for any device stages. The choice is only in who compares: Go between two stages on TS or TS itself.
- When the project is changed, the entire chain starts from scratch. When data is changed, the update is point-by-point.
- While every update process JS starts again and loads all the data; then the result is compared with the past. *Replaced by solutions from 20 November 2026 year:* update recognizes changes from the database log and calculates only what they touched - [Change tracking](change-tracking.md#decisions).
- Expressions and data code perform one JS process, not a page-to-page process: starting a process is more expensive than calculating a page.
- JS is performed by Bun and Node, both must work.
- Rantime always takes meanings from the saved and doesn't know how or when they were updated.
- The results are stored in SQLite.

## Implemented, pending approval

Below is how the stages are arranged in the final version ([Change tracking](change-tracking.md#final-version-waiting-for-approval)).

| Stage | Who performs | What gets | What gives |
| --- | --- | --- | --- |
| 1. Compilation | Go | project Sources | `_build/eval.mjs` — the entire JS project in one file; `_build/site.json` — project program; `_build/assets/` — styles |
| 2. Calculation | Bun or Node running Go | `eval.mjs`, project database and its log | changes in `_build/store.db` and report on them |
| 3. Rantime | Go | `store.db`, `public/`, `assets/` | HTML pages on request or changed page files |

### Compilation

1. Finds all `.shugo` files and parses each: data code and template.
2. For each file, the JS module writes: imports and `getStaticPaths` remain at the top, the rest of the data code and template become one normal function, without `async`. The template in it is the usual `if` and `for` and calls that record the value of each expression.
3. For each file, a program writes: markup of the template with places where values are substituted. The program has no expressions or project code.
4. Collects all modules, their imported code and the framework library in one file. TypeScript types are erased here.
5. Collects styles.

The compiler does not parse JavaScript itself: it does esbuild, which is connected as a library of Go. The compiler only needs to distinguish code from lines and comments to find the end of an expression.

### Evaluation

1. Go opens a transaction in the repository and reads the project database log: what lines have changed since the last update and what were. From the reading table, he finds who has read the changed columns of these lines: pages, common blocks, lists of route pages. If you can't find anyone, JS won't start.
2. Go launches `eval.mjs` in Bun or Node and one message says what is known from past launches: types of readings and questions to the database for which there is already an index.
3. If the route pages are affected, JS lists it again (`getStaticPaths`). New Go pages add to what needs to be calculated, and the missing ones delete.
4. Go asks you to calculate pages and common blocks. JS calculates them one by one, without waiting: data code, template, components, layouts. Each line from the database records which columns are read. JS asks about the new type of request Go, and he builds an index in the project database.
5. JS sends Go along with readings via a separate channel. What prints the project code does not get into this channel.
6. Go checks each record and compares it to the saved one. Differs — writes along with the storage version, no — skips. Common blocks that records are called for the first time, Go asks to calculate in the next round.
7. At the end, Go removes shared blocks that no one else shows, finds pages where HTML has changed, and completes the transaction.

The first build follows the same path: everything is calculated. If the project code falls, the transaction is canceled, and the error is shown with the space in the project file.

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

- Approve [final Version](change-tracking.md#final-version-waiting-for-approval).
- Several JS processes for the first build of large sites: pages are divided between processes, time falls almost in proportion to the number of cores.
- Sandbox for stage JS: who sets it up and what it is allowed to do.
- Reassembly of only changed files in development mode. Now compiles everything: examples are 15–50 ms.
