// The JavaScript side of Shugo. Svelte renders the pages: the code its
// compiler made of the templates runs here, on the server runtime of Svelte
// that Shugo ships. This side cuts what Svelte renders into blocks, so that
// what many pages show is computed and kept once, and hands the blocks to the
// Go side, which keeps them and puts the pages together.
//
// The Go side leads. It asks which pages the project has, then names the
// pages and the shared blocks to compute; this side answers each request and
// waits for the next.
//
// Runs on Bun and on Node.

import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { dirname } from "node:path";
import { fileURLToPath } from "node:url";

import { getAllContexts } from "svelte";
import { render } from "svelte/server";

import { during, open, Query, Reads } from "shugo:query";
import { buildChannel } from "shugo:wire";

// The packages of npm are bundled with the code of the site, and a package
// written for CommonJS reaches for require, __filename and __dirname, which
// an ES module of Node does not have. It gets those of the bundle. This
// module is imported first, so they are there before any package runs.
globalThis.require ??= createRequire(import.meta.url);
globalThis.__filename ??= fileURLToPath(import.meta.url);
globalThis.__dirname ??= dirname(globalThis.__filename);

// A failure while the code of the site loads, before run is called: a
// module or a package that throws at its top. The Go side hears of it as of
// any failure, with its place in the files of the project.
process.on("uncaughtException", (error) => {
  const wire = buildChannel();
  if (wire === null) {
    console.error(error);
  } else {
    wire.send("E", failure(error));
  }
  process.exit(1);
});

// The data database of the project, once it is open; null if it has none.
let data = null;

// The skeleton of the site: its files and its routes.
let site = null;

// Search results are ordinary view data. Keep their source out of the JSON
// so the same result can be used in a static template or a request binding.
const requestSources = new WeakMap();

export function requestSource(value, source) {
  requestSources.set(value, source);
  return value;
}

// ---------------------------------------------------------------------------
// Marks
//
// Svelte renders a page as one text. Where a shared block stands in it is
// marked in the text, and so is the raw HTML of {@html}: with marks that no
// page can write, a byte 0, a word chosen at random for the run, what the
// mark says and a byte 0 again.
//
//   c3      use 3 of the record stands here: a shared block, see use
//   gname   what the place that shows this shared block gave it as name
//   i2      island 2 of the record
//   s<…>    the styles of a component compiled with css="injected", which go
//           into the <head> once however many times it is shown
//   [ ]     the HTML of {@html} starts, and ends
//   h       what goes into the <head> ends; the <title> follows
//
// The word comes from the crypto of the web, which Bun and Node both have at
// hand; node:crypto takes Bun milliseconds to load.
const word = Buffer.from(crypto.getRandomValues(new Uint8Array(12))).toString("base64url");
const mark = (what) => "\0" + word + what + "\0";
const marks = new RegExp("\0" + word + "([^\0]*)\0");
const headEnd = mark("h");

// The comments Svelte writes for the browser, to hydrate the page with: a
// static page has no use for them. Svelte 5.59 writes <!----> between texts
// and around {@html}; <!--[-->, <!--[!-->, <!--[1-->, <!--[-1--> and
// <!--]--> around blocks; <!--$s1--> for $props.id(); the hash of a
// component around what it gives the head; and <!> at the end of a <select>
// or an <option> whose content is more than text. A comment the page writes
// itself, with {@html}, stays.
const comments = /<!--(?:\[(?:!|-?\d+)?|\]|\$[\w-]*|[a-z0-9]*)-->|<!>/g;

// rendered renders with Svelte what show writes, and returns its HTML, what
// it gives the head and the <title> among that, each as Svelte wrote it,
// marks and all.
function rendered(show, options) {
  const { head, body } = render(Rendered, { props: { show }, ...options });
  const end = head.lastIndexOf(headEnd);
  // After the head Svelte writes the <title>, then the styles of the
  // components compiled with css="injected".
  const after = head.slice(end + headEnd.length);
  const close = after.startsWith("<title") ? after.indexOf("</title>") + "</title>".length : 0;
  const styles = after.slice(close).replace(injected, (style) => mark("s" + style));
  return { body, head: head.slice(0, end) + styles, title: after.slice(0, close) };
}

// injected finds a <style> that Svelte writes for a component compiled with
// css="injected".
const injected = /<style id="[^"]*">[\s\S]*?<\/style>/g;

