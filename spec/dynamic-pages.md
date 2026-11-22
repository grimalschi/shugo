# Dynamic pages

Dynamic page - one file, which collects many pages. Their list is taken from the data.

## Decisions

- The dynamic page list returns `getStaticPaths` in the page file code.
- `getStaticPaths` is a way to get data for all pages at once with one request, rather than a request for each. The *Changed 22 November 2026 year:* route only gives addresses, the `{ params }`, and each page takes data to its own address. `props` and `paginate(list)` are removed. So changing one product calculates one page, not all the route pages; in SvelteKit, `entries` also returns only parameters.
- The page that is missing from the list is deleted: the server responds to its address 404, `shugo export` deletes its file (21 solution November 2026 year).
- The address that the list returned twice is one page, without error, as in SvelteKit (22 solution of November 2026 year).
- `query`, `paginate` and `page` are imported from the `shugo` module everywhere, and in `getStaticPaths` too: the function gets nothing by argument (22 solution November 2026 year).

## Implemented, pending approval

```svelte
<!-- src/pages/products/[slug].svelte -->
<script module>
  import { query, page } from "shugo"

  export function getStaticPaths() {
    return query("products").select("slug").all()   // one request for all pages
      .map(({ slug }) => ({ params: { slug } }))     // part of address
  }
</script>

<script>
  const product = query("products").where({ slug: page.params.slug }).first()
</script>

<h1>{product.title}</h1>
<p>Address: {page.url.pathname}, Parameter: {page.params.slug}</p>
```

- `getStaticPaths` is exported from `<script module>`: it runs once per route, and `<script>` is for each page. `getStaticPaths` in `<script>` is a build error with a hint.
- `getStaticPaths` is synchronous and receives nothing. It takes data from the project database via `query` from the `shugo` module. The function that awaits arguments, `getStaticPaths({ paginate })`, gets an error with the hint of what to import.
- It is called when the page list needs to be read: during assembly and when the string that he read has changed. Returns the list of `{ params }`. The element with `props` is a build error with a hint.
- The page code is for each address in the list. It has `page.params` address settings; the page asks for its data. `page` is read in the page and in what it shows, not in `getStaticPaths`: there is no page yet.
- The value of the parameter is non-empty text; the number is converted to text. In `[...path]`, the value can contain `/`, and `undefined` means an address without remainder.

### Pagination

```svelte
<!-- src/pages/blog/[...page].svelte -->
<script module>
  import { query, paginate } from "shugo"

  export function getStaticPaths() {
    return paginate(query("posts").count(), { pageSize: 10 })
  }
</script>

<script>
  const list = paginate(query("posts").order("date", "desc"), { pageSize: 10 })
</script>

{#each list.data as post}…{/each}
{#if list.url.next}<a href={list.url.next}>Next</a>{/if}
```

`paginate` the City of London `getStaticPaths` It gets the number of items and tells you which pages are available. `paginate` in the page receives a request or array and returns its part: the request asks only this part of the database. This is one function: what to do, it learns from where it was called. In response lie `data`, `currentPage`, `lastPage`, `total`, `size`, `start`, `end` and `url` with addresses `current`, `prev`, `next`, `first`, `last`C. `[...page]` first page gets address without number, with `[page]` — `/1`.

### Errors

The assembly stops with a clear message if:

- file with parameter does not have `getStaticPaths`;
- function returned not a list;
- the item does not have `params` or they do not have the desired parameter;
- the element has `props`.

## Open questions

- In SvelteKit, the list of route pages is a function of `entries` in `+page.ts`, not in the component itself. In Shugo, `getStaticPaths` stands in `<script module>` pages.
