<script lang="ts">
  type Event = { date: string; time: string; title: string; cancelled?: boolean };
  let { events, formatDay }: { events: Event[]; formatDay: (iso: string) => string } = $props();

  const byString = (a: string, b: string) => (a < b ? -1 : a > b ? 1 : 0);
  // The sort is stable: events with the same date and time keep their order.
  const days = $derived(
    Map.groupBy([...events].sort((a, b) => byString(a.date, b.date) || byString(a.time, b.time)), (event) => event.date),
  );
</script>

{#if events.length === 0}
  <p class="empty">No events</p>
{:else}
  <div class="agenda">
    {#each [...days] as [date, list] (date)}
      <section class="day" data-date={date}>
        <h3>{formatDay(date)}</h3>
        <ul>
          {#each list as event}
            <li class={{ cancelled: event.cancelled }}>
              <time datetime="{date}T{event.time}">{event.time}</time>
              {#if event.cancelled}<s>{event.title}</s>{:else}{event.title}{/if}
            </li>
          {/each}
        </ul>
      </section>
    {/each}
  </div>
{/if}