// Rendered is what every render starts from: what it is to show, then a
// mark where what that gave the head ends. Svelte writes the <title> after
// the head.
function Rendered($$renderer, { show }) {
  show($$renderer);
  $$renderer.head(($$renderer) => $$renderer.push(headEnd));
}

// ---------------------------------------------------------------------------
// What is computed

// A record is everything computed for one page, or for one shared block: its
// blocks, made as what Svelte rendered is read. Blocks of a record name each
// other by their place in it.
class Record {
  constructor(kind, key, route) {
    this.kind = kind; // "p" for a page, "s" for a shared block
    this.key = key; // the address of the page, or the name of the shared block
    this.route = route; // the route of a page, by its place among the routes
    this.blocks = [];
    this.top = 0;
    // The shared blocks marked while Svelte renders, { name, snippets }, and
    // the islands, in JSON. A render that Svelte does again, as it does
    // for bind:, leaves some of them unmarked: only what is marked in the
    // end becomes a block.
    this.uses = [];
    this.islands = [];
    // What was asked of the data database on the way. The rows it gave note
    // what is read of them until the record is complete: see encode.
    this.reads = new Reads();
  }

  // A block is what Svelte rendered: HTML, what it gives the head of the
  // page and the <title> among that; or a use, which stands for a shared
  // block. HTML is a fragment, see fragment.
  block() {
    const block = { index: this.blocks.length, shared: null, snippets: null, body: [""], head: [""], title: "" };
    this.blocks.push(block);
    return block;
  }

  // render renders show with Svelte and keeps what it rendered as a block.
  // idPrefix starts the ids of $props.id().
  render(show, idPrefix) {
    const { body, head, title } = rendered(show, { idPrefix });
    const block = this.block();
    block.body = this.fragment(body);
    block.head = this.fragment(head);
    block.title = title;
    return block.index;
  }

  // use makes the block that stands where a shared block is shown, and
  // names it. The shared block is kept once; the snippets this place gave
  // it are kept here, a block each.
  use({ name, snippets }) {
    const use = this.block();
    use.shared = name;
    use.snippets = new Map();
    for (const snippet of snippets) {
      const block = this.block();
      block.body = this.fragment(snippet.body);
      block.head = this.fragment(snippet.head);
      block.title = snippet.title;
      use.snippets.set(snippet.name, [block.index]);
    }
    return use;
  }

  // fragment reads what Svelte rendered: HTML with marks. It takes the
  // comments of Svelte out of the HTML, outside {@html}, and makes the
  // blocks of the shared blocks marked in it. A fragment is texts with holes
  // between them, [text, hole, text, …, text]; a hole is c and the number
  // of a block, gname or i2.
  fragment(text) {
    const parts = text.split(marks);
    const out = [];
    let html = "";
    let raw = false;
    for (let at = 0; at < parts.length; at++) {
      const part = parts[at];
      if (at % 2 === 0) {
        html += raw ? part : part.replace(comments, "");
        continue;
      }
      switch (part[0]) {
        case "[":
          raw = true;
          break;
        case "]":
          raw = false;
          break;
        case "c":
          out.push(html, "c" + this.use(this.uses[Number(part.slice(1))]).index);
          html = "";
          break;
        default:
          out.push(html, part);
          html = "";
      }
    }
    out.push(html);
    return out;
  }

  // island keeps an island and returns its mark.
  island(json) {
    return mark("i" + (this.islands.push(json) - 1));
  }
}

// ---------------------------------------------------------------------------
// Components

// The components Svelte compiled from the templates, by their files.
const components = new Map();

// component is what a template exports in the bundle: the component Svelte
// compiled from it, in a function that sees where it is shown. A component
// that shows the same wherever it stands, for the props it is given, is
// computed once, as a shared block, and only marked where it stands.
export function component(file, compiled) {
  components.set(file, compiled);
  return function shown($$renderer, props) {
    const shared = present?.record ? share(file, props) : null;
    const snippets = shared && apart(shared.given, props);
    if (snippets) {
      $$renderer.push(mark("c" + (present.record.uses.push({ name: shared.name, snippets }) - 1)));
      return;
    }
    if (showing.length === deepest) throw new Error(endless(file));
    showing.push(file);
    try {
      return compiled($$renderer, props);
    } finally {
      showing.pop();
    }
  };
}

// The files of the components being rendered in place, the outermost first.
// A component that shows itself without end is told so, with the way round,
// before the stack of JavaScript runs out.
const showing = [];
const deepest = 512;

