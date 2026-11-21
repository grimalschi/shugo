# Styles

The document describes working with styles: Tailwind and the usual CSS.

## Decisions

- Styles are collected when the project is compiled. At this stage, you can run Tailwind and other utilities.
- Tailwind finds classes by viewing project files as text.

## Implemented, pending approval

The reasons are in [15](decisions.md#15-styles).

### Connection

```
<!-- pages/+layout.shugo -->
---
import "../styles/main.css"
---
```

The style file is connected by importing in the code, as in Astro. You can connect it in any `.shugo` file or module; this is usually a root layout.

### One stylesheet per site

All the connected files and contents of all `<style>` tags are collected into one `_build/assets/styles.<hash>.css` file. The order is import order. A link to the file is added before `</head>` each page. The file name changes along with the content, so the browser caches it forever.

Fonts and images that styles link to via `url(…)` are copied alongside.

### Tailwind

If styles have Tailwind directives (`@import "tailwindcss"`, `@apply`, `@theme`), the file collects Tailwind from the `node_modules` project:

```
npm install tailwindcss @tailwindcss/cli
```

```css
/* styles/main.css */
@import "tailwindcss";
```

Classes are in both markup and code lines: `` class={`rounded-lg ${look}`} ``. An example is [examples/tailwind](../examples/tailwind).

### `<style>` in template

```
<a class="card">…</a>
<style>
  .card { border: 1px solid #ddd; }
</style>
```

The contents of `<style>` go into the general file and act on the entire site. `<style is:inline>` remains in the page as written.

## Open questions

- Styles that only work inside their component. In Astro, each component tag is added with a hash attribute, and selectors are rewritten. Agents who know Astro expect exactly this behavior from `<style>`.
- Multiple style files: a separate file for the site partition.
- Critical styles are right on the page.
