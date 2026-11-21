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
import { catalog } from "shugo:catalog";

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
  // a snippet (a part of the template it is written in).
  block(file, part) {
    const block = { index: this.blocks.length, file, part, values: [], children: [], snippets: null, shared: null };
    this.blocks.push(block);
    return block;
  }

  // A use stands where a shared block is shown, and names it. The shared
  // block is kept once; the snippets this place gave it are kept here.
  use(shared, snippets) {
    const block = this.block(-1, 0);
    block.shared = shared;
    block.snippets = snippets;
    return block;
  }
}

// A scope is what a template can see around itself.
//   page      the page being computed: { params, url }; null inside a shared block
//   snippets  the snippets without parameters the caller gave, computed:
//             Map of name → blocks; null for none
//   provided  what the page gives to its layouts: Map of name → blocks; null inside a shared block
//   reads     where the queries to the data database leave their readings
//   frame     the component being shown: { outer } — the frame of its caller
//   props     what $props() gives: the parameters and the snippets

// Writer is "$$" in generated code: it writes the values of one block.
class Writer {
  constructor(run, record, block, scope) {
    this.run = run;
    this.record = record;
    this.block = block;
    this.scope = scope;
  }

  // $props()
  props() {
    return this.scope.props;
  }

  // {expression}
  text(value) {
    this.block.values.push(escape(stringify(value)));
  }

  // {@html expression}
  html(value) {
    this.block.values.push(escape(String(value ?? "")));
  }

  island(target, { shugoSearch = false, ...props }) {
    if (shugoSearch) props.initial = catalog();
    // Normalize before SSR so the browser hydrates with the same values.
    const serial = JSON.stringify(props, (_, value) => {
      if (typeof value === "function" || typeof value === "symbol") {
        throw new TypeError("Svelte island props must be JSON data");
      }
      return value;
    });
    const normalized = JSON.parse(serial);
    const { body, head } = target.render(normalized);
    if (head) throw new Error("Put the island’s head elements in its Shugo page");
    this.block.values.push(escape(JSON.stringify({
      id: target.id, url: target.url, css: target.css, search: Boolean(shugoSearch),
      props: JSON.parse(serial), body,
    })));
  }

  // "text {expression}": an expression inside the value of an attribute.
  str(value) {
    return stringify(value);
  }

  // name={expression}: "-" for no attribute, "+" for one written without a
  // value, "=" and the value.
  attr(name, value, boolean) {
    this.block.values.push(attributeOf(name, value, boolean));
  }

  // class={…} with class: directives and the class that scopes the styles
  // of the component.
  cls(value, hash, directives) {
    return to_class(clsx(value), hash || null, directives);
  }

  // style={…} with style: directives, [normal, !important].
  sty(value, directives) {
    return to_style(value, directives);
  }

  // Every attribute of an element with {...attributes}: their number, then
  // the name and the value of each. For a <select>, value is not written: it
  // is returned, to choose an option.
  spread(attributes, hash, classes, styles, preserve, select) {
    let chosen;
    if (select) {
      chosen = attributes.value;
      delete attributes.value;
    }
    const pairs = spreadOf(attributes, hash || null, classes, styles, preserve);
    this.block.values.push(pairs.length / 2, ...pairs);
    return chosen;
  }

  // Which branch of {#if} … {:else} was taken, from 1; 0 for none.
  branch(number) {
    this.block.values.push(number);
  }

  // {#each}: the number of items goes before the items, and is known after them.
  loop() {
    return this.block.values.push(0) - 1;
  }

  count(at, items) {
    this.block.values[at] = items;
  }

  items(list) {
    return ensure_array_like(list);
  }

  // {#await value}: a promise is not waited for on a server.
  pending(value) {
    return typeof value?.then === "function";
  }

