# Catalog and incremental update benchmark

A shop with a configurable number of products, a page for each product, category listings of 24 products per page, a header with navigation and a featured product, and recommendations.

The shop stores its data in SQLite. Pages read it with `query` from `shugo`, and each row records which columns were read. The benchmark checks that a data change reevaluates only the readers of the changed columns in the changed rows. Export writes only pages whose HTML changed. See [how it works](../../README.md#how-it-works) and the [change tracking design](../../docs/change-tracking.md).

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

To view the site, run `node examples/catalog/scripts/generate.mjs 1000`, then `bin/shugo dev examples/catalog`.

## How the example works

Only the list of product URLs specifies columns. Each page depends on the columns it actually reads.

| File | What it shows |
| --- | --- |
| `scripts/generate.mjs` | Shop tables with single-column keys and separate ordering columns. The generator leaves index creation to the build. |
| `src/site.html` | Shell for every page, with `<html lang="en">`, `<head>`, `<body>`, and the `%shugo.head%` and `%shugo.body%` placeholders. `src/pages/+layout.svelte` starts with the header. |
| `src/components/Header.svelte` | Shared block with no parameters. It reads its own configuration, navigation, and category counts. |
| `src/components/ProductCard.svelte` | Shared block that takes a product ID and reads the product. The template calls `finalPrice(item)`, so the card depends on the `price` and `discount` fields that the function reads. |
| `src/components/Variant.svelte` | Regular component that receives the full variant row. Reading `title` and `stock` adds them as page dependencies. The variant's shelf location is never read. |
| `src/pages/products/[slug].svelte` | The route lists URLs, and each page queries its own product, variants, reviews, badges, and recommendations. The page reads the expected arrival date only when the product is out of stock, and the discount expiration date only when a discount applies. |
| `src/pages/categories/[category]/[...page].svelte` | The route passes a product count to `paginate` from `shugo`. The page passes a query. |
| `src/lib/catalog.ts` | Price helpers that load no data. |

## Results

The November 22, 2026 benchmark used Bun on eight cores and an ext4 filesystem. Svelte 5.59.1 compiled and rendered the templates. The run included 100,000 products, 104,202 pages, and 100,009 shared blocks.

- The initial build took 14.6 s, including 0.8 s to create eight indexes automatically. The first export took 3.5 s.
- Changing one price evaluated one page and one card and changed the HTML on seven pages. The update took 31 ms, the full command took 39 ms, and the export took 7 ms.
- Changing an unread column on 5,000 products or one variant's shelf location triggered no evaluations.
- Updating the header evaluated one shared block record and took 0.12–0.17 s. Export rewrote all 104,202 files in 4.4–4.6 s. `shugo serve` displayed the change immediately.
- Deleting a product took 0.43 s to update. Export wrote 88 pages and deleted one file in 23 ms.
- The store was 202 MB.

| Change | Rows | Pages / shared blocks evaluated | Pages with changed HTML | Update (ms) | Export (ms) |
| --- | ---: | ---: | ---: | ---: | ---: |
| No changes | 0 | 0 / 0 | 0 | 28 | 6 |
| Unread column on every 20th product | 5,000 | 0 / 0 | 0 | 114 | 6 |
| Price of one product | 1 | 1 / 1 | 7 | 31 | 7 |
| Prices of 20 products | 20 | 20 / 20 | 70 | 37 | 14 |
| Title of the featured product | 1 | 1 / 2 | 104,202 | 129 | 4,446 |
| Footer phone number | 1 | 0 / 1 | 104,202 | 152 | 4,494 |
| Move a product to another category | 1 | 168 / 1 | 104,202 | 268 | 4,562 |
| Approved review text | 1 | 1 / 0 | 1 | 25 | 6 |
| Stock of one variant | 1 | 1 / 0 | 1 | 26 | 6 |
| Delete the last product | 15 | 89 / 0 | 88 | 427 | 23 |
| Add a product | 5 | 86 / 2 | 104,202 | 493 | 4,528 |

The November 21, 2026 run used Shugo's own template compiler and held values in the records. The first build took 14.5 s and the first export took 5.0 s. Changing a price took 29 ms. Updating the header took 0.12–0.13 s and exporting after it took 8.2–8.5 s. Deleting a product took 0.40 s to update. The store was 140 MB. Storing HTML gives similar update times, a twofold speedup for full-site exports, and a 44% larger store.

For a change visible on all pages, export must rewrite every file.
