# Shugo and Svelte

Ordinary `.svelte` Shugo the Official Parser Svelte 5.59.1. The markup program and the code for reading are built from the resulting tree. To `.island.svelte` Svelte It also compiles browser code, and Shugo prepares the program SSR with hydration marks. [Templates](../spec/templates.md) · [Svelte islands](islands.md)

## How Shugo repeats Svelte

Parser, tree conversion and HTML generation use the following parts of Svelte and Shugo.

| What | Where to Svelte | Where in Shugo |
| --- | --- | --- |
| Analysis of markup, blocks and tags | `svelte/compiler.parse` | `internal/syntax/svelte-ast.mjs` translates official AST to compile nodes Shugo |
| Spaces: what is removed, what is compressed, `<pre>` and `<textarea>` | `clean_nodes` | `internal/compile/lower.go` |
| What not to invest in: mistakes and warnings | `html-tree-validation.js` | `internal/compile/nesting.go` |
| Component Style Class and Selector Rewriting | hash `cssHash`, `phases/3-transform/css` | `internal/compile/css.go` |
| Text and attributes: `stringify`, `attr`, `clsx`, `to_class`, `to_style`, attributes from `{...}` | `internal/shared/attributes.js`, `internal/server` | `internal/jsrt/runtime.js` |
| Screening: `&` and `<` in text, more `"` in attributes | `escape_html` | `internal/render/escape.go` |
| Page head in order of document, one `<title>` is the last | `renderer.js` | `internal/render` |

The value of the expressions is considered to be the real JS Bun or Node so everything that depends on JS coincides with the same: numbers, dates, `String()` arrays and objects.

The Svelte snippets and component parameters are placed on Shugo blocks: a parameter-free snippet written inside the component tag is calculated where it is written and stored in the caller's record; a parameter-based snippet is calculated where it is shown by `{@render}`. See also [Blocks](blocks.md).

The document around the Svelte page does not write: `render()` gives the head and body separately. Shugo, like SvelteKit with `src/app.html`, puts them in the shell of `src/site.html`: head in place of `%shugo.head%`, body in place of `%shugo.body%`. Therefore, the root layout starts immediately with the contents of `<body>`, like `+layout.svelte`, and the template that `<html>`, `<head>` or `<body>` writes is an error. The address and page parameters also give the object modeled on SvelteKit: `page` from the `shugo` module with `url`, `params`, `route.id` and `status` fields, as `page` from `$app/state`.

## How it is checked

- **Tasks of testing.** V `test/svelte/` The reference components 30 The test (below) and what Svelte 5.59.1 It gives on each data set. `TestSvelteTasks` collects each task as a site and compares pages. The standards do it yourself Svelte: `cd test/svelte && bun install && bun render.mjs`.
- **Tests of patterns.** The expected HTML in `test/template_test.go` is taken from the output of Svelte on the same templates.
- **Random pages.** `TestRandomPages` writes pages at random and considers them HTML as its model on Go, independent of the Shugo code. The model is checked with Svelte 5.59.1 on 400 random pages: matched all. In the course of reconciliation, two discrepancies of Shugo with Svelte were found and corrected: `class` and `style` without value.
- Before comparing, Svelte removes hydration tags, and the style class is removed from both: Svelte considers it from a different file path and does not put it on all elements ([Styles](../spec/styles.md)).

When updating Svelte synchronously change the built-in compiler, the version of the package in `examples/search` and `test/svelte` and benchmark tests. [Built-in compiler](../internal/syntax/vendor/README.md)

## Cost of analysis and small changes

For example, `examples/basic` cold build from the new catalog took 43 ms with the former parser and 152 ms with the official. The compilation phase was taken by 20 and 130 ms respectively. These are the medians of seven alternate runs on Bun.

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

We write the Shugo compiler, so the behavior can be made as models expect. In Astro syntax: slot under false condition is not passed; component can call itself by name; `return` in code between `---` means "show nothing". When Svelte syntax: the space at the edge of the block is saved. In this run, such rules would remove 12 from the 13 errors on Astro and 7 from the 14 on the Svelte.

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

Conclusion: the paragraph is cheap and works, but it only cures what is named in it, and it itself shifts mistakes. The most reliable behavior that models expect without a clue: a rule that does not exist, can not be forgotten. For Shugo, this means first removing the traps in the compiler that are left to describe for agents in short text and catching clear messages from the compiler.

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
- Where to deviate from Svelte consciously is decided separately: for example, whether to block addresses of `javascript:`. The author of the project first decided to block, and 22 November 2026 year canceled: addresses are displayed as in Svelte.

Checking is in the trust store: `research/svelte-exact` (`README.md`).
