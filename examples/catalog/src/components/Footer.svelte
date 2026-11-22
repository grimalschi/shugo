<script>
  import { query } from "shugo"
  const shop = query("shop").get(1)
</script>

<footer>{shop.name} · {shop.phone}</footer>
