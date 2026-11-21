# Shugo architecture

These documents describe how Shugo works internally: stages, formats, storage, server and how data changes reach pages. How to collect and run projects on Shugo is written in [specifications](../spec/README.md). Proposals that have not yet been made lie in [proposals](proposals/README.md).

Documents are maintained in the same way as the specification: in each of up to three sections - **Decisions** (only what the author of the project confirmed), **Implemented, pending approval** and **Open questions**.

| Document | About what |
| --- | --- |
| [Build pipeline](build-pipeline.md) | Stages: compilation, calculation, runtime; what everyone gets and what they give out |
| [Blocks and incremental updates](blocks.md) | What is the page in the repository and how does the change of data reach only those parts of the site that show it |
| [Change tracking](change-tracking.md) | How to find out what has changed, and only this is calculated; trial implementations and measurements |
| [Storage](storage.md) | Tables, records, value tapes and what is deleted |
| [Runtime](runtime.md) | Server on Go: page from storage for each request, export to files, development mode |
| [Shugo and Svelte](svelte.md) | How Shugo replicates HTML of the Svelte server and how it is checked; test by which template syntax was selected |

All intermediate formats on the small store show the stand "Shugo inside":

```sh
go build -o bin/shugo ./cmd/shugo
go run ./cmd/inside examples/inside      # report: examples/inside/report/index.html
```
