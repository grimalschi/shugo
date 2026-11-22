# Shugo architecture

These documents describe the build stages, file formats, store, and server. They also explain how changes in site data reach the pages that depend on them. See the [specification](../spec/README.md) for building and running Shugo sites and [proposals](proposals/README.md) for potential additions.

Each document can cover accepted decisions, implemented behavior awaiting approval, and open questions.

| Document | Contents |
| --- | --- |
| [Build pipeline](build-pipeline.md) | Compilation, evaluation, and serving, with the inputs and outputs of each stage |
| [Blocks and incremental updates](blocks.md) | How the store represents pages and how data changes reach their readers |
| [Change tracking](change-tracking.md) | How Shugo identifies changes and reevaluates the affected content, with experiments and benchmarks |
| [Storage](storage.md) | Tables, page records, and deletion |
| [Runtime](runtime.md) | Serving pages from the store, exporting files, and development mode |
| [Shugo and Svelte](svelte.md) | Where their template behavior differs and how the Svelte fixture suite checks it |
| [Svelte islands](islands.md) | Island delivery, request-time SSR in Go, table search, and the approaches tested |

The [internals example](../examples/inside/README.md) shows the intermediate formats for a small shop.

```sh
go build -o bin/shugo ./cmd/shugo
go run ./cmd/inside examples/inside      # writes examples/inside/report/index.html
```
