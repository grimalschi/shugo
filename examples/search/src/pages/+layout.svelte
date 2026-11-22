<script>
  let { children } = $props()
</script>

<nav aria-label="Collections">
  <a href="/products">Products</a>
  <a href="/articles">Articles</a>
  <a href="/counters">Two islands</a>
</nav>
<main>{@render children()}</main>
