<script lang="ts">
  type Section = { level: number; title: string; slug?: string; text?: string };
  let { sections }: { sections: Section[] } = $props();
  const clamp = (level: number) => Math.min(6, Math.max(2, level));
</script>

<article class="doc">
  {#each sections as section}
    {@const level = clamp(section.level)}
    <svelte:element this={`h${level}`} class={["heading", `level-${level}`]} id={section.slug || undefined}>
      {section.title}
      {#if section.slug}<a class="anchor" href="#{section.slug}" aria-hidden="true">#</a>{/if}
    </svelte:element>
    {#if section.text}<p>{section.text}</p>{/if}
  {/each}
</article>
