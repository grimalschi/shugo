# Proposals

Here lies what Shugo does not have yet: proposals that are waiting for the decision of the author of the project, and postponed. When the proposal is accepted and made, his decisions go to [specification](../../spec/README.md) - how to use Shugo - and [description of device](../README.md).

| Proposal | About what | Status |
| --- | --- | --- |
| [Image cache](image-cache.md) | Own project folder outside `_build/`, hash names, write without half files, clean | decided, waiting for pictures |
| [Live Reload](live-reload.md) | Development mode calculates the page on request, not the entire site; reboots only the changed, replaces styles, shows an error in the browser | page on request solved; rest waiting for decision; not done |
| [Data adapter](data-adapter.md) | How data from any source gets into the project database: full load with comparison, point changes, a scheme that signals, libraries that can only wait | waiting for a decision; advice is the next big step |
| [Pictures](images.md) | Built-in compression with WebP and, optionally, libvips for speed and AVIF; format, addresses, export | two modes and format are solved; the rest is waiting for the decision |
| [Internationalization](i18n.md) | One page for all languages, lines in the database, addresses `/about` and `/en/about`, optional parameter `[[lang]]` | lines in the database are solved; the rest is waiting for the decision |
| [Multiple JS processes for first build](parallel-build.md) | First Assembly Divides Calculation Between Processes by Number of Cores | postponed |
| [Comparisons and windows in readings](range-queries.md) | Accurate update of comparison queries and `limit`: Reading boundaries and R*Tree index | postponed |

Accepted and done 22 November 2026 years, so we left this folder: `query`, `paginate` and `page` from module `shugo` the National Museum [Templates](../../spec/templates.md#file-structure); shell `src/site.html` the National Museum [Layouts](../../spec/layouts.md#shell-srcsitehtml). Closed on the same day: compatibility with Svelte templates `.svelte` compile and render itself Svelte 5.59.1, TypeScript Written with `lang="ts"`And that little that Shugo does around Svelte In its own way, it is listed in [Templates](../../spec/templates.md#how-shugo-differs-from-svelte).

## Done and waiting for approval later

These solutions are already working; the author of the project postponed their approval of 22 on November 2026.

- **Breakdown into pages.** `paginate(number)` in `getStaticPaths` says what pages there are, `paginate(request)` in the page takes part of it; addresses `/blog`, `/blog/2` are [Dynamic pages](../../spec/dynamic-pages.md#pagination).
- **Service symbols in meanings.** Symbols with codes 0–4 in that JS Translate Go, c 22 November 2026 the Year HTML records screened, not cut; text reaches the page as it was [25](../../spec/decisions.md#25-service-symbols-in-values-are-shielded).
- **Details of `src/site.html` shell, files from modules and their export** — [36–38](../../spec/decisions.md#36-the-srcsitehtml-shell).
- **Warning thresholds for the common component**, which shares almost nothing: 100 blocks and one and a half places on average [3](../../spec/decisions.md#3-which-blocks-are-shared).
