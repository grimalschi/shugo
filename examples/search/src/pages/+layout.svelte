<script>
  let { children } = $props()
</script>

<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link rel="icon" href="data:,">
  </head>
  <body>
    <nav aria-label="Collections">
      <a href="/products">Products</a>
      <a href="/articles">Articles</a>
    </nav>
    <main>{@render children()}</main>
  </body>
</html>
