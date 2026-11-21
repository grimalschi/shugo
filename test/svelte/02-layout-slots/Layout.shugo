<script lang="ts">
  import type { Snippet } from "svelte";
  let { title, children, aside, footer }: { title: string; children: Snippet; aside?: Snippet; footer?: Snippet } = $props();
</script>

<div class="layout">
  <header><h1>{title}</h1></header>
  <main>{@render children()}</main>
  {#if aside}<aside>{@render aside()}</aside>{/if}
  <footer>{#if footer}{@render footer()}{:else}<p>© Pottery Shop</p>{/if}</footer>
</div>
