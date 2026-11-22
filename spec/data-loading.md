# Data loading

The document describes where the data for pages comes from and when they are loaded.

## Decisions

- Data gets code on TypeScript in `<script>` files `.shugo`.
- Your data can load pages, layouts and components.
- The framework is not tied to a data source: it can be a database, file or someone else's API. What has changed is determined by comparison, not tracking the source.
- While each update, the process starts anew and loads all the data; then the result is compared to the past. *Replaced by solutions from 20 November 2026 year:* update recognizes changes from the database log and calculates only what they touched - [Change tracking](../docs/change-tracking.md#decisions).
- Updating part of the data on signals from sources is a good idea for the future. *Updated 21 November 2026 year:* point data changes bring adapter — [proposal](../docs/proposals/data-adapter.md).
- The framework stores a copy of the project data in SQLite; data from the source brings an adapter to it: [Data adapter](../docs/proposals/data-adapter.md).
- Pages are calculated without waiting: `await`, `getStaticPaths` synchronous is not in the code of pages and components. The wait is left from the first version, where the code itself loaded data over the network. 21 solved on November 2026 year.
- The query constructor remains as it is; a new one is added as needed, and the first in line is to track the links between tables (21 solution of November 2026 year).

## Implemented, pending approval

### Data comes from the project database

Pages, layouts and components read data only from the SQLite database of the project, named in `shugo.json`: `{ "data": "data/site.db" }`. Read via `Shugo.query`, route via `query` in `getStaticPaths`. The request responds immediately: SQLite works in the same process, and in Bun, and in Node its interface is synchronous.

Each answer line remembers which columns are read in the page code, in the template, in the functions to which it was passed, in the components. It depends on what is calculated after changing the line: [Change tracking](../docs/change-tracking.md).

Data code and template expressions are executed in one process. Therefore, everything that is in the code is available in the template: functions, dates, `Map`, classes. In Go, only ready-made expression values are left - text.

### Where to ask

| What you need | Where to ask |
| --- | --- |
| Single page data | in the page code, according to its parameters from the address |
| Data that shows many pages, such as a product card | in a component with simple parameter: `<ProductCard id={row.id} />`. It is calculated once for each product |
| Data shared across all pages: menus, settings | in a component without parameters: it is calculated once |
| Route pages list | in `getStaticPaths`; it is better to return only `params`, and the page will take the data itself |

The module from `src/lib/` cannot ask the base: the request belongs to the page or component that asked it. The function from the module receives a string or the request itself by argument.

### Data from other locations

What was read from outside the database -- a file at the time of computing, a network, a value that the module stored between pages -- does not track changes: `shugo update` will not notice them. `shugo build` will compute and reread them, and `shugo update -check` will name the pages that depend on them.

The file can be imported: `import posts from "../../data/posts.json"`. Then it becomes part of the compiled code, and after changing it, the project is compiled again. `shugo update` does it himself.

### Without waiting

The compiler writes normal functions, without `async`. Pages and common blocks are calculated one at a time. `await` in the page code or component and asynchronous `getStaticPaths` is a build error with a hint. What is expected and done once, for example, creating a code highlight, wait at the top level of the module from `src/lib/`, and the page calls ready in synchronous.

What it gave:

- reading past the base via `await fetch(…)` — build error, not a page that silently becomes obsolete;
- the line left in the module can always be caught: pages do not interleave;
- first assembly of the store on 100,000 products: calculation on Bun 7.41–7.44 s instead of 7.50–7.75 s, on Node 9.66–9.82 s instead of 10.88–11.46 s; there is no difference on updates from several pages;
- the framework code and the generated code are simpler: there is no pool of 32 pages and `AsyncLocalStorage`.

What is lost: a library without a synchronous option cannot be called for each page. Its place is an adapter that prepares data, or the top of the module.

### What is worth knowing

- Page code is executed for each page, component code is executed for each display. Exception - common blocks: they are executed once.
- Modules from `src/lib/` and `<script module>` are performed once per calculation.
- Creating `Intl.NumberFormat` and similar objects is expensive. You need to create them once in the module, not every call: at the stand, this accelerated the calculation threefold.

- Time and random numbers (`Date.now()`, `Math.random()`) do not calculate anything themselves: a page is calculated only when what it read from the database has changed. But when it is calculated for another reason, the value is different and the page is considered changed, and neighboring pages show the time of their calculation. Whether to take time from the data, decides the author of the site: Shugo does not prohibit this (the decision of 22 November 2026 year).

## Open questions

- Libraries without a synchronous version: whether the top of the module and adapter is enough is postponed in [proposal for an adapter](../docs/proposals/data-adapter.md#libraries-without-synchronous-option).
