// The compiler of Svelte as Shugo runs it. It reads requests from its input,
// one JSON line each, and answers each with one JSON line, or with the
// errors in the file:
//
//   { path, source }                a template: the code Svelte compiled for
//                                   a server, its source map, the styles of
//                                   the component, the warnings of Svelte and
//                                   what Shugo needs to know of the file
//   { path, source, module }        a module with runes, a .svelte.js file:
//                                   its code for a server
//   { path, source, island }        a component of an island, or a module
//                                   with runes when module is set too: its
//                                   code for the browser and for the server
//   { path, source, lower }         the server code of an island bound to a
//                                   request: the program Go builds its HTML
//                                   by, see island-program.mjs
//
// The compiler is that of the package svelte, which the package shugo
// depends on, and so is acorn, the parser of JavaScript Svelte uses. The
// folder of the package and the version Shugo is made for come after the
// code.

import { createRequire } from 'node:module';
import { join } from 'node:path';
import { createInterface } from 'node:readline';

const [svelteFolder, version] = process.argv.slice(-2);
const fromSvelte = createRequire(join(svelteFolder, 'package.json'));
const svelte = fromSvelte('./compiler/index.js');
if (svelte.VERSION !== version) throw new Error(`Shugo is made for Svelte ${version}, and the compiler of ${svelteFolder} is ${svelte.VERSION}`);
const acorn = fromSvelte('acorn');
const parseGenerated = (code) => acorn.parse(code, { ecmaVersion: 'latest', sourceType: 'module', locations: true });


// compileTemplate compiles one template of a project for the server.
function compileTemplate(path, input) {
  const source = input.charCodeAt(0) === 0xfeff ? input.slice(1) : input;
  let result;
  try {
    result = svelte.compile(source, {
      filename: path,
      generate: 'server',
      css: 'external',
      runes: true,
      modernAst: true,
      discloseVersion: false,
    });
  } catch (error) {
    if (!Array.isArray(error?.position)) throw error;
    return { errors: [problem(source, error.position[0], told(source, error))] };
  }
  const look = new Look(source, result.ast);
  if (look.errors.length > 0) return { errors: look.errors };
  return {
    code: result.js.code,
    map: JSON.stringify(result.js.map),
    css: result.css?.code ?? '',
    warnings: result.warnings
      .filter((w) => !staticPages.has(w.code) && !look.quiet.has(w.position?.[0]))
      .map((w) => problem(source, w.position?.[0] ?? 0, w.message)),
    facts: look.facts,
  };
}

// compileModule compiles a module with runes for the server.
function compileModule(path, source) {
  let result;
  try {
    result = svelte.compileModule(source, { filename: path, generate: 'server' });
  } catch (error) {
    if (!Array.isArray(error?.position)) throw error;
    return { errors: [problem(source, error.position[0], told(source, error))] };
  }
  return {
    code: result.js.code,
    map: JSON.stringify(result.js.map),
    warnings: result.warnings.filter((w) => !staticPages.has(w.code)).map((w) => problem(source, w.position?.[0] ?? 0, w.message)),
  };
}

// compileIsland compiles a component of an island, or a module with runes,
// for the browser and for the server, with the same options as a template.
// Its styles come from the code for the browser.
function compileIsland(path, source, module) {
  let client, server;
  try {
    if (module) {
      client = svelte.compileModule(source, { filename: path, generate: 'client' });
      server = svelte.compileModule(source, { filename: path, generate: 'server' });
    } else {
      const options = { filename: path, css: 'external', runes: true, discloseVersion: false };
      client = svelte.compile(source, { ...options, generate: 'client' });
      server = svelte.compile(source, { ...options, generate: 'server' });
    }
  } catch (error) {
    if (!Array.isArray(error?.position)) throw error;
    return { errors: [problem(source, error.position[0], told(source, error))] };
  }
  return {
    client: client.js.code,
    server: server.js.code,
    map: JSON.stringify(server.js.map),
    css: client.css?.code ?? '',
    warnings: client.warnings.map((w) => problem(source, w.position?.[0] ?? 0, w.message)),
  };
}

