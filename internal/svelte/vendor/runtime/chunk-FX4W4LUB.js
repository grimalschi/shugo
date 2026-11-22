import {
  attr,
  clsx,
  escape_html,
  to_class,
  to_style,
  validate_store
} from "./chunk-SZUTS3VD.js";
import {
  REGEX_VALID_TAG_NAME,
  hash,
  is_boolean_attribute,
  is_raw_text_element,
  is_void
} from "./chunk-6ZQVU4YW.js";
import {
  subscribe_to_store
} from "./chunk-UC3IQMGR.js";
import {
  FILENAME,
  STALE_REACTION,
  UNINITIALIZED,
  async_mode_flag,
  create_context,
  deferred,
  get_or_init_context_map,
  get_stack,
  has_own_property,
  invalid_snippet_arguments,
  is_array,
  is_promise,
  noop,
  push_array,
  set_context_after_init
} from "./chunk-GFOPKZ2A.js";
import {
  false_default
} from "./chunk-VCLKK7XB.js";

// node_modules/svelte/src/internal/server/hydration.js
var BLOCK_OPEN = `<!--${"["}-->`, BLOCK_OPEN_ELSE = `<!--${"[!"}-->`, BLOCK_CLOSE = `<!--${"]"}-->`, EMPTY_COMMENT = "<!---->";

// node_modules/svelte/src/internal/server/errors.js
function async_local_storage_unavailable() {
  let error = new Error("async_local_storage_unavailable\nThe node API `AsyncLocalStorage` is not available, but is required to use async server rendering.\nhttps://svelte.dev/e/async_local_storage_unavailable");
  throw error.name = "Svelte error", error;
}
function await_invalid() {
  let error = new Error(`await_invalid
Encountered asynchronous work while rendering synchronously.
https://svelte.dev/e/await_invalid`);
  throw error.name = "Svelte error", error;
}
function dynamic_element_invalid_tag(tag) {
  let error = new Error(`dynamic_element_invalid_tag
\`<svelte:element this="${tag}">\` is not a valid element name — the element will not be rendered
https://svelte.dev/e/dynamic_element_invalid_tag`);
  throw error.name = "Svelte error", error;
}
function html_deprecated() {
  let error = new Error("html_deprecated\nThe `html` property of server render results has been deprecated. Use `body` instead.\nhttps://svelte.dev/e/html_deprecated");
  throw error.name = "Svelte error", error;
}
function hydratable_clobbering(key, stack) {
  let error = new Error(`hydratable_clobbering
Attempted to set \`hydratable\` with key \`${key}\` twice with different values.

${stack}
https://svelte.dev/e/hydratable_clobbering`);
  throw error.name = "Svelte error", error;
}
function hydratable_serialization_failed(key, stack) {
  let error = new Error(`hydratable_serialization_failed
Failed to serialize \`hydratable\` data for key \`${key}\`.

\`hydratable\` can serialize anything [\`uneval\` from \`devalue\`](https://npmjs.com/package/devalue) can, plus Promises.

Cause:
${stack}
https://svelte.dev/e/hydratable_serialization_failed`);
  throw error.name = "Svelte error", error;
}
function invalid_csp() {
  let error = new Error("invalid_csp\n`csp.nonce` was set while `csp.hash` was `true`. These options cannot be used simultaneously.\nhttps://svelte.dev/e/invalid_csp");
  throw error.name = "Svelte error", error;
}
function invalid_id_prefix() {
  let error = new Error("invalid_id_prefix\nThe `idPrefix` option cannot include `--`.\nhttps://svelte.dev/e/invalid_id_prefix");
  throw error.name = "Svelte error", error;
}
function lifecycle_function_unavailable(name) {
  let error = new Error(`lifecycle_function_unavailable
\`${name}(...)\` is not available on the server
https://svelte.dev/e/lifecycle_function_unavailable`);
  throw error.name = "Svelte error", error;
}
function server_context_required() {
  let error = new Error("server_context_required\nCould not resolve `render` context.\nhttps://svelte.dev/e/server_context_required");
  throw error.name = "Svelte error", error;
}

// node_modules/svelte/src/internal/server/context.js
var ssr_context = null;
function set_ssr_context(v) {
  ssr_context = v;
}
function createContext() {
  return (
    /** @type {[() => T, (context: T) => T, () => boolean]} */
    create_context(getContext, setContext, hasContext)
  );
}
function getContext(key) {
  return (
    /** @type {T} */
    get_or_init_context_map(ssr_context, "getContext").get(key)
  );
}
function setContext(key, context2) {
  let context_map = get_or_init_context_map(ssr_context, "setContext");
  return (
    /** @type {SSRContext} */
    ssr_context.i && set_context_after_init(), context_map.set(key, context2), context2
  );
}
function hasContext(key) {
  return get_or_init_context_map(ssr_context, "hasContext").has(key);
}
function getAllContexts() {
  return get_or_init_context_map(ssr_context, "getAllContexts");
}
function push(fn) {
  ssr_context = { p: ssr_context, c: null, r: null, i: !1 }, false_default && (ssr_context.function = fn, ssr_context.element = ssr_context.p?.element);
}
function pop() {
  ssr_context = /** @type {SSRContext} */
  ssr_context.p;
}
async function save(promise) {
  var previous_context = ssr_context, value = await promise;
  return () => (ssr_context = previous_context, value);
}

// node_modules/svelte/src/internal/server/warnings.js
var bold = "font-weight: bold", normal = "font-weight: normal";
function unresolved_hydratable(key, stack) {
  false_default ? console.warn(
    `%c[svelte] unresolved_hydratable
%cA \`hydratable\` value with key \`${key}\` was created, but at least part of it was not used during the render.

The \`hydratable\` was initialized in:
${stack}
https://svelte.dev/e/unresolved_hydratable`,
    bold,
    normal
  ) : console.warn("https://svelte.dev/e/unresolved_hydratable");
}

// node_modules/svelte/src/internal/server/render-context.js
var current_render = null, context = null;
function get_render_context() {
  let store = context ?? als?.getStore();
  return store || server_context_required(), store;
}
async function with_render_context(fn) {
  if (context = {
    hydratable: {
      lookup: /* @__PURE__ */ new Map(),
      comparisons: [],
      unresolved_promises: /* @__PURE__ */ new Map()
    }
  }, in_webcontainer()) {
    let { promise, resolve } = deferred(), previous_render = current_render;
    return current_render = promise, await previous_render, fn().finally(resolve);
  }
  try {
    return als === null && async_local_storage_unavailable(), als.run(context, fn);
  } finally {
    context = null;
  }
}
var als = null, als_import = null;
function init_render_context() {
  return als_import ??= import("node:async_hooks").then((hooks) => {
    als = new hooks.AsyncLocalStorage();
  }).then(noop, noop), als_import;
}
function in_webcontainer() {
  return !!globalThis.process?.versions?.webcontainer;
}

// node_modules/svelte/src/internal/server/crypto.js
var text_encoder, crypto, obfuscated_import = (module_name) => import(
  /* @vite-ignore */
  module_name
);
async function sha256(data) {
  text_encoder ??= new TextEncoder(), crypto ??= globalThis.crypto?.subtle?.digest ? globalThis.crypto : (
    // @ts-ignore - we don't install node types in the prod build
    // don't use import('node:crypto') directly because static analysers will think we rely on node when we don't
    (await obfuscated_import("node:crypto")).webcrypto
  );
  let hash_buffer = await crypto.subtle.digest("SHA-256", text_encoder.encode(data));
  return base64_encode(hash_buffer);
}
function base64_encode(bytes) {
  if (!false_default && globalThis.Buffer)
    return globalThis.Buffer.from(bytes).toString("base64");
  let binary = "";
  for (let i = 0; i < bytes.length; i++)
    binary += String.fromCharCode(bytes[i]);
  return btoa(binary);
}

