# Variant 1: fingerprinting with a dependency tree

`lab/fingerprints` branch. This is a test application of the [spec/change-tracking.md](spec/change-tracking.md) sentence: the update only calculates those pages and common blocks whose templates read the changed ones. It is based on a common base of `lab/base`, where Go calls itself what to calculate and injects the partial result into the storage facility.

The second option, SQLite data under the control of the frame and queries via `Shugo.query`, is in the `lab/query` branch.

## How it works

1. **Compilation.** is the result of three things from each `.shugo` file: a data code, a template and a tree of dependence -- that the template reads from the variable data code. The compilation tree removes the template from the expression.
2. **Two files collected.** The data code of all files is collected in `_build/load.mjs`, the templates are in `_build/render.cjs`. They do not have common modules: each has its own copy of everything it imports.
3. **Loading.** The `load.mjs` data code of all pages and common blocks that the vault knows. This is the downloading of all data.
4. **Print.** For each page and block `load.mjs` is a tree of dependence and counts the print: 64 bits from exactly the values that read the template. The values are taken from living objects, nothing is serilyized.
5. **Comparison.** Go gets prints, eight bytes per page or block, and compares it to saved ones. He calls what's changed or appeared.
6. **Calculation.** only for the named `load.mjs`, by the same passage, copies what reads the template, "its own part", and gives the template to `render.cjs`. The template gives values, Go compares the record to the saved one and writes the differences.

Both files are running in one JS process: `load.mjs` is launched by `shugo` and itself connects `render.cjs`.

```
project data ──► load.mjs: data code for all pages and common blocks
                       │  print of what the template reads, 8 bytes
                       ▼
                   Go: Comparison with saved prints
                       │  "comply these"
                       ▼
                   load.mjs: Part ──► render.cjs: template ──► record ──► Go: storage facility
```

## Rules for site authors

The rules three, the breach of each one is a mistake with a location in the file and a clue.

1. **Data downloads pages, models and common blocks.** A component whose data code downloads something only gets simple parameters and is calculated once on the entire site. The component to which the object was transferred only shows the transfer.
2. **The shuttle reads data and causes imported functions.** Data are variable data codes and imported values. The function declared in the data code does not see the template.
3. **The module whose function causes the template does not load the data itself.** The same for conventional component modules.

That the compiler considers downloading data in the component code:

- `await` outside its declared functions;
- `new Date()` without arguments, `Date.now()`, `Math.random()`;
- the words `fetch`, `process`, `Bun`, `Deno`, `XMLHttpRequest`, `WebSocket`;
- import a module that itself or through its imports connects `node:fs`, `node:http`, `node:net`, `node:child_process`, `bun:sqlite` and similar or contains the above listed, including any `await`.

Pages and mock-ups are always considered downloaders.

Communications received by the author:

| What's done? | What's it say? |
| --- | --- |
| the template calls a function from a module that reads a file | `pages/index.shugo → lib/data.ts → node:fs: the template calls shout of a module that reads files…` |
| the module from which the template takes function waits for import data | `this module waits for data when it is imported (await at its top)…` |
| code that causes the template, calls `fetch`, `Date.now()`, `Math.random()`, reads `process` | `fetch is called in code that a template calls…` — When computing page |
| the object was transferred to the component that downloaded the data | `components/More.shugo loads data: its script waits for data (await). So it is computed once for the whole site and takes only plain values… Here post is an object` |
| component downloads data and reads page | `this component loads data — its script … — and it reads the page it is shown on…` |
| the template uses a function from the data code | `the template uses label, a function that the script declares…` |
| in a variable that reads a template, function or a copy of a class | `pages/index.shugo: the template reads tools.shout, which is a function…` |
| `new Date()` or `await` in the template | i'm gonna have to get a message for each of them. |
| one data code changed the data that had already read the other. | `what the template reads has changed since the script ran…` |

## What a template reads

| In a template | What Reads |
| --- | --- |
| `{item.title}`, `href={item.a.b}`, `{item?.a}` | one value |
| `cond={item.discount}`, `!x`, `x && …`, `x ? … : …`, `x == null` | whether or not the template is given to `true` at the location of what is |
| `<g-for each={item.variants} as="v">` | and each element has what the body reads. |
| `{reviews.length}` | length |
| `{price(item.price)}` | `item.price`; `price` — Import, this is code |
| `{shop.name}` where `shop` is imported | `shop.name`: Imported value — Data |
| `{f(item)}`, `{item.tags.join()}`, `{items[0]}`, `{...attrs}`, `{item.a + 1}`, `{typeof x}` | value in whole |
| `<Variant variant={v} />` | `v` Total |
| `<ProductCard id={id} />` | value `id` |
| `as="{ id, title }"`, cycle on the calculated list | each element in its entirety |
| `Shugo.props.x` | `x`; `Shugo.url`, `Shugo.params`, `Shugo.slots` — code not specified |

