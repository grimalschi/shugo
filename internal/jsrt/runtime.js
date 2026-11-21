// The JavaScript side of Shugo. The code generated from templates calls this
// library while it computes pages: it runs the scripts of pages and
// components, collects the value of every expression in the order of the
// template, and hands the result to the Go side, which keeps it and builds
// HTML from it. No markup is produced here.
//
// The Go side leads. It asks which pages the project has, then names the
// pages and the shared blocks to compute; this side answers each request and
// waits for the next.
//
// Runs on Bun and on Node.

import { readFileSync, readSync, writeSync } from "node:fs";

import { during, open, Query, Reads } from "shugo:query";

// The data database of the project, once it is open; null if it has none.
let data = null;

// ---------------------------------------------------------------------------
// What is computed

// A record is everything computed for one page, or for one shared block: its
// blocks in the order they were started. Blocks of a record name each other
// by their place in it.
class Record {
  constructor(kind, key, route) {
    this.kind = kind; // "p" for a page, "s" for a shared block
    this.key = key; // the address of the page, or the name of the shared block
    this.route = route; // the route of a page, by its place among the routes
    this.blocks = [];
    this.top = 0;
    // What was asked of the data database on the way. The rows it gave note
    // what is read of them until the record is complete: see encode.
    this.reads = new Reads();
  }

  // A block is one template being shown: a page, a layout, a component, or
  // markup put into a component (a part of the template it is written in).
  block(file, part) {
    const block = { index: this.blocks.length, file, part, values: [], children: [], slots: null, shared: null };
    this.blocks.push(block);
    return block;
  }

