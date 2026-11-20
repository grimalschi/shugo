// The data of the shop. It is read once, when the pages are computed. Only
// scripts import this module: a template is given what its script took from
// here, and what templates call lives in money.ts, which loads nothing.
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
