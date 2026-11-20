# Dynamic pages

Dynamic page - one file, which collects many pages. Their list is taken from the data.

## Decisions

- The dynamic page list returns `getStaticPaths` in the page file code, as in Next.js and Astro.
- `getStaticPaths` is a way to get data for all pages at once with one request, rather than a request for each.

## Made in the first version, waiting for approval

The reasons are in [11](decisions.md#11-getstaticpaths).

```
<!-- pages/products/[slug].shugo -->
---
import { getProducts, type Product } from "@/lib/shop"

export async function getStaticPaths() {
  const products = await getProducts()            // one request for all pages
  return products.map((product) => ({
    params: { slug: product.slug },                // part of address
    props: { product },                            // this page
  }))
}

const { product } = Shugo.props as { product: Product }
---
<h1>{product.title}</h1>
<p>Address: {Shugo.url.pathname}, Parameter: {Shugo.params.slug}</p>
```

- `getStaticPaths` is called once for calculation and returns the list of `{ params, props }`. `props` is optional.
- The page code is executed for each item in the list. It has `Shugo.params` - address parameters, `Shugo.props` - `props` of this item.
- `getStaticPaths` does not see variables of the rest of the page code: only imports.
- The value of the parameter is non-empty text; the number is converted to text. In `[...path]`, the value can contain `/`, and `undefined` means an address without remainder.

### Pagination

```
<!-- pages/blog/[...page].shugo -->
---
export async function getStaticPaths({ paginate }) {
  return paginate(await getPosts(), { pageSize: 10 })
}
const { page } = Shugo.props
---
<g-for each={page.data} as="post">…</g-for>
<g-if cond={page.url.next}><a href={page.url.next}>Next</a></g-if>
```

`paginate` works as in Astro. In `Shugo.props.page` Ladder `data`, `currentPage`, `lastPage`, `total`, `size`, `start`, `end` and `url` with addresses `current`, `prev`, `next`, `first`, `last`C. `[...page]` first page gets address without number, with `[page]` — `/1`.

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
