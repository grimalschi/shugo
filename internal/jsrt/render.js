// The render side of Shugo's JavaScript. The code generated from templates
// calls this library while it computes pages: it collects the value of
// every expression in the order of the template. No markup is produced
// here; the Go side builds HTML from the values.
//
// Nothing of the project's data exists on this side. A template of a file
// that loads data is given a copy of exactly what it reads of the variables
// of its script, made on the load side; a component that only shows what it
// is given gets it from the template that calls it. So whatever is computed
// here depends on what was handed in and on nothing else, and nothing here
// waits for anything.
//
// Runs on Bun and on Node.

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
//   parts     what the templates of the files that load data are given: Map of file → copy

// Writer is "$$" in generated code: it writes the values of one block.
class Writer {
  constructor(renderer, record, block, scope) {
    this.renderer = renderer;
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
    // A component that the script of the file has chosen comes as the
    // number of its file.
    if (Number.isInteger(target?.shugoComponent)) target = this.renderer.modules.get(target.shugoComponent)?.default;
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
        body(new Writer(this.renderer, this.record, fill, this.scope));
        slots.set(name, [fill.index]);
      }
    }
    this.block.children.push(this.renderer.show(this.record, target, props, slots, this.scope));
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
    body(new Writer(this.renderer, this.record, fill, this.scope));
    const provided = this.scope.provided;
    if (provided.has(name)) provided.get(name).push(fill.index);
    else provided.set(name, [fill.index]);
  }
}

// Context is "Shugo" in the expressions of a template, and in the script of
// a component that only shows what it is given.
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

  // The data database is asked on the load side, by the script of a page, a
  // layout or a shared block. The compiler sends there every script that
  // asks; what comes here is a template, or a call the compiler did not see.
  query() {
    throw new Error("Shugo.query is called in a template, or in code that a template calls. A template shows what its script gives it and loads nothing itself: ask in the script and show the variable");
  }

  paginate() {
    throw new Error("Shugo.paginate is called in a template, or in code that a template calls. A template shows what its script gives it and loads nothing itself: ask in the script and show the variable");
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
// The files

// component is what a file exports that only shows what it is given: its
// script and its template run together, on the parameters of the call.
export function component(file, render) {
  return { file, render, loads: false };
}

// template is what a file exports that loads data: only its template is
// here, and it runs on the copy of what it reads, made on the load side.
export function template(file, render) {
  return { file, render, loads: true };
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
    if (!plain(value)) return null;
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

// plain tells whether a parameter is a plain value: one that can stand in a name.
function plain(value) {
  const type = typeof value;
  return value === null || type === "string" || type === "boolean" || (type === "number" && Number.isFinite(value));
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
// are split by \x02, and every value of a block follows a \x00. A block that
// stands for a shared block carries its name. The header ends with what the
// load side knows of the record: the fingerprint of what it was computed
// from, and what was read from the data database on the way.
function encode(record, print, reads) {
  const lines = [record.kind + "\x02" + record.key + "\x02" + record.route + "\x02" + record.top + "\x02" + print + "\x02" + reads];
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
// The renderer

class Renderer {
  constructor(site, modules) {
    this.site = site;
    this.modules = modules; // Map of file number → what the file exports
    this.files = new Map(site.files.map((file, number) => [file.path, number]));
  }

  // unnamed tells the load side what a name holds: it runs the script of a
  // shared block with the parameters that this side put into the name.
  unnamed(name) {
    return unnamed(name);
  }

  // show computes a component at one place of a record and returns the block
  // that stands for it there. Where the component gives a shared block, that
  // block only gets named: the Go side knows whether it has to be computed.
  show(record, target, props, slots, scope) {
    const file = this.site.files[target.file];
    const name = file.shared ? sharedName(file.path, props, slots) : null;
    if (name !== null) return record.use(name, slots).index;
    const block = record.block(target.file, 0);
    block.slots = slots;
    const inner = { page: scope.page, slots, provided: scope.provided, parts: scope.parts };
    this.fill(target, new Writer(this, record, block, inner), new Context(target, props, inner), scope.parts?.get(target.file), props);
    return block.index;
  }

  // fill runs the code of a file on one block. part is the copy of what the
  // template reads, for a file that loads data.
  fill(target, writer, context, part, props) {
    if (!target.loads) return target.render(writer, context);
    if (part !== undefined) return target.render(writer, context, part);
    // A component that loads data is computed once for the whole site,
    // from its name. Here it was called in a way that has no name.
    const { path, why } = this.site.files[target.file];
    const given = Object.keys(props ?? {}).find((name) => props[name] !== undefined && !plain(props[name]));
    throw new Error(
      `${path} loads data: its script ${why}. So it is computed once for the whole site and takes only plain values: text, numbers, true, false, null. ` +
        (given === undefined ? "" : `Here ${given} is ${describe(props[given])}. `) +
        `Pass what names the thing — an id — and let ${path} find it. Or load the data in the page and keep ${path} to showing what it is given: no await, no module that reads files or the network`,
    );
  }

  // block computes a shared block from its name: the name holds the file,
  // the parameters and the names of the slots.
  block(name, part, print, reads) {
    const { path, props, slots } = unnamed(name);
    const target = this.modules.get(this.files.get(path))?.default;
    if (target === undefined) throw new Error(`there is no file ${path} to compute the shared block from`);
    const record = new Record("s", name, 0);
    const block = record.block(target.file, 0);
    // Inside a shared block only the names of the slots matter: what fills
    // them belongs to the place the block is used at.
    const scope = { page: null, slots, provided: null, parts: null };
    this.fill(target, new Writer(this, record, block, scope), new Context(target, props, scope), part, props);
    return encode(record, print, reads);
  }

  // page computes one page: the page itself, then every layout around it,
  // from the innermost. parts holds what the template of the page is given,
  // and the same for every layout that is computed with the page.
  page(path, index, params, parts, print, reads) {
    const route = this.site.routes[index];
    const record = new Record("p", path, index);
    const scope = { page: { params, path, origin: this.site.origin, url: undefined }, slots: null, provided: new Map(), parts };
    let top = this.show(record, this.modules.get(route.file).default, {}, null, scope);
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
    return encode(record, print, reads);
  }
}

// renderer is what the render side gives to the load side: it computes a
// page or a shared block and returns its record as text.
export function renderer(site, modules) {
  return new Renderer(site, new Map(modules));
}

// ---------------------------------------------------------------------------
// The ways out

// The compiler puts these names where the code of the render side says
// fetch, Date.now, Math.random or process. A template that could ask the
// network or the clock would show something else tomorrow for the same data,
// and nobody would know that the page needs computing again.
const refuse = (what) => () => {
  throw new Error(`${what} in code that a template calls. A template works on the data of its script and must give the same for the same data: do this in the script of a page, a layout or a shared block and show the variable`);
};
globalThis.__shugoNoNetwork = refuse("fetch is called");
globalThis.__shugoNoClock = refuse("the time of the moment is asked for with Date.now()");
globalThis.__shugoNoChance = refuse("Math.random() is called");
globalThis.__shugoNoProcess = new Proxy({}, { get: refuse("the environment of the process is read") });

// Authors who know Astro reach for its global by habit.
globalThis.Astro ??= new Proxy({}, {
  get(_, name) {
    throw new Error(`There is no Astro.${String(name)} in Shugo: write Shugo.${String(name)} (Shugo.props, Shugo.params, Shugo.url, Shugo.slots.has())`);
  },
});
