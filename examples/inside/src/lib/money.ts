// Making a formatter is slow, so there is one for all prices.
const dollars = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD", minimumFractionDigits: 0, maximumFractionDigits: 3 })

export function price(value: number): string {
  return dollars.format(value)
}
