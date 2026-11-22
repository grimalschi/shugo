# Runtime

Rantime is a server on Go that collects HTML pages from storage.

## Decisions

- Rantime uses the project framework and stored values. The project code it does not execute, requests to data does not do, does not calculate anything.
- All pages and all data are known in advance. There are no pages that depend on cookies or query strings, API routes and processing forms.
- The main mode is to assemble the page on the fly with each request. When data changes often, writing thousands of HTML files in advance makes no sense: the runtime collects a page from ready-made pieces very quickly.
- The second mode is a one-time generation of all HTML-files of the site.
- Rantime always takes meanings from the saved and doesn't know how or when they were updated.
- Several processes work with one storage: one updates the values, others give the pages. The page is entirely collected from one data state.
- The common runtime, not the one collected for the project. One project per process; multi-project service is foreseen for the future. Confirmed by 21 on November 2026: Several projects in one process are postponed, but the Rantime device should allow this. How the server selects a project on request will be decided then.
- Everything that runs Go is safe and can work outside containers.
- The stored values for the runtime are untrusted data: each is screened at the insertion location.
- Development mode is needed: reassembly when changing files and updating a page in the browser. How to speed it up - [live Reload](proposals/live-reload.md).
- The server does not keep the site in memory: for each request, it gets from SQLite what the page needs. Storage requests should be optimized. Solved 20 November 2026 year. Confirmed by 21 November: there will be no copies in memory and as acceleration, including a copy that loads in the background after the first request runtime in the future can serve many sites.
- The page that is no longer in the route list is deleted: the server responds to its 404 address, the export deletes its file. It was solved by 21 on November 2026 year.
- After the first time, exports only write what has changed. The folder belongs to the export: a file corrected by hand in it lives until its page is changed. `/about` exports page writes in `about.html`, `/` in `index.html`, page for non-existent addresses in `404.html` as SvelteKit writes pages that build in advance. Solved 21 November 2026 year.
- The server compresses the answers and responds 304 to the page that has not changed. Solved 22 November 2026 year.
- Addresses in `href`, `src` and similar attributes are displayed as is in Svelte. Resolved by 22 on November 2026: the previous `javascript:` address change is canceled.

## Implemented, pending approval

### Page assembly

There are no copies of the site in the server memory. For each request, it:

