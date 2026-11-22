# Layout of a separate entry for each page

Proposal by the author of the project, 22 November 2026 year: layout wraps pages, and changing the layout should not touch the page entries, and changing the page - a layout. None of this has yet been done.

## Decisions

Nothing yet.

## Like now

Layout is a component that receives a page with the `children` snippet. How it is stored depends on whether it reads the page address.

- **Does not read** `Shugo.url` and `Shugo.params`. Then it is a common block: one record for the entire site. The page only keeps a link to it and its snippets are `children` and what it gave in `<head>`. The menu in the layout has changed - one entry is calculated, pages are not touched. It works as the author wants.
- **Reads** address - for example, highlights the current menu item: `class:active={Shugo.url.pathname === item.href}`. Then its output is different on each page, and now it is calculated inside the page record: its blocks lie in it, its readings from the database are written to the page account.

In the second case:

- changed menu - all 100,000 page records are calculated again, together with the code of the pages themselves: their requests, templates, components;
- the product has changed - its page is calculated together with the layout, although the layout does not depend on the product.

## Proposal

The layout that reads the address receives its entry to each page: "the layout is so-and-so for the address of such and such." The page record stores a link to it, not its contents - just like now a link to the general layout. The layout records its readings on itself.

Then:

| What has changed | Now | With layout records |
| --- | --- | --- |
| Menu that reads a layout with backlight | 100,000 page entries: page code and layout | 100,000 layout entries: layout code only; page entries are not affected |
| Menu that reads the layout without lighting | 1 recording | 1 record as now |
| Product | his page together with the layout | only his page |

The output of the layout that reads the address is still different on each page, so after changing its data, it is calculated for each page: it is cheaper not to do without removing the dependence on the address. But the layout can do anything — read the address, ask the database — and it never makes you re-compute the pages themselves.

### How it falls on what is

This is a common block, which, in addition to the file and parameters, includes the page address: `src/pages/+layout.shugo@/products/jug{|children,head,title}`. Everything else is already there:

- the common block is calculated by one name: `Shugo.url` and `Shugo.params` are restored from the address;
- the page record links to it and keeps its snippets;
- its readings from the database are written to it, and the update calculates it separately;
- page is gone its layout record is no longer shown, and it is deleted like any unnecessary shared block.

The thinnest `<head>`. The layout that `<head>` writes shows both its contents of `<svelte:head>` and what the page and internal layouts have given. The blocks will lie in the layout entry and the page blocks in the page entry, so the end of `<head>` is going to be from two places; the "last `<title>`" rule remains the same.

### More specifically.

If the layout does not read the entire address, but only the parameter - for example, `Shugo.params.category`, to show the section menu - you can put the values of the read parameters in the name of the entry, not the address. Then one layout record serves all the pages of the section. This is the second step.

### Price

- For each page, another record is made if the layout reads the address: for 100,000 pages, 100,000 records, and the server makes another key search for each request, around 5 µs.
- The same records could be made for ordinary components that read the address, for example, bread crumbs. I suggest starting with layouts: there is more winning and the rule is clearer.

## Open questions

- Do it now.
- Only for layouts or for any component that reads the address and is called with simple parameters.
- Key - the address of the whole or the values of the read parameters.
