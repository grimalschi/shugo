export function searchParams(filters) {
  const params = new URLSearchParams();
  if (filters.q) params.set("q", filters.q);
  for (const category of filters.category) params.append("category", category);
  for (const size of filters.size) params.append("size", size);
  if (filters.stock) params.set("stock", "1");
  if (filters.priceFrom != null) params.set("price_from", filters.priceFrom);
  if (filters.priceTo != null) params.set("price_to", filters.priceTo);
  if (filters.sort !== "popular") params.set("sort", filters.sort);
  return params;
}
export function readFilters(params) {
  const number = (name) => {
    const value = params.get(name);
    return value !== null && value !== "" && Number.isFinite(Number(value)) ? Number(value) : null;
  };
  return {
    q: params.get("q")?.trim() ?? "",
    category: params.getAll("category"),
    size: params.getAll("size"),
    stock: params.get("stock") === "1",
    priceFrom: number("price_from"),
    priceTo: number("price_to"),
    sort: params.get("sort") || "popular",
  };
}
