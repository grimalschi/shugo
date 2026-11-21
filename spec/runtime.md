# Runtime

Rantime is a server on Go that collects HTML pages from storage.

## Decisions

- Runtime uses the project program and stored values. The project code it does not execute, requests data does not do, does not calculate anything.
- All pages and all data are known in advance. There are no pages that depend on cookies or query strings, API routes and processing forms.
- The main mode is to assemble the page on the fly with each request. When data changes often, writing thousands of HTML files in advance makes no sense: the runtime collects a page from ready-made pieces very quickly.
- The second mode is a one-time generation of all HTML-files of the site.
- Rantime always takes meanings from the saved and doesn't know how or when they were updated.
- Several processes work with one storage: one updates the values, others give the pages. The page is entirely collected from one data state.
- The total runtime is common, not collected for the project. One project per process is basic; maintenance of several projects by one process is provided for the future.
- Everything that runs Go is safe and can work outside containers.
- The stored values for the runtime are untrusted data: each is screened at the insertion location.
- Development mode is needed: reassembly when changing files and updating the page in the browser.
- The server does not keep the site in memory: for each request it gets from SQLite what the page needs. Requests to the storage should be optimized. The 20 November 2026 is solved.

## Implemented, pending approval

### Page assembly

There are no copies of the site in the server memory. For each request, it:

1. It takes a connection from the reading pool to the storage and opens a reading transaction.
2. Reads the build name. It parses the template program once for each build and keeps it in memory: it is code, not data, and updating the data does not change it.
3. Finds a page entry at the address - one search by key. Values ribbons lie inside the record.
4. It follows the program of the template of the first block. The markup is copied, the next value is taken from the block tape and screens it on the place of expression.
5. At the call point, the component moves to the next sub-block. The shared block reads from its own record, also by a single search; the block shown on the page twice is read once.
6. In place, `<g-slot>` shows the blocks that the caller has transmitted.
7. Closes the transaction and returns the connection to the pool.

The entire page is assembled from one state: the update that ended in the middle does not affect it. The next request already sees it - without a storage survey and without re-reading.

The connections in the pool are twice as large as the cores; everyone has pre-prepared queries for the key. The storage file is displayed in memory (`mmap`): SQLite reads its pages where the system holds them without copying.

### What the server answers

| Request | Response |
| --- | --- |
| page Address | 200, HTML, `Cache-Control: no-cache` |
| address of the page with a slash at the end | 308 to address without slash |
| address from `redirects` | 301 |
| file from `public/` | file |
| `/_assets/…` | file collected by the compiler; cached forever: name changes with content |
| everything else | 404 and page `pages/404.shugo`, if any |
| not `GET` and not `HEAD` | 405 |

An update made by another process, the server sees from the next request.

### SQLite read performance

The first version kept a copy of all the pages and after each update built it all over again: on 100,000 pages it is 0.2 s for each update, and the first request after the update waited for them. The author of the project decided not to keep a copy. How to read quickly, measured on copies of store storage in 100,000 products: 20,000 random pages in a row, one reading transaction per page, time per page. Two SQLite drivers: on the pure Go that stands now, and on C via cgo.

| How to read | Go Driver | C Driver |
| --- | ---: | ---: |
| Request for each record and each tape, tapes by separate table | 142–150 µs | 120 µs |
| The same file is displayed in memory (`mmap`) | 107–112 µs | 92–95 µs |
| Packs: all common blocks, then all tapes | 120–126 µs | 95–103 µs |
| One SQL request for the entire page | 124–142 µs | 92–121 µs |
| **Tapes inside records**, write request, `mmap` | **61–65 µs** | **53–55 µs** |
| Same without `mmap` | 74–76 µs | 66–69 µs |
| For comparison: a copy in memory | 9 µs | — |

Time determines the number of searches by key, not the number of SQL queries: to collect them in bundles or in one query does not help. The store page is her entry and about 10 shared blocks; with the tapes a separate table was added to them about 15 search feeds. So it is done: tapes inside records, the file is displayed in memory, the driver on Go.

### What Rantime Doesn't Trust

Values came from the stage that executed the project code. Therefore:

