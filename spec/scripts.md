# Page scripts

The document describes the code that is executed in the visitor's browser.

## Decisions

- `<script>` inside the markup gets into the page as written; the curly brackets inside it are plain text (21 decision of November 2026 year).
- The basis for the code in the browser is Svelte (21 solution November 2026 year). Later on it will appear islands Svelte.

## Implemented, pending approval

The top `<script>` of the `.shugo` file is a component code, it is executed when calculating a page and does not get into the browser. The code for the browser is written in `<script>` inside the element or in `<svelte:head>`, as in Svelte:

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

## Open questions

- Svelte Islands: a component that works in the browser - with the same syntax as the rest of the project. As noted by the island, what gets into the browser and how it receives data.
- Assembly of scripts for the browser via esbuild: TypeScript and imports, files with hash in the name, one instance per page. If the islands appear, the issue may be solved by them.
