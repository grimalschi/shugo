# File-based routing

The page addresses are specified by the location and file names in `src/pages/`.

## Decisions

- Routing is built on files, with modifiers in names.
- Files with `+` at the beginning of the name are service files, modeled on SvelteKit and PocketPages. There are no other files except `+layout` yet (21 decision of November 2026 year).
- When one address gives a page with an exact name and a page with a parameter, the exact name wins.
- The page is written in the same way as SvelteKit writes pages under the prerender: `/about` - `about.html`, `/` - `index.html` (21 solution November 2026 year). Details - in [Rantime](../docs/runtime.md).
- The site can be published in the subfolder of the domain: `base` setting in `shugo.json` (21 solution November 2026 year). Not done yet.
- The official name of the layout remains `+layout.shugo` (22 solution of November 2026 year).
- Files not in HTML — `sitemap.xml`, `robots.txt`, RSS — writes a module on TypeScript among the pages: `src/pages/sitemap.xml.ts` with the function `GET`, which returns the text of the file. What he read from the database is tracked like a page; in the export folder it is written with the file `sitemap.xml` (the decision of 22 November 2026 year).

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

- Page — any `.shugo` file in `src/pages/` whose name does not start with `_` or `+`. The `.ts` module or `.js` there also gives a file not in HTML, see below.
- Names with `+` are occupied by the framework. Now there is only `+layout.shugo`; another name with `+` is an assembly error.
- Part of the path is either text or one parameter. `post-[id].shugo` is an assembly error.
- `[...path]` can only stand last.
- The page with the parameter is required to export `getStaticPaths` from `<script module>`, see [Dynamic pages](dynamic-pages.md).
- Two files with the same address type (`a.shugo` and `a/index.shugo`, `[id].shugo` and `[slug].shugo`) are build errors.
- If the address is given two different routes, it gets the one that calls the address more accurately: the text is stronger than the parameter, the parameter is stronger than the remainder of the path. In the report, such addresses are considered separately.
- The page has one address, without a slash at the end, as default in SvelteKit. On `/about/`, the server responds by redirecting to `/about`.
- Redirects: `"redirects": {"/old": "/new"}` to `shugo.json`. Page at this address is stronger than redirect.

### Files not in HTML

The `.ts` or `.js` module in `src/pages/` gives the site file type which is in the name before `.ts`:

| File | Address |
| --- | --- |
| `src/pages/sitemap.xml.ts` | `/sitemap.xml` |
| `src/pages/robots.txt.ts` | `/robots.txt` |
| `src/pages/blog/rss.xml.ts` | `/blog/rss.xml` |
| `src/pages/api/[slug].json.ts` | `/api/jug.json`, `/api/cup.json`: parameter as in page |
| `src/pages/_feed.ts` | not file: code that others import |

```ts
// src/pages/sitemap.xml.ts
import { query, page } from "shugo"

export function GET() {
  const products = query("products").select("slug").order("slug").all()
  const urls = products.map((p) => `<url><loc>${new URL("/products/" + p.slug, page.url)}</loc></url>`)
  return `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">${urls.join("")}</urlset>`
}
```

- `GET` receives nothing and returns the file text -- the string. You cannot wait, as on the page: `async` and `Response` -- a collection error with a clue. Data are from `query`, an address from `page` of the `shugo` module: `page.url` -- the complete address of the `shugo.json` file.
- What `GET` has read from the database is tracked like a page: the file is calculated again when the lines it has read are changed. The site map above reads the addresses of goods, so the new price does not calculate it, and the new product calculates.
- The file with the option exports `getStaticPaths` as a page.
- The text gets into the file as it is: without shell, layouts and shielding.
- The server responds with an extension type, as for files from `public/`: `.xml` — `text/xml`, `.json` — `application/json`, `.txt` — `text/plain`. Export writes the file under its address: `sitemap.xml`, `api/jug.json`.
- The module without the type in the name, `data.ts`, is an assembly error: the code that is imported is put into `src/lib/` or called with `_`.
- There are no layouts in the file.

## Open questions

- The optional parameter `[[lang]]` is in [proposal for internationalization](../docs/proposals/i18n.md).
