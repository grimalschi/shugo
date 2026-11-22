# Styles

The document describes how to work with styles: component styles, common files CSS and Tailwind.

## Decisions

- Styles are collected when the project is compiled. At this stage, you can run Tailwind and other utilities.
- Tailwind finds classes by viewing project files as text.
- All styles of the site are collected in one file with a hash in the name, a link to it is before `</head>`; Tailwind connects if it is installed in the project (21 solution November 2026 year).
- `<style>` component acts only on this component, as in Svelte (21 solution November 2026 year).
- Multiple style files and critical styles right in the page postponed (21 November 2026 solution).

## Implemented, pending approval

### Component Styles

```svelte
<!-- src/components/Card.shugo -->
<a class="card">…</a>

<style>
  .card { border: 1px solid #ddd; }
  a:hover { color: red; }
  :global(body.dark) .card { border-color: #444; }
</style>
```

As in Svelte:

- Each component element receives a `svelte-…` class from the file path hash, and each selector requires this class. `.card` will not touch another component of `.card` from this file.
- `:global(…)` and the `:global { … }` block are valid for the entire site.
- The `@keyframes` names get the component prefix; the `-global-` at the beginning of the name leaves it as is.
- The component does not stylize the elements that came into it with a snippet: they belong to the template that wrote them.

One difference: Svelte puts the class only on elements that some selector chooses, and throws selectors that do not choose anything. Shugo puts the class on all elements of the component and leaves all selectors. The page looks the same.

`<style>` inside the element or in `<svelte:head>` remains in the page as written, as in Svelte.

### General Style Files

```svelte
<!-- src/pages/+layout.shugo -->
<script>
  import "../styles/main.css"
</script>
```

The style file is connected by import to `<script>`. You can connect it in any `.shugo` file or module; this is usually the root layout.

### One stylesheet per site

All connected files and styles of all components are collected into one `_build/assets/styles.<hash>.css` file. Order of Imports. A link to the file is added to the `<head>` of each page, in place of the `%shugo.head%` shell of the `src/site.html`, after having given the templates. The file name changes with the content, so the browser caches it forever.

Fonts and images that styles link to via `url(…)` are copied alongside.

### Tailwind

If styles have Tailwind directives (`@import "tailwindcss"`, `@apply`, `@theme`), the file collects Tailwind from the `node_modules` project:

```
npm install tailwindcss @tailwindcss/cli
```

```css
/* src/styles/main.css */
@import "tailwindcss";
```

Classes are in both markup and code lines: `` class={`rounded-lg ${look}`} ``. An example is [examples/tailwind](../examples/tailwind).

## Open questions

- Whether to put the component class only on selected elements and throw out extra selectors, like Svelte. Delayed 22 November 2026 years; how to do it - in [the Compatibility Proposal](../docs/proposals/svelte-compatibility.md).
