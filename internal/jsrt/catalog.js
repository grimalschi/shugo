// A fresh copy of the default result from the same catalog snapshot used by
// the Go server. This module is available only during Shugo evaluation.
export function catalog() {
  const result = process.env.SHUGO_CATALOG;
  if (!result) throw new Error("shugo:catalog needs catalogIslands enabled in shugo.json");
  return JSON.parse(result);
}
