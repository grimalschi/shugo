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

## Made in the first version, waiting for approval

### Page assembly

The server keeps the structure of the site in one state: records of all pages and common blocks. Tapes are read from the repository at the first access and remain in memory: under its hash, the tape does not change.

1. The page is being recorded.
2. Runtime follows the pattern program of the first block. The markup is copied, the next value is taken from the block tape and screens it on the place of expression.
3. At the call point, the component moves to the next nested block. The shared block is taken from its own record.
4. In place, `<g-slot>` shows the blocks that the caller has transmitted.

The entire page is assembled from one state: the update that ended in the middle does not affect it.

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

The server notices the update of the storage itself. The update made by another process, it sees no later than through 50 ms and re-reads the structure of the site; before that corresponds to the previous state.

### What Rantime Doesn't Trust

Values came from the stage that executed the project code. Therefore:

- text and attribute values are always screened; without screening, only the value of `<g-html>` is inserted;
- address in `href`, `src` and similar attributes is replaced if it can execute the code ([14](decisions.md#14-addresses-in-links-are-checked));
- attribute name from `{...attributes}` is checked;
- tape that does not match the template gives the answer 500, and not a corrupted page - as it happens, described in [Storage](storage.md#how-the-tape-can-not-fit-with-the-page);
- the nesting of components is limited by 256 levels, the page size is 64 MB.

The project program is data, not code. The worst thing that a wrong program or tape can do is to have HTML or an error.

### Export

`shugo export` collects all pages and writes them to a folder with `public/` and `_assets/`. The rules are in [21](decisions.md#21-export-to-files).

#### How much does it cost to export after the change

Measured on the fifth trial version ([analysis of changes](change-tracking.md)): store on 100,000 products, 104,202 pages, disk ext4, Bun. After each change — `shugo update`, then `shugo export`.

| What changed | Pages with new HTML | Update | Export | Files recorded |
| --- | ---: | ---: | ---: | ---: |
| First export to an empty folder | all | — | 12.3 s | 104 203 |
| Nothing. | 0 | 0.23 s | 7.1 s | 0 |
| Price of one product | 7 | 0.04 s | 7.0 s | 7 |
| Prices 1,000 products | 1 129 | 0.45 s | 7.0 s | 1 129 |
| Product name of the month: header on all pages | all | 0.17 s | 12.8 s | 104 202 |
| Product added: counters in the menu on all pages | all | 0.45 s | 12.5 s | 104 203 |

Export only writes different files, but collects HTML of all pages to compare with what lies on the disk. Therefore, it costs 7 s even if one price has changed.

The update knows which pages HTML has become different. Exports that collect and write only them are measured separately:

| Pages | Build HTML | Write Files |
| ---: | ---: | ---: |
| 7 | 1.4–1.8 ms | 0.4–0.5 ms |
| 100 | 12–13 ms | 4–6 ms |
| 1 129 | 132–135 ms | 47 ms |
| 20 000 | 1.9 s | 0.8 s |
| 104 202 | 6.7–6.8 s | 4.2 s |

To this is added the reading of the site device, 0.2–0.27 s, - the same thing that the server does after updating. If you read only the records of the desired pages, it will not be.

- The change that is visible on all pages overwrites all files: around 11 s even when exported by list. For disk files, this is not to be circumvented. `shugo serve` shows the same change through 0.1–0.3 s.
- HTML Om 288 MBOn the disk, 838 MB: each page is a folder with `index.html`and each file and folder is not less 4 KB. With addresses of the type `about.html` would be about half; this is an open question about the type of addresses[10](decisions.md#10-routes)).

### Development Mode

`shugo dev` collects the project, runs the server and monitors the project files. After the change, the project is reassembled, and pages opened in the browser are rebooted. If there is an error in the project, it is printed, and the server continues to show the last successful build.

### Speed

Shop from the stand on 100,000 products, product page — 2.7 KB HTML of 12 blocks:

| What is measured | Result |
| --- | ---: |
| Page assembly, tape is already in memory | 7 µs |
| Page assembly when you first access it | 47 µs |
| Request to server whole, 8 clients on same machine | 33–39 thousand requests per second |
| Generate all 104,202 pages into files | 8.5 s |

Details are in [README booth](../examples/catalog/README.md).

## Open questions

- Compression of answers, `ETag`, response 304.
- Multiple projects in one process: how the server selects a project on request.
- Go-code generation as acceleration ([5](decisions.md#5-the-program-of-the-whole-project-is-one-file-the-rantime-is-common)).
- Memory: rantime keeps records of all pages (about 40 MB on 100,000 store pages) and tapes that have already read, up to 256 MB. For sites in millions of pages, entries will have to be read on request.
- The development mode reassembles the entire project and reloads the page as a whole.
