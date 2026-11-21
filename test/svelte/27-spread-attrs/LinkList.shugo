<script lang="ts">
  type Attrs = { [name: string]: string };
  type Item = { text: string; href?: string; current?: boolean; attrs?: Attrs };
  let { items, attrs, extraClass }: { items: Item[]; attrs?: Attrs; extraClass?: string } = $props();
</script>

<ul {...attrs} class={["list", extraClass]}>
  {#each items as item}
    <li {...item.attrs} class={["list-item", { "is-current": item.current }]}>
      {#if item.href}<a href={item.href} aria-current={item.current ? "page" : undefined}>{item.text}</a>{:else}{item.text}{/if}
    </li>
  {/each}
</ul>
