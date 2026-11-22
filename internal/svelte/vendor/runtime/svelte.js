import {
  createContext,
  getAllContexts,
  getContext,
  get_render_context,
  get_user_code_location,
  hasContext,
  hydratable_clobbering,
  hydratable_serialization_failed,
  lifecycle_function_unavailable,
  setContext,
  ssr_context,
  uneval
} from "./chunk-FX4W4LUB.js";
import "./chunk-SZUTS3VD.js";
import "./chunk-6ZQVU4YW.js";
import "./chunk-UC3IQMGR.js";
import {
  async_mode_flag,
  experimental_async_required,
  noop,
  run
} from "./chunk-GFOPKZ2A.js";
import {
  false_default
} from "./chunk-VCLKK7XB.js";

// node_modules/svelte/src/internal/server/abort-signal.js
function getAbortSignal() {
  let context = ssr_context;
  for (; context !== null; ) {
    if (context.r !== null) return context.r.global.get_abort_signal();
    context = context.p;
  }
  return new AbortController().signal;
}

// node_modules/svelte/src/internal/server/hydratable.js
function hydratable(key, fn) {
  async_mode_flag || experimental_async_required("hydratable");
  let { hydratable: hydratable2 } = get_render_context(), entry = hydratable2.lookup.get(key);
  if (entry !== void 0) {
    if (false_default) {
      let comparison = compare(key, entry, encode(key, fn()));
      comparison.catch(() => {
      }), hydratable2.comparisons.push(comparison);
    }
    return (
      /** @type {T} */
      entry.value
    );
  }
  let value = fn();
  return entry = encode(key, value, hydratable2.unresolved_promises), hydratable2.lookup.set(key, entry), value;
}
function encode(key, value, unresolved) {
  let entry = { value, serialized: "" };
  false_default && (entry.stack = get_user_code_location());
  let uid = 1;
  return entry.serialized = uneval(entry.value, (value2, uneval2) => {
    if (is_promise(value2)) {
      let placeholder = `"${uid++}"`, p = value2.then((v) => {
        entry.serialized = entry.serialized.replace(
          placeholder,
          // use the function form here to prevent any string replacement characters from being interpreted
          // in `v`, as it's potentially user-controlled and therefore potentially malicious.
          // https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/String/replace#specifying_a_string_as_the_replacement
          () => `r(${uneval2(v)})`
        );
      }).catch(
        (devalue_error) => hydratable_serialization_failed(
          key,
          serialization_stack(entry.stack, devalue_error?.stack)
        )
      );
      return unresolved?.set(p, key), p.catch(() => {
      }).finally(() => unresolved?.delete(p)), (entry.promises ??= []).push(p), placeholder;
    }
  }), entry;
}
function is_promise(value) {
  return Object.prototype.toString.call(value) === "[object Promise]";
}
async function compare(key, a, b) {
  for (let p of a?.promises ?? [])
    await p;
  for (let p of b?.promises ?? [])
    await p;
  if (a.serialized !== b.serialized) {
    let a_stack = (
      /** @type {string} */
      a.stack
    ), b_stack = (
      /** @type {string} */
      b.stack
    ), stack = a_stack === b_stack ? `Occurred at:
${a_stack}` : `First occurrence at:
${a_stack}

Second occurrence at:
${b_stack}`;
    hydratable_clobbering(key, stack);
  }
}
function serialization_stack(root_stack, uneval_stack) {
  let out = "";
  return root_stack && (out += root_stack + `
`), uneval_stack && (out += `Caused by:
` + uneval_stack + `
`), out || "<missing stack trace>";
}

// node_modules/svelte/src/internal/server/blocks/snippet.js
function createRawSnippet(fn) {
  return (renderer, ...args) => {
    var getters = (
      /** @type {Getters<Params>} */
      args.map((value) => () => value)
    );
    renderer.push(
      fn(...getters).render().trim()
    );
  };
}

// node_modules/svelte/src/index-server.js
function onDestroy(fn) {
  /** @type {SSRContext} */
  ssr_context.r.on_destroy(fn);
}
function createEventDispatcher() {
  return noop;
}
function mount() {
  lifecycle_function_unavailable("mount");
}
function hydrate() {
  lifecycle_function_unavailable("hydrate");
}
function unmount() {
  lifecycle_function_unavailable("unmount");
}
function fork() {
  lifecycle_function_unavailable("fork");
}
async function tick() {
}
async function settled() {
}
export {
  noop as afterUpdate,
  noop as beforeUpdate,
  createContext,
  createEventDispatcher,
  createRawSnippet,
  noop as flushSync,
  fork,
  getAbortSignal,
  getAllContexts,
  getContext,
  hasContext,
  hydratable,
  hydrate,
  mount,
  onDestroy,
  noop as onMount,
  setContext,
  settled,
  tick,
  unmount,
  run as untrack
};
