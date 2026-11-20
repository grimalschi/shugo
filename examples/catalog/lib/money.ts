// Prices, for scripts and for templates alike. The module loads nothing,
// which is what lets a template call it.

// Making a formatter is slow, so there is one for all prices.
const dollars = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD", minimumFractionDigits: 0, maximumFractionDigits: 3 })

export function price(value: number): string {
  return dollars.format(value)
}

// The price after the discount, if there is one: the discount is a number
// of percent.
export function finalPrice(product: { price: number; discount: number | null }): number {
  return product.discount ? Math.round(product.price * (100 - product.discount) / 100) : product.price
}
