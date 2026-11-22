# Multiple JS processes for first build

Proposal on how to make the first build of a large site faster. None of this has yet been done; the author of the project postponed the question here 22 November 2026 years.

## Decisions

Nothing yet.

## Like now

The project code performs one JS process: pages and common blocks are calculated one by one, in a row. Go at this time accepts records, compares them with storage and writes.

First Assembly Shop on 100,000 products, 21 November 2026 year, Bun, 8 cores:

| Step | Time |
| --- | ---: |
| compilation | 0.02 s |
| route Page Lists | 0.23 s |
| calculation of pages and common blocks (JS) | 9.1 s |
| storage record (Go) | 5.1 s |
| total, with base indexes | 14.5 s |

With the 22 November 2026 page year rendering of the Svelte, and the first build is almost the same: 14.6 s ([README booth](../../examples/catalog/README.md#results)).

Updates after data changes calculate dozens of records, and they are not hindered by one process.

## Proposal

The first assembly and `shugo build` run several processes of JS - by the number of cores, but not more than necessary: each Go gives its part of the pages. All processes perform the same `eval.mjs`; common blocks that have met multiple processes compute one of them the one to which Go gave the name first.

- The calculation is divided between processes almost without a remainder: pages do not depend on each other.
- The record in the storage remains one, in Go. It can also be accelerated: write in packs more and count the hashes of records in parallel.
- Each process keeps its memory and its connection to the project database; they only read the database.
- Updates don’t need to be: they only have one process.

## Open questions

- When to do: Advice - when the first assembly will become a bottleneck.
- How many processes: by number of cores or setting.