- text and attribute values are always screened; without screening, only the value of `<g-html>` is inserted;
- address in `href`, `src` and similar attributes is replaced if it can execute the code ([14](decisions.md#14-addresses-in-links-are-checked));
- attribute name from `{...attributes}` is checked;
- tape that does not match the template gives the answer 500, and not a corrupted page - as it happens, described in [Storage](storage.md#how-the-tape-can-not-fit-with-the-page);
- the nesting of components is limited by 256 levels, the page size is 64 MB.

The project program is data, not code. The worst thing that a wrong program or tape can do is to have HTML or an error.

### Export

`shugo export` writes the site into a folder: page `/about` - in `about/index.html`, page 404 - in `404.html`, next to `public/` and `_assets/`. Rules - in [21](decisions.md#21-export-to-files).

Export follows the previous one. In the folder, he leaves a list of `.shugo-export.json`: what storage and what version he recorded, and which files he copied - with the size and time of the original change. The next export from the same storage:

- builds anew pages whose entry has changed after this version, and pages that show the shared block with the changed entry, directly or through other common blocks. It finds them by the version indexes and the `shows` table;
- deletes files of pages that were missing after this version;
- copies only files that have changed size or time of change.

His work grows with the change, not with the site. Update for this nothing to note: version of the recording it writes along with the recording itself.

If there is no list, exports build all pages and do not delete anything. If the list is from another repository or version before the project is completely reassembled, the export builds all pages and removes from the folder the page files that the site does not have. Files that don't look like pages, and folders whose name starts with a dot, it doesn't touch. The file with the previous content is not rewritten.

Measured on the store in 100,000 products, 104,202 pages, Bun, disk ext4. After each change — `shugo update`, then `shugo export`:

| What changed | Pages with new HTML | Update | Export |
| --- | ---: | ---: | ---: |
| First export to an empty folder | all | — | 7.1 s |
| Nothing. | 0 | 0.42 s | 6 ms |
| Price of one product | 7 | 28 ms | 8 ms |
| Prices 20 products | 70 | 36 ms | 17 ms |
| Last product removed: 88 pages, one file removed | 88 | 0.41 s | 32 ms |
| Product name of the month: header on all pages | all | 0.12 s | 8.1 s |
| Product added: counters in the menu on all pages | all | 0.48 s | 8.3 s |

After 21 changes the folder that the export updated in steps, the file is the same as the file with the export of the same base from scratch.

Previously, exports collected HTML of all pages to compare with what lies on the disk, and cost 7 s after any change.

- The change that is visible on all pages overwrites all files — about 8 s. For disk files, this is not to be circumvented. `shugo serve` shows the same change from the next query.
- HTML Om 288 MBOn the disk, 838 MB: each page is a folder with `index.html`and each file and folder is not less 4 KB. With addresses of the type `about.html` would be about half; this is an open question about the type of addresses[10](decisions.md#10-routes)).

### Development Mode

`shugo dev` updates the site in the same way as `shugo update`, runs the server and monitors the project files. After changing the source, the project is compiled again and all pages are calculated; after changing the database, what the changed lines read is calculated; after changing other files, for example in `public/`, nothing is calculated. The pages opened in the browser are rebooted. If there is an error in the project, it is printed, and the server continues to show the last successful build.

### Speed

Shop on 100,000 products, product page — 2.8 KB HTML:

| What is measured | Result |
| --- | ---: |
| Assembling a page from storage | 58–67 µs |
| Request to server whole, 8 clients in same process, 8 cores | 16–18 thousand requests per second |
| Generate all 104,202 pages into files | 7.1 s |

With a copy in memory, the server gave 33–39 thousand requests per second: so much is the decision not to keep a copy. Details are in [README booth](../examples/catalog/README.md).

## Open questions

- Compression of answers, `ETag`, response 304.
- Multiple projects in one process: how the server selects a project on request.
- Go-code generation as acceleration ([5](decisions.md#5-the-program-of-the-whole-project-is-one-file-the-rantime-is-common)).
- The development mode compiles the entire project and reloads the page as a whole.
- If you need to give more 16–18 thousand requests per second: the cache of the finished HTML in front of the server, which is reset to the list of pages with the new HTML from the update report, or the driver SQLite on C - it reads the page on 15–20% faster.
