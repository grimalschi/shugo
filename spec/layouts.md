# Layouts

A layout wraps pages in shared markup such as a header and footer. Put the document itself, including `<html>`, `<head>`, and `<body>`, in the site shell at `src/site.html`.

## Decisions

- Use `.svelte` files for layouts.
- Layouts follow the SvelteKit `+layout` convention. A layout file wraps the pages in its directory.
- Shugo invokes layouts automatically. They behave like components and can load their own data.
- Importing `+layout.svelte` is a build error. Put markup that templates import in a component (November 22, 2026).
- Each page queries the database for its own data. Layouts have no equivalent of the `load` export in SvelteKit's `+layout.ts` (November 22, 2026).
- Shugo stores layouts and pages separately. Changes to a layout's data recompute the layout, and changes to a page recompute the page. A layout that depends on the URL has one record per page. Other layouts have one record for the site (November 22, 2026).
- A layout is a Svelte 5 component. It receives the page as a `children` snippet and renders it with `{@render children()}`. Pages contribute titles and metadata through `<svelte:head>`. This replaces `<g-slot />` and `<title slot="head">` (November 21, 2026).
- `src/site.html` holds `<html>`, `<head>`, and `<body>`, with placeholders `%shugo.head%` and `%shugo.body%`. It serves the same role as `src/app.html` in SvelteKit. The root layout is a regular component (November 22, 2026).

## Implemented, pending approval

### Layout location and scope

```text
src/pages/
  +layout.svelte            wraps every page
  index.svelte
  docs/
    +layout.svelte          wraps pages in docs/, inside the root layout
    index.svelte
    (guides)/
      +layout.svelte        adds a layout without adding a URL segment
      start.svelte          served at /docs/start
```

Nested layouts render inside their parents, as in SvelteKit. Three layouts wrap `/docs/start`.

### Site shell

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

- `%shugo.body%` inserts the page wrapped in its layouts.
- `%shugo.head%` inserts the templates' `<svelte:head>` contributions, followed by a stylesheet link.
- Include each placeholder exactly once, with the head before the body. The build rejects missing, duplicated, or out-of-order placeholders.
- The shell is plain HTML copied verbatim. Its `lang` attribute sets the language for the whole site.
- If `src/site.html` is missing, Shugo uses the shell above without the `lang` attribute.
- Put `<html>`, `<head>`, `<body>`, and `<!doctype>` in the shell. The build rejects them in layouts, pages, and other components.

### Rendering the page

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

`{@render children()}` marks where the page appears. Every layout must render its children. Like a SvelteKit root layout, it starts with content that belongs inside `<body>` and can be moved between the frameworks as written.

### Page titles and metadata

```svelte
<!-- src/pages/about.svelte -->
<svelte:head>
  <title>About the Shop</title>
  <meta name="description" content="A pottery studio">
</svelte:head>

<h1>About the Shop</h1>
```

- Put `<svelte:head>` at the top level of a page, layout, or other component. Svelte rejects it inside elements and blocks.
- Its content goes in the shell at `%shugo.head%`.
- A component's head content comes before that of the components it renders, regardless of where its own `<svelte:head>` appears. This places outer layouts before inner layouts and pages.
- Shugo keeps the last `<title>` in render order and puts it after the other head contributions. Set a default in the root layout and override it in individual pages.

  ```svelte
  <!-- src/pages/+layout.svelte -->
  <svelte:head><title>Measure</title></svelte:head>
  ```

### Framework additions

Shugo appends the site stylesheet link to the content inserted at `%shugo.head%`. In development, it adds a script after the page at `%shugo.body%` so the browser reloads when the site updates.

### Layouts are stored separately from pages

A layout receives the page as `children`. Each has its own record.

- A layout that never imports `page` from `shugo`, directly or through another module, uses one record for the site. A menu change recomputes that record once.
- A layout that reads the URL uses a record per page. For example, `class:active={page.url.pathname === item.href}` with `import { page } from "shugo"` can highlight the current menu item. A menu change recomputes each layout record while the page records stay current.
- A change that affects only a page recomputes that page.

Shugo renders layouts separately. A layout's `setContext` values reach its own components. Nested layouts and pages have their own contexts and can get data via `query` or a module in `src/lib/`. Shugo prefixes `$props.id()` values with the layout's file number, as in `b3-s1`, to avoid collisions with IDs from the page.

See [Blocks](../docs/blocks.md).

## Open questions

- Resetting the layout chain for a page or section. SvelteKit uses `+page@.svelte` and `+layout@.svelte`.
- Setting `lang` per page on a multilingual site. The [internationalization proposal](../docs/proposals/i18n.md) describes a placeholder for it.
