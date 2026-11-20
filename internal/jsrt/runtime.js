// The JavaScript side of Shugo. The code generated from templates calls this
// library while it computes pages: it runs the scripts of pages and
// components, collects the value of every expression in the order of the
// template, and hands the result to the Go side, which keeps it and builds
// HTML from it. No markup is produced here.
//
// Runs on Bun and on Node.

import { readFileSync, writeSync } from "node:fs";

// ---------------------------------------------------------------------------
// What is computed

// A record is everything computed for one page, or for one shared block: its
// blocks in the order they were started. Blocks of a record name each other
// by their place in it.
class Record {
  constructor(kind, key, origin) {
    this.kind = kind; // "p" for a page, "s" for a shared block
    this.key = key; // the address of the page, or the name of the shared block
    this.origin = origin; // the route of the page, or the number of the shared block
    this.blocks = [];
    this.top = 0;
  }

  // A block is one template being shown: a page, a layout, a component, or
  // markup put into a component (a part of the template it is written in).
  block(file, part) {
    const block = { index: this.blocks.length, file, part, values: [], children: [], slots: null, shared: -1 };
    this.blocks.push(block);
    return block;
  }

  // A use stands where a shared block is shown. The shared block is kept
  // once; what this place put into its slots is kept here.
  use(shared, slots) {
    const block = this.block(-1, 0);
    block.shared = shared;
    block.slots = slots;
    return block;
  }
}

// A scope is what a template can see around itself.
//   page      the page being computed: { params, url }; null inside a shared block
//   slots     what the caller put into the slots: Map of name → blocks; null for none
//   provided  what the page gives to its layouts: Map of name → blocks; null inside a shared block
//   owner     the name of the shared block being computed, or null

// Writer is "$$" in generated code: it writes the values of one block.
class Writer {
  constructor(run, record, block, scope) {
    this.run = run;
    this.record = record;
    this.block = block;
    this.scope = scope;
  }

  // {expression}
  text(value) {
    this.block.values.push(textOf(value));
  }

  // <g-html value={expression} />
  html(value) {
    this.block.values.push(value == null || value === false ? "" : clean(String(value)));
  }

  // name={expression}
  attr(name, value) {
    this.block.values.push(attributeOf(name, value));
  }

  // {...attributes}
  spread(attributes) {
    const values = this.block.values;
    const at = values.length;
    values.push(0);
    if (attributes == null) return;
    if (typeof attributes !== "object") throw new TypeError(`{...} among attributes expects an object, got ${describe(attributes)}`);
    let count = 0;
    for (const name of Object.keys(attributes)) {
      values.push(clean(name), attributeOf(name, attributes[name]));
      count++;
    }
    values[at] = count;
  }

  // Which branch of <g-if> … <g-else> was taken, from 1; 0 for none.
  branch(number) {
    this.block.values.push(number);
  }

  // <g-for>: the number of items goes before the items, and is known after them.
  loop() {
    return this.block.values.push(0) - 1;
  }

  count(at, items) {
    this.block.values[at] = items;
  }

  items(list) {
    if (list == null) return [];
    if (typeof list[Symbol.iterator] !== "function") throw new TypeError(`<g-for each={…}> expects a list, got ${describe(list)}`);
    return list;
  }

  // <Component prop={…}>markup</Component>
  async call(target, props, fills) {
    if (target === null || typeof target !== "object" || typeof target.render !== "function") {
      throw new TypeError(`this tag is not a component: it is ${describe(target)}. A component is what a .shugo file exports: import Card from "./Card.shugo"`);
    }
    // What stands inside the tag is computed here, in the template that
    // wrote it, whether or not the component shows it.
    let slots = null;
    if (fills !== undefined) {
      slots = new Map();
      for (const [name, part, body] of fills) {
        const fill = this.record.block(this.block.file, part);
        await body(new Writer(this.run, this.record, fill, this.scope));
        slots.set(name, [fill.index]);
      }
    }
    this.block.children.push(await this.run.show(this.record, target, props, slots, this.scope));
  }

