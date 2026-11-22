# Project structure

The document describes what folders and files are in the project on Shugo and what they mean.

## Decisions

- Pages, layouts and components are `.shugo` files. The rest of the project code is TypeScript.
- Project settings are the `shugo.json` file at the root.
- Everything that framework writes is stored in the `_build` folder.
- The site code is in the folder `src/` (the decision of the author of the project 21 November 2026 year).

## Implemented, pending approval

```
shugo.json settings
package.json dependencies from npm if they are needed
src/
  pages/ pages and layouts; the location of the file specifies the address
  components/ components
  lib/ general code on TypeScript: functions, constants
  styles/ styles
data/ project database, if available: data/site.db
public/ files that are given as is: /robots.txt, /img/logo.svg
_build/ result of the framework; git is not included
```

- Only the `src/pages/` folder is required.
- `src/components/`, `src/lib/` and `src/styles/` are agreements: framework finds files by import, not by folder names.
- `.shugo` files are only searched in `src/`, except for `node_modules` and hidden folders. Importing a `.shugo` file from another location is an build error with a hint of where it belongs.
- `@/` in import means folder `src/`: `import { price } from "@/lib/shop"`.
- `public/`, `data/` and `shugo.json` lie at the root, as do `static/` and settings in SvelteKit and `public/` in Astro.

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
| `site` | address at which the site is published; it is built `Shugo.url` |
| `data` | project database read by `Shugo.query`; see [Data Download](data-loading.md) |
| `runtime` | than to perform JS: `bun` or `node` |
| `redirects` | redirects from old addresses |

All fields are optional. Unknown field is an error.

### `_build`

| File | What is it |
| --- | --- |
| `eval.mjs`, `eval.mjs.map` | the entire JS project for the calculation stage and source map |
| `site.json` | project program: templates and routes |
| `sources.json` | from which files the project is collected, with each hash: `shugo update` understands that the project has changed |
| `assets/` | collected Styles |
| `store.db` | storage |
| `dist/` | site in files after `shugo export` |

Packages from npm to `eval.mjs` are not sewn: they are taken from the `node_modules` project at launch.

## Open questions

- `shugo init` is a new project with an example, `tsconfig.json`, `Shugo` types for the editor and a memo for the agent: it is decided to do (21 decision of November 2026 years), has not yet been done.
- As in SvelteKit: `src/routes/` with folder per page and files `+page`, or as in Fresh: `_layout`. The pages are now files in `src/pages/`. See [question about service files](open-questions.md).
- `$lib` of SvelteKit along with `@/` is in [Svelte Compatibility Offer](../docs/proposals/svelte-compatibility.md).
- Settings in `shugo.json` that are not available yet: languages, pictures, build folder, site address in the subfolder (`base`, it is decided to do).
