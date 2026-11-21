# Dynamic pages

Dynamic page - one file, which collects many pages. Their list is taken from the data.

## Decisions

- The dynamic page list returns `getStaticPaths` in the page file code, as in Next.js and Astro.
- `getStaticPaths` is a way to get data for all pages at once with one request, rather than a request for each.

## Implemented, pending approval

The reasons are in [11](decisions.md#11-getstaticpaths).

```
<!-- pages/products/[slug].shugo -->
---
export function getStaticPaths({ query }) {
  return query("products").select("slug").all()   // one request for all pages
    .map(({ slug }) => ({ params: { slug } }))     // part of address
}

const product = Shugo.query("products").where({ slug: Shugo.params.slug }).first()
---
<h1>{product.title}</h1>
<p>Address: {Shugo.url.pathname}, Parameter: {Shugo.params.slug}</p>
```

- `getStaticPaths` is synchronous. It takes data from the project database through `query`, which comes into function.
- It is called when the page list needs to be read: during assembly and when the line he read has changed. Returns the list of `{ params, props }`. `props` is optional.
- It is better to return only `params`, and let the page take it yourself through `Shugo.query`. Changing one product will then calculate one page. If the route is given by `props`, what the page shows only knows the route: after changing the line it has read, all its pages are calculated.
- The page code is executed for each item in the list. It has `Shugo.params` - address parameters, `Shugo.props` - `props` of this item.
- `getStaticPaths` does not see variables of the rest of the page code: only imports.
- The value of the parameter is non-empty text; the number is converted to text. In `[...path]`, the value can contain `/`, and `undefined` means an address without remainder.

### Pagination

```
<!-- pages/blog/[...page].shugo -->
---
export function getStaticPaths({ query, paginate }) {
  return paginate(query("posts").count(), { pageSize: 10 })
}
const page = Shugo.paginate(Shugo.query("posts").order("date", "desc"), { pageSize: 10 })
---
<g-for each={page.data} as="post">…</g-for>
<g-if cond={page.url.next}><a href={page.url.next}>Next</a></g-if>
```

`paginate` the City of London `getStaticPaths` It gets the number of items and tells you which pages are available. `Shugo.paginate` in the page receives a request and returns part of it. In response lie `data`, `currentPage`, `lastPage`, `total`, `size`, `start`, `end` and `url` with addresses `current`, `prev`, `next`, `first`, `last`as in Astro. C `[...page]` first page gets address without number, with `[page]` — `/1`. `paginate` accepts the list itself, as in Astro, and distributes parts of it through `props` At the same price as `props` above.

### Errors

The assembly stops with a clear message if:

- file with parameter does not have `getStaticPaths`;
- function returned not a list;
- the item does not have `params` or they do not have the desired parameter;
- the same address is returned twice.

### Page is missing from the list

It is removed from the storage. The server responds to its address 404, and `shugo export` deletes its file.

## Open questions

- Same address from two list items: error now. Astro has this warning.
- Very large lists: 100,000 elements with `props` live entirely in the memory of the JS process.
- Do you need `props` from `getStaticPaths` now that the data is in the database and the page quickly gets its line itself. Because of them, changing one line that you read the route calculates all its pages. The decision above that `getStaticPaths` receives data for all pages with one request was made when data was downloaded from sources over the network.
