# Shugo and Svelte

Shugo templates — pages, layouts and components, `.svelte` files are components of Svelte 5. Compiles them Svelte 5.59.1 s output for the server, as in SvelteKit, and renders when assembling its server part on Bun or Node. The compiler, server and browser parts are sewn into Shugo: the project does not need to put the `svelte` package. Shugo has its own blocks, which are the pages, and what is around the pages: the shell, head, storage, server. `.island.svelte` Island the same Svelte compiles for server and browser; for the island, tied to the query data, Shugo prepares from the server output Svelte program SSR. [Templates](../spec/templates.md) · [Svelte islands](islands.md) · [Blocks](blocks.md)

## What Svelte and Shugo do

| What | Who | Where |
| --- | --- | --- |
| Analysis, errors and warnings, including about availability (`a11y_…`) | Svelte, `svelte/compiler.compile` with `generate: "server"` and `runes: true` | `internal/svelte/compile.mjs` |
| Spaces, shielding, attributes, `class` and `style`, `{#each}`, `{#if}`, snippets, `<svelte:element>`, `<select value>` | Svelte: compiled code and its server part | `internal/svelte/vendor/runtime` |
| Component Styles: `svelte-…` class, rewritten selectors, unused selectors | Svelte, `css: "external"` | styles assemble in file Shugo, `internal/compile/styles.go` |
| Types of TypeScript | `<script lang="ts">` erases Svelte, `enum` and other things that TypeScript turns into code — error Svelte; in modules `.ts` — esbuild | `internal/svelte/compile.mjs`, `internal/compile/build.go` |
| Your checks: shell, islands, tags of the first syntax Shugo, slots Svelte 4, `runes={false}`; hint when `{…}` is worth marking | Shugo, by tree, which gave the same call to the compiler, and by its error | `internal/svelte/compile.mjs` |
| Import layout, components from npm packages, tip about type without `type` | Shugo, in assembler | `internal/compile/build.go` |
| A component that endlessly shows itself | Shugo, when calculating: error with file chain | `internal/jsrt/runtime.js` |
| What component is the common block and what renders on the spot | Shugo: when assembled - by template tree and imports, with each display - by parameters, in `component()` wrapper around each component | `internal/svelte/compile.mjs`, `internal/compile/build.go`, `internal/jsrt/runtime.js` |
| Common block labels, snippets of common blocks in separate renders, cutting HTML into blocks, `<title>` separately from the head | Shugo | `internal/jsrt/runtime.js` |
| Modules with runes, `.svelte.js` and `.svelte.ts` | Svelte, `compileModule`; types `.svelte.ts` removes esbuild, source cards add up | `internal/compile/build.go` |
| Comments on hydration `<!--[-->`, `<!--]-->`, `<!---->` and others | writes Svelte, cuts Shugo — except for the islands and `{@html}` | `internal/jsrt/runtime.js` |
| Page head in order of blocks, one `<title>` is the last | Shugo: server when collecting page | `internal/render` |
| Document around the page, `src/site.html` | Shugo | `internal/skeleton`, `internal/render` |

The pages code is performed by Bun or Node, and HTML writes Svelte: how numbers, dates, arrays and objects, spaces and attributes are displayed, he decides.

