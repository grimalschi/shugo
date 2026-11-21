<script lang="ts">
  type Format = "text" | "money" | "percent";
  type Column = { key: string; label: string; format: Format; align?: "left" | "center" | "right" };
  type Value = string | number | null | undefined;
  type Row = Record<string, Value>;
  let { columns, rows, caption }: { columns: Column[]; rows: Row[]; caption?: string } = $props();

  const alignOf = (column: Column) => `align-${column.align ?? (column.format === "text" ? "left" : "right")}`;
  const missing = (value: Value): value is null | undefined | "" => value === undefined || value === null || value === "";
  const money = (value: number) => `$${value.toFixed(2)}`;
  function format(value: Value, column: Column) {
    if (missing(value)) return "—";
    if (column.format === "money") return money(Number(value));
    if (column.format === "percent") return `${Math.round(Number(value) * 100)}%`;
    return String(value);
  }
  const sum = (key: string) => rows.reduce((total, row) => total + (missing(row[key]) ? 0 : Number(row[key])), 0);
  const hasFooter = $derived(rows.length > 0 && columns.some((column) => column.format === "money"));
</script>

<table class="data">
  {#if caption}<caption>{caption}</caption>{/if}
  <thead>
    <tr>{#each columns as column}<th class={alignOf(column)} scope="col">{column.label}</th>{/each}</tr>
  </thead>
  <tbody>
    {#each rows as row}
      <tr>{#each columns as column}<td class={alignOf(column)}>{format(row[column.key], column)}</td>{/each}</tr>
    {:else}
      <tr><td class="empty" colspan={columns.length}>No data</td></tr>
    {/each}
  </tbody>
  {#if hasFooter}
    <tfoot>
      <tr>
        {#each columns as column, i}
          <td class={alignOf(column)}>{#if column.format === "money"}{money(sum(column.key))}{:else if i === 0}Total{/if}</td>
        {/each}
      </tr>
    </tfoot>
  {/if}
</table>
