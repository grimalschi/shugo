// Run from the repository after building bin/shugo and installing the demo's
// dependencies. Each variant gets fresh copies of the same seeded database.
//
// node examples/islands/scripts/bench.mjs --sizes 1000,10000,100000 --runs 3
// Add --clients 8 to also measure overlapping filtered SSR and JSON requests.
import { execFileSync, spawn } from "node:child_process";
import { createHash } from "node:crypto";
import { cpSync, copyFileSync, mkdirSync, mkdtempSync, readFileSync, readdirSync, realpathSync, rmSync, symlinkSync, writeFileSync } from "node:fs";
import { Agent, get } from "node:http";
import { arch, cpus, platform, tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { parseArgs } from "node:util";
import { gzipSync } from "node:zlib";
import { open } from "../../catalog/scripts/db.mjs";
import { generate, write } from "../../catalog/scripts/generate.mjs";

const here = fileURLToPath(new URL("..", import.meta.url));
const { values: options } = parseArgs({
  options: {
    sizes: { type: "string", default: "1000,10000,100000" },
    runs: { type: "string", default: "3" },
    requests: { type: "string", default: "60" },
    clients: { type: "string", default: "1" },
    shugo: { type: "string", default: join(here, "../../bin/shugo") },
    fixture: { type: "string", default: here },
    runtime: { type: "string" },
  },
});

function integer(value, name, minimum = 1) {
  const result = Number(value);
  if (!Number.isSafeInteger(result) || result < minimum) {
    throw new Error(`Expected ${name} to be an integer of at least ${minimum}`);
  }
  return result;
}

const sizes = options.sizes.split(",").map((n) => integer(n, "each size", 2));
const runs = integer(options.runs, "--runs");
const requests = integer(options.requests, "--requests");
const clients = integer(options.clients, "--clients");
if (clients > requests) throw new Error("--requests must cover every --clients worker");
const shugo = resolve(options.shugo);
const demo = resolve(options.fixture);
const modules = realpathSync(join(demo, "node_modules"));
if (options.runtime && !["bun", "node"].includes(options.runtime)) {
  throw new Error("Expected --runtime to be bun or node");
}

const filters = new URLSearchParams({
  q: "handmade",
  stock: "1",
  price_to: "6000",
  sort: "cheap",
});
filters.append("category", "c1");
filters.append("category", "c2");
const filteredPath = `/catalog?${filters}`;
const apiPath = `/_islands/catalog?${filters}`;

function prepare(folder, database, variant) {
  mkdirSync(join(folder, "data"), { recursive: true });
  copyFileSync(database, join(folder, "data/catalog.db"));
  cpSync(join(demo, "src"), join(folder, "src"), { recursive: true });
  const pages = join(folder, "src/pages");
  rmSync(join(pages, "multiple.shugo"));
  if (variant === "baseline") copyFileSync(join(pages, "baseline.shugo"), join(pages, "catalog.shugo"));
  rmSync(join(pages, "baseline.shugo"));
  // Both projects use the same catalog projection and default result. Only
  // the island variant needs to compile Svelte and publish browser assets.
  const config = JSON.parse(readFileSync(join(demo, "shugo.json"), "utf8"));
  if (options.runtime) config.runtime = options.runtime;
  writeFileSync(join(folder, "shugo.json"), JSON.stringify(config));
  if (variant === "islands") {
    for (const name of ["package.json", "bun.lock"]) copyFileSync(join(demo, name), join(folder, name));
    symlinkSync(modules, join(folder, "node_modules"), "junction");
  }
}

function shugoCommand(command, folder) {
  const started = performance.now();
  const out = execFileSync(shugo, [command, "-json", folder], { encoding: "utf8", maxBuffer: 64 << 20 });
  const wall = performance.now() - started;
  const report = JSON.parse(out);
  return { ...report, wall };
}

function measureCommand(report) {
  return {
    wall: report.wall,
    compile: report.time.compile ?? 0,
    catalog: report.time.catalog ?? 0,
    compute: report.time.compute ?? 0,
    total: report.time.total,
    pages: report.computed.pages,
    shared: report.computed.shared,
  };
}

async function withServer(folder, run) {
  const started = performance.now();
  const server = spawn(shugo, ["serve", "-addr", "127.0.0.1:0", folder], { stdio: ["ignore", "pipe", "pipe"] });
  const closed = new Promise((done) => server.once("close", done));
  const agent = new Agent({ keepAlive: true, maxSockets: clients });
  let diagnostics = "";
  server.stderr.on("data", (chunk) => { diagnostics += chunk; });
  try {
    const origin = await new Promise((accept, reject) => {
      let stdout = "";
      const deadline = setTimeout(() => reject(new Error(`Server readiness timed out: ${diagnostics}`)), 15000);
      server.on("error", (err) => { clearTimeout(deadline); reject(err); });
      closed.then((code) => { clearTimeout(deadline); reject(new Error(`Server exited (${code}): ${diagnostics}`)); });
      server.stdout.on("data", (chunk) => {
        stdout += chunk;
        const address = stdout.match(/the site is at (http:\/\/127\.0\.0\.1:\d+)\r?\n/);
        if (address) {
          clearTimeout(deadline);
          accept(address[1]);
        }
      });
    });
    const ready = performance.now() - started;
    return await run((path) => request(origin + path, agent), ready);
  } catch (err) {
    throw diagnostics ? new Error(diagnostics.trim(), { cause: err }) : err;
  } finally {
    agent.destroy();
    server.kill();
    await closed;
  }
}

function request(address, agent) {
  return new Promise((accept, reject) => {
    const started = performance.now();
    const req = get(address, { agent }, (res) => {
      const firstByte = performance.now() - started;
      const chunks = [];
      res.on("data", (chunk) => chunks.push(chunk));
      res.on("error", reject);
      res.on("end", () => {
        const total = performance.now() - started;
        const body = Buffer.concat(chunks);
        if (res.statusCode !== 200) {
          reject(new Error(`${address} returned ${res.statusCode}: ${body.toString()}`));
          return;
        }
        accept({ body, time: { firstByte, total } });
      });
    });
    req.setTimeout(15000, () => req.destroy(new Error(`Request timed out: ${address}`)));
    req.on("error", reject);
  });
}

function quantile(samples, percentile) {
  const ordered = [...samples].sort((a, b) => a - b);
  const rank = (ordered.length - 1) * percentile;
  const lower = ordered[Math.floor(rank)];
  const upper = ordered[Math.ceil(rank)];
  return +(lower + (upper - lower) * (rank % 1)).toFixed(3);
}

async function steady(request, path, parallel = 1) {
  for (let i = 0; i < 5; i++) await request(path);
  if (parallel > 1) await Promise.all(Array.from({ length: parallel }, () => request(path)));
  const measurements = new Array(requests);
  let next = 0;
  const client = async () => {
    while (next < requests) {
      const at = next++;
      measurements[at] = (await request(path)).time;
    }
  };
  const started = performance.now();
  await Promise.all(Array.from({ length: parallel }, client));
  const elapsed = performance.now() - started;
  return {
    firstByteP50: quantile(measurements.map((time) => time.firstByte), .5),
    firstByteP95: quantile(measurements.map((time) => time.firstByte), .95),
    totalP50: quantile(measurements.map((time) => time.total), .5),
    totalP95: quantile(measurements.map((time) => time.total), .95),
    elapsed: +elapsed.toFixed(3),
    requestsPerSecond: +(requests * 1000 / elapsed).toFixed(3),
    ...(clients > 1 ? { requestTimes: measurements } : {}),
  };
}

function weight(body) {
  return { raw: body.length, gzip: gzipSync(body, { level: 6 }).length };
}

function assetWeights(build) {
  const totals = { js: { raw: 0, gzip: 0 }, css: { raw: 0, gzip: 0 } };
  const folder = join(build, "assets");
  for (const file of readdirSync(folder)) {
    const type = file.endsWith(".js") ? "js" : file.endsWith(".css") ? "css" : null;
    if (!type) continue;
    const sizes = weight(readFileSync(join(folder, file)));
    totals[type].raw += sizes.raw;
    totals[type].gzip += sizes.gzip;
  }
  return totals;
}

const hitIDs = (html) => [...html.toString().matchAll(/data-product-id="([^"]+)"/g)].map((match) => match[1]);

async function measure(folder, variant, size) {
  const build = measureCommand(shugoCommand("build", folder));
  const buildFolder = join(folder, "_build");
  const indexName = readdirSync(buildFolder).find((name) => /^catalog-[a-f\d]+\.json$/.test(name));
  if (!indexName) throw new Error("Catalog build did not produce an index");
  const index = weight(readFileSync(join(buildFolder, indexName)));
  const assets = assetWeights(buildFolder);
  const found = await withServer(folder, async (request, ready) => {
    const first = await request(variant === "islands" ? filteredPath : "/catalog");
    const initial = variant === "islands" ? await request("/catalog") : first;
    const html = weight(initial.body);
    const hits = hitIDs(initial.body);
    if (hits.length !== Math.min(24, size)) throw new Error(`Expected initial ${variant} page to contain the catalog cards`);
    const doc = await steady(request, "/catalog");
    if (variant === "baseline") return { hits, ready, cold: first.time, doc, html };
    const json = await request(apiPath);
    const asJSON = JSON.parse(json.body);
    const filteredHits = hitIDs(first.body);
    if (JSON.stringify(filteredHits) !== JSON.stringify(asJSON.hits.map((p) => p.id))) {
      throw new Error("Filtered SSR and JSON cards disagree");
    }
    const filtered = await steady(request, filteredPath);
    const api = await steady(request, apiPath);
    const server = { hits, ready, cold: first.time, doc, html, filtered, api, filteredHTML: weight(first.body), filteredJSON: weight(json.body) };
    if (clients > 1) {
      server.overlap = {
        filtered: await steady(request, filteredPath, clients),
        api: await steady(request, apiPath, clients),
      };
    }
    return server;
  });
  const { hits, ...server } = found;
  if (variant === "islands") {
    const cold = await withServer(folder, async (request) => {
      const firstAPI = await request(apiPath);
      const firstSSR = await request(filteredPath);
      return { api: firstAPI.time, html: firstSSR.time };
    });
    // The JSON request opens the index. The filtered HTML request uses
    // that already loaded index without earlier HTML requests.
    server.coldAPI = cold.api;
    server.warmIndexFirstHTML = cold.html;
  }

  const noop = measureCommand(shugoCommand("update", folder));
  const db = await open(join(folder, "data/catalog.db"));
  try {
    db.prepare("UPDATE products SET price = price + 100 WHERE id = 'p1'").run();
  } finally {
    db.close();
  }
  const update = measureCommand(shugoCommand("update", folder));
  return { hits, build, index, assets, server, noop, update };
}

function medians(samples) {
  const summary = Object.keys(samples[0]).filter((key) => !Array.isArray(samples[0][key]));
  return Object.fromEntries(summary.map((key) => {
    const column = samples.map((sample) => sample[key]);
    return [key, typeof column[0] === "number" ? quantile(column, .5) : medians(column)];
  }));
}

const result = {
  machine: { platform: platform(), arch: arch(), cpu: cpus()[0]?.model, logicalCPUs: cpus().length, node: process.version },
  runtime: options.runtime ?? "project default (Bun when available)",
  fixture: demo,
  command: process.argv,
  binarySHA256: createHash("sha256").update(readFileSync(shugo)).digest("hex"),
  runs,
  requests,
  clients,
  filteredPath,
  sizes: [],
};

const scratch = mkdtempSync(join(tmpdir(), "shugo-island-bench-"));
try {
  for (const size of sizes) {
    const database = join(scratch, `seed-${size}.db`);
    await write(database, generate(size));
    const samples = { baseline: [], islands: [] };
    for (let round = 0; round < runs; round++) {
      let expected;
      const variants = round % 2 === 0 ? ["baseline", "islands"] : ["islands", "baseline"];
      for (const variant of variants) {
        const folder = join(scratch, `${size}-${variant}`);
        try {
          prepare(folder, database, variant);
          const { hits, ...sample } = await measure(folder, variant, size);
          if (expected && JSON.stringify(hits) !== expected) throw new Error("Initial baseline and island cards disagree");
          expected = JSON.stringify(hits);
          samples[variant].push(sample);
        } finally {
          rmSync(folder, { recursive: true, force: true });
        }
      }
    }
    result.sizes.push({ size, median: { baseline: medians(samples.baseline), islands: medians(samples.islands) }, samples });
  }
  console.log(JSON.stringify(result, null, 2));
} finally {
  rmSync(scratch, { recursive: true, force: true });
}
