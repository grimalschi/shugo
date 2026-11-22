package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The modules among the pages give files that are not HTML: sitemap.xml.ts
// gives /sitemap.xml, the text its function GET returns. What GET reads of
// the data is followed as for a page; a module whose name starts with _ is
// code for the others to import.
const fileModules = `
-- shugo.json --
{"data": "data/shop.db", "site": "https://shop.example"}
-- src/pages/products/[slug].svelte --
<script module>
import { query } from "shugo"
export function getStaticPaths() {
  return query("products").select("slug").all().map(({ slug }) => ({ params: { slug } }))
}
</script>
<script>
import { query, page } from "shugo"
const product = query("products").where({ slug: page.params.slug }).first()
console.log("page " + product.slug)
</script>
<h1>{product.title}</h1>
-- src/pages/sitemap.xml.ts --
import { page } from "shugo"
import { addresses } from "./_addresses"

export function GET() {
  console.log("sitemap")
  const urls = addresses().map((address) => "<url><loc>" + new URL(address, page.url) + "</loc></url>")
  return '<?xml version="1.0" encoding="UTF-8"?>\n<urlset>' + urls.join("") + "</urlset>"
}
-- src/pages/_addresses.ts --
import { query } from "shugo"

export function addresses(): string[] {
  return query("products").select("slug").order("slug").all().map((p) => "/products/" + p.slug)
}
-- src/pages/robots.txt.ts --
import { page } from "shugo"

export function GET() {
  return "User-agent: *\nSitemap: " + new URL("/sitemap.xml", page.url) + "\n"
}
-- src/pages/api/[slug].json.ts --
import { query, page } from "shugo"

export function getStaticPaths() {
  return query("products").select("slug").all().map(({ slug }) => ({ params: { slug } }))
}

export function GET() {
  const { slug } = page.params
  console.log("json " + slug)
  const { title, price } = query("products").where({ slug }).first()
  return JSON.stringify({ slug, title, price, address: page.url.pathname })
}
`

func TestFilesOfModules(t *testing.T) {
	s := newSite(t, fileModules)
	s.sql(queryRows)
	report := s.build()
	if report.Pages.Total != 10 {
		t.Errorf("pages: %d, want 4 of products, 4 of JSON, the sitemap and robots.txt", report.Pages.Total)
	}
	s.computed()
	// A file is the text GET returned, as it is: no shell, no escaping.
	sitemap := func(slugs ...string) string {
		out := `<?xml version="1.0" encoding="UTF-8"?>` + "\n<urlset>"
		for _, slug := range slugs {
			out += "<url><loc>https://shop.example/products/" + slug + "</loc></url>"
		}
		return out + "</urlset>"
	}
	expect(t, s.page("/sitemap.xml"), sitemap("cup", "jug", "tray", "vase"))
	expect(t, s.page("/robots.txt"), "User-agent: *\nSitemap: https://shop.example/sitemap.xml\n")
	expect(t, s.page("/api/jug.json"), `{"slug":"jug","title":"Jug","price":10,"address":"/api/jug.json"}`)
	if s.has("/_addresses") || s.has("/api/jug") || s.has("/sitemap") {
		t.Error("an address without the extension of the file, or of code to import")
	}

	// The server tells the type by the extension, as it does for the
	// public files; a page of templates is HTML.
	ts := serve(t, s)
	for address, want := range map[string]string{"/sitemap.xml": "text/xml", "/api/jug.json": "application/json", "/robots.txt": "text/plain", "/products/jug": "text/html"} {
		if status, header, _ := get(t, ts, address); status != 200 || !strings.HasPrefix(header.Get("Content-Type"), want) {
			t.Errorf("%s: %d %s, want %s", address, status, header.Get("Content-Type"), want)
		}
	}

	// A file is computed again when what it read changes, and only then:
	// the sitemap reads the addresses of the products, and the page of a
	// product shows no price.
	s.changed(`UPDATE products SET price = 11 WHERE slug = 'jug'`, "json jug")
	expect(t, s.page("/api/jug.json"), `{"slug":"jug","title":"Jug","price":11,"address":"/api/jug.json"}`)
	s.changed(`INSERT INTO products VALUES ('p5', 'bowl', 'Bowl', 4, 'c2')`, "json bowl", "page bowl", "sitemap")
	expect(t, s.page("/sitemap.xml"), sitemap("bowl", "cup", "jug", "tray", "vase"))

	// An export writes a file by its address, and removes the file of a
	// page that went away by its name.
	out := filepath.Join(s.root, "dist")
	s.export(out)
	exported := func(name string) bool {
		_, err := os.Stat(filepath.Join(out, filepath.FromSlash(name)))
		return err == nil
	}
	for _, name := range []string{"sitemap.xml", "robots.txt", "api/bowl.json", "products/bowl.html"} {
		if !exported(name) {
			t.Errorf("%s is not exported", name)
		}
	}
	s.changed(`DELETE FROM products WHERE slug = 'bowl'`, "sitemap")
	if done := s.export(out); done.Removed != 2 || exported("api/bowl.json") || exported("products/bowl.html") {
		t.Errorf("after the bowl is gone: %+v", done)
	}

	// An export after the project has changed builds every page; it knows
	// from its list which files the modules gave before.
	s.remove("src/pages/robots.txt.ts")
	s.build()
	if done := s.export(out); exported("robots.txt") || !exported("sitemap.xml") || done.Removed != 1 {
		t.Errorf("after robots.txt.ts is gone: %+v", done)
	}
}

// What a module among the pages does wrong is told where it is.
func TestFileMistakes(t *testing.T) {
	for _, c := range []struct{ name, files, want string }{
		{"a module without the type of its file", `
-- src/pages/data.ts --
export function GET() { return "x" }`, "src/pages/data.ts:1:1: a module among the pages gives a file of the site, and its name tells the type of that file before .ts: sitemap.xml.ts gives /sitemap.xml; a module of helpers goes to src/lib/, or its name starts with _"},
		{"a template and a module with one address", `
-- src/pages/feed.xml.svelte --
<p>x</p>
-- src/pages/feed.xml.ts --
export function GET() { return "x" }`, "give the same addresses"},
		{"no GET", `
-- src/pages/feed.xml.ts --
export function get() { return "x" }`, "src/pages/feed.xml.ts gives the file /feed.xml, so it must export the function GET, which returns the text of the file"},
		{"a GET that waits", `
-- src/pages/feed.xml.ts --
export async function GET() { return "x" }`, "GET of src/pages/feed.xml.ts waits (it is async or returns a promise)"},
		{"a GET that returns data", `
-- src/pages/feed.json.ts --
export function GET() { return { a: 1 } }`, "GET of src/pages/feed.json.ts must return the text of the file, and returned an object: JSON.stringify(…) makes one of data"},
		{"a GET that returns a Response", `
-- src/pages/feed.xml.ts --
export function GET() { return new Response("x") }`, "a Response is read by waiting, so return the text itself"},
		{"a GET that takes the params of Astro", `
-- src/pages/[id].txt.ts --
export const getStaticPaths = () => [{ params: { id: "a" } }]
export function GET({ params }) { return params.id }`, `GET gets no params: import { query, page } from "shugo" at the top of the module, and read page.params and page.url`},
		{"a module with a parameter and no list", `
-- src/pages/[id].txt.ts --
export function GET() { return "x" }`, "/[id].txt has parameters, so the module must export getStaticPaths"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := newSite(t, c.files).tryBuild()
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("got %v, want %q", err, c.want)
			}
		})
	}
}