function endless(file) {
  const from = showing.lastIndexOf(file);
  if (from < 0) return `components are shown inside each other more than ${deepest} deep, the innermost ${site.files[file].path}`;
  const round = [...showing.slice(from), file].map((one) => site.files[one].path);
  return `${round[0]} shows itself without end: ${round.join(" → ")}`;
}

// share tells whether a component is shown as a shared block: its file reads
// nothing of the page and of the place it stands at, and it is given plain
// values and snippets without parameters, written inside its tag. The name
// of the block is its file and those values, and the names of the snippets
// after "|": components/Card.svelte{"id":"p7"|children}. null: it is not.
// A prop of bind: is read and written through the component: it is not a
// value.
function share(file, props) {
  const { path, shared } = site.files[file];
  if (!shared) return null;
  const slots = props.$$slots;
  const values = [];
  const given = [];
  for (const name of Object.keys(props)) {
    if (name === "$$slots") continue;
    const property = Object.getOwnPropertyDescriptor(props, name);
    if (property.get || property.set) return null;
    const value = property.value;
    if (value === undefined) continue;
    if (typeof value === "function" && slots?.[name === "children" ? "default" : name]) {
      // A snippet with parameters is rendered where it is shown.
      if (takesParameters(value)) return null;
      given.push(name);
      continue;
    }
    const type = typeof value;
    if (value !== null && type !== "string" && type !== "boolean" && !(type === "number" && Number.isFinite(value))) return null;
    values.push(name);
  }
  return { name: nameOf(path, props, values, given), given };
}

// takesParameters tells whether a snippet compiled by Svelte takes
// parameters besides the renderer: function row($$renderer, item = 1).
// Parameters with default values do not count in the length of a function,
// so its source is read.
function takesParameters(snippet) {
  if (snippet.length > 1) return true;
  const source = Function.prototype.toString.call(snippet);
  const open = source.indexOf("(");
  const arrow = source.indexOf("=>");
  if (open < 0 || (arrow >= 0 && arrow < open)) return false; // $$renderer => …
  return /^\s*[\w$]+\s*,/.test(source.slice(open + 1));
}

// apart renders the snippets a place gives a shared block, each as a render
// of its own: a snippet without parameters shows the same wherever the
// shared block shows it. It sees the context of the place. null when one
// cannot be rendered apart: it fails where the shared block would not show
// it, or it calls $props.id(), whose ids count what was rendered before them
// on the page. Then the component is rendered in place, with its snippets,
// as Svelte renders them.
function apart(names, props) {
  const out = [];
  for (const name of names) {
    let snippet;
    try {
      snippet = rendered(($$renderer) => props[name]($$renderer), { context: getAllContexts() });
    } catch {
      return null;
    }
    if (snippet.body.includes("<!--$") || snippet.head.includes("<!--$")) return null;
    out.push({ name, ...snippet });
  }
  return out;
}

// nameOf names a shared block. Props in another order are the same props.
// JSON writes every control character as an escape, so a name never holds
// one of the separators of the wire.
function nameOf(path, props, values, given) {
  const key = values.sort().map((name) => JSON.stringify(name) + ":" + JSON.stringify(props[name])).join(",");
  return path + "{" + key + (given.length > 0 ? "|" + given.sort().join(",") : "") + "}";
}

// unnamed reads a name that nameOf wrote. The names of the snippets follow
// the last "|" and are names of JavaScript; a "|" inside a prop is followed
// by the quote that closes its text at the least.
function unnamed(name) {
  const start = name.indexOf("{");
  let inside = name.slice(start + 1, -1);
  let given = [];
  const bar = inside.lastIndexOf("|");
  if (bar >= 0 && /^[\p{ID_Continue}$,]*$/u.test(inside.slice(bar + 1))) {
    given = inside.slice(bar + 1).split(",").filter((one) => one !== "");
    inside = inside.slice(0, bar);
  }
  return { path: name.slice(0, start), props: JSON.parse("{" + inside + "}"), given };
}

// givenBy stands, inside a shared block, for a snippet that the place that
// shows the block gave it: the place rendered the snippet, and the Go side
// puts it here. The snippet takes no parameters, so what the shared block
// passes it changes nothing, as in Svelte.
function givenBy(name) {
  return ($$renderer) => $$renderer.push(mark("g" + name));
}

