# Project structure

The document describes what folders and files are in the project on Shugo and what they mean.

## Decisions

- Pages, layouts and common components are `.svelte` files. The entry points for hydration are indicated by the suffix `.island.svelte`. The rest of the project code is TypeScript.
- Project settings are the `shugo.json` file at the root.
- Everything that framework writes is stored in the `_build` folder.
- The site code is in the folder `src/` (the decision of the author of the project 21 November 2026 year).
- The document around the pages — `<html>`, `<head>`, `<body>` — is written in `src/site.html` (22 decision of November 2026 year), see [Layouts](layouts.md#site-shell).
- `sitemap.xml`, `robots.txt` and other files not in HTML write `.ts` modules among pages (22 solution November 2026 years), see [Routing](routing.md#files-not-in-html).
- Svelte sewn in Shugo One version - compiler, server and browser parts: the project does not need to put a package `svelte`neither for pages nor for islands (decision by the author of the draft 22 November 2026 years). *Changed by the decision of the author of the project 23 November 2026 year: see the following item.*
- Svelte, `svelte-check` and TypeScript are the relationship of the `shugo` package to the exact versions; of them `shugo` is not sewn. The project still does not place them: they come from `bun add shugo` (the decision of the sponsor on 23 November 2026).
- `.svelte` components from npm packages are not compiled yet: their import is an assembly error with a prompt to copy the component to `src/components/`. Compile them with Svelte sewn when component libraries are needed (22 project author’s decision of November 2026 year). *This is about the pages: the island is compiled entirely by the sewn Svelte, and the package component works in it now (checked 22 November 2026 year).*
- Shugo itself is put into the project by npm, `bun add shugo` package, and the `shugo` module types come into the project with it (the decision of the author of the 23 project is November 2026 year), see [Command line](cli.md#the-shugo-npm-package).
- npm packages are put by the project package manager, `bun add` or `npm install`, in `node_modules`, as in any project on JavaScript; Shugo only reads them. The `svelte` package from `node_modules` is never taken. The package that does not exist, and the package that needs another version of Svelte, are build errors with a hint (the decision of the author of the project 22 November 2026 year).
- Packages that import the page code are collected with it in `_build/eval.mjs`As in the islands: Svelte The one who puts the package in `shugo`each file in the build is a source with a hash, the missing package is an assembly error at the point of import, the package with the native code is a prompt error (project author's decision) 22 November 2026 years).

## Implemented, pending approval

```
shugo.json settings
package.json dependencies from npm: shugo and what imports the code
tsconfig.json    { "extends": "shugo/tsconfig.json" }
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
- Svelte is not necessary to put: Svelte 5.59.1 comes into the project by the dependence of the package `shugo`, together with `svelte-check` and TypeScript for `shugo check`. The assembly takes Svelte only from there: importing `svelte` and `svelte/…` both in the project code and in the packages leads into it, even if the `node_modules` is another Svelte. Svelte another version in its place - an error: Shugo is made for one.
- The npm package is placed by `bun add date-fns` or `npm install date-fns` and imported by name. What the assembly can't take, she says at the import site: the package is not – `date-fns is not installed: bun add date-fns`; the command is on the project blocking file; the package needs another version of Svelte on its `peerDependencies` or `dependencies` – an error with both versions; the package does not bring JavaScript, such as the `.node` home code – an error with a clue to take the package into one JavaScript. The package that reads the file next to its code doesn't find it -- the package code now in `_build/` -- and falls when you compute with a clue.
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

Packages from npm that import page code are collected in `eval.mjs` with it: when calculating `node_modules` is not readable. Each package file included in the build is a source with a hash, like a project file: after updating the package, `shugo update` collects the project again. `package.json` and packet manager lock files — `bun.lock`, `package-lock.json`, `pnpm-lock.yaml`, `yarn.lock` — are also sources: they can be seen updating what is taken from `node_modules` past the build, for example Tailwind. The server part of Svelte is going to be the same — from the package `svelte`, which puts the package `shugo`.

### Switching from `.shugo`

Rename `.shugo` templates to `.svelte` and fix the import paths: the build finds both old files and old imports and tells you how to name them. In TypeScript components, add `lang="ts"`. Types there erases Svelte, so type import via `import type`, and instead of `enum` write object `as const`. Then run `shugo build`: the repository of the previous version of Shugo is not readable, and the site is re-calculated.

## Open questions

- `shugo init` is a new project with an example, `src/site.html`, `package.json` with a package of `shugo`, `tsconfig.json` in one line and a memo for the agent: it is decided to do (the decision of 21 on November 2026 years), until done.
- `$lib` from SvelteKit is on par with `@/`.
- Tailwind is a project package. Whether to embed it in Shugo, like Svelte, or leave it as a package.
- Whether the `shugo add` team needs a shortcut to the project package manager or the agent has enough memo.
- Settings in `shugo.json` that are not available yet: languages, pictures, build folder, site address in the subfolder (`base`, it is decided to do).
