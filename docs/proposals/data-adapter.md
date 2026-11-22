# Data adapter

Proposal on how data from any source — CMS, API, files, other database — get into the project database. Adapter is not yet available.

## Decisions

The decision of the author of the project, 21 November 2026 year: data changes come through the adapter; partial reboot by a separate mechanism is not done.

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

## What already has

Site data is in one database SQLite project; its path calls `shugo.json`: `{ "data": "data/site.db" }`. Pages read it only through `Shugo.query` ([Data loading](../../spec/data-loading.md)). Any program can fill the base of the usual SQL: so does the store stand.

The rest of Shugo does it yourself:

- puts on each table triggers that record in the journal every added, changed and deleted line with what it was;
- by journal finds what has changed since the last update, and only calculates those who read the changed ([Change tracking](../change-tracking.md));
- builds indexes that page requests need.

The adapter adds to this one proven way to bring the database in line with the source: download data, compare with what is already in the database, and describe the tables with a schema. The base at the same time is a copy of the source: Shugo knows what has changed.

## Proposal, pending approval

### The overall picture

```
source ──► adapter ──► project Database ──► Shugo.query ──► pages and common blocks
             full article on triggers: who read what
             loading what actually
             or points have changed
             changes
```

The left half is this document. Right done: queries through `Shugo.query` and calculation of only those pages that have read the changed lines, - [Change tracking](../change-tracking.md).

The adapter knows nothing about the pages. He is responsible for one thing: to make the base coincide with the source. What exactly has changed, the base finds out itself.

### Main mode: full load

1. The adapter reads the source and gives all records of each collection.
2. A new set is placed next to the base.
3. SQLite compares itself, with a single transaction: records that are not in the new set are deleted; new and different ones are recorded; matching ones are not touched.
4. Triggers write to the magazine exactly those records that have changed.

The correspondence between the old and new record is the key. Therefore, the main data requirement: **the key does not change from download to download.** The string order number cannot be the key: on the measurement, the recall of a new product received a remote review number, and the journal recorded "review changed" instead of "one deleted, another added".

Measured on the stand data, 100,000 products, 950,686 records in seven tables:

| Step | Time |
| --- | ---: |
| Write a new set of separate base, Bun | 1.6–1.7 s |
| Compare with base of means SQLite, Go | 1.9–2.1 s |
| In the journal after comparison | 40 rows: exactly those entries that have changed |
| The same comparison when nothing has changed | 2.1 s, there is nothing new in the magazine |

Full load depends on the size of the data: about 3.5 s per million records, not counting reading the source itself. Further, the site update depends only on the size of the change.

### Additional mode: point changes

The adapter is transmitted by the changes themselves: lines that have appeared or changed, and line keys that are no longer there. The database changes to them, the journal records what it would have written at full load. The source is not re-read, and the time depends only on the number of lines transmitted.

So webhooks and source change feeds are connected. Full load remains a way to put the base in order: it will fix everything that point changes missed.

### Who gives the signal

The adapter does not follow the source itself: it is launched. Who starts - solves the project; the framework is enough to update after the changes in the base.

| Signal | What does | When it fits |
| --- | --- | --- |
| Webhook Source | CMS calls address after record save; handler transfers changed records to adapter pointwise | source knows how to webhooks, changes are rare and speed is important |
| Queue | source changes go messages; the handler takes them in bundles and transmits pointwise | many changes, the source is already writing in the queue |
| Schedule | once in N minutes full load or sample “changed after such and such date” by date field | source does not report anything; at the same time insurance against missed signals |

Webhoek and queue handler is not a part of Shugo: these are a few lines that cause `shugo data put` and `shugo data drop` (commands below). Full schedule download stands next to them as insurance: the point signal can get lost, full download will fix everything.

There used to be a separate document on partial data reset: how not to load all the data at each update with tags or a data layer that remembers who read what. This is closed by the journal and requests via `Shugo.query`: the update reads only the changed lines from the journal and finds who read them. The question of several JS processes for the first assembly remained open from that document it is in [the Offer](parallel-build.md).

### What the Adapter Author Says

