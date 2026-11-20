# Catalog and incremental update benchmark

A shop with a configurable number of products, a page for each product, category listings of 24 products per page, a header with navigation and a featured product, and recommendations.

The benchmark checks that a data change reevaluates and updates only the parts of the site that display that data.

## Run the benchmark

```sh
go build -o bin/shugo ./cmd/shugo                    # from the repository root
node examples/catalog/stand.mjs --size 10000         # or use Bun. Pass --runtime node to evaluate with Node.
```

The benchmark runs the following steps.

1. Generates shop data for `--size` products. The same size always produces the same shop.
2. Builds the site with `shugo build`.
3. Changes the data 21 times and runs `shugo update` after each change. It checks the update report against the expected evaluations.
4. Rebuilds the site from the same data in another directory and compares the resulting files.

To view the site, run `node examples/catalog/lib/generate.mjs 1000`, then `bin/shugo dev examples/catalog`.

## How the example works

| File | What it shows |
| --- | --- |
| `lib/catalog.ts` | Shop data. The module reads a file and only the data-loading code imports it. |
| `lib/money.ts` | Price helpers available to templates without loading data. |
| `components/Header.shugo` | Shared block with no parameters. It loads its own data and runs once for the site, even though it appears on every page. |
| `components/ProductCard.shugo` | Shared block with one product ID parameter. It fetches its own product and runs again when the displayed data changes. |
| `components/Variant.shugo` | Regular component that displays an input object in a block owned by its page. |
| `pages/products/[slug].shugo` | One file produces 100,000 pages. All page data arrives in a single list. |
| `pages/categories/[category]/[...page].shugo` | Pagination with `paginate`. |
| `pages/+layout.shugo` | Layout with a `head` slot for the page title. |

## Results

This branch tests dependency-tree fingerprinting. [Variant notes](../../VARIANT.md) show what each update evaluates and how long it takes.

With 100,000 products, a price edit evaluates one page and one card out of 204,000 pages and shared blocks. An edit to a field the site never displays triggers no evaluations. In preliminary Bun measurements, an update took 1.2–1.4 s, down from 3.2–3.7 s in the first version.
