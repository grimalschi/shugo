# Shugo specification

Shugo is framework for static website generation. These documents describe in simple terms how to assemble and run projects on Shugo. How Shugo is arranged inside, described separately in [docs](../docs/README.md); proposals that do not yet exist are in [docs/proposals](../docs/proposals/README.md).

## Status of decisions

Documents are written together. Each of the three sections:

- **Decisions** is just what the author of the project confirmed.
- **Implemented, pending approval** — how it works now. The reasons for each such decision are collected in one place: [Decisions pending approval](decisions.md).
- **Open questions** is something that still needs to be solved. Common list with numbers: [Open questions](open-questions.md).

## Documents

| Document | About what |
| --- | --- |
| [Project structure](project-structure.md) | What folders and files are in the project and what they mean |
| [Templates](templates.md) | `.svelte` files are components of Svelte 5: code, markup, blocks that output expressions where Shugo deviates from Svelte |
| [Components](components.md) | Reusable parts of templates: parameters, snippets, common components |
| [Layouts](layouts.md) | Overall binding of pages, `src/site.html` and `<svelte:head>` shell |
| [File-based routing](routing.md) | How page files are converted to addresses; files are not in HTML: `sitemap.xml.ts` |
| [Dynamic pages](dynamic-pages.md) | Pages that come from data |
| [Data loading](data-loading.md) | Where the data comes from and what you should know about it |
| [Styles](styles.md) | Component styles, common files CSS and Tailwind |
| [Page scripts](scripts.md) | Code that runs in the browser and islands Svelte |
| [Command line and development mode](cli.md) | `shugo` commands and their reports |
| [A note for the agent](authoring.md) | A short guide for someone writing a website on Shugo (in English) |
| [Decisions pending approval](decisions.md) | What is decided without the author’s confirmation and why |
| [Open questions](open-questions.md) | Questions with numbers and answers to the author's questions |

## General decisions

- Shugo generates static sites. All pages and all data are known in advance; the project code is not executed at the request of the visitor.
- **This is framework for agents.** Project code is written by agents, not people. Two solutions are chosen to make the agent make fewer mistakes. Therefore, expressions are written in JS, not something faster: agents know JS perfectly.
- **Templates use syntax Svelte 5.** Ordinary pages, layouts and components are written in `.svelte`. The entry points for the browser-based Svelte are marked with the suffix `.island.svelte`. [Svelte islands](scripts.md#svelte-islands)
- The entire framework is written in Go: project compilation, start calculation, page assembly. *Since November 22 year 2026 templates compile and render itself Svelte (the author's decision): its compiler and server part are sewn into Shugo and run on Bun or Node running Go.*
- In the source code of the project everywhere TypeScript. This is a rule and requirement.
- The project is divided into stages. The result of each stage is the following, and the boundary between the stages is clear. Details - in [Assembly Sequences](../docs/build-pipeline.md).
- Everything that runs Go must be safe and can work outside of containers. The stage with JS can be run in the sandbox or without it, the result does not change. The sandbox is not part of the Shugo: it is enough to describe how to run it (21 solution of November 2026 year).
- JS is both Bun and Node: both must work.
- The code of the framework should be clean and beautiful. It is better to spend more time finding the right solution than doing a workaround. At the same time, framework should be fast.
