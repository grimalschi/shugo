// Renders the reference components of test/svelte with Svelte itself and
// writes what it gives into expected.json of every task: the HTML Shugo must
// give for the same component and data. The components are kept as .shugo
// files; for Svelte they are copied as .svelte files, with the imports
// renamed back. The marks of hydration are taken out: a static page does
// not need them, and Shugo does not write them.
//
//   cd test/svelte && bun install && bun render.mjs [task…]
//
// Shugo follows one version of Svelte: change it here and in package.json,
// render, and look at what differs in git.
import { plugin } from "bun";
import { mkdtempSync, readFileSync, readdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
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
  const copy = mkdtempSync(join(tmpdir(), "shugo-svelte-"));
  for (const file of readdirSync(dir)) {
    if (file.endsWith(".shugo")) {
      const source = readFileSync(join(dir, file), "utf8").replaceAll('.shugo"', '.svelte"').replaceAll(".shugo'", ".svelte'");
      writeFileSync(join(copy, file.replace(/\.shugo$/, ".svelte")), source);
    } else if (file === "helpers.js") {
      writeFileSync(join(copy, file), readFileSync(join(dir, file)));
    }
  }
  const helpers = readdirSync(dir).includes("helpers.js") ? (await import(join(copy, "helpers.js"))).default : {};
  const component = (await import(resolve(copy, task.entry + ".svelte"))).default;
  const expected = task.datasets.map((data) => {
    const { body, head } = render(component, { props: { ...data, ...helpers } });
    return { body: unmark(body), head: unmark(head) };
  });
  writeFileSync(join(dir, "expected.json"), JSON.stringify(expected, null, 1) + "\n");
  rmSync(copy, { recursive: true, force: true });
  console.log(id, expected.length);
}
