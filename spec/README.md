# Shugo specification

Shugo is a framework for static website generation. These documents describe in simple terms how it works and how projects are collected on it.

## Status of decisions

Documents are written together. Each of the three sections:

- **Decisions** is just what the author of the project confirmed.
- **Implemented, pending approval** — how it works now. The reasons for each such decision are collected in one place: [Decisions pending approval](decisions.md).
- **Open questions** is something that needs to be decided. General list with order of discussion: [Open questions](open-questions.md).

## Documents

| Document | About what |
| --- | --- |
| [Project structure](project-structure.md) | What folders and files are in the project and what they mean |
| [Build pipeline](build-pipeline.md) | Stages: compilation, calculation, runtime |
| [File-based routing](routing.md) | How Page Files Turn into Addresses |
| [Page Templates](templates.md) | Format `.shugo`: data code, markup, expressions, control tags |
| [Dynamic pages](dynamic-pages.md) | Pages that come from data |
| [Components](components.md) | Reusable template parts, parameters, slots |
| [Layouts](layouts.md) | Overall page binding |
| [Data loading](data-loading.md) | Where data comes from |
| [Data adapter and data copy](data-adapter.md) | How data from any source gets into the copy that builds dependencies |
| [Blocks and incremental updates](blocks.md) | How data changes only reach the parts of the site that show it |
| [Change tracking](change-tracking.md) | How to find out what has changed, and only this is calculated: the sentence and measurements |
| [Storage](storage.md) | What lies between the stages and in what form |
| [Runtime](runtime.md) | Server on Go: pages on the fly or the entire site in files |
| [Command line and development mode](cli.md) | `shugo` commands and their reports |
| [Styles](styles.md) | Tailwind and CSS |
| [Page scripts](scripts.md) | Code that runs in the browser |
| [Internationalization](i18n.md) | Languages and translations: offer |
| [Pictures](images.md) | Compression on the fly with cache: offer and measurements |
| [Partial data reboot](partial-reload.md) | Update on signals from sources: proposal |

## General decisions

- Shugo generates static sites. All pages and all data are known in advance; the project code is not executed at the request of the visitor.
- **This is framework for agents.** Project code is written by agents, not people. Two solutions are chosen to make the agent make fewer mistakes. Therefore, expressions are written in JS, not something faster: agents know JS perfectly.
- The entire framework is written in Go: project compilation, start-up calculation, page assembly.
- Expressions in templates are written in JS. Conditions, cycles and other control structures are their tags, not JSThey are tied to the work of the framework, and he needs to understand the structure of the page.
- In the source code of the project everywhere TypeScript. This is a rule and requirement.
- The project is divided into stages. The result of each stage is the following, and the boundary between the stages is clear. Details - in [Assembly Sequences](build-pipeline.md).
- Everything that runs Go must be safe and can work outside the containers. The stage with JS can be run in the sandbox or without it, the result does not change.
- JS is both Bun and Node: both must work.
- The code of the framework should be clean and beautiful. It is better to spend more time finding the right solution than doing a workaround. At the same time, framework should be fast.
