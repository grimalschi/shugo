// What the pages of the shop do with prices. The data itself is in the
// database: the pages ask for it with Shugo.query.

// Making a formatter is slow, so there is one for all prices.
const dollars = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD", minimumFractionDigits: 0, maximumFractionDigits: 3 })

export function price(value: number): string {
  return dollars.format(value)
}

// The price after the discount, if there is one.
export function finalPrice(product: { price: number; discount: number | null }): number {
  return product.discount ? Math.round(product.price * (100 - product.discount) / 100) : product.price
}
