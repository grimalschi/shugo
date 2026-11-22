# Storage

Storage is what remains between the computation stage and the runtime: the project framework and HTML of all pages, laid out in blocks.

## Decisions

- The storage is SQLite.
- Everything is in disassembled form: the page stores links to its parts, not their contents.
- Several processes work with one storage: one updates, others give pages. The page is completely assembled from one state.
- Expression values are written in a flat list in the order of the template. *Replaced by the decision of the author of the project 22 November 2026 year - templates render Svelte ("okay, take a swelt"):* is written in HTML, which renders Svelte, with holes for common blocks.
- Value tapes lie inside page records and shared blocks, not a separate table: the page is read by one search per record. Solved by 21 November 2026 year. With 22 November 2026 year records also lies HTML.
- SQLite driver is on pure Go, without cgo. Solved 21 November 2026 year.

## Implemented, pending approval

The reasons are in [20](../spec/decisions.md#20-storage).

### Tables

`_build/store.db` file.

| Table | Key | What it holds |
| --- | --- | --- |
| `page` | page Address | page entry, hash, route number that the page read from the database, version in which the entry was last changed |
| `shared` | number, name | recording the shared block, her hash that he read, version |
| `shows` | the block and who shows it | who shows what common block: pages and other common blocks |
| `gone` | address | missing pages, their route and the version in which they disappeared: on them, export deletes their files |
| `route` | route number | I read the list of the route pages. |
| `shape`, `reads` | type of reading; view, search values and reader | reads Expanded for Search: Those who read the changed line are found ([Change tracking](change-tracking.md)) |
| `plan` | question to the database | questions for which the project database already has an index |
| `meta` | name | project framework, build name, storage version, its random name, the beginning of the history of changes, the results, to which line of the log data are taken into account |

The project framework along with the `src/site.html` shell lies in storage next to the records and changes with them with one transaction. Rantime does not need other build files.

The storage page entry is given along with the route number. According to it, the server and export know what it is: a page in HTML or a file of another type that gave the module among the pages, for example `sitemap.xml`. The missing page carries the route number to `gone` so that the export deletes the file with the name under which it was recorded.

The version grows with each update. The version of the record is the one in which the record was last changed; it is written by the update itself along with the record, there are no other marks. Exports find what has changed since last time: [Runtime](runtime.md#export). A complete reassembly of the project begins anew: `origin` in `meta` is the version with which the repository knows all the changes.

### Record

Recording - a list of blocks and the block number from which the display begins. The block is of two types.

- **HTML** is what rendered one component with everything it showed on the spot, or the snippet that the spot gave the shared block. It records HTML with holes, what it gives `<head>` pages, also HTML with holes, and the last `<title>` among this; the head of the page is assembled by the server ([Blocks](blocks.md#page-head)). The head snippet and `<title>` have no: that the snippet gave `<head>`, got into the head of the one who rendered it.
- **Link to the common block** is the number of the common block and the snippets that gave it this place: name and blocks. The snippet blocks themselves lie in the same record. How the snippets get into records is in [Blocks](blocks.md#snippets).

HTML with holes is written as texts between which there are holes. Holes are of three types:

| Hole | What's in it | What is filled |
| --- | --- | --- |
| `c` and Number | block of the same record - link to the common block | the General Block |
| `g` and name | snippet that gave the location where this common block is shown | snippet blocks from the location recording |
| `i` and JSON | island: module, styles, parameters, HTML from Svelte, bindings to request data | HTML Islands with data for hydration ([Svelte islands](islands.md)) |

Writing a file not in HTML, which gives a module among pages, is one block of HTML: text that returned `GET`, without holes.

The numbers are written in the record as varint. In the store on 100,000 goods page entry takes an average of 854 byte, the record of the common block - 135 byte; while the record was tape values, not HTML, - 399 and 75 byte.

### Why HTML, not Values

Before 22 November 2026 years in the record lay not HTML, and the values of expressions - tape, and marking each template Shugo kept in the frame, one for all impressions. The records were half as large: in the store on 10,000 products 4.9 MB tapes against 10.2 MB HTML. But the tape and frame was written by his compiler Shugo, which repeated the rules of Svelte: spaces, attributes, style classes. The author of the project decided that the templates compiles Svelte itself, - then discrepancies with Svelte can not be built, and the record is what Svelte renders.

Tracking changes from this has not changed: the records are the same - a page, a common block, a layout - and they read the same. On the catalog stand, all 21 data changes calculate and change exactly the same as before.

### How the recording can not come together

The recording writes a step that executes the project code, and this step is not trusted by rantime. Therefore, the record is checked when received: it is read without trusting the lengths, its blocks are folded into one tree - each block is shown in exactly one place - and the island that the server collects anew for the request gives the same HTML as Svelte. Otherwise, the update will stop. When a page is assembled, the runtime checks for what might go wrong: a missing shared block, a hole of incomprehensible appearance, too deep nestingand responds with an error, not a corrupted HTML.

0–3 service characters share messages in which JS transmits entries to Go. Inside HTML, JS shields them and the 4 symbol also shields them, and Go takes off the shielding: HTML reaches the page as it was; corrupted shielding stops the update.

### What is removed

The update only calculates what the changes touched and deletes in the same transaction:

- pages that the route no longer gives: the server responds to their 404 address, and the address, route, and version fall into the `gone` table, through which the export deletes their files ([Runtime](runtime.md#decisions));
- common blocks that no one else shows are on the `shows` table, without reading the rest of the site.

When the project is compiled again, all records are written again: in the names of common blocks are file paths, and HTML could change along with the templates. So it is conceived: changing the code calculates the entire site ([Build pipeline](build-pipeline.md#decisions)).

### Size

Store on 100,000 products: 104,202 pages, 100,009 shared blocks - file 202 MB. Of these, the page table is 114 MB, the readings (`reads`) are 27 MB, `shows` is 24 MB, the common blocks are 24 MB, and the indexes are around 14 MB. While the records contained value feeds, the file was 140 MB, the page table was 58 MB, and the common blocks were 17 MB.

The page table is with rowid: the entry in a kilobyte or other remains on the file page along with its string. In a table without a rouvid SQLite holds about a quarter of it on the file page, the rest takes to its almost empty page: the store file on 10,000 goods was 32 MB instead of 20.

## Open questions

- Reading the saved hashes at the beginning of the complete build takes 0.5 s on 100,000 pages.
