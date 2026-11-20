# Catalog and incremental update benchmark

A shop with a configurable number of products, a page for each product, category listings of 24 products per page, a header with navigation and a featured product, and recommendations.

Variant 2 stores the shop data in SQLite, and pages read it through `Shugo.query`. The benchmark checks that a change reevaluates only the readers of the changed records. See the [design and decisions](../../VARIANT.md).

## Run the benchmark

```sh
go build -o bin/shugo ./cmd/shugo                    # from the repository root
node examples/catalog/stand.mjs --size 10000         # or use Bun. Pass --runtime node to evaluate with Node.
```

The benchmark runs the following steps.

1. Generates `data/catalog.db` for `--size` products. The same size always produces the same shop.
2. Builds the site with `shugo build`.
3. Changes the data 20 times using SQL and runs `shugo update` after each change. It checks the report against the expected evaluations and changes.
4. Rebuilds the site from the same data in another directory and compares the resulting files.

To view the site, run `node examples/catalog/lib/generate.mjs 1000`, then `bin/shugo dev examples/catalog`.

## How the example works

| File | What it shows |
| --- | --- |
| `lib/generate.mjs` | Shop tables. Each has a single-column key and a separate ordering column. |
| `components/Header.shugo` | Shared block with no parameters. It reads its own configuration, navigation, and category counts. |
| `components/ProductCard.shugo` | Shared block that takes a product ID and reads the product. A product edit reevaluates the card on its own. |
| `components/Variant.shugo` | Regular component that displays a row read by the page. |
| `pages/products/[slug].shugo` | The route lists URLs, and each page queries its own product, variants, reviews, badges, and recommendations. |
| `pages/categories/[category]/[...page].shugo` | The route uses `paginate` with a product count. The page uses `Shugo.paginate` with a query. |
| `lib/catalog.ts` | Price helpers that load no data. |

## Results

The [results](../../VARIANT.md#results) have tables for every size, initial build times, and analysis. With 100,000 products and 104,202 pages, changing one price evaluates one page and one card, and the full `shugo update` command takes 38 ms. A change that no page reads triggers no evaluations. The initial build takes 13–15 s.
