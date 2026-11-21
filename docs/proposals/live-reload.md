# Live Reload: Development Mode

The proposal of how `shugo dev` shows changes in the browser: what is rebooted, when and in what order the pages are calculated. None of this has yet been done.

## Decisions

- Development mode is needed: reassembly when changing files and updating the page in the browser ([Command line](../../spec/cli.md)).
- Changing the project code calculates the entire site anew - and it should be. The words of the author of the project, 21 November 2026 year.
- For development you need Live Reload. It is done separately; while this is a proposal, without implementation. Words of the author of the project, 21 November 2026 year.

## Like now

`shugo dev` is a `shugo update` server and file tracking together:

1. Follows all project files except `_build`, `node_modules` and hidden. After the change, 80 ms is waiting: editors write the file in several receptions.
2. Does the same thing as `shugo update`. Changed source code - compiles the project and calculates all pages. Changed database - calculates what read the changed lines. Changed other files, for example in `public/` - does not calculate anything.
3. In each page before `</body>` adds a short script. It keeps a connection to the server (`EventSource` on `/_shugo/live`) and according to the word of the server does `location.reload()`.
4. After each successful update, **all** is reloaded with open pages.
5. A project error does not stop development: it is printed in the terminal, and the server shows the last successful site until the next change.

What is missing:

- **The code has changed waiting for the whole site.** The page in the browser will only reboot when all pages have been calculated. In examples, this is a fraction of a second; on the store in 100,000, the products are about 14 s, although the developer looks at one page.
- **Reload and pages that have not changed.** Changed the price of one product - all open tabs were reloaded.
- **The error is visible only in the terminal.** The browser has an old page, and it is easy not to notice that the edit did not come.
- **Changed only styles - the page is rebooted entirely**, and before that the entire site is calculated: the style file is included in the collected project, and its change is considered a code change.
- **Scrolling.** The browser usually returns it after the reboot itself, but not always: if the page changes height or loads longer.

## Proposal

### What is restarted and when

| What has changed | What is calculated | What the browser sees |
| --- | --- | --- |
| Templates and code: `.shugo`, `.ts` in `src/`, `shugo.json` | the whole site, the open pages are the first | open pages are reloaded as soon as they are calculated. |
| Data | what the changed lines read, as it is now | reload only open pages whose HTML has changed |
| Styles only: `.css` or `<style>` in template | nothing | new style file is replaced in open pages without rebooting |
| Files in `public/` | nothing | reload all open pages |

To know which pages are open, the page script calls its address when it connects: `/_shugo/live?path=/products/jug`. The list of pages with the new HTML update is already — this is `renderNames` in its report ([Command line](../../spec/cli.md)).

### The code has changed: open page first

The solution remains the same: after changing the code, the entire site is calculated, and the storage receives a new state with one transaction. Only the order changes:

1. Compilation.
2. The first to calculate the open pages. Once the page is calculated, the server shows it from memory, and the browser reboots.
3. The rest of the pages are calculated by the next, in the background.
4. If, while the background is running, the browser asks for a page that has not yet been calculated, it stands in the first queue, and the answer is waiting for it. The old code page is never shown: there are no outdated pages, there are only not yet calculated.
5. When everything is calculated, the storage is written as a whole, as in `shugo update`. The calculation of another page has fallen - the storage remains the same, and the error is visible in the browser (below).

### Styles Only: No Reset

All styles of the site are collected in one file with a hash in the name, and a link to it framework inserts each page into `<head>` when assembling it HTML. The page entries on this file know nothing. Therefore, changing some styles does not require calculating any pages: it is enough to write a new link to the site program.

Now this does not happen: the name of the project assembly is made up of the entire program, together with a reference to styles, and the new name makes you calculate everything. The proposal is to count the reference to styles outside the assembly name. Then:

- changed `.css` or only `<style>` component - the compiler sees that the code and pattern layout are the same, and updates only the link;
- open pages get the word `css` with a new address and replace `<link>` without rebooting so does Vite.

### Error in browser

The compile or compute error is shown on top of the open page: the same text as in the terminal is the space in the project file and the lines of code around. There is a good page below to see what was on it. Fixed - the message disappears, the page is reloaded. So do Vite and SvelteKit.

The page that was opened while the project is broken also shows this message, not silently the old version.

### Scrolling

Before rebooting, the script remembers scrolling the page in `sessionStorage`, after downloading it returns it. The form status is not saved.

### Speed

- **Compilation** is a project based on examples of 15–50 ms. If it becomes visible on large projects, the compiler can store the result of each file under the hash of its contents and recompile only changed files.
- **Launch JS** is a big part of the small upgrade price: on Bun one page costs 20–24 ms, almost all of it is process start; on Node - 80–90 ms ([Change tracking](../change-tracking.md)). After changing the data, the project code is the same, and the JS process can be kept running between updates. After changing the code, the process starts again: it has a new code.

### What is beyond the box

- Replacement of components on the fly, without reboot (HMR). It can be taken from Svelte when the Svelte islets appear in the browser.
- Replacing HTML pages without rebooting (morphdom and the like). A complete reboot is easier and not wrong; you can return to the substitution if the reboot becomes interfered with.
- Calculate only open pages and do not calculate the rest. The remaining pages would be left from the old code - this is an outdated site.
- Live Reload in `shugo serve`: This is a server for visitors.

## Open questions

- Show an open page as soon as it is calculated, until the end of the calculation of the entire site, or wait for the whole site as now.
- Error on top of last successful page or separate page with error instead.
- Keep the JS process running between data updates in `shugo dev`.
- Change styles without rebooting: is it worth changing what the build name is made of?
