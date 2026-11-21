# File-based routing

The page addresses are specified by the location and file names in `src/pages/`.

## Decisions

- Routing is built on files, with modifiers in names.
- Files with `+` at the beginning of the name are service files, modeled on SvelteKit and PocketPages. There are no other files except `+layout` yet (21 decision of November 2026 year).
- When one address gives a page with an exact name and a page with a parameter, the exact name wins.
- The page is written in the same way as SvelteKit writes pages under the prerender: `/about` - `about.html`, `/` - `index.html` (21 solution November 2026 year). Details - in [Rantime](../docs/runtime.md).
- The site can be published in the subfolder of the domain: `base` setting in `shugo.json` (21 solution November 2026 year). Not done yet.

## Implemented, pending approval

| File | Address |
| --- | --- |
| `src/pages/index.shugo` | `/` |
| `src/pages/about.shugo` | `/about` |
| `src/pages/blog/index.shugo` | `/blog` |
| `src/pages/blog/[slug].shugo` | `/blog/…`: single parameter `slug` |
| `src/pages/docs/[...path].shugo` | `/docs`, `/docs/a`, `/docs/a/b`: parameter `path` — the rest of the way |
| `src/pages/(shop)/cart.shugo` | `/cart`: folder in brackets is not included |
| `src/pages/_parts/card.shugo`, `src/pages/blog/_draft.shugo` | no Page |
| `src/pages/+layout.shugo` | layout, see [Layouts](layouts.md) |
| `src/pages/404.shugo` | page for addresses that do not have a page |

Rules:

- Page — any `.shugo` file in `src/pages/` whose name does not start with `_` or `+`.
- Names with `+` are occupied by the framework. Now there is only `+layout.shugo`; another name with `+` is an assembly error.
- Part of the path is either text or one parameter. `post-[id].shugo` is an assembly error.
- `[...path]` can only stand last.
- The page with the parameter is required to export `getStaticPaths` from `<script module>`, see [Dynamic pages](dynamic-pages.md).
- Two files with the same address type (`a.shugo` and `a/index.shugo`, `[id].shugo` and `[slug].shugo`) are build errors.
- If the address is given two different routes, it gets the one that calls the address more accurately: the text is stronger than the parameter, the parameter is stronger than the remainder of the path. In the report, such addresses are considered separately.
- The page has one address, without a slash at the end, as default in SvelteKit. On `/about/`, the server responds by redirecting to `/about`.
- Redirects: `"redirects": {"/old": "/new"}` to `shugo.json`. Page at this address is stronger than redirect.

## Open questions

- Routes as in SvelteKit: folder to address and file `+page` in it (`src/routes/blog/[slug]/+page.shugo`), `+error` instead of `404.shugo`, `[[non-binding]]` parameter, `[x=matcher]`. Comparison is in [the Question of Routes](open-questions.md).
- The optional parameter `[[lang]]` is in [proposal for internationalization](../docs/proposals/i18n.md).
- Pages not in HTML: `sitemap.xml`, `robots.txt`, RSS. In SvelteKit they are written in a file on TypeScript (`+server.ts` with `prerender = true`), rather than a template: see [question](open-questions.md).