// node_modules/devalue/src/utils.js
var escaped = {
  "<": "\\u003C",
  "\\": "\\\\",
  "\b": "\\b",
  "\f": "\\f",
  "\n": "\\n",
  "\r": "\\r",
  "	": "\\t",
  "\u2028": "\\u2028",
  "\u2029": "\\u2029"
}, DevalueError = class extends Error {
  /**
   * @param {string} message
   * @param {string[]} keys
   * @param {any} [value] - The value that failed to be serialized
   * @param {any} [root] - The root value being serialized
   */
  constructor(message, keys, value, root) {
    super(message), this.name = "DevalueError", this.path = keys.join(""), this.value = value, this.root = root;
  }
};
function is_primitive(thing) {
  return thing === null || typeof thing != "object" && typeof thing != "function";
}
var object_proto_names = /* @__PURE__ */ Object.getOwnPropertyNames(Object.prototype).sort().join("\0");
function is_plain_object(thing) {
  let proto = Object.getPrototypeOf(thing);
  return proto === Object.prototype || proto === null || Object.getPrototypeOf(proto) === null || Object.getOwnPropertyNames(proto).sort().join("\0") === object_proto_names;
}
function get_type(thing) {
  return Object.prototype.toString.call(thing).slice(8, -1);
}
function is_buffer(thing) {
  return typeof Buffer < "u" && Buffer.isBuffer(thing);
}
function get_escaped_char(char) {
  switch (char) {
    case '"':
      return '\\"';
    case "<":
      return "\\u003C";
    case "\\":
      return "\\\\";
    case `
`:
      return "\\n";
    case "\r":
      return "\\r";
    case "	":
      return "\\t";
    case "\b":
      return "\\b";
    case "\f":
      return "\\f";
    case "\u2028":
      return "\\u2028";
    case "\u2029":
      return "\\u2029";
    default:
      return char < " " ? `\\u${char.charCodeAt(0).toString(16).padStart(4, "0")}` : "";
  }
}
function stringify_string(str) {
  let result = "", last_pos = 0, len = str.length;
  for (let i = 0; i < len; i += 1) {
    let char = str[i], replacement = get_escaped_char(char);
    replacement && (result += str.slice(last_pos, i) + replacement, last_pos = i + 1);
  }
  return `"${last_pos === 0 ? str : result + str.slice(last_pos)}"`;
}
function enumerable_symbols(object) {
  return Object.getOwnPropertySymbols(object).filter(
    (symbol) => Object.getOwnPropertyDescriptor(object, symbol).enumerable
  );
}
var is_identifier = /^[a-zA-Z_$][a-zA-Z_$0-9]*$/;
function stringify_key(key) {
  return is_identifier.test(key) ? "." + key : "[" + JSON.stringify(key) + "]";
}
function is_valid_array_index(n) {
  return !(!Number.isInteger(n) || n < 0 || n > 4294967294);
}
function is_valid_array_index_string(s) {
  if (s.length === 0 || s.length > 1 && s.charCodeAt(0) === 48) return !1;
  for (let i = 0; i < s.length; i++) {
    let c = s.charCodeAt(i);
    if (c < 48 || c > 57) return !1;
  }
  return is_valid_array_index(+s);
}
function array_index_cut(keys) {
  for (var i = keys.length - 1; i >= 0 && !is_valid_array_index_string(keys[i]); i--)
    ;
  return i + 1;
}
function valid_array_indices(array) {
  let keys = Object.keys(array);
  return keys.length = array_index_cut(keys), keys;
}

