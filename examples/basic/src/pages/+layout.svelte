<script lang="ts">
  import Header from "../components/Header.svelte"
  import "../styles/main.css"

  let { children } = $props()
</script>

<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
  </head>
  <body>
    <Header />
    <main>{@render children()}</main>
    <footer>© 2026</footer>
  </body>
</html>
