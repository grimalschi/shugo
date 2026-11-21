# Command line and development mode

## Decisions

- Development mode is needed: reassembly when changing files and updating the page in the browser.

## Made in the first version, waiting for approval

```
shugo build  [directory]   compilation and calculation of all pages
shugo update [directory]   calculation again after data change; only difference is written
shugo serve  [directory]   server: pages are collected on request
shugo export [directory]   site to files for any web server
shugo dev    [directory]   assembly, server and reassembly with each file change
```

A folder is a project; without it, the current one is taken.

| Parameter | What does |
| --- | --- |
| `-json` | `build`, `update` and `export` report in JSON |
| `-runtime bun` or `node` | than to perform JS; otherwise `"runtime"` from `shugo.json`, otherwise Bun if it is installed, otherwise Node |
| `-addr address` | where they listen to `serve` and `dev`; by default `localhost:4321` |
| `-out folder` | where `export` writes; default `_build/dist` |

### When a team

- The project (templates, code, styles) - `shugo build` has changed.
- Only the data has changed — `shugo update`. It does not compile the project, but runs the already collected JS.
- If you run `shugo update` after the project itself is changed, it will notice this and compile the project first. To do this, the compiler writes hashes of all files from which the project is assembled. So the site cannot show the result of the old code.
- `shugo serve` can not be restarted: it sees every update.

### Reports

```
$ shugo update
updated in 68 ms: 1052 pages, 1009 shared blocks, 3150 values
  computed 1052 pages and 1009 shared blocks, 12214 blocks in all
  pages: 0 added, 1 changed, 0 removed
    changed: /products/product-5
  shared blocks: 0 added, 1 changed, 0 removed
    changed: components/ProductCard.shugo{"id":"p5"}
  the HTML differs on 7 pages: they show a shared block that has changed
```

The report has two different numbers. “Changed pages” are page entries in the repository: one. “HTML is different” are the pages that a visitor will see others: the product page itself and the six pages where his card stands. Their records are not touched because the card is kept separately.

In JSON, the same time of each step is in milliseconds:

```json
{
  "build": "…",
  "rebuilt": false,
  "pages": { "total": 1052, "added": 0, "changed": 1, "removed": 0, "changedNames": ["/products/product-5"] },
  "shared": { "total": 1009, "added": 0, "changed": 1, "removed": 0, "changedNames": ["components/ProductCard.shugo{\"id\":\"p5\"}"] },
  "blocks": 12214,
  "values": { "total": 3150, "added": 2, "removed": 2 },
  "render": 7,
  "renderNames": ["/", "/categories/c5", "/products/product-5", "…"],
  "shadowed": 0,
  "time": { "open": 4.9, "compute": 61.3, "save": 1.1, "total": 67.6 }
}
```

- `render` — how many pages of HTML have changed: these are the changed pages and pages that show the changed common block. You can reset the cache in front of the site from this list.
- The names in the report are no more than a thousand of each species.
- `rebuilt` means that the project has been compiled anew and everything is calculated from scratch.
- `shadowed` - how many addresses did the dynamic page give way to a page with an exact name.

### Errors

An error in the project is shown with the space in the project file and a line of code:

```
error: pages/about.shugo:4:14: </div> does not close <span> opened at 4:3
     3 | <div>
     4  <span>text</div>
       |              ^
```

The error during the calculation shows the page that was calculated and the place in the original file, not in the JS collected:

```
error: Cannot read properties of undefined (reading 'title')
  while computing the page /about (pages/about.shugo)
  at pages/about.shugo:5:21
     4 | const first = products[10]
     5 | const title = first.title.toUpperCase()
       |                     ^
```

Node indicates the location accurately. Bun sometimes points to a nearby string: it rearranges the code itself before executing.

### Development Mode

`shugo dev` monitors all project files except `_build`, `node_modules` and hidden files. This means that changing the data file inside the project also causes a reassembly.

## Open questions

- `shugo init`: creating a new project with an example and a memo for the agent.
- Project validation command without assembly.
- Message language ([22](decisions.md#22-message-language)).
- Installation: how the `shugo` gets on the machine (ready-made binary, npm package).