  // <svelte:element this={tag}>: "" shows nothing.
  tag(name) {
    if (!name) {
      this.block.values.push("");
      return "";
    }
    const tag = String(name);
    if (!/^[a-zA-Z][a-zA-Z0-9:._-]*$/.test(tag)) throw new TypeError(`<svelte:element this={…}> needs the name of a tag, got ${describe(name)}`);
    this.block.values.push(tag);
    return tag;
  }

  void(tag) {
    return VOID.has(tag.toLowerCase());
  }

  // Whether an <option> is the one its <select> has as its value.
  selected(value, chosen) {
    return Array.isArray(chosen) ? chosen.includes(value) : value === chosen;
  }

  // <Component prop={…}>children{#snippet name()}…{/snippet}</Component>.
  // The snippets without parameters written inside the tag come as [name,
  // part, body].
  call(target, props, given) {
    if (typeof target?.id === "string" && typeof target.url === "string" && typeof target.render === "function") {
      throw new TypeError("a Svelte island must be rendered by its imported tag");
    }
    if (target === null || typeof target !== "object" || typeof target.render !== "function") {
      throw new TypeError(`this tag is not a component: it is ${describe(target)}. A component is what a .shugo file exports: import Card from "./Card.shugo"`);
    }
    // What stands inside the tag is computed here, in the template that
    // wrote it, whether or not the component shows it.
    let snippets = null;
    if (given.length > 0) {
      snippets = new Map();
      for (const [name, part, body] of given) {
        const block = this.record.block(this.block.file, part);
        body(new Writer(this.run, this.record, block, this.scope));
        snippets.set(name, [block.index]);
      }
    }
    this.block.children.push(this.run.show(this.record, target, props, snippets, this.scope));
  }

  // {#snippet name(parameters)}: a snippet that is computed where it is rendered.
  snippet(part, body) {
    return snippetOf({ kind: "live", file: this.block.file, part, body, scope: this.scope });
  }

  // {@render snippet(…)}
  render(snippet, args, optional) {
    const values = this.block.values;
    if (snippet == null && optional) {
      values.push("-");
      return;
    }
    const info = snippet?.[SNIPPET];
    if (info === undefined) throw new TypeError(`{@render …} shows a snippet, and this is ${describe(snippet)}`);
    if (info.kind === "given") {
      // What a caller gave stays with the caller: the Go side shows it
      // from there.
      if (args.length > 0) throw new TypeError(`the snippet ${info.name} takes no parameters`);
      values.push(`f${depth(this.scope.frame, info.frame)}:${info.name}`);
      return;
    }
    const block = this.record.block(info.file, info.part);
    info.body(new Writer(this.run, this.record, block, info.scope), ...args);
    this.block.children.push(block.index);
    values.push(`c${depth(this.scope.frame, info.scope.frame)}`);
  }

  // <svelte:head> of a page or a layout: markup for the <head> of the page,
  // under "head", and its <title> under "title".
  provide(name, part, body) {
    const block = this.record.block(this.block.file, part);
    body(new Writer(this.run, this.record, block, this.scope));
    const provided = this.scope.provided;
    if (provided === null) throw new Error("<svelte:head> was computed inside a block that is shared by all pages. This should not happen: please report it");
    if (provided.has(name)) provided.get(name).push(block.index);
    else provided.set(name, [block.index]);
  }

  // The end of the <head> a page or a layout writes, where what was given
  // the head is shown in the order of the document: what this template gave
  // itself, then what the templates inside it gave. They come to the Go side
  // as snippets of the template's own block.
  outlet(name) {
    const own = this.scope.provided?.get(name) ?? [];
    const given = this.scope.snippets?.get(name);
    if (own.length > 0) {
      this.scope.provided.delete(name);
      const block = this.scope.frame.block;
      (block.snippets ??= new Map()).set(name, [...own, ...(given ?? [])]);
    }
    this.block.values.push(own.length > 0 || given !== undefined ? `f0:${name}` : "-");
  }
}

