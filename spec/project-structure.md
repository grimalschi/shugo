# Project structure

The document describes what folders and files are in the project on Shugo and what they mean.

## Decisions

- Pages, layouts and common components are `.svelte` files. The entry points for hydration are indicated by the suffix `.island.svelte`. The rest of the project code is TypeScript.
- Project settings are the `shugo.json` file at the root.
- Everything that framework writes is stored in the `_build` folder.
- The site code is in the folder `src/` (the decision of the author of the project 21 November 2026 year).
- The document around the pages — `<html>`, `<head>`, `<body>` — is written in `src/site.html` (22 decision of November 2026 year), see [Layouts](layouts.md#site-shell).
- `sitemap.xml`, `robots.txt` and other files not in HTML write `.ts` modules among pages (22 solution November 2026 years), see [Routing](routing.md#files-not-in-html).
- Svelte is sewn into Shugo by one version - the compiler, server and browser parts: the project does not need to put the `svelte` package, neither for pages nor for islands (the decision of the author of the project 22 November 2026 year).
- `.svelte` components from npm packages are not compiled yet: their import is an assembly error with a prompt to copy the component to `src/components/`. Compile them with Svelte sewn when component libraries are needed (22 project author’s decision of November 2026 year).

## Implemented, pending approval

```
shugo.json settings
package.json dependencies from npm if they are needed
src/
  site.html every page shell: <html>, <head>, <body>
  pages/ pages and layouts; the location of the file specifies the address;
                 modules like sitemap.xml.ts give files not in HTML
  components/ components
  lib/ general code on TypeScript: functions, constants
  styles/ styles
data/ project database, if available: data/site.db
public/ files that are given as is: /robots.txt, /img/logo.svg
_build/ result of the framework; git is not included
```

- Only the `src/pages/` folder is required. Without `src/site.html`, the framework shell is taken.
- Svelte is not required: Svelte 5.59.1, a compiler, server and browser parts — Shugo brings with it, for pages and islands. Imports of `svelte` and `svelte/…` in the project code always take it even if `node_modules` contains another Svelte.
- `src/components/`, `src/lib/` and `src/styles/` are agreements: framework finds files by import, not by folder names.
- `.svelte` files are only searched in `src/`, except for `node_modules` and hidden folders. Importing a `.svelte` file on the way out of `src/` is an assembly error with a hint of where it belongs. The `.svelte` component of the npm package and the package of such components are a build error with a hint: the components of the Shugo packages do not compile yet, the component is copied to `src/components/`.
- Islands can lie next to pages in `src/pages/`. The address is set by the page using the island. For ordinary components, the `_components/` folder will be suitable next to the page.
- `@/` in import means folder `src/`: `import { price } from "@/lib/shop"`.
- `public/`, `data/` and `shugo.json` lie at the root, like `static/` and the settings in SvelteKit.

### `shugo.json`

```json
{
  "site": "https://example.com",
  "data": "data/site.db",
  "runtime": "bun",
  "redirects": { "/old": "/new" }
}
```

| Field | What does it mean? |
| --- | --- |
| `site` | address at which the site is published; it is built `page.url` |
| `data` | project database read by `query`; see [Data Download](data-loading.md) |
| `runtime` | than to perform JS: `bun` or `node` |
| `redirects` | redirects from old addresses |

All fields are optional. Unknown field is an error.

### `_build`

| File | What is it |
| --- | --- |
| `eval.mjs`, `eval.mjs.map` | the entire JS project for the computation phase, together with the server part of Svelte, and the source map |
| `site.json` | project framework: files, routes, page shell, link to island styles and programs |
| `sources.json` | from which files the project is assembled, with each hash, together with `src/site.html`; frame format, version of Svelte and build revision: `shugo update` understands that the project needs to be reassembled |
| `assets/` | collected island styles and browser modules; files of previous builds are not deleted so that the server that still shows their pages finds them |
| `search-*.sqlite` | `search(…)` Search Indexes |
| `store.db` | storage |
| `dist/` | site in files after `shugo export` |

Packages from npm to `eval.mjs` are not sewn: they are taken from the `node_modules` project at launch. Therefore, `package.json` and packet manager lock files — `bun.lock`, `package-lock.json`, `pnpm-lock.yaml`, `yarn.lock` — are considered the sources of the project: after their change, `shugo update` collects the project again. The server part of Svelte is not a project package: its Shugo brings and sews into `eval.mjs`.

### Switching from `.shugo`

Rename `.shugo` templates to `.svelte` and fix the import paths: the build finds both old files and old imports and tells you how to name them. In TypeScript components, add `lang="ts"`. Types there erases Svelte, so type import via `import type`, and instead of `enum` write object `as const`. Then run `shugo build`: the repository of the previous version of Shugo is not readable, and the site is re-calculated.

## Open questions

- `shugo init` is a new project with an example, `src/site.html`, `tsconfig.json`, `shugo` module types for the editor and a memo for the agent: it is decided to do (21 solution November 2026 years), has not yet been done.
- `$lib` from SvelteKit is on par with `@/`.
- Settings in `shugo.json` that are not available yet: languages, pictures, build folder, site address in the subfolder (`base`, it is decided to do).
