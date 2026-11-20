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

## Made in the first version, waiting for approval

### Code

```
---
import Card from "../components/Card.shugo"
import { getProducts, price, type Product } from "@/lib/shop"

interface Props { limit?: number }
const { limit = 10 } = Shugo.props as Props

const products: Product[] = (await getProducts()).slice(0, limit)
const total = (items: Product[]) => items.reduce((sum, item) => sum + item.price, 0)
---
<p>Total: {price(total(products))}</p>
```

- The code is executed every time the file is shown: once for the page, once for each component call. Common blocks are an exception, see [Components](components.md).
- `await` is allowed on the top level.
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
