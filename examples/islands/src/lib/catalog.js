const dollars = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD", minimumFractionDigits: 0, maximumFractionDigits: 3 });

export function money(value) {
  return dollars.format(value);
}

export const sorts = [
  { value: "popular", label: "Most popular" },
  { value: "cheap", label: "Lowest price first" },
  { value: "expensive", label: "Highest price first" },
  { value: "new", label: "Newest first" },
];

export function searchParams(filters, page = 1) {
  const params = new URLSearchParams();
  if (filters.q) params.set("q", filters.q);
  for (const category of filters.category) params.append("category", category);
  for (const size of filters.size) params.append("size", size);
  if (filters.stock) params.set("stock", "1");
  if (filters.priceFrom != null) params.set("price_from", filters.priceFrom);
  if (filters.priceTo != null) params.set("price_to", filters.priceTo);
  if (filters.sort !== "popular") params.set("sort", filters.sort);
  if (page > 1) params.set("page", page);
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

export function pageHref(action, filters, page) {
  const query = searchParams(filters, page).toString();
  return query ? action + "?" + query : action;
}

export function pageNumbers(page, pages) {
  const visible = [];
  for (const number of [...new Set([1, page - 1, page, page + 1, pages])].sort((a, b) => a - b)) {
    if (number < 1 || number > pages) continue;
    if (visible.length && number - visible.at(-1) > 1) visible.push(null);
    visible.push(number);
  }
  return visible;
}
