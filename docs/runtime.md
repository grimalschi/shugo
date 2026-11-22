# Runtime

The Go server assembles pages from records in the store.

## Decisions

- The runtime uses the site skeleton and persisted build output. For a request-bound search island, it queries the published index with the URL parameters and renders the island with the SSR program generated during the build. Go performs this rendering during the request. On November 22, 2026, the project adopted Svelte to compile and render the templates. The store now contains fragments of HTML rendered by Svelte, and the runtime assembles the page from those fragments.
- Pages and their data are determined during the build. Cookie-dependent pages, API routes, and form processing are outside the runtime's scope. Query parameters affect only request-bound search islands and the JSON search endpoint they use.
- The primary mode assembles each page on request. It can serve frequent data updates without regenerating thousands of HTML files in advance.
- The second mode exports the site as HTML files.
- The runtime always reads persisted output, regardless of when or how it was updated.
- Several processes can use one store. One updates the records and others serve pages. Each page is assembled from a single snapshot.
- The same runtime serves any project. It currently serves one project per process. On November 21, 2026, multi-project serving was deferred, with the design intended to accommodate it. Project selection on each request will be decided as part of that work.
- The Go stages can run outside containers. Project JavaScript runs in the evaluation stage.
- The original runtime escaped stored values as it inserted them. Since the November 22, 2026 adoption of Svelte, Svelte escapes HTML during rendering, as in SvelteKit. The runtime still validates each record's structure. [Record validation](#record-validation)
- Development mode rebuilds when source files change and reloads the browser page. [Live Reload proposal](proposals/live-reload.md)
- The November 20, 2026 decision requires the server to read only what the page needs from SQLite on each request. On November 21, this was confirmed even for a background-loaded cache of the full site. The design keeps per-project memory low for possible multi-project serving.
- On November 21, 2026, pages that disappeared from their route's list were marked for deletion. The server returns 404 for their URLs and export removes their files.
- Since the November 21, 2026 decision, subsequent exports write only changes. A manually edited file in the export directory stays until its page changes. Paths follow SvelteKit's prerendered output, with `/about` at `about.html`, `/` at `index.html`, and the fallback page at `404.html`.
- On November 22, 2026, compression and 304 responses for unchanged pages were approved.
- `href`, `src`, and similar URL attributes are rendered as written. The November 22, 2026 decision removed the previous substitution for `javascript:` URLs.

## Current implementation, awaiting approval

### Page assembly

The server reads the page records needed by each request.

