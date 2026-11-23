# Shugo specification

Shugo is a framework for static sites. This specification covers building and running projects. See the [architecture docs](../docs/README.md) and [proposals](../docs/proposals/README.md) for internals and potential additions.

## Status of decisions

- **Decisions** records choices the project author has confirmed.
- **Implemented, pending approval** describes current behavior. See the [decision log](decisions.md) for the reasoning.
- **Open questions** tracks remaining decisions. See the [numbered list](open-questions.md).

## Documents

| Document | Contents |
| --- | --- |
| [Project structure](project-structure.md) | Project files and directories |
| [Templates](templates.md) | Svelte 5 components, code, markup, blocks, and expressions |
| [Components](components.md) | Reusable template code, props, snippets, and shared components |
| [Layouts](layouts.md) | Common page chrome, the `src/site.html` shell, and `<svelte:head>` |
| [File-based routing](routing.md) | URLs from page filenames and non-HTML endpoints such as `sitemap.xml.ts` |
| [Dynamic pages](dynamic-pages.md) | Pages whose routes come from data |
| [Data loading](data-loading.md) | Data sources and query behavior |
| [Styles](styles.md) | Component styles, shared CSS files, and Tailwind |
| [Page scripts](scripts.md) | Browser code and Svelte islands |
| [CLI and development mode](cli.md) | The `shugo` commands and their reports |
| [Authoring guide](authoring.md) | Quick reference for an agent creating a Shugo site |
| [Decision log](decisions.md) | Proposed and implemented decisions, with reasoning |
| [Open questions](open-questions.md) | Numbered questions and answers from the project author |

## General decisions

- Shugo generates static sites. Routes and data are known in advance. The server assembles the response without executing project code.
- Shugo is designed for coding agents. When two approaches are otherwise viable, prefer the one on which agents make fewer mistakes. Template expressions use JavaScript for this reason.
- Pages, layouts, and components are written as Svelte 5 components. Browser entry points use the `.island.svelte` suffix. See [Svelte islands](scripts.md#svelte-islands).
- Go handles project compilation, launching evaluation, and assembling pages. Since November 22, 2026, Shugo has used Svelte itself to compile and render templates under Bun or Node. Since November 23, Svelte has been an npm dependency of `shugo`.
- Project source uses TypeScript.
- Each stage produces an explicit output for the next stage. See the [build pipeline](../docs/build-pipeline.md).
- Go stages can run outside a container. The JavaScript stage can run with or without a sandbox and produces the same result. The November 21, 2026 decision leaves sandbox setup to the caller.
- The JavaScript runtime supports both Bun and Node.
- Favor clear, maintainable implementations and keep the framework fast.
