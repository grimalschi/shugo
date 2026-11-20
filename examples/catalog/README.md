# Catalog and incremental update benchmark

A shop with a configurable number of products, a page for each product, category listings of 24 products per page, a header with navigation and a featured product, and recommendations.

The benchmark checks that a data change updates only the parts of the site that display that data.

## Run the benchmark

```sh
go build -o bin/shugo ./cmd/shugo                    # from the repository root
node examples/catalog/stand.mjs --size 10000         # or use Bun. Pass --runtime node to evaluate with Node.
```

The benchmark runs the following steps.

1. Generates shop data for `--size` products. The same size always produces the same shop.
2. Builds the site with `shugo build`.
3. Changes the data 20 times and runs `shugo update` after each change. It checks the update report against the expected changes.
4. Rebuilds the site from the same data in another directory and compares the resulting files.

To view the site, run `node examples/catalog/lib/generate.mjs 1000`, then `bin/shugo dev examples/catalog`.

## How the example works

| File | What it shows |
| --- | --- |
| `components/Header.shugo` | Shared block with no parameters. The site computes it once and displays it on every page. |
| `components/ProductCard.shugo` | Shared block with a product ID parameter. The site computes one card per product. |
| `components/Variant.shugo` | Regular component that takes an object and produces a block owned by the page. |
| `pages/products/[slug].shugo` | One file produces 100,000 pages. All page data arrives in a single list. |
| `pages/categories/[category]/[...page].shugo` | Pagination with `paginate`. |
| `pages/+layout.shugo` | Layout with a `head` slot for the page title. |

## Results

Xeon E3-1270 v6 (4 cores, 8 threads), Go 1.27.4, Bun 1.4.4, Node 24.22.1. November 20, 2026.

### Size

| Products | Pages | Shared blocks | Total blocks | Distinct tapes | Store |
| ---: | ---: | ---: | ---: | ---: | ---: |
| 1,000 | 1,052 | 1,009 | 12,214 | 3,150 | 0.9 MB |
| 10,000 | 10,452 | 10,009 | 121,514 | 30,550 | 8.4 MB |
| 100,000 | 104,202 | 100,009 | 1,213 196 | 304,300 | 85 MB |

### Timing

| Products | Initial build | Update with Bun | Update with Node |
| ---: | ---: | ---: | ---: |
| 1,000 | 0.11–0.15 s | 0.11–0.13 s | — |
| 10,000 | 0.64–0.69 s | 0.40–0.45 s | 0.51 s |
| 100,000 | 8.6 s | 3.3–3.7 s | 4.3–4.7 s |

Update time is similar across edits because the build evaluates the whole site and writes only the differences. A typical run for 100,000 products takes 3.5 s.

- JavaScript takes about 2.7 s, including 0.4 s to read the 65 MB data file.
- Go takes 0.5 s to read existing hashes from the store.
- Writing the changes takes 0.05 s.

The first build takes longer because it writes all 204,211 records and 304,300 tapes.

### Changes with 100,000 products

| Data change | Pages | Shared blocks | Tapes | Time |
| --- | ---: | ---: | ---: | ---: |
| No changes | 0 | 0 | +0 −0 | 3.6 s |
| Undisplayed field on 5,000 products | 0 | 0 | +0 −0 | 3.5 s |
| Price of one product | 1 | 1 | +2 −2 | 3.4 s |
| Prices of 20 products | 20 | 20 | +40 −40 | 3.5 s |
| Title of the featured product | 1 | 2 | +4 −4 | 3.7 s |
| Change the featured product | 0 | 1 | +1 −1 | 3.5 s |
| Clear the featured product | 0 | 1 | +1 −1 | 3.3 s |
| Move a product to another category | 168 | 1 | +169 −169 | 3.7 s |
| Footer phone number | 0 | 1 | +1 −1 | 3.7 s |
| Discount one product | 1 | 1 | +2 −2 | 3.4 s |
| Approved review text | 1 | 0 | +1 −1 | 3.6 s |
| Hidden review text | 0 | 0 | +0 −0 | 3.6 s |
| Stock of one variant | 1 | 0 | +1 −0 | 3.5 s |
| Insert a variant first | 1 | 0 | +2 −1 | 3.4 s |
| Remove the first variant | 1 | 0 | +1 −2 | 3.7 s |
| Badges on one product | 1 | 1 added | +2 −1 | 3.5 s |
| Swap two recommendations | 1 | 0 | +0 −0 | 3.3 s |
| Remove a recommendation | 1 | 0 | +1 −1 | 3.6 s |
| Delete the last product | 89 (1 deleted) | 1 deleted | +88 −91 | 3.6 s |
| Add a product | 85 (1 added) | 2 (1 added) | +88 −85 | 3.5 s |

The examples below describe how the counts arise.

- **Product price.** The price appears on the product page, its category listing, and four other pages that recommend it. The update rewrites the product page and its card. The other pages reference the card by name, so their records stay the same.
- **Header.** The header appears on all 104,202 pages. Changes to the featured product, category counts, or footer phone number each rewrite one record.
- **Category move.** Moving a product shifts the listings in two categories, affecting 168 pages.
- **Recommendation order.** Reordering recommendations changes the order of blocks in the page record and adds no tapes.
- **Hidden review and internal field.** These values never appear in the output, so neither edit changes it.

After 20 changes, the incrementally updated site matched the clean build on all 104,202 pages.

### Runtime

A product page contains 2.7 KB of HTML from 12 blocks.

| Measurement | Result |
| --- | ---: |
| Assemble a page with its tapes in memory | 7 µs |
| Assemble a page on first request, reading its tapes from SQLite | 47 µs |
| Full server request with 8 clients on the same machine | 25–30 µs per request, 33,000–39,000 requests per second |
| Export all 104,202 pages to files | 8.5 s, 427 MB |

Run `go test ./test/ -run xxx -bench .` to reproduce the benchmark. Set the shop size with `SHUGO_BENCH_SIZE`.

## Findings

- The build evaluates the whole site and writes only changes. A subsecond update for 100,000 pages would require distributing evaluation across processes or tracking what changed. See the [proposal](../../spec/partial-reload.md).
- Project code and data reads account for most of the time. The first version of this example created an `Intl.NumberFormat` for every price, making evaluation three times slower.
