// The load side of Shugo's JavaScript. It runs the scripts of pages, layouts
// and shared blocks with the data of the project, and it answers the Go
// side, which leads: which pages there are, and what the records of the
// pages and shared blocks named to it are.
//
// The compiler has worked out from the expressions of every template what
// the template reads of the variables of its script, and written that next
// to the script as a list of shapes. This side walks the live variables
// along those shapes and copies what they name. A template is computed on
// the render side, which is another bundle with modules of its own, and that
// copy is all it is given: a template cannot depend on anything else.
//
// The walk is also how this side knows what a page depends on. A project
// with a data database reads it through Shugo.query, and a row notes every
// column that is read of it: by the script, and by this walk. Those are the
// readings of the page, and nothing but a change of them has it computed
// again. A project without a database is told by fingerprints instead: the
// same walk makes one of exactly what the template reads, and a page whose
// fingerprint is what it was would show the same.
//
// Runs on Bun and on Node.

import { readFileSync, readSync, writeSync } from "node:fs";
import { createRequire } from "node:module";

import { open, Query, Reads } from "shugo:query";

// The data database of the project, once it is open; null if it has none.
let data = null;

// ---------------------------------------------------------------------------
// Fingerprints

// NotData is thrown when a template is found to read something that is not
// data: a function, an instance of a class.
class NotData extends Error {
  constructor(value) {
    super("not data");
    this.value = value;
  }
}

const float = new Float64Array(1);
const halves = new Uint32Array(float.buffer);

// The seeds of the two halves of a fingerprint.
const seedA = 0;
const seedB = 0x9747b28c | 0;

// How deep data may go. Data that contains itself goes deeper than anything.
const deepest = 500;

// The two digits of every byte, for writing a fingerprint as text.
const digits = Array.from({ length: 256 }, (_, byte) => byte.toString(16).padStart(2, "0"));

// hex writes the two halves of a fingerprint as sixteen digits.
export function hex(a, b) {
  return (
    digits[a >>> 24] + digits[(a >>> 16) & 255] + digits[(a >>> 8) & 255] + digits[a & 255] +
    digits[b >>> 24] + digits[(b >>> 16) & 255] + digits[(b >>> 8) & 255] + digits[b & 255]
  );
}

// Hasher makes a fingerprint of 64 bits out of the values fed to it:
// MurmurHash3 (x86, 32 bits) of the same words under two seeds. The words
// are the values themselves, each after a word that tells its kind, so that
// no two different sequences of values give the same words. Nothing is
// turned into text on the way.
export class Hasher {
  constructor() {
    this.reset();
  }

  reset() {
    this.a = seedA;
    this.b = seedB;
    this.words = 0;
    this.depth = 0;
  }

  // word takes the next four bytes.
  word(k) {
    k = Math.imul(k, 0xcc9e2d51);
    k = (k << 15) | (k >>> 17);
    k = Math.imul(k, 0x1b873593);
    let h = this.a ^ k;
    h = (h << 13) | (h >>> 19);
    this.a = (Math.imul(h, 5) + 0xe6546b64) | 0;
    h = this.b ^ k;
    h = (h << 13) | (h >>> 19);
    this.b = (Math.imul(h, 5) + 0xe6546b64) | 0;
    this.words++;
  }

  // finish makes the fingerprint of what was fed since reset: its two
  // halves are left in a and b.
  finish() {
    const length = this.words << 2;
    this.a = mix(this.a ^ length);
    this.b = mix(this.b ^ length);
  }

  // string takes text: its length, then two of its characters in a word.
  // This is where most of the time of a fingerprint goes, so the loop is
  // word() written out, with the two halves kept at hand.
  string(s) {
    const length = s.length;
    this.word(length);
    let a = this.a;
    let b = this.b;
    let i = 0;
    for (; i + 1 < length; i += 2) {
      let k = s.charCodeAt(i) | (s.charCodeAt(i + 1) << 16);
      k = Math.imul(k, 0xcc9e2d51);
      k = (k << 15) | (k >>> 17);
      k = Math.imul(k, 0x1b873593);
      a ^= k;
      a = (a << 13) | (a >>> 19);
      a = (Math.imul(a, 5) + 0xe6546b64) | 0;
      b ^= k;
      b = (b << 13) | (b >>> 19);
      b = (Math.imul(b, 5) + 0xe6546b64) | 0;
    }
    this.a = a;
    this.b = b;
    this.words += length >> 1;
    if (i < length) this.word(s.charCodeAt(i));
  }

