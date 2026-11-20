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

## Open questions

- Approve [proposal for analysis of changes](change-tracking.md). Calculate everything and compare the result, as in the first version, the author refused.
- Partial Reboot: [proposal](partial-reload.md).
- Memory between calculations for expensive requests to strangers API.
- Time and random numbers in the data code: each update will give a new value, and the page will be considered changed.
