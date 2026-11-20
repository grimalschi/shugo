# Data loading

The document describes where the data for pages comes from, when they are loaded and how framework learns what has changed in them.

## Decisions

- The data gets a TS code at the top of the `.shugo` files.
- Data loading is a separate stage. It performs JS files collected during compilation, and they load data immediately for the entire project.
- After downloading, the new data is compared with the past. As a result, the next stage recalculates only what is associated with changes. The comparison most likely makes Go.
- Changes are about comparing data, not tracking the source. Therefore, framework is not tied to the database, its capabilities and structure.

## From the prototype, pending approval

- Data is the usual values: objects, lists, strings, numbers. Fields that are calculated when accessed are not tracked.
- If the page data has not changed by a single byte, the page is not further understood.
- Shared data (cap, menu) should come with one shared input, not a copy in each page's data.
- Selection from big general data ("more products of this category", recommendations) is done at the data stage. The template comes with a ready list.
- The time to compare grows with the amount of data, not with the size of the change.

## Open questions

- Can components and layouts load their data?
- Data shared throughout the project.
- Reloading part of the data: by signal from the source, by life, by webhook.
- In what form data is transferred from JS to Go and stored between stages.
- What can be returned from the data code, except for the usual values (dates, functions).
