<script>
  import { onMount } from "svelte"

  let { initial, action, groupLabel, rangeLabel, flagLabel } = $props()
  // svelte-ignore state_referenced_locally
  let result = $state.raw(initial)
  // svelte-ignore state_referenced_locally
  let query = $state(initial.query)
  // svelte-ignore state_referenced_locally
  let facets = $state(initial.facets)
  let sortControl = $state()
  let pending = $state(false)
  let failed = $state(false)
  let request

  onMount(() => {
    const restore = () => load(new URL(window.location.href))
    window.addEventListener("popstate", restore)
    return () => {
      request?.abort()
      window.removeEventListener("popstate", restore)
    }
  })

  $effect(() => {
    if (sortControl) sortControl.value = query.sort
  })

  async function load(url) {
    request?.abort()
    const controller = new AbortController()
    request = controller
    pending = true
    failed = false
    try {
      const response = await fetch(result.endpoint + url.search, { signal: controller.signal })
      if (!response.ok) throw new Error("Search request failed")
      const next = await response.json()
      if (controller.signal.aborted) return
      result = next
      query = next.query
      facets = next.facets
    } catch {
      if (!controller.signal.aborted) failed = true
    } finally {
      if (request === controller) pending = false
    }
  }

  function navigate(url) {
    if (!url.search) url.search = ""
    if (url.href !== window.location.href) history.pushState(null, "", url)
    load(url)
  }

  function submit(event) {
    event.preventDefault()
    const form = event.currentTarget
    const url = new URL(form.action)
    for (const [name, value] of new FormData(form)) {
      if (value !== "") url.searchParams.append(name, value)
    }
    navigate(url)
  }

  function follow(event) {
    if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return
    event.preventDefault()
    navigate(new URL(event.currentTarget.href))
  }
</script>

<form method="get" {action} onsubmit={submit}>
  <label>Search <input type="search" name="q" bind:value={query.q}></label>
  <fieldset>
    <legend>{groupLabel}</legend>
    {#each facets.category.options as option (option.value)}
      <label>
        <input type="checkbox" name="category" value={option.value} bind:checked={option.selected}>
        {option.label}
      </label>
    {/each}
  </fieldset>
  <label>{rangeLabel} from <input type="number" min="0" name="price_from" bind:value={query.filters.price.from}></label>
  <label>{rangeLabel} up to <input type="number" min="0" name="price_to" bind:value={query.filters.price.to}></label>
  <label><input type="checkbox" name="available" value="1" bind:checked={query.filters.available}>{flagLabel}</label>
  <label>Sort by
    <select name="sort" bind:this={sortControl}>
      {#each result.sorts as sort (sort.value)}
        <option value={sort.value} selected={sort.selected}>{sort.label}</option>
      {/each}
    </select>
  </label>
  <button type="submit">Search</button>
  <a href={action} onclick={follow}>Reset</a>
</form>

<section data-search-results aria-busy={pending}>
  <p data-search-total role="status">Results: {result.total}</p>
  {#if failed}<p role="alert">Search is temporarily unavailable.</p>{/if}
  {#if result.hasHits}
    <ul>
      {#each result.hits as hit (hit.id)}
        <li data-hit-id={hit.id}>{hit.title}</li>
      {/each}
    </ul>
  {:else}
    <p>No results found.</p>
  {/if}
</section>

{#if result.hasPages}
  <nav aria-label="Results pages">
    {#if result.hasPrev}<a href={result.prevHref} onclick={follow}>Previous</a>{/if}
    {#if result.hasNext}<a href={result.nextHref} onclick={follow}>Next</a>{/if}
  </nav>
{/if}
