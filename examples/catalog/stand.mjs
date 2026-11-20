// A stand for point updates. It makes a shop of the given size, builds it,
// then changes the data in twenty ways, one after another. After every
// change it runs `shugo update` and prints what the update touched and how
// long it took. At the end it builds the same data from scratch and checks
// that the site updated step by step is the same, file for file.
//
//   node stand.mjs --size 10000 [--runtime bun|node] [--shugo ../../bin/shugo]

import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { cpSync, mkdirSync, mkdtempSync, readdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

import { generate } from "./lib/generate.mjs";

const here = fileURLToPath(new URL(".", import.meta.url));
const option = (name, fallback) => {
  const at = process.argv.indexOf("--" + name);
  return at < 0 ? fallback : process.argv[at + 1];
};
const size = Number(option("size", 1000));
const shugo = option("shugo", join(here, "../../bin/shugo"));
const runtime = option("runtime", "");

const data = generate(size);
const product = (number) => data.products[number - 1];
mkdirSync(join(here, "data"), { recursive: true });
const save = () => writeFileSync(join(here, "data/catalog.json"), JSON.stringify(data));
const run = (command, folder = here, ...more) =>
  JSON.parse(execFileSync(shugo, [command, "-json", ...(runtime ? ["-runtime", runtime] : []), ...more, folder], { encoding: "utf8", maxBuffer: 1 << 30 }));

// Every change says what it does to the data and, where that is certain,
// how many records of pages and of shared blocks must differ afterwards.
// The table also shows on how many pages the HTML differs: a page that
// shows a changed shared block is such a page, though its own record is
// what it was.
const changes = [
  { name: "nothing", edit() {}, pages: 0, shared: 0 },
  { name: "a field no page shows, on every 20th product", edit() { data.products.forEach((p, i) => { if (i % 20 === 0) p.note = "checked"; }); }, pages: 0, shared: 0 },
  { name: "the price of one product", edit() { product(5).price += 100; }, pages: 1, shared: 1 },
  { name: "the prices of 20 products", edit() { for (let n = 10; n < 30; n++) product(n).price += 100; }, pages: 20, shared: 20 },
  { name: "the title of the product of the month", edit() { product(1).title += " (new)"; }, pages: 1, shared: 2 },
  { name: "another product of the month", edit() { data.shop.featured = "p2"; }, pages: 0, shared: 1 },
  { name: "no product of the month", edit() { data.shop.featured = null; }, pages: 0, shared: 1 },
  { name: "a product moves to another category", edit() { product(3).category = "c1"; }, shared: 1 },
  { name: "the phone in the footer", edit() { data.shop.phone = "+1 202-555-0174"; }, pages: 0, shared: 1 },
  { name: "a discount appears", edit() { product(6).discount = { percent: 10, until: "2026-12-31" }; }, pages: 1, shared: 1 },
  { name: "the text of an approved review", edit() { product(7).reviews = [{ author: "Jane", text: "Excellent", approved: true }]; }, pages: 1, shared: 0 },
  { name: "the text of a hidden review", edit() { product(8).reviews.push({ author: "Spam", text: "Buy now", approved: false }); }, pages: 0, shared: 0 },
  { name: "the stock of one variant", edit() { product(9).variants[0].stock += 50; }, pages: 1, shared: 0 },
  { name: "a variant is added first", edit() { product(9).variants.unshift({ sku: "p9-0", title: "tiny", stock: 1 }); }, pages: 1, shared: 0 },
  { name: "the first variant is removed", edit() { product(9).variants.shift(); }, pages: 1, shared: 0 },
  { name: "the badges of one product", edit() { product(10).badges = ["one of a kind"]; }, pages: 1 },
  { name: "two related products change places", edit() { product(11).related.reverse(); }, pages: 1, shared: 0 },
  { name: "a related product is removed", edit() { product(12).related.pop(); }, pages: 1, shared: 0 },
  {
    name: "the last product is deleted",
    edit() {
      const gone = data.products.pop();
      for (const p of data.products) p.related = p.related.filter((id) => id !== gone.id);
    },
  },
  {
    name: "a product is added",
    edit() {
      const number = data.products.length + 2;
      data.products.push({ ...structuredClone(product(4)), id: "p" + number, slug: "product-" + number, title: "New item", related: [] });
    },
  },
];

// The files of an exported site with the hash of each.
function digest(folder) {
  const out = new Map();
  for (const entry of readdirSync(folder, { recursive: true, withFileTypes: true })) {
    if (!entry.isFile() || entry.name.startsWith(".")) continue;
    const path = join(entry.parentPath, entry.name);
    out.set(path.slice(folder.length), createHash("sha256").update(readFileSync(path)).digest("hex"));
  }
  return out;
}

rmSync(join(here, "_build"), { recursive: true, force: true });
save();
const first = run("build");
console.log(`${size} products: ${first.pages.total} pages, ${first.shared.total} shared blocks, ${first.blocks} blocks, ${first.values.total} distinct values`);
console.log(`first build ${Math.round(first.time.total)} ms: compile ${Math.round(first.time.compile)}, compute ${Math.round(first.time.compute)}, save ${Math.round(first.time.save)}\n`);

const rows = [];
let failed = 0;
for (const change of changes) {
  change.edit();
  save();
  const report = run("update");
  const pages = report.pages.added + report.pages.changed + report.pages.removed;
  const shared = report.shared.added + report.shared.changed + report.shared.removed;
  const ok = (change.pages === undefined || change.pages === pages) && (change.shared === undefined || change.shared === shared);
  if (!ok) failed++;
  rows.push({
    change: change.name,
    pages: `${pages}` + (report.pages.added || report.pages.removed ? ` (+${report.pages.added} −${report.pages.removed})` : ""),
    shared: `${shared}` + (report.shared.added || report.shared.removed ? ` (+${report.shared.added} −${report.shared.removed})` : ""),
    "new HTML": report.render,
    values: `+${report.values.added} −${report.values.removed}`,
    "compute, ms": Math.round(report.time.compute),
    "save, ms": Math.round(report.time.save),
    "total, ms": Math.round(report.time.total),
    check: ok ? "ok" : `expected ${change.pages ?? "any"} pages, ${change.shared ?? "any"} shared`,
  });
}
console.table(rows);

// The same data built from scratch must give the same site.
const exported = run("export");
const scratch = mkdtempSync(join(tmpdir(), "shugo-stand-"));
cpSync(here, scratch, { recursive: true, filter: (path) => !path.includes("_build") && !path.includes("node_modules") });
run("build", scratch);
run("export", scratch);
const [stepByStep, fromScratch] = [digest(join(here, "_build/dist")), digest(join(scratch, "_build/dist"))];
rmSync(scratch, { recursive: true, force: true });
let different = Math.abs(stepByStep.size - fromScratch.size);
for (const [path, hash] of fromScratch) if (stepByStep.get(path) !== hash) different++;
console.log(`\nexport: ${exported.pages} pages. Updated step by step against built from scratch: ${different === 0 ? "the same, file for file" : different + " files differ"}`);
if (failed || different) {
  console.error(`FAILED: ${failed} changes touched something else than expected, ${different} files differ`);
  process.exit(1);
}
