// A stand for point updates. It makes a shop of the given size, builds and
// exports it, then changes the data in twenty-one ways, one after another,
// the way any program would: with SQL against the data database. After
// every change it runs `shugo update` and `shugo export`, and prints what
// the update computed, what it changed, what the export built and how long
// it all took. At the end it builds the same data from scratch and checks
// that the site exported step by step is the same, file for file.
//
//   node stand.mjs --size 10000 [--runtime bun|node] [--shugo ../../bin/shugo]

import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { cpSync, mkdirSync, mkdtempSync, readdirSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { basename, join } from "node:path";
import { fileURLToPath } from "node:url";

import { open } from "./scripts/db.mjs";
import { generate, write } from "./scripts/generate.mjs";

const here = fileURLToPath(new URL(".", import.meta.url));
const option = (name, fallback) => {
  const at = process.argv.indexOf("--" + name);
  return at < 0 ? fallback : process.argv[at + 1];
};
const size = Number(option("size", 1000));
const shugo = option("shugo", join(here, "../../bin/shugo"));
const runtime = option("runtime", "");

// run runs a command of shugo and returns its report, with the time the
// whole command took: starting it is a part of what an update costs.
const run = (command, folder = here, ...more) => {
  const started = performance.now();
  const args = [command, "-json"];
  if (runtime && (command === "build" || command === "update")) args.push("-runtime", runtime);
  const report = JSON.parse(execFileSync(shugo, [...args, ...more, folder], { encoding: "utf8", maxBuffer: 1 << 30 }));
  report.took = performance.now() - started;
  return report;
};

mkdirSync(join(here, "data"), { recursive: true });
const database = join(here, "data/catalog.db");
await write(database, generate(size));
// sql changes the data. The connection is opened for the change alone, as
// a program that brings new data would do.
const sql = async (...statements) => {
  const db = await open(database);
  db.exec("BEGIN");
  for (const [text, ...values] of statements.map((one) => [one].flat())) db.prepare(text).run(...values);
  db.exec("COMMIT");
  db.close();
};

// Every change says what it does to the data and, where that is certain,
// how many pages and shared blocks the update must compute, and how many of
// their records must differ afterwards. The table also shows on how many
// pages the HTML differs: a page that shows a changed shared block is such
// a page, though nothing of it was computed.
const last = "p" + size;
const added = "p" + (size + 1);
const changes = [
  { name: "nothing", statements: [], computed: [0, 0], pages: 0, shared: 0 },
  { name: "a field no page shows, on every 20th product", statements: ["UPDATE products SET note = 'checked' WHERE (position - 1) % 20 = 0"], computed: [0, 0], pages: 0, shared: 0 },
  { name: "the price of one product", statements: ["UPDATE products SET price = price + 100 WHERE id = 'p5'"], computed: [1, 1], pages: 1, shared: 1 },
  { name: "the prices of 20 products", statements: ["UPDATE products SET price = price + 100 WHERE position BETWEEN 10 AND 29"], computed: [20, 20], pages: 20, shared: 20 },
  { name: "the title of the product of the month", statements: ["UPDATE products SET title = title || ' (new)' WHERE id = 'p1'"], computed: [1, 2], pages: 1, shared: 2 },
  { name: "another product of the month", statements: ["UPDATE shop SET featured = 'p2'"], computed: [0, 1], pages: 0, shared: 1 },
  { name: "no product of the month", statements: ["UPDATE shop SET featured = NULL"], computed: [0, 1], pages: 0, shared: 1 },
  { name: "a product moves to another category", statements: ["UPDATE products SET category = 'c1' WHERE id = 'p3'"], shared: 1 },
  { name: "the phone in the footer", statements: ["UPDATE shop SET phone = '+1 202-555-0174'"], computed: [0, 1], pages: 0, shared: 1 },
  { name: "a discount appears", statements: ["UPDATE products SET discount = 10, discount_until = '2026-12-31' WHERE id = 'p6'"], computed: [1, 1], pages: 1, shared: 1 },
  { name: "the text of an approved review", statements: ["DELETE FROM reviews WHERE product = 'p7'", "INSERT INTO reviews (product, author, text, approved) VALUES ('p7', 'Jane', 'Excellent', 1)"], computed: [1, 0], pages: 1, shared: 0 },
  { name: "the text of a hidden review", statements: ["INSERT INTO reviews (product, author, text, approved) VALUES ('p8', 'Spam', 'Buy now', 0)"], computed: [0, 0], pages: 0, shared: 0 },
  { name: "the stock of one variant", statements: ["UPDATE variants SET stock = stock + 50 WHERE sku = 'p9-1'"], computed: [1, 0], pages: 1, shared: 0 },
  // The page hands every variant whole to a component. The component reads
  // the title and the stock, so the shelf reaches nobody.
  { name: "the shelf of one variant, which no page shows", statements: ["UPDATE variants SET shelf = 'B7' WHERE sku = 'p9-1'"], computed: [0, 0], pages: 0, shared: 0 },
  { name: "a variant is added first", statements: ["INSERT INTO variants VALUES ('p9-0', 'p9', 'tiny', 1, 0, 'A1')"], computed: [1, 0], pages: 1, shared: 0 },
  { name: "the first variant is removed", statements: ["DELETE FROM variants WHERE sku = 'p9-0'"], computed: [1, 0], pages: 1, shared: 0 },
  { name: "the badges of one product", statements: ["DELETE FROM badges WHERE product = 'p10'", "INSERT INTO badges (product, text, position) VALUES ('p10', 'one of a kind', 1)"], computed: [1, 1], pages: 1 },
  { name: "two related products change places", statements: ["UPDATE related SET position = 5 - position WHERE product = 'p11'"], computed: [1, 0], pages: 1, shared: 0 },
  { name: "a related product is removed", statements: ["DELETE FROM related WHERE product = 'p12' AND position = (SELECT max(position) FROM related WHERE product = 'p12')"], computed: [1, 0], pages: 1, shared: 0 },
  {
    name: "the last product is deleted",
    statements: [
      ...["variants", "reviews", "badges", "related"].map((table) => [`DELETE FROM ${table} WHERE product = ?`, last]),
      ["DELETE FROM related WHERE other = ?", last],
      ["DELETE FROM products WHERE id = ?", last],
    ],
  },
  {
    name: "a product is added",
    statements: [
      [`INSERT INTO products SELECT ?, ?, 'New item', price, discount, discount_until, available, arrival, description, category, note, ? FROM products WHERE id = 'p4'`, added, "product-" + (size + 1), size + 1],
      [`INSERT INTO variants SELECT ? || substr(sku, 3), ?, title, stock, position, shelf FROM variants WHERE product = 'p4'`, added, added],
      [`INSERT INTO reviews (product, author, text, approved) SELECT ?, author, text, approved FROM reviews WHERE product = 'p4'`, added],
      [`INSERT INTO badges (product, text, position) SELECT ?, text, position FROM badges WHERE product = 'p4'`, added],
    ],
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
const first = run("build");
console.log(`${size} products: ${first.pages.total} pages, ${first.shared.total} shared blocks, ${first.blocks} blocks`);
console.log(`first build ${Math.round(first.time.total)} ms: compile ${Math.round(first.time.compile)}, list ${Math.round(first.time.list)}, compute ${Math.round(first.time.compute)}, save ${Math.round(first.time.save)}`);
// The data has no indexes of its own: the first build makes the ones the
// questions of the pages need, and that time is a part of it.
const indexes = first.data.indexes ?? [];
console.log(`indexes made by the first build, ${Math.round(first.time.indexes ?? 0)} ms: ${indexes.join(", ") || "none"}`);
const whole = run("export");
console.log(`first export ${Math.round(whole.took)} ms: ${whole.pages} pages built, ${whole.files} files\n`);

const rows = [];
let failed = 0;
for (const change of changes) {
  await sql(...change.statements);
  const report = run("update");
  // The export builds the pages whose HTML the update found changed, and
  // no others.
  const exported = run("export");
  const pages = report.pages.added + report.pages.changed + report.pages.removed;
  const shared = report.shared.added + report.shared.changed + report.shared.removed;
  const problems = [];
  if (report.data.whole) problems.push("everything was computed");
  if (change.computed !== undefined && (change.computed[0] !== report.computed.pages || change.computed[1] !== report.computed.shared)) problems.push(`expected ${change.computed.join(" and ")} computed`);
  if ((change.pages !== undefined && change.pages !== pages) || (change.shared !== undefined && change.shared !== shared)) problems.push(`expected ${change.pages ?? "any"} pages, ${change.shared ?? "any"} shared changed`);
  if (exported.built !== report.render) problems.push(`the export built ${exported.built} pages`);
  if (problems.length > 0) failed++;
  rows.push({
    change: change.name,
    "rows changed": report.data.rows,
    "pages computed": report.computed.pages,
    "shared computed": report.computed.shared,
    "pages changed": `${pages}` + (report.pages.added || report.pages.removed ? ` (+${report.pages.added} −${report.pages.removed})` : ""),
    "shared changed": `${shared}` + (report.shared.added || report.shared.removed ? ` (+${report.shared.added} −${report.shared.removed})` : ""),
    "new HTML": report.render,
    "update, ms": Math.round(report.time.total),
    "command, ms": Math.round(report.took),
    "export built": exported.built,
    "export, ms": Math.round(exported.took),
    check: problems.join("; ") || "ok",
  });
}
console.table(rows);

// The same data built from scratch must give the same site.
const scratch = mkdtempSync(join(tmpdir(), "shugo-stand-"));
cpSync(here, scratch, { recursive: true, filter: (path) => !["_build", "node_modules"].includes(basename(path)) });
run("build", scratch);
run("export", scratch);
const [stepByStep, fromScratch] = [digest(join(here, "_build/dist")), digest(join(scratch, "_build/dist"))];
rmSync(scratch, { recursive: true, force: true });
let different = Math.abs(stepByStep.size - fromScratch.size);
for (const [path, hash] of fromScratch) if (stepByStep.get(path) !== hash) different++;
console.log(`\n${fromScratch.size} files. Exported step by step against built from scratch: ${different === 0 ? "the same, file for file" : different + " files differ"}`);
if (failed || different) {
  console.error(`FAILED: ${failed} changes did something else than expected, ${different} files differ`);
  process.exit(1);
}
