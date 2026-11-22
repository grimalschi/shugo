# Open questions

Questions are numbered: answer the numbers. Where there is advice, yes - to accept it. Decisions taken without confirmation are collected in [Decisions awaiting approval](decisions.md); proposals that do not yet exist are in [docs/proposals](../docs/proposals/README.md).

### 1. Types for editor

The editor does not know the `shugo` module: on `import { query, page, search } from "shugo"` type check swears. If you put type announcements in the project once, after updating Shugo they will become obsolete.

So this problem is solved SvelteKit and Astro: framework writes types to your build folder every time you run `.svelte-kit/` and `.astro/`and the project once refers to it: `tsconfig.json` SvelteKit Heir `.svelte-kit/tsconfig.json`Astro is connected. `.astro/types.d.ts`. Types are always the version that collects the site.

For Shugo, this is what it looks like. Each build writes `_build/types/shugo.d.ts` — the `shugo` module ads that Shugo builds — and `_build/tsconfig.json`: where to look for `shugo`, which means `@/`, what settings TypeScript needs for Svelte. `tsconfig.json` is one line that does not change: `{ "extends": "./_build/tsconfig.json" }`; it is written by `shugo init`, which is decided to do. Updated Shugo the next build rewrote the types.

With 22 November 2026 year Svelte is sewn into Shugo, and the project without islands package `svelte` is not needed. Then the editor will not find and module `svelte`: `import type { Snippet } from "svelte"` and `import { getContext } from "svelte"` it will emphasize the same as `shugo`. The Svelte assembly can be placed in `_build/types/`.

Another way is the `shugo` package in npm with some types. Its version would have to be updated along with Shugo, and the assembly would check that they match.

**The Council:** types write assembly in `_build/`, like SvelteKit and Astro.

### 2. Approved items that compare with Svelte

In the section “Decision” remained the wording “as in Svelte“The templates use syntax. Svelte 5"I understand the official parser. Svelte“The layout is written as a component Svelte 5“All of framework written in Go.” They are true, but now read as a comparison: templates and there is Svelte. Without your consent I do not change them, only added in italics, what has changed 22 November 2026 years.

- (a) leave as is, with notes;
- b) rewrite without comparison, without changing the meaning and dates: "What brings out expressions - as in Svelte"  "Expressions bring out Svelte";
- c) points about templates in [Templates](templates.md#decisions) to be reduced into one - "Chambles are components of Svelte 5; they are compiled and rendered by Svelte sewn into Shugo (22 November 2026 year)" - and the former move to history.

**The Council:** b). Items with the words "values" and "expressions" in [Rantime](../docs/runtime.md#decisions) and [Assembly Sequences](../docs/build-pipeline.md#decisions) should be left: they are read as "saved" and "calculated".

### 3. The word "template"

`.svelte` files are components of Svelte, and the documents are called their templates: the document "Templates", "templates" in the memo for agents.

- a) leave the "template" as a common word for any `.svelte` file of the site - page, layout, component; "component" - the reusable part;
- b) say "component Svelte" and rename "templates".

**The Council:** a).

### 4. Two “Soluted” items in Blocks that are not now true to the end

- “Blocks are only calculated if the page reaches it.” The snippet given to the shared block is rendered even if the block does not show it now: so the place record is ready when the shared block after changing its data begins to show it ([43](decisions.md#43-render-shared-block-snippets-separately)). This was the case before 22 in November.
- "Same blocks do not repeat". Common blocks are not repeated; HTML component, which is rendered in place, is repeated in each page where it stands.

Options:

- (a) to add in italics the clarification as to the item “Unit of renewal”;
- b) render the snippet only when the shared block shows it. Then the change of the shared block data counts all the places where it stands.

**The Council:** a).

### 5. `<svelte:options runes={false} />`

With it, the component is compiled as Svelte 4: `export let` and `$:` work, although the templates are rune-only ([41](decisions.md#41-rune-only-templates)).

- a) error Shugo with a hint;
- b. Allow and describe.

**The Council:** a).

### 6. `<svelte:options css="injected" />`

The styles of this component go to `<style>` in `<head>` of each page where it is shown, past the overall style file. It is claimed that all styles are in one file, and critical styles in the page are postponed.

- a) allow as an explicit choice the author of the site and add this exception to the "Decision" in [Styles](styles.md#decisions);
- b. Error with a hint.

**The Council:** a): This is a clear choice, and it works right.

### 7. `<svelte:boundary>` on static page

