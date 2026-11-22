<script lang="ts">
  type Photo = { src: string; width: number; height: number; alt?: string; caption?: string };
  let { images, widths, sizes }: { images: Photo[]; widths: number[]; sizes: string } = $props();

  const srcsetOf = (image: Photo) =>
    widths.filter((w) => w <= image.width).map((w) => `${image.src}?w=${w} ${w}w`).join(", ");
</script>

{#snippet photo(image: Photo, first: boolean)}
  {@const srcset = srcsetOf(image)}
  <img
    src={image.src}
    srcset={srcset || undefined}
    sizes={srcset ? sizes : undefined}
    width={image.width}
    height={image.height}
    alt={image.alt ?? ""}
    loading={first ? "eager" : "lazy"}
    fetchpriority={first ? "high" : undefined}
    decoding={first ? undefined : "async"}
  />
{/snippet}

<div class="gallery">
  {#each images as image, i}
    {#if image.caption}
      <figure>{@render photo(image, i === 0)}<figcaption>{image.caption}</figcaption></figure>
    {:else}
      {@render photo(image, i === 0)}
    {/if}
  {:else}
    <p class="gallery-empty">No photos yet</p>
  {/each}
</div>
