# Compatible with Svelte

Proposal on how to make the `.shugo` file even closer to the Svelte component: that what the agent writes is transferred to Svelte without editing, and Shugo swears in the same place where Svelte swears. None of this has yet been done.

## Decisions

- Templates are components of Svelte 5; syntax and behavior are copied by Svelte (the decision of the author of the project 21 November 2026).
- The file extension remains `.shugo` (22 solution of November 2026 year).
- `javascript:` addresses are displayed as is, as in Svelte: replacement cancelled (22 November 2026 decision).
- Compatibility improvement is a separate proposal, this document (22 decision of November 2026 year).

## What's going on

HTML pages match Svelte 5.59.1 bytes per byte, except for hydration marks. Checks and trivia around:

| What | Svelte | Shugo |
| --- | --- | --- |
| TypeScript without `lang="ts"` | error | adopted |
| Accessibility Warnings | several tens (`a11y_missing_attribute` and others) | none |
| Runes: `$props()` in markup, second `$props()`, rune out of start of announcement | errors | not checked |
| Component Style Class | only on the elements that the selector chooses | on all components of the component |
| A man who chooses nothing. | commented on CSS, warning `css_unused_selector` | remains |
| `<svelte:head>` in a conventional component | works | error |
| `$lib` in import | `src/lib` in SvelteKit | not understood |

Example of Style Class:

```svelte
<div class="card">
  <h2>Pitcher</h2>
  <p>Clay, handmade</p>
</div>

<style>
  .card { border: 1px solid #ddd; }
  h2 { font-size: 1.5rem; }
  .price { color: red; }
</style>
```

| | HTML | CSS |
| --- | --- | --- |
| Svelte | `<div class="card svelte-x"><h2 class="svelte-x">Pitcher</h2> <p>Clay…</p></div>` | `.price` commented, warning |
| Shugo | `<div class="card svelte-x"><h2 class="svelte-x">Pitcher</h2> <p class="svelte-x">Clay…</p></div>` | `.price.svelte-x` |

The page view is the same: `<p>` has no rules, and the extra class doesn’t change anything.

## Proposal

### What to do

1. **`lang="ts"`.** TypeScript in `<script>` without `lang="ts"` is a prompt error like in Svelte. Then the file is transferred to Svelte without editing.
2. **Rune places.** Same errors as Svelte: `$props()` only in the top level announcement of `<script>` and once, runes are not in markup, `$state` and `$derived` are only in the announcements.
3. **Style class is like Svelte.** The compiler matches selectors with template elements and puts the class only on the selected ones; a selector that does not select anything is commented on in CSS, and the build warns. Then HTML will coincide with Svelte and in the class: `test/svelte` standards will not be an exception.
4. **Accessibility alerts.** Same as Svelte: picture without `alt`, link without address, roles, captions at fields. Agent they are useful, and in Svelte he is waiting for them.
5. **`<svelte:head>` in a conventional component.** Allow in a component that is calculated inside the page. In the general component that is calculated once per entire site, there is an error with an explanation: it does not have a page in which `<head>` to write.
6. **`$lib`.** Understand `$lib/…` as `src/lib/…`, along with `@/`: code from SvelteKit projects writes just like that.

### How to check: by yourself or by the compiler Svelte

The 2, 3 and 4 paragraphs are several thousand lines of checks in the Svelte compiler. Two ways:

| | Your checks on Go | Svelte compiler in `shugo check` |
| --- | --- | --- |
| How | transfer Svelte rules to Go, as transferred the output of HTML | `shugo check` gives each `.shugo` file to the present `svelte/compiler` and prints its errors and warnings with a place in the file |
| For | works with each assembly, without dependencies | match by construction, all rules at once, new versions of Svelte - free |
| Against | big work, lag from Svelte | need `svelte` package in the project; check by separate command, not for every build |

The `.shugo` file is a component of the Svelte: `Shugo.query` for the Svelte compiler is just a name, `<script module>` with `getStaticPaths` is a regular module. Therefore, the second path works without reworking files.

**The Council:** Items 1, 5 and 6 in compiler Shugo: they are short. Item 3 Also in the compiler: it changes HTMLParagraphs 2 and 4 compiler Svelte the City of London `shugo check`which is already decided to do for type checking.

## Open questions

- Do `shugo check` with the Svelte compiler or transfer checks to Go.
- Whether to repeat the Svelte warnings at each build or only in `shugo check`.
