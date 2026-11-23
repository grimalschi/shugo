package test

import (
	"strings"
	"testing"

	"github.com/grimalschi/shugo/internal/compile"
)

// The packages of npm are bundled with the code of the site: Svelte in
// them is the one Shugo has inside, whatever node_modules holds, a package
// written for CommonJS finds require and __dirname on Node too, and every
// file of a package in the bundle is a source of the build.
func TestPackagesAreBundled(t *testing.T) {
	s := newSite(t, `
-- node_modules/shout/package.json --
{"name": "shout", "type": "module", "main": "index.js"}
-- node_modules/shout/index.js --
export const shout = (text) => text.toUpperCase();
-- node_modules/legacy/package.json --
{"name": "legacy", "main": "index.js"}
-- node_modules/legacy/index.js --
const path = require("node:path");
exports.folder = () => typeof __dirname === "string" && path.isAbsolute(__dirname);
-- node_modules/theme/package.json --
{"name": "theme", "type": "module", "main": "index.js", "peerDependencies": {"svelte": "^5.0.0"}}
-- node_modules/theme/index.js --
import { getContext } from "svelte";
import { writable, get } from "svelte/store";
const count = writable(3);
export const theme = () => getContext("theme") ?? "none";
export const counted = () => get(count);
-- node_modules/svelte/package.json --
{"name": "svelte", "version": "4.2.0", "type": "module", "exports": {".": "./index.js", "./store": "./index.js"}}
-- node_modules/svelte/index.js --
throw new Error("the Svelte of node_modules runs");
-- src/pages/index.svelte --
<script>
import { setContext } from "svelte"
import { shout } from "shout"
import { folder } from "legacy"
import Theme from "../components/Theme.svelte"
setContext("theme", "dark")
</script>
<p>{shout("jug")} {folder()}</p><Theme />
-- src/components/Theme.svelte --
<script>
import { theme, counted } from "theme"
</script>
<i>{theme()} {counted()}</i>
`)
	s.build()
	expect(t, s.page("/"), "<p>JUG true</p><i>dark 3</i>")
	if changed, err := compile.Changed(s.project()); err != nil || changed {
		t.Fatalf("fresh build: changed = %v, error = %v", changed, err)
	}
	s.write("-- node_modules/shout/index.js --\nexport const shout = (text) => text + \"!\";")
	if changed, err := compile.Changed(s.project()); err != nil || !changed {
		t.Fatalf("a file of a package changed: changed = %v, error = %v", changed, err)
	}
}

// What the build cannot take of a package is told at the import, with
// what to do.
func TestPackageMistakes(t *testing.T) {
	const page = "-- src/pages/index.svelte --\n<script>\nimport { format } from \"date-fns\"\n</script>\n<p>{format()}</p>\n"
	cases := []struct{ name, files, want string }{
		{"a package that is not installed", page + "-- bun.lock --\n{}", `src/pages/index.svelte:2:24: date-fns is not installed: bun add date-fns`},
		{"a package that npm installs", page + "-- package-lock.json --\n{}", `date-fns is not installed: npm install date-fns`},
		{"a package for Svelte 4", page + "-- node_modules/date-fns/package.json --\n{\"name\": \"date-fns\", \"main\": \"index.js\", \"peerDependencies\": {\"svelte\": \"^3.0.0 || ^4.0.0\"}}\n-- node_modules/date-fns/index.js --\nexports.format = () => \"\"",
			"date-fns needs Svelte ^3.0.0 || ^4.0.0, and Shugo is made for Svelte 5.59.1: take a version of date-fns made for Svelte 5"},
		{"native code in a package", page + "-- node_modules/date-fns/package.json --\n{\"name\": \"date-fns\", \"main\": \"index.js\"}\n-- node_modules/date-fns/index.js --\nexports.format = require(\"./build/format.node\").format\n-- node_modules/date-fns/build/format.node --\nnot javascript",
			`No loader is configured for ".node" files: node_modules/date-fns/build/format.node: the package date-fns brings a file that is not JavaScript`},
		{"a package that reads its own files", page + "-- node_modules/date-fns/package.json --\n{\"name\": \"date-fns\", \"main\": \"index.js\"}\n-- node_modules/date-fns/index.js --\nconst words = require(\"node:fs\").readFileSync(__dirname + \"/words.txt\", \"utf8\")\nexports.format = () => words\n-- node_modules/date-fns/words.txt --\nwords",
			"the package date-fns reads a file next to its own code, and its code is bundled with the code of the site into _build/eval.mjs, where that file is not"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := newSite(t, c.files).tryBuild()
			if err == nil {
				t.Fatal("no error")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("\n got: %v\nwant: %s", err, c.want)
			}
		})
	}
}

// The code of the site may wait at the top of <script module> and of a
// module: it waits once, when the code is loaded, before any page is
// computed. In <script> and in markup a page does not wait: that is a
// mistake of its own.
func TestModulesWaitAtTheirTop(t *testing.T) {
	s := newSite(t, `
-- src/lib/words.ts --
export const words: string[] = await Promise.resolve(["jug", "cup"])
-- src/lib/count.svelte.js --
const start = await Promise.resolve(2)
export const counter = $state({ value: start })
-- src/pages/index.svelte --
<script module>
	const title = await Promise.resolve("Shop")
</script>
<script>
	import { words } from "@/lib/words"
	import { counter } from "@/lib/count.svelte.js"
</script>
<h1>{title}</h1><p>{words.join(" ")} {counter.value}</p>
-- src/pages/feed.json.ts --
const items = await Promise.resolve(["a"])
export function GET() { return JSON.stringify(items) }
`)
	s.build()
	expect(t, s.page("/"), "<h1>Shop</h1><p>jug cup 2</p>")
	expect(t, s.page("/feed.json"), `["a"]`)
}
