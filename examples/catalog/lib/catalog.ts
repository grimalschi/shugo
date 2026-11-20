// The data of the shop. It is read once, when the pages are computed.
import { readFileSync } from "node:fs"

export interface Variant { sku: string; title: string; stock: number }
export interface Review { author: string; text: string; approved: boolean }
export interface Product {
  id: string
  slug: string
  title: string
  price: number
  discount: { percent: number; until: string } | null
  available: boolean
  arrival: string
  description: string
  category: string
  badges: string[]
  variants: Variant[]
  reviews: Review[]
  related: string[]
}
export interface Category { id: string; title: string }

const data = JSON.parse(readFileSync("data/catalog.json", "utf8"))

export const shop: { name: string; phone: string; featured: string | null } = data.shop
export const categories: Category[] = data.categories
export const products: Product[] = data.products

const byId = new Map(products.map((product) => [product.id, product]))

export function product(id: string): Product {
  const found = byId.get(id)
  if (!found) throw new Error(`there is no product ${id}`)
  return found
}

export function inCategory(id: string): Product[] {
  return products.filter((product) => product.category === id)
}

// Making a formatter is slow, so there is one for all prices.
const dollars = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD", minimumFractionDigits: 0, maximumFractionDigits: 3 })

export function price(value: number): string {
  return dollars.format(value)
}

// The price after the discount, if there is one.
export function finalPrice(product: Product): number {
  return product.discount ? Math.round(product.price * (100 - product.discount.percent) / 100) : product.price
}
