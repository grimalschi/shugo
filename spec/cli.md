# Command line and development mode

## Decisions

- Development mode is needed: reassembly when changing files and updating the page in the browser.
- Changing the project code calculates the entire site anew, and this is correct. The development mode will receive a real Live Reload separately: [proposal](../docs/proposals/live-reload.md) (21 solution November 2026 year).
- CLI messages and the specification use English so agents can read them and search for their wording. Confirmed November 21, 2026.
- `shugo init` is needed: a new project with an example, `tsconfig.json`, types for the editor and a memo for the agent (21 solution November 2026 years). Not yet done.
- Type check — a separate command `shugo check`, together with verification of component parameters (21 solution November 2026 year). Not yet done.

## Implemented, pending approval

```
shugo build  [directory]   compilation and calculation of all pages
shugo update [directory]   calculate what the data or project changes have touched; only the difference is written
shugo serve  [directory]   server: pages are collected on request
shugo export [directory]   site to files for any web server; after the first time - only changed
shugo dev    [directory]   update, server and update every time you change files
```

A folder is a project; without it, the current one is taken.

| Parameter | What does |
| --- | --- |
| `-json` | `build`, `update` and `export` report in JSON |
| `-check` | with `update`: calculate everything and name pages that have changed, although no line of the database they read has changed |
| `-runtime bun` or `node` | than to perform JS in `build`, `update` and `dev`; otherwise `"runtime"` from `shugo.json`, otherwise Bun if it is installed, otherwise Node |
| `-addr address` | where they listen to `serve` and `dev`; by default `localhost:4321` |
| `-out folder` | where `export` writes; default `_build/dist` |

The option that the command does not use is an error ([27](decisions.md#27-commands)).

### How Teams are Different

| Command | Compiles | Calculates | Gives pages | Follows files |
| --- | --- | --- | --- | --- |
| `build` | always | all | none | none |
| `update` | if the project has changed | what changes have occurred | none | none |
| `serve` | none | none | from storage | none |
| `export` | none | none | write to Files | none |
| `dev` | if the project has changed | what changes have occurred | from storage | yes, and resets open pages |

`dev` is `update` and `serve` together, plus tracking: after each file change, it makes `update` and says pages open in the browser to reboot. It calculates nothing of its own: the pages are the same as after `shugo update`.

### When a team

- The data has changed — `shugo update`. It does not compile the project, but runs the already collected JS and calculates only what the changed lines read.
- If you run `shugo update` after changing the project itself, it will notice this, compile the project and calculate everything. To do this, the compiler writes hashes of all files from which the project is collected. So the site can not show the result of the old code.
- `shugo build` calculates everything always. It is needed if pages read something past the database.
- `shugo serve` can not be restarted: it sees each update from the next request.

### Reports

Shop on 100,000 products, changed the price of one product:

```
$ shugo update
updated in 28 ms: 104202 pages, 100010 shared blocks
  computed 1 pages and 1 shared blocks, 13 blocks in all
  1 rows of the data have changed
  pages: 0 added, 1 changed, 0 removed
    changed: /products/product-5
  shared blocks: 0 added, 1 changed, 0 removed
    changed: src/components/ProductCard.svelte{"id":"p5"}
  the HTML differs on 7 pages: they show a shared block that has changed
```

The report has two different numbers. “Changed pages” are page entries in the repository: one. “HTML is different” are the pages that a visitor will see others: the product page itself and the six pages where his card stands. Their records are not touched because the card is kept separately.

In JSON, the same time of each step is in milliseconds:

```json
{
  "build": "7e2a9db7a114efe7bddf2f8b9ea84dbe",
  "rebuilt": false,
  "data": { "whole": false, "rows": 1, "routes": 0, "position": 5060 },
  "computed": { "pages": 1, "shared": 1 },
  "blocks": 13,
  "pages": { "total": 104202, "added": 0, "changed": 1, "removed": 0, "changedNames": ["/products/product-5"] },
  "shared": { "total": 100010, "added": 0, "changed": 1, "removed": 0, "changedNames": ["src/components/ProductCard.svelte{\"id\":\"p5\"}"] },
  "render": 7,
  "renderNames": ["/", "/categories/c5", "/products/product-5", "…"],
  "shadowed": 0,
  "time": { "open": 0.3, "data": 5.1, "list": 0.1, "compute": 20.9, "save": 0.5, "total": 27 }
}
```

- `data` - what the database log said: how many lines changed, how many lists of route pages were read again and to which log line the data was taken into account. `whole` means that the log could not say what had changed, and everything was calculated.
- `render` — how many pages of HTML have changed: these are the changed pages and pages that show the changed common block. You can reset the cache in front of the site from this list.
- The names in the report are no more than a thousand of each species.
- `rebuilt` means that the project has been compiled anew and everything is calculated from scratch.
- `shadowed` - how many addresses did the dynamic page give way to a page with an exact name.

### Errors

An error in the project is shown with the space in the project file and a line of code:

```
error: src/pages/about.svelte:4:14: </div> does not close <span> opened at 4:3
     3 | <div>
     4  <span>text</div>
       |              ^
```

The error during the calculation shows the page that was calculated and the place in the original file, not in the JS collected:

```
error: Cannot read properties of undefined (reading 'title')
  while computing the page /about (src/pages/about.svelte)
  at src/pages/about.svelte:5:21
     4 | const first = products[10]
     5 | const title = first.title.toUpperCase()
       |                     ^
```

Node indicates the location accurately. Bun sometimes points to a nearby string: it rearranges the code itself before executing.

### Development Mode

`shugo dev` monitors all project files except `_build`, `node_modules` and hidden files. After the change, it does what `shugo update` did: the source code has changed compiles the project and calculates everything; the database has changed calculates what the changed lines have read; other files have changed calculates nothing. The pages are reloaded anyway. What to improve in it - in [the Live Reload](../docs/proposals/live-reload.md).

## Open questions

- Installation: how the `shugo` gets on the machine. Hugo, also one file on Go, is distributed as follows: ready files for all systems in releases on GitHub, Homebrew, Snap, Winget, Scoop and Linux distribution packages, Docker image and `go install` for those who have Go. Through npm Hugo put other people's wrapping packages. esbuild is distributed by the npm package, which puts the finished file under the system. Tip: GitHub releases, npm package like esbuild — agents work in projects with `package.json` — and Homebrew.
