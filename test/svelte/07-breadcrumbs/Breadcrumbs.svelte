<script lang="ts">
  type Crumb = { label: string; href?: string };
  let { items }: { items: Crumb[] } = $props();

  const max = 20;
</script>

{#if items.length > 0}
  <nav class="breadcrumbs" aria-label="Breadcrumbs">
    <ol>
      {#each items as item, i}
        {@const last = i === items.length - 1}
        {@const cut = item.label.length > max}
        {@const link = !last && !!item.href}
        {#if i > 0}<li class="sep" aria-hidden="true">/</li>{/if}
        <li>
          <svelte:element
            this={link ? "a" : "span"}
            href={link ? item.href : undefined}
            aria-current={last ? "page" : undefined}
            title={cut ? item.label : undefined}
          >{cut ? item.label.slice(0, max - 1) + "…" : item.label}</svelte:element>
        </li>
      {/each}
    </ol>
  </nav>
{/if}
