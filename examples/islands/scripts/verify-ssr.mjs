// Install the example's dependencies, then run this file with Bun.
// Compare HTTP bodies with Svelte itself and serve a runtime-free copy.
import assert from "node:assert/strict";
import { execFileSync, spawn, spawnSync } from "node:child_process";
import { cpSync, mkdirSync, mkdtempSync, readFileSync, readdirSync, realpathSync, rmSync, symlinkSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, relative } from "node:path";
import { fileURLToPath } from "node:url";
import { plugin } from "bun";
import { compile, VERSION } from "svelte/compiler";
import { render } from "svelte/server";
import { generate, write } from "../../catalog/scripts/generate.mjs";

const demo = fileURLToPath(new URL("../", import.meta.url));
const workspace = mkdtempSync(join(tmpdir(), "shugo-island-ssr-"));
const project = join(workspace, "project");
const deployment = join(workspace, "deployment");
const binary = join(workspace, "shugo");
const runtime = process.env.SHUGO_RUNTIME ?? "bun";
let server;

try {
  assert.equal(VERSION, "5.59.1");
  assert(["bun", "node"].includes(runtime), "SHUGO_RUNTIME must name bun or node");
  mkdirSync(join(project, "data"), { recursive: true });
  cpSync(join(demo, "src"), join(project, "src"), { recursive: true });
  for (const name of ["shugo.json", "package.json", "bun.lock"]) cpSync(join(demo, name), join(project, name));
  symlinkSync(realpathSync(join(demo, "node_modules")), join(project, "node_modules"), "dir");
  await write(join(project, "data", "catalog.db"), generate(1000));
  execFileSync("go", ["build", "-o", binary, "./cmd/shugo"], { cwd: join(demo, "../.."), stdio: "inherit" });
  execFileSync(binary, ["build", "-runtime", runtime, project], { stdio: "inherit" });

  // The CLI checks for the directory but serve needs no source files,
  // database inputs, Node/Bun, dependencies, or build-time JS bundles.
  mkdirSync(join(deployment, "src", "pages"), { recursive: true });
  mkdirSync(join(deployment, "_build"));
  const missingRuntime = join(deployment, "no-javascript-runtime");
  const config = JSON.parse(readFileSync(join(project, "shugo.json"), "utf8"));
  writeFileSync(join(deployment, "shugo.json"), JSON.stringify({ ...config, runtime: missingRuntime }));
  for (const name of readdirSync(join(project, "_build"))) {
    if (name === "assets" || name === "store.db" || /^catalog-[a-f0-9]+\.json$/.test(name)) {
      cpSync(join(project, "_build", name), join(deployment, "_build", name), { recursive: true });
    }
  }
  server = spawn(binary, ["serve", "-addr", "127.0.0.1:0", deployment], {
    cwd: deployment,
    env: { ...process.env, PATH: missingRuntime },
    stdio: ["ignore", "pipe", "inherit"],
  });
  const origin = await listening(server);

  // This process is the reference implementation, outside the deployment.
  plugin({ name: "svelte-reference", setup(build) {
    build.onLoad({ filter: /\.(svelte|shugo)$/ }, ({ path }) => {
      const source = readFileSync(path, "utf8");
      const { js } = compile(source, { filename: relative(project, path), generate: "server", css: "external" });
      return { contents: js.code, loader: "js" };
    });
  } });
  const catalogPath = join(project, "src", "components", "Catalog.svelte");
  const { default: Catalog } = await import(catalogPath);
  const queries = [
    {},
    { q: "cup", sort: "cheap", stock: "1", page: "2" },
    { q: "thisproductdoesnotexist" },
    { category: "c1", stock: "1", price_from: "800", price_to: "5000" },
    { page: "2" },
    { page: "4" },
    { page: "99" },
    { q: "</script><span data-test>\"&" },
    { sort: "expensive", page: "2" },
    { price_to: "0" },
    [["category", "c1"], ["category", "c2"], ["size", "medium"], ["sort", "new"]],
  ];
  for (const values of queries) {
    const search = String(new URLSearchParams(values));
    const query = search ? "?" + search : "";
    const [page, api] = await Promise.all([
      fetch(origin + "/catalog" + query),
      fetch(origin + "/_islands/catalog" + query),
    ]);
    assert.equal(page.status, 200, query);
    assert.equal(api.status, 200, query);
    const { body, props } = island(await page.text());
    assert.deepEqual(props.initial, await api.json(), "hydration props must match the API " + query);
    const expected = render(Catalog, { props }).body;
    if (body !== expected) {
      let at = 0;
      while (at < body.length && at < expected.length && body[at] === expected[at]) at++;
      throw new Error(`SSR differs at character ${at} for ${query || "the default page"}\n` +
        `Svelte: ${JSON.stringify(expected.slice(at, at + 100))}\nShugo:  ${JSON.stringify(body.slice(at, at + 100))}`);
    }
  }
  if (process.platform === "linux") {
    const tasks = `/proc/${server.pid}/task`;
    for (const tid of readdirSync(tasks)) {
      try {
        assert.equal(readFileSync(join(tasks, tid, "children"), "utf8").trim(), "", "serve has no child process");
      } catch (error) {
        if (error.code !== "ENOENT") throw error; // The runtime can discard an idle OS thread.
      }
    }
  }
  console.log(`${queries.length} catalog responses and hydration payloads match Svelte in a deployment without JavaScript tools.`);

  const original = readFileSync(catalogPath, "utf8");
  writeFileSync(catalogPath, original + "\n<p>{result.total.toFixed(0)}</p>\n");
  const rejected = spawnSync(binary, ["build", "-runtime", runtime, project], { encoding: "utf8" });
  assert.notEqual(rejected.status, 0);
  assert.match(rejected.stderr, /request markup needs a JSON path or a literal/);
  console.log("Build rejected a JavaScript call in request markup.");
} finally {
  if (server?.pid && server.exitCode === null && server.signalCode === null) {
    const stopped = new Promise((resolve) => server.once("exit", resolve));
    server.kill();
    await stopped;
  }
  rmSync(workspace, { recursive: true, force: true });
}

function listening(child) {
  return new Promise((resolve, reject) => {
    let log = "";
    const timeout = setTimeout(() => reject(new Error(`Shugo did not listen within 15 s\n${log}`)), 15_000);
    child.stdout.on("data", (chunk) => {
      log += chunk;
      const address = log.match(/the site is at (http:\/\/127\.0\.0\.1:\d+)\n/);
      if (address) {
        clearTimeout(timeout);
        resolve(address[1]);
      }
    });
    child.on("error", (error) => {
      clearTimeout(timeout);
      reject(error);
    });
    child.on("exit", (status) => {
      clearTimeout(timeout);
      reject(new Error(`Shugo failed to start: ${status}\n${log}`));
    });
  });
}

function island(html) {
  const open = "<div data-shugo-target>";
  const payload = '</div><script type="application/json" data-shugo-props>';
  const start = html.indexOf(open);
  const middle = html.indexOf(payload, start);
  const end = html.indexOf("</script>", middle);
  assert(start >= 0 && middle > start && end > middle, "missing server-rendered island and hydration props");
  return { body: html.slice(start + open.length, middle), props: JSON.parse(html.slice(middle + payload.length, end)) };
}
