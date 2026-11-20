# Shugo specification

Shugo is a framework for static website generation. These documents describe in simple terms how it works and how projects are collected on it.

## Status of decisions

Documents are written together. In each of the three sections:

- **Decisions** is just what the author of the project confirmed.
- **From the prototype, pending approval** tested on prototype, but not yet approved as a framework solution.
- **Open questions** is something that needs to be solved.

## Documents

| Document | About what |
| --- | --- |
| [Project structure](project-structure.md) | What folders and files are in the project and what they mean |
| [Build pipeline](build-pipeline.md) | Four steps: compilation, data loading, expression execution, page assembly |
| [File-based routing](routing.md) | How Page Files Turn into Addresses |
| [Page Templates](templates.md) | Format `.shugo`: data code, markup, expressions, control tags |
| [Dynamic pages](dynamic-pages.md) | Pages that come from data |
| [Components](components.md) | Reusable parts of templates and their parameters |
| [Layouts](layouts.md) | Overall page binding |
| [Data loading](data-loading.md) | Where data comes from and how it compares to past data |
| [Pointed page update](incremental-updates.md) | How data changes only to the parts of pages that depend on it |
| [Runtime](runtime.md) | Server on Go: pages on the fly or the entire site in files |
| [Internationalization](i18n.md) | Languages and translations |
| [Styles](styles.md) | Tailwind and CSS |
| [Pictures](images.md) | Compression on the fly with cache |

## General decisions

- Shugo generates static sites.
- The entire framework is written in Go: project compilation, data upload, page assembly.
- Expressions in templates are written in JS. Conditions, cycles and other control structures are their tags, not JSThey are tied to the work of the framework, and he needs to understand the structure of the page.
- In the project source code by default everywhere TypeScript.
- The project is divided into stages. The result of each stage is the following, and the boundary between the stages is clear. Details - in [Assembly Sequences](build-pipeline.md).
- The code of the framework should be clean and beautiful. It is better to spend more time finding the right solution than doing a workaround. At the same time, framework should be fast.
