<script lang="ts">
  import { query } from "shugo"
  import { finalPrice, price } from "../lib/catalog"

  // The card takes the number of a product, not the product: called with a
  // plain value, it is computed and kept once for every product, however many
  // pages show it. It asks for its product itself, so a product that changes
  // has its card computed, and none of the pages that show the card.
  //
  // The query names no columns. The card depends on the ones that are read
  // while it is computed: the address and the title here, the price and the
  // discount inside finalPrice. A change of any other column of the product
  // does not reach it.
  let { id }: { id: string } = $props()
  const item = query("products").get(id)
  if (!item) throw new Error(`there is no product ${id}`)
</script>

<a class="card" href={`/products/${item.slug}`}>
  <b>{item.title}</b>
  <span class="price">{price(finalPrice(item))}</span>
</a>