  // <g-slot name="…">: tells whether the caller put something there.
  slot(name) {
    const filled = this.scope.slots !== null && this.scope.slots.has(name);
    this.block.values.push(filled ? 1 : 0);
    return filled;
  }

  // slot="name" at the top of a page: markup for the layouts around it.
  async provide(name, part, body) {
    const fill = this.record.block(this.block.file, part);
    await body(new Writer(this.run, this.record, fill, this.scope));
    const provided = this.scope.provided;
    if (provided.has(name)) provided.get(name).push(fill.index);
    else provided.set(name, [fill.index]);
  }
}

// Context is "Shugo" in the script and in the expressions of a template.
class Context {
  #scope;

  constructor(component, props, scope) {
    this.props = props;
    this.self = component;
    this.#scope = scope;
  }

  get slots() {
    const slots = this.#scope.slots;
    return { has: (name) => slots !== null && slots.has(name) };
  }

  get params() {
    return this.#page("params").params;
  }

  get url() {
    const page = this.#page("url");
    return (page.url ??= new URL(encodeURI(page.path), page.origin));
  }

  #page(what) {
    if (this.#scope.page === null) {
      throw new Error(`Shugo.${what} was read inside a block that is computed once for all pages. This should not happen: please report it`);
    }
    return this.#scope.page;
  }
}

// ---------------------------------------------------------------------------
// Values

// The Go side splits what it receives by these characters, so no value may hold them.
const separators = /[\x00-\x03]/;
const everySeparator = /[\x00-\x03]/g;

function clean(text) {
  return separators.test(text) ? text.replace(everySeparator, "") : text;
}

// What {value} shows: nothing for null, undefined and false, the items of a
// list one after another, and otherwise what String() gives.
function textOf(value) {
  if (typeof value === "string") return clean(value);
  if (value == null || value === false) return "";
  if (Array.isArray(value)) {
    let out = "";
    for (const item of value) out += textOf(item);
    return out;
  }
  return clean(String(value));
}

// Attributes that are either there or not: disabled, hidden.
const flags = new Set([
  "allowfullscreen", "async", "autofocus", "autoplay", "checked", "controls", "default", "defer", "disabled",
  "formnovalidate", "hidden", "inert", "ismap", "itemscope", "loop", "multiple", "muted", "nomodule", "novalidate",
  "open", "playsinline", "readonly", "required", "reversed", "selected",
]);

// Attributes where "false" is a value of its own: aria-expanded="false".
const spelled = new Set(["contenteditable", "draggable", "spellcheck"]);

// What name={value} gives: "-" for no attribute, "+" for the name alone, "="
// and the text for a value.
//   null, undefined   no attribute
//   true              the name alone for disabled and the like, "true" otherwise
//   false             no attribute; "false" for aria-*, data-* and a few like draggable
function attributeOf(name, value) {
  if (typeof value === "string") return "=" + clean(value);
  if (value == null) return "-";
  if (value === true) return flags.has(name) ? "+" : "=true";
  if (value === false) return spelled.has(name) || name.startsWith("aria-") || name.startsWith("data-") ? "=false" : "-";
  return "=" + clean(String(value));
}

function describe(value) {
  if (value === null) return "null";
  if (Array.isArray(value)) return "a list";
  if (typeof value === "string") return `the string ${JSON.stringify(value.length > 40 ? value.slice(0, 40) + "…" : value)}`;
  if (typeof value === "object") return "an object";
  return typeof value === "undefined" ? "undefined" : `${typeof value} ${String(value)}`;
}

// ---------------------------------------------------------------------------
// The run

// component is what every .shugo file exports.
export function component(file, render) {
  return { file, render };
}

