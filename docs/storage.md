# Storage

Storage is what remains between the computation stage and the runtime: the project framework and the values of all pages.

## Decisions

- The storage is SQLite.
- Everything is in disassembled form: the page stores links to its parts, not their contents.
- Several processes work with one storage: one updates, others give pages. The page is completely assembled from one state.
- Expression values are written in a flat list in the order of the pattern.
- Value tapes lie inside page records and shared blocks, not a separate table: the page is read by one search per record. Solved by 21 on November 2026 year.
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

Record - a list of blocks and the block number from which the display begins. The block has the following entries:

- template: file number and part number (0 is the template itself, then its snippets and its `<svelte:head>`);
- tape values;
- blocks inside it in order of display: components and snippets with the parameters that he showed;
- blocks that its template gave `<head>` pages in `<svelte:head>`, and separately - `<title>` blocks from there; the head of the page is collected by the server ([Blocks](blocks.md#page-head));
- snippets without the parameters that the caller gave him: name and blocks. The blocks themselves lie in the same record as the caller's block.

The block that stands in place of the shared block stores instead the common block number and the snippets that gave it that place. How the snippets get into the records is in [Blocks](blocks.md#snippets).

The file entry is not in HTML, which gives the module among the pages, is one block: the module number, part of 0 and one value is text that returned `GET`. Modules are numbered after templates; the module framework is one part with one operation that writes the value as is.

The numbers are written in a compact way. In the store on 100,000 goods page recording together with tapes takes up an average of 399 bytes, the record of the common block is 75 bytes.

### Tape

Tape - values of block expressions in a row. Before each value, its length is written.

| Place in template | What's in the tape |
| --- | --- |
| `{expression}` | text |
| `{@html expression}` | text, inserted without shielding |
| `attribute={expression}`, `class`, `style` | `-` (attribute no), `+` (name with no value) or `=` and text |
| `{...attributes}` | number of attributes, then name and value of each |
| `{#if}`, `{:else if}`, `{:else}` | number of selected branch from unit; `0` if none is selected |
| `{#await}` | `1` — branch to `{:then}`, `2` — branch to `{:then}` |
| `{#each}` | number of elements, then the value of each element; if there are no elements, the value of the branch `{:else}` |
| `{@render snippet()}` | `-` is a snippet (`snippet?.()`); `c` and number are calculated here, this is the next nested block; `f`, a number and name, for example `f0:children`, is a snippet without parameters that gave the caller |
| `<svelte:element this={tag}>` | tag name; empty - no element. Then the values of its attributes and contents |
| `<svelte:component this={…}>` | `1` — component shown, `0` — value not available |
| component Call | nothing: Attached blocks are recorded in the record, not in the tape |

The numbers are needed because the snippet can show not the component to which it was given: `<Inner content={children} />`, but in `Inner` - `{@render content()}`. The number after `f` - how many components outside of the place `{@render}` is the one to whom the snippet was given: its block has written down what blocks the snippet has. The number after `c` is how many components outside the template in which the snippet is written: `{@render children()}` inside the snippet refers to this template. Rantime on these numbers finds whose blocks they are and with which snippets to show them.

The same tapes of different records are stored each in its own record. Places for this take more than when stored under the hash, but there is no table of tapes and records, how many records use each: measurement - in [Rantime](runtime.md#sqlite-read-performance).

### What was missing in the prototype tape

The prototype tape knew three kinds of values: text, `1`/`0` for the condition and number for the cycle. For a full language it took:

1. **An attribute that may not be.** One text is not enough: absence, name without value and empty value are different things.
2. **Branch number instead of `1`/`0`** to chain with `{:else if}` occupying one value.
3. **The difference between “no value” and “one empty value”.** Each value has its own length, so the empty value is the length value of 0.
4. **The symbols inside the value.** JS transmits entries in Go messages, separated characters with 0–3 codes. Inside the value JS shields them and the symbol 4, and Go removes the shielding. The value comes to the page as it was; corrupted shielding stops the update.
5. **Attached blocks and snippets outside the tape.** Otherwise, the tape would depend on the location of the block on the page.
6. **Checking in.** Tape checks against template before writing to storage. The descending tape stops the update.

What remains is a limitation: all values are text. If the runtime needs to distinguish a number from a string, the tape will have to be expanded.

### How the tape can not “fit” with the page

Rantime follows the frame of the template and takes values from the tape in order. The framework says: here is the text, here is the branch number, here is the number of elements. The tape does not converge if the branch number is not in place of the number, values are less or more than the template reads, or blocks inside are not so much as calls of components and snippets calculated on the spot.

When working properly, this does not happen. But the tape writes a stage that performs the project code, and this stage does not trust the runtime. Therefore, there are two checks: when accepting a record and when assembling a page. In the second case, the runtime responds with an error, and not with a corrupted HTML.

### What is removed

The update only calculates what the changes touched and deletes in the same transaction:

- pages that the route no longer gives: the server responds to their 404 address, and the address, route, and version fall into the `gone` table, through which the export deletes their files ([Runtime](runtime.md#decisions));
- common blocks that no one else shows are on the `shows` table, without reading the rest of the site.

When the project is compiled again, all records are written again: they contain the file numbers of this build. So it is conceived: changing the code calculates the entire site ([Build pipeline](build-pipeline.md#decisions)).

### Size

The 100 000-page store: 104,202 pages, 100,009 common blocks - 140 MB file. Of these, 58 MB page records, 27 MB readings (`reads`) - 27 MB, `shows` - 24 MB, 17 MB block, and indices - about 14 MB.

## Open questions

- Reading the saved hashes at the beginning of the complete build takes 0.5 s on 100,000 pages.