  number(x) {
    if ((x | 0) === x) {
      this.word(2);
      this.word(x);
      return;
    }
    float[0] = x;
    this.word(3);
    this.word(halves[0]);
    this.word(halves[1]);
  }

  // value takes a value as it is, with everything in it.
  value(v) {
    switch (typeof v) {
      case "string":
        this.word(1);
        this.string(v);
        return;
      case "number":
        this.number(v);
        return;
      case "boolean":
        this.word(v ? 4 : 5);
        return;
      case "undefined":
        this.word(6);
        return;
      case "bigint":
        this.word(7);
        this.string(v.toString());
        return;
      case "object":
        if (v === null) this.word(8);
        else this.inside(v);
        return;
      default:
        throw new NotData(v);
    }
  }

  // truth takes only whether a value is there. What is not there — null,
  // undefined, false, 0, "" — is taken as it is: these differ from each
  // other for the one who reads them.
  truth(v) {
    if (v) this.word(16);
    else this.value(v);
  }

  // inside takes an object with everything in it.
  inside(v) {
    if (++this.depth > deepest) throw new NotData(v);
    if (Array.isArray(v)) {
      this.word(10);
      this.word(v.length);
      for (let i = 0; i < v.length; i++) this.value(v[i]);
    } else {
      const kind = Object.getPrototypeOf(v);
      if (kind === Object.prototype || kind === null) {
        // Every member starts with a word of its own, and another word
        // ends the object: where the members end is never in doubt.
        this.word(11);
        for (const key in v) {
          this.word(21);
          this.string(key);
          this.value(v[key]);
        }
        this.word(12);
      } else if (v instanceof Date) {
        this.word(13);
        this.number(v.getTime());
      } else if (v instanceof Map) {
        this.word(14);
        this.word(v.size);
        for (const [key, value] of v) {
          this.value(key);
          this.value(value);
        }
      } else if (v instanceof Set) {
        this.word(15);
        this.word(v.size);
        for (const value of v) this.value(value);
      } else if (v instanceof Loader) {
        // A component that a script has chosen for its template to show:
        // which file it is.
        this.word(20);
        this.word(v.file);
      } else {
        throw new NotData(v);
      }
    }
    this.depth--;
  }
}

// mix is the last step of MurmurHash3: it spreads every bit over the word.
function mix(h) {
  h ^= h >>> 16;
  h = Math.imul(h, 0x85ebca6b);
  h ^= h >>> 13;
  h = Math.imul(h, 0xc2b2ae35);
  return h ^ (h >>> 16);
}



// ---------------------------------------------------------------------------
// What a template reads

// copy copies data with everything in it. It takes what Hasher.value takes,
// and like it gives up on data that contains itself.
function copy(v, depth = 0) {
  if (v === null || typeof v !== "object") {
    if (typeof v === "function" || typeof v === "symbol") throw new NotData(v);
    return v;
  }
  if (depth++ > deepest) throw new NotData(v);
  if (Array.isArray(v)) {
    const out = new Array(v.length);
    for (let i = 0; i < v.length; i++) out[i] = copy(v[i], depth);
    return out;
  }
  const kind = Object.getPrototypeOf(v);
  if (kind === Object.prototype || kind === null) {
    const out = kind === null ? Object.create(null) : {};
    for (const key in v) out[key] = copy(v[key], depth);
    return out;
  }
  if (v instanceof Date) return new Date(v.getTime());
  if (v instanceof Map) return new Map(Array.from(v, ([key, value]) => [copy(key, depth), copy(value, depth)]));
  if (v instanceof Set) return new Set(Array.from(v, (value) => copy(value, depth)));
  // A component is code: the render side has its own. It is told which.
  if (v instanceof Loader) return { shugoComponent: v.file };
  throw new NotData(v);
}

