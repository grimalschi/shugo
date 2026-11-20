# Runtime

Rantime is a server on Go that collects HTML pages.

## Decisions

- Rantime uses the compilation result (1 step) and stored expression values (3 step).
- Two modes:
  - a server that collects pages on the fly as quickly as possible;
  - one-time generation of all HTML files of the site.
- Rantime always takes meanings from the saved and doesn't know how or when they were updated.

## From the prototype, pending approval

- The stored values for the runtime are untrusted data. Each value is screened by insertion location. If values do not match the page, the runtime returns an error, rather than a corrupted HTML.
- Multiple processes can work with one storage: one updates the values, others give the pages. The page is entirely collected from one data state.

## Open questions

- Go-code pages or a page program that performs one common runtime.
- One project per process or multiple projects in one process.
- File generation after updating: all pages or only changed.
- Static file output, compression, caching headers.
- Development mode: reassembly when changing files and updating the page in the browser.