With the `failed` snippet, the error is not caught inside: the assembly stops with this error. With the `pending` snippet, the static page forever shows the `pending`.

- a) leave: error is visible immediately;
- b) catch the error and show `failed`, with a warning during assembly;
- c) Warn about `pending` outside the islands.

**The Council:** (a) and (c).

### 8. `await` in `<script module>`

At the top level `<script module>` `await` works: the module waits for loading, until the first page. In `<script>` this is an error, and the documents are advised to wait only in modules from `src/lib/`.

- (a) Authorize and describe;
- b is banned.

**The Council:** a).

### 9. Layout imported by page

It is claimed that the layout cannot be called, but the assembly does not check this: the page can import `./+layout.svelte` and show it, and then the layout comes out on the page twice.

- a) Error when importing `+layout.svelte`;
- b. Remove the restriction.

**The Council:** a).

### 10. Components of `.svelte` from packages npm

The component from the package, for example, the interface library, is not compiled now. The page falls on Bun, the process falls on Node, there are no hints. SvelteKit compiles such libraries.

- a) compile them stitched with Svelte;
- b) build error with a hint.

**The Council:** b) immediately, a) when you need component libraries.

### 11. Module with runes among pages

`src/pages/shop/cart.svelte.ts` near the island becomes a `/shop/cart.svelte` site file and falls with "must export GET": [37](decisions.md#37-files-not-in-html-module-with-get) and [48](decisions.md#48-modules-with-runes) collide.

- a) Consider `*.svelte.ts` and `*.svelte.js` in `src/pages/` as import code, such as `.d.ts` and files with `_`;
- b. Give and give a hint.

**The Council:** a).

### 12. Svelte for Islands

Pages render Svelte, sewn in Shugo, and the island needs a package of `svelte@5.59.1` in the project: it takes the browser part of Svelte. These are two copies of one version, and the agent can put the wrong one.

- (a) As it is now;
- b) sew and browser part of Svelte, so that the project with the islands also do not put anything.

**The Council:** b).

### 13. Clear messages instead of naked errors

These cases now give a message that does not understand what to fix:

- `set:html={…}` from Astro is only a warning, and the attribute hits the page; `class:list={[…]}` silently puts the class `list`;
- marking inside `{…}`, `{cond && <b>…</b>}`, — `Unexpected token`;
- type imported without `type` `No matching export`;
- the component on the spot that endlessly shows itself is `Maximum call stack size exceeded` without a file and string; [23](decisions.md#23-small) promises a clear error, and it only has common blocks.

Options:

- a) its Shugo prompts for these cases, as for `await`;
- b) describe in the memo and leave messages as is.

**The Council:** a).

### 14. `shugo check`

It is decided to check the types and parameters of components by a separate command. Now `svelte-check` can do it.

- a) `shugo check` launches `svelte-check` with types from `_build/` ([question 1](#1-types-for-editor)); the project then needs `svelte-check` and `typescript`;
- b is for verification.

**The Council:** a).

### 15. Pictures: component or tag

`<enhanced:img>` in SvelteKit is not a component, but a tag that the plugin rewrites in the source to the Svelte compiler. Without this step, Svelte takes it to the page as is, and it only takes files known during assembly, not addresses from the base. [Proposal](../docs/proposals/images.md)

- a) its Svelte component from Shugo and the function of `image()` from the `shugo` module;
- b) `<enhanced:img>` record that Shugo rewrites to Svelte.

**The Council:** a): The template is compiled as written and addresses from the data work.

### 16. Pictures: where does export know addresses `/_img/…`

- a) Export looking for them in HTML, which writes;
- b) The update keeps a list of addresses along with each record.

**The Council:** a) and lists - if `shugo images clean` becomes slow.

### 17. Live Reload: "Styles Only"

Svelte puts the class `svelte-…` only on elements that selects some selector. Therefore, the new selector in `<style>` changes HTML, and the other color does not. [Proposal](../docs/proposals/live-reload.md)

- a) without page reloading when the code of the component that gave Svelte has not changed; otherwise it is a change in code;
- b) Any edit of `<style>` resets the page.

**The Council:** a).

### 18. Internationalization: `t` in `.ts` modules

The calculated key `t(…)` in the assembly template finds Svelte compiler tree; `.ts` modules do not have such tree. [Proposal](../docs/proposals/i18n.md)

- a) Allow `t` in modules without this check: the missing key will find `t` itself when it reads the row table;
- b) disassemble the module code separately;
- c) Prohibit `t` in modules.

**The Council:** a).
