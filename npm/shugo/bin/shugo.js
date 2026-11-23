#!/usr/bin/env node
// The shugo command, as the package shugo gives it: it runs the program of
// this system, which comes in a package of its own, @shugo/<system>-<cpu>.
// The package manager installs only the one that fits the machine, as it
// does for esbuild. Runs on Bun and on Node.

import { spawnSync } from "node:child_process";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const system = `${process.platform}-${process.arch}`;
const name = `@shugo/${system}`;

let program;
try {
  program = require.resolve(`${name}/bin/shugo${process.platform === "win32" ? ".exe" : ""}`);
} catch {
  console.error(`shugo: the package ${name}, the shugo command for this system, is not installed. Install shugo again, with its optional dependencies: bun add shugo`);
  process.exit(1);
}

const run = spawnSync(program, process.argv.slice(2), { stdio: "inherit" });
if (run.error) {
  console.error(`shugo: ${run.error.message}`);
  process.exit(1);
}
if (run.signal) process.kill(process.pid, run.signal);
process.exit(run.status ?? 1);