class Run {
  constructor(site, modules, send) {
    this.site = site;
    this.modules = modules; // Map of file number → module, for pages and layouts
    this.send = send;
    this.shared = new Map(); // name → promise of the number of the shared block
    this.numbers = new Map(); // name → the number, once the block is computed
    this.waits = new Map(); // which shared block every shared block being computed waits for
    this.claims = new Map(); // address → route
    this.batch = [];
    this.batchSize = 0;
    this.counts = { pages: 0, shared: 0, blocks: 0, shadowed: 0 };
  }

  // show computes a component at one place of a record and returns the block
  // that stands for it there.
  async show(record, target, props, slots, scope) {
    const name = this.site.files[target.file].shared ? sharedName(this.site.files[target.file].path, props, slots) : null;
    if (name !== null) {
      // Most uses of a shared block come after it has been computed.
      const number = this.numbers.get(name) ?? (await this.share(name, target, props, slots, scope));
      return record.use(number, slots).index;
    }
    const block = record.block(target.file, 0);
    block.slots = slots;
    const inner = { page: scope.page, slots, provided: scope.provided, owner: scope.owner };
    await target.render(new Writer(this, record, block, inner), new Context(target, props, inner));
    return block.index;
  }

  // share returns the number of a shared block, computing it at the first call.
  async share(name, target, props, slots, scope) {
    let pending = this.shared.get(name);
    if (pending === undefined) {
      pending = this.compute(name, target, props, slots);
      this.shared.set(name, pending);
    }
    if (scope.owner === null) return pending;
    // One shared block is waiting for another: make sure that one is not
    // waiting for this one in turn.
    for (let at = name; at !== undefined; at = this.waits.get(at)) {
      if (at === scope.owner) throw new Error(`${name} shows itself again with the same parameters, which would never end`);
    }
    this.waits.set(scope.owner, name);
    try {
      return await pending;
    } finally {
      this.waits.delete(scope.owner);
    }
  }

  async compute(name, target, props, slots) {
    const number = this.counts.shared++;
    const record = new Record("s", name, number);
    const block = record.block(target.file, 0);
    // Inside a shared block only the names of the slots matter: what fills
    // them belongs to the place the block is used at.
    const inner = { page: null, slots, provided: null, owner: name };
    await target.render(new Writer(this, record, block, inner), new Context(target, props, inner));
    this.emit(record);
    this.numbers.set(name, number);
    return number;
  }

  // page computes one page: the page itself, then every layout around it,
  // from the innermost.
  async page(index, route, entry, path) {
    const record = new Record("p", path, index);
    const params = Object.freeze({ ...entry.params });
    const scope = { page: { params, path, origin: this.site.origin, url: undefined }, slots: null, provided: new Map(), owner: null };
    let top = await this.show(record, this.modules.get(route.file).default, entry.props ?? {}, null, scope);
    for (let i = route.layouts.length - 1; i >= 0; i--) {
      // A layout shows the page in its default slot. What the page, or a
      // layout inside this one, gave for a slot goes to the nearest layout
      // that has a slot of that name.
      const layout = this.modules.get(route.layouts[i]).default;
      const slots = new Map([["default", [top]]]);
      for (const name of this.site.files[layout.file].slots ?? []) {
        if (name !== "default" && scope.provided.has(name)) {
          slots.set(name, scope.provided.get(name));
          scope.provided.delete(name);
        }
      }
      top = await this.show(record, layout, {}, slots, scope);
    }
    record.top = top;
    this.counts.pages++;
    this.emit(record);
  }

  emit(record) {
    this.counts.blocks += record.blocks.length;
    const text = encode(record);
    this.batch.push(text);
    if ((this.batchSize += text.length) > 1 << 18) this.flush();
  }

  flush() {
    if (this.batch.length === 0) return;
    this.send("R", this.batch.join("\x03"));
    this.batch = [];
    this.batchSize = 0;
  }
}