// node_modules/devalue/src/uneval.js
var chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_$", MIN_STRING_LENGTH = 128, unsafe_chars = /[<\b\f\n\r\t\0\u2028\u2029]/g, reserved = /^(?:do|if|in|for|int|let|new|try|var|byte|case|char|else|enum|goto|long|this|void|with|await|break|catch|class|const|final|float|short|super|throw|while|yield|delete|double|export|import|native|return|switch|throws|typeof|boolean|default|extends|finally|package|private|abstract|continue|debugger|function|volatile|interface|protected|transient|implements|instanceof|synchronized)$/;
function uneval(value, replacer) {
  let counts = /* @__PURE__ */ new Map(), keys = [], custom = /* @__PURE__ */ new Map(), primitives;
  function stringify_cached_primitive(thing) {
    if (typeof thing == "string" && thing.length >= MIN_STRING_LENGTH || typeof thing == "bigint") {
      primitives ??= /* @__PURE__ */ new Map();
      let literal = primitives.get(thing);
      return literal === void 0 && (literal = stringify_primitive(thing), primitives.set(thing, literal)), literal;
    }
    return stringify_primitive(thing);
  }
  function walk(thing) {
    if (is_primitive(thing)) {
      if (typeof thing == "symbol")
        throw new DevalueError("Cannot stringify a Symbol primitive", keys, thing, value);
      (typeof thing == "string" && thing.length >= MIN_STRING_LENGTH || typeof thing == "bigint") && counts.set(thing, (counts.get(thing) || 0) + 1);
    } else {
      if (counts.has(thing)) {
        counts.set(thing, counts.get(thing) + 1);
        return;
      }
      if (counts.set(thing, 1), replacer) {
        let str2 = replacer(thing, (value2) => uneval(value2, replacer));
        if (typeof str2 == "string") {
          custom.set(thing, str2);
          return;
        }
      }
      if (typeof thing == "function")
        throw new DevalueError("Cannot stringify a function", keys, thing, value);
      switch (get_type(thing)) {
        case "BigInt":
        case "String":
          walk(thing.valueOf());
          return;
        case "Number":
        case "Boolean":
        case "Date":
        case "RegExp":
        case "URL":
        case "URLSearchParams":
          return;
        case "Array":
          for (let i of valid_array_indices(thing))
            keys.push(`[${i}]`), walk(thing[i]), keys.pop();
          break;
        case "Set":
          Array.from(thing).forEach(walk);
          break;
        case "Map":
          for (let [key, value2] of thing)
            keys.push(`.get(${is_primitive(key) ? stringify_cached_primitive(key) : "..."})`), walk(key), walk(value2), keys.pop();
          break;
        case "Int8Array":
        case "Uint8Array":
        case "Uint8ClampedArray":
        case "Int16Array":
        case "Uint16Array":
        case "Float16Array":
        case "Int32Array":
        case "Uint32Array":
        case "Float32Array":
        case "Float64Array":
        case "BigInt64Array":
        case "BigUint64Array":
        case "DataView":
          is_buffer(thing) || walk(thing.buffer);
          return;
        case "ArrayBuffer":
          return;
        case "Temporal.Duration":
        case "Temporal.Instant":
        case "Temporal.PlainDate":
        case "Temporal.PlainTime":
        case "Temporal.PlainDateTime":
        case "Temporal.PlainMonthDay":
        case "Temporal.PlainYearMonth":
        case "Temporal.ZonedDateTime":
          return;
        default:
          if (!is_plain_object(thing))
            throw new DevalueError("Cannot stringify arbitrary non-POJOs", keys, thing, value);
          if (enumerable_symbols(thing).length > 0)
            throw new DevalueError("Cannot stringify POJOs with symbolic keys", keys, thing, value);
          for (let key of Object.keys(thing)) {
            if (key === "__proto__")
              throw new DevalueError(
                "Cannot stringify objects with __proto__ keys",
                keys,
                thing,
                value
              );
            keys.push(stringify_key(key)), walk(thing[key]), keys.pop();
          }
      }
    }
  }
  walk(value);
  let names = /* @__PURE__ */ new Map(), repeated = [];
  counts.forEach((count, thing) => {
    count > 1 && repeated.push([thing, count]);
  }), repeated.sort((a, b) => b[1] - a[1]).forEach(([thing, count]) => {
    let name = get_name(names.size);
    if (is_primitive(thing)) {
      let length = stringify_cached_primitive(thing).length;
      if (length * count <= length + (count + 1) * name.length + 25) return;
    }
    names.set(thing, name);
  });
  function stringify2(thing) {
    if (names.has(thing))
      return names.get(thing);
    if (is_primitive(thing))
      return stringify_primitive(thing);
    if (custom.has(thing))
      return custom.get(thing);
    let type = get_type(thing);
    switch (type) {
      case "Number":
      case "String":
      case "Boolean":
      case "BigInt":
        return `Object(${stringify2(thing.valueOf())})`;
      case "RegExp":
        let { source, flags } = thing;
        return flags ? `new RegExp(${stringify_string(source)},"${flags}")` : `new RegExp(${stringify_string(source)})`;
      case "Date":
        return `new Date(${thing.getTime()})`;
      case "URL":
        return `new URL(${stringify_string(thing.toString())})`;
      case "URLSearchParams":
        return `new URLSearchParams(${stringify_string(thing.toString())})`;
      case "Array": {
        let has_holes = !1, result = "[";
        for (let i = 0; i < thing.length; i += 1)
          if (i > 0 && (result += ","), Object.hasOwn(thing, i))
            result += stringify2(thing[i]);
          else if (!has_holes) {
            let populated_keys = valid_array_indices(thing), population = populated_keys.length, d = String(thing.length).length, array = stringify_sparse_array(thing.length), hole_cost = thing.length + 2, sparse_cost = array.length + 18 + population * (d + 2);
            if (hole_cost > sparse_cost) {
              let entries = populated_keys.map((k) => `${k}:${stringify2(thing[k])}`).join(",");
              return `Object.assign(${array},{${entries}})`;
            }
            has_holes = !0;
          }
        let tail = thing.length === 0 || Object.hasOwn(thing, thing.length - 1) ? "" : ",";
        return result + tail + "]";
      }
      case "Set":
      case "Map":
        return `new ${type}([${Array.from(thing).map(stringify2).join(",")}])`;
      case "Int8Array":
      case "Uint8Array":
      case "Uint8ClampedArray":
      case "Int16Array":
      case "Uint16Array":
      case "Float16Array":
      case "Int32Array":
      case "Uint32Array":
      case "Float32Array":
      case "Float64Array":
      case "BigInt64Array":
      case "BigUint64Array": {
        is_buffer(thing) && (thing = new Uint8Array(thing));
        let str2 = `new ${type}`;
        if (names.has(thing.buffer) ? str2 += `(${stringify2(thing.buffer)})` : str2 += `([${stringify_typed_array_elements(type, thing.buffer)}])`, thing.byteLength !== thing.buffer.byteLength) {
          let start = thing.byteOffset / thing.BYTES_PER_ELEMENT, end = start + thing.length;
          str2 += `.subarray(${start},${end})`;
        }
        return str2;
      }
      case "DataView": {
        let str2 = "new DataView";
        return names.has(thing.buffer) ? str2 += `(${stringify2(thing.buffer)}` : str2 += `(new Uint8Array([${new Uint8Array(thing.buffer)}]).buffer`, thing.byteLength !== thing.buffer.byteLength && (str2 += `,${thing.byteOffset},${thing.byteLength}`), str2 + ")";
      }
      case "ArrayBuffer":
        return `new Uint8Array([${new Uint8Array(thing).toString()}]).buffer`;
      case "Temporal.Duration":
      case "Temporal.Instant":
      case "Temporal.PlainDate":
      case "Temporal.PlainTime":
      case "Temporal.PlainDateTime":
      case "Temporal.PlainMonthDay":
      case "Temporal.PlainYearMonth":
      case "Temporal.ZonedDateTime":
        return `${type}.from(${stringify_string(thing.toString())})`;
      default:
        let keys2 = Object.keys(thing), obj = keys2.map((key) => `${safe_key(key)}:${stringify2(thing[key])}`).join(",");
        return Object.getPrototypeOf(thing) === null ? keys2.length > 0 ? `{${obj},__proto__:null}` : "{__proto__:null}" : `{${obj}}`;
    }
  }
  let str = stringify2(value);
  if (names.size) {
    let params = [], statements = [], values = [], reconstructions = [];
    names.forEach((name, thing) => {
      if (params.push(name), custom.has(thing)) {
        values.push(
          /** @type {string} */
          custom.get(thing)
        );
        return;
      }
      if (is_primitive(thing)) {
        values.push(stringify_cached_primitive(thing));
        return;
      }
      let type = get_type(thing);
      switch (type) {
        case "Number":
        case "String":
        case "Boolean":
        case "BigInt": {
          let primitive = thing.valueOf();
          names.has(primitive) ? (values.push("{}"), reconstructions.push(`${name}=Object(${stringify2(primitive)})`)) : values.push(`Object(${stringify2(primitive)})`);
          break;
        }
        case "RegExp":
          let { source, flags } = thing, regexp = flags ? `new RegExp(${stringify_string(source)},"${flags}")` : `new RegExp(${stringify_string(source)})`;
          values.push(regexp);
          break;
        case "Date":
          values.push(`new Date(${thing.getTime()})`);
          break;
        case "URL":
          values.push(`new URL(${stringify_string(thing.toString())})`);
          break;
        case "URLSearchParams":
          values.push(`new URLSearchParams(${stringify_string(thing.toString())})`);
          break;
        case "Array": {
          let populated_keys = valid_array_indices(thing);
          values.push(
            thing.length > 32 + 2 * populated_keys.length ? stringify_sparse_array(thing.length) : `Array(${thing.length})`
          );
          for (let i of populated_keys)
            statements.push(`${name}[${i}]=${stringify2(thing[i])}`);
          break;
        }
        case "Set": {
          values.push("new Set");
          let adds = Array.from(thing).map((v) => `.add(${stringify2(v)})`);
          adds.length > 0 && statements.push(name + adds.join(""));
          break;
        }
        case "Map": {
          values.push("new Map");
          let sets = Array.from(thing).map(
            ([k, v]) => `.set(${stringify2(k)}, ${stringify2(v)})`
          );
          sets.length > 0 && statements.push(name + sets.join(""));
          break;
        }
        case "Int8Array":
        case "Uint8Array":
        case "Uint8ClampedArray":
        case "Int16Array":
        case "Uint16Array":
        case "Float16Array":
        case "Int32Array":
        case "Uint32Array":
        case "Float32Array":
        case "Float64Array":
        case "BigInt64Array":
        case "BigUint64Array": {
          is_buffer(thing) && (thing = new Uint8Array(thing));
          let str2 = `new ${type}`;
          if (names.has(thing.buffer) ? str2 += `(${stringify2(thing.buffer)})` : str2 += `([${stringify_typed_array_elements(type, thing.buffer)}])`, thing.byteLength !== thing.buffer.byteLength) {
            let start = thing.byteOffset / thing.BYTES_PER_ELEMENT, end = start + thing.length;
            str2 += `.subarray(${start},${end})`;
          }
          values.push("{}"), reconstructions.push(`${name}=${str2}`);
          break;
        }
        case "DataView": {
          let str2 = "new DataView";
          names.has(thing.buffer) ? str2 += `(${stringify2(thing.buffer)}` : str2 += `(new Uint8Array([${new Uint8Array(thing.buffer)}]).buffer`, thing.byteLength !== thing.buffer.byteLength && (str2 += `,${thing.byteOffset},${thing.byteLength}`), str2 += ")", values.push("{}"), reconstructions.push(`${name}=${str2}`);
          break;
        }
        case "ArrayBuffer":
          values.push(`new Uint8Array([${new Uint8Array(thing)}]).buffer`);
          break;
        case "Temporal.Duration":
        case "Temporal.Instant":
        case "Temporal.PlainDate":
        case "Temporal.PlainTime":
        case "Temporal.PlainDateTime":
        case "Temporal.PlainMonthDay":
        case "Temporal.PlainYearMonth":
        case "Temporal.ZonedDateTime":
          values.push(`${type}.from(${stringify_string(thing.toString())})`);
          break;
        default:
          values.push(Object.getPrototypeOf(thing) === null ? "Object.create(null)" : "{}"), Object.keys(thing).forEach((key) => {
            statements.push(`${name}${safe_prop(key)}=${stringify2(thing[key])}`);
          });
      }
    }), statements.push(`return ${str}`);
    let body = [...reconstructions, ...statements].join(";");
    return params.length > 65534 ? `(function(){var[${params.join(",")}]=arguments[0];${body}}([${values.join(",")}]))` : `(function(${params.join(",")}){${body}}(${values.join(",")}))`;
  } else
    return str;
}
function stringify_sparse_array(length) {
  return `(function(a){a[${4294967294}]=0;delete a[${4294967294}];a.length=${length};return a}([]))`;
}
function stringify_typed_array_elements(type, buffer) {
  let array = new /** @type {any} */
  globalThis[type](buffer);
  return type === "BigInt64Array" || type === "BigUint64Array" ? Array.from(array, (element2) => `${element2}n`).join(",") : array instanceof Float32Array || array instanceof Float64Array || typeof Float16Array < "u" && array instanceof Float16Array ? Array.from(array, (element2) => Object.is(element2, -0) ? "-0" : `${element2}`).join(",") : array.toString();
}
function get_name(num) {
  let name = "";
  do
    name = chars[num % chars.length] + name, num = ~~(num / chars.length) - 1;
  while (num >= 0);
  return reserved.test(name) ? `${name}0` : name;
}
function escape_unsafe_char(c) {
  return escaped[c] || c;
}
function escape_unsafe_chars(str) {
  return str.replace(unsafe_chars, escape_unsafe_char);
}
function safe_key(key) {
  return /^[_$a-zA-Z][_$a-zA-Z0-9]*$/.test(key) ? key : escape_unsafe_chars(JSON.stringify(key));
}
function safe_prop(key) {
  return /^[_$a-zA-Z][_$a-zA-Z0-9]*$/.test(key) ? `.${key}` : `[${escape_unsafe_chars(JSON.stringify(key))}]`;
}
function stringify_primitive(thing) {
  let type = typeof thing;
  if (type === "string") return stringify_string(thing);
  if (thing === void 0) return "void 0";
  if (thing === 0 && 1 / thing < 0) return "-0";
  let str = String(thing);
  return type === "number" ? str.replace(/^(-)?0\./, "$1.") : type === "bigint" ? thing + "n" : str;
}

