<script>
  import { search } from "shugo:search"
  import Search from "../components/Search.island.svelte"
</script>

<svelte:head><title>Articles</title></svelte:head>

<h1>Articles</h1>
<Search
  request:initial={search("articles", {
    text: ["title"],
    fields: ["title", "section", "minutes", "published"],
    filters: {
      category: { column: "section", type: "multi" },
      price: { column: "minutes", type: "range" },
      available: { column: "published", type: "boolean" },
    },
    sorts: {
      short: { by: [{ column: "minutes", direction: "asc" }], label: "Shortest first" },
      long: { by: [{ column: "minutes", direction: "desc" }], label: "Longest first" },
    },
    defaultSort: "short",
    pageSize: 2,
  })}
  action="/articles"
  groupLabel="Section"
  rangeLabel="Reading time"
  flagLabel="Published"
/>
