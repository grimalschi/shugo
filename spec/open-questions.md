# Open questions

Questions are numbered: answer the numbers. Where there is advice, yes - to accept it. Decisions taken without confirmation are collected in [Decisions awaiting approval](decisions.md); proposals that do not yet exist are in [docs/proposals](../docs/proposals/README.md).

### 1. Types for editor

The editor does not know the module `shugo`: `import { query, page, search } from "shugo"` He points out as a mistake. 22 November 2026 Year Svelte sewn in Shugo, package `svelte` No project, and it is also highlighted. `import type { Snippet } from "svelte"`. Same types needed `shugo check`He will be `svelte-check` S TypeScriptsewn into Shugo.

- a) types write assembly like SvelteKit in `.svelte-kit/`: `_build/types/shugo.d.ts`, ads sewn by Svelte and `_build/tsconfig.json` - where to look for them and what `@/` means. `tsconfig.json` is one line that does not change: `{ "extends": "./_build/tsconfig.json" }`; it is written by `shugo init`. Types of always that version of Shugo that the site assembles; until the first build, the editor does not see them;
- b) npm package with the same types, which are put like any package. Works before the first build, but the version of the package needs to be updated together with Shugo, and the assembly - to check that they coincide;
- c) `shugo init` puts a file with types in the project once. The easiest way, but after updating Shugo types become obsolete silently.

**The Council:** a).

### 2. `<svelte:boundary>` on static page

`<svelte:boundary>` is made for the browser: it catches an error inside itself and shows instead of content the snippet `failed`, and while the content is waiting for data, the snippet `pending` shows. There is no browser part on the static page, and both snippets do not do what the author who knows Svelte expects. 22 November 2026 year:

- with the `failed` snippet, the error is not caught inside: the assembly stops with this error. The error is visible immediately, this is good, but `failed` will never appear, and the assembly is silent about it;
- with Snippet `pending` Svelte On the server, the render is not content, but a waiting branch: the page forever shows “Loading...”, and there is no content in it. The assembly takes place without a single word this is a quietly broken page.

There is nothing to wait on the page now: `await` is prohibited.

- a) `<svelte:boundary>` outside the islands — a bug with a hint like `await`: on a static page it has nothing to catch and nothing to wait. In the island it works;
- b) leave and warn when assembling: about `failed` - that it will not appear, about `pending` - that the page will show only it;
- c) catch an error and show `failed`, with build warning; `pending` is an error.

**The Council:** a): One entry that does nothing on the page is less error for the agent.

### 3. Reading past the database: files in the page code

The network code pages is now not allowed: `await` is prohibited. But the file can be read immediately, without waiting: `readFileSync` from `node:fs`. This update does not track the file, and the page remains the same until it is calculated by `shugo build` or finds `shugo update -check`. The file that is imported, `import posts from "../../data/posts.json"`, is part of the code, and changing it reassembles the project; it's not about it.

- (a) Imports `node:fs` and neighboring modules that read past the base - `fs`, `child_process`, `net`, `http`, in the site code build error, intentionally, as `await`; in the package - a warning with the name of the package: many packages take `fs` for what pages are not needed;
- b) only warnings, both in the site code and in packages;
- c) leave as is: this is found by `shugo update -check`.

**The Council:** a): This shows where the site wants to take data past the database, and this place will be a task for the data adapter.