// node_modules/svelte/src/internal/server/renderer.js
var RenderResult = class {
  /** @type {() => AccumulatedContent} */
  #render;
  /** @type {() => Promise<AccumulatedContent & { hashes: { script: Sha256Source[] } }>} */
  #render_async;
  /** @type {AccumulatedContent | undefined} */
  #sync;
  /** @type {{ script: '' }} */
  #hashes = { script: "" };
  /** @type {Promise<AccumulatedContent & { hashes: { script: Sha256Source[] } }> | undefined} */
  #promise;
  /**
   * @param {() => AccumulatedContent} render
   * @param {() => Promise<AccumulatedContent & { hashes: { script: Sha256Source[] } }>} render_async
   */
  constructor(render2, render_async) {
    this.#render = render2, this.#render_async = render_async;
  }
  #get() {
    return this.#sync ??= this.#render();
  }
  get html() {
    return this.#get().body;
  }
  get head() {
    return this.#get().head;
  }
  get body() {
    return this.#get().body;
  }
  get hashes() {
    return this.#hashes;
  }
  /**
   * This is not type-safe, but honestly it's the best I can do right now, and it's a straightforward function.
   *
   * @template TResult1
   * @template [TResult2=never]
   * @param {(value: SyncRenderOutput) => TResult1} onfulfilled
   * @param {(reason: unknown) => TResult2} onrejected
   */
  then(onfulfilled, onrejected) {
    if (!async_mode_flag) {
      let result = this.#get(), user_result = onfulfilled({
        head: result.head,
        body: result.body,
        html: result.body,
        hashes: { script: [] }
      });
      return Promise.resolve(user_result);
    }
    return this.#promise ??= this.#render_async().then((result) => (Object.defineProperty(result, "html", {
      // eslint-disable-next-line getter-return
      get: () => {
        html_deprecated();
      }
    }), result)), this.#promise.then(
      (result) => onfulfilled(
        /** @type {SyncRenderOutput} */
        result
      ),
      onrejected
    );
  }
}, Renderer = class _Renderer {
  /**
   * The contents of the renderer.
   * @type {RendererItem[]}
   */
  #out = [];
  /**
   * Any `onDestroy` callbacks registered during execution of this renderer.
   * @type {(() => void)[] | undefined}
   */
  #on_destroy = void 0;
  /**
   * Whether this renderer is a component body.
   * @type {boolean}
   */
  #is_component_body = !1;
  /** @type {boolean} */
  #is_title = !1;
  /**
   * If set, this renderer is an error boundary. When async collection
   * of the children fails, the failed snippet is rendered instead.
   * @type {{
   * 	failed: (renderer: Renderer, error: unknown, reset: () => void) => void;
   * 	transformError: (error: unknown) => unknown;
   * 	context: SSRContext | null;
   * } | null}
   */
  #boundary = null;
  /**
   * The type of string content that this renderer is accumulating.
   * @type {RendererType}
   */
  type;
  /** @type {Renderer | undefined} */
  #parent;
  /**
   * Asynchronous work associated with this renderer
   * @type {Promise<void> | undefined}
   */
  promise = void 0;
  /**
   * State which is associated with the content tree as a whole.
   * It will be re-exposed, uncopied, on all children.
   * @type {SSRState}
   * @readonly
   */
  global;
  /**
   * State that is local to the branch it is declared in.
   * It will be shallow-copied to all children.
   *
   * @type {{ select_value: any, multiple: boolean }}
   */
  local;
  /**
   * @param {SSRState} global
   * @param {Renderer | undefined} [parent]
   */
  constructor(global, parent) {
    this.#parent = parent, this.global = global, this.local = parent ? { ...parent.local } : { select_value: void 0, multiple: !1 }, this.type = parent ? parent.type : "body";
  }
  /**
   * @param {(renderer: Renderer) => void} fn
   */
  head(fn) {
    let head2 = new _Renderer(this.global, this);
    head2.type = "head", this.#out.push(head2), head2.child(fn);
  }
  /**
   * @param {Array<Promise<void>>} blockers
   * @param {(renderer: Renderer) => void} fn
   */
  async_block(blockers, fn) {
    this.#out.push(BLOCK_OPEN), this.async(blockers, fn), this.#out.push(BLOCK_CLOSE);
  }
  /**
   * @param {Array<Promise<void>>} blockers
   * @param {(renderer: Renderer) => void} fn
   */
  async(blockers, fn) {
    let callback = fn;
    if (blockers.length > 0) {
      let context2 = ssr_context;
      callback = (renderer) => Promise.all(blockers).then(() => {
        let previous_context = ssr_context;
        try {
          return set_ssr_context(context2), fn(renderer);
        } finally {
          set_ssr_context(previous_context);
        }
      });
    }
    this.child(callback);
  }
  /**
   * @param {Array<() => void>} thunks
   */
  run(thunks) {
    let context2 = ssr_context, promise = Promise.resolve(thunks[0]()), promises = [promise];
    context2 !== null && thunks.length > 1 && (context2.i = !0);
    for (let fn of thunks.slice(1))
      promise = promise.then(() => {
        let previous_context = ssr_context;
        set_ssr_context(context2);
        try {
          return fn();
        } finally {
          set_ssr_context(previous_context);
        }
      }), promises.push(promise);
    return promise.catch(noop), this.promise = this.global.track(promise), promises;
  }
  /**
   * @param {(renderer: Renderer) => MaybePromise<void>} fn
   */
  child_block(fn) {
    this.#out.push(BLOCK_OPEN), this.child(fn), this.#out.push(BLOCK_CLOSE);
  }
  /**
   * Create a child renderer. The child renderer inherits the state from the parent,
   * but has its own content.
   * @param {(renderer: Renderer) => MaybePromise<void>} fn
   */
  child(fn) {
    let child = new _Renderer(this.global, this);
    this.#out.push(child);
    let parent = ssr_context;
    set_ssr_context({
      ...ssr_context,
      p: parent,
      c: null,
      r: child,
      i: ssr_context?.i ?? !1
    });
    let result = fn(child);
    return set_ssr_context(parent), result instanceof Promise && (result.catch(noop), result.finally(() => set_ssr_context(null)).catch(noop), child.global.mode === "sync" && await_invalid(), child.promise = child.global.track(result)), child;
  }
  /**
   * Render children inside an error boundary. If the children throw and the API-level
   * `transformError` transform handles the error (doesn't re-throw), the `failed` snippet is
   * rendered instead. Otherwise the error propagates.
   *
   * @param {{ failed?: (renderer: Renderer, error: unknown, reset: () => void) => void }} props
   * @param {(renderer: Renderer) => MaybePromise<void>} children_fn
   */
  boundary(props, children_fn) {
    let child = new _Renderer(this.global, this);
    this.#out.push(child);
    let parent_context = ssr_context;
    props.failed && (child.#boundary = {
      failed: props.failed,
      transformError: this.global.transformError,
      context: parent_context
    }), set_ssr_context({
      ...ssr_context,
      p: parent_context,
      c: null,
      r: child,
      i: ssr_context?.i ?? !1
    });
    try {
      let result = children_fn(child);
      set_ssr_context(parent_context), result instanceof Promise && (child.global.mode === "sync" && await_invalid(), result.catch(noop), child.promise = child.global.track(result));
    } catch (error) {
      set_ssr_context(parent_context);
      let failed_snippet = props.failed;
      if (!failed_snippet) throw error;
      let result = this.global.transformError(error);
      child.#out.length = 0, child.#boundary = null, result instanceof Promise ? (this.global.mode === "sync" && await_invalid(), child.promise = child.global.track(
        /** @type {Promise<unknown>} */
        result.then((transformed) => {
          set_ssr_context(parent_context), child.#out.push(_Renderer.#serialize_failed_boundary(transformed)), failed_snippet(child, transformed, noop), child.#out.push(BLOCK_CLOSE);
        })
      ), child.promise.catch(noop)) : (child.#out.push(_Renderer.#serialize_failed_boundary(result)), failed_snippet(child, result, noop), child.#out.push(BLOCK_CLOSE));
    }
  }
  /**
   * Create a component renderer. The component renderer inherits the state from the parent,
   * but has its own content. It is treated as an ordering boundary for ondestroy callbacks.
   * @param {(renderer: Renderer) => MaybePromise<void>} fn
   * @param {Function} [component_fn]
   * @returns {void}
   */
  component(fn, component_fn) {
    push(component_fn), this.child((renderer) => (renderer.#is_component_body = !0, fn(renderer))), pop();
  }
  /**
   * @param {Record<string, any>} attrs
   * @param {(renderer: Renderer) => void} fn
   * @param {string | undefined} [css_hash]
   * @param {Record<string, boolean> | undefined} [classes]
   * @param {Record<string, string> | undefined} [styles]
   * @param {number | undefined} [flags]
   * @param {boolean | undefined} [is_rich]
   * @returns {void}
   */
  select(attrs, fn, css_hash, classes, styles, flags, is_rich) {
    let { value, defaultValue, ...select_attrs } = attrs;
    select_attrs.multiple === "" && (select_attrs.multiple = !0), this.push(`<select${attributes(select_attrs, css_hash, classes, styles, flags)}>`), this.child((renderer) => {
      renderer.local.select_value = value === void 0 ? defaultValue : value, renderer.local.multiple = !!select_attrs.multiple, fn(renderer);
    }), this.push(`${is_rich ? "<!>" : ""}</select>`);
  }
  /**
   * @param {Record<string, any>} attrs
   * @param {string | number | boolean | ((renderer: Renderer) => void)} body
   * @param {string | undefined} [css_hash]
   * @param {Record<string, boolean> | undefined} [classes]
   * @param {Record<string, string> | undefined} [styles]
   * @param {number | undefined} [flags]
   * @param {boolean | undefined} [is_rich]
   */
  option(attrs, body, css_hash, classes, styles, flags, is_rich) {
    this.#out.push(`<option${attributes(attrs, css_hash, classes, styles, flags)}`);
    let close = (renderer, value, { head: head2, body: body2 }) => {
      has_own_property.call(attrs, "value") && (value = attrs.value);
      var select_value = this.local.select_value;
      // Super edge-case, but theoretically someone could use arrays with non-multiple selects,
      // so we gotta check for the multiple attribute presence, too.
      (this.local.multiple && is_array(select_value) ? select_value.includes(value) : value === select_value) && renderer.#out.push(' selected=""'), renderer.#out.push(`>${body2}${is_rich ? "<!>" : ""}</option>`), head2 && renderer.head((child) => child.push(head2));
    };
    typeof body == "function" ? this.child((renderer) => {
      let r = new _Renderer(this.global, this);
      if (body(r), this.global.mode === "async")
        return r.#collect_content_async().then((content) => {
          close(renderer, content.body.replaceAll("<!---->", ""), content);
        });
      {
        let content = r.#collect_content();
        close(renderer, content.body.replaceAll("<!---->", ""), content);
      }
    }) : close(this, body, { body: escape_html(body) });
  }
  /**
   * @param {(renderer: Renderer) => void} fn
   */
  title(fn) {
    let title = new _Renderer(this.global, this);
    title.type = "head", title.#is_title = !0, this.#out.push(title), title.child(fn);
  }
  /**
   * @param {string | (() => Promise<string>)} content
   */
  push(content) {
    typeof content == "function" ? this.child(async (renderer) => renderer.push(await content())) : this.#out.push(content);
  }
  /**
   * @param {() => void} fn
   */
  on_destroy(fn) {
    (this.#on_destroy ??= []).push(fn);
  }
  /**
   * @deprecated this is needed for legacy component bindings
   */
  copy() {
    let copy = new _Renderer(this.global, this.#parent);
    return copy.type = this.type, copy.#is_title = this.#is_title, copy.#out = this.#out.map((item) => item instanceof _Renderer ? item.copy() : item), copy.promise = this.promise, copy;
  }
  /**
   * @param {Renderer} other
   * @deprecated this is needed for legacy component bindings
   */
  subsume(other) {
    if (this.global.mode !== other.global.mode)
      throw new Error(
        "invariant: A renderer cannot switch modes. If you're seeing this, there's a compiler bug. File an issue!"
      );
    this.local = other.local, this.#out = other.#out.map((item, i) => {
      let current = this.#out[i];
      return current instanceof _Renderer && item instanceof _Renderer ? (current.subsume(item), current) : item;
    }), this.promise = other.promise, this.type = other.type, this.#is_title = other.#is_title;
  }
  get length() {
    return this.#out.length;
  }
  /**
   * Creates the hydration comment that marks the start of a failed boundary.
   * The error is JSON-serialized and embedded inside an HTML comment for the client
   * to parse during hydration. The JSON is escaped to prevent `-->` or `<!--` sequences
   * from breaking out of the comment (XSS). Uses unicode escapes which `JSON.parse()`
   * handles transparently.
   * @param {unknown} error
   * @returns {string}
   */
  static #serialize_failed_boundary(error) {
    var json = JSON.stringify(error), escaped2 = json.replace(/>/g, "\\u003e").replace(/</g, "\\u003c");
    return `<!--${"[?"}${escaped2}-->`;
  }
  /**
   * Only available on the server and when compiling with the `server` option.
   * Takes a component and returns an object with `body` and `head` properties on it, which you can use to populate the HTML when server-rendering your app.
   * @template {Record<string, any>} Props
   * @param {Component<Props>} component
   * @param {{ props?: Omit<Props, '$$slots' | '$$events'>; context?: Map<any, any>; idPrefix?: string; csp?: Csp }} [options]
   * @returns {RenderOutput}
   */
  static render(component, options = {}) {
    return (
      /** @type {RenderOutput} */
      /** @type {unknown} */
      new RenderResult(
        () => _Renderer.#render(component, options),
        () => init_render_context().then(
          () => with_render_context(() => _Renderer.#render_async(component, options))
        )
      )
    );
  }
  /**
   * runs onDestroy callbacks deepest component first, each with its child renderers in document order
   * only call once the content has been collected
   * @param {boolean} suppress_errors so callback errors on a failed render do not mask the render error
   */
  #run_on_destroy(suppress_errors) {
    let first_error, has_error = !1, visit = (renderer, callbacks) => {
      renderer.#is_component_body && (callbacks = []), renderer.#on_destroy && push_array(callbacks, renderer.#on_destroy);
      for (let child of renderer.#out)
        child instanceof _Renderer && visit(child, callbacks);
      if (renderer.#is_component_body)
        for (let cleanup of callbacks)
          try {
            cleanup();
          } catch (error) {
            !suppress_errors && !has_error && (first_error = error, has_error = !0);
          }
    };
    if (visit(this, []), has_error) throw first_error;
  }
  /**
   * @param {'sync' | 'async'} mode
   * @param {{ idPrefix?: string; csp?: Csp; transformError?: (error: unknown) => unknown }} options
   * @returns {Renderer}
   */
  static #create(mode, options) {
    return options.idPrefix?.includes("--") && invalid_id_prefix(), new _Renderer(
      new SSRState(
        mode,
        options.idPrefix ? options.idPrefix + "-" : "",
        options.csp,
        options.transformError
      )
    );
  }
  /**
   * Render a component. Throws if any of the children are performing asynchronous work.
   *
   * @template {Record<string, any>} Props
   * @param {Component<Props>} component
   * @param {{ props?: Omit<Props, '$$slots' | '$$events'>; context?: Map<any, any>; idPrefix?: string }} options
   * @returns {AccumulatedContent}
   */
  static #render(component, options) {
    var previous_context = ssr_context;
    let renderer = _Renderer.#create("sync", options), result, render_error, failed = !1;
    try {
      try {
        _Renderer.#open_render(renderer, component, options), result = _Renderer.#close_render(renderer.#collect_content(), renderer);
      } catch (error) {
        render_error = error, failed = !0;
      }
      if (renderer.#run_on_destroy(failed), failed) throw render_error;
      return (
        /** @type {AccumulatedContent} */
        result
      );
    } finally {
      renderer.global.abort(), set_ssr_context(previous_context);
    }
  }
  /**
   * Render a component.
   *
   * @template {Record<string, any>} Props
   * @param {Component<Props>} component
   * @param {{ props?: Omit<Props, '$$slots' | '$$events'>; context?: Map<any, any>; idPrefix?: string; csp?: Csp }} options
   * @returns {Promise<AccumulatedContent & { hashes: { script: Sha256Source[] } }>}
   */
  static async #render_async(component, options) {
    let previous_context = ssr_context, renderer = _Renderer.#create("async", options), result, render_error, failed = !1;
    try {
      try {
        _Renderer.#open_render(renderer, component, options);
        let content = await renderer.#collect_content_async(), hydratables = await renderer.#collect_hydratables();
        hydratables !== null && (content.head = hydratables + content.head), result = _Renderer.#close_render(content, renderer);
      } catch (error) {
        render_error = error, failed = !0, renderer.global.abort(), await renderer.global.settle();
      }
      if (renderer.#run_on_destroy(failed), failed) throw render_error;
      return (
        /** @type {AccumulatedContent & { hashes: { script: Sha256Source[] } }} */
        result
      );
    } finally {
      set_ssr_context(previous_context), renderer.global.abort();
    }
  }
  /**
   * Collect all of the code from the `out` array and return it as a string, or a promise resolving to a string.
   * @param {AccumulatedContent} content
   * @returns {AccumulatedContent}
   */
  #collect_content(content = { head: "", body: "" }) {
    for (let item of this.#out)
      typeof item == "string" ? content[this.type] += item : item.#is_title ? this.global.title = item.#collect_content().head : item.#collect_content(content);
    return content;
  }
  /**
   * Collect all of the code from the `out` array and return it as a string.
   * @param {AccumulatedContent} content
   * @returns {Promise<AccumulatedContent>}
   */
  async #collect_content_async(content = { head: "", body: "" }) {
    await this.promise;
    for (let item of this.#out)
      if (typeof item == "string")
        content[this.type] += item;
      else if (item.#is_title)
        this.global.title = (await item.#collect_content_async()).head;
      else if (item.#boundary) {
        let boundary_content = { head: "", body: "" }, title = this.global.title;
        try {
          await item.#collect_content_async(boundary_content), content.head += boundary_content.head, content.body += boundary_content.body;
        } catch (error) {
          this.global.title = title;
          let { context: context2, failed, transformError } = item.#boundary;
          set_ssr_context(context2);
          let promise = transformError(error);
          set_ssr_context(null);
          let transformed = await promise;
          set_ssr_context(context2);
          let failed_renderer = new _Renderer(item.global, item);
          failed_renderer.type = item.type, failed_renderer.#out.push(_Renderer.#serialize_failed_boundary(transformed)), failed(failed_renderer, transformed, noop), failed_renderer.#out.push(BLOCK_CLOSE), await failed_renderer.#collect_content_async(content);
        }
      } else
        await item.#collect_content_async(content);
    return content;
  }
  async #collect_hydratables() {
    let ctx = get_render_context().hydratable;
    for (let [_, key] of ctx.unresolved_promises)
      unresolved_hydratable(key, ctx.lookup.get(key)?.stack ?? "<missing stack trace>");
    for (let comparison of ctx.comparisons)
      await comparison;
    return await this.#hydratable_block(ctx);
  }
  /**
   * @template {Record<string, any>} Props
   * @param {Renderer} renderer
   * @param {import('svelte').Component<Props>} component
   * @param {{ props?: Omit<Props, '$$slots' | '$$events'>; context?: Map<any, any>; idPrefix?: string; csp?: Csp; transformError?: (error: unknown) => unknown }} options
   * @returns {void}
   */
  static #open_render(renderer, component, options) {
    var previous_context = ssr_context;
    try {
      let context2 = { p: null, c: options.context ?? null, r: renderer, i: !1 };
      set_ssr_context(context2), renderer.push(BLOCK_OPEN), component(renderer, options.props ?? {}), renderer.push(BLOCK_CLOSE);
    } finally {
      set_ssr_context(previous_context);
    }
  }
  /**
   * @param {AccumulatedContent} content
   * @param {Renderer} renderer
   * @returns {AccumulatedContent & { hashes: { script: Sha256Source[] } }}
   */
  static #close_render(content, renderer) {
    let head2 = content.head + renderer.global.title, body = content.body;
    for (let { hash: hash2, code } of renderer.global.css)
      head2 += `<style id="${hash2}">${code}</style>`;
    return {
      head: head2,
      body,
      hashes: {
        script: renderer.global.csp.script_hashes
      }
    };
  }
  /**
   * @param {HydratableContext} ctx
   */
  async #hydratable_block(ctx) {
    if (ctx.lookup.size === 0)
      return null;
    let entries = [], has_promises = !1;
    for (let [k, v] of ctx.lookup) {
      if (v.promises) {
        has_promises = !0;
        for (let p of v.promises) await p;
      }
      entries.push(`[${uneval(k)},${v.serialized}]`);
    }
    let prelude = "const h = (window.__svelte ??= {}).h ??= new Map();";
    has_promises && (prelude = `const r = (v) => Promise.resolve(v);
				${prelude}`);
    let body = `
			{
				${prelude}

				for (const [k, v] of [
					${entries.join(`,
					`)}
				]) {
					h.set(k, v);
				}
			}
		`, csp_attr = "";
    if (this.global.csp.nonce)
      csp_attr = ` nonce="${this.global.csp.nonce}"`;
    else if (this.global.csp.hash) {
      let hash2 = await sha256(body);
      this.global.csp.script_hashes.push(`sha256-${hash2}`);
    }
    return `
		<script${csp_attr}>${body}</script>`;
  }
}, SSRState = class {
  /** @readonly @type {Csp & { script_hashes: Sha256Source[] }} */
  csp;
  /** @readonly @type {'sync' | 'async'} */
  mode;
  /** @readonly @type {() => string} */
  uid;
  /** @readonly @type {Set<{ hash: string; code: string }>} */
  css = /* @__PURE__ */ new Set();
  /** set while collecting so the last title in document order wins */
  title = "";
  /** @type {Set<Promise<unknown>>} */
  #pending = /* @__PURE__ */ new Set();
  /** @type {AbortController | null} */
  #controller = null;
  #aborted = !1;
  /**
   * `transformError` passed to `render`. Called when an error boundary catches an error.
   * Throws by default if unset in `render`.
   * @type {(error: unknown) => unknown}
   */
  transformError;
  /**
   * @param {'sync' | 'async'} mode
   * @param {string} id_prefix
   * @param {Csp} csp
   * @param {((error: unknown) => unknown) | undefined} [transformError]
   */
  constructor(mode, id_prefix = "", csp = { hash: !1 }, transformError) {
    this.mode = mode, this.csp = { ...csp, script_hashes: [] }, this.transformError = transformError ?? ((error) => {
      throw error;
    });
    let uid = 1;
    this.uid = () => `${id_prefix}s${uid++}`;
  }
  /**
   * @template T
   * @param {Promise<T>} promise
   * @returns {Promise<T>}
   */
  track(promise) {
    return this.#pending.add(promise), promise.then(
      () => this.#pending.delete(promise),
      () => this.#pending.delete(promise)
    ), promise;
  }
  async settle() {
    for (; this.#pending.size > 0; )
      await Promise.allSettled([...this.#pending]);
  }
  abort() {
    this.#aborted || (this.#aborted = !0, this.#controller?.abort(STALE_REACTION));
  }
  get_abort_signal() {
    let controller = this.#controller ??= new AbortController();
    return this.#aborted && controller.abort(STALE_REACTION), controller.signal;
  }
};

// node_modules/svelte/src/internal/server/blocks/html.js
function html(value) {
  var html2 = String(value ?? ""), open = false_default ? `<!--${hash(html2)}-->` : "<!---->";
  return open + html2 + "<!---->";
}

// node_modules/svelte/src/html-tree-validation.js
var autoclosing_children = {
  // based on http://developers.whatwg.org/syntax.html#syntax-tag-omission
  li: { direct: ["li"] },
  // https://developer.mozilla.org/en-US/docs/Web/HTML/Element/dt#technical_summary
  dt: { descendant: ["dt", "dd"], reset_by: ["dl"] },
  dd: { descendant: ["dt", "dd"], reset_by: ["dl"] },
  p: {
    descendant: [
      "address",
      "article",
      "aside",
      "blockquote",
      "div",
      "dl",
      "fieldset",
      "footer",
      "form",
      "h1",
      "h2",
      "h3",
      "h4",
      "h5",
      "h6",
      "header",
      "hgroup",
      "hr",
      "main",
      "menu",
      "nav",
      "ol",
      "p",
      "pre",
      "section",
      "table",
      "ul"
    ]
  },
  rt: { descendant: ["rt", "rp"] },
  rp: { descendant: ["rt", "rp"] },
  optgroup: { descendant: ["optgroup"] },
  option: { descendant: ["option", "optgroup"] },
  thead: { direct: ["tbody", "tfoot"] },
  tbody: { direct: ["tbody", "tfoot"] },
  tfoot: { direct: ["tbody"] },
  tr: { direct: ["tr", "tbody"] },
  td: { direct: ["td", "th", "tr"] },
  th: { direct: ["td", "th", "tr"] }
};
var disallowed_children = {
  ...autoclosing_children,
  // Strictly speaking, seeing an <option> doesn't mean we're in a <select>, but we assume it here
  // option or optgroup does not have an `only` restriction because newer browsers support rich HTML content
  // inside option elements. For older browsers, hydration will handle the mismatch.
  form: { descendant: ["form"] },
  a: { descendant: ["a"] },
  button: { descendant: ["button"] },
  h1: { descendant: ["h1", "h2", "h3", "h4", "h5", "h6"] },
  h2: { descendant: ["h1", "h2", "h3", "h4", "h5", "h6"] },
  h3: { descendant: ["h1", "h2", "h3", "h4", "h5", "h6"] },
  h4: { descendant: ["h1", "h2", "h3", "h4", "h5", "h6"] },
  h5: { descendant: ["h1", "h2", "h3", "h4", "h5", "h6"] },
  h6: { descendant: ["h1", "h2", "h3", "h4", "h5", "h6"] },
  // https://html.spec.whatwg.org/multipage/syntax.html#parsing-main-intd
  // https://html.spec.whatwg.org/multipage/syntax.html#parsing-main-incaption
  // No special behavior since these rules fall back to "in body" mode for
  // all except special table nodes which cause bad parsing behavior anyway.
  // https://html.spec.whatwg.org/multipage/syntax.html#parsing-main-intd
  tr: { only: ["th", "td", "style", "script", "template"] },
  // https://html.spec.whatwg.org/multipage/syntax.html#parsing-main-intbody
  tbody: { only: ["tr", "style", "script", "template"] },
  thead: { only: ["tr", "style", "script", "template"] },
  tfoot: { only: ["tr", "style", "script", "template"] },
  // https://html.spec.whatwg.org/multipage/syntax.html#parsing-main-incolgroup
  colgroup: { only: ["col", "template"] },
  // https://html.spec.whatwg.org/multipage/syntax.html#parsing-main-intable
  table: {
    only: ["caption", "colgroup", "tbody", "thead", "tfoot", "style", "script", "template"]
  },
  // https://html.spec.whatwg.org/multipage/syntax.html#parsing-main-inhead
  head: {
    only: [
      "base",
      "basefont",
      "bgsound",
      "link",
      "meta",
      "title",
      "noscript",
      "noframes",
      "style",
      "script",
      "template"
    ]
  },
  // https://html.spec.whatwg.org/multipage/semantics.html#the-html-element
  html: { only: ["head", "body", "frameset"] },
  frameset: { only: ["frame"] },
  "#document": { only: ["html"] }
};
function is_tag_valid_with_ancestor(child_tag, ancestors, child_loc, ancestor_loc) {
  if (child_tag.includes("-")) return null;
  let ancestor_tag = ancestors[ancestors.length - 1], disallowed = disallowed_children[ancestor_tag];
  if (!disallowed) return null;
  if ("reset_by" in disallowed && disallowed.reset_by) {
    for (let i = ancestors.length - 2; i >= 0; i--)
      if (ancestors[i].includes("-") || disallowed.reset_by.includes(ancestors[i]))
        return null;
  }
  if ("descendant" in disallowed && disallowed.descendant.includes(child_tag)) {
    let child = child_loc ? `\`<${child_tag}>\` (${child_loc})` : `\`<${child_tag}>\``, ancestor = ancestor_loc ? `\`<${ancestor_tag}>\` (${ancestor_loc})` : `\`<${ancestor_tag}>\``;
    return `${child} cannot be a descendant of ${ancestor}`;
  }
  return null;
}
function is_tag_valid_with_parent(child_tag, parent_tag, child_loc, parent_loc) {
  if (child_tag.includes("-") || parent_tag?.includes("-") || parent_tag === "template") return null;
  let disallowed = disallowed_children[parent_tag], child = child_loc ? `\`<${child_tag}>\` (${child_loc})` : `\`<${child_tag}>\``, parent = parent_loc ? `\`<${parent_tag}>\` (${parent_loc})` : `\`<${parent_tag}>\``;
  if (disallowed) {
    if ("direct" in disallowed && disallowed.direct.includes(child_tag))
      return `${child} cannot be a direct child of ${parent}`;
    if ("descendant" in disallowed && disallowed.descendant.includes(child_tag))
      return `${child} cannot be a child of ${parent}`;
    if ("only" in disallowed && disallowed.only)
      return disallowed.only.includes(child_tag) ? null : `${child} cannot be a child of ${parent}. \`<${parent_tag}>\` only allows these children: ${disallowed.only.map((d) => `\`<${d}>\``).join(", ")}`;
  }
  switch (child_tag) {
    case "body":
    case "caption":
    case "col":
    case "colgroup":
    case "frameset":
    case "frame":
    case "head":
    case "html":
      return `${child} cannot be a child of ${parent}`;
    case "thead":
    case "tbody":
    case "tfoot":
      return `${child} must be the child of a \`<table>\`, not a ${parent}`;
    case "td":
    case "th":
      return `${child} must be the child of a \`<tr>\`, not a ${parent}`;
    case "tr":
      return `\`<tr>\` must be the child of a \`<thead>\`, \`<tbody>\`, or \`<tfoot>\`, not a ${parent}`;
  }
  return null;
}