| What | I will. | Why |
| --- | --- | --- |
| Collections: name, key field and recording scheme | yeah. | Shugo creates base tables and checks what has come |
| Download: give all records of the collection | yeah. | main Mode |
| Field with date of change | none | adapter can request from the source only changed and transmit it pointwise |
| How to Dispose of Attached | none | project Solution |

Outline, not solution:

```ts
// src/adapters/shop.ts
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

### Writing scheme and column types

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

| In the scheme | In the database | What the page code sees |
| --- | --- | --- |
| string, enumeration | `TEXT` | line |
| whole number | `INTEGER` | number |
| number | `REAL` | number |
| yes or no | `INTEGER`, 0 or 1 | `true` or `false` |
| optional or `null` | column allows empty value | `null` |
| attached object or list | `TEXT` with JSON | object or list; read in full |
| date | `TEXT` as ISO 8601 | `Date` |

About the date. The schema itself says that the field is a date: both in Valibot and in Zod, such a field has the form `date`, and both accept `Date` in it, and the text is rejected. Words for date are not only in JSON Schema, so when translating Shugo itself says what to write there; checked on both libraries.

SQLite has no separate date type: the date is stored in text or number, and its date functions take both. The text in the form of ISO 8601, `2026-11-20T10:30:00.000Z` is offered: such dates are compared and sorted as rows, read by eyes, functions of `date`, `strftime`, `unixepoch` work with them. The page code is returned to `Date`; it is checked that the value is returned without loss.

### Indices

An index needs a query that searches for rows by column values or takes them in column order. What exactly searches for and how to organize each project request, Shugo learns when the request is first set, and then builds an index: first the columns that are searched for, then the order columns. You don’t need an index for a key and a unique column: it already exists.

Why it is necessary, measured on the example of a store without indexes, except for keys: the first assembly 1,000 Goods 0.7 s instead of 0.2 s, 2,000 — 1.9 s instead of 0.4 s, 5,000 — 11 s instead of 0.7 s. Time grows as a square of the number of records: each page takes the whole table. The data generator of the index stand does not create, except for the keys and the address of the goods: the rest builds Shugo.

This is done and works without an adapter:

- JS base only reads, so asks for the index from Go and waits for an answer. Go builds an index if there is no suitable one: a key, a unique column, an index made by hands, its former.
- On the example of the store, eight indices are built. On 100,000 products, this is 0.8 s once per base; the first assembly with them is 14.1 s on Bun ([README booth](../../examples/catalog/README.md)).
- An index for a query with an order ending in a table key, and strings with the same order value going along the key. Otherwise, the new index would change the order of the rows on a page that is already computed.
- To see the new index, JS starts reading the database again. If at this time the database is written, the page can show two data states; the next update corrects this because the journal space is remembered before reading.
- Indices that are no longer needed by any query are not removed.

The scheme has changed the database tables have changed. The journal can not say what has changed in the data, and the site is calculated as a whole; this is how the database works now.

### Libraries without synchronous option

Page is calculated without waiting: `await` is not in its code. A library that can only wait Markdown parsing with plugins, API query, code highlighting cannot be called for each page. Its place is an adapter: it waits as long as necessary, and puts the finished HTML articles in the database, the answer is API, the data is analyzed. What you need once for the entire calculation, for example, a created backlight, you can wait at the top level of the module from `src/lib/`. The author of the project postponed the question of whether this is enough, here 22 November 2026 year; advice - enough.

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
- What are the limitations of page requests to the database; your SQL can only be tracked roughly, by tables and columns. Now requests are only through the designer `Shugo.query`.
- After the `shugo data …` commands, the site is updated by itself or by a separate `shugo update` command.
- Multiple adapters in one project and connections between their collections.
- Ready adapters: JSON and Markdown files, databases with SQL, common CMS.
- Source, which itself is the base with SQL: can you do without a copy if the journal is running it.
- Where the adapter is located: in `src/`, like the rest of the project code, or near the base.
- Images from the source: the adapter downloads them into the project along with the data ([Pictures](images.md)).
- Whether to make the adapter the next big step. Advice is yes: without it, data in the database bring their scripts.
