<script lang="ts">
  type Item = { question: string; answer: string; open?: boolean };
  let { items, title = "Frequently asked questions" }: { items: Item[]; title?: string } = $props();

  const paragraphs = (answer: string) =>
    answer.split(/\n\s*\n/).map((part) => part.trim()).filter(Boolean).map((part) => part.split("\n").map((line) => line.trim()));
  const anyOpen = $derived(items.some((item) => item.open));
</script>

<section class="faq">
  <h2>{title}</h2>
  {#each items as item, i}
    {@const answer = paragraphs(item.answer)}
    <details class="faq-item" open={item.open || (!anyOpen && i === 0)}>
      <summary>{i + 1}. {item.question}</summary>
      <div class="answer">
        {#each answer as lines}
          <p>{#each lines as line, j}{#if j > 0}<br />{/if}{line}{/each}</p>
        {:else}
          <p class="empty">No answer yet</p>
        {/each}
      </div>
    </details>
  {:else}
    <p class="empty">No questions yet</p>
  {/each}
</section>
