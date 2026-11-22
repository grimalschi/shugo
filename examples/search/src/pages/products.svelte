<script>
  import { search } from "shugo:search"
  import Search from "../components/Search.island.svelte"
</script>

<svelte:head><title>Products</title></svelte:head>

<h1>Products</h1>
<Search
  request:initial={search("products", {
    text: ["title"],
    fields: ["title", "category", "price", "available"],
    filters: { category: "multi", price: "range", available: "boolean" },
    sorts: { cheap: { price: "asc" }, expensive: { price: "desc" } },
    defaultSort: "cheap",
    pageSize: 2,
  })}
  action="/products"
  groupLabel="Category"
  rangeLabel="Price"
  flagLabel="In stock"
/>
