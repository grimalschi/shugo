# Page scripts

The document describes the code that is executed in the visitor's browser.

## Decisions

So far, nothing. Script processing is related to what is done immediately after the base.

## Made in the first version, waiting for approval

```
<button id="menu">Menu</button>
<script>
  document.getElementById("menu").addEventListener("click", () => { … })
</script>
<script src="/js/slider.js" defer></script>
```

- The contents of `<script>` fall into the page as written. The brackets inside it are plain text, expressions are not calculated.
- The script files are put into `public/` and connected to the address.
- A component with `<script>` shown ten times will give ten identical scripts.

The value from the data is transferred to the script via the attribute:

```
<div id="map" data-lat={shop.lat} data-lon={shop.lon} data-places={JSON.stringify(places)}></div>
<script>
  const map = document.getElementById("map")
  const places = JSON.parse(map.dataset.places)
</script>
```

## Open questions

As in Astro, `<script>` processing may include:

- TypeScript and imports within `<script>`;
- assembly of all site scripts into files with a hash in the name;
- one instance of the script per page, no matter how many times the component is shown;
- `<script is:inline>` for the script to be left as is;
- transfer values from code to script (in Astro it is `define:vars`).

The assembler already exists: the project styles and code are collected by esbuild. You need to decide which of these rules to take.