Imports that create or tag the template anywhere are considered coded everywhere. The other imports that read the template are considered data and come from the data code.

## Decisions made during implementation

We can review each one.

1. **The tree is data, not the code generated.** compiles a list of forms near the data code: `1`, the whole value, `0`, the only presence, the object member, the one form list, the elements. `load.mjs` turns shapes into circuits. The code generated was 12% faster: 109 ms versus 124 ms per 100,000 pages of products. Two code generators aren't worth it for.
2. **At the location where only the presence is checked, the template receives `true`.** Missing — `null`, `undefined`, `0`, `""`, `false` - comes as it is. `{a && b}` and `{a == null}` They give you the same thing as living data.
3. **Imported values are data.** `{shop.name}` in the template is running: `shop` is being taken on the side of the download; otherwise each such import would have to be rewritten into a variable.
4. **Functions from the data code are not permitted in all files**, not just the downloaders. One rule is easier than two, and the component does not break when it starts loading something.
5. **The downloader is decided by modules rather than by functions.** A component that imports something from a file read module into the data code is considered to be downloadable. For templates, the check is more accurate: esbuild releases the unused and the function from such a module is allowed if the file is not reached before reading.
6. **`render.cjs` is collected in CommonJS format.** is not `await` at the top level, so esbuild itself refuses the module that waits for data at the import. The consequence is that the npm packages connect the templates through `require`; Node 22.12+ and Bun are connected to ESM.
7. **Print is MurmurHasch3 (x86, 32 bits) under two pills.**, 0 and `0x9747b28c`. The words are: type of value, then the value itself; text is length and two characters in word. Implementation is checked against published values in Go and JS.
8. **The whole value is re-hashed on each reading.** Memory of "site ~ its hash" slowed down: small objects read once cost a search and recording at `Map`.
9. **The prints are in the vault separate from the records, 256 pieces.** Every update compares all prints: reading parts takes 25–30 ms by 204,000, reading the same table lines as 150–200 ms. The update only rewrites parts where something has changed.
10. **Print response is 8 bytes per unit on request.**, without names and text.
11. **The data code without `await` is compiled into a normal function.** One hundred thousand of these pages are downloaded in a row, without one `Promise`.
12. **The lost data code of the common block does not drop the update at once.** The Warehouse can remember a block that no one else is showing and whose data has disappeared. If no one needs it after the upgrade, it is removed. If someone shows it, the update falls with its error.
13. **The component selected by the data code passes the boundary as a file number.** `const View = compact ? List : Grid`and then `<View />` It works like it did in the first version.
14. **The common unit without loading shall not be rechecked.** His name says everything he shows, his print is zero.
15. **The complete test is `shugo update -check`.** calculates everything and calls the pages and blocks that have changed with the previous print. The storage facility is then correct, the team ends with a mistake.
16. **Comparison on Go**, as in the sentence.
17. **Two numbers have been reported:** `loaded` — how many pages and blocks have a data code, `computed` — how many have the templates been calculated.

## Limitations

- **npm package that reads the file or database in a synchronous way** inside the template or the conventional component is not visible. The page will not be updated until something visible changes. Only `shugo update -check` finds it; the `TestCheckFindsWhatIsReadPastShugo` test shows both a hole and a check.
- **`new Date()` without an argument in the module**, whose function causes the template, is not caught. It is captured in the template and in the data code; `Date.now()` is caught anywhere in the project code.
- **Accuracy doesn't go through components and functions.** The object that is passed to the ordinary component or function is read in its entirety. On the stand, the change in the article of the option calculates the page of the product, although the record is the same. You can narrow it down, but you can't do it cleanly: a parameter that doesn't read a component decides whether or not the common block will work.
- **The data code of all pages and blocks is maintained for each update.** For 100,000 products, the data code and prints of 204,000 units are 0.46- 0.59 s on Bun. The request in the page code will be fulfilled for each page.
- **The loading party shall hold the results of the data code of all units until the end of the update.** This is a reference to project data, but at a million pages, memory will have to be counted.
- **Syntax that the dissertator doesn't read** `tdewolff/parse`: `using` announcements and decorators in the data code.
- **The analysis error is the beginning of the expression**, not the exact column: the disserator tree does not hold the positions.
- **`Shugo.self` not available in the data code** downloading file; available in the template.
- **Documents in `spec/` in this branch were not updated** except `docs/authoring.md`.

