# Page scripts

The document describes the code that is executed in the visitor's browser.

## Decisions

- `<script>` inside the markup gets into the page as written; the curly brackets inside it are plain text (21 decision of November 2026 year).
- The `*.island.svelte` component becomes a hydration point when called from the Shugo template. The file name is visible in the source code and in the import.

## Implemented, pending approval

The top `<script>` of a regular `.svelte` file is performed when calculating the page. For small browser scripts, you can use `<script>` inside the element or in `<svelte:head>`:

```svelte
<button id="menu">Menu</button>
<div>
  <script>
    document.getElementById("menu").addEventListener("click", () => { … })
  </script>
</div>

<svelte:head>
  <script src="/js/slider.js" defer></script>
</svelte:head>
```

- The contents of such `<script>` fall into the page as written. Expressions in it are not calculated: `{x}` remains the text of `{x}`.
- The attributes of the tag itself are common attributes: `defer` is displayed as `defer=""`, `src={…}` is calculated.
- The script files are put into `public/` and connected to the address.
- A component with a script that is shown ten times will give ten identical scripts.
- Event handlers in markup (`onclick={…}`) do not get into the page, as on the Svelte server: there is no one to perform them on the static page.

The value from the data is transferred to the script via the attribute:

```svelte
<div id="map" data-lat={shop.lat} data-lon={shop.lon} data-places={JSON.stringify(places)}></div>
<div>
  <script>
    const map = document.getElementById("map")
    const places = JSON.parse(map.dataset.places)
  </script>
</div>
```

## Svelte islands

The page imports the island by name and passes it data as parameters. The file can contain standard runes, actions and handlers of Svelte. The assembly of the project with islands needs the `svelte@5.59.1` package.

```svelte
<script>
  import Catalog from "../components/Catalog.island.svelte"
  import { search } from "shugo:catalog"
</script>

<Catalog request:initial={search()} action="/catalog" endpoint="/_islands/catalog" />
```

The suffix selects the boundary of hydration. A subsidiary of `*.island.svelte` inside another island is included in its tree Svelte. The regular `ProductCard.svelte` can be imported into both the server template and the island if the component takes data through the parameters and uses only Svelte tools. `Shugo.query` is executed in a normal server template that transmits the result to the island.

`request:initial={search()}` binds the parameter `initial` to the output of URL. The `search()` call registers the connection for initial issuance during the calculation, including calls within components and `getStaticPaths`. The call when loading the module is registered for the entire site. The usual parameter `initial={search()}` receives the result of the calculation step.

When assembling Shugo prepares the program SSR from the output of Svelte. On request Go reads the published index and applies the program, including branches, lists and hydration tags. In the browser Svelte connects handlers to the received markup. [How SSR Island is going](../examples/islands/GO-SSR.md)

## Open questions

- Do you need a separate way to collect browser scripts with imports for pages without islands?
