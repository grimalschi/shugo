package test

import (
	"strings"
	"testing"
)

// When the whole site is computed, a component that is kept apart for
// nothing is told of: it has many shared blocks, and each is shown in about
// one place.
func TestSharingThatGainsNothing(t *testing.T) {
	s := newSite(t, `
-- src/pages/+layout.svelte --
<script>
import { page } from "shugo"
let { children } = $props()
</script>
<nav>{page.url.pathname}</nav>{@render children()}
-- src/pages/[n].svelte --
<script module>
import { page } from "shugo"
export function getStaticPaths() {
  return Array.from({ length: 120 }, (_, i) => ({ params: { n: String(i + 1) } }))
}
</script>
<script>
import Price from "../components/Price.svelte"
import Badge from "../components/Badge.svelte"
</script>
<Price value={Number(page.params.n) * 10} /><Badge kind="new" />
-- src/components/Price.svelte --
<script>
let { value } = $props()
</script>
<b>${value}</b>
-- src/components/Badge.svelte --
<script>
let { kind } = $props()
</script>
<i>{kind}</i>
`)
	report := s.build()
	// The price of every page is its own, the badge is one for all pages,
	// and the layout reads the page: it is kept for every page by design.
	if len(report.Warnings) != 1 || !strings.HasPrefix(report.Warnings[0], "src/components/Price.svelte is kept apart as 120 shared blocks, each shown in 1.0 places on average.") {
		t.Errorf("warnings: %q", report.Warnings)
	}
	expect(t, s.page("/7"), "<nav>/7</nav><b>$70</b><i>new</i>")
	// An update computes a part of the site, and weighs nothing.
	if report := s.update(); len(report.Warnings) != 0 {
		t.Errorf("an update warns: %q", report.Warnings)
	}
}