// lower makes the program of an island bound to a request from its server
// code. A mistake is told at its place in that code.
function lower(code) {
  try {
    return { program: lowerIsland(code, parseGenerated) };
  } catch (error) {
    const at = error.start ?? error.loc;
    if (at === undefined) throw error;
    return { errors: [{ line: at.line, column: at.column + 1, message: error.message.replace(/\s*\(\d+:\d+\)$/, '') }] };
  }
}

// told is what Shugo says of a mistake: the message of Svelte, or one that
// tells better what to write.
function told(source, error) {
  if (error.code === 'js_parse_error' && /^<[a-zA-Z]/.test(source.slice(error.position[0]))) {
    return 'markup cannot stand inside {…}: show it under a condition with {#if …}…{/if}, and repeat it with {#each … as …}…{/each}';
  }
  return messages[error.code] ?? error.message;
}

// What Shugo says in place of Svelte, for a mistake it explains better.
const messages = {
  experimental_async: 'a page is computed without waiting: its data comes from query of the module shugo, which answers at once, so there is nothing to wait for. What takes time and needs doing once, such as making a highlighter, can be awaited at the top of <script module> or of a module in src/lib/ and used here',
};

// The warnings of Svelte about what changes in a browser: a page of a
// static site is rendered once, and nothing on it changes. An island is
// compiled for the browser too, and warned about there.
const staticPages = new Set([
  'state_referenced_locally',
  'non_reactive_update',
  'binding_property_non_reactive',
  'perf_avoid_inline_class',
  'perf_avoid_nested_class',
  'store_rune_conflict',
]);

// problem is a message about a place in a file: its line and its column,
// both from 1, the column in characters.
function problem(source, at, message) {
  let line = 1;
  let start = 0;
  for (let i = source.indexOf('\n'); i !== -1 && i < at; i = source.indexOf('\n', i + 1)) {
    line++;
    start = i + 1;
  }
  return { line, column: [...source.slice(start, at)].length + 1, message: message.replace(/\s*\n\s*/g, ' ') };
}

// The tags of Shugo's first syntax and of Svelte 4, with what to write instead.
const oldTags = {
  'g-if': '{#if condition}…{:else if other}…{:else}…{/if}',
  'g-else-if': '{:else if condition} inside {#if}',
  'g-else': '{:else} inside {#if} or {#each}',
  'g-for': '{#each items as item, i}…{/each}',
  'g-html': '{@html value}',
  'g-slot': '{@render children?.()}, with children taken from $props()',
  'g-fragment': "{#snippet name()}…{/snippet} inside the component's tag",
  slot: '{@render children?.()}, with children taken from $props(): Svelte 5 passes markup as snippets',
};

// The functions of Svelte that pass values down the tree of components
// without parameters: a component that uses them depends on where it stands.
const contextFunctions = new Set(['getContext', 'setContext', 'hasContext', 'getAllContexts', 'createContext']);

// Look reads what Shugo needs to know of a template from its tree, and finds
// the mistakes that are Shugo's own.
//
//   page     it imports page or paginate from the module shugo: it shows
//            what differs from page to page
//   context  it uses the context of Svelte
//   id       it calls $props.id()
//   select   it shows a snippet inside a <select>, whose value chooses the
//            <option> the snippet shows
//   renders  it shows a snippet: a layout shows the page with {@render children()}
//   paths    its <script module> exports getStaticPaths
//   islands  the islands it imports: [{ source, request }], request when a
//            tag binds a prop of the island to the data of a request
class Look {
  constructor(source, ast) {
    this.source = source;
    this.errors = [];
    this.quiet = new Set(); // where a warning of Svelte is answered by Shugo
    this.facts = { page: false, context: false, id: false, select: false, renders: false, paths: false, islands: [] };
    this.islands = new Map(); // the name an island is imported under → { source, request }
    if (ast.options?.runes === false) {
      const at = ast.options.attributes.find((a) => a.name === 'runes')?.start ?? ast.options.start;
      this.error(at, 'runes={false} turns on the syntax of Svelte 4, and the templates of Shugo are written with runes: take this option away, take props with $props() and write $derived(…) in place of $:');
    }
    if (ast.module) this.script(ast.module, true);
    if (ast.instance) this.script(ast.instance, false);
    this.fragment(ast.fragment, [], null);
    // An island imported under two names is one island, bound to a request
    // when either tag binds it.
    const bySource = new Map();
    for (const { source, request } of this.islands.values()) {
      bySource.set(source, { source, request: request || (bySource.get(source)?.request ?? false) });
    }
    this.facts.islands = [...bySource.values()];
  }

