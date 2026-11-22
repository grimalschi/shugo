# Open questions

Questions are numbered: answer the numbers. Where there is advice, yes - to accept it. Decisions taken without confirmation are collected in [Decisions awaiting approval](decisions.md); proposals that do not yet exist are in [docs/proposals](../docs/proposals/README.md).

### 1. Two search engines

In the merged work on the islands, there are two searches. First, the experiment was done by `search()` of `shugo:catalog`: one index of goods with a store scheme, `examples/islands`. Then there is the general `search("products", { … })` from `shugo:search`: the table and fields call the call, indexes as many as you want, `examples/search`. The old way takes its `internal/catalog` package, individual branches in computing, server and storage, and its flag in the frame.

**The Council:** remove `shugo:catalog` and translate `examples/islands` to `search("products", { … })`: the agent will not have to choose from two entries of one thing.

### 2. `search` from the module `shugo`

`query`, `paginate` and `page` are imported from `shugo`, and `search` is imported from `shugo:search`, although this is the same framework feature.

**The Council:** `import { query, page, search } from "shugo"`, and recording `shugo:search` is a prompt error like with `Shugo.query`.

### 3. Types for editor

The `.svelte` and `.island.svelte` extensions are self-recognized by the editor, while the `shugo` module, `shugo:search`, and the `request:` directive are not: checking the types in the editor is swearing at them.

**The Council:** ad types module `shugo` puts in the project `shugo init`, which is decided to do; before him - file `src/shugo.d.ts` in examples.
