# File-based routing

The page addresses are given by the location and file names.

## Decisions

- Routing is built on files, with modifiers in names.

## From the prototype, pending approval

So far, nothing: there was one page in the prototype.

## Open questions

- What modifiers are needed: parameter in the name, path remainder, group folders without influence on addresses, hidden files, language, pages not in HTML (`sitemap.xml`, `robots.txt`).
- How each modifier is written.
- `index` files and the final slash in addresses.
- Page 404 and redirects.
- What to do when two files give one address