Shugo has blocks. A component that shows the same thing everywhere it stands with the same parameters is rendered once, and pages are stored in its place a hole. Snippets without the parameters that the place gave such a component are rendered separately and stored in the place record. What component is common and how HTML is cut into blocks is in [Blocks](blocks.md#shared-components).

The document around the Svelte page does not write: `render()` gives the head and body separately. Shugo, like SvelteKit with `src/app.html`, puts them in the shell of `src/site.html`: head in place of `%shugo.head%`, body in place of `%shugo.body%`. Therefore, the root layout starts immediately with the contents of `<body>`, like `+layout.svelte`, and the template that `<html>`, `<head>` or `<body>` writes is an error. The address and page parameters also give the object modeled on SvelteKit: `page` from the `shugo` module with `url`, `params`, `route.id` and `status` fields, as `page` from `$app/state`.

## Where Shugo and Svelte diverge

- **Comments of hydration** Shugo cuts: static page they are not needed, and this is a quarter of its byte. They remain at the islands — on them the island comes to life in the browser — and in HTML, which inserts `{@html}`. As well as what — in [Blocks](blocks.md#how-html-is-split-into-blocks).
- **The layout is rendered separately** from the page and from the layouts inside it. The context he sets through `setContext` sees its components, but not the page and not the nested layouts. Names from `$props.id()` in the layout begin with his file number: `b3-s1`.
- **Only runes.** Templates compile with `runes: true`: Svelte syntax 4 is `export let`, `$:` is a Svelte error. `<slot>`, `slot="…"`, `let:` and `<svelte:fragment>` Svelte still accepts with a warning in this mode, and Shugo considers a mistake with a hint about snippets: the slot that the place gave the common block could not show the common block.
- **Warnings** about what changes in the browser — `state_referenced_locally` and neighboring — Shugo does not show: a static page is rendered once. They are printed about the island: its Svelte compiles for the browser as well.

## How it is checked

HTML writes Svelte. Tests check that after cutting into blocks, cutting out comments and assembling the head, the page remains the one given by Svelte, without hydration marks.

- **Tasks of testing.** The `test/svelte/` contains the reference components of the 30 test tasks (below) and what the Svelte 5.59.1 gives on each data set. `TestSvelteTasks` collects each task as a site and compares pages with a reference. The Svelte: `cd test/svelte && bun install && bun render.mjs` itself makes the reference. Before the 22 of November, 2026 year this test was verified with the Svelte compiler Shugo.
- **Comments.** `TestNoCommentsOfSvelte` collects a page with all kinds of Svelte blocks, common components in the head and `$props.id()` and checks that Svelte comments are not left in it, and comments from `{@html}` remain.
- **Tests of patterns.** The expected HTML in `test/template_test.go` is taken from the output of Svelte on the same templates.
- **Svelte across the blocks.** `TestSvelteSemanticsAcrossBlocks` collects templates where shared blocks could change the meaning: `bind:`Snippets on condition, `<select>` With a snippet, `$props.id()` In the snippet, which is shown twice, `<title>` From the snippet, the names of the snippets are not Latin, the extra arguments are snippets, `slot` at the custom element. Expected HTML Everyone is what gives Svelte on the same files.
- **Random pages.** `TestRandomPages` writes pages at random and considers them HTML as its model on Go, independent of the Shugo code. The model is checked with Svelte 5.59.1 on 400 random pages: all matched.
- Before comparing, Svelte output removes the hydration labels, and both removes the style class: the reference is compiled with the other file path, and the hash class is the other.

When updating Svelte synchronously change the sewn compiler, server and browser parts and parser acorn, the version of the package in `test/svelte` and benchmark tests, and programs SSR The islands are re-aligned with HTML Svelte ([Svelte islands](islands.md#what-does-the-ssr-program-of-the-island)). [How a copy is made](../internal/svelte/vendor/README.md)

## How much does the compilation cost

In the example of `examples/basic` seven templates cold assembly took 0.21 s, of which the compilation is 0.18 s. 22 November 2026 years, while Svelte only parsed the templates, and compiled them Shugo, was 0.15 and 0.13 s. The Svelte compiler runs behind the 64 ms; the first template compiles 40 ms while Bun warms up the compiler code, following 1–12 ms. A large project is compiled several processes at once, one on the 32 template, no more than four.

On the store in 100,000, the first assembly is 14.6 s, an update after a change in one price is 31 ms. 21 November 2026 year, when the templates compiled their compiler Shugo, was 14.5 s and 29 ms. Almost all of the difference is the time Bun loads the bundle: it now has the server part of Svelte, 123 KB code instead of 73 KB. Exporting the entire site has become twice as fast, 3.5 s instead of 5.0 s: Go stitches finished pieces of HTML rather than assembles a page from values on the frame of templates. Storage on 44% is larger, 202 MB instead of 140: the records contain HTML, not values. See also [README booth](../examples/catalog/README.md#results).

How much does it cost to update a site with a search - changing the column outside the index and inside it - in [Svelte](islands.md#search).

## Test: How models write templates without documentation

Conducted by 21 on November 2026 year at the request of the author of the project. This is a study, not a solution.

Three models through OpenRouter — DeepSeek V4.1 Flash, Muse Ember, GLM 5.3 Flash — wrote 30 templates on Astro, React and Svelte, each three times. They did not have documentation and search: only a task, a type of parameters and one example of HTML. Tasks are conditions, cycles, classes and styles from data, attributes, raw HTML, slots, recursion, sorting and grouping. The answer is rendered by the real engine — Astro 7.4, React 19.3, Svelte 5.59 — on the example and on 2–3 datasets that the model has not seen. The reference solutions on the three syntaxes give the same pages on all sets.

| Model | Astro | React | Svelte |
| --- | ---: | ---: | ---: |
| DeepSeek V4.1 Flash | 86/90 | 90/90 | 82/90 |
| Muse Ember | 86/90 | 90/90 | 89/90 |
| GLM 5.3 Flash | 85/90 | 89/90 | 85/90 |
| All three | 257/270 | 269/270 | 256/270 |

The logic of model templates is known on all three syntaxes. Errors are several traps where the behavior of the framework diverges from what the model expects:

- **Astro, 9 answers from 13 infidels:** slot, transmitted under a false condition (`{note && <p slot="footer">…</p>}`), Astro considers transferred: `Astro.slots.has()` returns `true`, spare content is not shown. So all models are wrong in all attempts; the usual assembly of `astro build` behaves the same way.
- **Astro, the rest of 4:** habits from React and Svelte are markup in code between `---`, a component calling itself by name instead of `Astro.self`, `return` in code between `---`, `{…}` inside the attribute string.
- **Svelte, 7 responses from 14 infidels:** space at the edge of the `{#if}` block Svelte removes, and the reader sees "$2,800$2,400". Another 2 response did not import the component, 1 wrote `</script>` in comments inside `<script>`.
- **React:** is the only error in logic, the same attempt of the same model was also mistaken on Svelte.

The compiler Shugo then wrote we, so the behavior could be made as expected models. In Astro syntax: slot under false condition is not passed; component can call itself by name; `return` in code between `---` means "show nothing". When Svelte syntax: the space at the edge of the block is saved. In this run, such rules would remove 12 from the 13 errors on Astro and 7 from the 14 on the Svelte. *With 22 November 2026 year templates compiles itself Svelte, and its behavior Shugo does not change.*

### Paragraph about traps in the pump

Second run, same day: same tasks, models and attempts. The only difference is that in the prompt before the last indications there is one paragraph about the rules of syntax, on which the models were wrong for the first time. For Astro: markup only in the template, `return` in the code between `---` can not, call themselves through `<Astro.self />`, `{…}` inside the attribute string is not calculated, conditional slot is considered transferred, space with line translation disappears. For Svelte: the gap at the edge of the block is gone, each component is imported, `</script>` anywhere the code block closes it. For React, the JSX rules are: spaces, `{count && …}` at zero, and DOM attribute names.

| Model | Astro | React | Svelte |
| --- | ---: | ---: | ---: |
| DeepSeek V4.1 Flash | 86 → 89 | 90 → 90 | 82 → 88 |
| Muse Ember | 86 → 89 | 90 → 90 | 89 → 89 |
| GLM 5.3 Flash | 85 → 87 | 89 → 87 | 85 → 87 |
| All three | 257 → 265 | 269 → 267 | 256 → 264 |

The number of incorrect answers was half as high: 14 instead of 28. Each incorrect answer of both runs read:

| Reason | Without paragraph | With paragraph |
| --- | ---: | ---: |
| The traps that the paragraph refers to | 23 | 0 |
| Other syntax traps | 0 | 6 |
| Errors of the models themselves: logic, JavaScript, broken answer, no code answer | 5 | 8 |

- All these traps are gone: such rules of the model are followed. But the paragraph was written on the errors of the first run on the same tasks, so this is the best case.
- Half of the new traps caused the paragraph itself: the model bypasses one trap and falls into the next. Ember has placed the suggested `{" "}` next to the element inside the expression, and Astro does not disassemble such without `<>…</>`. DeepSeek, warned about `</script>`, moved the JSON-LD tag from line in code to markup, and Svelte mistook it for a second block of code twice.
- The other half are the traps that the paragraph is silent about: forgotten layout imports to Astro; fictional `import { Fragment } from "astro/components"`; `onload` and `onerror` handlers, which Svelte himself adds to `<img {...attrs}>`.
- React lost two answers not because of syntax: GLM twice did not give working code and once made a mistake in logic. Of the 8 errors of the 7 models themselves, GLM has.
- The run cost $2.98 instead of $2.76: DeepSeek with a paragraph argues a quarter longer.

Conclusion: the paragraph is cheap and works, but it only cures what is named in it, and it itself shifts mistakes. The most reliable behavior that models expect without a clue: a rule that does not exist, can not be forgotten. For Shugo, this meant first removing the traps in the compiler that remained to describe for agents in short text and catching clear messages from the compiler. *With 22 November 2026 year templates compiles itself Svelte: traps of its syntax remain, they can only be described and caught by messages.*

The stand, tasks, standards and all answers are in the repository of the trust: `research/template-syntax` (`README.md`, `report/index.html`, `report/pitfalls.html`).

### HTML is exactly like Svelte

21 was reviewed on November by 2026 on the question of the author of the project: can the Shugo runtime issue the same HTML that the Svelte server.

Six versions of Svelte 5 have rendered the same set of standards 30 test objectives and 538 model responses to Svelte, on all data sets. Each page was compared with that given by the version of 5.59.1.

| Version | Pages | Byte byte | No hydration marks | Structure and visible text |
| --- | ---: | ---: | ---: | ---: |
| 5.0.0, October 2024 | 2 208 | 55 | 1 505 | 2 152 |
| 5.16.0, December 2024 | 2 212 | 63 | 1 601 | 2 211 |
| 5.30.0, May 2025 | 2 212 | 70 | 1 787 | 2 211 |
| 5.38.0, August 2025 | 2 212 | 70 | 1 984 | 2 212 |
| 5.45.0, November 2025 | 2 212 | 70 | 1 984 | 2 212 |

- What the reader sees has not changed since the version of 5.38. Previously, the meaning also changed: `class` from the array and the object appeared in 5.16, the server selected by `<option>` from `<select value>` notes with 5.38.
- Bytes are constantly changing. Hydration tags — comments `<!--[-->`, `<!--]-->`, `<!--[-1-->` this 27% byte of the page, and their appearance has changed in two years. Without labels, pages diverge in small things: `disabled` Stylish `disabled=""` the City of London 5.50, `<img>` Stylish `<img/>` between 5.30 and 5.38There is an extra gap in `class` disappeared between 5.16 and 5.30.

How Svelte 5.59 writes values and how it differed from Shugo before switching to Svelte:

| In the template | Svelte 5.59 | Shugo before the transition |
| --- | --- | --- |
| `{false}` | `false` | nothing |
| `{[1, 2]}` | `1,2` | `12` |
| `disabled={true}` | `disabled=""` | `disabled` |
| `class={["a", f && "b", {c: true}]}` | `class="a c"` | `class="a,false,[object Object]"` |
| `"<&>'` string in attribute | `"&quot;&lt;&amp;>'"` | `"&#34;&lt;&amp;&gt;&#39;"` |
| `href={"javascript:…"}` | as is | `about:invalid#blocked` |
| `&nbsp;` and `&copy;` in text | symbols | how written |
| moving a string between tags | is compressed into a space, at the edges of the element and the block is removed | how it is written |
| `title="{x}"` | value `x` | text `{x}` |
| `<!-- comment -->` |  | remains |

What follows:

- The visible result can be repeated in full. The rules by which Svelte writes HTML are unambiguous and small: about 2,100 rows of the server compiler, 181 row of the space rules and about 800 rows of the runtime. The Shugo expressions are already computed in the present JS; if you convert values to text and attributes by Svelte functions taken as is, the match will be by construction.
- Match is checked, not promised: the real Svelte serves as a benchmark, the Shugo page is compared to its `render()` byte per byte. Examples for comparison are enough: in the Svelte repository, 136 examples of server render and 1,348 examples on runes, plus the tasks and answers of this test.
- Byte byte - only with one fixed version of Svelte. Update version is a comparison run and correction of discrepancies, judging by the history - several times a year.
- It is not necessary to repeat the hydration labels: this is the internal format of Svelte, it changes most often, and it is not needed for a static page. The island of the label is given by Svelte itself, which renders it.
- The one-to-one match also includes Svelte traps: the gap at the edge of `{#if}` will disappear in the same way as in Svelte. The compiler warning about such a gap HTML does not change.
- Where to deviate from Svelte consciously is decided separately: for example, whether to block the addresses of `javascript:`. The author of the project first decided to block, and 22 November 2026 canceled: the addresses are displayed as is.

*22 November 2026 year author of the project decided not to repeat Svelte, and take it himself ("okay, take the swelt"): templates compiles and renders Svelte 5.59.1, sewn into Shugo. It is no longer necessary to repeat its rules; the fixed version, cut hydration marks and Svelte traps remain.*

Checking is in the trust store: `research/svelte-exact` (`README.md`).
