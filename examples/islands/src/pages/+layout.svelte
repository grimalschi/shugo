<script>
  import "../styles/main.css"
  let { children } = $props()
</script>

<header class="site-header">
  <div class="site-width">
    <a class="wordmark" href="/catalog">Measure<span>Ceramics</span></a>
    <nav aria-label="About the demo">
      <a href="/catalog">Catalog</a>
      <a href="/baseline">Static version</a>
      <a href="/multiple">Two islands</a>
    </nav>
  </div>
</header>
<main class="site-width">{@render children()}</main>
