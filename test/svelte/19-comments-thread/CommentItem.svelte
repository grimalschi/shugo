<script lang="ts" module>
  export type Comment = { id: number; author: string; text: string; deleted?: boolean; replies: Comment[] };
  export const count = (list: Comment[]): number => list.reduce((sum, c) => sum + 1 + count(c.replies), 0);

  function word(n: number) {
    return n === 1 ? "reply" : "replies";
  }
</script>

<script lang="ts">
  import CommentItem from "./CommentItem.svelte";
  let { comment, depth }: { comment: Comment; depth: number } = $props();
  const hidden = $derived(count(comment.replies));
</script>

<li id="comment-{comment.id}" class={["comment", comment.deleted && "deleted"]} style:margin-left={depth > 0 ? `${depth * 20}px` : undefined}>
  {#if comment.deleted}
    <p class="text">Comment deleted</p>
  {:else}
    <p class="author">{comment.author}</p>
    <p class="text">{comment.text}</p>
  {/if}
  {#if comment.replies.length > 0}
    {#if depth < 3}
      <ul class="replies">
        {#each comment.replies as reply}<CommentItem comment={reply} depth={depth + 1} />{/each}
      </ul>
    {:else}
      <a class="more" href="/comments/{comment.id}">{hidden} more {word(hidden)}</a>
    {/if}
  {/if}
</li>
