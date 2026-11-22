import { mkdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { generate, write } from "../../catalog/scripts/generate.mjs";

const count = Number(process.argv[2] ?? 1000);
if (!Number.isSafeInteger(count) || count < 2) throw new Error("Expected a product count of at least 2.");
const path = new URL("../data/catalog.db", import.meta.url);
mkdirSync(new URL("../data/", import.meta.url), { recursive: true });
await write(fileURLToPath(path), generate(count));
console.log(`data/catalog.db: ${count} products`);
