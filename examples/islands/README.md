# Catalog with a Svelte island

The demo uses the same deterministic product database as the main catalog example. Run it from the repository root. [Main catalog example](../catalog/README.md)

[Experiment report](REPORT.md)

```sh
go build -o bin/shugo ./cmd/shugo
bun install --cwd examples/islands --frozen-lockfile
bun examples/islands/scripts/seed.mjs 1000
bin/shugo build examples/islands
bin/shugo serve examples/islands -addr 127.0.0.1:4321
```

- [Catalog](http://127.0.0.1:4321/catalog) supports search, price, availability, category and size filters, sorting, and pagination. Apply text and price filters with **Show results**. The other filters take effect as soon as they change after hydration.
- [Deep link](http://127.0.0.1:4321/catalog?q=cup&stock=1&sort=cheap) opens the server-rendered HTML with the results already filtered. Disable JavaScript to try the GET form with a full navigation.
- [Static version](http://127.0.0.1:4321/baseline) uses the initial results from the same index, with the same interface and shared stylesheet. It runs without client JavaScript, and its form opens the catalog. Both templates use [the same product card](src/components/ProductCard.shugo).
- [Two islands](http://127.0.0.1:4321/multiple) shows two copies of the same counter with independent state. The [home page](http://127.0.0.1:4321/) has no islands.

The catalog client hydrates the rendered markup, fetches JSON from `/_islands/catalog` whenever the query changes, and responds to Back and Forward navigation. Svelte's marker comments remain in the HTML for hydration. File exports contain the initial results and their client code. A running Shugo server handles results for query parameters.

## Validation and benchmarks

Browser checks require a running demo with 1,000 products and Chrome or Chromium for Playwright CLI. The first script checks HTML, hydration, the address bar, canceled requests, error recovery, and form submission with JavaScript disabled. The second measures page load and result updates with network and CPU throttling. Playwright writes its log to a temporary directory.

```sh
shugo_checkout="$PWD"
cd "$(mktemp -d)"
npx --yes --package @playwright/cli playwright-cli -s=shugo-islands open http://127.0.0.1:4321/
npx --yes --package @playwright/cli playwright-cli -s=shugo-islands run-code --filename "$shugo_checkout/examples/islands/scripts/browser.mjs"
npx --yes --package @playwright/cli playwright-cli -s=shugo-islands run-code --filename "$shugo_checkout/examples/islands/scripts/browser-timing.mjs"
npx --yes --package @playwright/cli playwright-cli -s=shugo-islands close
```

Run the build timing, HTTP timing, and asset size comparisons from the repository root.

```sh
node examples/islands/scripts/bench.mjs --sizes 1000,10000,100000 --runs 3
node examples/islands/scripts/bench.mjs --sizes 100000 --runs 3 --requests 120 --clients 8
```