// A component that depends on nothing of the page and is called with plain
// values shows the same wherever it stands, so it is computed once under a
// name made of its file and those values: components/Card.shugo{"id":"p7"}.
// null means: not shared. JSON writes every control character as an escape,
// so a name never holds one of the separators.
function sharedName(path, props, slots) {
  let key = "";
  let count = 0;
  for (const name in props) {
    const value = props[name];
    if (value === undefined) continue;
    const type = typeof value;
    if (value !== null && type !== "string" && type !== "boolean" && !(type === "number" && Number.isFinite(value))) return null;
    key += (count++ === 0 ? "" : ",") + JSON.stringify(name) + ":" + JSON.stringify(value);
  }
  // Parameters in another order are the same parameters.
  if (count > 1) {
    key = Object.keys(props).filter((name) => props[name] !== undefined).sort()
      .map((name) => JSON.stringify(name) + ":" + JSON.stringify(props[name])).join(",");
  }
  if (slots !== null) key += "|" + [...slots.keys()].sort().join(",");
  return path + "{" + key + "}";
}

// On the wire a record is its header and its blocks, split by \x01; fields
// are split by \x02, and every value of a block follows a \x00.
function encode(record) {
  const lines = [record.kind + "\x02" + record.key + "\x02" + record.origin + "\x02" + record.top];
  for (const block of record.blocks) {
    lines.push(
      block.shared >= 0
        ? "u\x02" + block.shared + "\x02" + slotsOf(block.slots)
        : "b\x02" + block.file + "\x02" + block.part + "\x02" + block.children.join(",") + "\x02" + slotsOf(block.slots) + "\x02" + valuesOf(block.values),
    );
  }
  return lines.join("\x01");
}

// Every value is written after \x00, so that no values and one empty value differ.
function valuesOf(values) {
  return values.length === 0 ? "" : "\x00" + values.join("\x00");
}

function slotsOf(slots) {
  if (slots === null) return "";
  let out = "";
  for (const [name, blocks] of slots) out += (out ? ";" : "") + name + "=" + blocks.join(",");
  return out;
}

// ---------------------------------------------------------------------------
// Routes

