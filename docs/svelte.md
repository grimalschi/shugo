# Shugo and Svelte

Shugo pages, layouts, and components are Svelte 5 components in `.svelte` files. As in SvelteKit, Svelte 5.59.1 compiles them for the server. During the build, its server runtime renders them on Bun or Node. The `shugo` package pins `svelte`, which provides the compiler and both runtimes. Shugo supplies the page shell, head assembly, reusable blocks, storage, and HTTP server. Svelte compiles `.island.svelte` components for both the server and browser. When an island reads request data, Shugo prepares a rendering program from Svelte's server output. [Templates](../spec/templates.md) · [Svelte islands](islands.md) · [Blocks](blocks.md)

## Responsibilities

| Feature | Implementation | Where |
| --- | --- | --- |
| Parsing, errors, and warnings, including accessibility warnings (`a11y_…`) | Svelte's `compile` from `svelte/compiler` with `generate: "server"` and `runes: true` | `internal/svelte/compile.mjs` |
| Whitespace, escaping, attributes, `class` and `style`, `{#each}`, `{#if}`, snippets, `<svelte:element>`, and `<select value>` | Svelte's compiled output and server runtime | `svelte` package |
| Component CSS, including the `svelte-…` class, rewritten selectors, and unused selector detection | Svelte with `css: "external"` | Shugo collects the emitted CSS into a file in `internal/compile/styles.go` |
| TypeScript types | Svelte erases types in `<script lang="ts">` and rejects constructs that emit JavaScript, such as `enum`. esbuild handles `.ts` modules. | `internal/svelte/compile.mjs`, `internal/compile/build.go` |
| Checks for the document shell, islands, legacy Shugo tags, Svelte 4 slots, and `runes={false}`, plus a diagnostic for markup inside `{…}` | Shugo uses the AST and diagnostics from the same Svelte compile call. | `internal/svelte/compile.mjs` |
| Layout imports and a diagnostic for type imports missing `type` | Shugo's bundler | `internal/compile/build.go` |
| npm packages, including resolution to the pinned `svelte` dependency and diagnostics for missing packages, incompatible Svelte versions, component imports in page data code, and unsupported file types | Shugo's bundler, at the import site | `internal/compile/packages.go` |
| Infinite component recursion | Shugo reports the import and render chain during evaluation. | `internal/jsrt/runtime.js` |
| Whether a component is rendered inline or stored as a shared block | Shugo checks the AST and imports at compile time. The `component()` wrapper checks props at each call. | `internal/svelte/compile.mjs`, `internal/compile/build.go`, `internal/jsrt/runtime.js` |
| Shared block markers, separate rendering of shared block snippets, splitting HTML into blocks, and separate storage of `<title>` | Shugo | `internal/jsrt/runtime.js` |
| Runes in `.svelte.js` and `.svelte.ts` modules | Svelte's `compileModule`. esbuild erases the `.svelte.ts` types, and Shugo composes source maps. | `internal/compile/build.go` |
| Hydration comments such as `<!--[-->`, `<!--]-->`, and `<!---->` | Svelte writes the comments. Shugo preserves them in islands and `{@html}` and removes them from static markup. | `internal/jsrt/runtime.js` |
| The page head in block render order, with the last `<title>` | Shugo's server assembles the head. | `internal/render` |
| The document shell in `src/site.html` | Shugo | `internal/skeleton`, `internal/render` |

Bun or Node executes page code. Svelte writes the HTML and determines how to render numbers, dates, arrays, objects, whitespace, and attributes.

