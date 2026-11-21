# Storage

Storage is what remains between the computation stage and the runtime: the project program and the values of all pages.

## Decisions

- The storage is SQLite.
- Everything is in disassembled form: the page stores links to its parts, not their contents.
- Several processes work with one storage: one updates, others give pages. The page is completely assembled from one state.
- Expression values are written in a flat list in the order of the pattern.
- Value tapes lie inside page records and shared blocks, not a separate table: the page is read by one search per record. Solved by 21 on November 2026 year.

## Implemented, pending approval

The reasons are in [20](decisions.md#20-storage).

### Tables

`_build/store.db` file.

| Table | Key | What it holds |
| --- | --- | --- |
| `page` | page Address | page entry, hash, route number that the page read from the database, version in which the entry was last changed |
| `shared` | number, name | recording the shared block, her hash that he read, version |
| `shows` | the block and who shows it | who shows what common block: pages and other common blocks |
| `gone` | address | missing pages and the version in which they disappeared: export deletes their files |
| `route` | route number | that read the list of route pages and whether it distributes `props` |
| `shape`, `reads` | type of reading; view, search values and reader | reads Expanded for Search: Those who read the changed line are found ([Change tracking](change-tracking.md)) |
| `plan` | question to the database | questions for which the project database already has an index |
| `meta` | name | project program, build name, storage version, its random name, the beginning of the history of changes, the results, to which line of the log data are taken into account |

The project program is located in the storage next to the records and changes with them with one transaction. Rantime does not need other build files.

The version grows with each update. The version of the record is the one in which the record was last changed; it is written by the update itself along with the record, there are no other marks. Exports find what has changed since last time: [Runtime](runtime.md#export). A complete reassembly of the project begins anew: `origin` in `meta` is the version with which the repository knows all the changes.

### Record

Record - a list of blocks and the block number from which the display begins. The block has the following entries:

- template: file number and part number (0 is the template itself, then - the markup that it transfers to slots);
- tape values;
- blocks inside it, in order of display;
- slots: name and blocks that are transferred to it.

The block that stands in place of the common block stores the common block number and its slots instead.

The numbers are written in a compact way. In the store on 100,000 goods page recording together with tapes takes up an average of 399 bytes, the record of the common block is 75 bytes.

### Tape

Tape - values of block expressions in a row. Before each value, its length is written.

| Place in template | What's in the tape |
| --- | --- |
| `{expression}` | text |
| `<g-html>` | text, inserted without shielding |
| `attribute={expression}` | `-` (attribute no), `+` (name with no value) or `=` and text |
| `{...attributes}` | number of attributes, then name and value of each |
| `g-if` | number of selected branch from unit; `0` if none is selected |
| `g-for` | number of elements, then the values of each element |
| `<g-slot>` | `1`, if something is transferred to the slot, otherwise `0` and the values of the spare contents |
| component Call | nothing: Attached blocks are recorded in the record, not in the tape |

The same tapes of different records are stored each in its own record. Places for this take more than when stored under the hash, but there is no table of tapes and records, how many records use each: measurement - in [Rantime](runtime.md#sqlite-read-performance).

### What was missing in the prototype tape

The prototype tape knew three kinds of values: text, `1`/`0` for the condition and number for the cycle. For a full language it took:

1. **An attribute that may not be.** One text is not enough: absence, name without value and empty value are different things.
2. **Branch number instead of `1`/`0`** to chain with `g-else-if` occupying one value.
3. **The difference between “no value” and “one empty value”.** Each value has its own length, so the empty value is the length value of 0.
4. **The symbols inside the value.** JS transmits entries in Go messages, separated characters with 0–3 codes. Inside the value JS shields them and the symbol 4, and Go removes the shielding. The value comes to the page as it was; corrupted shielding stops the update.
5. **Attached blocks and slots outside the tape.** Otherwise, the tape would depend on the location of the block on the page.
6. **Checking in.** Tape checks against template before writing to storage. The descending tape stops the update.

What remains is a limitation: all values are text. If the runtime needs to distinguish a number from a string, the tape will have to be expanded.

### How the tape can not “fit” with the page

Rantime follows the pattern program and takes values from the tape in order. The program says: here is the text, here is the branch number, here is the number of elements. The tape does not converge if the branch number is not a number, values are less or more than the template reads, or blocks inside are not so much as calls of components.

When working properly, this does not happen. But the tape writes a stage that performs the project code, and this stage does not trust the runtime. Therefore, there are two checks: when accepting a record and when assembling a page. In the second case, the runtime responds with an error, and not with a corrupted HTML.

### What is removed

The update only calculates what the changes touched and deletes in the same transaction:

- pages that the route no longer gives;
- common blocks that no one else shows are on the `shows` table, without reading the rest of the site.

When the project is compiled again, all records are written again: they contain the file numbers of this build.

### Size

The 100 000-page store: 104,202 pages, 100,009 common blocks - 140 MB file. Of these, 58 MB page records, 27 MB readings (`reads`) - 27 MB, `shows` - 24 MB, 17 MB block, and indices - about 14 MB.

## Open questions

- SQLite driver. Now it is without cgo. The driver is faster on cgo, but complicates the assembly of the framework.
- Reading the saved hashes at the beginning of the complete build takes 0.5 s on 100,000 pages.
- File numbers in records. If replaced with permanent names, only the changed templates will be rewritten after recompilation.
