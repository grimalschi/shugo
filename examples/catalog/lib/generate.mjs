// Makes the data of a shop: `node lib/generate.mjs 1000` writes
// data/catalog.json with that many products. The same number always gives
// the same shop.
import { mkdirSync, writeFileSync } from "node:fs";

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

if (process.argv[1] && import.meta.url.endsWith(process.argv[1].split("/").pop())) {
  const count = Number(process.argv[2] ?? 1000);
  mkdirSync(new URL("../data/", import.meta.url), { recursive: true });
  writeFileSync(new URL("../data/catalog.json", import.meta.url), JSON.stringify(generate(count)));
  console.log(`data/catalog.json: ${count} products`);
}
