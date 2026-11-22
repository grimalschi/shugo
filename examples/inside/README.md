# Inside Shugo

This small shop shows what passes between the stages of the framework: the source files, the JavaScript and template program that the compiler generates, the project database and its journal, the messages between Go and JavaScript, the stored records and their relationships, and what each update changes.

```sh
go build -o bin/shugo ./cmd/shugo          # from the repository root
go run ./cmd/inside examples/inside        # writes examples/inside/report/index.html
```

The `cmd/inside` program creates the database from `data/seed.sql`, builds the project, applies the changes in `data/steps.json` in order, and runs `shugo update` after each one. The report captures the actual state of that run. Edit or add steps in `data/steps.json` to try other updates.

The `tap.sh` wrapper captures messages between Go and JavaScript. Shugo runs it in place of Bun, and it copies both streams into `report/talk/`.

| File | Contents |
| --- | --- |
| `src/pages/products/[slug].svelte` | Data-driven page: a query supplies the list of URLs, and each page displays the product and its variants. |
| `src/pages/categories/[id].svelte` | Category page with product cards. |
| `src/pages/index.svelte` | Home page with cards for all products. |
| `src/pages/+layout.svelte` | Layout with a header. Its `<head>` receives the pages' `<svelte:head>` content. |
| `src/components/Header.svelte` | Shared block with no parameters that displays navigation and counts. |
| `src/components/ProductCard.svelte` | Shared block keyed by product. |
| `src/components/Variant.svelte` | Regular component that receives a variant row. |
