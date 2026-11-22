import { mkdirSync, readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { open } from "../../catalog/scripts/db.mjs";

mkdirSync(new URL("../data/", import.meta.url), { recursive: true });
const db = await open(fileURLToPath(new URL("../data/shop.db", import.meta.url)));
try {
  db.exec(readFileSync(new URL("./fixture.sql", import.meta.url), "utf8"));
} finally {
  db.close();
}
