# Data loading

The document describes where the data for pages comes from and when they are loaded.

## Decisions

- The data gets code on TypeScript at the top of the `.shugo` files.
- Your data can load pages, layouts and components.
- The framework is not tied to a data source: it can be a database, file or someone else's API. What has changed is determined by comparison, not tracking the source.
- For now, every update will start anew and load all the data; then the result is compared to the past.
- Updating part of the data on signals from sources is a good idea for the future: [proposal](partial-reload.md).
- The framework stores a copy of the project data in SQLite; data from the source brings an adapter to it: [Data adapter and data copy](data-adapter.md).

## Made in the first version, waiting for approval

### Data does not leave JS

Data code and template expressions are executed in one process ([1](decisions.md#1-data-loading-and-expression-are-counted-together-go-compares-the-result)). Therefore, everything that is in the code is available in the template: functions, dates, `Map`, classes. Serialization is not needed, and a library like Seroval is also available.

In Go, only the ready expression values are left text.

According to [proposal for Analysis of Changes](change-tracking.md), data does not leave JS either, but the template sees a copy of it: it will not see the function declared in the data code.

### Where to download

| What you need | Where to download |
| --- | --- |
| Single page data | the Page Code |
| Data of many pages of one type | in `getStaticPaths`, by one request, and distribute via `props` |
| Data shared across all pages (menu, settings) | in a component without parameters: it is executed once |
| Data needed in several locations | in the module from `lib/`: the module is performed once per calculation |

There is no separate concept of “project data”: this is a regular module that is imported.

### Data Files

The data file is read during the calculation: `JSON.parse(readFileSync("data/posts.json", "utf8"))`. The path is counted from the root of the project.

You can import the file: `import posts from "../data/posts.json"`. Then it becomes part of the compiled code, and after changing it, the project needs to be compiled again. `shugo update` does it himself.

### What is worth knowing

- Page code is executed for each page, component code is executed for each display. The request to the database in the product card code will be executed as many times as the cards on the site. The exception is common blocks: they are executed once.
- Modules from `lib/` are run once. Loaded in the module lives to the end of the calculation.
- Simultaneously, 32 pages are calculated to allow for parallel responses from sources.
- Creating `Intl.NumberFormat` and similar objects is expensive. You need to create them once in the module, not every call: at the stand, this accelerated the calculation threefold.

## Offer: Calculation without waiting

Now the function of each page and each component is asynchronous, the component call and the slot content go through `await`, and 32 pages are calculated intermingled. This is left over from the first version: there, the data code itself downloaded data from anywhere from the network, from files and while one page waited for a response, others were considered.

There is nothing to expect with a copy of the data in SQLite. The request via `Shugo.query` responds immediately: SQLite works in the same process, and in Bun, and in Node, its interface is synchronous. The request for an index is also a synchronous exchange with Go. Go is also waiting for nothing: it asks JS a question and reads the answer.

It is proposed to calculate pages without waiting:

- compiler writes normal functions, without `async` and `await`;
- pages and common blocks are calculated one by one, in a row;
- who is being calculated now is a common variable, not `AsyncLocalStorage`: pages no longer go intermingled;
- `await` in the page or component code and asynchronous `getStaticPaths` is a prompt error: data comes from the database via `Shugo.query`, and what is waiting is done in the adapter or once when the module is loaded.

What it gives:

- reading past the base via `await fetch(…)` becomes an build error, not a page that silently becomes obsolete;
- the line left in the module can always be caught: pages do not interleave;
- first assembly of the store on 100,000 products: calculation on Bun 7.41–7.44 s instead of 7.50–7.75 s, on Node 9.66–9.82 s instead of 10.88–11.46 s; there is no difference on updates from several pages;
- the code of the framework and the generated code is simpler.

What is lost: Libraries that do not have a synchronous option cannot be called for each page. Their place is an adapter that prepares data, or the top of the module where they are once created: `const highlighter = await createHighlighter(…)` is performed when loaded, and then the page calls it in synchronously.

Trial implementation is a branch of `lab/sync` from the fifth option. The site comes out the same: at the stand of 10,000, the updated products coincided with the assembly from scratch. Do not pass the tests that are waiting in the page code or in `getStaticPaths` - so it should be. Waiting for approval.

## Open questions

- Calculation without waiting: approve, and what to do libraries without a synchronous option.
- Approve [proposal for analysis of changes](change-tracking.md). Calculate everything and compare the result, as in the first version, the author refused.
- Partial Reboot: [proposal](partial-reload.md).
- Memory between calculations for expensive requests to strangers API.
- Time and random numbers in the data code: each update will give a new value, and the page will be considered changed.
