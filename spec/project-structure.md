# Project structure

The document describes what folders and files are in the project on Shugo and what they mean.

## Decisions

- Pages, layouts and components are `.shugo` files. The rest of the project code is TypeScript.
- Project settings are the `shugo.json` file at the root.
- Everything that framework writes is stored in the `_build` folder.

## Implemented, pending approval

The reasons are in [18](decisions.md#18-project-structure).

```
shugo.json settings
package.json dependencies from npm if they are needed
pages/ pages and layouts; the location of the file specifies the address
components/ components
lib/ general code on TypeScript: data, functions
styles/ styles
public/ files that are given as is: /robots.txt, /img/logo.svg
_build/ result of the framework; git is not included
```

- Only the `pages/` folder is required.
- `components/`, `lib/` and `styles/` are agreements: framework finds files by import, not by folder names.
- `.shugo` files are searched throughout the project except for `_build`, `public`, `node_modules` and hidden folders.
- `@/` in import means project root: `import { price } from "@/lib/shop"`.

### `shugo.json`

```json
{
  "site": "https://example.com",
  "runtime": "bun",
  "redirects": { "/old": "/new" }
}
```

| Field | What does it mean? |
| --- | --- |
| `site` | address at which the site is published; it is built `Shugo.url` |
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

- `src` folder: is it needed as in Astro.
- `tsconfig.json` and description of `Shugo` types for editors.
- Settings in `shugo.json` that are not available yet: languages, pictures, build folder.
