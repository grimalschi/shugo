<script lang="ts">
  type Kind = "info" | "warning" | "error" | "success";
  type Notification =
    | { kind: "info" | "warning" | "error"; text: string }
    | { kind: "success"; text: string; link?: { href: string; label: string } };
  let { notifications }: { notifications: Notification[] } = $props();

  const labels: Record<Kind, string> = { error: "Errors", warning: "Warnings", success: "Successes", info: "Messages" };
  const icons: Record<Kind, string> = { info: "icon-info", warning: "icon-alert", error: "icon-cross", success: "icon-check" };
  const summary = $derived(
    (Object.keys(labels) as Kind[])
      .map((kind) => ({ kind, count: notifications.filter((n) => n.kind === kind).length }))
      .filter(({ count }) => count > 0),
  );
</script>

<section class="notifications">
  {#if notifications.length === 0}
    <p class="empty">No notifications</p>
  {:else}
    <ul class="summary">
      {#each summary as { kind, count }}<li class="summary-{kind}">{labels[kind]}: {count}</li>{/each}
    </ul>
    <ul class="list">
      {#each notifications as n}
        <li class={["note", `note-${n.kind}`]} role={n.kind === "error" ? "alert" : undefined}>
          <span class={["icon", icons[n.kind]]} aria-hidden="true"></span>
          {#if n.kind === "warning"}
            <p><strong>Attention:</strong> {n.text}</p>
          {:else if n.kind === "success" && n.link}
            <p>{n.text} <a href={n.link.href}>{n.link.label}</a></p>
          {:else}
            <p>{n.text}</p>
          {/if}
        </li>
      {/each}
    </ul>
  {/if}
</section>
