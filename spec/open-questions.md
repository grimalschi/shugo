# Open questions

Questions are numbered: answer the numbers. Where there is advice, yes - to accept it. Decisions taken without confirmation are collected in [Decisions awaiting approval](decisions.md); proposals that do not yet exist are in [docs/proposals](../docs/proposals/README.md).

### 1. Types for editor

The editor does not know the `shugo` module: on `import { query, page, search } from "shugo"` type check swears. If you put type announcements in the project once, after updating Shugo they will become obsolete.

So this problem is solved SvelteKit and Astro: framework writes types to your build folder every time you run `.svelte-kit/` and `.astro/`and the project once refers to it: `tsconfig.json` SvelteKit Heir `.svelte-kit/tsconfig.json`Astro is connected. `.astro/types.d.ts`. Types are always the version that collects the site.

For Shugo, this is what it looks like. Each build writes `_build/types/shugo.d.ts` — the `shugo` module ads that Shugo builds — and `_build/tsconfig.json`: where to look for `shugo`, which means `@/`, what settings TypeScript needs for Svelte. `tsconfig.json` is one line that does not change: `{ "extends": "./_build/tsconfig.json" }`; it is written by `shugo init`, which is decided to do. Updated Shugo the next build rewrote the types.

Another way is the `shugo` package in npm with some types. Its version would have to be updated along with Shugo, and the assembly would check that they match.

**The Council:** types write assembly in `_build/`, like SvelteKit and Astro.

### 2. Checks Svelte

The templates are parsed by the parser Svelte, but the page is compiled by the compiler Shugo: from the tree Svelte it makes the calculation code and framework, and HTML writes Go. Therefore, there are differences from Svelte, which in [Templates](templates.md#how-shugo-differs-from-svelte): the style class is on all elements of the component, `<svelte:head>` works only in the page and layout, there are no warnings about the availability and checks of runes. The full Svelte compiler is already in the assembly: its analysis of Shugo and uses.

**The Council:** will run each template through the Svelte build test itself — its warnings and errors will be the same as in Svelte — and the style class will be put on those elements that CSS selects. Then only hydration marks will remain from the differences, which are not needed for a static page.