1. Take a connection from the read pool and start a read transaction.
2. Read the build ID. Parse and cache the site skeleton, including the shell, routes, and island programs, once per build. Data updates use the same skeleton.
3. Look up the page record and its route number by URL with one key lookup. The HTML is inside the record.
4. Write the first `src/site.html` fragment, remember the `%shugo.head%` insertion point, and write the next fragment up to `%shugo.body%`. The skeleton already holds the split shell.
5. Write the root block's HTML and fill each placeholder as it is encountered. [Record format](storage.md#record)
6. Resolve a shared block placeholder with one lookup for its record. Reuse the record if the same page shows the block again.
7. Resolve a snippet placeholder inside a shared block with the blocks supplied by its caller. The reference to the shared block identifies those blocks.
8. At an island placeholder, write the stored HTML, hydration data, and browser module link. If a `request:` binding connects the island to search and the URL contains search parameters, read the index version in the same transaction. Query it with those parameters and render the island with its SSR program.
9. Write any framework markup for the body, such as the development reload script, followed by the last shell fragment.
10. Insert the collected `<head>` fragments at `%shugo.head%` in render order. Keep only the last `<title>` and add the stylesheet link. [Page head](blocks.md#page-head)
11. Close the read transaction and return the connection to the pool.

A file route carries an extension. For example, `/sitemap.xml` comes from `src/pages/sitemap.xml.ts`. Its record contains a single block with the text returned by `GET`. The server sends that text directly.

The entire response uses one snapshot, even if an update commits while it is being assembled. The next request sees the committed state automatically.

The read pool has twice as many connections as CPU cores. Each connection has prepared statements for key lookups. The store file is memory-mapped with `mmap`, so SQLite can read the pages held by the operating system without copying them.

### HTTP responses

| Request | Response |
| --- | --- |
| Page URL | 200 with HTML, `Cache-Control: no-cache`, and `ETag` |
| URL produced by a file route such as `/sitemap.xml` | Same caching as a page, with a content type from `mime.TypeByExtension`, as for files in `public/`. Unknown extensions use `application/octet-stream`. |
| Page URL with a matching tag in `If-None-Match` | 304 with no body |
| Page URL with a trailing slash | 308 to the URL without the slash |
| URL in `redirects` | 301 |
| File from `public/` | The file |
| `/_assets/…` | A build asset with a permanent caching policy. The filename changes with the content. |
| `/_islands/search/<search-id>?…` | JSON from `search(…)` using the published index. The build generates the search ID. |
| Page with a request-bound search island and search query parameters | 200 with filtered HTML, `X-Robots-Tag: noindex`, and a `rel="canonical"` link to the URL without parameters |
| Unmatched URL | 404 with `src/pages/404.svelte` if present |
| Method other than `GET` or `HEAD` | 405 |

The next request sees an update committed by another process.

### Compression and 304 responses

- A page's `ETag` is a CRC-64 checksum of its HTML. It is weak (`W/"…"`) because the same page can be sent compressed or uncompressed. A changed page receives a different tag. A matching `If-None-Match` returns 304 without a response body.
- HTML and text files such as `.css`, `.js`, `.svg`, `.json`, `.txt`, and `.xml` are gzip-compressed from 1 KB upward when the client accepts gzip. Smaller responses, images, fonts, and responses to `Range` requests are sent as-is.
- A response that can be compressed includes `Vary: Accept-Encoding`, so an upstream cache can keep the encodings separate.
- Gzip level 1 reduced a product page from 2.5 KB to 1.1 KB. The measured request time was 52 µs with gzip and 59 µs without it because less data crossed the connection. The benchmark used eight clients and 1,000 products (`go test ./test/ -run xxx -bench Serve`).
- The server does not retain compressed responses. Brotli usually compresses HTML better than gzip. It could be added through a pure Go package if needed because the Go standard library has no Brotli encoder.

### SQLite read performance

The first implementation kept a copy of every page record in memory and rebuilt it after each update. With 100,000 pages, the first request after every update waited another 0.2 s. The project author chose to serve pages from SQLite directly. The November 21, 2026 benchmark used a copy of the store for a 100,000-product shop while it still stored value tapes. It measured 20,000 sequential requests for random pages, with one read transaction per page. It compared the current pure Go driver with a C driver accessed through cgo.

| Read strategy | Go driver | C driver |
| --- | ---: | ---: |
| One query per record and per tape, with tapes in a separate table | 142–150 µs | 120 µs |
| Same, with the file mapped into memory (`mmap`) | 107–112 µs | 92–95 µs |
| Batch all shared blocks, then batch all tapes | 120–126 µs | 95–103 µs |
| One SQL query for the whole page | 124–142 µs | 92–121 µs |
| **Tapes inside records**, one query per record, with `mmap` | **61–65 µs** | **53–55 µs** |
| Same, without `mmap` | 74–76 µs | 66–69 µs |
| In-memory copy for comparison | 9 µs | — |

The number of key lookups dominated the time. Batching SQL queries did little to help. A shop page uses its own record and around 10 shared blocks. A separate tape table added about 15 lookups. The runtime therefore uses the Go driver, maps the file into memory, and keeps render data inside the records. The records have held HTML since November 22, 2026.

### Record validation

The evaluation stage runs project code and writes records. Svelte escapes `&` and `<` in text and also escapes `"` in attributes. `{@html}` inserts raw HTML. A file route stores the `GET` result verbatim as the file contents. The runtime uses the stored output and validates the record structure.

- The record decoder checks declared lengths. Before accepting a record, Go checks that its blocks form one tree and each has exactly one place in that tree. [Invalid records](storage.md#how-the-recording-can-not-come-together)
- Before accepting a request-bound island, Go checks that its SSR program produces the same bytes as Svelte for the stored props. Go also escapes island props and the HTML substitutions it renders during a request.
- An unknown placeholder type, a missing shared block, or a recursive reference to a shared block returns 500.
- Shared blocks can nest up to 256 levels. The page limit is 64 MB.

The skeleton describes the site as data. An invalid skeleton or record can cause invalid HTML or an error.

### Export

`shugo export` writes the site in the same layout as SvelteKit's prerendered pages. It writes `/about` to `about.html`, `/blog/2` to `blog/2.html`, `/` to `index.html`, and the fallback page to `404.html`. It includes files from `public/` and `_assets/` alongside them. Static hosts can serve `/about` from `about.html`. For nginx, use `try_files $uri $uri.html =404`. Module routes retain their file paths, so `/sitemap.xml` becomes `sitemap.xml` and `/api/jug.json` becomes `api/jug.json`. Export derives the output name from the route number stored with each record.

Export resumes from the previous run. Its `.shugo-export.json` manifest records the store and version exported, copied files with their source size and modification time, and the module-generated files in `files`. On the next export from the same store, it does the following.

- Regenerates pages whose own record changed after that version, as well as pages that use a changed shared block directly or indirectly. It finds them through the version indexes and the `shows` table.
- Deletes files for pages that disappeared after that version. The `gone` table retains each removed page's URL and route, so export can delete the correct path, such as `about.html` or `sitemap.xml`.
- Copies only assets whose source size or modification time changed.

The work scales with the changes. Updates already write a version with every changed record, which supplies the information export needs.

With no manifest, export writes all pages and leaves existing files in place. If the manifest refers to a different store or predates a full rebuild, export regenerates all pages. It deletes `.html` files that are neither site pages nor copied files, along with previously exported file routes the site no longer produces. It leaves other files and dot-prefixed directories alone and skips rewriting files whose contents are unchanged.

The November 22, 2026 benchmark used Bun and ext4 with 100,000 products and 104,202 pages. It ran `shugo update` and then `shugo export` after each change.

| Change | Pages with changed HTML | Update | Export |
| --- | ---: | ---: | ---: |
| First export to an empty directory | All | — | 3.5 s |
| No changes | 0 | 28 ms | 6 ms |
| One product price | 7 | 31 ms | 7 ms |
| Prices of 20 products | 70 | 37 ms | 14 ms |
| Delete the last product and remove its file | 88 | 0.43 s | 23 ms |
| Rename the featured product in the shared header | All | 0.13 s | 4.4 s |
| Add a product and update the navigation counts | All | 0.49 s | 4.5 s |

On November 21, 2026, pages were still assembled from value tapes using the template skeleton. Initial export took 5.0 s and export after a header change took 8.2 s. Assembling rendered HTML fragments almost halves the export time after that header change.

After 21 changes, the incrementally exported directory matched an export of the same database into an empty directory file for file.

Before November 21, 2026, export generated all pages to compare the HTML with files on disk. That took 7 s after any change.

- A change visible on every page rewrites every file in about 4.5 s. `shugo serve` shows the same change on the next request.

### Development mode

`shugo dev` runs the same update as `shugo update`, starts the server, and watches project files. A change to project sources, including `src/site.html`, recompiles the project and evaluates all pages. A database change evaluates the readers of the changed data. Other changes, such as changes in `public/`, require no evaluation. After each update, open browser pages reload. The server inserts a script after the page content at `%shugo.body%`, and the script listens on `/_shugo/live` with `EventSource`. If a build fails, the server reports the error and continues serving the last successful build.

The [Live Reload proposal](proposals/live-reload.md) describes ways to shorten feedback on code edits in large projects.

### Performance

The November 22, 2026 benchmark used 100,000 products. The product page was 2.5 KB of HTML.

| Measurement | Result |
| --- | ---: |
| Assemble a product page from the store | 64 µs |
| Full HTTP request, eight clients in the same process, eight CPU cores | About 16,000 requests per second |
| Export all 104,202 pages | 3.5 s |

On November 21, 2026, the server delivered 16,000–18,000 requests per second with value tapes inside the records. An in-memory copy of the site delivered 33,000–39,000 requests per second before it was removed on November 20, 2026. [Catalog benchmark](../examples/catalog/README.md)

## Open questions

- For throughput beyond 16,000 requests per second, a cache in front of the server could store assembled HTML and invalidate URLs from the update report's list of changed pages.
