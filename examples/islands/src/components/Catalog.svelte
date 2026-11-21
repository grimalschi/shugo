<script>
  import { onMount, tick } from "svelte"
  import ProductCard from "./ProductCard.shugo"
  import { pageHref, pageNumbers, readFilters, searchParams, sorts } from "../lib/catalog.js"

  let { initial, action, endpoint } = $props()
  const initialResult = () => initial
  let result = $state.raw(initialResult())
  let draft = $state(copyFilters(initialResult().filters))
  let ready = $state(false)
  let pending = $state(false)
  let failed = $state(false)
  let heading
  let request
  let revision = 0

  function copyFilters(filters) {
    return { ...filters, category: [...filters.category], size: [...filters.size] }
  }

  onMount(() => {
    const restore = () => navigate(new URL(window.location.href), false)
    window.addEventListener("popstate", restore)
    ready = true
    return () => {
      window.removeEventListener("popstate", restore)
      request?.abort()
    }
  })

  async function navigate(url, push, scrollToResults = false) {
    if (push && url.href !== window.location.href) {
      window.history.pushState(null, "", url)
    }
    request?.abort()
    const controller = new AbortController()
    request = controller
    const startedAt = revision
    draft = readFilters(url.searchParams)
    pending = true
    failed = false

    try {
      const dataURL = new URL(endpoint, window.location.href)
      dataURL.search = url.search
      const response = await fetch(dataURL, { signal: controller.signal })
      if (!response.ok) throw new Error(`Search returned ${response.status}`)
      const next = await response.json()
      if (controller.signal.aborted) return
      result = next
      // Preserve any text or price the visitor entered while this query ran.
      if (revision === startedAt) draft = copyFilters(next.filters)
      if (scrollToResults) {
        await tick()
        if (!controller.signal.aborted) {
          heading.focus({ preventScroll: true })
          heading.scrollIntoView({ block: "start" })
        }
      }
    } catch {
      if (!controller.signal.aborted) failed = true
    } finally {
      if (request === controller) {
        request = undefined
        pending = false
      }
    }
  }

  function submit(event) {
    event.preventDefault()
    const form = event.currentTarget
    const filters = readFilters(new URLSearchParams(new FormData(form)))
    const url = new URL(form.action)
    url.search = searchParams(filters).toString()
    if (url.href === window.location.href && !pending && !failed) return
    navigate(url, true)
  }

  function change(event) {
    if (event.target.type === "checkbox" || event.target.tagName === "SELECT") {
      event.currentTarget.requestSubmit()
    }
  }

  function follow(event) {
    if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return
    event.preventDefault()
    const url = new URL(event.currentTarget.href)
    if (url.href !== window.location.href || pending || failed) {
      navigate(url, true, Boolean(event.currentTarget.closest(".pagination")))
    } else {
      draft = copyFilters(result.filters)
    }
  }
</script>

<div class="catalog" data-catalog-ready={ready ? "" : undefined}>
  <form class="filters" method="get" {action} onsubmit={submit} onchange={change} oninput={() => revision++}>
    <label class="field">Search
      <input type="search" name="q" maxlength="120" placeholder="For example, cup" bind:value={draft.q}>
    </label>

    <fieldset>
      <legend>Categories</legend>
      <div class="options category-options">
        {#each result.facets.category as option (option.value)}
          <label><input type="checkbox" name="category" value={option.value} bind:group={draft.category}><span>{option.label}</span><small aria-label={option.count + " products"}>{option.count}</small></label>
        {/each}
      </div>
    </fieldset>

    <fieldset>
      <legend>Sizes</legend>
      <div class="options">
        {#each result.facets.size as option (option.value)}
          <label><input type="checkbox" name="size" value={option.value} bind:group={draft.size}><span>{option.label}</span><small aria-label={option.count + " products"}>{option.count}</small></label>
        {/each}
      </div>
    </fieldset>

    <label class="stock-option"><input type="checkbox" name="stock" value="1" bind:checked={draft.stock}> In stock only <small aria-label={result.facets.stock + " products"}>{result.facets.stock}</small></label>

    <div class="price-range">
      <label class="field">Minimum price<input type="number" name="price_from" min="0" step="1" placeholder="$" bind:value={draft.priceFrom}></label>
      <label class="field">Maximum price<input type="number" name="price_to" min="0" step="1" placeholder="$" bind:value={draft.priceTo}></label>
    </div>

    <label class="field">Sort by
      <select name="sort" bind:value={draft.sort}>
        {#each sorts as sort (sort.value)}<option value={sort.value}>{sort.label}</option>{/each}
      </select>
    </label>

    <div class="filter-actions">
      <button type="submit">Show results</button>
      <a href={action} onclick={follow}>Reset</a>
    </div>
  </form>

  <section class="results" data-catalog-results aria-labelledby="results-heading">
    <div class="results-header">
      <h2 id="results-heading" bind:this={heading} tabindex="-1"><span role="status" aria-atomic="true">Products: {result.total}</span></h2>
      <p class="pending" aria-live="polite">{pending ? "Updating…" : ""}</p>
    </div>

    {#if failed}
      <p class="error" role="alert">Could not load the results. The last set of results is still shown.
        <button type="button" onclick={() => navigate(new URL(window.location.href), false)}>Retry</button>
      </p>
    {/if}

    <div aria-busy={pending}>
      {#if result.total}
        <ul class="products">
          {#each result.hits as item (item.id)}<li><ProductCard {item} /></li>{/each}
        </ul>
      {:else}
        <p class="empty">No products match these filters.</p>
      {/if}
    </div>

    {#if result.pages > 1}
      <nav class="pagination" aria-label="Catalog pages">
        {#if result.page > 1}<a href={pageHref(action, result.filters, result.page - 1)} onclick={follow}>Previous</a>{/if}
        {#each pageNumbers(result.page, result.pages) as number}
          {#if number === null}<span aria-hidden="true">…</span>
          {:else if number === result.page}<span aria-current="page">{number}</span>
          {:else}<a href={pageHref(action, result.filters, number)} onclick={follow}>{number}</a>{/if}
        {/each}
        {#if result.page < result.pages}<a href={pageHref(action, result.filters, result.page + 1)} onclick={follow}>Next</a>{/if}
      </nav>
    {/if}
  </section>
</div>
