<script lang="ts">
  type Line = { name: string; qty: number; price: number };
  let { lines }: { lines: Line[] } = $props();
  const total = $derived(lines.reduce((sum, line) => sum + line.qty * line.price, 0));
</script>

<table class="order">
  <thead>
    <tr><th>Item</th><th class="num">Qty</th><th class="num">Price</th><th class="num">Subtotal</th></tr>
  </thead>
  <tbody>
    {#each lines as line, i}
      <tr class={[i % 2 === 0 ? "odd" : "even", { zero: line.qty === 0 }]}>
        <td>{line.name}</td>
        <td class="num">{line.qty}</td>
        <td class="num">{line.price.toFixed(2)}</td>
        <td class="num">{(line.qty * line.price).toFixed(2)}</td>
      </tr>
    {:else}
      <tr><td class="empty" colspan="4">Your order is empty</td></tr>
    {/each}
  </tbody>
  {#if lines.length > 0}
    <tfoot>
      <tr><th colspan="3">Total</th><td class="num">{total.toFixed(2)}</td></tr>
    </tfoot>
  {/if}
</table>
