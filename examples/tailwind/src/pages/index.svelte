<script>
  import Button from "../components/Button.svelte"

  const features = ["Tailwind classes directly in the templates", "Custom colors with @theme", "Custom rules with @apply"]
</script>

<svelte:head><title>Shugo and Tailwind</title></svelte:head>
<h1 class="text-3xl font-bold text-brand">Shugo and Tailwind</h1>
<ul class="mt-4 list-disc pl-6">
  {#each features as feature}<li>{feature}</li>{/each}
</ul>
<div class="prose-lite"><p>Tailwind builds the styles from the project directory.</p></div>
<p class="mt-6 flex gap-3">
  <Button href="/about">Learn more</Button>
  <Button href="/" kind="quiet">Home</Button>
</p>
