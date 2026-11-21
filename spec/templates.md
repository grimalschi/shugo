# Page Templates

The document describes the `.shugo` file format: what parts the file is made of, how expressions are written, and control tags.

## Decisions

- The framework has its own file format - `.shugo`. It is similar to Astro.
- The file consists of two parts:

  ```
  ---
  code on TypeScript
  ---
  page, layout or component layout
  ```

- The code at the top is arranged like in Astro: it is a top-level code, and its variables are visible to the template. Among them may be functions.
- Component parameters are read in this code, as in Astro.
- The code can export `getStaticPaths`, a list of dynamic pages.
- The expressions in the markup are JS in curly brackets: `{expression}`. The attribute with the expression is written `attribute={expression}`.
- Terms and cycles — their tags: `g-if`, `g-else-if`, `g-else`, `g-for`.
- The functions are connected by obvious imports: less magic.
- An expression is any single expression of JS, including method calls and arrow functions.
- Expression does not change the data that is read.
- The values are screened when inserted into the page.
- An error in expression shows a row and column in the template.
- `null` and `undefined` are displayed as an empty string, and scalars as is.

## Implemented, pending approval

### Code

```
---
import Card from "../components/Card.shugo"
import { price, type Product } from "@/lib/shop"

interface Props { limit?: number }
const { limit = 10 } = Shugo.props as Props

const products: Product[] = Shugo.query("products").order("title").limit(limit).all()
const total = (items: Product[]) => items.reduce((sum, item) => sum + item.price, 0)
---
<p>Total: {price(total(products))}</p>
```

- The code is executed every time the file is shown: once for the page, once for each component call. Common blocks are an exception, see [Components](components.md).
- `await` no: the page is calculated without waiting, because `Shugo.query` responds immediately ([Data loading](data-loading.md)). What is expected and done once, for example, creating code highlighting, is expected at the top level of the module from `lib/`.
- Imports and exports remain at the module level and are performed once. `getStaticPaths` does not see the variables of the rest of the code, as in Astro.
- `return` at the top level and `export default` are build errors.
- `@/` in import path means project root.
- The object `Shugo` is described in [7](decisions.md#7-object-shugo).

### Expressions

```
<h1>{product.title}</h1>
<p>{price(product.price * (100 - discount) / 100)}</p>
<p>{product.tags.join(", ")}</p>
<p>{user ? user.name : "guest"}</p>
```

The expression is a regular TypeScript, and only it: you can not write the markup inside the braces. The compiler answers that `<g-for>` is needed for writing `{items.map(item => <li>{item}</li>)}`.

What is displayed for each type of value is in [12](decisions.md#12-what-outputs-expression-and-attributeexpression).

### Attributes

```
<a href={url} class={active ? "link active" : "link"} title="plain text">…</a>
<input disabled={!ready} value={query} {...rest}>
<img {src} alt="">                 <!-- the same, what src={src} -->
```

The brackets inside the quotes are plain text: `title="{x}"` will output `{x}`. The line with the substitution is written as follows: ``class={`card ${kind}`}``.

### Condition

```
<g-if cond={product.stock > 3In Stock</g-if>
<g-else-if cond={product.stock > 0Remained {product.stock}</g-else-if>
<g-else>Ordered</g-else>
```

Branches are close by, only spaces are allowed between them. One branch is executed and gets to the page.

### Cycle

```
<g-for each={products} as="product" index="i">
  <li>{i + 1}. {product.title}</li>
</g-for>
```

- `each` is any object being searched: array, `Set`, `Map`. `null` and `undefined` are considered an empty list.
- `as` is the name of the element. You can disassemble the element immediately: `as="{ id, title }"`, `as="[key, value]"`.
- `index` — element number from scratch, optional.

### Ready HTML

```
<article><g-html value={post.body} /></article>
```

Value is inserted without shielding.

### Components and Slots

```
<Card title="Pitcher" featured>
  <p>content</p>
  <small slot="footer">v Footer</small> Slot
</Card>
```

See also [Components](components.md).

### Markup

- Each tag closes. Exceptions are: tags without content (`<br>`, `<img>`, `<input>` and others) and a `<div />` entry, which means empty tag.
- The contents of `<script>` and `<style>` are not understood: the curly brackets there are ordinary text.
- Comments from `<!-- -->` are included in the page. Comments that do not appear in the page are: `{/* text */}`.
- Spaces and line transfers are displayed as written. Spaces around the edges of the entire template are cropped.
- The literal brace in the text is `{"{"}` or `&#123;`.

### What is not and what the compiler answers

| Record from Astro | Response |
| --- | --- |
| `{cond && <b>…</b>}`, `{list.map(x => <li>…</li>)}` | markup can not stand inside `{…}`: you need `<g-if>` and `<g-for>` |
| `set:html={…}` | `<g-html value={…} />` |
| `class:list={…}` | collect a line in the expression |
| `<slot />`, `<Fragment>` | `<g-slot>` and `<g-fragment>` |
| `client:load` and similar | pages are static, for the browser you need `<script>` |
| `Astro.props` | `Shugo.props` |

## Open questions

- Type checking in templates: the compiler can release for each `.ts` file that checks the standard `tsc`. In the first version, types only erase.
- The key element in `<g-for>`: not needed now, see [4](decisions.md#4-the-key-in-g-for-is-not-entered).
- `<g-else>` after `<g-for>` for empty list.
- The HTML tag whose name is calculated: `const Tag = "h2"`. Now the capital-letter tag is always a component.
- Warning when an object is displayed in the text: it is almost always a mistake.
- Remove spaces between tags, as Astro 7 does.
- Template syntax: own tags `g-*`, Astro syntax or Svelte. Below is the test to choose from.

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

How Svelte 5.59 writes values and how it differs from Shugo now:

| In the template | Svelte 5.59 | Shugo |
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
- Where to deviate from Svelte consciously is decided separately: for example, whether to block `javascript:` addresses, as now.

Checking is in the trust store: `research/svelte-exact` (`README.md`).
