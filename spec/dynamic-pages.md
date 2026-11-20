# Dynamic pages

Dynamic page - one file, which collects many pages. Their list is taken from the data.

## Decisions

- The dynamic page list returns the data code in the page file, `getStaticPaths`, as in Next.js and Astro.

## From the prototype, pending approval

- Data for many pages is better to be received with one query for all, rather than a request for each page.

## Open questions

- What returns `getStaticPaths`: only address options or data of each page at once.
- Do you need a separate step "data of one page on its parameters".
- Breaking long lists into pages.
- What happens to a page that is missing from the list?
