import { readSearch } from "shugo:query";
import { requestSource } from "shugo:runtime";
import { buildChannel } from "shugo:wire";

let wire;
let prepared;

// Ask the Go build process for the default result of a search declaration.
// The Go process answers this before run() signals that the modules loaded.
export function search(table, options = {}) {
  if (typeof table !== "string" || table.trim() === "") throw new TypeError("search() needs a table name");
  if (options === null || typeof options !== "object" || Array.isArray(options)) {
    throw new TypeError("search() options must be a JSON object");
  }
  const declaration = JSON.stringify({ table, options }, (_, value) => {
    const type = typeof value;
    if (type === "function" || type === "symbol" || type === "bigint" || (type === "number" && !Number.isFinite(value))) {
      throw new TypeError("search() options must contain JSON values");
    }
    return value;
  });

  const cache = (prepared ??= new Map());
  let json = cache.get(declaration);
  if (json === undefined) {
    if (wire === undefined) {
      const channel = buildChannel();
      if (process.env.SHUGO_SEARCH !== "1" || channel === null) {
        throw new Error("shugo:search needs the Shugo build process and a data database in shugo.json");
      }
      wire = channel;
    }
    wire.send("J", declaration);
    const reply = wire.receive();
    if (reply?.kind === "E") throw new Error(JSON.parse(reply.text).message);
    if (reply?.kind !== "J") throw new Error("the Shugo build process did not answer search()");
    json = reply.text;
  }
  const { id, result } = JSON.parse(json);
  if (typeof id !== "string" || id === "" || result === null || typeof result !== "object" || Array.isArray(result)) {
    throw new Error("the Shugo build process sent an invalid search result");
  }
  cache.set(declaration, json);
  readSearch(id);
  return requestSource(result, "search:" + id);
}
