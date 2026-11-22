<script lang="ts">
  type Props = { year: number; month: number; today?: number; events: Record<string, number> };
  let { year, month, today, events }: Props = $props();

  const weekdays = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];
  const weeks = $derived.by(() => {
    const offset = (new Date(year, month - 1, 1).getDay() + 6) % 7; // Monday is 0
    const days = new Date(year, month, 0).getDate();
    // The cells of the month, Monday first: a day, or null for an empty cell.
    const cells: (number | null)[] = [...Array(offset).fill(null), ...Array.from({ length: days }, (_, i) => i + 1)];
    while (cells.length % 7 !== 0) cells.push(null);
    return Array.from({ length: cells.length / 7 }, (_, w) => cells.slice(w * 7, w * 7 + 7));
  });
</script>

<table class="calendar">
  <caption>{String(month).padStart(2, "0")}.{year}</caption>
  <thead>
    <tr>
      {#each weekdays as name, i}<th class={{ weekend: i >= 5 }}>{name}</th>{/each}
    </tr>
  </thead>
  <tbody>
    {#each weeks as week}
      <tr>
        {#each week as day, i}
          {#if day === null}
            <td class="empty"></td>
          {:else}
            {@const count = events[day] ?? 0}
            <td
              class={["day", { weekend: i >= 5, today: day === today, "has-events": count > 0 }]}
              aria-current={day === today ? "date" : undefined}
            >{day}{#if count > 0}<span class="badge">{count}</span>{/if}</td>
          {/if}
        {/each}
      </tr>
    {/each}
  </tbody>
</table>
