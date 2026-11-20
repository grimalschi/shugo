export interface Product {
  slug: string
  title: string
  price: number
  discount?: number
  tags: string[]
}

const products: Product[] = [
  { slug: "jug", title: "Line Pitcher", price: 2400, discount: 10, tags: ["ceramics", "new"] },
  { slug: "cup", title: "Cup <white>", price: 600, tags: ["ceramics"] },
  { slug: "tray", title: "Tray", price: 1800, tags: [] },
]

export async function getProducts(): Promise<Product[]> {
  return products
}

export function formatPrice(value: number): string {
  return new Intl.NumberFormat("en-US", { style: "currency", currency: "USD", minimumFractionDigits: 0, maximumFractionDigits: 3 }).format(value)
}