// A reader is a shape made ready to work: print adds to a fingerprint what
// the shape says is read of a value, and part copies the same. The two go
// the same way through the value, which is what makes a template unable to
// depend on something its fingerprint does not cover.
//
// A shape, as the compiler writes it: 1 is the value whole, 0 is only
// whether it is there, an object names the members that are read, and a
// list of one shape says what is read of every item.

const whole = {
  print: (h, v) => h.value(v),
  part: (v) => copy(v),
};

// Of a value that is only asked whether it is there, the template gets
// true in its place. What is not there stays as it is.
const presence = {
  print: (h, v) => h.truth(v),
  part: (v) => (v ? true : v),
};

function members(names, readers) {
  return {
    print(h, v) {
      // A member of text or of a number — its length, say — comes with the value.
      if (v === null || typeof v !== "object") return h.value(v);
      h.word(17);
      for (let i = 0; i < names.length; i++) readers[i].print(h, v[names[i]]);
    },
    part(v) {
      if (v === null || typeof v !== "object") return copy(v);
      const out = {};
      for (let i = 0; i < names.length; i++) out[names[i]] = readers[i].part(v[names[i]]);
      return out;
    },
  };
}

function items(reader) {
  return {
    print(h, v) {
      // The template gets a list of the items whatever they were kept in.
      if (Array.isArray(v)) {
        h.word(18);
        h.word(v.length);
        for (let i = 0; i < v.length; i++) reader.print(h, v[i]);
      } else if (v instanceof Set || v instanceof Map) {
        h.word(18);
        h.word(v.size);
        for (const item of v) reader.print(h, item);
      } else {
        // Nothing, or something <g-for> itself will refuse.
        h.value(v);
      }
    },
    part(v) {
      if (Array.isArray(v)) {
        const out = new Array(v.length);
        for (let i = 0; i < v.length; i++) out[i] = reader.part(v[i]);
        return out;
      }
      if (v instanceof Set || v instanceof Map) return Array.from(v, (item) => reader.part(item));
      return copy(v);
    },
  };
}

export function reader(shape) {
  if (shape === 1) return whole;
  if (shape === 0) return presence;
  if (Array.isArray(shape)) return items(reader(shape[0]));
  const names = Object.keys(shape);
  return members(names, names.map((name) => reader(shape[name])));
}

// fault finds what makes a value not data, going the way of a shape, and
// says where it is. It is slow and runs only to explain a failure.
function fault(v, shape, path, seen = new Set()) {
  if (shape === 0 || v instanceof Loader) return null;
  if (v === null || typeof v !== "object") return typeof v === "function" || typeof v === "symbol" ? `${path}, which is a ${typeof v}` : null;
  if (Array.isArray(shape) && (Array.isArray(v) || v instanceof Set || v instanceof Map)) {
    let i = 0;
    for (const item of v) {
      const found = fault(item, shape[0], `${path}[${i++}]`, seen);
      if (found !== null) return found;
    }
    return null;
  }
  if (shape !== 1 && !Array.isArray(shape)) {
    for (const name in shape) {
      const found = fault(v[name], shape[name], `${path}.${name}`, seen);
      if (found !== null) return found;
    }
    return null;
  }
  // The value whole.
  if (seen.has(v)) return `${path}, which contains itself: data must end somewhere`;
  seen.add(v);
  const kind = Object.getPrototypeOf(v);
  let entries;
  if (Array.isArray(v)) entries = v.map((item, i) => [`${path}[${i}]`, item]);
  else if (kind === Object.prototype || kind === null) entries = Object.keys(v).map((key) => [`${path}.${key}`, v[key]]);
  else if (v instanceof Map) entries = Array.from(v, ([key, value], i) => [`${path} (entry ${i})`, [key, value]]);
  else if (v instanceof Set) entries = Array.from(v, (value, i) => [`${path} (item ${i})`, value]);
  else if (v instanceof Date) entries = [];
  else return `${path}, which is an instance of ${kind?.constructor?.name ?? "a class"}`;
  for (const [where, value] of entries) {
    const found = fault(value, 1, where, seen);
    if (found !== null) return found;
  }
  seen.delete(v);
  return null;
}

