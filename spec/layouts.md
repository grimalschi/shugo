# Layouts

Layout — a general binding of pages: a cap, footer and place for the content of the page. The document itself — `<html>`, `<head>` and `<body>` — is written once, in the shell of `src/site.html`.

## Decisions

- The layout is a `.svelte` file.
- Layouts are modeled after `+layout` from SvelteKit: the layout file is located in the folder with the pages and acts on them.
- The layout works as a component, but it cannot be called.
- The model can upload its data.
- Data from page to page, how `load` the City of London `+layout.ts` SvelteKit, no: page asks the base itself (solution 22 November 2026 years).
- The layout is kept separate from the pages: changing what it reads does not compute the pages, and changing the page does not compute the layout (decision 22 November 2026 A layout that does not read the address is one entry for the entire site; a layout that reads the address is its own entry for each page.
- The layout is written as a component Svelte 5 (Decision of the author 21 November 2026 Year: Page he gets snippet `children` and shows her `{@render children()}`How `+layout.svelte` the City of London SvelteKit; title and meta tags page writes in `<svelte:head>`. It replaces the old `<g-slot />` and `<title slot="head">`.
- `<html>`, `<head>` and `<body>` in a separate file `src/site.html` with places `%shugo.head%` and `%shugo.body%`How `src/app.html` the City of London SvelteKit; root layout - a common component (solution) 22 November 2026 years).

## Implemented, pending approval

### Where it is and what it works for

```
src/pages/
  +layout.svelte wraps all pages
  index.svelte
  docs/
    +layout.svelte wraps docs/ pages, itself lying inside the root
    index.svelte
    (guides)/
      +layout.svelte folder in parentheses: another layout without part of the address
      start.svelte address /docs/start
```

The nested layout is inside the external one, as in SvelteKit. The `/docs/start` page is wrapped with three layouts.

### Shell: `src/site.html`

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    %shugo.head%
  </head>
  <body>
    %shugo.body%
  </body>
</html>
```

- `%shugo.body%` is a page in its layouts.
- `%shugo.head%` is what the page templates gave in `<svelte:head>`, followed by a link to the site style file.
- Each place is in the file exactly once, the head - before the body; otherwise, the assembly will stop with an error.
- The shell is a regular HTML, not a template: there are no expressions and code in it, it is copied as written. Page language, `<html lang="en">`, is here, one on the site.
- No file - takes the shell of the framework, the same as above, but without `lang`.
- A layout, page, or component that writes `<html>`, `<head>`, `<body>`, or `<!doctype>` is an assembly error: it is written in the shell.

### How the layout shows the page

```svelte
<!-- src/pages/+layout.svelte -->
<script>
  import Header from "../components/Header.svelte"
  import "../styles/main.css"

  let { children } = $props()
</script>

<Header />
<main>{@render children()}</main>
```

`{@render children()}` is the place of the page. The layout that does not show it is an assembly error. The root layout starts immediately with what is in `<body>`, like `+layout.svelte` in SvelteKit, and is transferred between them without editing.

### How a page conveys title and meta tags

```svelte
<!-- src/pages/about.svelte -->
<svelte:head>
  <title>About the store</title>
  <meta name="description" content="Pottery workshop">
</svelte:head>

<h1>About the store</h1>
```

- `<svelte:head>` stands at the top level of the component, as required by Svelte: pages, layout or any other component.
- Its contents fall into the place of `%shugo.head%` shell.
- Order is like in a Svelte document: first external layouts, then internal layouts, then pages.
- `<title>` has one page like in Svelte: last in this order, and it stands after the rest. Therefore, the default title is written in `<svelte:head>` root layout, and the page replaces it with its own:

  ```svelte
  <!-- src/pages/+layout.svelte -->
  <svelte:head><title>Measure</title></svelte:head>
  ```

### What does framework add

In place of `%shugo.head%`, after the templates have been given, a link to the site style file is added. In the post-page development mode, in place of `%shugo.body%`, a script is added that restarts the page after updating.

### The layout is kept separate from the pages

Page comes to the layout with a snippet `children` and is stored in its record, and the layout is in its:

- a layout that does not take `page` from the `shugo` module - neither itself nor through the modules that it imports - one entry per entire site; the menu has changed - one entry is calculated;
- a layout that reads the address - for example, highlights the current menu item, `class:active={page.url.pathname === item.href}` with `import { page } from "shugo"` - its own entry for each page; the menu has changed - the layout for each page is calculated, and the pages themselves are not calculated;
- the page has changed it is calculated alone, without a layout.

Therefore, the layout is rendered separately from the page: the context it sets through `setContext` is seen by its components, but not by the page inside it. What the page needs, it takes through `query` or from the module in `src/lib/`.

How it works in [Blocks](../docs/blocks.md).

## Open questions

- Reset chain: page or section without external layouts. In SvelteKit it is `+page@.svelte` and `+layout@.svelte`.
- Language of the page for the site in several languages: the shell will need space for `lang` - it will give [internationalization](../docs/proposals/i18n.md).
