# Data adapter and data copy

The document describes how data from any source gets into the project and how framework keeps it to know exactly what has changed.

## Decisions

The words of the author of the project, 20 November 2026 year.

- The framework stores a copy of the project data in SQLite. This is part of the overall architecture: a new copy is compared with the copy, and on its basis, links between data and pages are built.
- The capabilities of SQLite are fully used. This is said about the internal work of Shugo: comparison, communication, journal. Page requests to a copy may be limited; this is discussed separately.
- The adapter connects the project to any data source and gives the data in one form: lists of records with a key. What field serves as a key and what stores the date of change is specified in the adapter settings.
- Adapter is a class that is inherited: the author of the project uses an heir.
- **Primary mode.** The adapter loads the new data set as a whole. What has changed, created and deleted is the matching of the new set with the copy.
- **Additional regime.** After full load, you can make point changes: new lines, changed strings, remote identifiers. They are transmitted manually.
- The nested data, for example, options within the product, is decomposed by the author of the project: a separate collection or one field. A separate collection looks more correct, but this is the solution of the project, and framework does not impose it.
- Changes to the copy are recognized from the log that triggers its tables, rather than the mandatory `updated_at` field.
- The adapter is most likely running from the command line.
- The copy tables describe Shugo. The adapter tells you what data will come: a set of fields and their types - a scheme on TypeScript, probably on Valibot or Zod. The adapter works with TS data types; it writes Shugo into a copy and decides which types of SQLite to take based on what the adapter has sent.
- The date the adapter sent is saved as a date in the copy.
- The indexes in the copy are built by Shugo itself: it knows all the project requests and columns they are looking for. The author of the adapter does not describe the indexes.

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
| Collections: name, key field and recording scheme | yeah. | according to the scheme Shugo creates a copy table and checks what has come |
| Download: give all records of the collection | yeah. | main Mode |
| Field with date of change | none | adapter can request from the source only changed and transmit it pointwise |
| How to Dispose of Attached | none | project Solution |

Outline, not solution:

```ts
// data/shop.ts
import { Adapter } from "shugo/data"

export default class Shop extends Adapter {
  collections = {
    products: { key: "id", updated: "updated_at", schema: Product },
    variants: { key: "sku", schema: Variant },   // the item is in your collection
  }

  // Main mode: everything that is in the source.
  async *load(collection) {
    for await (const row of readSource(collection)) yield row
  }
}
```

Point changes base class takes itself; the author has nothing to write for them.

### Recording scheme and copy types

The author writes the recording scheme on the schema library, types TypeScript are derived from it:

```ts
const Product = z.object({
  id: z.string(),
  title: z.string(),
  price: z.number().int(),
  available: z.boolean(),
  discount: z.object({ percent: z.number().int(), until: z.string() }).nullable(),
  badges: z.array(z.string()),
})
```

Shugo needs two things from the schematic: check the entry that has come and find out the fields with their view to create a table. Tested on the Zod 4.6 and Valibot 1.5: both can do both. The recording is checked through a common interface for them Standard Schema. The fields and their appearance are both given the same, in the form of JSON Schema: `string`, `integer`, `number`, `boolean`, the nested object, the list of which fields are required.

How Shugo chooses column type:

| In the scheme | In a copy | What the data code sees |
| --- | --- | --- |
| string, enumeration | `TEXT` | line |
| whole number | `INTEGER` | number |
| number | `REAL` | number |
| yes or no | `INTEGER`, 0 or 1 | `true` or `false` |
| optional or `null` | column allows empty value | `null` |
| attached object or list | `TEXT` with JSON | object or list; read in full |
| date | `TEXT` as ISO 8601 | `Date` |

About the date. The schema itself says that the field is a date: both in Valibot and in Zod, such a field has the form `date`, and both accept `Date` in it, and the text is rejected. Words for date are not only in JSON Schema, so when translating Shugo itself says what to write there; checked on both libraries.

SQLite has no separate date type: the date is stored in text or number, and its date functions take both. The text in the form of ISO 8601, `2026-11-20T10:30:00.000Z` is offered: such dates are compared and sorted as rows, read by eyes, functions of `date`, `strftime`, `unixepoch` work with them. The data code is returned to `Date`; it is checked that the value is returned without loss.

### Indices

An index needs a query that searches for rows by column values or takes them in column order. What exactly searches for and how to organize each project request, Shugo learns when the request is first set, and then builds an index: first the columns that are searched for, then the order columns. You don’t need an index for a key and a unique column: it already exists.

Why it is necessary, measured on the example of a store without indexes, except for keys: the first assembly 1,000 Goods 0.7 s instead of 0.2 s, 2,000 — 1.9 s instead of 0.4 s, 5,000 — 11 s instead of 0.7 s. Time grows like a square of the number of records: each page goes through the whole table. Stands still worked with indexes that were created by the data generator.

Made in the fifth trial branch, `lab/query-signals`:

- JS base only reads, so asks for the index from Go and waits for an answer. Go builds an index if there is no suitable one: a key, a unique column, an index made by hands, its former.
- On the example of the store, eight indexes are built. On 100,000 products, this is 0.8 s once per base. The first assembly with them was taken by 15.4 s on Bun; on hand-made indices, the 2 variant was collected by 13.9 s.
- An index for a query with an order ending in a table key, and strings with the same order value going along the key. Otherwise, the new index would change the order of the rows on a page that is already computed.
- To see the new index, JS starts reading the database again. If at this time the database is written, the page can show two data states; the next update corrects this because the journal space is remembered before reading.
- Indices that are no longer needed by any query are not removed.

Changed the scheme — changed the tables copy. The journal after that can not say what has changed in the data, and the site is calculated as a whole; so already works option with the database.

### Commands

Outline:

```
shugo data load                      download all adapter and find differences
shugo data put products rows.json    new and Modified Lines
shugo data drop products p17 p18     lines that no longer exist
shugo update                         find pages that have read the changed
```

## Open questions

- What is the schema library: Valibot, Zod or any that gives JSON Schema and supports Standard Schema.
- What are the limitations of page requests to a copy; your SQL can only be tracked roughly, by tables and columns.
- After the `shugo data …` commands, the site is updated by itself or by a separate `shugo update` command.
- Multiple adapters in one project and connections between their collections.
- Ready adapters: JSON and Markdown files, databases with SQL, common CMS.
- Source, which itself is the base with SQL: can you do without a copy if the journal is running it.
