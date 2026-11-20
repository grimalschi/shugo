# Catalog and incremental update benchmark

A shop with a configurable number of products, a page for each product, category listings of 24 products per page, a header with navigation and a featured product, and recommendations.

Variant 4 stores the shop data in SQLite. The data-loading code reads it through `Shugo.query`, and templates work on a copy of what they read. Queries omit the columns, and each page depends on the columns it actually reads. The benchmark checks that a data change reevaluates only the readers of the changed values. See the [design and decisions](../../VARIANT.md).

## Run the benchmark

```sh
go build -o bin/shugo ./cmd/shugo                    # from the repository root
node examples/catalog/stand.mjs --size 10000         # or use Bun. Pass --runtime node to evaluate with Node.
```

The benchmark runs the following steps.

1. Generates `data/catalog.db` for `--size` products. The same size always produces the same shop.
2. Builds the site with `shugo build`.
3. Changes the data 21 times using SQL and runs `shugo update` after each change. It checks the report against the expected evaluations and changes.
4. Rebuilds the site from the same data in another directory and compares the resulting files.

To view the site, run `node examples/catalog/lib/generate.mjs 1000`, then `bin/shugo dev examples/catalog`.

## How the example works

| File | What it shows |
| --- | --- |
| `lib/generate.mjs` | Shop tables. Each has a single-column key and a separate ordering column. |
| `components/Header.shugo` | Shared block with no parameters. It reads its own configuration, navigation, and category counts. |
| `components/ProductCard.shugo` | Shared block that takes a product ID and reads the product. A product edit reevaluates the card on its own. The query loads a complete product, and the card depends on the four columns that it reads. |
| `components/Variant.shugo` | Regular component that displays a row read by the page. The page passes the full row, so it depends on every column. |
| `pages/products/[slug].shugo` | The route lists URLs, and each page queries its own product, variants, reviews, badges, and recommendations. The data-loading code computes the discounted price. |
| `pages/categories/[category]/[...page].shugo` | The route uses `paginate` with a product count. The page uses `Shugo.paginate` with a query. |
| `lib/money.ts` | Price helpers available to templates without loading data. |

The list of product URLs is the only query in the example that uses `select`, with just the `slug` column. It produces the same result and reads 100,000 single-column rows three times faster than reading the full rows.

## Results

The [benchmark](../../VARIANT.md#benchmark) has tables for every size, initial build times, and analysis. With 100,000 products and 104,202 pages, changing one price evaluates one page and one card, and the full `shugo update` command takes 37 ms. Changing an unread column for 5,000 products evaluates nothing. The initial build takes 14.0–14.5 s.
