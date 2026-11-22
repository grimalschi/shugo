<script lang="ts">
  type Author = { name: string; url?: string };
  let { authors }: { authors: Author[] } = $props();

  // The separator before the name number i. Spaces at the edges of a block are
  // dropped, so the separators are expressions, not text in the markup.
  const separator = (i: number) => (i === 0 ? "" : i === authors.length - 1 ? " and " : ", ");
</script>

{#snippet name(author: Author)}
  {#if author.url}<a href={author.url}>{author.name}</a>{:else}<b>{author.name}</b>{/if}
{/snippet}

<p class="byline">
  {#if authors.length === 0}
    No author listed
  {:else}
    {authors.length === 1 ? "By:" : "Authors:"}
    {#each authors as author, i}{separator(i)}{@render name(author)}{/each}
  {/if}
</p>
