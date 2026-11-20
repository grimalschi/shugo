# Pointed page update

The paper describes how a change in data reaches only those parts of pages that depend on it.

The section name has not been selected yet.

## Decisions

- After the first complete build, the project is updated pointwise: new data is compared with the past, and only what is associated with changes is recalculated.
- The link between the data and the pages is taken from the dependency tree. It is built when compiling the project.
- The list element is recognized by the key, not by the place in the list. If the fifth product was removed from 100,000, the rest of 99,995 is not considered changed. The simplest is to take the field `id`, `key` or `_id` with the key.

## From the prototype, pending approval

- Unit of update is a block: a page or call of the component together with the part of the data it reads itself.
- The block depends on its fields and on which child blocks it contains, but not on their contents. Therefore, the change does not spread up and to the sides.
- Blocks with the same data are counted once, no matter how many pages they show.
- The element key can be specified explicitly in the loop. The key is the path to the element field, not the expression. Without an explicit key, `id` is taken, then `key`, then `_id`, and if there is none, the checksum of the element content.
- Accuracy depends on how the expression is written. `a.b.c` is one field. A method call or a calculated key binds a block to the entire object.
- The list calculated in the template (filter, sorting) is updated entirely together with its block.
- A block is calculated only if the page reaches it.
- Extra recalculation does not reach the page: if the block values are the same, the page is not redrawn.
- The same keys in one list are data errors that need to be reported.

## Open questions

- Title of section.
- Blocks at components with several parameters and with computable parameters.
- Blocks and contents transferred inside the component (slots).
- The general block for the entire site (cap): how to change it not to touch the records of all pages.
- Update Report: What is reported after each update.
