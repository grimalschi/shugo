<script lang="ts">
  type Props = {
    label: string;
    variant: "primary" | "secondary" | "danger" | "ghost";
    size?: "sm" | "md" | "lg";
    href?: string;
    icon?: string;
    loading?: boolean;
    disabled?: boolean;
  };
  let { label, variant, size = "md", href, icon, loading = false, disabled = false }: Props = $props();
  const isLink = $derived(Boolean(href) && !disabled);
</script>

<svelte:element
  this={isLink ? "a" : "button"}
  class={["btn", `btn-${variant}`, `btn-${size}`, { "is-loading": loading, "is-disabled": disabled }]}
  href={isLink ? href : undefined}
  type={isLink ? undefined : "button"}
  {disabled}
  aria-busy={loading ? "true" : undefined}
>
  {#if icon}<i class="icon icon-{icon}" aria-hidden="true"></i>{/if}
  <span class="label">{label}</span>
</svelte:element>
