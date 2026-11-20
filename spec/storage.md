# Storage

Storage is what remains between the computation stage and the runtime: the project program and the values of all pages.

## Decisions

- The storage is SQLite.
- Everything is in disassembled form: the page stores links to its parts, not their contents.
- Several processes work with one storage: one updates, others give pages. The page is completely assembled from one state.
- Expression values are written in a flat list in the order of the pattern.

## Made in the first version, waiting for approval

The reasons are in [20](decisions.md#20-storage).

### Tables

`_build/store.db` file.

| Table | Key | What it holds |
| --- | --- | --- |
| `page` | page Address | page entry and its hash |
| `shared` | number, name | recording of the general block and its hash |
| `value` | hash Tape | values Tape |
| `meta` | name | project program, build name, update number |

The project program is stored next to the values and changes with them in a single transaction. Rantime does not need other build files.

### Record

Record - a list of blocks and the block number from which the display begins. The block has the following entries:

- template: file number and part number (0 is the template itself, then - the markup that it transfers to slots);
- hash tape;
- blocks inside it, in order of display;
- slots: name and blocks that are transferred to it.

The block that stands in place of the common block stores the common block number and its slots instead.

The numbers are written in a compact way. In the store from the stand, the page record takes an average of 134 byte, the record of the common block is 22 byte, the tape is 114 byte.

### Tape

Tape is the value of the block expressions in a row. Each value is preceded by a zero byte.

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

### What was missing in the prototype tape

The prototype tape knew three kinds of values: text, `1`/`0` for the condition and number for the cycle. For a full language it took:

1. **An attribute that may not be.** One text is not enough: absence, name without value and empty value are different things.
2. **Branch number instead of `1`/`0`** to chain with `g-else-if` occupying one value.
3. **The difference between “no value” and “one empty value”.** used to be an empty string; now there is a separator in front of each value.
4. **Separator inside value.** used to break the tape. Now JS removes four service characters from the values (with 0–3 codes); in HTML they are unacceptable.
5. **Attached blocks and slots outside the tape.** Otherwise, the tape would depend on the location of the block on the page, and the same blocks could not be stored once.
6. **Checking in.** Tape checks against template before writing to storage. The descending tape stops the update.

What remains is a limitation: all values are text. If the runtime needs to distinguish a number from a string, the tape will have to be expanded.

### How the tape can not “fit” with the page

Rantime follows the pattern program and takes values from the tape in order. The program says: here is the text, here is the branch number, here is the number of elements. The tape does not converge if the branch number is not a number, values are less or more than the template reads, or blocks inside are not so much as calls of components.

When working properly, this does not happen. But the tape writes a stage that performs the project code, and this stage does not trust the runtime. Therefore, there are two checks: when accepting a record and when assembling a page. In the second case, the runtime responds with an error, and not with a corrupted HTML.

### What is removed

Every update sees all the pages, so it knows what else is needed. Pages, shared blocks and feeds that are not in the new state are deleted in the same transaction.

When the project is compiled again, all records are written again: they contain the file numbers of this build. Tapes remain: most of them will turn out the same.

### Size

Store on 100,000 products: 104,202 pages, 100,009 shared blocks, 304,300 tapes — file 85 MB.

## Open questions

- SQLite driver. Now it is without cgo. The driver is faster on cgo, but complicates the assembly of the framework.
- Reading saved hashes at the beginning of the update takes 0.5 s on 100,000 pages.
- File numbers in records. If replaced with permanent names, only the changed templates will be rewritten after recompilation.
- Very small tapes can be stored directly in the record: a link to the tape takes 16 byte.
