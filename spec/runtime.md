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

### Development Mode

`shugo dev` collects the project, runs the server and monitors the project files. After the change, the project is reassembled, and pages opened in the browser are rebooted. If there is an error in the project, it is printed, and the server continues to show the last successful build.

### Speed

Shop from the stand, 104,202 pages, one stream. Measurements — in [README booth](../examples/catalog/README.md).

## Open questions

- Compression of answers, `ETag`, response 304.
- Multiple projects in one process: how the server selects a project on request.
- Go-code generation as acceleration ([5](decisions.md#5-the-program-of-the-whole-project-is-one-file-the-rantime-is-common)).
- Memory: rantime keeps records of all pages (about 40 MB on 100,000 store pages) and tapes that have already read, up to 256 MB. For sites in millions of pages, entries will have to be read on request.
- The development mode reassembles the entire project and reloads the page as a whole.