## Benchmark

```sh
go build -o bin/shugo ./cmd/shugo
node examples/catalog/stand.mjs --size 10000        # --runtime node -- compute on Node
```

The counter changes the data 21 times. After each change, it checks how many pages and common blocks are calculated and at the end compares a step-up site with a zero-size build, a file by file. For 1,000, 10,000 and 100,000 products, all checks are made, and sites are consistent.

In the example of the store, three locations were changed: the functions of `price` and `finalPrice` moved to `lib/money.ts`; the hat finds the item of the month in the data code; the card and page of the products consider the price to be a discount in the data code so that the template reads the number rather than the whole item.

### What gets evaluated

Same size except for the one marked.

| What changed the data | Pages calculated | Common blocks calculated | Pages with new HTML (100,000) |
| --- | ---: | ---: | ---: |
| Nothing. | 0 | 0 | 0 |
| A field that is nowhere to be shown, one in every 20 items | 0 | 0 | 0 |
| Price of one commodity | 1 | 1 | 7 |
| Prices of 20 products | 20 | 20 | 70 |
| Name of item of month | 1 | 2 | all |
| The products of the month were another commodity | 0 | 1 | all |
| Products of the month are no longer available | 0 | 1 | all |
| Products moved to another category | 168 per 100,000, 18 per 10,000 | 1 | all |
| The phone's in the basement. | 0 | 1 | all |
| The products have a discount. | 1 | 1 | 7 |
| Text of the approved withdrawal | 1 | 0 | 1 |
| Text of the hidden withdrawal | 0 | 0 | 0 |
| Balance of one option | 1 | 0 | 1 |
| Articule of an option that is nowhere to be shown | 1, previous entry | 0 | 0 |
| Option added first | 1 | 0 | 1 |
| The first option was deleted | 1 | 0 | 1 |
| One commodity values | 1 | 1, new | 1 |
| Two recommended items were replaced | 1 | 0 | 1 |
| The recommended products were removed | 1 | 0 | 1 |
| The last item was removed. | 88 per 100,000, 13 per 10,000 | 0; card removed | 88 |
| Products added | 85 per 100,000, 10 per 10,000 | 2 | all |

A hidden withdrawal is not computing anything because the page code itself selects the approved feedback: the template reads the already selected.

### Timings

Xeon E3-1270 v6, Go 1.27.4, Bun 1.4.4, Node 24.22.1. **Preliminary:** was in the second branch during the car measurements, with a load of 1.4–2.5.

| | 10,000 products | 100,000 products |
| --- | ---: | ---: |
| First build, Bun | 1.06 s | 9.2 s |
| Update, Bun | 0.14–0.20 s | 1.16–1.41 s |
| Update, Node | — | 1.89–1.99 s |
| Update in version 1, Bun | 0.40–0.45 s | 3.2–3.7 s |
| Update on a general basis, Bun | — | 2.9–3.2 s |

From which is the update of 100,000 items, Bun:

| Step | Time |
| --- | ---: |
| Open the vault and read the prints | 0.03 s |
| Start JS, upload data, list pages | 0.63–0.77 s |
| Data code 204 units, prints, comparison | 0.46–0.59 s |
| Compute changed, complete process | 0.04–0.05 s |
| Write | 0.002 s; 0.09 s when the cap changed and the number of pages with the new HTML is counted |

The second step is almost the whole example code: reading a file in 65 MB and 58 passageways on the list of products.

The storage facility for 100,000 items was 116 MB: 103 MB base and 13 MB with fingerprints.

## Where to look in the code

| File | What's in it? |
| --- | --- |
| `internal/compile/reads.go` | expression analysis: what reads the template |
| `internal/compile/shape.go` | form read and its recording |
| `internal/compile/bind.go` | that declares the data code; |
| `internal/compile/unit.go` | two modules from one file |
| `internal/compile/bundle.go` | two assemblies who downloads data, check the sides of the templates |
| `internal/jsrt/load.js` | data code, print, own part, Go responses |
| `internal/jsrt/render.js` | boxes; plugs to exits outside |
| `internal/evaluate/evaluate.go` | `changed`: What to Compute |
| `internal/evaluate/session.go` | printing request |
| `internal/store/store.go` | `Prints`: Fingerprints by parts |

Test: `internal/compile/reads_test.go` — rules of accuracy; `internal/jsrt/load_test.mjs` — hash and that the print covers a accurately transmitted template; `test/fingerprints_test.go` — which calculates updates, rules and their messages, complete verification; `test/random_test.go` — random templates and random corrections of data in one place against the independent calculation of HTML.
