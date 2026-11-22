# Components

A component is a reusable part of a template, such as a header, product card, or footer. Like pages and layouts, components are Svelte 5 components in `.svelte` files.

## Decisions

- Define components in `.svelte` files, just like pages and layouts.
- Follow Svelte 5 conventions. Read props with `$props()`, pass markup with snippets, and scope styles to the component. This supersedes reading props through the framework and using `<g-slot>` (November 21, 2026).
- Components can load their own data. A calling template can pass markup to a component.
- A component call originally formed a block boundary, allowing each block to be stored and updated separately. Since Shugo adopted Svelte's compiler and renderer on November 22, 2026, only a shared component call creates a separate block. An in-place component contributes HTML to its caller. See [Blocks and incremental updates](../docs/blocks.md).
- Import functions and components explicitly.
- Passing an object, such as a database row, makes the component render on each page that uses it. Pass the row's key to share one block across pages. This also avoids making pages depend on every column in the row (November 22, 2026).
- A component can become a shared block automatically when it receives only plain props and neither it nor its imports uses `page` or `paginate` from `shugo`. Each distinct set of props produces one record that all matching calls can reference. Snippets declared in the caller's tag do not count toward the props key. The build warns if a component produces many shared blocks that are rarely reused (November 22, 2026).
- The Svelte renderer adds other conditions for sharing. A component must render in place when it uses Svelte context or `$props.id()`, renders a snippet inside `<select>`, receives a `bind:` directive, or receives a snippet with parameters. See [decision 42](decisions.md#42-shared-components-do-not-use-context-or-propsid) and [decision 43](decisions.md#43-render-shared-block-snippets-separately), pending approval.

## Implemented, pending approval

### Importing and rendering

```svelte
<script>
  import Card from "../components/Card.svelte"
  import * as ui from "@/components/ui"
</script>

<Card title="Pitcher" price={2400} featured />
<ui.Button href="/cart">Add to cart</ui.Button>
```

Import a component in `<script>`. A tag with an uppercase initial or a dot names a component. A component can live anywhere under `src/`. By convention, reusable components go in `src/components/`.

Pass props as attributes. `title="Pitcher"` passes a string, `featured` passes `true`, and `price={2400}` passes the value of an expression. `{price}` abbreviates `price={price}`. Use `{...props}` to pass all fields of an object.

### Props

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

A component receives markup inside its tag through the `children` prop. The caller can also declare named snippets and pass them as props.

```svelte
<!-- src/components/Card.svelte -->
<script>
  let { children, footer } = $props()
</script>

<section class="card">
  {#if children}{@render children()}{:else}No content supplied{/if}
  {#if footer}<footer>{@render footer()}</footer>{/if}
</section>

<!-- At the call site -->
<Card>
  <p>This is the card content</p>
  {#snippet footer()}<small>This is the footer</small>{/snippet}
</Card>
```

- `{@render children()}` renders the markup inside the component tag. Use `{@render children?.()}` when the caller may omit it.
- A declaration like `{#snippet name()}…{/snippet}` inside the tag becomes a prop named `name`.
- A snippet can access variables in the template that declares it.
- Snippets can take parameters. The component supplies the arguments when it renders the snippet.

```svelte
<!-- src/components/List.svelte -->
<script>
  let { items, row } = $props()
</script>
<ul>{#each items as item}<li>{@render row(item)}</li>{/each}</ul>

<!-- At the call site -->
<List items={products}>
  {#snippet row(product)}<a href="/products/{product.slug}">{product.title}</a>{/snippet}
</List>
```

- You can declare a snippet anywhere in the template and pass it as a prop, as in `<List items={products} {row} />`.
- A component can forward a snippet with `<Inner>{@render children?.()}</Inner>` or `<Inner {children} />`.

### Loading component data

```svelte
<script>
  import { query } from "shugo"

  const menu = query("menu").order("position").all()
</script>

<nav>{#each menu as item}<a href={item.href}>{item.title}</a>{/each}</nav>
```

A component's `<script>` runs each time it renders. An in-place component used on 100,000 pages runs 100,000 times. Matching calls to a shared component reuse the same rendered block.

### Shared components

Shugo can compute and store a component's output once for a given set of props if the component is independent of the page and its position in the template.

```svelte
<Header />                      <!-- once for the site -->
<ProductCard id={product.id} /> <!-- once per product -->
<ProductCard item={product} />  <!-- each page passes an object -->
```

- Imports of `page` or `paginate` from `shugo`, either directly or through another module, make the component depend on the current page. The compiler detects this from the import graph and renders the component separately for each page.
- A component depends on its position if it or something it imports and renders uses Svelte context (`getContext`, `setContext`, `hasContext`, `getAllContexts`, or `createContext`), calls `$props.id()`, or renders a snippet inside `<select>`. Context comes from surrounding components, the generated ID depends on components already rendered on the page, and a `<select>` chooses among `<option>` elements supplied by its snippet. A `bind:` also ties the component to its caller. These components render in place. See [decision 42](decisions.md#42-shared-components-do-not-use-context-or-propsid).
- Shared props can be strings, finite numbers, booleans, or `null`. Props may also be absent.
- A parameterless snippet declared inside a shared component's tag can render separately where it is declared, including any contributions to `<head>`. Shugo stores its output in the caller's record and inserts it where the shared component renders it. If the separate render fails or calls `$props.id()`, the component renders in place. This mechanism also lets [layouts store their records separately](layouts.md#layouts-are-stored-separately-from-pages) and receive a page as `children`.
- A snippet with parameters renders with arguments supplied by the component, even if all parameters have defaults. Passing such a snippet, or passing a snippet as an attribute like `<List {row} />`, makes the component render in place.

For a product card, pass the product ID and let the card query the product. Updating the product then changes one shared block even if many pages render it.

After a full computation, such as `shugo build` or a code change, the build checks the reuse of shared components. It warns if a component produces at least 100 shared blocks and each block appears in fewer than 1.5 places on average.

```text
warning: src/components/Price.svelte is kept apart as 9000 shared blocks, each shown in 1.0 places on average. …
```

This happens when nearly every page supplies a different value to `<Price value={n} />`. Storing each of those blocks separately adds a lookup to each page with little reuse. A component that receives a row object renders as part of its page. Layouts are excluded from this check because a URL-dependent layout intentionally uses a record per page.

### Recursive components

A recursive component imports itself.

```svelte
<!-- src/components/Tree.svelte -->
<script>
  import Tree from "./Tree.svelte"
  let { node } = $props()
</script>

<li>{node.title}
  {#if node.children.length}
    <ul>{#each node.children as child}<Tree node={child} />{/each}</ul>
  {/if}
</li>
```

Svelte still accepts `<svelte:self>` and emits a deprecation warning.

When a shared component calls itself with the same props, directly or through other shared components, Shugo reports a build error with the block names. Unbounded in-place recursion can fail with JavaScript's `Maximum call stack size exceeded` without a source location.

### Dynamic components

```svelte
<script>
  import Grid from "./Grid.svelte"
  import List from "./List.svelte"
  let { compact, items } = $props()
  const View = compact ? List : Grid
</script>

<View {items} />
```

Svelte still accepts `<svelte:component this={View}>` and emits a deprecation warning.

### Component styles

A component's `<style>` applies to its own elements. See [Styles](styles.md).

## Open questions

- Build-time prop type checking is planned as part of `shugo check`.
