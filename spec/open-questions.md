# Open questions

Questions are numbered: answer the numbers. Where there is advice, yes - to accept it. Decisions taken without confirmation are collected in [Decisions awaiting approval](decisions.md); proposals that do not yet exist are in [docs/proposals](../docs/proposals/README.md).

### 1. `<svelte:boundary>` on static page

`<svelte:boundary>` is made for the browser: it catches an error inside itself and shows instead of content the snippet `failed`, and while the content is waiting for data, the snippet `pending` shows. There is no browser part on the static page, and both snippets do not do what the author who knows Svelte expects. 22 November 2026 year:

- with the `failed` snippet, the error is not caught inside: the assembly stops with this error. The error is visible immediately, this is good, but `failed` will never appear, and the assembly is silent about it;
- with Snippet `pending` Svelte On the server, the render is not content, but a waiting branch: the page forever shows “Loading...”, and there is no content in it. The assembly takes place without a single word this is a quietly broken page.

There is nothing to wait on the page now: `await` is prohibited.

- a) `<svelte:boundary>` outside the islands — a bug with a hint like `await`: on a static page it has nothing to catch and nothing to wait. In the island it works;
- b) leave and warn when assembling: about `failed` - that it will not appear, about `pending` - that the page will show only it;
- c) catch an error and show `failed`, with build warning; `pending` is an error.

**The Council:** a): One entry that does nothing on the page is less error for the agent.

### 2. Reading past the database: files in the page code

The network code pages is now not allowed: `await` is prohibited. But the file can be read immediately, without waiting: `readFileSync` from `node:fs`. This update does not track the file, and the page remains the same until it is calculated by `shugo build` or finds `shugo update -check`. The file that is imported, `import posts from "../../data/posts.json"`, is part of the code, and changing it reassembles the project; it's not about it.

- (a) Imports `node:fs` and neighboring modules that read past the base - `fs`, `child_process`, `net`, `http`, in the site code build error, intentionally, as `await`; in the package - a warning with the name of the package: many packages take `fs` for what pages are not needed;
- b) only warnings, both in the site code and in packages;
- c) leave as is: this is found by `shugo update -check`.

**The Council:** a): This shows where the site wants to take data past the database, and this place will be a task for the data adapter.