// address builds the address of a page from the route and the parameters
// that getStaticPaths returned for it.
function address(route, params) {
  const parts = [];
  for (const segment of route.segments) {
    if (segment.text !== undefined) {
      parts.push(segment.text);
      continue;
    }
    const name = segment.param ?? segment.rest;
    let value = params[name];
    if (typeof value === "number") value = String(value);
    if (segment.rest !== undefined && (value === undefined || value === "")) continue;
    if (typeof value !== "string" || value === "") {
      throw new Error(`getStaticPaths of ${route.pattern} returned params without "${name}": it must be a text that is not empty, got ${describe(params[name])}`);
    }
    for (const part of segment.rest !== undefined ? value.split("/") : [value]) {
      if (part === "" || part === "." || part === ".." || /[\x00-\x1f\\/?#]/.test(part)) {
        throw new Error(`getStaticPaths of ${route.pattern} returned ${name}: ${JSON.stringify(value)}, which cannot be a part of an address`);
      }
      parts.push(part);
    }
  }
  return "/" + parts.join("/");
}

// paginate is what getStaticPaths({ paginate }) receives: it cuts a list into
// pages the way Astro does. The route must have a [page] or [...page] parameter.
function paginator(route) {
  return (data, { pageSize = 10, params = {}, props = {} } = {}) => {
    const segment = route.segments.find((s) => (s.param ?? s.rest) === "page");
    if (segment === undefined) throw new Error(`paginate() needs a [page] or [...page] parameter in the name of the page file, and ${route.pattern} has none`);
    const last = Math.max(1, Math.ceil(data.length / pageSize));
    // With [...page] the first page has no number in its address.
    const number = (page) => (segment.rest !== undefined && page === 1 ? undefined : String(page));
    const link = (page) => (page < 1 || page > last ? undefined : address(route, { ...params, page: number(page) }));
    return Array.from({ length: last }, (_, i) => {
      const start = i * pageSize;
      const end = Math.min(start + pageSize, data.length);
      return {
        params: { ...params, page: number(i + 1) },
        props: {
          ...props,
          page: {
            data: data.slice(start, end), start, end: end - 1, size: pageSize, total: data.length,
            currentPage: i + 1, lastPage: last,
            url: { current: link(i + 1), prev: link(i), next: link(i + 2), first: i > 0 ? link(1) : undefined, last: i + 1 < last ? link(last) : undefined },
          },
        },
      };
    });
  };
}

// entries returns the pages of a route: one for a page without parameters,
// and what getStaticPaths says for the others.
async function entries(route, module) {
  if (!route.segments.some((segment) => segment.text === undefined)) return [{ params: {}, props: {} }];
  if (typeof module.getStaticPaths !== "function") throw new Error(`${route.pattern} has parameters, so its script must export getStaticPaths`);
  const list = await module.getStaticPaths({ paginate: paginator(route) });
  if (!Array.isArray(list)) throw new Error(`getStaticPaths of ${route.pattern} must return a list of { params, props }, got ${describe(list)}`);
  for (const entry of list) {
    if (entry === null || typeof entry !== "object" || entry.params === null || typeof entry.params !== "object") {
      throw new Error(`getStaticPaths of ${route.pattern} must return a list of { params: { … }, props: { … } }, and one item is ${describe(entry)}`);
    }
  }
  return list;
}

// ---------------------------------------------------------------------------
// The entry point

// How many pages are computed at once. More does not use more processor,
// only lets pages wait for their data together.
const concurrency = 32;

// run computes every page of the site. modules are the compiled page and
// layout files by their numbers.
export async function run(modules) {
  const out = Number(process.env.SHUGO_OUT);
  if (!Number.isInteger(out)) {
    console.error("This file is a part of a Shugo project and is run by the shugo command: `shugo build` or `shugo update`.");
    process.exit(2);
  }
  const send = (kind, text) => {
    const body = Buffer.from(kind + text);
    const head = Buffer.allocUnsafe(4);
    head.writeUInt32LE(body.length, 0);
    for (const buffer of [head, body]) {
      for (let sent = 0; sent < buffer.length; ) sent += writeSync(out, buffer, sent, buffer.length - sent);
    }
  };
  let place = "";
  try {
    const site = JSON.parse(readFileSync(new URL("./site.json", import.meta.url), "utf8"));
    const run = new Run(site, new Map(modules), send);

    // First the list of all pages: every route says which pages it has. A
    // route that names more of an address comes earlier and takes it.
    const pages = [];
    for (const [index, route] of site.routes.entries()) {
      place = `getStaticPaths of ${site.files[route.file].path}`;
      for (const entry of await entries(route, run.modules.get(route.file))) {
        const path = address(route, entry.params);
        const owner = run.claims.get(path);
        if (owner === index) throw new Error(`getStaticPaths of ${route.pattern} returned the address ${path} twice`);
        if (owner !== undefined) {
          run.counts.shadowed++;
          continue;
        }
        run.claims.set(path, index);
        pages.push({ index, route, entry, path });
      }
    }

    // Then the pages themselves, several at a time.
    let next = 0;
    const worker = async () => {
      while (next < pages.length) {
        const { index, route, entry, path } = pages[next++];
        place = "";
        try {
          await run.page(index, route, entry, path);
        } catch (error) {
          next = pages.length; // the others stop after the page they are at
          throw Object.assign(error instanceof Error ? error : new Error(String(error)), { place: `the page ${path} (${site.files[route.file].path})` });
        }
      }
    };
    await Promise.all(Array.from({ length: concurrency }, worker));
    run.flush();
    send("D", JSON.stringify(run.counts));
  } catch (error) {
    send("E", JSON.stringify({ message: String(error?.message ?? error), stack: String(error?.stack ?? ""), place: error?.place ?? place }));
    process.exit(1);
  }
}

// Authors who know Astro reach for its global by habit.
globalThis.Astro ??= new Proxy({}, {
  get(_, name) {
    throw new Error(`There is no Astro.${String(name)} in Shugo: write Shugo.${String(name)} (Shugo.props, Shugo.params, Shugo.url, Shugo.slots.has())`);
  },
});