// island is what a template imports of an island: a component that renders
// the island with the Svelte of the project and marks it in the page. The Go
// side writes it with what hydrates it in the browser.
export function island(target) {
  return function shown($$renderer, given) {
    const props = {};
    const request = {};
    for (const name in given) {
      if (name === "$$slots") continue;
      if (!name.startsWith("request:")) {
        props[name] = given[name];
        continue;
      }
      const prop = name.slice("request:".length);
      const source = requestSources.get(given[name]);
      if (!source) throw new TypeError(`request:${prop} binds the prop to the results of a search: request:${prop}={search("products", { … })}, with import { search } from "shugo"`);
      request[prop] = source;
      props[prop] = given[name];
    }
    // Normalize before SSR so the browser hydrates with the same values.
    const serial = JSON.stringify(props, (_, value) => {
      if (typeof value === "function" || typeof value === "symbol") {
        throw new TypeError("Svelte island props must be JSON data");
      }
      return value;
    });
    const { body, head } = target.render(JSON.parse(serial));
    if (head) throw new Error("Put the island’s head elements in its Shugo page");
    $$renderer.push(present.record.island(JSON.stringify({
      id: target.id, url: target.url, css: target.css,
      request: Object.keys(request).length > 0 ? request : undefined,
      props: JSON.parse(serial), body,
    })));
  };
}

// html is {@html …} of Svelte for the templates of the project: the markup
// as it is, between marks, so that the comments in it stay.
export function html(value) {
  return mark("[") + String(value ?? "") + mark("]");
}

// ---------------------------------------------------------------------------
// The module "shugo": what the code of a project asks of the framework.
//
//   import { query, paginate, page } from "shugo"
//
// What they answer depends on what is being computed: present is the page or
// shared block being computed, or the list of pages of a route. Nothing
// waits, so one computation runs at a time.
//   reads   where its queries leave their readings
//   page    the page: { params, path, route, origin, url }; null in a block
//           kept once for all pages, and while a list of pages is made
//   route   the route whose list of pages is made; null otherwise
//   record  the record of the page or of the shared block; null otherwise
let present = null;

// computing runs compute as the given computation.
function computing(now, compute) {
  const outer = present;
  present = now;
  try {
    return during(now.reads, compute);
  } finally {
    present = outer;
  }
}

// query starts a question to a table of the data database. What it reads is
// noted for the page, the shared block or the list of pages being computed.
export function query(table) {
  if (present === null) {
    throw new Error(`query("${table}") is asked while nothing is computed: ask in a page, a component or getStaticPaths, or in a function they call, not at the top of a module`);
  }
  if (data === null) {
    throw new Error('query() reads the data database of the project, and this project names none. Name it in shugo.json: { "data": "data/site.db" }');
  }
  return new Query(data, present.reads, table);
}

// paginate cuts a list into pages. In
// getStaticPaths it takes how many items there are and says which pages
// there are. In a page it takes the list itself, a query or an array, and
// returns the page whose number stands in the address: { data, total,
// currentPage, lastPage, url: { prev, next, … } }; a query asks the data
// database for that page alone.
export function paginate(list, options = {}) {
  if (present?.route) return paginator(present.route)(list, options);
  const { pageSize = 10 } = options;
  const asked = list instanceof Query;
  if (!asked && !Array.isArray(list)) throw new TypeError(`paginate() in a page takes a query or an array, and got ${describe(list)}: paginate(query("posts").order("date", "desc"), { pageSize: 10 })`);
  const { route, params } = pageNow("paginate()");
  const segment = pageSegment(route);
  const total = asked ? list.count() : list.length;
  const current = params.page === undefined ? 1 : Number(params.page);
  const start = (current - 1) * pageSize;
  const found = asked ? list.limit(pageSize).offset(start).all() : list.slice(start, start + pageSize);
  return paged(route, segment, params, found, start, pageSize, total, current);
}

// page is the page being computed, as page of $app/state is in SvelteKit.
export const page = Object.freeze({
  get url() {
    const now = pageNow("page.url");
    return (now.url ??= new URL(encodeURI(now.path), now.origin));
  },
  get params() {
    return pageNow("page.params").params;
  },
  get route() {
    return { id: pageNow("page.route").route.pattern };
  },
  get status() {
    return pageNow("page.status").path === "/404" ? 404 : 200;
  },
});