  // A use stands where a shared block is shown, and names it. The shared
  // block is kept once; what this place put into its slots is kept here.
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
//   reads     where the queries to the data database leave their readings

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
  call(target, props, fills) {
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
        body(new Writer(this.run, this.record, fill, this.scope));
        slots.set(name, [fill.index]);
      }
    }
    this.block.children.push(this.run.show(this.record, target, props, slots, this.scope));
  }

  // <g-slot name="…">: tells whether the caller put something there.
  slot(name) {
    const filled = this.scope.slots !== null && this.scope.slots.has(name);
    this.block.values.push(filled ? 1 : 0);
    return filled;
  }

  // slot="name" at the top of a page: markup for the layouts around it.
  provide(name, part, body) {
    const fill = this.record.block(this.block.file, part);
    body(new Writer(this.run, this.record, fill, this.scope));
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

  // query starts a question to a table of the data database. What it reads
  // is noted for the page or the shared block being computed.
  query(table) {
    return query(this.#scope.reads, table);
  }

  // paginate cuts the rows of a query into pages and returns the page whose
  // number stands in the address, the way getStaticPaths({ paginate }) gives
  // it in Astro: { data, total, currentPage, lastPage, url: { prev, next, … } }.
  paginate(rows, { pageSize = 10 } = {}) {
    if (!(rows instanceof Query)) throw new TypeError(`Shugo.paginate() takes a query: Shugo.paginate(Shugo.query("posts").order("date", "desc"), { pageSize: 10 })`);
    const { route, params } = this.#page("paginate");
    const segment = pageSegment(route);
    const total = rows.count();
    const current = params.page === undefined ? 1 : Number(params.page);
    const start = (current - 1) * pageSize;
    const found = rows.limit(pageSize).offset(start).all();
    return paged(route, segment, params, found, start, pageSize, total, current);
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

// query starts a question to the data database for one reader.
function query(reads, table) {
  if (data === null) {
    throw new Error('Shugo.query reads the data database of the project, and this project names none. Name it in shugo.json: { "data": "data/site.db" }');
  }
  return new Query(data, reads, table);
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
    this.modules = modules; // Map of file number → module
    this.send = send;
    this.files = new Map(site.files.map((file, number) => [file.path, number]));
    this.pages = new Map(); // address → { index, route, entry }, once the pages are listed
    this.batch = [];
    this.batchSize = 0;
  }

  // list finds the pages of routes: of the ones named, or of all. Every
  // route says which pages it has, and what it read from the data to know.
  // Among the routes listed together, a route that names more of an address
  // comes earlier and takes it.
  list(wanted) {
    const routes = wanted.length > 0 ? wanted : this.site.routes.map((_, index) => index);
    for (const [path, page] of this.pages) if (routes.includes(page.index)) this.pages.delete(path);
    let shadowed = 0;
    let lines = [];
    for (const index of routes) {
      const route = this.site.routes[index];
      if (route === undefined) throw new Error(`the shugo command asked for the pages of route ${index}, which this project's compiled code does not have: build the project again`);
      const reads = new Reads();
      let props = false;
      try {
        for (const entry of during(reads, () => entries(route, this.modules.get(route.file), reads))) {
          const path = address(route, entry.params);
          const taken = this.pages.get(path);
          if (taken !== undefined && taken.index === index) throw new Error(`getStaticPaths of ${route.pattern} returned the address ${path} twice`);
          if (taken !== undefined && taken.index < index) {
            shadowed++;
            continue;
          }
          props ||= entry.props != null && Object.keys(entry.props).length > 0;
          this.pages.set(path, { index, route, entry });
          if (lines.push(index + "\x02" + path) === 4096) {
            this.send("P", lines.join("\x03"));
            lines = [];
          }
        }
      } catch (error) {
        throw placed(error, `getStaticPaths of ${this.site.files[route.file].path}`);
      }
      // Rows that the list gives to its pages are read by the pages later,
      // so every column of what the list asked for counts as read.
      this.send("G", index + "\x02" + (props ? 1 : 0) + "\x02" + reads.text(props));
    }
    if (lines.length > 0) this.send("P", lines.join("\x03"));
    return { shadowed };
  }

  // compute computes the units the Go side named, several at a time. A unit
  // is "p", the address of a page and its route, or "s" and the name of a
  // shared block, split by \x02.
  //
  // A page that fails ends the run. A shared block that fails is told to the
  // Go side and the run goes on: after a change of data a block may be
  // computed that nothing shows any more, and only the Go side knows.
  compute(units) {
    for (const unit of units) {
      const [kind, name, route] = unit.split("\x02");
      if (kind === "s") {
        try {
          this.block(name);
        } catch (error) {
          this.send("F", name + "\x02" + failure(placed(error, `the shared block ${name}`)));
        }
        continue;
      }
      try {
        this.page(name, Number(route));
      } catch (error) {
        throw placed(error, `the page ${name} (${this.site.files[this.site.routes[Number(route)]?.file]?.path})`);
      }
    }
    this.flush();
  }

  // show computes a component at one place of a record and returns the block
  // that stands for it there. Where the component gives a shared block, that
  // block only gets named: the Go side knows whether it has to be computed.
  show(record, target, props, slots, scope) {
    const name = this.site.files[target.file].shared ? sharedName(this.site.files[target.file].path, props, slots) : null;
    if (name !== null) return record.use(name, slots).index;
    const block = record.block(target.file, 0);
    block.slots = slots;
    const inner = { page: scope.page, slots, provided: scope.provided, reads: scope.reads };
    target.render(new Writer(this, record, block, inner), new Context(target, props, inner));
    return block.index;
  }

  // block computes a shared block from its name alone: the name holds the
  // file, the parameters and the names of the slots.
  block(name) {
    const { path, props, slots } = unnamed(name);
    const target = this.modules.get(this.files.get(path))?.default;
    if (target === undefined) throw new Error(`there is no file ${path} to compute the shared block from`);
    const record = new Record("s", name, 0);
    const block = record.block(target.file, 0);
    // Inside a shared block only the names of the slots matter: what fills
    // them belongs to the place the block is used at.
    const inner = { page: null, slots, provided: null, reads: record.reads };
    during(record.reads, () => target.render(new Writer(this, record, block, inner), new Context(target, props, inner)));
    this.emit(record);
  }

  // page computes one page of a route: the page itself, then every layout
  // around it, from the innermost. A page of a route that was listed in this
  // run gets what the list gave it beside its address.
  //
  // The parameters are always read from the address. A page is computed now
  // from a list, now from its address alone, and must come out the same:
  // what getStaticPaths wrote as a number is a text in the address.
  page(path, index) {
    const route = this.site.routes[index];
    if (route === undefined) throw new Error(`the page ${path} was asked for under route ${index}, which this project's compiled code does not have: build the project again`);
    const listed = this.pages.get(path);
    const props = listed !== undefined && listed.index === index ? listed.entry.props ?? {} : {};
    const record = new Record("p", path, index);
    const params = Object.freeze(parameters(route, path));
    const scope = { page: { params, path, route, origin: this.site.origin, url: undefined }, slots: null, provided: new Map(), reads: record.reads };
    during(record.reads, () => this.around(record, route, props, scope));
    this.emit(record);
  }

  // around computes a page with its layouts into the record.
  around(record, route, props, scope) {
    let top = this.show(record, this.modules.get(route.file).default, props, null, scope);
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
      top = this.show(record, layout, {}, slots, scope);
    }
    record.top = top;
  }

  // emit hands a complete record to the Go side. Writing it settles what the
  // record read: from here on nothing may read its rows.
  emit(record) {
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

// unnamed reads a name that sharedName wrote. The names of slots follow the
// last "|" and are made of letters, digits, "-" and "_"; a "|" inside a
// parameter is followed by the quote that closes its text at the least.
function unnamed(name) {
  const start = name.indexOf("{");
  let inside = name.slice(start + 1, -1);
  let slots = null;
  const bar = inside.lastIndexOf("|");
  if (bar >= 0 && /^[\w,-]*$/.test(inside.slice(bar + 1))) {
    slots = new Map(inside.slice(bar + 1).split(",").filter((slot) => slot !== "").map((slot) => [slot, []]));
    inside = inside.slice(0, bar);
  }
  return { path: name.slice(0, start), props: JSON.parse("{" + inside + "}"), slots };
}

// On the wire a record is its header and its blocks, split by \x01; fields
// are split by \x02, and every value of a block follows a \x00. The header
// ends with what the record read from the data database. A block that stands
// for a shared block carries its name.
function encode(record) {
  const lines = [record.kind + "\x02" + record.key + "\x02" + record.route + "\x02" + record.top + "\x02" + record.reads.text()];
  for (const block of record.blocks) {
    lines.push(
      block.shared !== null
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

// parameters reads the parameters of a page from its address: what address()
// does, backwards. The route is known, so the address fits it.
function parameters(route, path) {
  const parts = path === "/" ? [] : path.slice(1).split("/");
  const params = {};
  let at = 0;
  for (const segment of route.segments) {
    if (segment.text !== undefined) at++;
    else if (segment.param !== undefined) params[segment.param] = parts[at++];
    else {
      const rest = parts.slice(at, parts.length - (route.segments.length - route.segments.indexOf(segment) - 1));
      params[segment.rest] = rest.length === 0 ? undefined : rest.join("/");
      at += rest.length;
    }
  }
  return params;
}

// A list is cut into pages by a [page] or [...page] parameter of the route.
function pageSegment(route) {
  const segment = route.segments.find((s) => (s.param ?? s.rest) === "page");
  if (segment === undefined) throw new Error(`paginate() needs a [page] or [...page] parameter in the name of the page file, and ${route.pattern} has none`);
  return segment;
}

// With [...page] the first page has no number in its address.
const pageNumber = (segment, page) => (segment.rest !== undefined && page === 1 ? undefined : String(page));

// paged describes one page of a list the way Astro does.
function paged(route, segment, params, rows, start, size, total, current) {
  const last = Math.max(1, Math.ceil(total / size));
  const link = (page) => (page < 1 || page > last ? undefined : address(route, { ...params, page: pageNumber(segment, page) }));
  return {
    data: rows, start, end: Math.min(start + size, total) - 1, size, total,
    currentPage: current, lastPage: last,
    url: { current: link(current), prev: link(current - 1), next: link(current + 1), first: current > 1 ? link(1) : undefined, last: current < last ? link(last) : undefined },
  };
}

// paginate is what getStaticPaths({ paginate }) receives: it cuts a list into
// pages the way Astro does. Given the list, every page gets its part of it.
// Given only how many items there are, the pages get their addresses alone,
// and each asks for its part itself: Shugo.paginate(query, { pageSize }).
function paginator(route) {
  return (list, { pageSize = 10, params = {}, props = {} } = {}) => {
    const segment = pageSegment(route);
    const total = typeof list === "number" ? list : list.length;
    const last = Math.max(1, Math.ceil(total / pageSize));
    return Array.from({ length: last }, (_, i) => {
      const entry = { params: { ...params, page: pageNumber(segment, i + 1) } };
      if (typeof list !== "number") {
        const start = i * pageSize;
        entry.props = { ...props, page: paged(route, segment, params, list.slice(start, start + pageSize), start, pageSize, total, i + 1) };
      } else if (Object.keys(props).length > 0) {
        entry.props = { ...props };
      }
      return entry;
    });
  };
}

// entries returns the pages of a route: one for a page without parameters,
// and what getStaticPaths says for the others.
function entries(route, module, reads) {
  if (!route.segments.some((segment) => segment.text === undefined)) return [{ params: {} }];
  if (typeof module.getStaticPaths !== "function") throw new Error(`${route.pattern} has parameters, so its script must export getStaticPaths`);
  const list = module.getStaticPaths({ paginate: paginator(route), query: (table) => query(reads, table) });
  if (typeof list?.then === "function") throw new Error(`getStaticPaths of ${route.pattern} waits (it is async or returns a promise): pages are computed without waiting; the data comes from the data database through query`);
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


// placed notes what was being computed when an error came up.
function placed(error, place) {
  const failure = error instanceof Error ? error : new Error(String(error));
  failure.place ??= place;
  return failure;
}

// failure writes an error for the Go side.
function failure(error) {
  return JSON.stringify({ message: String(error?.message ?? error), stack: String(error?.stack ?? ""), place: error?.place ?? "" });
}

// run answers the requests of the Go side until it has no more. modules are
// the compiled files by their numbers.
//
// A message in either direction is a length in four bytes, a letter that
// says what the message is, and its body.
export async function run(modules) {
  const [input, output] = [Number(process.env.SHUGO_IN), Number(process.env.SHUGO_OUT)];
  if (!Number.isInteger(input) || !Number.isInteger(output)) {
    console.error("This file is a part of a Shugo project and is run by the shugo command: `shugo build` or `shugo update`.");
    process.exit(2);
  }
  const send = (kind, text) => {
    const body = Buffer.from(kind + text);
    const head = Buffer.allocUnsafe(4);
    head.writeUInt32LE(body.length, 0);
    for (const buffer of [head, body]) {
      for (let sent = 0; sent < buffer.length; ) sent += writeSync(output, buffer, sent, buffer.length - sent);
    }
  };
  // Nothing happens here between two requests, so waiting for the next one
  // may hold everything else up.
  const fill = (buffer) => {
    for (let got = 0; got < buffer.length; ) {
      const count = readSync(input, buffer, got, buffer.length - got, null);
      if (count === 0) return false;
      got += count;
    }
    return true;
  };
  const receive = () => {
    const head = Buffer.allocUnsafe(4);
    if (!fill(head)) return null;
    const body = Buffer.allocUnsafe(head.readUInt32LE(0));
    return fill(body) ? { kind: String.fromCharCode(body[0]), text: body.toString("utf8", 1) } : null;
  };

  try {
    const site = JSON.parse(readFileSync(new URL("./site.json", import.meta.url), "utf8"));
    if (process.env.SHUGO_DATA) {
      data = await open(process.env.SHUGO_DATA, {
        declare: (number, spec) => send("S", number + "\x02" + spec),
        // The answer is waited for: the question that needs the index is
        // asked right after it.
        index(question) {
          send("I", question);
          const answer = receive();
          if (answer?.kind !== "I") throw new Error("the shugo command did not answer a request for an index: build the project again");
          return answer.text === "1";
        },
      });
    }
    const run = new Run(site, new Map(modules), send);
    for (let request = receive(); request !== null && request.kind !== "Q"; request = receive()) {
      switch (request.kind) {
        case "L":
          send("D", JSON.stringify(run.list(request.text === "" ? [] : request.text.split(",").map(Number))));
          break;
        case "C":
          run.compute(request.text.split("\x03"));
          send("D", "{}");
          break;
        case "K":
          // What the Go side knows from earlier runs: it needs no answer.
          data?.know(JSON.parse(request.text));
          break;
        default:
          throw new Error(`the shugo command asked for "${request.kind}", which this project's compiled code does not know: build the project again`);
      }
    }
  } catch (error) {
    send("E", failure(error));
    process.exit(1);
  }
}

// Authors who know Astro reach for its global by habit.
globalThis.Astro ??= new Proxy({}, {
  get(_, name) {
    throw new Error(`There is no Astro.${String(name)} in Shugo: write Shugo.${String(name)} (Shugo.props, Shugo.params, Shugo.url, Shugo.slots.has())`);
  },
});