// depth counts the components between two frames: how many levels out from
// the first the second is.
function depth(from, to) {
  let levels = 0;
  for (let frame = from; frame !== to; frame = frame.outer) {
    if (frame === null) throw new Error("a snippet is shown outside of the component it belongs to. This should not happen: please report it");
    levels++;
  }
  return levels;
}

// A snippet is a function, as in Svelte, that can only be shown with
// {@render}. What it is lies under SNIPPET: { kind: "given", name, frame }
// for a snippet without parameters that a caller gave a component, computed
// already and kept in the caller's record; { kind: "live", file, part, body,
// scope } for one that is computed where it is rendered.
const SNIPPET = Symbol("snippet");

function snippetOf(info) {
  const snippet = () => {
    throw new TypeError("a snippet is shown with {@render snippet()}, not called");
  };
  snippet[SNIPPET] = info;
  return snippet;
}

// propsOf gives a component its parameters with the snippets its caller
// wrote inside its tag.
function propsOf(props, snippets, frame) {
  if (snippets === null) return props;
  const out = { ...props };
  for (const name of snippets.keys()) out[name] = snippetOf({ kind: "given", name, frame });
  return out;
}

// Context is "Shugo" in the script and in the expressions of a template.
class Context {
  #scope;

  constructor(scope) {
    this.#scope = scope;
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

  get props() {
    throw new Error("Shugo.props is gone: write let { title } = $props(), as in Svelte");
  }

  get slots() {
    throw new Error("Shugo.slots is gone: a snippet the caller gave is a prop; check it with {#if children}");
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

// The runes of Svelte as a server sees them: state is the value it starts
// with, derived values are computed once, effects never run.
export function $state(value) {
  return value;
}
$state.raw = (value) => value;
$state.snapshot = (value) => value;
export function $derived(value) {
  return value;
}
$derived.by = (compute) => compute();
export function $effect() {}
$effect.pre = () => {};
$effect.root = () => () => {};
$effect.tracking = () => false;
$effect.pending = () => 0;
export function $bindable(fallback) {
  return fallback;
}
export function $inspect() {
  return { with() {} };
}
$inspect.trace = () => {};
export function $host() {
  throw new Error("$host() works in custom elements, which a static site does not have");
}

// ---------------------------------------------------------------------------
// Values
//
// What a value gives in the page is decided by the rules of the server
// renderer of Svelte 5.59, taken as they are: stringify, attr, clsx, to_class,
// to_style, attributes and ensure_array_like come from Svelte (MIT license,
// © Svelte contributors), clsx from Luke Edwards' clsx (MIT license). The Go
// side escapes what these give.

// The Go side splits what it receives by the characters 0 to 3. Inside a
// value they come escaped, as 4 and the character plus 0x40, and so does 4.
const special = /[\x00-\x04]/;
const everySpecial = /[\x00-\x04]/g;

function escape(text) {
  return special.test(text) ? text.replace(everySpecial, (c) => "\x04" + String.fromCharCode(0x40 + c.charCodeAt(0))) : text;
}

function stringify(value) {
  return typeof value === "string" ? value : value == null ? "" : value + "";
}

const VOID = new Set(["area", "base", "br", "col", "command", "embed", "hr", "img", "input", "keygen", "link", "meta", "param", "source", "track", "wbr"]);

// The attributes of HTML that are there or not.
const BOOLEAN = new Set([
  "allowfullscreen", "async", "autofocus", "autoplay", "checked", "controls", "default", "disabled",
  "formnovalidate", "indeterminate", "inert", "ismap", "loop", "multiple", "muted", "nomodule",
  "novalidate", "open", "playsinline", "readonly", "required", "reversed", "seamless", "selected",
  "webkitdirectory", "defer", "disablepictureinpicture", "disableremoteplayback",
]);

// translate={false} is translate="no", as in Svelte.
const replacements = { translate: new Map([[true, "yes"], [false, "no"]]) };

// attributeOf is attr() of Svelte, giving the sign and the value instead of
// the escaped text.
function attributeOf(name, value, boolean = false) {
  if (name === "hidden" && value !== "until-found") boolean = true;
  if (value == null || (boolean && !value && value !== "")) return "-";
  if (boolean) return "+";
  const normalized = (Object.hasOwn(replacements, name) && replacements[name].get(value)) || value;
  return "=" + escape(String(normalized));
}

function clsxValue(value) {
  let out = "";
  if (typeof value === "string" || typeof value === "number") out += value;
  else if (typeof value === "object") {
    if (Array.isArray(value)) {
      for (let i = 0; i < value.length; i++) {
        let part;
        if (value[i] && (part = clsxValue(value[i]))) {
          if (out) out += " ";
          out += part;
        }
      }
    } else {
      for (const key in value) {
        if (value[key]) {
          if (out) out += " ";
          out += key;
        }
      }
    }
  }
  return out;
}

function clsx(value) {
  if (typeof value === "object") return value ? clsxValue(value) : "";
  return value ?? "";
}

const whitespace = [..." \t\n\r\f \u000b﻿"];

function to_class(value, hash, directives) {
  let classname = value == null ? "" : "" + value;
  if (hash) classname = classname ? classname + " " + hash : hash;
  if (directives) {
    for (const key of Object.keys(directives)) {
      if (directives[key]) {
        classname = classname ? classname + " " + key : key;
      } else if (classname.length) {
        const len = key.length;
        let a = 0;
        while ((a = classname.indexOf(key, a)) >= 0) {
          const b = a + len;
          if ((a === 0 || whitespace.includes(classname[a - 1])) && (b === classname.length || whitespace.includes(classname[b]))) {
            classname = (a === 0 ? "" : classname.substring(0, a)) + classname.substring(b + 1);
          } else {
            a = b;
          }
        }
      }
    }
  }
  return classname === "" ? null : classname;
}

function append_styles(styles, important = false) {
  const separator = important ? " !important;" : ";";
  let css = "";
  for (const key of Object.keys(styles)) {
    const value = styles[key];
    if (value != null && value !== "") css += " " + key + ": " + value + separator;
  }
  return css;
}

function to_css_name(name) {
  if (name[0] !== "-" || name[1] !== "-") return name.toLowerCase();
  return name;
}

function to_style(value, styles) {
  if (styles) {
    let new_style = "";
    let normal_styles;
    let important_styles;
    if (Array.isArray(styles)) {
      normal_styles = styles[0];
      important_styles = styles[1];
    } else {
      normal_styles = styles;
    }
    if (value) {
      value = String(value).replaceAll(/\/\*.*?\*\//g, "").trim();
      let in_str = false;
      let in_apo = 0;
      let in_comment = false;
      const reserved_names = [];
      if (normal_styles) reserved_names.push(...Object.keys(normal_styles).map(to_css_name));
      if (important_styles) reserved_names.push(...Object.keys(important_styles).map(to_css_name));
      let start_index = 0;
      let name_index = -1;
      const len = value.length;
      for (let i = 0; i < len; i++) {
        const c = value[i];
        if (in_comment) {
          if (c === "/" && value[i - 1] === "*") in_comment = false;
        } else if (in_str) {
          if (in_str === c) in_str = false;
        } else if (c === "/" && value[i + 1] === "*") {
          in_comment = true;
        } else if (c === '"' || c === "'") {
          in_str = c;
        } else if (c === "(") {
          in_apo++;
        } else if (c === ")") {
          in_apo--;
        }
        if (!in_comment && in_str === false && in_apo === 0) {
          if (c === ":" && name_index === -1) {
            name_index = i;
          } else if (c === ";" || i === len - 1) {
            if (name_index !== -1) {
              const name = to_css_name(value.substring(start_index, name_index).trim());
              if (!reserved_names.includes(name)) {
                if (c !== ";") i++;
                const property = value.substring(start_index, i).trim();
                new_style += " " + property + ";";
              }
            }
            start_index = i + 1;
            name_index = -1;
          }
        }
      }
    }
    if (normal_styles) new_style += append_styles(normal_styles);
    if (important_styles) new_style += append_styles(important_styles, true);
    new_style = new_style.trim();
    return new_style === "" ? null : new_style;
  }
  return value == null ? null : String(value);
}

// Names that cannot be attributes: Svelte leaves them out of a spread.
const INVALID_ATTR_NAME_CHAR_REGEX = /[\s'">/=\u{FDD0}-\u{FDEF}\u{FFFE}\u{FFFF}\u{1FFFE}\u{1FFFF}\u{2FFFE}\u{2FFFF}\u{3FFFE}\u{3FFFF}\u{4FFFE}\u{4FFFF}\u{5FFFE}\u{5FFFF}\u{6FFFE}\u{6FFFF}\u{7FFFE}\u{7FFFF}\u{8FFFE}\u{8FFFF}\u{9FFFE}\u{9FFFF}\u{AFFFE}\u{AFFFF}\u{BFFFE}\u{BFFFF}\u{CFFFE}\u{CFFFF}\u{DFFFE}\u{DFFFF}\u{EFFFE}\u{EFFFF}\u{FFFFE}\u{FFFFF}\u{10FFFE}\u{10FFFF}]/u;

// spreadOf is attributes() of Svelte: every attribute of an element with a
// spread, in order, giving the names and the signed values.
function spreadOf(attrs, css_hash, classes, styles, preserve) {
  const hasStyles = styles && (Object.keys(styles[0]).length > 0 || Object.keys(styles[1]).length > 0);
  if (hasStyles) attrs.style = to_style(attrs.style, styles);
  if (attrs.class) attrs.class = clsx(attrs.class);
  const hasClasses = classes && Object.keys(classes).length > 0;
  if (css_hash || hasClasses) attrs.class = to_class(attrs.class, css_hash, hasClasses ? classes : undefined);
  const out = [];
  const names = Object.keys(attrs);
  outer: for (let i = 0; i < names.length; i++) {
    let name = names[i];
    if (typeof attrs[name] === "function") continue;
    if (name[0] === "$" && name[1] === "$") continue;
    if (name === "" || INVALID_ATTR_NAME_CHAR_REGEX.test(name)) continue;
    const value = attrs[name];
    const lower = name.toLowerCase();
    if (!preserve) name = lower;
    if (lower.length > 2 && lower.startsWith("on")) continue;
    if (name === "defaultvalue" || name === "defaultchecked") {
      name = name === "defaultvalue" ? "value" : "checked";
      if (name in attrs) continue;
      for (let j = 0; j < names.length; j++) {
        if (names[j].toLowerCase() === name) continue outer;
      }
    }
    const sign = attributeOf(name, value, !preserve && BOOLEAN.has(name));
    if (sign !== "-") out.push(escape(name), sign);
  }
  return out;
}

function ensure_array_like(array_like_or_iterator) {
  if (array_like_or_iterator) {
    return array_like_or_iterator.length !== undefined ? array_like_or_iterator : Array.from(array_like_or_iterator);
  }
  return [];
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
  show(record, target, props, snippets, scope) {
    const name = this.site.files[target.file].shared ? sharedName(this.site.files[target.file].path, props, snippets) : null;
    if (name !== null) return record.use(name, snippets).index;
    const block = record.block(target.file, 0);
    block.snippets = snippets;
    const frame = { outer: scope.frame, block };
    const inner = { page: scope.page, snippets, provided: scope.provided, reads: scope.reads, frame, props: propsOf(props, snippets, frame) };
    target.render(new Writer(this, record, block, inner), new Context(inner));
    return block.index;
  }

  // block computes a shared block from its name alone: the name holds the
  // file, the parameters and the names of the snippets the caller gave.
  block(name) {
    const { path, props, snippets } = unnamed(name);
    const target = this.modules.get(this.files.get(path))?.default;
    if (target === undefined) throw new Error(`there is no file ${path} to compute the shared block from`);
    const record = new Record("s", name, 0);
    const block = record.block(target.file, 0);
    // Inside a shared block only the names of the snippets the caller gave
    // matter: what they show belongs to the place the block is used at.
    const frame = { outer: null, block };
    const inner = { page: null, snippets, provided: null, reads: record.reads, frame, props: propsOf(props, snippets, frame) };
    during(record.reads, () => target.render(new Writer(this, record, block, inner), new Context(inner)));
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
    const scope = { page: { params, path, route, origin: this.site.origin, url: undefined }, snippets: null, provided: new Map(), reads: record.reads, frame: null, props: {} };
    during(record.reads, () => this.around(record, route, props, scope));
    this.emit(record);
  }

  // around computes a page with its layouts into the record.
  around(record, route, props, scope) {
    let top = this.show(record, this.modules.get(route.file).default, props, null, scope);
    for (let i = route.layouts.length - 1; i >= 0; i--) {
      // A layout shows the page as its children. What the page, or a
      // layout inside this one, gave the head goes to the nearest layout
      // that writes the <head>.
      const layout = this.modules.get(route.layouts[i]).default;
      const inner = scope.provided;
      const snippets = new Map([["children", [top]]]);
      for (const name of this.site.files[layout.file].outlets ?? []) {
        if (inner.has(name)) {
          snippets.set(name, inner.get(name));
          inner.delete(name);
        }
      }
      scope.provided = new Map();
      top = this.show(record, layout, {}, snippets, scope);
      // What goes on outwards keeps the order of the document: what this
      // layout gave comes before what the templates inside it gave.
      for (const [name, blocks] of inner) scope.provided.set(name, [...(scope.provided.get(name) ?? []), ...blocks]);
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
function sharedName(path, props, snippets) {
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
  if (snippets !== null) key += "|" + [...snippets.keys()].sort().join(",");
  return path + "{" + key + "}";
}

// unnamed reads a name that sharedName wrote. The names of the snippets
// follow the last "|" and are made of letters, digits, "-" and "_"; a "|"
// inside a parameter is followed by the quote that closes its text at the
// least.
function unnamed(name) {
  const start = name.indexOf("{");
  let inside = name.slice(start + 1, -1);
  let snippets = null;
  const bar = inside.lastIndexOf("|");
  if (bar >= 0 && /^[\w,-]*$/.test(inside.slice(bar + 1))) {
    snippets = new Map(inside.slice(bar + 1).split(",").filter((one) => one !== "").map((one) => [one, []]));
    inside = inside.slice(0, bar);
  }
  return { path: name.slice(0, start), props: JSON.parse("{" + inside + "}"), snippets };
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
        ? "u\x02" + block.shared + "\x02" + snippetsOf(block.snippets)
        : "b\x02" + block.file + "\x02" + block.part + "\x02" + block.children.join(",") + "\x02" + snippetsOf(block.snippets) + "\x02" + valuesOf(block.values),
    );
  }
  return lines.join("\x01");
}

// Every value is written after \x00, so that no values and one empty value differ.
function valuesOf(values) {
  return values.length === 0 ? "" : "\x00" + values.join("\x00");
}

function snippetsOf(snippets) {
  if (snippets === null) return "";
  let out = "";
  for (const [name, blocks] of snippets) out += (out ? ";" : "") + name + "=" + blocks.join(",");
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
          send("D", "");
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
    const instead = { props: "let { … } = $props()", params: "Shugo.params", url: "Shugo.url", slots: "a snippet taken from $props(): {#if children}" }[String(name)];
    throw new Error(`There is no Astro.${String(name)} in Shugo${instead ? `: write ${instead}` : ""}`);
  },
});
