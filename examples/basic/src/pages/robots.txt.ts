import { page } from "shugo"

export function GET() {
  return `User-agent: *
Allow: /

Sitemap: ${new URL("/sitemap.xml", page.url)}
`
}
