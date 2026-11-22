<script lang="ts">
  import Layout from "./Layout.svelte";
  type Post = { title: string; url: string };
  let { title, posts, related, note }: { title: string; posts: Post[]; related: Post[]; note?: string } = $props();
</script>

{#snippet aside()}
  <h2>Related</h2>
  <ul>{#each related as post}<li><a href={post.url}>{post.title}</a></li>{/each}</ul>
{/snippet}

{#snippet footer()}<p class="note">{note}</p>{/snippet}

<Layout {title} aside={related.length > 0 ? aside : undefined} footer={note ? footer : undefined}>
  {#if posts.length > 0}
    <ul class="posts">{#each posts as post}<li><a href={post.url}>{post.title}</a></li>{/each}</ul>
  {:else}
    <p>Nothing here yet</p>
  {/if}
</Layout>
