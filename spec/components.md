# Components

Component - the reusable part of the template: cap, product card, footer.

The name of the concept is not yet selected: "components" or "partials". The name of this document also depends on it.

## Decisions

- The component is a `.shugo` file, like page and layout.
- The components are connected in the template itself, not in the data code.
- The component parameters are not declared in JS, but structurally: a root element like `<partial props="…">` or system tags at the beginning of the file.

## From the prototype, pending approval

- Component call is a block boundary: such a block is cached and updated separately from the page. Details are in [Pointed Page Update](incremental-updates.md).
- The component cannot call itself either directly or through other components.

## Open questions

- Name: Components or parts.
- As a component connects: by itself by file name or explicit tag.
- As declared parameters and their types. In the prototype, the parameter was one.
- Content that the calling pattern transfers inside a component (slots).
- Can the component load its data?
