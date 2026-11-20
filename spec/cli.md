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
- `shugo serve` can not be restarted: it sees every update.

### Reports

```
$ shugo update
updated in 118 ms: 1052 pages, 1009 shared blocks, 12214 blocks, 3150 values
  pages: 0 added, 1 changed, 0 removed
    changed: /products/product-5
  shared blocks: 0 added, 1 changed, 0 removed
    changed: components/ProductCard.shugo{"id":"p5",}
```

In JSON, the same time of each step is in milliseconds:

```json
{
  "build": "…",
  "rebuilt": false,
  "pages": { "total": 1052, "added": 0, "changed": 1, "removed": 0, "changedNames": ["/products/product-5"] },
  "shared": { "total": 1009, "added": 0, "changed": 1, "removed": 0, "changedNames": ["components/ProductCard.shugo{\"id\":\"p5\",}"] },
  "blocks": 12214,
  "values": { "total": 3150, "added": 2, "removed": 2 },
  "shadowed": 0,
  "time": { "open": 5.1, "compute": 110.4, "save": 1.2, "total": 118.3 }
}
```

The names in the report are no more than a thousand of each species. `rebuilt` means that the project was compiled anew and everything is calculated from scratch. `shadowed` - how many addresses the dynamic page gave way to the page with the exact name.

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
