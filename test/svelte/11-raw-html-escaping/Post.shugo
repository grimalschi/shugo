<script lang="ts">
  type Props = {
    title: string;
    cover?: string;
    author: { name: string; url?: string };
    lead?: string;
    body: string;
    tags: string[];
  };
  let { title, cover, author, lead, body, tags }: Props = $props();
</script>

<article class="post">
  <h1>{title}</h1>
  {#if cover}<img class="cover" src={cover} alt={title} loading="lazy" />{/if}
  <p class="byline">By: {#if author.url}<a href={author.url} rel="author">{author.name}</a>{:else}<span>{author.name}</span>{/if}</p>
  {#if lead}<p class="lead">{lead}</p>{/if}
  <div class="body">{@html body}</div>
  {#if tags.length > 0}
    <ul class="tags">
      {#each tags as tag}<li><a href="/tags/{encodeURIComponent(tag)}" rel="tag">#{tag}</a></li>{/each}
    </ul>
  {/if}
</article>
