# Layouts

Layout - general binding of pages: `<html>`, `<head>`, cap, footer and place for the page content.

## Decisions

- The layout is a `.shugo` file.
- Layouts are modeled after `+layout` from SvelteKit: the layout file is located in the folder with the pages and acts on them.
- The layout works as a component, but it cannot be called.
- The model can upload its data.
- Data from page to page, how `load` the City of London `+layout.ts` SvelteKit, no: page asks the base itself (solution 22 November 2026 years).
- The layout is kept separate from the pages: changing what it reads does not compute the pages, and changing the page does not compute the layout (decision 22 November 2026 A layout that does not read the address is one entry for the entire site; a layout that reads the address is its own entry for each page.
- The layout is written as a component Svelte 5 (Decision of the author 21 November 2026 Year: Page he gets snippet `children` and shows her `{@render children()}`How `+layout.svelte` the City of London SvelteKit; title and meta tags page writes in `<svelte:head>`. It replaces the old `<g-slot />` and `<title slot="head">`.

## Implemented, pending approval

### Where it is and what it works for

```
src/pages/
  +layout.shugo wraps all pages
  index.shugo
  docs/
    +layout.shugo wraps docs/ pages, itself lying inside the root
    index.shugo
    (guides)/
      +layout.shugo folder in parentheses: another layout without part of the address
      start.shugo address /docs/start
```

The nested layout is inside the external one, as in SvelteKit. The `/docs/start` page is wrapped with three layouts.

### How the layout shows the page

```svelte
<!-- src/pages/+layout.shugo -->
<script>
  import Header from "../components/Header.shugo"
  import "../styles/main.css"

  let { children } = $props()
</script>

<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
  </head>
  <body>
    <Header />
    <main>{@render children()}</main>
  </body>
</html>
```

`{@render children()}` - the place of the page. The layout that does not show it is an assembly error.

### How a page conveys title and meta tags

```svelte
<!-- src/pages/about.shugo -->
<svelte:head>
  <title>About the store</title>
  <meta name="description" content="Pottery workshop">
</svelte:head>

<h1>About the store</h1>
```

- `<svelte:head>` stands at the top of the page or layout as required by Svelte.
- Its content gets to the end of the `<head>` page: the one that writes the nearest layout or the page itself. The layout that `<head>` writes may itself have `<svelte:head>`.
- Order is like in a Svelte document: first external layouts, then internal layouts, then pages.
- `<title>` has one page like in Svelte: last in this order, and it stands after the rest. Therefore, the default title is written in `<svelte:head>` root layout, and the page replaces it with its own:

  ```svelte
  <!-- src/pages/+layout.shugo -->
  <svelte:head><title>Measure</title></svelte:head>
  ```

- If `<head>` does not write any page, no layout around it, `<svelte:head>` is an build error.

### What does framework add

Before `</head>`, a link to the site style file is added. In development mode, before `</body>`, a script is added that reboots the page after updating.

### The layout is kept separate from the pages

Page comes to the layout with a snippet `children` and is stored in its record, and the layout is in its:

- a layout that does not read `Shugo.url` and `Shugo.params` - one entry per entire site; changed menu - one entry is calculated;
- layout that reads the address - for example, highlights the current menu item, `class:active={Shugo.url.pathname === item.href}` - its own entry for each page; changed menu - the layout is calculated for each page, and the pages themselves are not calculated;
- the page has changed it is calculated alone, without a layout.

How it works in [Blocks](../docs/blocks.md).

## Open questions

- Reset chain: page or section without external layouts. In SvelteKit it is `+page@.svelte` and `+layout@.svelte`.
- Page shell: `<html>`, `<head>` and `<body>` in the root layout as now, or in a separate file like `src/app.html` in SvelteKit - [proposal](../docs/proposals/app-html.md).
