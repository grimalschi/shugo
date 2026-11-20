# Components

Component - the reusable part of the template: cap, product card, footer.

## Decisions

- The term is called “component.”
- The component is a `.shugo` file, like page and layout.
- The component parameters are read in its code, as in Astro.
- The component can upload its data.
- The calling pattern can transfer the markup inside the component: slots are needed.
- Component call is the block boundary: the block is stored and updated separately. Details are in [Blocks and point update](blocks.md).
- The functions are connected by explicit imports.

## Made in the first version, waiting for approval

### Connection and challenge

```
---
import Card from "../components/Card.shugo"
import * as ui from "@/components/ui"
---
<Card title="Pitcher" price={2400} featured />
<ui.Button href="/cart">In Cart</ui.Button>
```

The component is imported in code, as in Astro ([6](decisions.md#6-components-and-functions-are-imported-in-data-code-as-in-astro)). The tag with a capital letter or with a dot is a component. The file can lie anywhere; the `components/` folder is an agreement.

The parameters are written as attributes: `title="text"` transmits a string, `featured` transmits `true`, `price={2400}` transmits an expression value, and `{...props}` transmits all object fields.

### Parameters

```
---
interface Props {
  title: string
  price: number
  featured?: boolean
}
const { title, price, featured = false } = Shugo.props as Props
---
<a class={featured ? "card featured" : "card"}>{title}: {price}</a>
```

### Slots

```
<!-- components/Card.shugo -->
<section class="card">
  <g-slot> is shown if nothing was passed inside</g-slot>
  <g-if cond={Shugo.slots.has("footer")}>
    <footer><g-slot name="footer" /></footer>
  </g-if>
</section>

<!-- usage -->
<Card>
  <p>in regular slot</p>
  <small slot="footer">v Footer</small> Slot
</Card>
```

- `<g-slot />` shows what is inside the component tag. `<g-slot name="footer" />` is what is labeled `slot="footer"`.
- `slot="name"` is put on a tag that stands right inside the component. To transfer multiple nodes without your tag, there is `<g-fragment slot="name">`.
- The contents of `<g-slot>…</g-slot>` are shown if nothing is transferred to the slot.
- The contents of a slot are counted in the template that wrote it: it sees the variables of that template, not the component.
- The component can pass its slot on: `<Inner><g-slot /></Inner>`.

Recording options and reasons for choosing [8](decisions.md#8-slots).

### Your data

```
---
import { getMenu } from "@/lib/cms"
const menu = await getMenu()
---
<nav><g-for each={menu} as="item"><a href={item.href}>{item.title}</a></g-for></nav>
```

The component code is executed every time it is displayed. For a component that is shown on 100,000 pages, it is 100,000 times — unless it is shared.

### Common components

The component is calculated and stored once if it is independent of the page and called with simple parameters ([3](decisions.md#3-which-blocks-are-shared)):

```
<Header />                      <!-- once for the entire site -->
<ProductCard id={product.id} /> <!-- once per product -->
<ProductCard item={product} />  <!-- once per page because the prop is an object -->
```

The component depends on the page if it itself or something imported by it reads `Shugo.url` or `Shugo.params`.

Hence the reception: it is better to give the card the number of goods, and she will find the goods herself. Then the card is considered once, and when changing the goods, one record changes, no matter how many pages it shows.

### Challenge yourself

```
<!-- components/Tree.shugo -->
---
const { node } = Shugo.props
---
<li>{node.title}
  <g-if cond={node.children.length}>
    <ul><g-for each={node.children} as="child"><Shugo.self node={child} /></g-for></ul>
  </g-if>
</li>
```

### Component of variable

```
---
import Grid from "./Grid.shugo"
import List from "./List.shugo"
const View = Shugo.props.compact ? List : Grid
---
<View items={items} />
```

## Open questions

- Styles that only work inside their component.
- Checking the types of parameters during assembly.
- Memory between calls for non-essential components: the object card in the parameter is now counted on each page.
- Warning when common blocks of one component become too many (`<Price value={n} />`).
