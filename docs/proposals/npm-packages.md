# npm Packages

Suggestion on how the site code takes packages from npm. Svelte is sewn into Shugo, `svelte-check` and TypeScript for `shugo check` decided to sew too, and the project npm itself is not needed. But the site code can import the package: Markdown parsing, working with dates, schedule in the island. What to do then is here.

## Decisions

- Svelte is sewn into Shugo by one version - the compiler, server and browser parts; it is not put in the draft (the decision of the author of the project 22 November 2026).
- `shugo check` is a `svelte-check` with TypeScript sewn into Shugo (the decision of the author of the project 22 November 2026). Not yet done.
- The `.svelte` component of the npm package is still a bug with a hint; compile such components stitched with Svelte — when you need component libraries (the decision of the author of the project 22 November 2026 year).

## Like now

The package is put as in any project on JavaScript — `bun add date-fns` or `npm install date-fns` — and it lies in `node_modules/`. Next:

| Where imported | What does Shugo do |
| --- | --- |
| code of the page, layout, component, module from `src/lib/` | `_build/eval.mjs` package does not include: Bun or Node download it from `node_modules` when calculating |
| island | package is included in the island browser module and in its server version |
| styles | Tailwind collects styles from `node_modules` if it is there |

22 November 2026 year:

- **Package No** in `node_modules` - compilation passes, and page calculation falls with the message Bun or Node, without the file that imports the package, and without the prompt: `Cannot find package 'date-fns' from '…/_build/eval.mjs'`.
- **Package itself imports Svelte** — for example, storage on `svelte/store` — and does not see the sewn Svelte: the import of the project code assembly leads to the sewn Svelte, and the package is loaded past the assembly. Without the `svelte` package, the calculation in the project drops: `Cannot find module 'svelte/store'`. With it, the page has two copies of Svelte: `getContext` in the package does not see the page context and falls off `lifecycle_outside_component`.
- **Package Update** the Assembly Sees `package.json` and lock files `bun.lock`, `package-lock.json`, `pnpm-lock.yaml`, `yarn.lock`: after their change, the project is collected and calculated again. Before 22 November 2026 In the past, they were considered only when the project had islands, and updating the package that takes the page left the pages old without warning.[53](../../spec/decisions.md#53-files-of-the-package-manager--source-of-the-project)). Editing files inside `node_modules` I can't see the assembly.
- **The package reads the file or goes to the network** when calculating is data past the database. The usual update does not see it. Find `shugo update -check` and update, which calculated the page anew and received another HTML at the same base lines.
- **On the server** packages are not needed: when you request a page, JavaScript is not executed. You need them where `shugo build` and `shugo update` go.

## What needs to be decided

Two questions: who puts the packages and how they take the calculation of pages.

### Who offers packages

**a) Package project manager, as everywhere.** `package.json`, `node_modules` and lock file; puts `bun add` or `npm install`. Shugo only reads `node_modules`.

- Agents know this. The editor and `shugo check` find the package types in `node_modules` themselves.
- Bun or Node on the machine is always: they Shugo calculates pages. Bun package manager is built, with Node usually put npm.
- Batch managers write different lock files; Shugo counts any.

**b) Packages put Shugo.** `shugo add date-fns` downloads the package from the npm registry to its cache and writes its lock file; `node_modules` is not in the project.

- Neither npm nor `node_modules`. Shugo knows every version and does not run package installation scripts.
- This is its own package manager: version ranges, dependencies, packages for different systems. Big work, and still poorer than npm.
- The types of packages for the Shugo editor would have to be laid out by itself.
- The agent of such a command does not know and will write `npm install` out of habit.

**(c) Version in import.** `import { format } from "npm:date-fns@4.1.0"`, as in Deno; Shugo downloads the package when assembled and stores it.

- The version is in the import itself, `package.json` is not needed.
- The editor and TypeScript do not understand this without their settings; one library in two files can be two versions; projects on Svelte do not write like this.

**The Council:** a). Putting packages is the work of the package manager, and he already has it on the machine. Shugo adds something without which it is impossible to pack: one Svelte, understandable errors, tracking changes.

### How to calculate packets

**a) Past the assembly, as it is now.** `eval.mjs` Imported packages, Bun or Node They find them in `node_modules`. To make the package see sewn SvelteIts import `svelte` the Module Load Hook Bun the University of California Node Yours.

- Works with any package, including native `.node` modules and files that the package reads next to its code.
- Three ways of one rule "Svelte - sewn": hook Bun, hook Node and the collector for the islands.
- Package changes are visible only by blocking files. The missing package must be checked separately, otherwise it will be found only when calculating.

**b. Inside the assembly.** Packages are part of `eval.mjs` as they are now part of the islands.

- Importing `svelte` from the package leads to a stitched Svelte in the same way as in the islands and in the project code.
- Each package file that was included in the build is a source with a hash: the update of the package and even the revision in `node_modules` are visible with your hands exactly.
- Package not available — build error with file and import string.
- You can see what package `node:fs` or network takes: you can warn about data past the database when assembling.
- A package with `.node` modules or files next to the code will not be included in the build: it needs a hint error and, perhaps, a setting that leaves such a package out of the build.
- The build is longer, and the `eval.mjs` is larger by the size of the packages.

**The Council:** b). So one rule works in one way, and package changes are visible exactly like changes to the project code. Package components, when they come to them, also have to pass through the assembly.

### What Shugo Adds to the Package Manager

1. **One Svelte.** Package `svelte` of `node_modules` Shugo never takes: import `svelte` and `svelte/…` from anywhere leads to sewn Svelte.
2. The **The Svelte version is in the package.** package, which `peerDependencies` calls Svelte other than sewn-in, such as `svelte: ^4.0.0`, is an build error with both versions.
3. **No package.** Build error with file and import string and command: `date-fns is not installed: bun add date-fns`. The Shugo command is selected by project lock file: `bun.lock` - `bun add`, `package-lock.json` - `npm install`; there is none - by what it calculates.
4. **Change of packages.** Package Change - Code Change: The project is collected and calculated again as after the template has been corrected (drafter &apos; s decision on 21 November 2026: Code Change Computes the entire site over again).
5. **Components of packages.** While an error with a hint. When you need libraries of components - compile them sewn Svelte, as does SvelteKit.
6. **Data past the base.** A note for an agent says pages take data only from the database — and through packages too. Packages that the build sees with `node:fs` or the network are a warning.

### What is not in the offer

- Its package registry and other sewn libraries, except Svelte and what `shugo check` needs.
- Package installation scripts: they are started or not run by the package manager as configured.

## Open questions

- Who puts the packets and how they are calculated is [questions 4 and 5](../../spec/open-questions.md#4-who-puts-the-packages-of-npm).
- Tailwind is now a project package. Whether to embed it as Svelte or leave it as a package.
- Whether the `shugo add` team needs a short path to the package project manager, or the agent will have enough memo.