// node_modules/svelte/src/internal/server/dev.js
var seen;
function print_error(renderer, message) {
  message = `node_invalid_placement_ssr: ${message}

This can cause content to shift around as the browser repairs the HTML, and will likely result in a \`hydration_mismatch\` warning.`, !(seen ??= /* @__PURE__ */ new Set()).has(message) && (seen.add(message), console.error(message), renderer.head(
    (r) => r.push(
      // ensure that `</script>` can't leak in to the script contents
      `<script>console.error(${JSON.stringify(message).replaceAll("</", "<\\u002f")})</script>`
    )
  ));
}
function push_element(renderer, tag, line, column) {
  var context2 = (
    /** @type {SSRContext} */
    ssr_context
  ), filename = context2.function[FILENAME], parent = context2.element, element2 = { tag, parent, filename, line, column };
  if (parent !== void 0) {
    var ancestor = parent.parent, ancestors = [parent.tag];
    let child_loc = filename ? `${filename}:${line}:${column}` : void 0, parent_loc = parent.filename ? `${parent.filename}:${parent.line}:${parent.column}` : void 0, message = is_tag_valid_with_parent(tag, parent.tag, child_loc, parent_loc);
    for (message && print_error(renderer, message); ancestor != null; ) {
      ancestors.push(ancestor.tag);
      let ancestor_loc = ancestor.filename ? `${ancestor.filename}:${ancestor.line}:${ancestor.column}` : void 0, message2 = is_tag_valid_with_ancestor(tag, ancestors, child_loc, ancestor_loc);
      message2 && print_error(renderer, message2), ancestor = ancestor.parent;
    }
  }
  set_ssr_context({ ...context2, p: context2, element: element2 });
}
function pop_element() {
  set_ssr_context(
    /** @type {SSRContext} */
    ssr_context.p
  );
}
function validate_snippet_args(renderer) {
  (typeof renderer != "object" || // for some reason typescript consider the type of renderer as never after the first instanceof
  !(renderer instanceof Renderer)) && invalid_snippet_arguments();
}
function get_user_code_location() {
  return get_stack().filter((line) => line.trim().startsWith("at ")).map((line) => line.replace(/\((.*):\d+:\d+\)$/, (_, file) => `(${file})`)).join(`
`);
}