  error(at, message) {
    this.errors.push(problem(this.source, at, message));
  }

  // shell tells, once for a template, that it writes the document around
  // the pages, which src/site.html writes.
  shell(node) {
    if (this.toldShell) return;
    this.toldShell = true;
    this.error(node.start, `${this.source.slice(node.start, this.source.indexOf('>', node.start) + 1)} is written in src/site.html, the shell of every page: a layout starts with what goes into the <body>, and a page gives the <head> what it needs with <svelte:head>`);
  }

  script(script, module) {
    for (const node of script.content.body) {
      switch (node.type) {
        case 'ImportDeclaration':
          this.imports(node);
          break;
        case 'ExportAllDeclaration':
          if (node.source.value === 'shugo') this.facts.page = true;
          break;
        case 'ExportNamedDeclaration':
          if (node.source?.value === 'shugo' && node.specifiers.some((s) => ['page', 'paginate'].includes(s.local.name))) this.facts.page = true;
          if (exportsName(node, 'getStaticPaths')) {
            if (module) this.facts.paths = true;
            else this.error(node.start, 'getStaticPaths goes into <script module>, which runs once for the file: <script module> export function getStaticPaths() { … } </script>');
          }
          break;
      }
    }
    if (!module) {
      walk(script.content, (node) => {
        if (node.type === 'MemberExpression' && node.object.type === 'Identifier' && node.object.name === '$props' && node.property.name === 'id') {
          this.facts.id = true;
        }
      });
    }
  }

  imports(node) {
    const from = node.source.value;
    for (const s of node.specifiers) {
      const name = s.type === 'ImportSpecifier' ? s.imported.name ?? s.imported.value : s.type === 'ImportNamespaceSpecifier' ? '*' : 'default';
      if (from === 'shugo' && (name === 'page' || name === 'paginate' || name === '*')) this.facts.page = true;
      if (from === 'svelte' && (contextFunctions.has(name) || name === '*')) this.facts.context = true;
      if (!from.endsWith('.island.svelte')) continue;
      if (name === 'default') this.islands.set(s.local.name, { source: from, request: false });
      else this.error(s.start, `import an island as a default component: import ${s.local.name} from "${from}"`);
    }
  }

  // fragment looks at the nodes of a template. inside are the names of the
  // elements around them, and parent the node they stand in.
  fragment(fragment, inside, parent) {
    for (const node of fragment?.nodes ?? []) this.node(node, inside, parent);
  }

  node(node, inside, parent) {
    if (Object.hasOwn(oldTags, node.name ?? '') && ['RegularElement', 'SlotElement', 'Component'].includes(node.type)) {
      this.error(node.start, `there is no <${node.name}>: write ${oldTags[node.name]}`);
    }
    // slot="…" right inside the tag of a component is a slot of Svelte 4;
    // elsewhere it is the slot of a custom element.
    const inComponent = ['Component', 'SvelteComponent', 'SvelteSelf'].includes(parent?.type);
    for (const a of node.attributes ?? []) {
      if ((a.type === 'Attribute' && a.name === 'slot' && inComponent) || a.type === 'LetDirective') {
        this.error(a.start, `${a.type === 'Attribute' ? 'slot="…"' : 'let:'} is a part of the slots of Svelte 4: write {#snippet name()}…{/snippet} inside the component's tag and {@render name()} in the component`);
      }
    }
    switch (node.type) {
      case 'SvelteFragment':
        this.error(node.start, "<svelte:fragment> is a part of the slots of Svelte 4: write {#snippet name()}…{/snippet} inside the component's tag");
        break;
      case 'RenderTag':
        this.facts.renders = true;
        if (inside.includes('select')) this.facts.select = true;
        break;
      case 'RegularElement':
        if (['!doctype', 'html', 'head', 'body'].includes(node.name.toLowerCase()) && !inside.includes('svg')) this.shell(node);
        this.notBound(node);
        break;
      case 'SvelteComponent':
        if (node.expression.type === 'Identifier' && this.islands.has(node.expression.name)) {
          this.error(node.start, `a Svelte island is rendered by its imported tag: <${node.expression.name} />`);
        }
        break;
      case 'Component': {
        const island = this.islands.get(node.name);
        if (island) this.island(node, island, inside);
        else this.notBound(node);
        break;
      }
    }
    const within = node.type === 'RegularElement' || node.type === 'SvelteHead' ? [...inside, node.type === 'SvelteHead' ? 'svelte:head' : node.name] : inside;
    for (const key of ['fragment', 'consequent', 'alternate', 'body', 'fallback', 'pending', 'then', 'catch']) {
      if (node[key]?.type === 'Fragment') this.fragment(node[key], within, node);
    }
  }

