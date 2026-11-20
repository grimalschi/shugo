# Partial data reboot

This document is about the step after [data Change Analysis](change-tracking.md). The change analysis only calculates the change, but the data loads everything. Here is a suggestion on how not to load everything when the data source can say what exactly has changed.

The document is written before the proposal for the analysis of changes. The options below remain valid, the order of steps at the end is corrected.

## Decisions

- While the process starts again and loads all the data, then the result is compared with the past.
- Updating signals is a good idea for the future.

## What is it like to “count one page”

The framework doesn’t know what data the page depends on: the data code is a regular TypeScript that reads anything. Changing one product can affect its page, category lists, counter in the cap and other people's recommendations. If you count only the product page, the rest will be silently outdated.

So, the link "data  pages" someone should name. Options are three.

## Variant A. Multiple processes instead of signals

Pages are divided between several processes of JS. Nothing is necessary to declare, there can be no obsolete, and time falls almost in proportion to the number of cores.

This is not a partial reboot, but the cheapest way to speed up the update. I suggest making it the first.

## Variant B. Labels

The code says he read it.

```
<!-- pages/products/[slug].shugo -->
---
const { product } = Shugo.props
Shugo.tag(`product:${product.id}`)
---
```

```
<!-- components/Header.shugo -->
---
Shugo.tag("menu")
const menu = await getMenu()
---
```

```
shugo update -tag product:7 -tag menu
```

- The page or shared block labels are written to the storage along with the record.
- `shugo update -tag …` only calculates entries with these tags. The rest is not touched or deleted.
- The list of the route pages can also change (product added). Therefore, `getStaticPaths`: `export const tags = ["products"]` also receives the label.
- The labelless entry is updated only with a full update.

The same is in Next.js (`cacheTag` and `revalidateTag`), the agents record familiar.

**Price.** The forgotten tag gives an outdated page, and it is not visible. Therefore, the full update should remain and run on schedule as insurance. The tags accelerate, and the correctness ensures a complete update.

**What is JS?** To calculate one page, you need its element from `getStaticPaths`. So, `getStaticPaths` route is still performed entirely. If this is one request for 100,000 products, the winnings are reduced.

## Option B. Data layer with memory

The framework gives functions for reading data that themselves remember who read what:

```
const product = await data.get("products", id)      // dependence: Products7
const list = await data.list("products", { category })
```

Signal "changed products7» exactly finds the records. This is what was investigated in the first topic: accuracy is obtained when reading is key or equality. The agent does not need to put labels, but the data has to be read through these functions, and not arbitrary code.

## What I offer

1. **First** - [analysis of data changes](change-tracking.md): downloads everything, only the changed is calculated.
2. **Then**, if all data is loaded at the main expense, are signals: tags (option B) or a data layer (option B). Fingerprints make it safe: any part of the data code can be re-executed and compared.
3. Several JS processes (option A) speed up the calculation and first assembly and require nothing from the author of the site. The template code does not store data after separation, so it is easy to distribute to processes.

## Open questions

- How often data changes on these sites and what size of the site to consider a goal. It depends on whether you need a step 3 at all.
- Who gives the signal: webhook source, queue, schedule.
- What to do with a record that does not have a label: warn when assembling or silently update only with a full update.
