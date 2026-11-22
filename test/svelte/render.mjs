// Renders the reference components of test/svelte with Svelte itself and
// writes what it gives into expected.json of every task: the HTML Shugo must
// give for the same component and data. Both compilers read the same
// .svelte files. Remove Svelte's hydration marks for the static comparison.
//
//   cd test/svelte && bun install && bun render.mjs [task…]
//
// Shugo follows one version of Svelte: change it here and in package.json,
// render, and look at what differs in git.
import { plugin } from "bun";
import { readFileSync, readdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { VERSION, compile } from "svelte/compiler";
import { render } from "svelte/server";

const WANTED = "5.59.1";
if (VERSION !== WANTED) throw new Error(`the references are made with Svelte ${WANTED}, and this is ${VERSION}`);

plugin({
  name: "svelte",
  setup(build) {
    build.onLoad({ filter: /\.svelte$/ }, ({ path }) => {
      const { js } = compile(readFileSync(path, "utf8"), { generate: "server", filename: path });
      return { contents: js.code, loader: "js" };
    });
  },
});

const here = import.meta.dir;
const unmark = (html) => html.replace(/<!--[\s\S]*?-->|<!>/g, "");
const only = process.argv.slice(2);

for (const id of readdirSync(here, { withFileTypes: true }).filter((e) => e.isDirectory()).map((e) => e.name).sort()) {
  if (id === "node_modules" || (only.length > 0 && !only.some((part) => id.includes(part)))) continue;
  const dir = join(here, id);
  const task = JSON.parse(readFileSync(join(dir, "task.json"), "utf8"));
  const helpers = readdirSync(dir).includes("helpers.js") ? (await import(join(dir, "helpers.js"))).default : {};
  const component = (await import(join(dir, task.entry + ".svelte"))).default;
  const expected = task.datasets.map((data) => {
    const { body, head } = render(component, { props: { ...data, ...helpers } });
    return { body: unmark(body), head: unmark(head) };
  });
  writeFileSync(join(dir, "expected.json"), JSON.stringify(expected, null, 1) + "\n");
  console.log(id, expected.length);
}
