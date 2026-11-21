<script lang="ts">
  import type { Snippet } from "svelte"

  interface Props {
    href: string
    kind?: "primary" | "quiet"
    children: Snippet
  }
  let { href, kind = "primary", children }: Props = $props()
  const look = kind === "primary" ? "bg-brand text-white hover:bg-orange-800" : "text-brand underline"
</script>

<a {href} class={`inline-block rounded-lg px-4 py-2 font-medium ${look}`}>{@render children()}</a>
