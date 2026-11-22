# Open questions

Questions are numbered: answer the numbers. Where there is advice, yes - to accept it. Decisions taken without confirmation are collected in [Decisions awaiting approval](decisions.md); proposals that do not yet exist are in [docs/proposals](../docs/proposals/README.md).

### 1. Types for editor

The editor does not know the `shugo` module: on `import { query, page, search } from "shugo"` type check swears. If you put type announcements in the project once, after updating Shugo they will become obsolete.

So this problem is solved by SvelteKit: he writes types in his `.svelte-kit/` build folder every time he starts, and `tsconfig.json` of the project inherits `.svelte-kit/tsconfig.json` once. Types are always the version that assembles the site.

For Shugo, this is what it looks like. Each build writes `_build/types/shugo.d.ts` — the `shugo` module ads that Shugo builds — and `_build/tsconfig.json`: where to look for `shugo`, which means `@/`, what settings TypeScript needs for Svelte. `tsconfig.json` is one line that does not change: `{ "extends": "./_build/tsconfig.json" }`; it is written by `shugo init`, which is decided to do. Updated Shugo the next build rewrote the types.

With 22 November 2026 year Svelte sewn into Shugo whole, for pages and for islands, and the package `svelte` project is not needed. Then the editor will not find and module `svelte`: `import type { Snippet } from "svelte"` and `import { getContext } from "svelte"` it will emphasize the same as `shugo`. The Svelte assembly can be placed in `_build/types/`. The same types will be read by `shugo check`: `svelte-check` and TypeScript decided to sew in Shugo.

Another way is the `shugo` package in npm with some types. Its version would have to be updated along with Shugo, and the assembly would check that they match.

**The Council:** types write assembly in `_build/`, like SvelteKit.

### 2. `<svelte:boundary>` on static page

With the `failed` snippet, the error is not caught inside: the assembly stops with this error. With the `pending` snippet, the static page forever shows the `pending`.

- a) leave: error is visible immediately;
- b) catch the error and show `failed`, with a warning during assembly;
- c) Warn about `pending` outside the islands.

**The Council:** (a) and (c).

### 3. `await` in `<script module>`

At the top level `<script module>` `await` works: the module waits for loading, until the first page. In `<script>` this is an error, and the documents are advised to wait only in modules from `src/lib/`.

- (a) Authorize and describe;
- b is banned.

**The Council:** a).

### 4. Who puts the packages of npm

Svelte is sewn into Shugo, `svelte-check` and TypeScript decided to sew too, and the project npm itself is not needed. But the site code can import the package: Markdown parsing, working with dates, schedule in the island. Now the package is put conventional `bun add` or `npm install`, and Shugo takes it from `node_modules`. It is available in [offer](../docs/proposals/npm-packages.md).

- a) batch project manager, as everywhere: `package.json`, `node_modules`, lock file; Shugo only reads `node_modules`;
- b) Packages puts Shugo: `shugo add date-fns`, your cache and your lock file, without `node_modules`;
- c) version in import, as in Deno: `import { format } from "npm:date-fns@4.1.0"`.

**The Council:** a). The package manager on the machine already has: Bun it is built, with Node usually put npm. Agents know it, and the editor finds the types of packages in `node_modules`. Shugo adds its own: the `svelte` package from `node_modules` is never taken, the package that needs another version of Svelte, and the package that is not there, is a build error with a hint.

### 5. Packages in the page code: past or within the build

Now the packages that import the page code are not included in `_build/eval.mjs`: Bun or Node download them from `node_modules` when calculating. Therefore, the package that imports Svelte itself does not see the sewn Svelte: without the `svelte` package, the calculation in the project drops, and with it, the page has two copies of Svelte, and `getContext` in the package falls. The missing packet is only found when calculating, without file and import string. The build package update sees on `package.json` and lock files ([53](decisions.md#53-files-of-the-package-manager--source-of-the-project)).

- a) leave the packages past assembly and intercept their import of `svelte` by loading hooks of Bun and Node modules;
- b) Collect packets in `eval.mjs`, as in islands: `svelte` leads to the sewn Svelte in the same way, each package file is a source with a hash, the missing package is an assembly error with the place of import. A package with `.node` modules or files next to the code will not be included in the build: for him, an error with a hint.

**The Council:** b): One rule is one way, and packet changes are seen as accurately as project code changes.
