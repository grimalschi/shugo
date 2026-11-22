<script>
  import { query } from "shugo"
  import { price } from "../lib/money"

  // A card takes the key of a product, a plain value: so it is a shared block,
  // computed once for every product, however many pages show it.
  let { id } = $props()
  const item = query("products").get(id)
</script>

<a class="card" href={`/products/${item.slug}`}>{item.title} — {price(item.price)}</a>