// Loader is what every file exports on this side: its script as a function
// that returns the variables its template reads, the names of those
// variables, and what the template reads of each.
class Loader {
  constructor(file, load, names, shapes) {
    this.file = file;
    this.load = load;
    this.names = names;
    this.shapes = shapes;
    this.readers = shapes.map(reader);
  }
}

export function loader(file, load, names, shapes) {
  return new Loader(file, load, names, shapes);
}

// ---------------------------------------------------------------------------
// Scripts

// Context is "Shugo" in a script.
class Context {
  #page;
  #slots;
  #reads;

  // page is the page being computed: { params, url }; null for a shared
  // block. slots are the names of the slots the caller filled, or null.
  // reads is where the queries of the script leave what they read.
  constructor(props, page, slots, reads) {
    this.props = props;
    this.#page = page;
    this.#slots = slots;
    this.#reads = reads;
  }

  get slots() {
    const slots = this.#slots;
    return { has: (name) => slots !== null && slots.has(name) };
  }

  get params() {
    return this.#of("params").params;
  }

  get url() {
    const page = this.#of("url");
    return (page.url ??= new URL(encodeURI(page.path), page.origin));
  }

  // query starts a question to a table of the data database. What it reads
  // is noted for the page or the shared block being computed.
  query(table) {
    return query(this.#reads, table);
  }

  // paginate cuts the rows of a query into pages and returns the page whose
  // number stands in the address, the way getStaticPaths({ paginate }) gives
  // it in Astro: { data, total, currentPage, lastPage, url: { prev, next, … } }.
  paginate(rows, { pageSize = 10 } = {}) {
    if (!(rows instanceof Query)) throw new TypeError(`Shugo.paginate() takes a query: Shugo.paginate(Shugo.query("posts").order("date", "desc"), { pageSize: 10 })`);
    const { route, params } = this.#of("paginate");
    const segment = pageSegment(route);
    const total = rows.count();
    const current = params.page === undefined ? 1 : Number(params.page);
    const start = (current - 1) * pageSize;
    const found = rows.limit(pageSize).offset(start).all();
    return paged(route, segment, params, found, start, pageSize, total, current);
  }

  #of(what) {
    if (this.#page === null) {
      throw new Error(`Shugo.${what} was read inside a block that is computed once for all pages. This should not happen: please report it`);
    }
    return this.#page;
  }
}

// query starts a question to the data database for one reader.
function query(reads, table) {
  if (data === null) {
    throw new Error('Shugo.query reads the data database of the project, and this project names none. Name it in shugo.json: { "data": "data/site.db" }');
  }
  return new Query(data, reads, table);
}