1. It takes a connection from the reading pool to the storage and opens a reading transaction.
2. Reads the build name. It parses the templates once for each build and keeps them in memory: it is code, not data, and updating the data does not change it.
3. Finds the page record and its route number - one search by key. Values ribbons lie inside the record.
4. Writes the `src/site.html` shell to the `%shugo.head%` location, remembers this location and writes the shell further, to the `%shugo.body%`. The shell lies in the frame already cut through these locations.
5. It follows the framework of the template of the first block. The markup is copied, the next value is taken from the block tape and screens it on the place of expression.
6. At the call point, the component moves to the next sub-block. The shared block reads from its own record, also by a single search; the block shown on the page twice is read once.
7. In place, `{@render}` shows a snippet. The snippet with the parameters is the next nested block. The snippet without the parameters that the caller gave is blocks from the caller's record: the component block has written what blocks they are ([Storage](storage.md#tape)).
8. When the page is shown, it adds a body framework markupin development mode, this is a reboot scriptand the rest of the shell.
9. The `<svelte:head>` blocks of each template that he showed he memorized along the way. Now shows them in order, from `<title>` - only the last one, adds a link to styles and inserts all this into the remembered place of `%shugo.head%` ([Blocks](blocks.md#page-head)).
10. Closes the transaction and returns the connection to the pool.

The file is not in HTML, for example `/sitemap.xml` from the `src/pages/sitemap.xml.ts` module, the server recognizes by route: the module route has an extension. The record of such a file is one block with the text that returned `GET`; the server gives this text as it is, without shell and without head.

The entire page is assembled from one state: the update that ended in the middle does not affect it. The next request already sees it - without a storage survey and without re-reading.

The connections in the pool are twice as large as the cores; everyone has pre-prepared queries for the key. The storage file is displayed in memory (`mmap`): SQLite reads its pages where the system holds them without copying.

### What the server answers

| Request | Response |
| --- | --- |
| page Address | 200, HTML, `Cache-Control: no-cache`, `ETag` |
| the file address that gives the module among pages, for example `/sitemap.xml` | same, but type — by extension, through `mime.TypeByExtension`, as in files from `public/`; unknown extension — `application/octet-stream` |
| the address of the page with `If-None-Match`, where is its `ETag` | 304 without body |
| address of the page with a slash at the end | 308 to address without slash |
| address from `redirects` | 301 |
| file from `public/` | file |
| `/_assets/…` | file collected by the compiler; cached forever: name changes with content |
| everything else | 404 and page `src/pages/404.shugo`, if any |
| not `GET` and not `HEAD` | 405 |

An update made by another process, the server sees from the next request.

### Compression and 304

- `ETag` pages are the checksum of its HTML (CRC-64), weak: `W/"…"` because the page leaves compressed or not. The page has changed the label has changed. The browser that has the page asks with a tag (`If-None-Match`), and the same page gets 304 without a body.
- Page and text file — `.css`, `.js`, `.svg`, `.json`, `.txt`, `.xml` and the like — from 1 KB go compressed gzip if the browser wrote that it understands. Less 1 KB, pictures and fonts as is. The answer to the request for a part of the file (`Range`) is as it is.
- The answer that can come compressed is `Vary: Accept-Encoding` so that the cache in front of the server does not give the compressed to someone who did not ask.
- Compression fast (gzip, level 1) and does not slow down the server: on the store page of the goods goes 1.1 KB instead of 2.5 KB, and the request with gzip takes 52 µs Against 59 µs without it - less data goes through the connection8 customers, 1,000 goods, `go test ./test/ -run xxx -bench Serve`).
- Compressed answers are not stored: the server does not keep the site in memory. Brotli usually compresses HTML better than gzip, but there is no standard library Go; if necessary, it can be added in a separate package on pure Go.

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

- text and attribute values are always saved, as in Svelte: in the text of `&` and `<`, in the attribute of `"`; without the screen, only the value of `{@html}` is inserted and the file text is not in HTML, which returned `GET`: it and the file;
- attribute name from `{...attributes}` and tag name `<svelte:element>` are checked;
- tape that does not match the template gives the answer 500, and not a corrupted page - as it happens, described in [Storage](storage.md#how-the-tape-can-not-fit-with-the-page);
- the nesting of components is limited by 256 levels, the page size is 64 MB.

The project framework is data, not code. The worst thing that a wrong frame or tape can do is to have a wrong HTML or an error.

### Export

`shugo export` Write the site in a folder as SvelteKit Writes pages that build in advance: page `/about` the National Museum `about.html`, `/blog/2` the National Museum `blog/2.html`, `/` the National Museum `index.html`, page 404 the National Museum `404.html`; near `public/` and `_assets/`. Static hostings give away `/about` the City of `about.html` themselves; nginx with `try_files $uri $uri.html =404`. The file that gives the module among the pages is written under its address: `/sitemap.xml` the National Museum `sitemap.xml`, `/api/jug.json` the National Museum `api/jug.json`. How to name a file, export will know the route of the page: the repository gives the route number along with the record.

Exports follow the previous one. In the folder, he leaves a list of `.shugo-export.json`: what storage and what version of it he wrote down, which files he copied with the size and time of the original change and which files gave modules among the pages (`files`). The following exports from the same storage:

- builds anew pages whose entry has changed after this version, and pages that show the shared block with the changed entry, directly or through other common blocks. It finds them by the version indexes and the `shows` table;
- deletes files of pages that were missing after this version. The `gone` table stores the address of the missing page along with its route, so the export deletes the desired file: `about.html` or `sitemap.xml`;
- copies only files that have changed size or time of change.

His work grows with the change, not with the site. Update for this nothing to note: version of the recording it writes along with the recording itself.

If there is no list, exports build all pages and do not delete anything. If the list is from another storage or version before the project is completely reassembled, the export builds all pages and removes `.html` files from the folder, which are not among the pages of the site or among the copied files, as well as module files from the previous list, which the site no longer gives. The rest of the files and folders whose name starts with a dot, it does not touch. The file with the previous content is not rewritten.

Measured on the store in 100,000 products, 104,202 pages, Bun disk ext4, one run of 21 November 2026 year. After each change — `shugo update`, then `shugo export`:

| What changed | Pages with new HTML | Update | Export |
| --- | ---: | ---: | ---: |
| First export to an empty folder | all | — | 5.0 s |
| Nothing. | 0 | 0.16 s | 7 ms |
| Price of one product | 7 | 29 ms | 8 ms |
| Prices 20 products | 70 | 34 ms | 17 ms |
| Last product removed: 88 pages, one file removed | 88 | 0.40 s | 36 ms |
| Product name of the month: header on all pages | all | 0.12 s | 8.2 s |
| Product added: counters in the menu on all pages | all | 0.48 s | 8.3 s |

After 21 changes the folder that the export updated in steps, the file is the same as the file with the export of the same base from scratch.

Previously, exports collected HTML of all pages to compare with what lies on the disk, and cost 7 s after any change.

- The change that is visible on all pages overwrites all files — about 8 s. For disk files, this is not to be circumvented. `shugo serve` shows the same change from the next query.

### Development Mode

`shugo dev` updates the site in the same way as `shugo update`, runs the server and monitors the project files. After changing the source code including the `src/site.html` shell the project is compiled anew and all pages are calculated; after changing the database, what the changed lines read is calculated; after changing other files, for example, in `public/`, nothing is calculated. After each update, the pages opened in the browser are rebooted completely: the server adds to each page, immediately after it, in place of `%shugo.body%`, a script that listens to `/_shugo/live` (`EventSource`). If there is an error in the project, it is printed, and the server continues to show the last successful build.

How to make code editing in a large project not wait for the calculation of the entire site [live Reload](proposals/live-reload.md).

### Speed

Shop on 100,000 products, product page — 2.5 KB HTML:

| What is measured | Result |
| --- | ---: |
| Assembling a page from storage | 58–67 µs |
| Request to server whole, 8 clients in same process, 8 cores | 16–18 thousand requests per second |
| Generate all 104,202 pages into files | 5.0 s |

With a copy in memory, the server gave 33–39 thousand requests per second: so much is the decision not to keep a copy. Details are in [README booth](../examples/catalog/README.md).

## Open questions

- If you need to give more 16–18 thousand requests per second: cache the finished HTML in front of the server, which is reset to a list of pages with the new HTML from the update report.
