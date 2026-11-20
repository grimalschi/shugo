# Layouts

Layout - general binding of pages: `<html>`, `<head>`, cap, footer and place for the page content.

## Decisions

- The layout is a `.shugo` file, as a page and component.

## From the prototype, pending approval

So far, nothing: there were no prototypes.

## Open questions

- How the layout is assigned to the page: the page chooses it itself, or it is applied to the folder, or there is a default layout.
- How the page passes to the layout header, description and other tags for `<head>`.
- nested models.
- Can the layout load its data?
- How the layout differs from the component with a place for content and whether it is different at all.