Shugo renders a shared component once for each distinct set of eligible props. Each page that uses it stores a reference at the call site. Argument-free snippets passed to these components render separately and are stored with the call site. [Shared components](blocks.md#shared-components)

Svelte's `render()` returns the head and body separately. Shugo inserts them into `%shugo.head%` and `%shugo.body%` in `src/site.html`, which plays the same role as SvelteKit's `src/app.html`. A root `+layout.svelte` starts with the content of `<body>`. Templates that emit `<html>`, `<head>`, or `<body>` produce an error. The `page` object from `shugo` follows SvelteKit's `$app/state` with `url`, `params`, `route.id`, and `status`.

## Differences from Svelte

- **Hydration comments.** Shugo removes Svelte's hydration comments from static pages, where the comments account for about a quarter of the bytes. Islands keep their markers for hydration, and comments in `{@html}` are preserved. [How HTML is split into blocks](blocks.md#how-html-is-split-into-blocks)
- **Layout rendering.** Shugo renders layouts separately from their pages and nested layouts. The context a layout sets with `setContext` is available to its own components. IDs from `$props.id()` in a layout start with its file number, for example `b3-s1`.
- **Runes only.** Templates compile with `runes: true`, so Svelte rejects `export let`, `$:`, and other Svelte 4 syntax. With runes enabled, Svelte accepts `<slot>`, `slot="…"`, `let:`, and `<svelte:fragment>` with a warning. Shugo rejects these and suggests snippets, which let a shared block render content supplied by its call site.
- **Browser-only warnings.** Shugo suppresses warnings such as `state_referenced_locally` for static pages because they render once. The island compilation emits these warnings because islands can update in the browser.

## Validation

Svelte renders the HTML. The tests compare the final pages with Svelte's output after block splitting, comment removal, and head assembly.

- **Reference tasks.** `test/svelte/` contains components for 30 benchmark tasks and the HTML that Svelte 5.59.1 returns for each dataset. `TestSvelteTasks` builds each task as a site and checks its pages against that HTML. Regenerate the reference output with `cd test/svelte && bun install && bun render.mjs`. Before November 22, 2026, this test compared Shugo's own compiler with Svelte.
- **Comments.** `TestNoCommentsOfSvelte` builds a page with all Svelte block types, shared components in the head, and `$props.id()`. It checks that Svelte's comments are stripped and comments from `{@html}` remain.
- **Templates.** The expected HTML in `test/template_test.go` comes from the same templates rendered by Svelte.
- **Behavior across block boundaries.** `TestSvelteSemanticsAcrossBlocks` covers `bind:`, conditional snippets, a `<select>` containing a snippet, `$props.id()` in a snippet rendered twice, `<title>` from a snippet, snippet names with accented letters, extra snippet arguments, and `slot` on a custom element. The expected HTML comes from rendering the same files in Svelte.
- **Random pages.** `TestRandomPages` generates pages and computes their HTML with an independent Go model. That model matched Svelte 5.59.1 on 400 random pages.
- **Comparison.** The tests strip hydration markers from Svelte's reference output and remove the generated style class from both outputs. The reference file paths differ, so their Svelte CSS hashes differ too.

When upgrading Svelte, update `Version` in `internal/svelte/svelte.go`, the dependency in `npm/shugo/package.json`, the repository and `test/svelte` manifests, and the reference outputs together. Then check the island SSR programs against Svelte's HTML again. [Island SSR program](islands.md#island-ssr-program)

## Compilation benchmarks

A cold build of the seven templates in `examples/basic` took 0.21 s, including 0.18 s for compilation. On November 22, 2026, when Svelte parsed the templates and Shugo compiled them, those times were 0.15 s and 0.13 s. The Svelte compiler takes 64 ms to start. The first template takes 40 ms while Bun warms up the compiler. The next templates take 1–12 ms each. Large projects compile in parallel using one process per 32 templates, capped at four processes.

With 100,000 products, the initial build took 14.6 s and a price change took 31 ms to update. On November 21, 2026, Shugo's own template compiler took 14.5 s and 29 ms for those operations. Most of the difference comes from loading the bundle in Bun. The bundle now includes the Svelte server runtime and grew from 73 KB to 123 KB. Full-site export dropped from 5.0 s to 3.5 s because Go assembles rendered HTML fragments directly. The store grew by 44%, from 140 MB to 202 MB, after the switch to storing HTML in the records. [Catalog benchmark results](../examples/catalog/README.md#results)

The search example measures updates to columns both inside and outside the search index. [Search indexes and update costs](islands.md#search)

## Experiment: template authoring without documentation

The project author requested this experiment on November 21, 2026. Its results informed the compiler discussion.

Three models accessed through OpenRouter, DeepSeek V4.1 Flash, Muse Ember, and GLM 5.3 Flash, each attempted 30 templates in Astro, React, and Svelte three times. They received the task, prop types, and one example of the expected HTML. They had no documentation or search tools. The tasks covered conditionals, loops, data-driven classes and styles, attributes, raw HTML, slots, recursion, sorting, and grouping. The harness rendered each response with Astro 7.4, React 19.3, or Svelte 5.59. It tested the example dataset and two or three datasets the model had never seen. The reference implementations in all three syntaxes produced the same pages for every dataset.

| Model | Astro | React | Svelte |
| --- | ---: | ---: | ---: |
| DeepSeek V4.1 Flash | 86/90 | 90/90 | 82/90 |
| Muse Ember | 86/90 | 90/90 | 89/90 |
| GLM 5.3 Flash | 85/90 | 89/90 | 85/90 |
| Total | 257/270 | 269/270 | 256/270 |

Most mistakes came from a few framework behaviors the models did not anticipate.

- **Astro, 9 of 13 wrong answers.** `Astro.slots.has()` returns `true` for a slot behind a false conditional such as `{note && <p slot="footer">…</p>}`, so it suppresses the fallback. Every model made this mistake on each attempt. A regular `astro build` behaves the same way.
- **Astro, the other four wrong answers.** The models attempted markup or `return` in frontmatter between `---` delimiters, a component that called itself by name instead of using `Astro.self`, or `{…}` inside a quoted attribute value.
- **Svelte, 7 of 14 wrong answers.** Svelte trims whitespace at the edge of `{#if}`, so the prices appeared as `$2,800$2,400`. Two other answers failed to import a component, and one placed `</script>` in a comment inside `<script>`.
- **React, one wrong answer.** The same model made the same logic error in React and Svelte in that attempt.

Shugo still used its own compiler during this experiment and could implement the behavior the models expected. An Astro-style syntax could treat a slot behind a false conditional as absent, allow components to call themselves by name, and let `return` in frontmatter render nothing. A Svelte-style syntax could preserve whitespace at block edges. Those changes would have avoided 12 of 13 Astro failures and 7 of 14 Svelte failures in this run. *Since November 22, 2026, Svelte compiles the templates and Shugo follows its behavior.*

### Adding a paragraph on syntax pitfalls

A second run on the same day used the same tasks, models, and number of attempts. A single paragraph before the prompt's closing instructions explained syntax that caused errors in the first run. For Astro it covered markup in templates, the lack of `return` in frontmatter, `<Astro.self />`, literal braces in quoted attributes, conditional slots, and whitespace beside line breaks. For Svelte it covered whitespace at block edges, importing every component, and `</script>` ending the script wherever it appears. For React it covered JSX whitespace, `{count && …}` when `count` is zero, and DOM attribute names.

| Model | Astro | React | Svelte |
| --- | ---: | ---: | ---: |
| DeepSeek V4.1 Flash | 86 → 89 | 90 → 90 | 82 → 88 |
| Muse Ember | 86 → 89 | 90 → 90 | 89 → 89 |
| GLM 5.3 Flash | 85 → 87 | 89 → 87 | 85 → 87 |
| Total | 257 → 265 | 269 → 267 | 256 → 264 |

The number of wrong answers fell from 28 to 14. Every wrong answer from both runs was reviewed.

| Cause | Without the paragraph | With the paragraph |
| --- | ---: | ---: |
| Pitfalls covered by the paragraph | 23 | 0 |
| Other syntax pitfalls | 0 | 6 |
| Model errors, including logic and JavaScript mistakes or missing or truncated code | 5 | 8 |

- The paragraph eliminated every pitfall it named. Since it was based on errors from the same tasks, these figures reflect a best-case use of targeted guidance.
- Half of the new syntax mistakes came from attempts to follow the guidance. Ember inserted `{" "}` next to an element within an Astro expression, which requires a fragment around the nodes. After the `</script>` warning, DeepSeek twice moved a JSON-LD script out of a JavaScript string and into the markup, where Svelte treated it as a second script block.
- The other mistakes involved cases the paragraph did not cover. A model left out an Astro layout import, one invented `import { Fragment } from "astro/components"`, and one did not account for the `onload` and `onerror` handlers Svelte adds to `<img {...attrs}>`.
- React had two fewer correct answers. GLM twice failed to provide usable code and once made a logic error. GLM accounted for seven of the eight model errors in this run.
- The run cost $2.98, up from $2.76. DeepSeek spent about 25% more time reasoning when it received the paragraph.

The paragraph was inexpensive and eliminated the specific errors it addressed, with some failures shifting to nearby syntax rules. At the time, the results favored simplifying Shugo's own compiler rules, describing the remaining pitfalls to agents, and reporting them with specific diagnostics. *Since the switch to the Svelte compiler on November 22, 2026, Shugo relies on documentation and diagnostics for Svelte's syntax rules.*

The harness, tasks, reference outputs, and model responses are in `research/template-syntax` in thread storage, including `README.md`, `report/index.html`, and `report/pitfalls.html`.

### Matching Svelte HTML

On November 21, 2026, the project author asked whether Shugo's runtime could emit the same HTML as Svelte's server renderer.

Six Svelte 5 releases rendered the same corpus. It contained the 30 reference tasks and 538 Svelte answers from the models, all rendered on every applicable dataset. Each result was compared with Svelte 5.59.1.

| Version | Pages | Identical bytes | Match after removing hydration markers | Same structure and visible text |
| --- | ---: | ---: | ---: | ---: |
| 5.0.0, October 2024 | 2,208 | 55 | 1,505 | 2,152 |
| 5.16.0, December 2024 | 2,212 | 63 | 1,601 | 2,211 |
| 5.30.0, May 2025 | 2,212 | 70 | 1,787 | 2,211 |
| 5.38.0, August 2025 | 2,212 | 70 | 1,984 | 2,212 |
| 5.45.0, November 2025 | 2,212 | 70 | 1,984 | 2,212 |

- The visible output has matched since 5.38. Earlier releases differed in behavior. Svelte 5.16 added support for arrays and objects in `class`, and 5.38's server output marked the selected `<option>` for `<select value>`.
- The bytes changed more frequently. Hydration markers such as `<!--[-->`, `<!--]-->`, and `<!--[-1-->` accounted for 27% of the page bytes and changed over the two years. Removing markers still leaves small differences. `disabled` became `disabled=""` in 5.50, `<img>` became `<img/>` between 5.30 and 5.38, and extra whitespace in `class` disappeared between 5.16 and 5.30.

The table compares Svelte 5.59's output with Shugo's renderer before the switch.

| Template | Svelte 5.59 | Shugo before the switch |
| --- | --- | --- |
| `{false}` | `false` | Empty |
| `{[1, 2]}` | `1,2` | `12` |
| `disabled={true}` | `disabled=""` | `disabled` |
| `class={["a", f && "b", {c: true}]}` | `class="a c"` | `class="a,false,[object Object]"` |
| Attribute value `"<&>'` | `"&quot;&lt;&amp;>'"` | `"&#34;&lt;&amp;&gt;&#39;"` |
| `href={"javascript:…"}` | Output as written | `about:invalid#blocked` |
| `&nbsp;` and `&copy;` in text | Decoded characters | Output as written |
| Line break between tags | Collapsed to a space and removed at element and block edges | Output as written |
| `title="{x}"` | Value of `x` | Literal `{x}` |
| `<!-- comment -->` | Removed | Preserved |

The comparison informed the following conclusions.

- The visible output can match exactly. Svelte's rules cover about 2,100 lines of server compiler code, 181 lines of whitespace handling, and about 800 lines of runtime code. Shugo already evaluates expressions in JavaScript and could reuse Svelte's functions for rendering text and attributes.
- The comparison can use the native `render()` output as a byte-level reference. The Svelte repository provides 136 SSR examples and 1,348 runes examples in addition to the benchmark tasks and model responses.
- Byte-level matching depends on a single pinned Svelte version. Upgrading means rerunning the comparison and addressing differences. The release history suggests this comes up several times a year.
- Hydration markers are an internal Svelte format and change frequently. Shugo can omit them from static pages and retain the markers Svelte generates for islands.
- Exact matching inherits Svelte's syntax pitfalls. Whitespace at the edge of `{#if}` is still trimmed, independent of any compiler warning.
- Intentional differences require a separate decision. The project initially blocked `javascript:` URLs. On November 22, 2026, it switched to rendering URLs as written.

*On November 22, 2026, the project author chose to adopt Svelte itself. The bundled Svelte 5.59.1 compiler and runtime took over template compilation and rendering. Shugo kept the version pinned and continued removing hydration markers from static pages.*

The comparison is in `research/svelte-exact/README.md` in thread storage.
