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

The idea of the author of the project, 22 November 2026 years: in development mode, you do not need to calculate the entire site - each request can be processed separately. So the development modes SvelteKit and Astro work: the page is calculated when asked.

### Page is calculated on request

- `shugo dev` compiles the project and keeps the JS process running. It does not compute the entire site.
- A request has come - this page is calculated, with the current code and current data, and given to the browser. Its record with what it read remains in the development mode store.
- The next request of the same page is given from the repository, if since then neither the code nor the lines it has read have changed. Otherwise it is calculated again. There is no obsolete page: it is either checked or calculated anew.
- Whether there is a page at the address says `getStaticPaths` of its route. It is calculated when first asked for the address of this route, and stored in the same way, until you change what you read.
- Price: in the running process, the recording is calculated in tens of microseconds - the first assembly of the store calculates 204,000 records of pages and common blocks for 9.1 s, about 45 µs per record. The roads are just starting the process, 20–24 ms on Bun ([Change tracking](../change-tracking.md)), and it happens once.

The error in the page that was not opened, the development mode will not show: it will find `shugo build` before laying out.

### What is restarted and when

| What has changed | What is calculated | What the browser sees |
| --- | --- | --- |
| Templates and code: `.shugo`, `.ts` in `src/`, `shugo.json` | compiling; pages - when asked | open pages are reloaded and re-computed |
| Data | nothing at once; a page whose lines have changed when asked | reloads open pages that have read the changed lines |
| Styles only: `.css` or `<style>` in template | nothing | new style file is replaced in open pages without rebooting |
| Files in `public/` | nothing | reload all open pages |

To know which pages are open, the page script names its address when it connects: `/_shugo/live?path=/products/jug`.

### Styles Only: No Reset

All site styles are collected in one file with a hash in the name, and a link to it framework inserts each page into `<head>` when it collects it HTML. The page entries on this file know nothing. Changing some styles therefore does not require calculating any pages: open pages get the word `css` with a new address and replace `<link>` as in Vite.

Now `.css` changes the build name, and everything is calculated anew: a link to styles is included in the site program. It must be considered outside the build name.

### Error in browser

The compile or compute error is shown on top of the open page: the same text as in the terminal is the space in the project file and the lines of code around. Fixed - the message disappears, the page is reloaded. So do Vite and SvelteKit. The page that was opened while the project is broken also shows a message, not silently an old version.

### Scrolling

Before rebooting, the script remembers scrolling the page in `sessionStorage`, after downloading it returns it. The form status is not saved.

### What is beyond the box

- Replacement of components on the fly, without reboot (HMR). It can be taken from Svelte when the islands of Svelte appear in the browser.
- Replacing HTML pages without rebooting (morphdom and the like). A complete reboot is easier and not wrong.
- Live Reload in `shugo serve`: This is a server for visitors.

## Open questions

- Development mode storage: separate file or the same `_build/store.db`. The separate one does not mix with the build for laying out, but `shugo serve` after `shugo dev` is then empty.
- Error on top of last successful page or separate page with error instead.
