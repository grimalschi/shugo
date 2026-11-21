# Catalog and incremental update benchmark

A shop with a configurable number of products, a page for each product, category listings of 24 products per page, a header with navigation and a featured product, and recommendations.

The shop stores its data in SQLite. Pages use `Shugo.query`, and each row records which columns were read. The benchmark checks that a data change reevaluates only the readers of the changed columns in the changed rows. Export writes only pages whose HTML changed. See [how it works](../../README.md#how-it-works) and the [change tracking specification](../../spec/change-tracking.md).

## Run the benchmark

```sh
go build -o bin/shugo ./cmd/shugo                    # from the repository root
node examples/catalog/stand.mjs --size 10000         # or use Bun. Pass --runtime node to evaluate with Node.
```

The benchmark runs the following steps.

1. Generates `data/catalog.db` for `--size` products. The same size always produces the same shop. The database starts with indexes for keys and product URLs only.
2. Builds the site with `shugo build`. On the first build, Shugo creates the indexes that page queries need, and the benchmark reports them.
3. Exports the site to files with `shugo export`.
4. Changes the data 21 times using SQL and runs `shugo update` and `shugo export` after each change. It checks the update report against the expected evaluations and changes. It also checks that export writes exactly the pages whose HTML the report says changed.
5. Rebuilds and exports the site from the same data in another directory and compares the two directories file by file.

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

The benchmark used 100,000 products, 104,202 pages, and 100,009 shared blocks. It ran with Bun on eight cores and an ext4 filesystem.

- The initial build took 13.7–14.1 s, including 0.8 s to create eight indexes automatically. The first export took 7.1–8.5 s.
- Changing one price evaluated one page and one card and changed the HTML on seven pages. The update took 28–32 ms, the full command took 36–41 ms, and the export took 7–8 ms.
- Changing an unread column on 5,000 products or one variant's shelf location triggered no evaluations.
- Updating the header evaluated one shared block record and took 0.11–0.12 s. Export rewrote all 104,202 files in 8.1–8.7 s. `shugo serve` displayed the change immediately.
- Deleting a product took 0.34–0.41 s to update. Export wrote 88 pages and deleted one file in 25–32 ms.

| Change | Rows | Pages / shared blocks evaluated | Pages with changed HTML | Update (ms) | Export (ms) |
| --- | ---: | ---: | ---: | ---: | ---: |
| No changes | 0 | 0 / 0 | 0 | 380–495 | 6–7 |
| Unread column on every 20th product | 5,000 | 0 / 0 | 0 | 104–113 | 7 |
| Price of one product | 1 | 1 / 1 | 7 | 28–32 | 7–8 |
| Prices of 20 products | 20 | 20 / 20 | 70 | 36–37 | 17 |
| Title of the featured product | 1 | 1 / 2 | 104,202 | 116 | 8,121 |
| Footer phone number | 1 | 0 / 1 | 104,202 | 118 | 8,225 |
| Move a product to another category | 1 | 168 / 1 | 104,202 | 249 | 8,509 |
| Approved review text | 1 | 1 / 0 | 1 | 22 | 7 |
| Stock of one variant | 1 | 1 / 0 | 1 | 25 | 7 |
| Delete the last product | 15 | 89 / 0 | 88 | 340–414 | 25–32 |
| Add a product | 5 | 86 / 2 | 104,202 | 481–920 | 8,314 |

The first update after the build is the no-op row. It takes longer than most updates because SQLite reads the journal for the first time since the build. For a change visible on all pages, export must rewrite every file.
