# Data adapter and data copy

The document describes how data from any source gets into the project and how framework keeps it to know exactly what has changed.

## Decisions

The words of the author of the project, 20 November 2026 year.

- The framework stores a copy of the project data in SQLite. This is part of the overall architecture: a new copy is compared with the copy, and on its basis, links between data and pages are built.
- The capabilities of SQLite are fully used.
- The adapter connects the project to any data source and gives the data in one form: lists of records with a key. What field serves as a key and what stores the date of change is specified in the adapter settings.
- Adapter is a class that is inherited: the author of the project uses an heir.
- **Primary mode.** The adapter loads the new data set as a whole. What has changed, created and deleted is the matching of the new set with the copy.
- **Additional regime.** After full load, you can make point changes: new lines, changed strings, remote identifiers. They are transmitted manually.
- The nested data, for example, options within the product, is decomposed by the author of the project: a separate collection or one field. A separate collection looks more correct, but this is the solution of the project, and framework does not impose it.
- Changes to the copy are recognized from the log that triggers its tables, rather than the mandatory `updated_at` field.
- The adapter is most likely running from the command line.

## Proposal, pending approval

### The overall picture

```
source ──► adapter ──► copy data in SQLite ──► page Requests ──► pages and common blocks
             full article on triggers: who read what
             loading what actually
             or points have changed
             changes
```

Left half is this document. Right half is [analysis of changes](change-tracking.md): queries through `Shugo.query` and recalculates only those pages that have read the changed entries.

The adapter knows nothing about the pages. He is responsible for one thing: to make the copy coincide with the source. What exactly has changed at the same time, the copy finds out itself.

### Main mode: full load

1. The adapter reads the source and gives all records of each collection.
2. A new set is placed next to the copy.
3. SQLite compares itself, with a single transaction: records that are not in the new set are deleted; new and different ones are recorded; matching ones are not touched.
4. Triggers write to the magazine exactly those records that have changed.

The correspondence between the old and new record is the key. Therefore, the main data requirement: **the key does not change from download to download.** The string order number cannot be the key: on the measurement, the recall of a new product received a remote review number, and the journal recorded "review changed" instead of "one deleted, another added".

Measured on the stand data, 100,000 products, 950,686 records in seven tables:

| Step | Time |
| --- | ---: |
| Write a new set of separate base, Bun | 1.6–1.7 s |
| Compare with copy of means SQLite, Go | 1.9–2.1 s |
| In the journal after comparison | 40 rows: exactly those entries that have changed |
| The same comparison when nothing has changed | 2.1 s, there is nothing new in the magazine |

Full load depends on the size of the data: about 3.5 s per million records, not counting reading the source itself. Further, the site update depends only on the size of the change.

### Additional mode: point changes

The adapter is transmitted by the changes themselves: lines that have appeared or changed, and line keys that are no longer there. The copy changes to them, the journal records what it would have written at full load. The source is not re-read, and the time depends only on the number of lines transmitted.

So webhooks and source change feeds are connected. Full download remains a way to put the copy in order: it will fix everything that point changes missed.

### What the Adapter Author Says

| What | I will. | Why |
| --- | --- | --- |
| Collections: name, key field, fields | yeah. | copy - these tables: filtered and sorted by fields |
| Download: give all records of the collection | yeah. | main Mode |
| Field with date of change | none | adapter can request from the source only changed and transmit it pointwise |
| How to Dispose of Attached | none | project Solution |

Outline, not solution:

```ts
// data/shop.ts
import { Adapter } from "shugo/data"

export default class Shop extends Adapter {
  collections = {
    products: { key: "id", updated: "updated_at" },
    variants: { key: "sku" },          // the item is in your collection
  }

  // Main mode: everything that is in the source.
  async *load(collection) {
    for await (const row of readSource(collection)) yield row
  }
}
```

Point changes base class takes itself; the author has nothing to write for them.

### Commands

Outline:

```
shugo data load                      download all adapter and find differences
shugo data put products rows.json    new and Modified Lines
shugo data drop products p17 p18     lines that no longer exist
shugo update                         find pages that have read the changed
```

## Open questions

- Who describes the copy tables: The adapter lists the fields and their view, or tables are derived from the first data downloaded.
- What does it mean to “fully use SQLite” for page requests: only the `Shugo.query` constructor or your SQL. You can track your SQL more crudely by the tables and columns that it reads.
- After the `shugo data …` commands, the site is updated by itself or by a separate `shugo update` command.
- Multiple adapters in one project and connections between their collections.
- Ready adapters: JSON and Markdown files, databases with SQL, common CMS.
- Source, which itself is the base with SQL: can you do without a copy if the journal is running it.