// pageNow returns the page being computed.
function pageNow(what) {
  if (present?.page) return present.page;
  if (present?.route) throw new Error(`${what} is read while the list of pages of ${present.route.pattern} is made: getStaticPaths says which pages there are, and no page is computed yet`);
  if (present) throw new Error(`${what} was read inside a block that is computed once for all pages. This should not happen: please report it`);
  throw new Error(`${what} is read while nothing is computed: read it in a page or a component, or in a function they call`);
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

class Run {
  constructor(modules, send) {
    this.modules = modules; // Map of file number → module
    this.send = send;
    this.files = new Map(site.files.map((file, number) => [file.path, number]));
    this.pages = new Map(); // address → the route that has the page, once the pages are listed
    this.batch = [];
    this.batchSize = 0;
  }

  // list finds the pages of routes: of the ones named, or of all. Every
  // route says which pages it has, and what it read from the data to know.
  // Among the routes listed together, a route that names more of an address
  // comes earlier and takes it. An address a route gives twice is one page,
  // as in SvelteKit: the page is computed from its address alone.
  list(wanted) {
    const routes = wanted.length > 0 ? wanted : site.routes.map((_, index) => index);
    for (const [path, index] of this.pages) if (routes.includes(index)) this.pages.delete(path);
    let shadowed = 0;
    let lines = [];
    for (const index of routes) {
      const route = site.routes[index];
      if (route === undefined) throw new Error(`the shugo command asked for the pages of route ${index}, which this project's compiled code does not have: build the project again`);
      const reads = new Reads();
      try {
        for (const entry of computing({ reads, page: null, route, record: null }, () => entries(route, this.modules.get(route.file)))) {
          const path = address(route, entry.params);
          const taken = this.pages.get(path);
          if (taken === index) continue;
          if (taken !== undefined && taken < index) {
            shadowed++;
            continue;
          }
          this.pages.set(path, index);
          if (lines.push(index + "\x02" + path) === 4096) {
            this.send("P", lines.join("\x03"));
            lines = [];
          }
        }
      } catch (error) {
        throw placed(error, `getStaticPaths of ${site.files[route.file].path}`);
      }
      this.send("G", index + "\x02" + reads.text());
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
        throw placed(error, `the page ${name} (${site.files[site.routes[Number(route)]?.file]?.path})`);
      }
    }
    this.flush();
  }

  // block computes a shared block from its name alone: the name holds the
  // file, the props and the names of the snippets the place gave it, and,
  // for a layout kept for one page, that page. The ids of $props.id() in a
  // block computed apart from the page start with the number of its file.
  block(name) {
    const { path, props, given } = unnamed(name);
    const file = this.files.get(path);
    const compiled = components.get(file);
    if (compiled === undefined) throw new Error(`there is no file ${path} to compute the shared block from`);
    const page = this.pageOf(props);
    if (given.length > 0) {
      props.$$slots = {};
      for (const snippet of given) {
        props[snippet] = givenBy(snippet);
        props.$$slots[snippet === "children" ? "default" : snippet] = true;
      }
    }
    const record = new Record("s", name, 0);
    computing({ reads: record.reads, page, route: null, record }, () => {
      record.top = record.render(($$renderer) => compiled($$renderer, props), "b" + file);
    });
    this.emit(record);
  }

  // page computes one page of a route from its address alone: a page of
  // templates and every layout around it, or the file that a module gives.
  // The parameters are read from the address: what getStaticPaths wrote as a
  // number is a text there.
  page(path, index) {
    const route = site.routes[index];
    if (route === undefined) throw new Error(`the page ${path} was asked for under route ${index}, which this project's compiled code does not have: build the project again`);
    const record = new Record("p", path, index);
    const page = this.pageOf({ "@route": index, "@path": path });
    computing({ reads: record.reads, page, route: null, record }, () => (route.extension === undefined ? this.around(record, index) : this.file(record, route)));
    this.emit(record);
  }

  // file computes a file that a module among the pages gives: the text its
  // function GET returns, the one block of the record.
  file(record, route) {
    const path = site.files[route.file].path;
    const { GET } = this.modules.get(route.file);
    if (typeof GET !== "function") throw new Error(`${path} gives the file ${record.key}, so it must export the function GET, which returns the text of the file: export function GET() { return "…" }`);
    const text = GET(noArguments("GET", 'import { query, page } from "shugo" at the top of the module, and read page.params and page.url'));
    if (typeof text?.then === "function") throw new Error(`GET of ${path} waits (it is async or returns a promise): files are computed without waiting; the data comes from the data database through query`);
    if (typeof text !== "string") {
      const hint = typeof Response === "function" && text instanceof Response ? ": a Response is read by waiting, so return the text itself" : typeof text === "object" && text !== null ? ": JSON.stringify(…) makes one of data" : "";
      throw new Error(`GET of ${path} must return the text of the file, and returned ${describe(text)}${hint}`);
    }
    const block = record.block();
    block.body = [text];
    record.top = block.index;
  }

  // pageOf returns the page a shared block is kept for: the route and the
  // address its name holds as "@route" and "@path", which it takes out of
  // the props. null for a block kept once for all pages. Its parameters are
  // read from the address: what getStaticPaths wrote as a number is a text
  // there.
  pageOf(props) {
    if (!Object.hasOwn(props, "@route")) return null;
    const index = props["@route"];
    const path = props["@path"];
    delete props["@route"];
    delete props["@path"];
    const route = site.routes[index];
    if (route === undefined || typeof path !== "string") throw new Error(`a block of the page ${path} under route ${index}, which this project's compiled code does not have: build the project again`);
    return { params: Object.freeze(parameters(route, path)), path, route, origin: site.origin, url: undefined };
  }

  // around computes a page into the record, and names its layouts. A layout
  // shows the page as its children and is kept apart from it: once for the
  // whole site, or, when it reads the page it is around, once for every
  // page. Then a change of what the layout reads computes the layout and
  // not the pages, and a change of a page computes the page alone.
  around(record, index) {
    const route = site.routes[index];
    const compiled = components.get(route.file);
    let top = record.render(($$renderer) => compiled($$renderer, {}), "");
    for (let i = route.layouts.length - 1; i >= 0; i--) {
      const file = site.files[route.layouts[i]];
      const props = file.shared ? {} : { "@route": index, "@path": record.key };
      const use = record.block();
      use.shared = nameOf(file.path, props, Object.keys(props), ["children"]);
      use.snippets = new Map([["children", [top]]]);
      top = use.index;
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

// On the wire a record is its header and its blocks, split by \x01; fields
// are split by \x02. The header ends with what the record read from the data
// database. A use is "u", the name of the shared block and the snippets the
// place gave it, name=1,2;other=3. A block of HTML is "b", its HTML, what it
// gives the head and its <title>; HTML is texts and holes split by \x00, a
// hole c3, gname, or i and the island in JSON.
function encode(record) {
  const lines = [record.kind + "\x02" + record.key + "\x02" + record.route + "\x02" + record.top + "\x02" + record.reads.text()];
  for (const block of record.blocks) {
    lines.push(
      block.shared !== null
        ? "u\x02" + block.shared + "\x02" + snippetsOf(block.snippets)
        : "b\x02" + fragmentOf(record, block.body) + "\x02" + fragmentOf(record, block.head) + "\x02" + escape(block.title),
    );
  }
  return lines.join("\x01");
}

function fragmentOf(record, parts) {
  let out = escape(parts[0]);
  for (let i = 1; i < parts.length; i += 2) {
    const hole = parts[i][0] === "i" ? "i" + record.islands[Number(parts[i].slice(1))] : parts[i];
    out += "\x00" + escape(hole) + "\x00" + escape(parts[i + 1]);
  }
  return out;
}

function snippetsOf(snippets) {
  let out = "";
  for (const [name, blocks] of snippets) out += (out ? ";" : "") + name + "=" + blocks.join(",");
  return out;
}

// The Go side splits what it receives by the characters 0 to 3. Inside a
// text they come escaped, as 4 and the character plus 0x40, and so does 4.
const special = /[\x00-\x04]/;
const everySpecial = /[\x00-\x04]/g;

function escape(text) {
  return special.test(text) ? text.replace(everySpecial, (c) => "\x04" + String.fromCharCode(0x40 + c.charCodeAt(0))) : text;
}

// ---------------------------------------------------------------------------
// Routes

// address builds the address of a page from the route and the parameters
// that getStaticPaths returned for it. The address of a file that a module
// gives ends with the extension of its route.
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
  if (route.extension === undefined) return "/" + parts.join("/");
  if (parts.length === 0) throw new Error(`getStaticPaths of ${route.pattern} returned params that leave no name before ${route.extension}`);
  return "/" + parts.join("/") + route.extension;
}

// parameters reads the parameters of a page from its address: what address()
// does, backwards. The route is known, so the address fits it.
function parameters(route, path) {
  if (route.extension !== undefined) path = path.slice(0, -route.extension.length);
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

// paged describes one page of a list: its rows, where they stand in the
// list, and the addresses of the pages around it.
function paged(route, segment, params, rows, start, size, total, current) {
  const last = Math.max(1, Math.ceil(total / size));
  const link = (page) => (page < 1 || page > last ? undefined : address(route, { ...params, page: pageNumber(segment, page) }));
  return {
    data: rows, start, end: Math.min(start + size, total) - 1, size, total,
    currentPage: current, lastPage: last,
    url: { current: link(current), prev: link(current - 1), next: link(current + 1), first: current > 1 ? link(1) : undefined, last: current < last ? link(last) : undefined },
  };
}

// paginator is paginate in getStaticPaths: given how many items there are,
// it says which pages a list cut into pages has. Every page asks for its
// part itself: paginate(query, { pageSize }).
function paginator(route) {
  return (total, { pageSize = 10, params = {} } = {}) => {
    if (typeof total !== "number" || !Number.isInteger(total) || total < 0) {
      throw new TypeError(`paginate() in getStaticPaths takes how many items there are, got ${describe(total)}: paginate(query("posts").count(), { pageSize: 10 }); every page takes its part with paginate(query("posts"), { pageSize: 10 })`);
    }
    const segment = pageSegment(route);
    const last = Math.max(1, Math.ceil(total / pageSize));
    return Array.from({ length: last }, (_, i) => ({ params: { ...params, page: pageNumber(segment, i + 1) } }));
  };
}

// entries returns the pages of a route: one for a page without parameters,
// and what getStaticPaths says for the others.
function entries(route, module) {
  if (!route.segments.some((segment) => segment.text === undefined)) return [{ params: {} }];
  const where = route.extension === undefined ? "<script module>" : "the module";
  if (typeof module.getStaticPaths !== "function") throw new Error(`${route.pattern} has parameters, so ${where} must export getStaticPaths`);
  const list = module.getStaticPaths(noArguments("getStaticPaths", `import { query, paginate } from "shugo" at the top of ${where}`));
  if (typeof list?.then === "function") throw new Error(`getStaticPaths of ${route.pattern} waits (it is async or returns a promise): pages are computed without waiting; the data comes from the data database through query`);
  if (!Array.isArray(list)) throw new Error(`getStaticPaths of ${route.pattern} must return a list of { params }, got ${describe(list)}`);
  for (const entry of list) {
    if (entry === null || typeof entry !== "object" || entry.params === null || typeof entry.params !== "object") {
      throw new Error(`getStaticPaths of ${route.pattern} must return a list of { params: { … } }, and one item is ${describe(entry)}`);
    }
    if (entry.props !== undefined) {
      throw new Error(`getStaticPaths of ${route.pattern} gives its pages addresses alone: return { params: { … } }, and let every page ask for its data with query(), by page.params`);
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
  const wire = buildChannel();
  if (wire === null) {
    console.error("This file is a part of a Shugo project and is run by the shugo command: `shugo build` or `shugo update`.");
    process.exit(2);
  }
  const { send, receive } = wire;

  try {
    site = JSON.parse(readFileSync(new URL("./site.json", import.meta.url), "utf8"));
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
    const run = new Run(new Map(modules), send);
    if (process.env.SHUGO_SEARCH === "1") send("A", "");
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

// getStaticPaths and GET take nothing: what they need comes from the module
// shugo. One that takes { query, paginate } or { params } is told where to
// find it.
function noArguments(what, instead) {
  return new Proxy({}, {
    get(_, name) {
      throw new Error(`${what} gets no ${String(name)}: ${instead}`);
    },
  });
}

// The global Shugo of the first version is gone: code that reaches for it
// is told what to write.
const hints = { props: "let { … } = $props()", params: "page.params, with import { page } from \"shugo\"", url: "page.url, with import { page } from \"shugo\"", query: "query(…), with import { query } from \"shugo\"", paginate: "paginate(…), with import { paginate } from \"shugo\"", slots: "a snippet taken from $props(): {#if children}" };
globalThis.Shugo ??= new Proxy({}, {
  get(_, name) {
    const instead = hints[String(name)];
    throw new Error(`There is no Shugo.${String(name)} in Shugo${instead ? `: write ${instead}` : ""}`);
  },
});
