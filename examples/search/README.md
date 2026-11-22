# Search two collections

One Svelte island serves both products and articles. The products page uses column names directly. The articles page binds the same form fields to section, reading time, and publication status. Each result set has its own API endpoint. [Products](src/pages/products.svelte) · [Articles](src/pages/articles.svelte) · [Search island](src/components/Search.island.svelte)

Run the example from the repository root. The seed command expects a new database for this example.

```sh
go build -o bin/shugo ./cmd/shugo
bun install --cwd examples/search --frozen-lockfile
bun examples/search/scripts/seed.mjs
bin/shugo build examples/search
bin/shugo serve examples/search -addr 127.0.0.1:4321
```

The server returns HTML filtered by the link. After hydration, the form updates the results and the URL. Back and Forward restore previous filters. With JavaScript disabled, the form submits a regular GET request. [In-stock cups up to $200](http://127.0.0.1:4321/products?q=cups&category=ceramics&price_to=200&available=1) · [Published guides of at least 5 minutes](http://127.0.0.1:4321/articles?category=guides&price_from=5&available=1)

The end-to-end test checks HTML, hydration data, both JSON endpoints, and a journaled update to one table that leaves the other result set unchanged.

```sh
go test ./test -run '^TestSearchIslands$'
```

For a browser check, open the demo in Playwright CLI and run the script on the current page.

```sh
npx --yes --package @playwright/cli playwright-cli open http://127.0.0.1:4321/
npx --yes --package @playwright/cli playwright-cli run-code --filename examples/search/scripts/browser.mjs
npx --yes --package @playwright/cli playwright-cli close
```

Deploying with `shugo serve` requires the Shugo executable, the published `_build/store.db`, `_build/assets`, and the `_build/search-*.sqlite` search snapshots. Use the store and the search indexes from the same update.

## Benchmark with 100,000 products

The benchmark added 99,995 synthetic rows to the example's five products. They used three categories, numeric prices, availability flags, and five common titles. A rare term appeared in about one row in 100. The benchmark ran on localhost with one sequential client and 30 warm requests per scenario.

| Request | p50 | p95 | Response body |
| --- | ---: | ---: | ---: |
| HTML with default results | 0.25 ms | 0.36 ms | 3.4 KB |
| HTML with a rare term and filters | 8.35 ms | 11.83 ms | 3.9 KB |
| JSON with the same term and filters | 6.96 ms | 10.33 ms | 1.7 KB |
| HTML with a common term and filters | 42.98 ms | 49.20 ms | 3.8 KB |

Building the site took 1.76 s. The product index was 11.6 MB and the island's browser bundle was 74 KB. Updating a field outside the projection took about 12 ms. Updating a searchable field rebuilt the product index in 1.2 s and kept the existing pages because the initial results were unchanged. The first filtered request to the running server took 60 ms.