// node_modules/svelte/src/internal/server/index.js
var INVALID_ATTR_NAME_CHAR_REGEX = /[\s'">/=\u{FDD0}-\u{FDEF}\u{FFFE}\u{FFFF}\u{1FFFE}\u{1FFFF}\u{2FFFE}\u{2FFFF}\u{3FFFE}\u{3FFFF}\u{4FFFE}\u{4FFFF}\u{5FFFE}\u{5FFFF}\u{6FFFE}\u{6FFFF}\u{7FFFE}\u{7FFFF}\u{8FFFE}\u{8FFFF}\u{9FFFE}\u{9FFFF}\u{AFFFE}\u{AFFFF}\u{BFFFE}\u{BFFFF}\u{CFFFE}\u{CFFFF}\u{DFFFE}\u{DFFFF}\u{EFFFE}\u{EFFFF}\u{FFFFE}\u{FFFFF}\u{10FFFE}\u{10FFFF}]/u;
function element(renderer, tag, attributes_fn = noop, children_fn = noop) {
  renderer.push("<!---->"), tag && (REGEX_VALID_TAG_NAME.test(tag) || dynamic_element_invalid_tag(tag), renderer.push(`<${tag}`), attributes_fn(), renderer.push(">"), is_void(tag) || (children_fn(), is_raw_text_element(tag) || renderer.push(EMPTY_COMMENT), renderer.push(`</${tag}>`))), renderer.push("<!---->");
}
function render(component, options = {}) {
  return options.csp?.hash && options.csp.nonce && invalid_csp(), Renderer.render(
    /** @type {Component<Props>} */
    component,
    options
  );
}
function head(hash2, renderer, fn) {
  renderer.head((renderer2) => {
    renderer2.push(`<!--${hash2}-->`), renderer2.child(fn), renderer2.push(EMPTY_COMMENT);
  });
}
function css_props(renderer, is_html, props, component, dynamic = !1) {
  let styles = style_object_to_string(props);
  is_html ? renderer.push(`<svelte-css-wrapper style="display: contents; ${styles}">`) : renderer.push(`<g style="${styles}">`), component(), dynamic || renderer.push("<!---->"), is_html ? renderer.push("</svelte-css-wrapper>") : renderer.push("</g>");
}
function attributes(attrs, css_hash, classes, styles, flags = 0) {
  styles && (attrs.style = to_style(attrs.style, styles)), attrs.class && (attrs.class = clsx(attrs.class)), (css_hash || classes) && (attrs.class = to_class(attrs.class, css_hash, classes));
  let attr_str = "", name, is_html = (flags & 1) === 0, lowercase = (flags & 2) === 0, is_input = (flags & 4) !== 0, names = Object.keys(attrs);
  outer: for (let i = 0; i < names.length; i++)
    if (name = names[i], typeof attrs[name] != "function" && !(name[0] === "$" && name[1] === "$") && !(name === "" || INVALID_ATTR_NAME_CHAR_REGEX.test(name))) {
      var value = attrs[name], lower = name.toLowerCase();
      if (lowercase && (name = lower), !(lower.length > 2 && lower.startsWith("on"))) {
        if (is_input && (name === "defaultvalue" || name === "defaultchecked")) {
          if (name = name === "defaultvalue" ? "value" : "checked", name in attrs) continue;
          for (let j = 0; j < names.length; j++)
            if (names[j].toLowerCase() === name) continue outer;
        }
        attr_str += attr(name, value, is_html && is_boolean_attribute(name));
      }
    }
  return attr_str;
}
function spread_props(props) {
  let merged_props = {}, key;
  for (let i = 0; i < props.length; i++) {
    let obj = props[i];
    if (obj != null)
      for (key of Object.keys(obj)) {
        let desc = Object.getOwnPropertyDescriptor(obj, key);
        desc ? Object.defineProperty(merged_props, key, desc) : merged_props[key] = obj[key];
      }
  }
  return merged_props;
}
function stringify(value) {
  return typeof value == "string" ? value : value == null ? "" : value + "";
}
function style_object_to_string(style_object) {
  return Object.keys(style_object).filter(
    /** @param {any} key */
    (key) => style_object[key] != null && style_object[key] !== ""
  ).map(
    /** @param {any} key */
    (key) => `${key}: ${escape_html(style_object[key], !0)};`
  ).join(" ");
}
function attr_class(value, hash2, directives) {
  var result = to_class(value, hash2, directives);
  return result ? ` class="${escape_html(result, !0)}"` : "";
}
function attr_style(value, directives) {
  var result = to_style(value, directives);
  return result ? ` style="${escape_html(result, !0)}"` : "";
}
function store_get(store_values, store_name, store) {
  if (false_default && validate_store(store, store_name.slice(1)), store_name in store_values && store_values[store_name][0] === store)
    return store_values[store_name][2];
  store_values[store_name]?.[1](), store_values[store_name] = [store, null, void 0];
  let unsub = subscribe_to_store(
    store,
    /** @param {any} v */
    (v) => store_values[store_name][2] = v
  );
  return store_values[store_name][1] = unsub, store_values[store_name][2];
}
function store_set(store, value) {
  return store.set(value), value;
}
function store_mutate(store_values, store_name, store, expression) {
  return store_set(store, store_get(store_values, store_name, store)), expression;
}
function update_store(store_values, store_name, store, d = 1) {
  let store_value = store_get(store_values, store_name, store);
  return store.set(store_value + d), store_value;
}
function update_store_pre(store_values, store_name, store, d = 1) {
  let value = store_get(store_values, store_name, store) + d;
  return store.set(value), value;
}
function unsubscribe_stores(store_values) {
  for (let store_name of Object.keys(store_values))
    store_values[store_name][1]();
}
function slot(renderer, $$props, name, slot_props, fallback_fn) {
  var slot_fn = $$props.$$slots?.[name];
  slot_fn === !0 && (slot_fn = $$props[name === "default" ? "children" : name]), slot_fn !== void 0 ? slot_fn(renderer, slot_props) : fallback_fn?.();
}
function rest_props(props, rest) {
  let rest_props2 = {}, key;
  for (key of Object.keys(props))
    rest.includes(key) || (rest_props2[key] = props[key]);
  return rest_props2;
}
function sanitize_props(props) {
  let { children, $$slots, ...sanitized } = props;
  return sanitized;
}
function sanitize_slots(props) {
  let sanitized = {};
  props.children && (sanitized.default = !0);
  for (let key of Object.keys(props.$$slots || {}))
    sanitized[key] = !0;
  return sanitized;
}
function bind_props(props_parent, props_now) {
  for (let key of Object.keys(props_now)) {
    let initial_value = props_parent[key], value = props_now[key];
    initial_value === void 0 && value !== void 0 && Object.getOwnPropertyDescriptor(props_parent, key)?.set && (props_parent[key] = value);
  }
}
function await_block(renderer, promise, pending_fn, then_fn) {
  is_promise(promise) ? (renderer.push(BLOCK_OPEN), promise.then(null, noop), pending_fn !== null && pending_fn()) : then_fn !== null && (renderer.push(BLOCK_OPEN_ELSE), then_fn(promise));
}
function ensure_array_like(array_like_or_iterator) {
  return array_like_or_iterator ? array_like_or_iterator.length !== void 0 ? array_like_or_iterator : Array.from(array_like_or_iterator) : [];
}
function once(get_value) {
  let value = (
    /** @type {V} */
    UNINITIALIZED
  );
  return () => (value === UNINITIALIZED && (value = get_value()), value);
}
function props_id(renderer) {
  let uid = renderer.global.uid();
  return renderer.push("<!--$" + uid + "-->"), uid;
}
function derived(fn) {
  let get_value = ssr_context === null ? fn : once(fn), updated_value;
  return function(new_value) {
    return arguments.length === 0 ? updated_value ?? get_value() : (updated_value = new_value, updated_value);
  };
}
function update_derived(derived2, d = 1) {
  let value = derived2(), increase = typeof value == "bigint" ? BigInt(d) : d;
  return derived2(value + /** @type {*} */
  increase), value;
}
function update_derived_pre(derived2, d = 1) {
  let old_value = derived2(), increase = typeof old_value == "bigint" ? BigInt(d) : d, value = old_value + /** @type {*} */
  increase;
  return derived2(value), value;
}
function async_derived(fn) {
  return Promise.resolve(fn()).then((value) => () => value);
}

export {
  hydratable_clobbering,
  hydratable_serialization_failed,
  lifecycle_function_unavailable,
  ssr_context,
  createContext,
  getContext,
  setContext,
  hasContext,
  getAllContexts,
  save,
  get_render_context,
  uneval,
  html,
  element,
  render,
  head,
  css_props,
  attributes,
  spread_props,
  stringify,
  attr_class,
  attr_style,
  store_get,
  store_set,
  store_mutate,
  update_store,
  update_store_pre,
  unsubscribe_stores,
  slot,
  rest_props,
  sanitize_props,
  sanitize_slots,
  bind_props,
  await_block,
  ensure_array_like,
  once,
  props_id,
  derived,
  update_derived,
  update_derived_pre,
  async_derived,
  push_element,
  pop_element,
  validate_snippet_args,
  get_user_code_location
};
