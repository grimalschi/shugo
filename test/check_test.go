package test

import (
	"path/filepath"
	"strings"
	"testing"
)

// shugo check runs svelte-check with the settings of the package shugo: it
// finds mistakes of types in modules and in components, and nothing in a
// project without them.
func TestTypesCheckCommand(t *testing.T) {
	types, err := filepath.Abs(filepath.Join("..", "npm", "shugo"))
	if err != nil {
		t.Fatal(err)
	}
	// The project extends the settings of the package as a project that has
	// it installed does; the module shugo is found where the package is.
	s := newSite(t, `
-- tsconfig.json --
{
	"extends": "`+filepath.ToSlash(filepath.Join(types, "tsconfig.json"))+`",
	"compilerOptions": {
		"paths": { "@/*": ["./src/*"], "shugo": ["`+filepath.ToSlash(filepath.Join(types, "index.d.ts"))+`"] }
	}
}
-- src/lib/shop.ts --
import { query } from "shugo"

export const count = (): number => query("products").where({ available: true }).count()
-- src/pages/index.svelte --
<script lang="ts">
	import { page } from "shugo"
	import { count } from "@/lib/shop"
	const path: string = page.url.pathname
</script>
<p>{path} {count()}</p>
`)
	out, errs, status := shugo(t, "check", s.root)
	if status != 0 {
		t.Fatalf("a project without mistakes: %d\n%s\n%s", status, out, errs)
	}
	s.write(`
-- src/lib/shop.ts --
import { query } from "shugo"

export const count = (): number => query("products").whre({ available: true }).count()
-- src/pages/index.svelte --
<script lang="ts">
	import { page } from "shugo"
	const path: number = page.url.pathname
</script>
<p>{path}</p>
`)
	out, errs, status = shugo(t, "check", s.root)
	if status != 1 || !strings.Contains(out, "Property 'whre' does not exist") || !strings.Contains(out, "Type 'string' is not assignable to type 'number'") || !strings.Contains(errs, "svelte-check found mistakes in the project") {
		t.Fatalf("a project with mistakes: %d\n%s\n%s", status, out, errs)
	}
}
