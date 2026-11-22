import { page } from "shugo"
import { products } from "../lib/shop"

// /sitemap.xml: a module among the pages gives a file that is not HTML, the
// text its function GET returns. page.url is the address of the file on
// the site named in shugo.json.
export function GET() {
  const addresses = ["/", "/about", ...products.map((product) => "/products/" + product.slug)]
  const urls = addresses.map((address) => `  <url><loc>${new URL(address, page.url)}</loc></url>`)
  return `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${urls.join("\n")}
</urlset>
`
}
