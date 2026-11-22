<script>
  import Header from "../components/Header.svelte"

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
  </body>
</html>