  // notBound tells of request:name={…} on a tag that is not of an island.
  notBound(node) {
    for (const a of node.attributes) {
      if (a.type === 'Attribute' && a.name.startsWith('request:')) {
        this.quiet.add(a.start);
        this.error(a.start, `${a.name} binds a prop on an .island.svelte component to the data of a request, and <${node.name}> is not one`);
      }
    }
  }

  // island looks at the tag of an island: it takes JSON props alone, and may
  // bind some of them to the data of a request with request:name={…}.
  island(node, island, inside) {
    const place = ['svelte:head', 'svg', 'select', 'option', 'optgroup', 'textarea', 'template'].find((name) => inside.includes(name));
    if (place) this.error(node.start, `the Svelte island <${node.name}> needs a place for its wrapper in the HTML body, and stands inside <${place}>`);
    if (node.fragment.nodes.some((n) => n.type !== 'Text' || n.data.trim() !== '')) {
      this.error(node.start, `the Svelte island <${node.name}> takes JSON props; its tag cannot pass children or snippets`);
    }
    const plain = new Set();
    const bound = [];
    for (const a of node.attributes) {
      if (a.type === 'SpreadAttribute') continue;
      if (a.type !== 'Attribute') {
        this.error(a.start, `pass a JSON prop to the Svelte island <${node.name}>; directives belong inside its .island.svelte file`);
        continue;
      }
      if (!a.name.startsWith('request:')) {
        plain.add(a.name);
        continue;
      }
      this.quiet.add(a.start);
      const name = a.name.slice('request:'.length);
      if (a.value === true || Array.isArray(a.value)) {
        this.error(a.start, `${a.name} needs one source expression, as in ${a.name}={search("products", options)}`);
        continue;
      }
      bound.push({ name, at: a.start });
      island.request = true;
    }
    for (const { name, at } of bound) {
      if (plain.has(name)) this.error(at, `prop ${name} already gets its value from request:${name}`);
    }
  }
}

// exportsName tells whether an export declaration exports a name.
function exportsName(node, name) {
  const d = node.declaration;
  if (d?.type === 'FunctionDeclaration') return d.id?.name === name;
  if (d?.type === 'VariableDeclaration') return d.declarations.some((v) => v.id.type === 'Identifier' && v.id.name === name);
  return node.specifiers.some((s) => (s.exported.name ?? s.exported.value) === name);
}

// walk calls visit for every node of a tree of JavaScript.
function walk(node, visit) {
  if (node === null || typeof node !== 'object') return;
  if (Array.isArray(node)) {
    for (const item of node) walk(item, visit);
    return;
  }
  if (typeof node.type === 'string') visit(node);
  for (const key in node) {
    if (key !== 'metadata' && key !== 'loc') walk(node[key], visit);
  }
}

// The requests are answered once all of the above is defined.
const reply = (message) => process.stdout.write(JSON.stringify(message) + '\n');
reply({ ready: true });
for await (const line of createInterface({ input: process.stdin })) {
  const { path, source, module, island, lower: program } = JSON.parse(line);
  if (program) reply(lower(source));
  else if (island) reply(compileIsland(path, source, module));
  else reply(module ? compileModule(path, source) : compileTemplate(path, source));
}
