# File-based routing

The page addresses are given by the location and file names.

## Decisions

- Routing is built on files, with modifiers in names.
- Files with `+` at the beginning of the name are service files, modeled on SvelteKit and PocketPages.
- When one address gives a page with an exact name and a page with a parameter, the exact name wins.

## Made in the first version, waiting for approval

The reasons are in [10](decisions.md#10-routes).

| File | Address |
| --- | --- |
| `pages/index.shugo` | `/` |
| `pages/about.shugo` | `/about` |
| `pages/blog/index.shugo` | `/blog` |
| `pages/blog/[slug].shugo` | `/blog/…`: single parameter `slug` |
| `pages/docs/[...path].shugo` | `/docs`, `/docs/a`, `/docs/a/b`: parameter `path` — the rest of the way |
| `pages/(shop)/cart.shugo` | `/cart`: folder in brackets is not included |
| `pages/_parts/card.shugo`, `pages/blog/_draft.shugo` | no Page |
| `pages/+layout.shugo` | layout, see [Layouts](layouts.md) |
| `pages/404.shugo` | page for addresses that do not have a page |

Rules:

- Page — any `.shugo` file in `pages/` whose name does not start with `_` or `+`.
- Names with `+` are occupied by the framework. Now there is only `+layout.shugo`; another name with `+` is an assembly error.
- Part of the path is either text or one parameter. `post-[id].shugo` is an assembly error.
- `[...path]` can only stand last.
- The page with the parameter is required to export `getStaticPaths`, see [Dynamic pages](dynamic-pages.md).
- Two files with the same address type (`a.shugo` and `a/index.shugo`, `[id].shugo` and `[slug].shugo`) are build errors.
- If the address is given two different routes, it gets the one that calls the address more accurately: the text is stronger than the parameter, the parameter is stronger than the remainder of the path. In the report, such addresses are considered separately.
- The page has one address, without a slash at the end. On `/about/`, the server responds by redirecting to `/about`.
- Redirects: `"redirects": {"/old": "/new"}` to `shugo.json`. Page at this address is stronger than redirect.

## Open questions

- Optional parameter (`[[lang]]` in SvelteKit and Nuxt).
- Pages not in HTML: `sitemap.xml`, `robots.txt`, RSS.
- Site in the subfolder of the domain.
- The choice between `/about` and `/about/`: Astro and Nuxt take both options, SvelteKit and Next.js without a slash.
- Other files with `+`: what is worth taking from PocketPages and SvelteKit (`+error`, `+config`).
