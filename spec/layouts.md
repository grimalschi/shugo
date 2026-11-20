# Layouts

Layout - general binding of pages: `<html>`, `<head>`, cap, footer and place for the page content.

## Decisions

- The layout is a `.shugo` file.
- Layouts are modeled after `+layout` from SvelteKit: the layout file is located in the folder with the pages and acts on them.
- The layout works as a component, but it cannot be called.
- The model can upload its data.

## Made in the first version, waiting for approval

The reasons are in [9](decisions.md#9-layouts).

### Where it is and what it works for

```
pages/
  +layout.shugo wraps all pages
  index.shugo
  docs/
    +layout.shugo wraps docs/ pages, itself lying inside the root
    index.shugo
    (guides)/
      +layout.shugo folder in parentheses: another layout without part of the address
      start.shugo address /docs/start
```

The nested layout is inside the external one. The `/docs/start` page is wrapped in three layouts.

### How the layout shows the page

```
<!-- pages/+layout.shugo -->
---
import Header from "../components/Header.shugo"
import "../styles/main.css"
---
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <g-slot name="head"><title>Shop</title></g-slot>
  </head>
  <body>
    <Header />
    <main><g-slot /></main>
  </body>
</html>
```

`<g-slot />` — place of the page. Layout without it — build error.

### How a page conveys title and meta tags

```
<!-- pages/about.shugo -->
<title slot="head">O Shop</title>
<meta slot="head" name="description" content="Pottery workshop">

<h1>O Shop</h1>
```

The tag with `slot="head"` at the top of the page goes to the layout. It gets the nearest layout, which has `<g-slot name="head" />`. The name `head` is not special: the slot can be called anything.

- If the page has not passed anything, the layout shows the contents of its `<g-slot name="head">…</g-slot>`.
- If a page has passed a markup for a slot that is not in any layout around it, it is an build error.
- The nested layout can also transmit the markup in the same way.

### What does framework add

Before `</head>`, a link to the site style file is added. In development mode, before `</body>`, a script is added that reboots the page after reassembly.

## Open questions

- Reset chain: page or partition without external layouts. In SvelteKit it is `+page@.svelte`, in Next.js - groups with separate root layouts.
- Data from layout to page. Now the general data is taken by import from one module.
- General layout: the layout without referring to `Shugo.url` is considered once per set of slots. The layout that reads the page address executes its code on each page. Should I warn if such a layout loads data itself.
