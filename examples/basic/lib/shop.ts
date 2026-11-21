export interface Product {
  slug: string
  title: string
  price: number
  discount?: number
  tags: string[]
}

// The data of a small site can live in its code: a change of it is a change
// of the project, and shugo compiles the project again.
export const products: Product[] = [
  { slug: "jug", title: "Line Pitcher", price: 2400, discount: 10, tags: ["ceramics", "new"] },
  { slug: "cup", title: "Cup <white>", price: 600, tags: ["ceramics"] },
  { slug: "tray", title: "Tray", price: 1800, tags: [] },
]

// Making a formatter is slow, so there is one for all prices.
const dollars = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD", minimumFractionDigits: 0, maximumFractionDigits: 3 })

export function formatPrice(value: number): string {
  return dollars.format(value)
}
