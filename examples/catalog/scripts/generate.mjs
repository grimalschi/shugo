// Makes the data of a shop: `node scripts/generate.mjs 1000` writes
// data/catalog.db with that many products. The same number always gives
// the same shop.
import { mkdirSync, rmSync } from "node:fs";
import { fileURLToPath } from "node:url";

import { open } from "./db.mjs";

// The tables of the shop. Every table has a key of one column: by it Shugo
// tells its rows apart. The order of products, variants, badges and related
// products is a column of its own.
//
// There are no indexes here beside what the tables say themselves. Shugo
// makes the ones the questions of the pages need, when they are first asked.
export const tables = `
CREATE TABLE shop (id INTEGER PRIMARY KEY, name TEXT NOT NULL, phone TEXT NOT NULL, featured TEXT);
CREATE TABLE categories (id TEXT PRIMARY KEY, title TEXT NOT NULL, position INTEGER NOT NULL);
CREATE TABLE products (
  id TEXT PRIMARY KEY, slug TEXT NOT NULL UNIQUE, title TEXT NOT NULL, price INTEGER NOT NULL,
  discount INTEGER, discount_until TEXT, available INTEGER NOT NULL, arrival TEXT NOT NULL,
  description TEXT NOT NULL, category TEXT NOT NULL, note TEXT NOT NULL, position INTEGER NOT NULL
);
CREATE TABLE variants (sku TEXT PRIMARY KEY, product TEXT NOT NULL, title TEXT NOT NULL, stock INTEGER NOT NULL, position INTEGER NOT NULL, shelf TEXT NOT NULL);
CREATE TABLE reviews (id INTEGER PRIMARY KEY, product TEXT NOT NULL, author TEXT NOT NULL, text TEXT NOT NULL, approved INTEGER NOT NULL);
CREATE TABLE badges (id INTEGER PRIMARY KEY, product TEXT NOT NULL, text TEXT NOT NULL, position INTEGER NOT NULL);
CREATE TABLE related (id INTEGER PRIMARY KEY, product TEXT NOT NULL, other TEXT NOT NULL, position INTEGER NOT NULL);
`;

export function generate(count) {
  // A small generator of numbers that repeats itself for the same seed.
  let seed = 20261120;
  const random = () => {
    seed = (seed * 1664525 + 1013904223) >>> 0;
    return seed / 2 ** 32;
  };
  const pick = (list) => list[Math.floor(random() * list.length)];
  const whole = (from, to) => from + Math.floor(random() * (to - from + 1));

  const kinds = ["Pitcher", "Cup", "Tray", "Vase", "Platter", "Bowl", "Candleholder", "Planter", "Plate", "Teapot"];
  const names = ["Line", "Shore", "North", "Morning", "Moss", "Clay", "Smoke", "Salt", "Flax", "Stone"];
  const words = ["handmade", "matte glaze", "dishwasher-safe", "small batch", "each item varies slightly", "dense clay body", "stable base"];
  const badges = ["new", "bestseller", "eco", "gift", "last one", "fresh from the kiln"];
  const authors = ["Anna", "Ben", "Vera", "Glen", "Dina", "Eric"];
  const categories = Array.from({ length: Math.max(2, Math.min(50, Math.round(count / 20))) }, (_, i) => ({
    id: "c" + (i + 1),
    title: pick(kinds) + " and " + pick(kinds).toLowerCase() + " #" + (i + 1),
  }));

  const products = Array.from({ length: count }, (_, i) => {
    const id = "p" + (i + 1);
    return {
      id,
      slug: "product-" + (i + 1),
      title: pick(kinds) + " «" + pick(names) + "» " + (i + 1),
      price: whole(3, 90) * 100,
      discount: random() < 0.2 ? { percent: pick([5, 10, 15, 30]), until: "2026-12-" + whole(10, 31) } : null,
      available: random() < 0.85,
      arrival: "2027-01-" + whole(10, 28),
      description: Array.from({ length: whole(2, 4) }, () => pick(words)).join(", ") + ".",
      category: categories[i % categories.length].id,
      note: "",
      badges: Array.from({ length: whole(0, 2) }, () => pick(badges)),
      variants: Array.from({ length: whole(1, 3) }, (_, v) => ({ sku: id + "-" + (v + 1), title: pick(["small", "medium", "large"]), stock: whole(0, 12) })),
      reviews: Array.from({ length: whole(0, 3) }, () => ({ author: pick(authors), text: pick(words), approved: random() < 0.8 })),
      related: Array.from({ length: Math.min(4, count - 1) }, (_, r) => "p" + (((i + 1 + (r + 1) * 7) % count) + 1)),
    };
  });
  return {
    shop: { name: "Measure", phone: "+1 202-555-0142", featured: "p1" },
    categories,
    products,
  };
}

// write makes the data database of a shop anew.
export async function write(path, shop) {
  for (const suffix of ["", "-wal", "-shm"]) rmSync(path + suffix, { force: true });
  const db = await open(path);
  db.exec("PRAGMA journal_mode = WAL; PRAGMA synchronous = OFF; BEGIN;" + tables);
  db.prepare("INSERT INTO shop VALUES (1, ?, ?, ?)").run(shop.shop.name, shop.shop.phone, shop.shop.featured);
  const category = db.prepare("INSERT INTO categories VALUES (?, ?, ?)");
  shop.categories.forEach((c, i) => category.run(c.id, c.title, i + 1));
  const product = db.prepare("INSERT INTO products VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)");
  const variant = db.prepare("INSERT INTO variants VALUES (?, ?, ?, ?, ?, ?)");
  const review = db.prepare("INSERT INTO reviews (product, author, text, approved) VALUES (?, ?, ?, ?)");
  const badge = db.prepare("INSERT INTO badges (product, text, position) VALUES (?, ?, ?)");
  const related = db.prepare("INSERT INTO related (product, other, position) VALUES (?, ?, ?)");
  shop.products.forEach((p, i) => {
    product.run(p.id, p.slug, p.title, p.price, p.discount?.percent ?? null, p.discount?.until ?? null, p.available ? 1 : 0, p.arrival, p.description, p.category, p.note, i + 1);
    p.variants.forEach((v, at) => variant.run(v.sku, p.id, v.title, v.stock, at + 1, "A1"));
    p.reviews.forEach((r) => review.run(p.id, r.author, r.text, r.approved ? 1 : 0));
    p.badges.forEach((text, at) => badge.run(p.id, text, at + 1));
    p.related.forEach((other, at) => related.run(p.id, other, at + 1));
  });
  db.exec("COMMIT;");
  db.close();
}

if (process.argv[1] && import.meta.url.endsWith(process.argv[1].split("/").pop())) {
  const count = Number(process.argv[2] ?? 1000);
  mkdirSync(new URL("../data/", import.meta.url), { recursive: true });
  await write(fileURLToPath(new URL("../data/catalog.db", import.meta.url)), generate(count));
  console.log(`data/catalog.db: ${count} products`);
}
