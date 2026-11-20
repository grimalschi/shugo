# Catalog and incremental update benchmark

A shop with a configurable number of products, a page for each product, category listings of 24 products per page, a header with navigation and a featured product, and recommendations.

Variant 5 stores the shop data in SQLite. Pages read it through `Shugo.query`, and each row records which of its columns were read. The benchmark checks that changes reevaluate only the readers of the changed columns in the changed rows. See the [design and decisions](../../VARIANT.md).

## Run the benchmark

```sh
go build -o bin/shugo ./cmd/shugo                    # from the repository root
node examples/catalog/stand.mjs --size 10000         # or use Bun. Pass --runtime node to evaluate with Node.
```

The benchmark runs the following steps.

1. Generates `data/catalog.db` for `--size` products. The same size always produces the same shop. The database starts with indexes for keys and product URLs only.
2. Builds the site with `shugo build`. On the first build, Shugo creates the indexes that page queries need, and the benchmark reports them.
3. Changes the data 21 times using SQL and runs `shugo update` after each change. It checks the report against the expected evaluations and changes.
4. Rebuilds the site from the same data in another directory and compares the resulting files.

To view the site, run `node examples/catalog/lib/generate.mjs 1000`, then `bin/shugo dev examples/catalog`.

## How the example works

Only the list of product URLs specifies columns. Each page depends on the columns it actually reads.

| File | What it shows |
| --- | --- |
| `lib/generate.mjs` | Shop tables with single-column keys and separate ordering columns. The generator leaves index creation to the build. |
| `components/Header.shugo` | Shared block with no parameters. It reads its own configuration, navigation, and category counts. |
| `components/ProductCard.shugo` | Shared block that takes a product ID and reads the product. The template calls `finalPrice(item)`, so the card depends on the `price` and `discount` fields that the function reads. |
| `components/Variant.shugo` | Regular component that receives the full variant row. Reading `title` and `stock` adds them as page dependencies. The variant's shelf location is never read. |
| `pages/products/[slug].shugo` | The route lists URLs, and each page queries its own product, variants, reviews, badges, and recommendations. The page reads the expected arrival date only when the product is out of stock, and the discount expiration date only when a discount applies. |
| `pages/categories/[category]/[...page].shugo` | The route uses `paginate` with a product count. The page uses `Shugo.paginate` with a query. |
| `lib/catalog.ts` | Price helpers that load no data. |

## Results

The [benchmark](../../VARIANT.md#benchmark) has detailed tables, initial build times, and analysis. The following figures use Bun with 100,000 products and 104,202 pages.

- Changing one price evaluates one page and one card. The full `shugo update` command takes 38 ms.
- Changing an unread column on 5,000 products or one variant's shelf location triggers no evaluations.
- An initial build takes 14.4–14.7 s. Building with eight new indexes takes 15.1–15.4 s.