function describe(value) {
  if (value === null) return "null";
  if (Array.isArray(value)) return "a list";
  if (typeof value === "string") return `the string ${JSON.stringify(value.length > 40 ? value.slice(0, 40) + "…" : value)}`;
  if (typeof value === "object") return "an object";
  return typeof value === "undefined" ? "undefined" : `${typeof value} ${String(value)}`;
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
// and what getStaticPaths says for the others. reads is where its queries
// leave what they read.
async function entries(route, module, reads) {
  if (!route.segments.some((segment) => segment.text === undefined)) return [{ params: {} }];
  if (typeof module.getStaticPaths !== "function") throw new Error(`${route.pattern} has parameters, so its script must export getStaticPaths`);
  const list = await module.getStaticPaths({ paginate: paginator(route), query: (table) => query(reads, table) });
  if (!Array.isArray(list)) throw new Error(`getStaticPaths of ${route.pattern} must return a list of { params, props }, got ${describe(list)}`);
  for (const entry of list) {
    if (entry === null || typeof entry !== "object" || entry.params === null || typeof entry.params !== "object") {
      throw new Error(`getStaticPaths of ${route.pattern} must return a list of { params: { … }, props: { … } }, and one item is ${describe(entry)}`);
    }
  }
  return list;
}

// ---------------------------------------------------------------------------
// The run

// How many units are loaded at once. More does not use more processor,
// only lets them wait for their data together.
const concurrency = 32;

// after hands a value to next. The value may be a promise of it — the script
// of a page waits only if it has await in it — and only then is it waited for.
function after(value, next) {
  return value instanceof Promise ? value.then(next) : next(value);
}

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

// A unit is what is computed as one: a page or a shared block. On the wire
// it is "p", the address of a page and its route, or "s" and the name of a
// shared block, split by \x02.
function unit(text) {
  const [kind, name, route] = text.split("\x02");
  return { text, page: kind === "p", name, route: Number(route) };
}

class Run {
  constructor(site, modules, render, send) {
    this.site = site;
    this.modules = modules; // Map of file number → what the file exports on this side
    this.render = render; // the render side: it computes a page or a block from what it is given
    this.send = send;
    this.files = new Map(site.files.map((file, number) => [file.path, number]));
    this.pages = new Map(); // address → { index, route, entry }, of the routes that were listed
    // What the scripts of a unit gave, from when its fingerprint is asked
    // for until it is computed: by the unit as it is written on the wire.
    this.loaded = new Map();
    this.given = new Map(); // route → the slots every layout of it is given
    // The layouts that are computed with the pages of every route: those
    // that read the page. The others are shared blocks.
    this.own = site.routes.map((route) => route.layouts.filter((layout) => !site.files[layout].shared));
    this.hasher = new Hasher();
    this.batch = [];
    this.batchSize = 0;
  }

  place(unit) {
    return unit.page ? `the page ${unit.name} (${this.site.files[this.site.routes[unit.route]?.file]?.path})` : `the shared block ${unit.name}`;
  }

  // list finds the pages of routes: of the ones named, or of all. Every
  // route says which pages it has, and what it read from the data to know.
  // Among the routes listed together, a route that names more of an address
  // comes earlier and takes it.
  async list(wanted) {
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
        for (const entry of await entries(route, this.modules.get(route.file), reads)) {
          const path = address(route, entry.params);
          const taken = this.pages.get(path);
          if (taken !== undefined && taken.index === index) throw new Error(`getStaticPaths of ${route.pattern} returned the address ${path} twice`);
          if (taken !== undefined && taken.index < index) {
            shadowed++;
            continue;
          }
          props ||= entry.props != null && Object.keys(entry.props).length > 0;
          // A page is given the parameters its address holds, and no more:
          // so it is, too, when it is computed from its address alone.
          this.pages.set(path, { index, route, entry: { params: parameters(route, path), props: entry.props } });
          if (lines.push(index + "\x02" + path) === 4096) {
            this.send("P", lines.join("\x03"));
            lines = [];
          }
        }
      } catch (error) {
        throw placed(error, `getStaticPaths of ${this.site.files[route.file].path}`);
      }
      // Rows that the list hands to its pages are read there, after the list
      // has told what it read: so all of them count as read whole.
      this.send("G", index + "\x02" + (props ? 1 : 0) + "\x02" + reads.text(props));
    }
    if (lines.length > 0) this.send("P", lines.join("\x03"));
    return { shadowed };
  }

  // each runs work for every unit, several at a time where work waits. work
  // is given the unit and its place among the units.
  //
  // A page that fails ends the run. A shared block that fails is told to
  // the Go side, and the run goes on: the store may know a block that
  // nothing shows any more and whose data is gone, and only the Go side
  // knows whether that is so.
  async each(units, work) {
    let next = 0;
    const failed = (unit, error) => {
      if (unit.page) {
        next = units.length; // the others stop after the unit they are at
        throw placed(error, this.place(unit));
      }
      this.send("X", unit.name + "\x02" + failure(placed(error, this.place(unit))));
    };
    const worker = async () => {
      while (next < units.length) {
        const unit = units[next++];
        try {
          const done = work(unit, next - 1);
          if (done instanceof Promise) await done;
        } catch (error) {
          failed(unit, error);
        }
      }
    };
    await Promise.all(Array.from({ length: concurrency }, worker));
  }

  // prints runs the scripts of the units and sends their fingerprints: eight
  // bytes for every unit, in the order of the units. What the scripts gave
  // is kept until the unit is computed.
  async prints(units) {
    const out = Buffer.alloc(units.length * 8);
    await this.each(units, (unit, place) =>
      after(this.load(unit), (loaded) => {
        this.loaded.set(unit.text, loaded);
        out.writeInt32LE(loaded.a, place * 8 + 4);
        out.writeInt32LE(loaded.b, place * 8);
      }),
    );
    this.send("F", out);
  }

  // compute sends the records of the units. A unit whose fingerprint was not
  // asked for is loaded here: every unit of a project with a data database,
  // and elsewhere a shared block met for the first time.
  async compute(units) {
    await this.each(units, (unit) => {
      const kept = this.loaded.get(unit.text);
      return after(kept ?? this.load(unit), (loaded) => {
        // What the scripts gave is not needed once the unit is computed.
        this.loaded.delete(unit.text);
        // The template is given what its script left, and that must be
        // what the fingerprint was made of: other scripts have run since.
        if (kept !== undefined && loaded.values !== null) {
          const { a, b } = loaded;
          this.printed(loaded);
          if (loaded.a !== a || loaded.b !== b) {
            throw new Error("what the template reads has changed since the script ran: another script changes data that this one has read. A script must not change data that it shares with others: sort or fill a copy, as in [...list].sort()");
          }
        }
        const parts = this.parts(loaded.values);
        const print = hex(loaded.a, loaded.b);
        // Nothing reads the rows of the unit any more: its templates have
        // their copies. What was read of them is what the unit depends on.
        const reads = loaded.reads === null ? "" : loaded.reads.text();
        this.emit(unit.page ? this.render.page(unit.name, loaded.index, loaded.params, parts, print, reads) : this.render.block(unit.name, parts?.get(loaded.values[0][0]), print, reads));
      });
    });
    this.flush();
  }

  // load runs the scripts of a unit, and makes the fingerprint of what its
  // templates read of them where fingerprints tell what to compute.
  load(unit) {
    return unit.page ? this.loadPage(unit.name, unit.route) : this.loadBlock(unit.name);
  }

  // A page of a route that was listed in this run is loaded with what the
  // list gave it; of another page only the address is known, and its
  // parameters are read from it.
  loadPage(path, index) {
    const route = this.site.routes[index];
    if (route === undefined) throw new Error(`the page ${path} was asked for under route ${index}, which this project's compiled code does not have: build the project again`);
    const listed = this.pages.get(path);
    const entry = listed !== undefined && listed.index === index ? listed.entry : { params: parameters(route, path) };
    const params = Object.freeze(entry.params);
    const page = { params, path, route, origin: this.site.origin, url: undefined };
    const reads = new Reads();
    const own = this.script(route.file, new Context(entry.props ?? {}, page, null, reads));
    // Most pages are computed alone. Some have layouts that are computed
    // with them: those that read the page. The other layouts are shared blocks.
    if (this.own[index].length === 0) return after(own, (got) => this.printed({ index, params, reads, values: [[route.file, got]] }));
    return this.withLayouts(index, route, params, page, reads, own);
  }

  async withLayouts(index, route, params, page, reads, own) {
    const values = [[route.file, await own]];
    for (const layout of this.own[index]) values.push([layout, await this.script(layout, new Context({}, page, this.slots(index, layout), reads))]);
    return this.printed({ index, params, reads, values });
  }

  loadBlock(name) {
    const { path, props, slots } = this.render.unnamed(name);
    const file = this.files.get(path);
    if (file === undefined) throw new Error(`there is no file ${path} to compute the shared block from`);
    // A component that only shows what it is given has nothing to load: its
    // name says all that it shows, and its fingerprint is that of nothing.
    if (!this.site.files[file].loads) return { values: null, reads: null, a: 0, b: 0 };
    const reads = new Reads();
    const own = this.script(file, new Context(props, null, slots === null ? null : new Set(slots.keys()), reads));
    return after(own, (got) => this.printed({ reads, values: [[file, got]] }));
  }

  // script runs the script of a file. It returns the variables the template
  // reads, or a promise of them if the script waits.
  script(file, context) {
    return this.modules.get(file).default.load(context);
  }

  // slots returns the names of the slots that a layout of a route is given:
  // the page in the default one, and what the page or a layout inside gave
  // for the slots this layout has. The render side hands them out the same way.
  slots(index, layout) {
    let given = this.given.get(index);
    if (given === undefined) {
      given = new Map();
      const route = this.site.routes[index];
      const provided = new Set(this.site.files[route.file].gives ?? []);
      for (let i = route.layouts.length - 1; i >= 0; i--) {
        const file = this.site.files[route.layouts[i]];
        const slots = new Set(["default"]);
        for (const name of file.slots ?? []) {
          if (name !== "default" && provided.delete(name)) slots.add(name);
        }
        given.set(route.layouts[i], slots);
        for (const name of file.gives ?? []) provided.add(name);
      }
      this.given.set(index, given);
    }
    return given.get(layout);
  }

  // printed makes the fingerprint of what the templates of a unit read of
  // what its scripts gave, and leaves its two halves in a and b. A project
  // with a data database is told by the database what to compute, and makes
  // no fingerprints: reading the variables here would count as the template
  // reading them, and the copy for the template reads them anyway.
  printed(loaded) {
    if (data !== null) {
      loaded.a = loaded.b = 0;
      return loaded;
    }
    const h = this.hasher;
    h.reset();
    for (const [file, got] of loaded.values) {
      const { readers } = this.modules.get(file).default;
      h.word(file);
      try {
        for (let i = 0; i < readers.length; i++) readers[i].print(h, got[i]);
      } catch (error) {
        throw this.explained(error, file, got);
      }
    }
    h.finish();
    loaded.a = h.a;
    loaded.b = h.b;
    return loaded;
  }

  // parts copies for every template of a unit what it reads.
  parts(values) {
    if (values === null) return undefined;
    const out = new Map();
    for (const [file, got] of values) {
      const { readers } = this.modules.get(file).default;
      try {
        out.set(file, readers.map((reader, i) => reader.part(got[i])));
      } catch (error) {
        throw this.explained(error, file, got);
      }
    }
    return out;
  }

  // explained says what a template was found to read that is not data.
  explained(error, file, got) {
    if (!(error instanceof NotData)) return error;
    const { names, shapes } = this.modules.get(file).default;
    let found = null;
    for (let i = 0; i < names.length && found === null; i++) found = fault(got[i], shapes[i], names[i]);
    return new Error(
      `${this.site.files[file].path}: the template reads ${found ?? "something that is not data"}. ` +
        "A template is given the data of its script — text, numbers, true and false, lists, plain objects, dates, Map and Set — and none of its code. " +
        "Show the result of a function instead of the function, turn an instance of a class into a plain object, and import in the script the functions that the template calls",
    );
  }

  emit(text) {
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
  // The body of a message is text, or bytes as they are.
  const send = (kind, body) => {
    const head = Buffer.allocUnsafe(5);
    if (typeof body === "string") body = Buffer.from(body);
    head.writeUInt32LE(body.length + 1, 0);
    head.write(kind, 4, "latin1");
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
    // A shape of readings is told to the Go side when it is first met.
    if (process.env.SHUGO_DATA) data = await open(process.env.SHUGO_DATA, (number, spec) => send("S", number + "\x02" + spec));
    // The render side is a bundle of its own, with its own copy of every
    // module: nothing that a script has loaded can be reached from it.
    const render = createRequire(import.meta.url)("./render.cjs").default(site);
    const run = new Run(site, new Map(modules), render, send);
    for (let request = receive(); request !== null && request.kind !== "Q"; request = receive()) {
      const named = request.text === "" ? [] : request.text.split(request.kind === "L" ? "," : "\x03");
      switch (request.kind) {
        case "L":
          send("D", JSON.stringify(await run.list(named.map(Number))));
          break;
        case "F":
          await run.prints(named.map(unit));
          send("D", "{}");
          break;
        case "C":
          await run.compute(named.map(unit));
          send("D", "{}");
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
