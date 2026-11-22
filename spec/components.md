# Components

Component is the reusable part of the template: the cap, the product card, the footer. This is a `.shugo` file, the same component of Svelte 5 as the page and layout.

## Decisions

- The term is called “component.”
- The component is a `.shugo` file, like page and layout.
- The components are written as in Svelte 5, with the same markup logic (the decision of the author of the project 21 November 2026 year): parameters - `$props()`, markup inside the component is transmitted by snippets, the styles of the component act only on it. This replaces the previous solutions "parameters are read in code, as in Astro" and `<g-slot>` slots.
- The component can upload its data.
- The calling pattern can transfer the markup inside the component.
- Component call is the block boundary: the block is stored and updated separately. As it is arranged in [Blocks and point update](../docs/blocks.md).
- Functions and components are connected by explicit imports.
- The component to which the object was transferred for example, a base line is calculated on each page where it is shown. Remember it by the content of the parameter is not necessary: so the page would depend on all columns of the line. Who needs a common unit, he transmits the key (solution 22 November 2026 year).
- **What a common component.** A component that does not take `page` and `paginate` from `shugo` neither itself nor through its imports and receives only simple values strings, numbers, `true`, `false`, `null` or nothing all of these are stored once per page. Snippets passed inside are not considered parameters. No markings needed. The build warns when the common component shares almost nothing: it has many entries, and each shows one place on average (22 solution of November 2026 year).

## Implemented, pending approval

### Connection and challenge

```svelte
<script>
  import Card from "../components/Card.shugo"
  import * as ui from "@/components/ui"
</script>

<Card title="Pitcher" price={2400} featured />
<ui.Button href="/cart">To cart</ui.Button>
```

The component is imported into `<script>`, as in Svelte. The capital letter or dot tag is a component. The file can lie anywhere inside `src/`; the `src/components/` folder is an agreement.

The parameters are written as attributes: `title="text"` transmits a string, `featured` transmits `true`, `price={2400}` transmits an expression value, `{price}` transmits the same value as `price={price}`, and `{...props}` transmits all object fields.

### Parameters

```svelte
<script lang="ts">
  interface Props {
    title: string
    price: number
    featured?: boolean
  }
  let { title, price, featured = false }: Props = $props()
</script>

<a class={["card", { featured }]}>{title}: {price}</a>
```

### Snippets

What is inside the component tag, the component gets the parameter `children`. Named pieces of markup are transmitted by snippets - also parameters.

```svelte
<!-- src/components/Card.shugo -->
<script>
  let { children, footer } = $props()
</script>

<section class="card">
  {#if children}{@render children()}{:else} is shown if nothing was passed inside{/if}
  {#if footer}<footer>{@render footer()}</footer>{/if}
</section>

<!-- usage -->
<Card>
  <p>this Children</p>
  {#snippet footer()}<small>this footer</small>{/snippet}
</Card>
```

- `{@render children()}` shows what is inside the component tag. `{@render children?.()}` shows nothing if the inside is empty.
- `{#snippet name()}…{/snippet}` is right inside the component tag, the `name` parameter.
- Snippet sees the variables of the template that wrote it, not the component.
- The snippet can accept parameters: the component transmits them when shown.

```svelte
<!-- src/components/List.shugo -->
<script>
  let { items, row } = $props()
</script>
<ul>{#each items as item}<li>{@render row(item)}</li>{/each}</ul>

<!-- usage -->
<List items={products}>
  {#snippet row(product)}<a href="/products/{product.slug}">{product.title}</a>{/snippet}
</List>
```

- The snippet can be declared anywhere in the template and passed as a normal parameter: `<List items={products} {row} />`.
- The component can transmit the snippet further: `<Inner>{@render children?.()}</Inner>` or `<Inner {children} />`.

### Your data

```svelte
<script>
  import { query } from "shugo"

  const menu = query("menu").order("position").all()
</script>

<nav>{#each menu as item}<a href={item.href}>{item.title}</a>{/each}</nav>
```

`<script>` component is executed at each display. For a component that is shown on 100,000 pages, this is 100,000 times — unless it is shared.

### Common components

The component is calculated and stored once, if it is independent of the page and called with simple parameters:

```svelte
<Header />                      <!-- once for the entire site -->
<ProductCard id={product.id} /> <!-- once per product -->
<ProductCard item={product} />  <!-- once per page because the prop is an object -->
```

- The component depends on the page, if it itself or the module it imports takes `page` or `paginate` from the `shugo` module: this is visible by imports, without starting the code. Such a component is calculated on each page anew.
- Simple parameters are strings, numbers, `true`, `false`, `null`.
- Snippets transferred to a common component do not prevent it from being shared: their markup is counted in the template that wrote them and stored with it. Therefore, the layout is usually shared: the page it receives with the snippet `children`.

Hence the reception: it is better to give the card the number of goods, and she will find the goods herself. Then the card is considered once, and when changing the goods, one record changes, no matter how many pages it shows.

When the entire site is calculated after `shugo build` or code edits the build weighs the common components. If the component 100 has shared blocks or more and each is shown on average in less than one and a half places, it warns:

```
warning: src/components/Price.shugo is kept apart as 9000 shared blocks, each shown in 1.0 places on average. …
```

This happens with `<Price value={n} />`, when almost every page has its own number: there is no need to store such a block separately, and it is worth the extra search in the storage. The component to which the object was transferred, such as the string it shows, is calculated as part of the page. Layouts are not weighted: A layout that reads a page is stored for each page on purpose.

### Challenge yourself

As in Svelte 5: the component imports itself.

```svelte
<!-- src/components/Tree.shugo -->
<script>
  import Tree from "./Tree.shugo"
  let { node } = $props()
</script>

<li>{node.title}
  {#if node.children.length}
    <ul>{#each node.children as child}<Tree node={child} />{/each}</ul>
  {/if}
</li>
```

`<svelte:self>` also works, with a warning: in Svelte 5 it is outdated. The component that shows itself with the same parameters without end is an assembly error.

### Component of variable

```svelte
<script>
  import Grid from "./Grid.shugo"
  import List from "./List.shugo"
  let { compact, items } = $props()
  const View = compact ? List : Grid
</script>

<View {items} />
```

`<svelte:component this={View}>` also works, with a warning like in Svelte 5.

### Component Styles

`<style>` component acts only on its elements, as in Svelte. Details - in [Styles](styles.md).

## Open questions

- Checking the types of parameters during assembly: it is decided to do together with the `shugo check` command.

