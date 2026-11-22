import {
  await_reactivity_loss,
  await_waterfall,
  derived_inert,
  false_default,
  hydration_mismatch,
  state_proxy_equality_mismatch
} from "./chunk-CG7SAXEK.js";

// node_modules/svelte/src/internal/shared/utils.js
var is_array = Array.isArray, index_of = Array.prototype.indexOf, includes = Array.prototype.includes, array_from = Array.from, object_keys = Object.keys, define_property = Object.defineProperty, get_descriptor = Object.getOwnPropertyDescriptor, get_descriptors = Object.getOwnPropertyDescriptors, object_prototype = Object.prototype, array_prototype = Array.prototype, get_prototype_of = Object.getPrototypeOf, is_extensible = Object.isExtensible, has_own_property = Object.prototype.hasOwnProperty;
function is_function(thing) {
  return typeof thing == "function";
}
var noop = () => {
};
function is_promise(value) {
  return typeof value?.then == "function";
}
function run(fn) {
  return fn();
}
function run_all(arr) {
  for (var i = 0; i < arr.length; i++)
    arr[i]();
}
function deferred() {
  var resolve, reject, promise = new Promise((res, rej) => {
    resolve = res, reject = rej;
  });
  return { promise, resolve, reject };
}
function fallback(value, fallback2, lazy = !1) {
  return value === void 0 ? lazy ? (
    /** @type {() => V} */
    fallback2()
  ) : (
    /** @type {V} */
    fallback2
  ) : value;
}
function to_array(value, n) {
  if (Array.isArray(value))
    return value;
  if (n === void 0 || !(Symbol.iterator in value))
    return Array.from(value);
  let array = [];
  for (let element of value)
    if (array.push(element), array.length === n) break;
  return array;
}
function exclude_from_object(obj, keys) {
  var result = {};
  for (var key in obj)
    keys.includes(key) || (result[key] = obj[key]);
  for (var symbol of Object.getOwnPropertySymbols(obj))
    Object.propertyIsEnumerable.call(obj, symbol) && !keys.includes(symbol) && (result[symbol] = obj[symbol]);
  return result;
}

// node_modules/svelte/src/constants.js
var HYDRATION_ERROR = {};
var UNINITIALIZED = /* @__PURE__ */ Symbol("uninitialized"), FILENAME = /* @__PURE__ */ Symbol("filename"), HMR = /* @__PURE__ */ Symbol("hmr"), NAMESPACE_HTML = "http://www.w3.org/1999/xhtml", NAMESPACE_SVG = "http://www.w3.org/2000/svg", NAMESPACE_MATHML = "http://www.w3.org/1998/Math/MathML";
var ATTACHMENT_KEY = "@attach";

// node_modules/svelte/src/internal/client/constants.js
var STATE_SYMBOL = /* @__PURE__ */ Symbol("$state"), COMPONENT_SYMBOL = /* @__PURE__ */ Symbol("component"), LEGACY_PROPS = /* @__PURE__ */ Symbol("legacy props"), LOADING_ATTR_SYMBOL = /* @__PURE__ */ Symbol(""), PROXY_PATH_SYMBOL = /* @__PURE__ */ Symbol("proxy path"), ATTRIBUTES_CACHE = /* @__PURE__ */ Symbol("attributes"), CLASS_CACHE = /* @__PURE__ */ Symbol("class"), STYLE_CACHE = /* @__PURE__ */ Symbol("style"), TEXT_CACHE = /* @__PURE__ */ Symbol("text"), FORM_RESET_HANDLER = /* @__PURE__ */ Symbol("form reset"), HMR_ANCHOR = /* @__PURE__ */ Symbol("hmr anchor"), STALE_REACTION = new class extends Error {
  name = "StaleReactionError";
  message = "The reaction that called `getAbortSignal()` was re-run or destroyed";
}(), IS_XHTML = (
  // We gotta write it like this because after downleveling the pure comment may end up in the wrong location
  !!globalThis.document?.contentType && /* @__PURE__ */ globalThis.document.contentType.includes("xml")
), ELEMENT_NODE = 1, TEXT_NODE = 3, COMMENT_NODE = 8, DOCUMENT_FRAGMENT_NODE = 11, CUSTOM_RENDERER_NODE_TYPE_MAP = (
  /** @type {const} */
  {
    fragment: DOCUMENT_FRAGMENT_NODE,
    element: ELEMENT_NODE,
    text: TEXT_NODE,
    comment: COMMENT_NODE
  }
);

// node_modules/svelte/src/internal/client/error-handling.js
var adjustments = /* @__PURE__ */ new WeakMap();
function handle_error(error) {
  var effect2 = active_effect;
  if (effect2 === null)
    return error;
  if (false_default && error instanceof Error && !adjustments.has(error) && adjustments.set(error, get_adjustments(error, effect2)), (effect2.f & 32768) === 0 && (effect2.f & 4) === 0)
    throw false_default && !effect2.parent && error instanceof Error && apply_adjustments(error), error;
  invoke_error_boundary(error, effect2);
}
function invoke_error_boundary(error, effect2) {
  if (error === HYDRATION_ERROR)
    throw error;
  if (!(effect2 !== null && (effect2.f & 16384) !== 0)) {
    for (; effect2 !== null; ) {
      if ((effect2.f & 128) !== 0 && (effect2.f & 33570816) === 0) {
        if ((effect2.f & 32768) === 0)
          throw error;
        try {
          effect2.b.error(error);
          return;
        } catch (e) {
          error = e;
        }
      }
      effect2 = effect2.parent;
    }
    throw false_default && error instanceof Error && apply_adjustments(error), error;
  }
}
function get_adjustments(error, effect2) {
  let message_descriptor = get_descriptor(error, "message");
  if (!(message_descriptor && !message_descriptor.configurable)) {
    for (var indent = is_firefox ? "  " : "	", component_stack = `
${indent}in ${effect2.fn?.name || "<unknown>"}`, context = effect2.ctx; context !== null; )
      component_stack += `
${indent}in ${context.function?.[FILENAME].split("/").pop()}`, context = context.p;
    return {
      message: error.message + `
${component_stack}
`,
      stack: error.stack?.split(`
`).filter((line) => !line.includes("svelte/src/internal")).join(`
`)
    };
  }
}
function apply_adjustments(error) {
  let adjusted = adjustments.get(error);
  adjusted && (define_property(error, "message", {
    value: adjusted.message
  }), define_property(error, "stack", {
    value: adjusted.stack
  }));
}

// node_modules/svelte/src/internal/shared/errors.js
function experimental_async_required(name) {
  if (false_default) {
    let error = new Error(`experimental_async_required
Cannot use \`${name}(...)\` unless the \`experimental.async\` compiler option is \`true\`
https://svelte.dev/e/experimental_async_required`);
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/experimental_async_required");
}
function invalid_default_snippet() {
  if (false_default) {
    let error = new Error("invalid_default_snippet\nCannot use `{@render children(...)}` if the parent component uses `let:` directives. Consider using a named snippet instead\nhttps://svelte.dev/e/invalid_default_snippet");
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/invalid_default_snippet");
}
function invalid_snippet_arguments() {
  if (false_default) {
    let error = new Error("invalid_snippet_arguments\nA snippet function was passed invalid arguments. Snippets should only be instantiated via `{@render ...}`\nhttps://svelte.dev/e/invalid_snippet_arguments");
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/invalid_snippet_arguments");
}
function invariant_violation(message) {
  if (false_default) {
    let error = new Error(`invariant_violation
An invariant violation occurred, meaning Svelte's internal assumptions were flawed. This is a bug in Svelte, not your app — please open an issue at https://github.com/sveltejs/svelte, citing the following message: "${message}"
https://svelte.dev/e/invariant_violation`);
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/invariant_violation");
}
function lifecycle_outside_component(name) {
  if (false_default) {
    let error = new Error(`lifecycle_outside_component
\`${name}(...)\` can only be used during component initialisation
https://svelte.dev/e/lifecycle_outside_component`);
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/lifecycle_outside_component");
}
function missing_context() {
  if (false_default) {
    let error = new Error(`missing_context
Context was not set in the current component or any of its ancestors
https://svelte.dev/e/missing_context`);
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/missing_context");
}
function set_context_after_init() {
  if (false_default) {
    let error = new Error("set_context_after_init\n`setContext` must be called when a component first initializes, not in a subsequent effect or after an `await` expression\nhttps://svelte.dev/e/set_context_after_init");
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/set_context_after_init");
}
function snippet_without_render_tag() {
  if (false_default) {
    let error = new Error("snippet_without_render_tag\nAttempted to render a snippet without a `{@render}` block. This would cause the snippet code to be stringified instead of its content being rendered to the DOM. To fix this, change `{snippet}` to `{@render snippet()}`.\nhttps://svelte.dev/e/snippet_without_render_tag");
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/snippet_without_render_tag");
}
function store_invalid_shape(name) {
  if (false_default) {
    let error = new Error(`store_invalid_shape
\`${name}\` is not a store with a \`subscribe\` method
https://svelte.dev/e/store_invalid_shape`);
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/store_invalid_shape");
}
function svelte_element_invalid_this_value() {
  if (false_default) {
    let error = new Error("svelte_element_invalid_this_value\nThe `this` prop on `<svelte:element>` must be a string, if defined\nhttps://svelte.dev/e/svelte_element_invalid_this_value");
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/svelte_element_invalid_this_value");
}

// node_modules/svelte/src/internal/client/errors.js
function async_derived_orphan() {
  if (false_default) {
    let error = new Error("async_derived_orphan\nCan only create a `$derived(...)` with an `await` expression during component initialization or in a declaration tag\nhttps://svelte.dev/e/async_derived_orphan");
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/async_derived_orphan");
}
function bind_invalid_checkbox_value() {
  if (false_default) {
    let error = new Error("bind_invalid_checkbox_value\nUsing `bind:value` together with a checkbox input is not allowed. Use `bind:checked` instead\nhttps://svelte.dev/e/bind_invalid_checkbox_value");
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/bind_invalid_checkbox_value");
}
function component_api_changed(method, component) {
  if (false_default) {
    let error = new Error(`component_api_changed
Calling \`${method}\` on a component instance (of ${component}) is no longer valid in Svelte 5
https://svelte.dev/e/component_api_changed`);
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/component_api_changed");
}
function component_api_invalid_new(component, name) {
  if (false_default) {
    let error = new Error(`component_api_invalid_new
Attempted to instantiate ${component} with \`new ${name}\`, which is no longer valid in Svelte 5. If this component is not under your control, set the \`compatibility.componentApi\` compiler option to \`4\` to keep it working.
https://svelte.dev/e/component_api_invalid_new`);
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/component_api_invalid_new");
}
function custom_renderer_unsupported_operation(operation) {
  if (false_default) {
    let error = new Error(`custom_renderer_unsupported_operation
\`${operation}\` is not supported when rendering with a custom renderer
https://svelte.dev/e/custom_renderer_unsupported_operation`);
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/custom_renderer_unsupported_operation");
}
function derived_references_self() {
  if (false_default) {
    let error = new Error(`derived_references_self
A derived value cannot reference itself recursively
https://svelte.dev/e/derived_references_self`);
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/derived_references_self");
}
function each_key_duplicate(a, b, value) {
  if (false_default) {
    let error = new Error(`each_key_duplicate
${value ? `Keyed each block has duplicate key \`${value}\` at indexes ${a} and ${b}` : `Keyed each block has duplicate key at indexes ${a} and ${b}`}
https://svelte.dev/e/each_key_duplicate`);
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/each_key_duplicate");
}
function each_key_volatile(index, a, b) {
  if (false_default) {
    let error = new Error(`each_key_volatile
Keyed each block has key that is not idempotent — the key for item at index ${index} was \`${a}\` but is now \`${b}\`. Keys must be the same each time for a given item
https://svelte.dev/e/each_key_volatile`);
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/each_key_volatile");
}
function effect_in_teardown(rune) {
  if (false_default) {
    let error = new Error(`effect_in_teardown
\`${rune}\` cannot be used inside an effect cleanup function
https://svelte.dev/e/effect_in_teardown`);
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/effect_in_teardown");
}
function effect_in_unowned_derived() {
  if (false_default) {
    let error = new Error("effect_in_unowned_derived\nEffect cannot be created inside a `$derived` value that was not itself created inside an effect\nhttps://svelte.dev/e/effect_in_unowned_derived");
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/effect_in_unowned_derived");
}
function effect_orphan(rune) {
  if (false_default) {
    let error = new Error(`effect_orphan
\`${rune}\` can only be used inside an effect (e.g. during component initialisation)
https://svelte.dev/e/effect_orphan`);
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/effect_orphan");
}
function effect_pending_outside_reaction() {
  if (false_default) {
    let error = new Error("effect_pending_outside_reaction\n`$effect.pending()` can only be called inside an effect or derived\nhttps://svelte.dev/e/effect_pending_outside_reaction");
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/effect_pending_outside_reaction");
}
function effect_update_depth_exceeded() {
  if (false_default) {
    let error = new Error(`effect_update_depth_exceeded
Maximum update depth exceeded. This typically indicates that an effect reads and writes the same piece of state
https://svelte.dev/e/effect_update_depth_exceeded`);
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/effect_update_depth_exceeded");
}
function fork_discarded() {
  if (false_default) {
    let error = new Error(`fork_discarded
Cannot commit a fork that was already discarded
https://svelte.dev/e/fork_discarded`);
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/fork_discarded");
}
function fork_timing() {
  if (false_default) {
    let error = new Error(`fork_timing
Cannot create a fork inside an effect or when state changes are pending
https://svelte.dev/e/fork_timing`);
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/fork_timing");
}
function get_abort_signal_after_await() {
  if (false_default) {
    let error = new Error("get_abort_signal_after_await\n`getAbortSignal()` cannot be called after an `await`\nhttps://svelte.dev/e/get_abort_signal_after_await");
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/get_abort_signal_after_await");
}
function get_abort_signal_outside_reaction() {
  if (false_default) {
    let error = new Error("get_abort_signal_outside_reaction\n`getAbortSignal()` can only be called inside an effect or derived\nhttps://svelte.dev/e/get_abort_signal_outside_reaction");
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/get_abort_signal_outside_reaction");
}
function hydratable_missing_but_required(key) {
  if (false_default) {
    let error = new Error(`hydratable_missing_but_required
Expected to find a hydratable with key \`${key}\` during hydration, but did not.
https://svelte.dev/e/hydratable_missing_but_required`);
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/hydratable_missing_but_required");
}
function hydration_failed() {
  if (false_default) {
    let error = new Error(`hydration_failed
Failed to hydrate the application
https://svelte.dev/e/hydration_failed`);
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/hydration_failed");
}
function invalid_snippet() {
  if (false_default) {
    let error = new Error("invalid_snippet\nCould not `{@render}` snippet due to the expression being `null` or `undefined`. Consider using optional chaining `{@render snippet?.()}`\nhttps://svelte.dev/e/invalid_snippet");
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/invalid_snippet");
}
function invalid_snippet_in_custom_renderer() {
  if (false_default) {
    let error = new Error("invalid_snippet_in_custom_renderer\n`createRawSnippet` cannot be used with a custom renderer\nhttps://svelte.dev/e/invalid_snippet_in_custom_renderer");
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/invalid_snippet_in_custom_renderer");
}
function lifecycle_legacy_only(name) {
  if (false_default) {
    let error = new Error(`lifecycle_legacy_only
\`${name}(...)\` cannot be used in runes mode
https://svelte.dev/e/lifecycle_legacy_only`);
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/lifecycle_legacy_only");
}
function props_invalid_value(key) {
  if (false_default) {
    let error = new Error(`props_invalid_value
Cannot do \`bind:${key}={undefined}\` when \`${key}\` has a fallback value
https://svelte.dev/e/props_invalid_value`);
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/props_invalid_value");
}
function props_rest_readonly(property) {
  if (false_default) {
    let error = new Error(`props_rest_readonly
Rest element properties of \`$props()\` such as \`${property}\` are readonly
https://svelte.dev/e/props_rest_readonly`);
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/props_rest_readonly");
}
function renderer_cannot_cross(guest, host) {
  if (false_default) {
    let error = new Error(`renderer_cannot_cross
Cannot render content belonging to ${guest} inside ${host}. A custom renderer must define \`embed\` or \`host\` to be combined with another renderer
https://svelte.dev/e/renderer_cannot_cross`);
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/renderer_cannot_cross");
}
function rune_outside_svelte(rune) {
  if (false_default) {
    let error = new Error(`rune_outside_svelte
The \`${rune}\` rune is only available inside \`.svelte\` and \`.svelte.js/ts\` files
https://svelte.dev/e/rune_outside_svelte`);
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/rune_outside_svelte");
}
function snippet_renderer_mismatch() {
  if (false_default) {
    let error = new Error("snippet_renderer_mismatch\nA snippet created in a component that was compiled without `experimental.customRenderer` cannot be rendered by a custom renderer\nhttps://svelte.dev/e/snippet_renderer_mismatch");
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/snippet_renderer_mismatch");
}
function state_descriptors_fixed() {
  if (false_default) {
    let error = new Error("state_descriptors_fixed\nProperty descriptors defined on `$state` objects must contain `value` and always be `enumerable`, `configurable` and `writable`.\nhttps://svelte.dev/e/state_descriptors_fixed");
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/state_descriptors_fixed");
}
function state_prototype_fixed() {
  if (false_default) {
    let error = new Error("state_prototype_fixed\nCannot set prototype of `$state` object\nhttps://svelte.dev/e/state_prototype_fixed");
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/state_prototype_fixed");
}
function state_unsafe_mutation() {
  if (false_default) {
    let error = new Error("state_unsafe_mutation\nUpdating state inside `$derived(...)`, `$inspect(...)` or a template expression is forbidden. If the value should not be reactive, declare it without `$state`\nhttps://svelte.dev/e/state_unsafe_mutation");
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/state_unsafe_mutation");
}
function svelte_boundary_reset_onerror() {
  if (false_default) {
    let error = new Error("svelte_boundary_reset_onerror\nA `<svelte:boundary>` `reset` function cannot be called while an error is still being handled\nhttps://svelte.dev/e/svelte_boundary_reset_onerror");
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/svelte_boundary_reset_onerror");
}

// node_modules/svelte/src/internal/flags/index.js
var async_mode_flag = !1, legacy_mode_flag = !1, tracing_mode_flag = !1, custom_renderers_flag = !1;

// node_modules/svelte/src/internal/shared/context.js
function create_context(get_context, set_context, has_context) {
  let key = {};
  return [
    () => (has_context(key) || missing_context(), get_context(key)),
    (context) => set_context(key, context),
    () => has_context(key)
  ];
}
function get_parent_context(context) {
  let parent = context.p;
  for (; parent !== null && parent.c === null; )
    parent = parent.p;
  return parent?.c ?? null;
}
function get_or_init_context_map(context, name) {
  return context === null && lifecycle_outside_component(name), context.c ??= new Map(get_parent_context(context) || void 0);
}

// node_modules/svelte/src/internal/client/context.js
var component_context = null;
function set_component_context(context) {
  component_context = context;
}
var dev_stack = null;
function set_dev_stack(stack2) {
  dev_stack = stack2;
}
function add_svelte_meta(callback, type, component, line, column, additional) {
  let parent = dev_stack;
  dev_stack = {
    type,
    file: component[FILENAME],
    line,
    column,
    parent,
    ...additional
  };
  try {
    return callback();
  } finally {
    dev_stack = parent;
  }
}
var dev_current_component_function = null;
function set_dev_current_component_function(fn) {
  dev_current_component_function = fn;
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
    get_or_init_context_map(component_context, "getContext").get(key)
  );
}
function setContext(key, context) {
  let context_map = get_or_init_context_map(component_context, "setContext");
  if (async_mode_flag) {
    var flags = (
      /** @type {Effect} */
      active_effect.f
    ), valid = !active_reaction && (flags & 32) !== 0 && // pop() runs synchronously, so this indicates we're setting context after an await
    !/** @type {ComponentContext} */
    component_context.i;
    valid || set_context_after_init();
  }
  return context_map.set(key, context), context;
}
function hasContext(key) {
  return get_or_init_context_map(component_context, "hasContext").has(key);
}
function getAllContexts() {
  return get_or_init_context_map(component_context, "getAllContexts");
}
function push(props, runes = !1, fn) {
  component_context = {
    p: component_context,
    i: !1,
    c: null,
    e: null,
    s: props,
    x: null,
    r: (
      /** @type {Effect} */
      active_effect
    ),
    l: legacy_mode_flag && !runes ? { s: null, u: null, $: [] } : null
  }, false_default && (component_context.function = fn, dev_current_component_function = fn);
}
function pop(component) {
  var context = (
    /** @type {ComponentContext} */
    component_context
  ), effects = context.e;
  if (effects !== null) {
    context.e = null;
    for (var fn of effects)
      create_user_effect(fn);
  }
  return component !== void 0 && (context.x = component), context.i = !0, component_context = context.p, false_default && (dev_current_component_function = component_context?.function ?? null), mark_as_component(component);
}
function mark_as_component(component = {}) {
  return define_property(component, COMPONENT_SYMBOL, { value: !0 }), component;
}
function is_runes() {
  return !legacy_mode_flag || component_context !== null && component_context.l === null;
}

// node_modules/svelte/src/internal/client/dom/task.js
var micro_tasks = [];
function run_micro_tasks() {
  var tasks = micro_tasks;
  micro_tasks = [], run_all(tasks);
}
function queue_micro_task(fn) {
  if (micro_tasks.length === 0 && !is_flushing_sync) {
    var tasks = micro_tasks;
    queueMicrotask(() => {
      tasks === micro_tasks && run_micro_tasks();
    });
  }
  micro_tasks.push(fn);
}
function flush_tasks() {
  for (; micro_tasks.length > 0; )
    run_micro_tasks();
}

// node_modules/svelte/src/internal/client/reactivity/equality.js
function equals(value) {
  return value === this.v;
}
function safe_not_equal(a, b) {
  return a != a ? b == b : a !== b || a !== null && typeof a == "object" || typeof a == "function";
}
function safe_equals(value) {
  return !safe_not_equal(value, this.v);
}

// node_modules/svelte/src/internal/shared/warnings.js
var bold = "font-weight: bold", normal = "font-weight: normal";
function dynamic_void_element_content(tag2) {
  false_default ? console.warn(`%c[svelte] dynamic_void_element_content
%c\`<svelte:element this="${tag2}">\` is a void element — it cannot have content
https://svelte.dev/e/dynamic_void_element_content`, bold, normal) : console.warn("https://svelte.dev/e/dynamic_void_element_content");
}
function state_snapshot_uncloneable(properties) {
  false_default ? console.warn(
    `%c[svelte] state_snapshot_uncloneable
%c${properties ? `The following properties cannot be cloned with \`$state.snapshot\` — the return value contains the originals:

${properties}` : "Value cannot be cloned with `$state.snapshot` — the original value was returned"}
https://svelte.dev/e/state_snapshot_uncloneable`,
    bold,
    normal
  ) : console.warn("https://svelte.dev/e/state_snapshot_uncloneable");
}

// node_modules/svelte/src/internal/shared/clone.js
var empty = [];
function snapshot(value, skip_warning = !1, no_tojson = !1) {
  if (false_default && !skip_warning) {
    let paths = [], copy = clone(value, /* @__PURE__ */ new Map(), "", paths, null, no_tojson);
    if (paths.length === 1 && paths[0] === "")
      state_snapshot_uncloneable();
    else if (paths.length > 0) {
      let slice = paths.length > 10 ? paths.slice(0, 7) : paths.slice(0, 10), excess = paths.length - slice.length, uncloned = slice.map((path) => `- <value>${path}`).join(`
`);
      excess > 0 && (uncloned += `
- ...and ${excess} more`), state_snapshot_uncloneable(uncloned);
    }
    return copy;
  }
  return clone(value, /* @__PURE__ */ new Map(), "", empty, null, no_tojson);
}
function clone(value, cloned, path, paths, original = null, no_tojson = !1) {
  if (typeof value == "object" && value !== null) {
    var unwrapped = cloned.get(value);
    if (unwrapped !== void 0) return unwrapped;
    if (value instanceof Map) return (
      /** @type {Snapshot<T>} */
      new Map(value)
    );
    if (value instanceof Set) return (
      /** @type {Snapshot<T>} */
      new Set(value)
    );
    if (is_array(value)) {
      var copy = (
        /** @type {Snapshot<any>} */
        Array(value.length)
      );
      cloned.set(value, copy), original !== null && cloned.set(original, copy);
      for (var i = 0; i < value.length; i += 1) {
        var element = value[i];
        i in value && (copy[i] = clone(element, cloned, false_default ? `${path}[${i}]` : path, paths, null, no_tojson));
      }
      return copy;
    }
    if (get_prototype_of(value) === object_prototype) {
      copy = {}, cloned.set(value, copy), original !== null && cloned.set(original, copy);
      for (var key of Object.keys(value))
        copy[key] = clone(
          // @ts-expect-error
          value[key],
          cloned,
          false_default ? `${path}.${key}` : path,
          paths,
          null,
          no_tojson
        );
      return copy;
    }
    if (value instanceof Date)
      return value.getTime(), /** @type {Snapshot<T>} */
      structuredClone(value);
    if (typeof /** @type {T & { toJSON?: any } } */
    value.toJSON == "function" && !no_tojson)
      return clone(
        /** @type {T & { toJSON(): any } } */
        value.toJSON(),
        cloned,
        false_default ? `${path}.toJSON()` : path,
        paths,
        // Associate the instance with the toJSON clone
        value
      );
  }
  if (value instanceof EventTarget)
    return (
      /** @type {Snapshot<T>} */
      value
    );
  try {
    return (
      /** @type {Snapshot<T>} */
      structuredClone(value)
    );
  } catch {
    return false_default && paths.push(path), /** @type {Snapshot<T>} */
    value;
  }
}

// node_modules/svelte/src/internal/shared/dev.js
function get_error(label2) {
  let error = new Error(), stack2 = get_stack();
  return stack2.length === 0 ? null : (stack2.unshift(`
`), define_property(error, "stack", {
    value: stack2.join(`
`)
  }), define_property(error, "name", {
    value: label2
  }), /** @type {Error & { stack: string }} */
  error);
}
function get_stack() {
  let limit = Error.stackTraceLimit;
  Error.stackTraceLimit = 1 / 0;
  let stack2 = new Error().stack;
  if (Error.stackTraceLimit = limit, !stack2) return [];
  let lines = stack2.split(`
`), new_lines = [];
  for (let i = 0; i < lines.length; i++) {
    let line = lines[i], posixified = line.replaceAll("\\", "/");
    if (line.trim() !== "Error") {
      if (line.includes("validate_each_keys"))
        return [];
      posixified.includes("svelte/src/internal") || posixified.includes("node_modules/.vite") || new_lines.push(line);
    }
  }
  return new_lines;
}
function invariant(condition, message) {
  if (!false_default)
    throw new Error("invariant(...) was not guarded by if (DEV)");
  condition || invariant_violation(message);
}

// node_modules/svelte/src/internal/client/custom-renderer/state.js
var current_renderer = null;
function set_renderer(value) {
  current_renderer = value;
}
var depth = 0, suspended_hydration_at = [];
function push_renderer(value) {
  var previous = current_renderer;
  return current_renderer = value, depth += 1, hydrating && value !== null && (set_hydrating(!1), suspended_hydration_at.push(depth)), previous;
}
function pop_renderer(previous) {
  suspended_hydration_at[suspended_hydration_at.length - 1] === depth && (suspended_hydration_at.pop(), set_hydrating(!0)), depth -= 1, current_renderer = previous;
}

// node_modules/svelte/src/internal/client/reactivity/async.js
function flatten(blockers, sync, async, fn) {
  var parent = (
    /** @type {Effect} */
    active_effect
  ), deriveds = sync.map(is_runes() ? derived : derived_safe_equal);
  false_default && deriveds.forEach((d, i) => {
    d.label = sync[i].toString().replace("() => ", "").replaceAll("$.eager(() => ", "$state.eager(").replace(/\$\.get\((.+?)\)/g, (_, id) => id);
  });
  var pending = blockers.filter((b) => !b.settled);
  if (pending.length > 0) {
    var restore = capture(), decrement_pending = increment_pending(), p = pending.length === 1 ? pending[0].promise : Promise.all(pending.map((b) => b.promise));
    p.then(() => {
      if ((parent.f & 16384) === 0) {
        restore(pending[0].run.get_batch());
        try {
          run3();
        } catch (error) {
          invoke_error_boundary(error, parent);
        }
        unset_context();
      }
    }).finally(decrement_pending);
  } else
    run3();
  function run3() {
    if (!blockers.some((b) => b.failed)) {
      if (async.length === 0) {
        fn(deriveds);
        return;
      }
      var signals = async.map((fn2) => async_expression(fn2)), effect2 = null;
      block(() => {
        for (var s of signals)
          if (is_uninitialized(s)) {
            get(s);
            return;
          }
        effect2 !== null ? current_batch.revalidate_branch(effect2) : effect2 = branch(() => {
          try {
            fn([...deriveds, ...signals]);
          } catch (error) {
            invoke_error_boundary(error, parent);
          }
        });
      }, 65536);
    }
  }
}
function run_after_blockers(blockers, fn) {
  flatten(blockers, [], [], fn);
}
function capture() {
  var previous_effect = (
    /** @type {Effect} */
    active_effect
  ), previous_reaction = active_reaction, previous_component_context = component_context, previous_batch = (
    /** @type {Batch} */
    current_batch
  ), previous_renderer = custom_renderers_flag ? current_renderer : null;
  if (false_default)
    var previous_dev_stack = dev_stack;
  return function(batch = previous_batch) {
    set_active_effect(previous_effect), set_active_reaction(previous_reaction), set_component_context(previous_component_context), custom_renderers_flag && set_renderer(previous_renderer), batch !== null && (previous_effect.f & 16384) === 0 && merge_target(batch).activate(), false_default && (set_reactivity_loss_tracker(null), set_dev_stack(previous_dev_stack));
  };
}
var restored = !1;
async function save(promise) {
  var restore = capture();
  unsave();
  var value = await promise;
  return () => (restore(), restored = !0, value);
}
function unsave(value) {
  return restored && unset_context(), /** @type {T} */
  value;
}
async function track_reactivity_loss(promise) {
  unsave();
  var previous_reactivity_loss_tracker = reactivity_loss_tracker;
  queueMicrotask(() => {
    reactivity_loss_tracker === previous_reactivity_loss_tracker && set_reactivity_loss_tracker(null);
  });
  var value = await promise;
  return () => (set_reactivity_loss_tracker(previous_reactivity_loss_tracker), queueMicrotask(() => {
    reactivity_loss_tracker === previous_reactivity_loss_tracker && set_reactivity_loss_tracker(null);
  }), value);
}
async function* for_await_track_reactivity_loss(iterable) {
  let iterator = iterable[Symbol.asyncIterator]?.() ?? iterable[Symbol.iterator]?.();
  if (iterator === void 0)
    throw new TypeError("value is not async iterable");
  let invoke_return = !0;
  try {
    for (; ; ) {
      let { done, value } = (await track_reactivity_loss(iterator.next()))();
      if (done) {
        invoke_return = !1;
        break;
      }
      var prev = reactivity_loss_tracker;
      try {
        yield value;
      } catch (e) {
        throw set_reactivity_loss_tracker(prev), iterator.return !== void 0 && (await track_reactivity_loss(iterator.return()))(), e;
      }
      set_reactivity_loss_tracker(prev);
    }
  } catch (error) {
    throw invoke_return = !1, error;
  } finally {
    if (invoke_return && iterator.return !== void 0)
      return (
        /** @type {TReturn} */
        (await track_reactivity_loss(iterator.return()))().value
      );
  }
}
function unset_context(deactivate_batch = !0) {
  restored = !1, set_active_effect(null), set_active_reaction(null), set_component_context(null), custom_renderers_flag && set_renderer(null), deactivate_batch && current_batch?.deactivate(), false_default && (set_reactivity_loss_tracker(null), set_dev_stack(null));
}
var current_runner = null, Runner = class {
  /** @type {ReturnType<typeof capture>} */
  #restore;
  /**
   * Other batches that share this run
   * @type {Set<Batch>}
   */
  #adopters = /* @__PURE__ */ new Set();
  /** @type {Batch} */
  #batch;
  /** @type {Array<() => void>} */
  continued = [];
  /**
   * Async deriveds created during this run
   * @type {Source[]}
   */
  signals = [];
  /** Whether the full initialization promise chain is still pending. */
  running = !0;
  constructor() {
    this.#batch = /** @type {Batch} */
    current_batch, this.#restore = capture();
  }
  /** The batch to continue in, even if the original fork was discarded. */
  get_batch() {
    if (!merge_target(this.#batch).linked)
      for (let b of this.#adopters) {
        var target = merge_target(b);
        target.linked && (this.#batch = target);
      }
    return this.#batch;
  }
  restore() {
    this.#restore(this.get_batch());
  }
  /**
   *
   * @param {Array<() => void | Promise<void>>} thunks
   */
  run(thunks) {
    var active = (
      /** @type {Effect} */
      active_effect
    ), promise = Promise.resolve(this.#invoke(thunks[0])), blockers = [new Blocker(this, promise)];
    for (let fn of thunks.slice(1))
      promise = promise.then(() => {
        if ((active.f & 16384) !== 0)
          throw STALE_REACTION;
        try {
          return this.restore(), this.#invoke(fn);
        } finally {
          unset_context(), this.#continue();
        }
      }), blockers.push(new Blocker(this, promise));
    return promise = promise.catch((error) => {
      if ((active.f & 16384) === 0) {
        this.restore();
        try {
          invoke_error_boundary(error, active);
        } finally {
          unset_context();
        }
      }
    }).finally(() => this.#continue()), async_effect(() => {
      var batch = (
        /** @type {Batch} */
        current_batch
      );
      this.#adopters.add(batch);
      var decrement_pending = increment_pending();
      promise.finally(() => {
        var target = merge_target(batch);
        target.linked && (active.f & 16384) === 0 && target.revalidate_branch(active), decrement_pending(), this.running = !1;
      });
    }), blockers;
  }
  /**
   * Switch away from the previous batch (which is now obsolete)
   * to another one
   * @param {Batch} batch
   */
  switch(batch) {
    this.#batch = batch;
  }
  #continue() {
    for (let fn of this.continued.splice(0)) fn();
  }
  /**
   * @param {() => void | Promise<void>} fn
   */
  #invoke(fn) {
    var previous_runner = current_runner;
    current_runner = this;
    try {
      return fn();
    } finally {
      current_runner = previous_runner;
    }
  }
}, Blocker = class {
  settled = !1;
  failed = !1;
  /**
   * @param {Runner} run
   * @param {Promise<any>} promise
   */
  constructor(run3, promise) {
    this.run = run3, this.promise = promise.catch(() => {
      this.failed = !0;
    }).finally(() => {
      this.settled = !0, unset_context();
    });
  }
};
function run2(thunks) {
  return new Runner().run(thunks);
}
function wait(blockers) {
  return Promise.all(blockers.map((b) => b.promise));
}
function increment_pending() {
  for (var effect2 = (
    /** @type {Effect} */
    active_effect
  ), boundary = effect2.b, batch = merge_target(
    /** @type {Batch} */
    current_batch
  ), pending = boundary; pending && !pending.is_pending; )
    pending = pending.parent;
  return batch.increment(effect2), pending?.increment(effect2, batch), boundary?.update_effect_pending(1), () => {
    merge_target(batch).decrement(effect2), pending?.decrement(effect2, batch), boundary?.update_effect_pending(-1);
  };
}
function merge_target(batch) {
  for (; batch.merged !== null; ) batch = batch.merged;
  return batch;
}

// node_modules/svelte/src/internal/client/reactivity/status.js
var STATUS_MASK = -7169;
function set_signal_status(signal, status) {
  signal.f = signal.f & STATUS_MASK | status;
}
function update_derived_status(derived2) {
  (derived2.f & 512) !== 0 || derived2.edges === null ? set_signal_status(derived2, 1024) : set_signal_status(derived2, 4096);
}

// node_modules/svelte/src/internal/client/dev/debug.js
function stringify_derived(fn) {
  return `{${fn.toString().replace(/^async \(\) => /, "").replace(/^\(/, "").replace(/\$\.get\((.+?)\)/g, (_, name) => name).replace(/\$\.track_reactivity_loss\((.+?)\)\(\)/g, (_, name) => name).replace(/\)$/, "")}}`;
}

// node_modules/svelte/src/internal/client/reactivity/deriveds.js
var reactivity_loss_tracker = null;
function set_reactivity_loss_tracker(v) {
  reactivity_loss_tracker = v;
}
var recent_async_deriveds = /* @__PURE__ */ new Set();
// @__NO_SIDE_EFFECTS__
function derived(fn) {
  var flags = 2050;
  active_effect !== null && (active_effect.f |= 524288);
  let signal = {
    ctx: component_context,
    deps: null,
    dv: null,
    edges: null,
    effects: null,
    equals,
    f: flags,
    fn,
    reactions: null,
    rv: 0,
    v: (
      /** @type {V} */
      UNINITIALIZED
    ),
    wv: 0,
    parent: active_effect,
    ac: null
  };
  return false_default && tracing_mode_flag && (signal.created = get_error("created at")), signal;
}
var OBSOLETE = /* @__PURE__ */ Symbol("obsolete");
// @__NO_SIDE_EFFECTS__
function async_expression(fn, label2, location, onready) {
  var signal = source(
    /** @type {V} */
    UNINITIALIZED
  );
  signal.f |= 4194304, false_default && (signal.label = label2 ?? stringify_derived(fn));
  var deferreds = /* @__PURE__ */ new Set(), runner = current_runner, resolver = runner && current_batch, waiting = runner && [];
  function flush_waiting(continued = !1) {
    var fns = (
      /** @type {Array<() => void>} */
      waiting
    );
    waiting = continued ? null : [], continued && (resolver = null);
    for (let fn2 of fns) fn2();
  }
  return async_effect(() => {
    var effect2 = (
      /** @type {Effect} */
      active_effect
    );
    false_default && (reactivity_loss_tracker = { effect: effect2, effect_deps: /* @__PURE__ */ new Set(), warned: !1 });
    var d = deferred();
    try {
      var result = fn();
      if (result === UNINITIALIZED) return;
      var promise = Promise.resolve(result), abort_signal2 = abort_controller?.signal;
      promise.then(
        (value) => {
          abort_signal2?.aborted || d.resolve(value);
        },
        (e) => {
          abort_signal2?.aborted || e === STALE_REACTION && runner === null || d.reject(e === STALE_REACTION ? OBSOLETE : e);
        }
      ).finally(unset_context);
    } catch (error) {
      d.reject(error);
    }
    if (false_default) {
      if (reactivity_loss_tracker) {
        if (effect2.deps !== null)
          for (let i = 0; i < skipped_deps; i += 1)
            reactivity_loss_tracker.effect_deps.add(effect2.deps[i]);
        if (new_deps !== null)
          for (let i = 0; i < new_deps.length; i += 1)
            reactivity_loss_tracker.effect_deps.add(new_deps[i]);
      }
      reactivity_loss_tracker = null;
    }
    var batch = (
      /** @type {Batch} */
      current_batch
    ), decrement_pending = increment_pending();
    batch.async_deriveds.get(effect2)?.reject(OBSOLETE), deferreds.add(d), batch.async_deriveds.set(effect2, d);
    let handler = (value, error = void 0) => {
      if (false_default && (invariant(view_batch === null, "Batch is currently active"), reactivity_loss_tracker = null), waiting !== null && error !== OBSOLETE && (resolver ??= batch, merge_target(batch) !== merge_target(resolver))) {
        waiting.push(() => {
          var obsolete = (effect2.f & 16384) !== 0 || (batch.linked ? batch.async_deriveds.get(effect2) !== d : !merge_target(batch).linked);
          handler(value, obsolete ? OBSOLETE : error);
        });
        return;
      }
      if (decrement_pending(), deferreds.delete(d), error) {
        if (waiting !== null && resolver !== null && (deferreds.size > 0 || error === OBSOLETE) && merge_target(resolver) === merge_target(batch) && merge_target(batch).async_deriveds.get(effect2) === d) {
          resolver = null, flush_waiting();
          return;
        } else if (error === OBSOLETE)
          return;
      }
      var same = signal.equals(value) && !error && (signal.f & 8388608) === 0;
      merge_target(batch).capture(signal, error ?? value, same, !!error), runner !== null && (runner.switch(batch), runner.continued.push(() => flush_waiting(!0)), runner = null), onready?.(error), onready = void 0, false_default && location !== void 0 && !signal.equals(value) && (recent_async_deriveds.add(signal), setTimeout(() => {
        recent_async_deriveds.has(signal) && (effect2.f & 16384) === 0 && (await_waterfall(
          /** @type {string} */
          signal.label,
          location
        ), recent_async_deriveds.delete(signal));
      }));
    };
    d.promise.then(handler, (e) => handler(null, e || "unknown"));
  }), teardown(() => {
    for (let d of deferreds)
      d.reject(OBSOLETE);
  }), signal;
}
// @__NO_SIDE_EFFECTS__
function async_derived(fn, label2, location) {
  current_runner || async_derived_orphan();
  let runner = current_runner, batch = (
    /** @type {Batch} */
    current_batch
  );
  return new Promise((fulfil, reject) => {
    var source2 = /* @__PURE__ */ async_expression(
      () => {
        var effect2 = (
          /** @type {Effect} */
          active_effect
        ), target = merge_target(
          /** @type {Batch} */
          current_batch
        );
        if (runner.running && target !== batch && is_uninitialized(source2))
          for (var i = index - 1; i >= 0; i--) {
            var s = runner.signals[i];
            if (is_uninitialized(s)) {
              get(s);
              var pending = (
                /** @type {Batch} */
                current_batch.async_deriveds
              );
              return pending.get(effect2)?.reject(OBSOLETE), pending.delete(effect2), UNINITIALIZED;
            }
          }
        return fn();
      },
      label2,
      location,
      (error) => {
        error === void 0 ? fulfil(
          /** @type {Source<V>} */
          source2
        ) : reject(error);
      }
    ), index = runner.signals.push(source2) - 1;
  });
}
function is_uninitialized(signal) {
  return (batch_values?.get(signal) ?? signal).v === UNINITIALIZED;
}
// @__NO_SIDE_EFFECTS__
function user_derived(fn) {
  let d = /* @__PURE__ */ derived(fn);
  return async_mode_flag || push_reaction_value(d), d;
}
// @__NO_SIDE_EFFECTS__
function derived_safe_equal(fn) {
  let signal = /* @__PURE__ */ derived(fn);
  return signal.equals = safe_equals, signal;
}
function destroy_derived_effects(derived2) {
  var effects = derived2.effects;
  if (effects !== null) {
    derived2.effects = null;
    for (var i = 0; i < effects.length; i += 1)
      destroy_effect(
        /** @type {Effect} */
        effects[i]
      );
  }
}
var stack = [];
function execute_derived(derived2) {
  var prev_active_effect = active_effect, parent = derived2.parent;
  if (!is_destroying_effect && parent !== null && derived2.v !== UNINITIALIZED && // if it was never evaluated before, it's guaranteed to fail downstream, so we try to execute instead
  (parent.f & 24576) !== 0)
    return derived_inert(), update_derived_status(derived2), derived2.v;
  false_default && includes.call(stack, derived2) && derived_references_self(), set_active_effect(parent);
  var prev_eager_effects = eager_effects;
  false_default && (set_eager_effects(/* @__PURE__ */ new Set()), stack.push(derived2));
  try {
    return destroy_derived_effects(derived2), update_reaction(derived2);
  } finally {
    set_active_effect(prev_active_effect), false_default && (set_eager_effects(prev_eager_effects), stack.pop());
  }
}
function update_derived(signal) {
  var derived2 = (
    /** @type {Derived} */
    signal
  );
  if (is_dirty(derived2)) {
    var records = view_batch?.records, record = records?.get(derived2);
    record !== void 0 && is_current(record.deps, record.dv) ? (records.delete(derived2), promote_record(derived2, record)) : execute_derived(derived2);
  }
}
function get_view(derived2) {
  var cache = (
    /** @type {Map<Derived, Derived | DerivedRecord | BatchValue>} */
    view_cache
  ), entry = cache.get(derived2);
  if (entry !== void 0) return entry;
  var batch = (
    /** @type {Batch} */
    view_batch
  ), written = (
    /** @type {Map<Value, BatchValue>} */
    batch_values.get(derived2)
  ), held = written ?? derived2, assigned = written === void 0 ? (derived2.f & 67108864) !== 0 : written.assigned, record = (
    /** @type {DerivedRecord | undefined} */
    batch.get_record(derived2)
  );
  return assigned && !changed_since((record ?? derived2).deps, held.wv) ? entry = held : record !== void 0 && is_current(record.deps, record.dv) ? entry = record : written === void 0 && !assigned && derived2.dv !== null && is_current(derived2.deps, derived2.dv) ? entry = derived2 : (execute_derived(derived2), entry = /** @type {DerivedRecord | undefined} */
  batch.records?.get(derived2) ?? derived2), (derived2.f & 512) !== 0 && cache.set(derived2, entry), entry;
}
function freeze_derived_effects(derived2) {
  if (derived2.effects !== null)
    for (let e of derived2.effects)
      (e.teardown || (e.f & 134217728) !== 0) && (e.teardown?.(), abort_all(e), e.fn !== null && (e.teardown = noop), remove_reactions(e), destroy_effect_children(e));
}
function unfreeze_derived_effects(derived2) {
  if (derived2.effects !== null)
    for (let e of derived2.effects)
      e.teardown && e.fn !== null && update_effect(e);
}

// node_modules/svelte/src/internal/client/dev/tracing.js
var tracing_expressions = null;
function view_of(signal) {
  return batch_values === null ? signal : (signal.f & 2) !== 0 ? get_view(
    /** @type {Derived} */
    signal
  ) : batch_values.get(signal) ?? signal;
}
function run_in_view(reaction) {
  return (reaction.f & 2) !== 0 && batch_values !== null ? view_batch?.get_record(reaction) ?? reaction : run_of(reaction);
}
function is_changed(signal, run3, since) {
  var i = run3.deps?.indexOf(signal) ?? -1, seen2 = i === -1 ? void 0 : run3.dv?.[i];
  return seen2 !== void 0 ? version_of(signal) !== seen2 : since === 0 || version_of(signal) > since;
}
function log_entry(signal, entry, run3, since) {
  let view = view_of(signal), value = view.v;
  if (value === UNINITIALIZED)
    return;
  let type = get_type(signal), dirty = is_changed(signal, run3, since), style = dirty ? "color: CornflowerBlue; font-weight: bold" : "color: grey; font-weight: normal";
  if (console.groupCollapsed(
    signal.label ? `%c${type}%c ${signal.label}` : `%c${type}%c`,
    style,
    dirty ? "font-weight: normal" : style,
    typeof value == "object" && value !== null && STATE_SYMBOL in value ? snapshot(value, !0) : value
  ), (signal.f & 2) !== 0) {
    let deps = new Set(
      ("deps" in view ? view : run_in_view(
        /** @type {Derived} */
        signal
      )).deps
    );
    for (let dep of deps)
      log_entry(dep, void 0, run3, since);
  }
  if (signal.created && console.log(signal.created), dirty && signal.updated)
    for (let updated of signal.updated.values())
      updated.error && console.log(updated.error);
  if (entry)
    for (var trace2 of entry.traces)
      console.log(trace2);
  console.groupEnd();
}
function get_type(signal) {
  return (signal.f & 4194306) !== 0 ? "$derived" : signal.label?.startsWith("$") ? "store" : "$state";
}
function trace(label2, fn) {
  var previously_tracing_expressions = tracing_expressions;
  try {
    tracing_expressions = { entries: /* @__PURE__ */ new Map(), reaction: active_reaction };
    var start = performance.now(), value = fn(), time = (performance.now() - start).toFixed(2), prefix = untrack(label2);
    if (!effect_tracking())
      console.log(`${prefix} %cran outside of an effect (${time}ms)`, "color: grey");
    else if (tracing_expressions.entries.size === 0)
      console.log(`${prefix} %cno reactive dependencies (${time}ms)`, "color: grey");
    else {
      console.group(`${prefix} %c(${time}ms)`, "color: grey");
      var entries = tracing_expressions.entries, reaction = (
        /** @type {Reaction} */
        active_reaction
      ), run3 = run_in_view(reaction), since = run3.wv ?? reaction.wv;
      untrack(() => {
        for (let [signal, traces] of entries)
          log_entry(signal, traces, run3, since);
      }), tracing_expressions = null, console.groupEnd();
    }
    return value;
  } finally {
    tracing_expressions = previously_tracing_expressions;
  }
}
function tag(source2, label2) {
  return source2.label = label2, tag_proxy(source2.v, label2), source2;
}
function tag_proxy(value, label2) {
  return value?.[PROXY_PATH_SYMBOL]?.(label2), value;
}
function label(value) {
  return typeof value == "symbol" ? `Symbol(${value.description})` : typeof value == "function" ? "<function>" : typeof value == "object" && value ? "<object>" : String(value);
}

// node_modules/svelte/src/internal/client/reactivity/sources.js
var eager_effects = /* @__PURE__ */ new Set();
function set_eager_effects(v) {
  eager_effects = v;
}
var eager_effects_deferred = !1;
function set_eager_effects_deferred() {
  eager_effects_deferred = !0;
}
function source(v, stack2) {
  var signal = {
    f: 0,
    v,
    reactions: null,
    equals,
    rv: 0,
    wv: 0
  };
  return false_default && tracing_mode_flag && (signal.created = stack2 ?? get_error("created at"), signal.updated = null, signal.set_during_effect = !1, signal.trace = null), signal;
}
// @__NO_SIDE_EFFECTS__
function state(v, stack2) {
  let s = source(v, stack2);
  return push_reaction_value(s), s;
}
// @__NO_SIDE_EFFECTS__
function mutable_source(initial_value, immutable = !1, trackable = !0) {
  let s = source(initial_value);
  return immutable || (s.equals = safe_equals), legacy_mode_flag && trackable && component_context !== null && component_context.l !== null && (component_context.l.s ??= []).push(s), s;
}
function mutate(source2, value) {
  return set(
    source2,
    untrack(() => get(source2))
  ), value;
}
function set(source2, value, should_proxy = !1) {
  active_reaction !== null && // since we are untracking the function inside `$inspect.with` we need to add this check
  // to ensure we error if state is set inside an inspect effect
  (!untracking || (active_reaction.f & 131072) !== 0) && is_runes() && (active_reaction.f & 4325394) !== 0 && (current_sources === null || !current_sources.has(source2)) && state_unsafe_mutation();
  let new_value = should_proxy ? proxy(value) : value;
  return false_default && tag_proxy(
    new_value,
    /** @type {string} */
    source2.label
  ), internal_set(source2, new_value, legacy_updates);
}
function internal_set(source2, value, updated_during_traversal = null) {
  var same = source2.equals(value) && (source2.f & 8388608) === 0, entry = batch_values?.get(source2);
  if (!same || entry !== void 0 && (entry.e || !source2.equals.call(entry, value))) {
    var batch = Batch.ensure();
    if ((source2.f & 67110914) === 2050 && execute_derived(
      /** @type {Derived} */
      source2
    ), batch.capture(source2, value, same, !1, updated_during_traversal), false_default) {
      if (tracing_mode_flag || active_effect !== null) {
        source2.updated ??= /* @__PURE__ */ new Map(), source_stacks.add(source2);
        let count = (source2.updated.get("")?.count ?? 0) + 1;
        if (source2.updated.set("", { error: (
          /** @type {any} */
          null
        ), count }), tracing_mode_flag || count > 5) {
          let error = get_error("updated at");
          if (error !== null) {
            let entry2 = source2.updated.get(error.stack);
            entry2 || (entry2 = { error, count: 0 }, source2.updated.set(error.stack, entry2)), entry2.count++;
          }
        }
      }
      active_effect !== null && (source2.set_during_effect = !0);
    }
    is_runes() && active_effect !== null && (active_effect.f & 1024) !== 0 && (active_effect.f & 96) === 0 && (untracked_writes === null ? set_untracked_writes([source2]) : untracked_writes.push(source2));
  } else is_destroying_effect && teardown_write(source2);
  return value;
}
function flush_eager_effects() {
  eager_effects_deferred = !1, in_real_world(run_eager_effects);
}
function run_eager_effects() {
  for (let effect2 of eager_effects) {
    (effect2.f & 1024) !== 0 && set_signal_status(effect2, 4096);
    let dirty;
    try {
      dirty = is_dirty(effect2);
    } catch {
      dirty = !0;
    }
    dirty && update_effect(effect2);
  }
  eager_effects.clear();
}
function update(source2, d = 1) {
  var value = get(source2), result = d === 1 ? value++ : value--;
  return set(source2, value), result;
}
function update_pre(source2, d = 1) {
  var value = get(source2);
  return set(source2, d === 1 ? ++value : --value);
}
function increment(source2) {
  set(source2, source2.v + 1);
}

// node_modules/svelte/src/internal/client/reactivity/batch.js
var first_batch = null, last_batch = null, current_batch = null, processing_batch = null, teardown_context = null;
function set_teardown_context(context) {
  teardown_context = context;
}
function push_teardown_context(values) {
  var context = teardown_context = {
    reverted_values: new Map(teardown_context?.reverted_values),
    derived_cache: /* @__PURE__ */ new Map(),
    parent: teardown_context
  };
  for (let [source2, value] of values ?? []) context.reverted_values.set(source2, value);
  return context;
}
function teardown_write(source2) {
  for (var context = teardown_context; context !== null; context = context.parent)
    context.reverted_values.delete(source2), context.derived_cache.clear();
}
var batch_values = null, view_cache = null, view_batch = null, last_scheduled_effect = null, rebases = /* @__PURE__ */ new Set(), is_flushing_sync = !1, collected_effects = null, legacy_updates = null, flush_count = 0, source_stacks = /* @__PURE__ */ new Set(), seen = null, count_deps = 0, uid = 1, last_processed = null, Batch = class _Batch {
  id = uid++;
  /** True as soon as `#process` was called */
  started = !1;
  linked = !0;
  /** @type {Batch | null} */
  prev = null;
  /** @type {Batch | null} */
  next = null;
  /** @type {Map<Effect, ReturnType<typeof deferred<any>>>} */
  async_deriveds = /* @__PURE__ */ new Map();
  /**
   * The current values (and their versions) of any sources that are updated in this batch,
   * including deriveds that were overridden via assignment — until it commits them (see `#commit`).
   * Its keys are those of `this.previous`, plus the sources this batch gave their first value. `e` says whether
   * the value is an error
   * @type {Map<Value, BatchValue>}
   */
  current = /* @__PURE__ */ new Map();
  /**
   * The values (and their versions) of any sources that are updated in this batch _before_ those updates took place.
   * For a derived, `assigned` says whether its value before was assigned, too (otherwise it was computed).
   * A source that this batch gave its first value (e.g. an async expression that resolved for the first time) has
   * no entry: there's no value from before it, so other views can't undo the write, and teardowns don't revert it.
   * @type {Map<Value, BatchValue>}
   */
  previous = /* @__PURE__ */ new Map();
  /**
   * Results of reactions that belong to this batch rather than to the real world:
   * deriveds evaluated in this batch's view of the world with values that are not the
   * latest ones, and block/async effects that ran while this batch was processed.
   * Effect records become global state when the batch commits (see `promote_record`)
   * @type {Map<Reaction, BatchRecord> | null}
   */
  records = null;
  /**
   * Earlier batches whose changes are part of this batch's view of the world, because
   * they intersect with it (this batch will be merged into them, see `#merge`)
   * @type {Batch[]}
   */
  folded = [];
  /**
   * When the batch is committed (and the DOM is updated), we need to remove old branches
   * and append new ones by calling the functions added inside (if/each/key/etc) blocks.
   * Each callback maps to the writes that apply in the teardowns it causes, if it was added
   * while block effects re-ran eagerly (see `eager_block_writes`)
   * @type {Map<(batch: Batch) => void, Map<Value, BatchValue> | null>}
   */
  #commit_callbacks = /* @__PURE__ */ new Map();
  /**
   * If a fork is discarded, we need to destroy any effects that are no longer needed
   * (see `#commit_callbacks`)
   * @type {Map<(batch: Batch) => void, Map<Value, BatchValue> | null>}
   */
  #discard_callbacks = /* @__PURE__ */ new Map();
  /**
   * Async effects that are currently in flight
   * @type {Map<Effect, number>}
   */
  #pending = /* @__PURE__ */ new Map();
  /**
   * A deferred that resolves when the batch is committed, used with `settled()`
   * TODO replace with Promise.withResolvers once supported widely enough
   * @type {{ promise: Promise<void>, resolve: (value?: any) => void, reject: (reason: unknown) => void } | null}
   */
  #deferred = null;
  /**
   * Effects to (re-)validate when this batch is processed next, and their status (DIRTY or MAYBE_DIRTY):
   * those that were scheduled while it wasn't being processed (see {@link schedule}), and those whose run
   * was deferred because it has pending work (see `#defer_effects`)
   * @type {Map<Effect, number>}
   */
  #unprocessed = /* @__PURE__ */ new Map();
  /**
   * Effects that were scheduled in this batch but not yet 'resolved' into the
   * root effects that need to be flushed. Resolving — the upwards traversal that
   * marks the path to each effect on the shared effect tree (see #resolve) — is
   * deferred until the batch is processed, so that the markers are created and
   * consumed within a single traversal. Scheduling into other batches (which can
   * happen concurrently, e.g. while a batch is committed) can therefore never
   * observe (and be confused by) this batch's markers.
   * May contain duplicates — deduplication happens during resolving
   * @type {Effect[]}
   */
  #scheduled = [];
  /**
   * Branches that still exist, but will be destroyed when this batch
   * is committed — we skip over these during `#process`
   * @type {Set<Effect>}
   */
  #skipped_branches = /* @__PURE__ */ new Set();
  /**
   * Branches that this batch selected (see `unskip_effect`). A branch that is offscreen until
   * it is committed (see `EFFECT_OFFSCREEN`) only exists in the views of the batches that selected it
   * @type {Set<Effect>}
   */
  #unskipped_branches = /* @__PURE__ */ new Set();
  is_fork = !1;
  /**
   * The batch that this one was merged into, if any (see `#merge`). Its async work that is still in flight
   * continues as part of that one
   * @type {Batch | null}
   */
  merged = null;
  /**
   * Batches whose runs this batch aborted. Merge into them before waiting for pending work.
   * Requests belong to this batch because a synchronous derived can abort outside traversal,
   * before an unrelated batch is processed (e.g. a fork being committed).
   * @type {Set<Batch>}
   */
  merge_requests = /* @__PURE__ */ new Set();
  #decrement_queued = !1;
  constructor() {
    last_batch === null ? first_batch = last_batch = this : (last_batch.next = this, this.prev = last_batch), last_batch = this;
  }
  #is_deferred() {
    if (this.is_fork) return !0;
    for (let effect2 of this.#pending.keys())
      if (this.is_blocked_by(effect2))
        return !0;
    return !1;
  }
  /**
   * Whether `effect` is a branch that this batch skips, or that only exists offscreen for other batches
   * @param {Effect} effect
   */
  #skips(effect2) {
    return this.#skipped_branches.has(effect2) || (effect2.f & 33554432) !== 0 && !this.#unskipped_branches.has(effect2);
  }
  /**
   * Whether `effect` is inside a branch that this batch skips, i.e. doesn't exist in its view of the world
   * @param {Effect} effect
   */
  is_skipped(effect2) {
    for (var e = (
      /** @type {Effect | null} */
      effect2
    ); e !== null; e = e.parent)
      if (this.#skips(e)) return !0;
    return !1;
  }
  /**
   * Whether async work in `effect` holds up this batch — or, with `root`, the boundary whose content `root` is.
   * It doesn't if the effect is inside a branch that this batch skips (the branch is going to be destroyed),
   * or inside a boundary that is pending (which shows its pending snippet in the meantime).
   * Unlike a skipped branch, the content of a pending boundary does exist in the view of the batch: its
   * effects are still re-validated (see `#revalidate`), they just don't keep the batch from committing
   * @param {Effect} effect
   * @param {Effect | null} [root]
   */
  is_blocked_by(effect2, root = null) {
    for (var e = (
      /** @type {Effect | null} */
      effect2
    ); e !== null && e !== root; e = e.parent)
      if (this.#skips(e) || (e.f & 16512) === 128 && /** @type {Boundary} */
      e.b.is_pending)
        return !1;
    return !0;
  }
  /**
   * @param {Effect} effect
   */
  skip_effect(effect2) {
    this.#skipped_branches.add(effect2), this.#unskipped_branches.delete(effect2);
  }
  /**
   * @param {Effect} effect
   */
  unskip_effect(effect2) {
    (this.#skipped_branches.delete(effect2) || !this.#unskipped_branches.has(effect2) && ((effect2.f & 33554432) !== 0 || this.folded.some((batch) => batch.#skipped_branches.has(effect2)))) && this.revalidate_branch(effect2), this.#unskipped_branches.add(effect2);
  }
  /**
   * Like `unskip_effect`, for a branch that was just created offscreen while this batch is processed.
   * Its effects ran in this batch's view of the world when they were created, so there's nothing to re-validate
   * TODO can we skip revalidation in `unskip_effect` in more cases, by detecting whether things changed
   * inside a branch while it was skipped?
   * @param {Effect} effect
   */
  unskip_new_effect(effect2) {
    this.#unskipped_branches.add(effect2);
  }
  /**
   * @param {Effect} effect
   */
  revalidate_branch(effect2) {
    for (var e = effect2.first; e !== null; e = e.next) {
      var flags = e.f;
      (flags & 24576) === 0 && !this.#skips(e) && ((flags & 131168) === 0 && e.fn !== null && this.schedule(e, 4096), this.revalidate_branch(e));
    }
  }
  /**
   * Convert the effects that were scheduled in this batch into the root effects
   * that need to be traversed, marking the path to each effect (by clearing the
   * `CLEAN` flag on ancestor branches) so that the traversal can find them.
   * This happens right before traversal rather than at scheduling time, so that
   * the markers left on the (shared) effect tree are created and consumed within
   * a single traversal — scheduling into other batches can never observe them
   * @returns {Effect[]}
   */
  #resolve() {
    var roots = [];
    for (let effect2 of this.#scheduled)
      if (!((effect2.f & 16384) !== 0 || (effect2.f & 6144) === 0)) {
        for (var e = effect2, covered = !1; e.parent !== null; ) {
          e = e.parent;
          var flags = e.f;
          if ((flags & 96) !== 0) {
            if ((flags & 1024) === 0) {
              covered = !0;
              break;
            }
            e.f ^= 1024;
          }
        }
        covered || roots.push(e);
      }
    return this.#scheduled = [], roots;
  }
  #process() {
    if (current_batch = processing_batch = this, teardown_context?.derived_cache.clear(), async_mode_flag && this.started && last_processed !== this.id && this.#revalidate(this.current.keys()), this.started = !0, last_processed = this.id, this.#unprocessed.size > 0) {
      for (let [e, status] of this.#unprocessed) this.#add(e, status);
      this.#unprocessed.clear();
    }
    if (false_default)
      for (let value of this.current.keys())
        source_stacks.add(value);
    this.activate();
    for (var effects = collected_effects = [], render_effects = [], revealed_boundaries = [], updates = legacy_updates = []; this.#scheduled.length > 0; ) {
      flush_count++ > 1e3 && (this.#unlink(), infinite_loop_guard());
      for (let root of this.#resolve())
        try {
          this.#traverse(root, effects, render_effects, revealed_boundaries);
        } catch (e) {
          throw reset_branch(root, !0), this.#is_deferred() || this.discard(), e;
        }
      if (this.#scheduled.length > 0) {
        for (let effect2 of render_effects) this.#scheduled.push(effect2);
        for (let effect2 of effects) this.#scheduled.push(effect2);
        render_effects.length = 0, effects.length = 0;
      }
    }
    if (current_batch = null, updates.length > 0) {
      var batch = _Batch.ensure();
      for (let e of updates)
        (e.f & 1024) === 0 && batch.schedule(e, 4096);
    }
    if (collected_effects = null, legacy_updates = null, this.merge_requests.size > 0) {
      let sorted = Array.from(this.merge_requests, merge_target).filter((batch2) => batch2 !== this && batch2.linked && !batch2.is_fork).sort((a, b) => a.id - b.id);
      if (this.merge_requests.clear(), sorted.length > 0) {
        this.#defer_effects(render_effects), this.#defer_effects(effects);
        for (var target = this, i = sorted.length; i--; ) {
          var earlier = sorted[i];
          earlier !== target && (earlier.#merge(target), target = earlier);
        }
        target.#process();
        return;
      }
    }
    if (this.#is_deferred()) {
      this.#defer_effects(render_effects), this.#defer_effects(effects);
      for (let e of this.#skipped_branches)
        reset_branch(e);
      updates.length > 0 && /** @type {unknown} */
      current_batch.#process();
      return;
    }
    let earlier_batch = this.#find_earlier_batch();
    if (earlier_batch) {
      this.#defer_effects(render_effects), this.#defer_effects(effects), earlier_batch.#merge(this), earlier_batch.#process();
      return;
    }
    for (let boundary of revealed_boundaries)
      boundary.is_pending && boundary.is_ready(this) && boundary.reveal(effects);
    this.#release_records(!0), this.#run_callbacks(this.#commit_callbacks);
    var earlier_writes = flush_queued_effects(render_effects);
    custom_renderers_flag && before_effects !== null && before_effects(), flush_queued_effects(effects, earlier_writes), this.#deferred?.resolve();
    var next_batch = (
      /** @type {Batch | null} */
      /** @type {unknown} */
      current_batch
    );
    if (this.#pending.size === 0 && (this.#scheduled.length === 0 || next_batch !== null) && this.#unlink(), async_mode_flag && this.commit_writes(), next_batch !== null) {
      if (teardown_context !== null)
        for (let [source2, snapshot2] of teardown_context.reverted_values)
          next_batch.previous.get(source2)?.wv !== snapshot2.wv && teardown_context.reverted_values.delete(source2);
      next_batch.#process();
    }
  }
  /**
   * Schedule the reactions that (transitively) depend on `sources` for re-validation in this batch.
   * They are marked MAYBE_DIRTY, which means that they are only re-run if the exact versions
   * of their dependencies differ from the ones they last saw (in this batch's view of the world)
   * @param {Iterable<Source>} sources
   */
  #revalidate(sources) {
    var values = new Set(sources), effects = /* @__PURE__ */ new Set();
    for (let value of values) {
      var reactions = value.reactions;
      if (reactions !== null)
        for (let reaction of reactions) {
          var flags = reaction.f;
          if ((flags & 2) !== 0) {
            (flags & 1024) !== 0 && set_signal_status(reaction, 4096), values.add(
              /** @type {Derived} */
              reaction
            );
            continue;
          }
          var effect2 = (
            /** @type {Effect} */
            reaction
          );
          effects.has(effect2) || (effects.add(effect2), (flags & 155648) === 0 && // branches that this batch skips don't exist in its view of the world
          !this.is_skipped(effect2) && this.schedule(effect2, 4096));
        }
    }
  }
  /**
   * Traverse the effect tree, executing effects or stashing
   * them for later execution as appropriate
   * @param {Effect} root
   * @param {Effect[]} effects
   * @param {Effect[]} render_effects
   * @param {Boundary[]} revealed_boundaries
   */
  #traverse(root, effects, render_effects, revealed_boundaries) {
    root.f ^= 1024;
    for (var effect2 = root.first; effect2 !== null; ) {
      var flags = effect2.f, is_branch = (flags & 96) !== 0, is_skippable_branch = is_branch && (flags & 1024) !== 0, skip = is_skippable_branch || (flags & 139264) !== 0 || this.#skips(effect2);
      if (!skip && effect2.fn !== null) {
        is_branch ? effect2.f ^= 1024 : (flags & 4) !== 0 ? effects.push(effect2) : async_mode_flag && (flags & 16777224) !== 0 ? render_effects.push(effect2) : is_dirty(effect2) && update_effect(effect2);
        var child2 = effect2.first;
        if (child2 !== null) {
          effect2 = child2;
          continue;
        }
      }
      for (; effect2 !== null; ) {
        if ((effect2.f & 16512) === 128) {
          var boundary = (
            /** @type {Boundary} */
            effect2.b
          );
          boundary.is_pending && boundary.is_ready(this) && revealed_boundaries.push(boundary);
        }
        var next2 = effect2.next;
        if (next2 !== null) {
          effect2 = next2;
          break;
        }
        effect2 = effect2.parent;
      }
    }
  }
  #find_earlier_batch() {
    for (var batch = this.prev; batch !== null; batch = batch.prev)
      if (!batch.is_fork && this.#intersects(batch)) return batch;
    return null;
  }
  /** @param {Batch} batch */
  #intersects(batch) {
    for (let source2 of this.current.keys())
      if (batch.current.has(source2)) return !0;
    return !1;
  }
  /**
   * Take over the writes of a later batch that depends on this one, because it finished first
   * or aborted one of this batch's runs.
   * Values it produced (async results, deriveds) remain valid; block effects select their
   * branches again for this batch, if needed (their selections are keyed by batch)
   * @param {Batch} batch
   */
  #merge(batch) {
    for (let target of batch.merge_requests) this.merge_requests.add(target);
    batch.merge_requests.clear();
    for (let [source2, entry] of batch.current) {
      if (!this.previous.has(source2)) {
        let previous = batch.previous.get(source2);
        previous && this.previous.set(source2, previous);
      }
      this.current.set(source2, entry);
    }
    for (let [effect2, deferred2] of batch.async_deriveds) {
      let d = this.async_deriveds.get(effect2);
      d && d !== deferred2 && deferred2.promise.then(d.resolve, d.reject), this.async_deriveds.set(effect2, deferred2);
    }
    for (let [reaction, record] of batch.records ?? [])
      if ((reaction.f & 16) === 0) {
        var replaced = this.records?.get(reaction);
        (this.records ??= /* @__PURE__ */ new Map()).set(reaction, record), batch.records?.delete(reaction), abort_record(reaction, replaced);
      }
    if (batch.async_deriveds.clear(), batch.#pending.size > 0) {
      for (let [effect2, count] of batch.#pending)
        this.#pending.set(effect2, (this.#pending.get(effect2) ?? 0) + count);
      batch.#pending.clear();
    }
    if (batch.#unprocessed.size > 0) {
      for (let [effect2, status] of batch.#unprocessed)
        this.#add_dirty_effect(effect2, status);
      batch.#unprocessed.clear();
    }
    let discard = () => batch.discard();
    this.oncommit(discard), this.ondiscard(discard), batch.merged = this, batch.#unlink();
  }
  /**
   * Note that `effect` needs to be (re-)validated, without downgrading a DIRTY effect to MAYBE_DIRTY
   * @param {Effect} effect
   * @param {number} status DIRTY or MAYBE_DIRTY
   */
  #add_dirty_effect(effect2, status) {
    (status === 2048 || this.#unprocessed.get(effect2) !== 2048) && this.#unprocessed.set(effect2, status);
  }
  /**
   * @param {Effect[]} effects
   */
  #defer_effects(effects) {
    for (var i = 0; i < effects.length; i += 1) {
      var effect2 = effects[i], status = effect2.f & 6144;
      status !== 0 && (this.#add_dirty_effect(effect2, effect2.dv === null ? 4096 : status), set_signal_status(effect2, 1024));
    }
  }
  /**
   * Associate a change to a given source with the current
   * batch, noting its previous and current values (and versions).
   * The source itself is only updated if this is its latest value, i.e. unless this
   * is a fork (whose writes only exist in its own view of the world) or a later batch
   * already wrote to it (e.g. because an async expression resolved in an earlier batch).
   * @param {Value} source
   * @param {any} value
   * @param {boolean} same whether the value equals the real one (it can, if this batch sees another one, see
   * `internal_set`) — in which case the source isn't updated, and the value gets the same version
   * @param {boolean} [error] whether the value is an error (thrown when read, see `runtime.js#get`)
   * @param {Effect[] | null} [updated_during_traversal]
   */
  capture(source2, value, same, error = !1, updated_during_traversal = null) {
    var previous = this.previous.get(source2), later = later_writer(this, source2);
    previous === void 0 && (previous = later?.previous.get(source2), previous === void 0 && source2.v !== UNINITIALIZED && (previous = {
      v: source2.v,
      wv: source2.wv,
      e: (source2.f & 8388608) !== 0,
      assigned: (source2.f & 67108864) !== 0
    }), previous !== void 0 && this.previous.set(source2, previous)), is_destroying_effect ? teardown_write(source2) : previous !== void 0 && (teardown_context?.reverted_values.set(source2, previous), teardown_context?.derived_cache.clear());
    var entry = {
      v: value,
      wv: same ? source2.wv : increment_write_version(),
      e: error,
      // a write to a derived is an assignment
      assigned: (source2.f & 2) !== 0,
      async: (source2.f & 4194304) !== 0
    };
    this.current.set(source2, entry), batch_values?.set(source2, entry), view_cache?.delete(
      /** @type {Derived} */
      source2
    ), !this.is_fork && later !== null && later.previous.set(source2, entry), !this.is_fork && !same && later === null && apply(source2, entry), same || (seen = null, count_deps = 0, this.mark(source2, 2048, updated_during_traversal), seen = null, eager_effects.size > 0 && !eager_effects_deferred && flush_eager_effects());
  }
  deactivate() {
    current_batch = processing_batch = batch_values = view_cache = view_batch = null;
  }
  flush() {
    var context = push_teardown_context(this.previous);
    try {
      if (this.#process(), rebases.size > 0)
        for (let batch of rebases)
          rebases.delete(batch), batch.linked && !batch.#decrement_queued && batch.#process();
    } finally {
      if (flush_count = 0, last_scheduled_effect = null, collected_effects = null, legacy_updates = null, teardown_context = context.parent, this.deactivate(), false_default) {
        for (let source2 of source_stacks)
          source2.updated = null;
        source_stacks.clear();
      }
    }
  }
  discard() {
    this.#run_callbacks(this.#discard_callbacks);
    for (let deferred2 of this.async_deriveds.values())
      deferred2.reject(OBSOLETE);
    this.#unlink(), this.#deferred?.resolve();
  }
  #commit() {
    for (let batch = first_batch; batch !== null; batch = batch.next)
      if (batch !== this) {
        var is_earlier = batch.id < this.id, sources = [];
        for (let [source2, entry] of this.current) {
          var own = batch.current.get(source2);
          if (own !== void 0)
            if (is_earlier && entry.v !== own.v && !own.async)
              batch.current.set(source2, entry);
            else
              continue;
          sources.push(source2);
        }
        if (!(!batch.started || batch.current.size === 0)) {
          var obsolete = !0;
          for (let source2 of batch.current.keys())
            if (!this.current.has(source2)) {
              obsolete = !1;
              break;
            }
          obsolete && is_earlier ? batch.discard() : sources.length > 0 && (false_default && !batch.#decrement_queued && invariant(batch.#scheduled.length === 0, "Batch has scheduled effects"), batch.#revalidate(sources), rebases.add(batch));
        }
      }
    this.current.clear(), this.previous.clear();
  }
  /**
   * Make this batch's writes the real world, part of every other batch's view (see `#commit`).
   * If it has pending work it stays linked, coming first from now on (see `#move_to_front`)
   */
  commit_writes() {
    this.#commit(), this.linked && this.prev !== null && this.#move_to_front();
  }
  /**
   * @param {Effect} effect
   */
  increment(effect2) {
    var count = this.#pending.get(effect2) ?? 0;
    this.#pending.set(effect2, count + 1);
  }
  /**
   * @param {Effect} effect
   */
  decrement(effect2) {
    var count = this.#pending.get(effect2) ?? 0;
    count === 1 ? this.#pending.delete(effect2) : this.#pending.set(effect2, count - 1), !this.#decrement_queued && (this.#decrement_queued = !0, queue_micro_task(() => {
      this.#decrement_queued = !1, this.linked && this.flush();
    }));
  }
  /** @param {(batch: Batch) => void} fn */
  oncommit(fn) {
    this.#commit_callbacks.set(fn, eager_block_writes?.size ? eager_block_writes : null);
  }
  /** @param {(batch: Batch) => void} fn */
  ondiscard(fn) {
    this.#discard_callbacks.set(fn, eager_block_writes?.size ? eager_block_writes : null);
  }
  /**
   * Runs the commit/discard callbacks.
   * A callback that was added while block effects re-ran eagerly detaches the branches that such a run
   * deselected (see `flush_queued_effects`). The teardowns it causes see the writes that applied in the
   * run's view, like they would have during the run: of the values that the current teardown context
   * reverts, the writes of the effects that ran before the one whose writes caused the run are restored.
   * Callbacks run untracked, because a batch can commit while an effect runs (e.g. one that calls `flushSync`)
   * @param {Map<(batch: Batch) => void, Map<Value, BatchValue> | null>} callbacks
   */
  #run_callbacks(callbacks) {
    untrack(() => {
      for (let [fn, writes] of callbacks) {
        if (writes === null) {
          fn(this);
          continue;
        }
        var context = push_teardown_context();
        for (let [source2, write] of writes)
          context.reverted_values.has(source2) && context.reverted_values.set(source2, write);
        try {
          fn(this);
        } finally {
          teardown_context = context.parent;
        }
      }
    }), callbacks.clear();
  }
  settled() {
    return (this.#deferred ??= deferred()).promise;
  }
  static ensure() {
    if (current_batch === null) {
      let batch = current_batch = new _Batch();
      processing_batch === null && !is_flushing_sync && queue_micro_task(() => {
        batch.started || batch.flush();
      });
    }
    return current_batch;
  }
  /**
   * Make this the current batch, and its view of the world the active one
   */
  activate() {
    if (current_batch = this, view_batch = this, last_processed !== this.id && (last_processed = null), !async_mode_flag) {
      batch_values = view_cache = null;
      return;
    }
    if (this.folded = [], !this.is_fork && this.prev === null && this.next === null) {
      batch_values = view_cache = null;
      return;
    }
    batch_values = new Map(this.current), view_cache = /* @__PURE__ */ new Map();
    for (let batch = first_batch; batch !== null; batch = batch.next)
      if (!(batch === this || batch.is_fork))
        if (batch.id < this.id && batch.#intersects(this))
          this.folded.push(batch);
        else
          for (let [source2, previous] of batch.previous)
            !batch_values.has(source2) && ((source2.f & 2) !== 0 || !source2.equals(previous.v)) && batch_values.set(source2, previous);
  }
  /**
   * The record of `reaction` in this batch's view of the world, i.e. that of this batch
   * or of the (earlier) batches whose changes it includes, if any
   * @param {Reaction} reaction
   * @returns {BatchRecord | undefined}
   */
  get_record(reaction) {
    for (var record = this.records?.get(reaction), i = this.folded.length; record === void 0 && i--; )
      record = this.folded[i].records?.get(reaction);
    return record;
  }
  /**
   * @param {Value} signal
   * @param {number} status should be DIRTY or MAYBE_DIRTY
   * @param {Effect[] | null} [updated_during_traversal]
   * @returns {void}
   */
  mark(signal, status, updated_during_traversal = null) {
    var reactions = signal.reactions;
    if (reactions !== null) {
      var runes = is_runes(), length = reactions.length;
      if (count_deps += length, count_deps > 1e5 && seen === null && (seen = /* @__PURE__ */ new Set()), seen !== null) {
        if (seen.has(signal)) return;
        seen.add(signal);
      }
      for (var i = 0; i < length; i++) {
        var reaction = reactions[i], flags = reaction.f;
        if (!(!runes && reaction === active_effect)) {
          var not_dirty = (flags & 2048) === 0;
          if ((flags & 131072) !== 0)
            this.is_fork || (not_dirty && set_signal_status(reaction, status), eager_effects.add(
              /** @type {Effect} */
              reaction
            ));
          else if ((flags & 2) !== 0) {
            not_dirty && set_signal_status(reaction, status);
            var derived2 = (
              /** @type {Derived} */
              reaction
            );
            view_cache?.delete(derived2), this.mark(derived2, 4096, updated_during_traversal);
          } else if (not_dirty) {
            var effect2 = (
              /** @type {Effect} */
              reaction
            );
            (flags & 16) !== 0 && eager_block_effects !== null && (set_signal_status(effect2, status), eager_block_effects.add(effect2)), updated_during_traversal !== null ? (set_signal_status(effect2, status), updated_during_traversal.push(effect2)) : this.schedule(effect2, status);
          }
        }
      }
    }
  }
  /**
   * Schedule `effect` to be (re-)validated when this batch is processed. Its status only applies right
   * away if this is the batch that the current flush processes (next), or if a reaction is running (which
   * may have read what changed). Otherwise, whether an effect is dirty depends on this batch's view of the
   * world, so other batches (which may be processed first) mustn't see it: it is kept with this batch until
   * it is processed, and then checked against the exact versions of its dependencies (i.e. MAYBE_DIRTY)
   * @param {Effect} effect
   * @param {number} status DIRTY or MAYBE_DIRTY
   */
  schedule(effect2, status) {
    if (last_scheduled_effect = effect2, view_batch !== null && view_batch !== this && effect2 !== active_reaction) {
      this.#add_dirty_effect(
        effect2,
        (effect2.f & 2097152) !== 0 ? status : 4096
      );
      return;
    }
    if (processing_batch !== this && active_reaction === null) {
      if (this.prev !== null || this.next !== null || this.is_fork) {
        this.#add_dirty_effect(effect2, 4096);
        return;
      }
      status = 4096;
    }
    this.#add(effect2, status);
  }
  /**
   * @param {Effect} effect
   * @param {number} status
   */
  #add(effect2, status) {
    (effect2.f & 2048) === 0 && set_signal_status(effect2, status), this.#scheduled.push(effect2);
  }
  /** @param {boolean} commit whether the effects' runs become global state */
  #release_records(commit = !1) {
    var records = this.records;
    if (records !== null) {
      this.records = null;
      for (let [reaction, record] of records) {
        if ((reaction.f & 16386) === 0) {
          var replaced = (reaction.f & 4194304) !== 0 && is_outdated(record) && !is_outdated(reaction);
          if (commit && !replaced && (promote_record(reaction, record), (reaction.f & 4194304) !== 0))
            for (let b = first_batch; b !== null; b = b.next)
              b !== this && !b.records?.has(reaction) && b.async_deriveds.get(
                /** @type {Effect} */
                reaction
              )?.reject(OBSOLETE);
          prune_edges(reaction);
        }
        abort_record(reaction, record);
      }
    }
  }
  /**
   * A batch that committed while work inside a pending boundary was in flight stays linked, so that the results
   * arrive in it. Its writes are the real world now, part of every other batch's view, so it comes first from now
   * on: This ensures logic about who sees what values in intersects/merge/etc work correctly.
   */
  #move_to_front() {
    var prev = (
      /** @type {Batch} */
      this.prev
    ), first = (
      /** @type {Batch} */
      first_batch
    );
    prev.next = this.next, this.next === null ? last_batch = prev : this.next.prev = prev, this.prev = null, this.next = first, first.prev = first_batch = this, this.id = first.id - 1;
  }
  #unlink() {
    if (this.linked) {
      var prev = this.prev, next2 = this.next;
      prev === null ? first_batch = next2 : prev.next = next2, next2 === null ? last_batch = prev : next2.prev = prev, this.linked = !1, this.#release_records();
    }
  }
}, before_effects = null;
function set_before_effects(fn) {
  before_effects = fn;
}
function flushSync(fn) {
  var was_flushing_sync = is_flushing_sync, previous_view_batch = view_batch !== current_batch ? view_batch : null;
  is_flushing_sync = !0;
  try {
    var result;
    for (fn && (flushSync(), result = fn()); ; ) {
      if (flush_tasks(), current_batch === null)
        return (
          /** @type {T} */
          result
        );
      current_batch.flush();
    }
  } finally {
    is_flushing_sync = was_flushing_sync, view_batch = previous_view_batch;
  }
}
function infinite_loop_guard() {
  if (false_default) {
    var updates = /* @__PURE__ */ new Map();
    for (
      let source2 of
      /** @type {Batch} */
      current_batch.current.keys()
    )
      for (let [stack2, update2] of source2.updated ?? []) {
        var entry = updates.get(stack2);
        entry || (entry = { error: update2.error, count: 0 }, updates.set(stack2, entry)), entry.count += update2.count;
      }
    for (let update2 of updates.values())
      update2.error && console.error(update2.error);
  }
  try {
    effect_update_depth_exceeded();
  } catch (error) {
    false_default && define_property(error, "stack", { value: "" }), invoke_error_boundary(error, last_scheduled_effect);
  }
}
var eager_block_effects = null, eager_block_writes = null;
function defer_if_pending(effect2) {
  if ((effect2.f & 32772) !== 4) return !1;
  for (var boundary = effect2.b; boundary !== null; boundary = boundary.parent)
    if (boundary.is_pending)
      return boundary.defer_effect(effect2), !0;
  return !1;
}
function flush_queued_effects(effects, earlier_writes = null) {
  var length = effects.length;
  if (length === 0) return earlier_writes;
  for (var i = 0; i < length; ) {
    var effect2 = effects[i++];
    if ((effect2.f & 24576) === 0 && // don't run if already marked for destruction in the next batch
    // (this can happen if an effect writes a value which e.g. makes an if block falsy)
    !current_batch?.is_skipped(effect2) && !defer_if_pending(effect2) && is_dirty(effect2)) {
      eager_block_effects = /* @__PURE__ */ new Set();
      var version = write_version;
      if (update_effect(effect2), effect2.edges === null && effect2.first === null && effect2.nodes === null && effect2.teardown === null && effect2.ac === null && unlink_effect(effect2), eager_block_effects?.size > 0) {
        var previous_teardown_context = teardown_context;
        earlier_writes !== null && push_teardown_context(earlier_writes);
        var previous_eager_block_writes = eager_block_writes;
        eager_block_writes = new Map(earlier_writes);
        try {
          for (let e of eager_block_effects) {
            if ((e.f & 24576) !== 0 || current_batch?.is_skipped(e)) continue;
            let ordered_effects = [e], ancestor = e.parent;
            for (; ancestor !== null; )
              eager_block_effects.has(ancestor) && (eager_block_effects.delete(ancestor), ordered_effects.push(ancestor)), ancestor = ancestor.parent;
            for (let j = ordered_effects.length - 1; j >= 0; j--) {
              let e2 = ordered_effects[j];
              (e2.f & 24576) !== 0 || current_batch?.is_skipped(e2) || update_effect(e2);
            }
          }
        } finally {
          eager_block_writes = previous_eager_block_writes, eager_block_effects?.clear(), teardown_context = previous_teardown_context;
        }
      }
      if (write_version !== version && current_batch !== null)
        for (let [value, write] of current_batch.current)
          write.wv > version && (earlier_writes ??= /* @__PURE__ */ new Map()).set(value, write);
    }
  }
  return eager_block_effects = null, earlier_writes;
}
function later_writer(batch, source2) {
  for (var later = batch.next; later !== null && (later.is_fork || !later.current.has(source2)); )
    later = later.next;
  return later;
}
function apply(source2, entry) {
  source2.v = entry.v, source2.wv = entry.wv, source2.f = entry.e ? source2.f | 8388608 : source2.f & -8388609, (source2.f & 2) !== 0 && (source2.f |= 67108864, update_derived_status(
    /** @type {Derived} */
    source2
  ));
}
var eager_versions = [];
function eager_flush() {
  flushSync(() => {
    let eager2 = eager_versions;
    eager_versions = [];
    for (let version of eager2)
      update(version);
  });
}
var version_map = /* @__PURE__ */ new Map();
function eager(fn) {
  var initial = !0, value = (
    /** @type {T} */
    void 0
  );
  if (active_reaction === null)
    return fn();
  let parent = active_reaction, version = version_map.get(parent) ?? source(0);
  return version_map.set(parent, version), false_default && (version.label ??= "$state.eager version"), teardown(() => {
    parent.f & 33554432 && version_map.delete(parent);
  }), get(version), in_real_world(() => {
    eager_effect(() => {
      if (initial) {
        value = fn();
        return;
      }
      eager_versions.length === 0 && queue_micro_task(eager_flush), eager_versions.push(version);
    });
  }), initial = !1, value;
}
function in_real_world(fn) {
  var previous_batch_values = batch_values, previous_view_cache = view_cache, previous_view_batch = view_batch;
  batch_values = view_cache = view_batch = null;
  try {
    fn();
  } finally {
    batch_values = previous_batch_values, view_cache = previous_view_cache, view_batch = previous_view_batch;
  }
}
function reset_branch(effect2, force = !1) {
  if (!(!force && (effect2.f & 32) !== 0 && (effect2.f & 1024) !== 0)) {
    set_signal_status(effect2, 1024);
    for (var e = effect2.first; e !== null; )
      reset_branch(e, force), e = e.next;
  }
}
function fork(fn) {
  async_mode_flag || experimental_async_required("fork"), (active_effect !== null || current_batch?.is_fork) && fork_timing();
  var batch = flushSync(() => {
    var batch2 = Batch.ensure();
    batch2.is_fork = !0;
    try {
      in_real_world(() => {
        batch_values = /* @__PURE__ */ new Map(), view_cache = /* @__PURE__ */ new Map(), view_batch = batch2, fn();
      });
    } catch (error) {
      throw batch2.discard(), batch2.deactivate(), error;
    }
    return batch2;
  }), settled2 = batch.settled(), committed = !1, discarded = !1;
  return {
    commit: async () => {
      if (committed) {
        await settled2;
        return;
      }
      if (discarded && fork_discarded(), !batch.linked) {
        committed = !0;
        return;
      }
      committed = !0, batch.is_fork = !1;
      for (let [source2, entry] of batch.current) {
        let later = later_writer(batch, source2);
        later !== null ? later.previous.set(source2, entry) : (batch.mark(source2, 4096), apply(source2, entry));
      }
      eager_effects.size > 0 && flush_eager_effects(), batch.flush(), await settled2;
    },
    discard: () => {
      !committed && batch.linked && batch.discard(), discarded = !0;
    }
  };
}

// node_modules/svelte/src/internal/client/reactivity/effects.js
function validate_effect(rune) {
  active_effect === null && (active_reaction === null && effect_orphan(rune), effect_in_unowned_derived()), is_destroying_effect && effect_in_teardown(rune);
}
function push_effect(effect2, parent_effect) {
  var parent_last = parent_effect.last;
  parent_last === null ? parent_effect.last = parent_effect.first = effect2 : (parent_last.next = effect2, effect2.prev = parent_last, parent_effect.last = effect2);
}
function create_effect(type, fn) {
  var parent = active_effect;
  if (false_default)
    for (; parent !== null && (parent.f & 131072) !== 0; )
      parent = parent.parent;
  parent !== null && (parent.f & 8192) !== 0 && (type |= 8192);
  var effect2 = {
    ctx: component_context,
    deps: null,
    dv: null,
    edges: null,
    nodes: null,
    f: type | 2048 | 512,
    first: null,
    fn,
    last: null,
    next: null,
    parent,
    b: parent && parent.b,
    prev: null,
    teardown: null,
    wv: 0,
    ac: null,
    r: current_renderer
  };
  false_default && (effect2.component_function = dev_current_component_function);
  var e = effect2;
  if ((type & 4) !== 0)
    collected_effects !== null ? collected_effects.push(effect2) : Batch.ensure().schedule(effect2, 2048);
  else if (fn !== null) {
    try {
      update_effect(effect2);
    } catch (e2) {
      throw destroy_effect(effect2), e2;
    }
    e.edges === null && e.teardown === null && e.nodes === null && e.first === e.last && // either `null`, or a singular child
    (e.f & 524288) === 0 && (e = e.first, (type & 16) !== 0 && (type & 65536) !== 0 && e !== null && (e.f |= 65536));
  }
  if (e !== null && (e.parent = parent, parent !== null && push_effect(e, parent), active_reaction !== null && (active_reaction.f & 2) !== 0 && (type & 64) === 0)) {
    var derived2 = (
      /** @type {Derived} */
      active_reaction
    );
    (derived2.effects ??= []).push(e);
  }
  return effect2;
}
function effect_tracking() {
  return active_reaction !== null && !untracking;
}
function teardown(fn) {
  let effect2 = create_effect(8, null);
  return set_signal_status(effect2, 1024), effect2.teardown = fn, effect2;
}
function user_effect(fn) {
  validate_effect("$effect"), false_default && define_property(fn, "name", {
    value: "$effect"
  });
  var flags = (
    /** @type {Effect} */
    active_effect.f
  ), defer = !active_reaction && (flags & 32) !== 0 && component_context !== null && !component_context.i;
  if (defer) {
    var context = (
      /** @type {ComponentContext} */
      component_context
    );
    (context.e ??= []).push(fn);
  } else
    return create_user_effect(fn);
}
function create_user_effect(fn) {
  return create_effect(1048580, fn);
}
function user_pre_effect(fn) {
  return validate_effect("$effect.pre"), false_default && define_property(fn, "name", {
    value: "$effect.pre"
  }), create_effect(1048584, fn);
}
function eager_effect(fn) {
  return create_effect(131072, fn);
}
function effect_root(fn) {
  Batch.ensure();
  let effect2 = create_effect(524352, fn);
  return () => {
    destroy_effect(effect2);
  };
}
function component_root(fn) {
  Batch.ensure();
  let effect2 = create_effect(524352, fn);
  return (options = {}) => new Promise((fulfil) => {
    options.outro ? pause_effect(effect2, () => {
      destroy_effect(effect2), fulfil(void 0);
    }) : (destroy_effect(effect2), fulfil(void 0));
  });
}
function effect(fn) {
  return create_effect(4, fn);
}
function legacy_pre_effect(deps, fn) {
  var context = (
    /** @type {ComponentContextLegacy} */
    component_context
  ), token = { effect: null, ran: !1, deps };
  context.l.$.push(token), token.effect = render_effect(() => {
    if (deps(), !token.ran) {
      token.ran = !0;
      var effect2 = (
        /** @type {Effect} */
        active_effect
      );
      try {
        set_active_effect(effect2.parent), untrack(fn);
      } finally {
        set_active_effect(effect2);
      }
    }
  });
}
function legacy_pre_effect_reset() {
  var context = (
    /** @type {ComponentContextLegacy} */
    component_context
  );
  render_effect(() => {
    for (var token of context.l.$) {
      token.deps();
      var effect2 = token.effect;
      (effect2.f & 1024) !== 0 && effect2.edges !== null && set_signal_status(effect2, 4096), is_dirty(effect2) && update_effect(effect2), token.ran = !1;
    }
  });
}
function async_effect(fn) {
  return create_effect(4718592, fn);
}
function render_effect(fn, flags = 0) {
  return create_effect(8 | flags, fn);
}
function template_effect(fn, sync = [], async = [], blockers = []) {
  flatten(blockers, sync, async, (values) => {
    create_effect(8, () => {
      fn(...values.map(get));
    });
  });
}
function deferred_template_effect(fn, sync = [], async = [], blockers = []) {
  flatten(blockers, sync, async, (values) => {
    create_effect(4, () => fn(...values.map(get)));
  });
}
function block(fn, flags = 0) {
  var effect2 = create_effect(16 | flags, fn);
  return false_default && (effect2.dev_stack = dev_stack), effect2;
}
function managed(fn, flags = 0) {
  var effect2 = create_effect(16777216 | flags, fn);
  return false_default && (effect2.dev_stack = dev_stack), effect2;
}
function branch(fn) {
  return create_effect(524320, fn);
}
function execute_effect_teardown(effect2) {
  var teardown2 = effect2.teardown;
  if (teardown2 !== null) {
    let previously_destroying_effect = is_destroying_effect, previous_teardown_version = teardown_version, previous_teardown_context = teardown_context, previous_reaction = active_reaction;
    set_is_destroying_effect(!0), set_active_reaction(null);
    try {
      teardown2.call(null);
    } catch (error) {
      invoke_error_boundary(error, effect2.parent);
    } finally {
      set_is_destroying_effect(previously_destroying_effect, previous_teardown_version), set_teardown_context(previous_teardown_context), set_active_reaction(previous_reaction);
    }
  }
}
function destroy_effect_children(signal, remove_dom = !1) {
  var effect2 = signal.first;
  for (signal.first = signal.last = null; effect2 !== null; ) {
    abort_all(effect2);
    var next2 = effect2.next;
    (effect2.f & 64) !== 0 ? effect2.parent = null : destroy_effect(effect2, remove_dom), effect2 = next2;
  }
}
function destroy_block_effect_children(signal) {
  for (var effect2 = signal.first; effect2 !== null; ) {
    var next2 = effect2.next;
    (effect2.f & 32) === 0 && destroy_effect(effect2), effect2 = next2;
  }
}
function destroy_effect(effect2, remove_dom = !0) {
  var removed = !1, previous_renderer = custom_renderers_flag ? push_renderer(effect2.r) : null;
  try {
    (remove_dom || (effect2.f & 262144) !== 0) && effect2.nodes !== null && effect2.nodes.end !== null && (remove_effect_dom(
      effect2.nodes.start,
      /** @type {TemplateNode} */
      effect2.nodes.end
    ), removed = !0), effect2.f |= 33554432, destroy_effect_children(effect2, remove_dom && !removed), remove_reactions(effect2);
    var transitions = effect2.nodes && effect2.nodes.t;
    if (transitions !== null)
      for (let transition of transitions)
        transition.stop();
    execute_effect_teardown(effect2);
  } finally {
    custom_renderers_flag && pop_renderer(previous_renderer);
  }
  effect2.f ^= 33554432, effect2.f |= 16384;
  var parent = effect2.parent;
  parent !== null && parent.first !== null && unlink_effect(effect2), false_default && (effect2.component_function = null), effect2.next = effect2.prev = effect2.teardown = effect2.ctx = effect2.deps = effect2.dv = effect2.edges = effect2.fn = effect2.nodes = effect2.ac = effect2.b = effect2.r = null;
}
function remove_effect_dom(node, end) {
  for (; node !== null; ) {
    var next2 = node === end ? null : get_next_sibling(node);
    remove_node(
      /** @type {ChildNode} */
      node
    ), node = next2;
  }
}
function unlink_effect(effect2) {
  var parent = effect2.parent, prev = effect2.prev, next2 = effect2.next;
  prev !== null && (prev.next = next2), next2 !== null && (next2.prev = prev), parent !== null && (parent.first === effect2 && (parent.first = next2), parent.last === effect2 && (parent.last = prev));
}
function pause_effect(effect2, callback, destroy = !0) {
  var transitions = [];
  effect2.f |= 256, pause_children(effect2, transitions, !0);
  var fn = () => {
    destroy && destroy_effect(effect2), callback && callback();
  }, remaining = transitions.length;
  if (remaining > 0) {
    var context = teardown_context, check = () => {
      if (!--remaining) {
        var previous_context = teardown_context;
        try {
          context && (context.derived_cache.clear(), set_teardown_context(context)), fn();
        } finally {
          context && set_teardown_context(previous_context);
        }
      }
    };
    for (var transition of transitions)
      transition.out(check);
  } else
    fn();
}
function pause_children(effect2, transitions, local) {
  if ((effect2.f & 8192) === 0) {
    effect2.f ^= 8192;
    var t = effect2.nodes && effect2.nodes.t;
    if (t !== null)
      for (let transition of t)
        (transition.is_global || local) && transitions.push(transition);
    for (var child2 = effect2.first; child2 !== null; ) {
      var sibling2 = child2.next;
      if ((child2.f & 64) === 0) {
        var transparent = (child2.f & 65536) !== 0 || // If this is a branch effect without a block effect parent,
        // it means the parent block effect was pruned. In that case,
        // transparency information was transferred to the branch effect.
        (child2.f & 32) !== 0 && (effect2.f & 16) !== 0;
        pause_children(child2, transitions, transparent ? local : !1);
      }
      child2 = sibling2;
    }
  }
}
function resume_effect(effect2) {
  effect2.f &= -257, resume_children(effect2, !0);
}
function resume_children(effect2, local) {
  if ((effect2.f & 256) === 0 && (effect2.f & 8192) !== 0) {
    effect2.f ^= 8192, (effect2.f & 1024) === 0 && Batch.ensure().schedule(effect2, 4096);
    for (var child2 = effect2.first; child2 !== null; ) {
      var sibling2 = child2.next, transparent = (child2.f & 65536) !== 0 || (child2.f & 32) !== 0;
      resume_children(child2, transparent ? local : !1), child2 = sibling2;
    }
    var t = effect2.nodes && effect2.nodes.t;
    if (t !== null)
      for (let transition of t)
        (transition.is_global || local) && transition.in();
  }
}
function move_effect(effect2, fragment) {
  if (effect2.nodes) {
    var previous_renderer = custom_renderers_flag ? push_renderer(effect2.r) : null;
    try {
      for (var node = effect2.nodes.start, end = effect2.nodes.end; node !== null; ) {
        var next2 = node === end ? null : get_next_sibling(node);
        append_child(fragment, node), node = next2;
      }
    } finally {
      custom_renderers_flag && pop_renderer(previous_renderer);
    }
  }
}

// node_modules/svelte/src/internal/client/legacy.js
var captured_signals = null;
function capture_signals(fn) {
  var previous_captured_signals = captured_signals;
  try {
    if (captured_signals = /* @__PURE__ */ new Set(), untrack(fn), previous_captured_signals !== null)
      for (var signal of captured_signals)
        previous_captured_signals.add(signal);
    return captured_signals;
  } finally {
    captured_signals = previous_captured_signals;
  }
}
function invalidate_inner_signals(fn) {
  for (var signal of capture_signals(fn))
    internal_set(signal, signal.v);
}

// node_modules/svelte/src/internal/client/dom/elements/misc.js
function autofocus(dom, value) {
  if (value) {
    let body = document.body;
    dom.autofocus = !0, queue_micro_task(() => {
      document.activeElement === body && dom.focus();
    });
  }
}
function remove_textarea_child(dom) {
  hydrating && get_first_child(dom) !== null && clear_text_content(dom);
}
var listening_to_form_reset = !1;
function add_form_reset_listener() {
  listening_to_form_reset || (listening_to_form_reset = !0, add_event_listener(
    document,
    "reset",
    (evt) => {
      Promise.resolve().then(() => {
        if (!evt.defaultPrevented)
          for (
            let e of
            /**@type {HTMLFormElement} */
            evt.target.elements
          )
            e[FORM_RESET_HANDLER]?.();
      });
    },
    // In the capture phase to guarantee we get noticed of it (no possibility of stopPropagation)
    { capture: !0 }
  ));
}

// node_modules/svelte/src/internal/client/dom/elements/bindings/shared.js
function listen(target, events, handler, call_handler_immediately = !0) {
  call_handler_immediately && handler();
  for (var name of events)
    target.addEventListener(name, handler);
  teardown(() => {
    for (var name2 of events)
      target.removeEventListener(name2, handler);
  });
}
function without_reactive_context(fn) {
  var previous_reaction = active_reaction, previous_effect = active_effect;
  set_active_reaction(null), set_active_effect(null);
  try {
    return fn();
  } finally {
    set_active_reaction(previous_reaction), set_active_effect(previous_effect);
  }
}
function listen_to_event_and_reset_event(element, event, handler, on_reset = handler) {
  element.addEventListener(event, () => without_reactive_context(handler));
  let prev = (
    /** @type {any} */
    element[FORM_RESET_HANDLER]
  );
  prev ? element[FORM_RESET_HANDLER] = () => {
    prev(), on_reset(!0);
  } : element[FORM_RESET_HANDLER] = () => on_reset(!0), add_form_reset_listener();
}

// node_modules/svelte/src/internal/client/runtime.js
var is_destroying_effect = !1, teardown_version = 0;
function set_is_destroying_effect(value, version = write_version) {
  is_destroying_effect = value, teardown_version = version;
}
var teardown_run = { v: void 0, wv: 0, e: !1, assigned: !1 }, active_reaction = null, untracking = !1;
function set_active_reaction(reaction) {
  active_reaction = reaction;
}
var active_effect = null;
function set_active_effect(effect2) {
  active_effect = effect2;
}
var current_sources = null;
function push_reaction_value(value) {
  active_reaction !== null && (!async_mode_flag && (active_reaction.f & 2097152) !== 0 || (active_reaction.f & 2) !== 0) && (current_sources ??= /* @__PURE__ */ new Set()).add(value);
}
var new_deps = null, skipped_deps = 0, untracked_writes = null;
function set_untracked_writes(value) {
  untracked_writes = value;
}
var write_version = 1, read_version = 0, update_version = read_version;
function set_update_version(value) {
  update_version = value;
}
function increment_write_version() {
  return ++write_version;
}
function version_of(signal) {
  return batch_values === null ? (update_derived(signal), signal.wv) : (signal.f & 2) !== 0 ? get_view(
    /** @type {Derived} */
    signal
  ).wv : (batch_values.get(signal) ?? signal).wv;
}
function is_current(deps, dv) {
  if (deps !== null) {
    for (var i = 0; i < dv.length; i++)
      if (version_of(deps[i]) !== dv[i]) return !1;
  }
  return !0;
}
function changed_since(deps, wv, unchanged = null) {
  if (deps !== null)
    for (var i = 0; i < deps.length; i++) {
      var dep = deps[i];
      if ((dep.f & 2) === 0) {
        if (version_of(dep) > wv) return !0;
        continue;
      }
      var run3 = (
        /** @type {Derived} */
        dep
      );
      if (batch_values === null ? update_derived(
        /** @type {Derived} */
        dep
      ) : run3 = get_view(
        /** @type {Derived} */
        dep
      ), run3.wv > wv) {
        var assigned = run3 === dep ? (dep.f & 67108864) !== 0 : !("dv" in run3);
        if (assigned) return !0;
        if (unchanged === null)
          unchanged = /* @__PURE__ */ new Set();
        else if (unchanged.has(run3))
          continue;
        if (changed_since(
          /** @type {DerivedRecord} */
          run3.deps,
          wv,
          unchanged
        ))
          return !0;
        unchanged.add(run3);
      }
    }
  return !1;
}
function run_of(reaction) {
  return (reaction.f & 4194320) !== 0 && view_batch?.get_record(reaction) || reaction;
}
function is_dirty(reaction) {
  var flags = reaction.f;
  if ((flags & 6144) === 0)
    return !1;
  if ((flags & 67108864) !== 0) {
    if (changed_since(reaction.deps, reaction.wv)) return !0;
  } else {
    if ((flags & 2048) !== 0)
      return !0;
    var run3 = run_of(reaction), deps = run3.deps, dv = run3.dv;
    if (dv === null) return !0;
    if (batch_values === null)
      for (var i = 0, length = dv.length; i < length; i++) {
        var dependency = (
          /** @type {Value[]} */
          deps[i]
        );
        if (update_derived(dependency), dependency.wv !== dv[i]) return !0;
      }
    else if (!is_current(deps, dv))
      return !0;
  }
  return (flags & 512) !== 0 && set_signal_status(reaction, 1024), !1;
}
function schedule_possible_effect_self_invalidation(signal, effect2, root = !0) {
  var reactions = signal.reactions;
  if (reactions !== null && !(!async_mode_flag && current_sources !== null && current_sources.has(signal)))
    for (var i = 0; i < reactions.length; i++) {
      var reaction = reactions[i];
      (reaction.f & 2) !== 0 ? schedule_possible_effect_self_invalidation(
        /** @type {Derived} */
        reaction,
        effect2,
        !1
      ) : effect2 === reaction && current_batch.schedule(
        /** @type {Effect} */
        reaction,
        root ? 2048 : 4096
      );
    }
}
var abort_controller = null;
function get_abort_signal(reaction) {
  return (reaction.f & 2097152) === 0 && get_abort_signal_after_await(), reaction.f |= 134217728, (abort_controller ??= new AbortController()).signal;
}
function abort_signal(controller) {
  controller && without_reactive_context(() => controller.abort(STALE_REACTION));
}
function abort_record(reaction, record, rerun = !1) {
  if (record?.ac) {
    for (var b = first_batch; b !== null; b = b.next)
      if (b.records?.get(reaction) === record && !(rerun && (b === view_batch || supersedes(current_batch, b))))
        return;
    abort_signal(record.ac), record.ac = null;
  }
}
function abort_replaced(reaction, flags) {
  var batch = current_batch, is_derived = (flags & 2) !== 0;
  if (!(is_derived && is_destroying_effect)) {
    var real = batch === null || !batch.is_fork;
    abort_record(reaction, view_batch?.records?.get(reaction), !0), (is_derived ? batch_values === null || real && is_outdated(reaction) : real) && abort_signal(reaction.ac);
    for (var b = first_batch; b !== null; b = b.next)
      supersedes(batch, b) && abort_record(reaction, b.records?.get(reaction), !0);
  }
}
function set_global_signal(reaction) {
  abort_signal(reaction.ac), reaction.ac = abort_controller;
}
function is_outdated({ deps, dv }) {
  return dv === null || dv.some((version, i) => (
    /** @type {Value[]} */
    deps[i].wv !== version
  ));
}
function supersedes(batch, other) {
  return batch !== null && !batch.is_fork && !other.is_fork && other.id < batch.id && (batch_values === null || other.previous.size === 0 || batch.folded.includes(other));
}
function abort_all(reaction) {
  if ((reaction.f & 134217728) === 0) return !1;
  reaction.f ^= 134217728, abort_signal(reaction.ac), reaction.ac = null;
  for (var b = first_batch; b !== null; b = b.next) {
    var record = b.records?.get(reaction);
    record?.ac && (abort_signal(record.ac), record.ac = null);
  }
  return !0;
}
function is_recorded_effect(flags) {
  return (flags & 4194304) === 0 && (flags & 32784) !== 32784 ? !1 : view_batch !== null && view_batch === current_batch;
}
function update_reaction(reaction) {
  let batch = current_batch;
  var previous_deps = new_deps, previous_skipped_deps = skipped_deps, previous_untracked_writes = untracked_writes, previous_reaction = active_reaction, previous_sources = current_sources, previous_component_context = component_context, previous_untracking = untracking, previous_update_version = update_version, flags = reaction.f;
  new_deps = /** @type {null | Value[]} */
  null, skipped_deps = 0, untracked_writes = null, active_reaction = (flags & 96) === 0 ? reaction : null, current_sources = null, set_component_context(reaction.ctx), untracking = !1, update_version = ++read_version;
  var previous_abort_controller = abort_controller;
  abort_controller = null, (flags & 134217728) !== 0 && abort_replaced(reaction, flags);
  try {
    reaction.f |= 2097152;
    var fn = (
      /** @type {Function} */
      reaction.fn
    ), result = fn();
    if (reaction.f |= 32768, update_dependencies(reaction, flags, result, !1), is_runes() && untracked_writes !== null && !untracking && // bail out if the effect didn't read anything
    (new_deps !== null || skipped_deps > 0) && (reaction.f & 6146) === 0)
      for (var i = 0; i < /** @type {Source[]} */
      untracked_writes.length; i++)
        schedule_possible_effect_self_invalidation(
          untracked_writes[i],
          /** @type {Effect} */
          reaction
        );
    if (previous_reaction !== null && previous_reaction !== reaction) {
      if (read_version++, previous_reaction.deps !== null)
        for (let i2 = 0; i2 < previous_skipped_deps; i2 += 1)
          previous_reaction.deps[i2].rv = read_version;
      if (previous_deps !== null)
        for (let dep of previous_deps)
          dep.rv = read_version;
      untracked_writes !== null && (previous_untracked_writes === null ? previous_untracked_writes = untracked_writes : previous_untracked_writes.push(.../** @type {Source[]} */
      untracked_writes));
    }
    return result;
  } catch (error) {
    var value = error;
    try {
      value = handle_error(error);
    } finally {
      update_dependencies(reaction, flags, value, value === error);
    }
    return value;
  } finally {
    var signal = (
      /** @type {AbortController | null} */
      abort_controller?.signal
    );
    batch !== null && signal !== void 0 && signal.addEventListener("abort", () => {
      if (current_batch !== null) {
        let target = merge_target(batch);
        target !== current_batch && target.linked && !target.is_fork && current_batch.merge_requests.add(target);
      }
    }), reaction.f ^= 2097152, new_deps = previous_deps, skipped_deps = previous_skipped_deps, untracked_writes = previous_untracked_writes, active_reaction = previous_reaction, current_sources = previous_sources, set_component_context(previous_component_context), untracking = previous_untracking, update_version = previous_update_version, abort_controller = previous_abort_controller;
  }
}
function update_dependencies(reaction, flags, value, error) {
  var deps = reaction.deps, length = skipped_deps + (new_deps === null ? 0 : new_deps.length), dv = reaction.dv;
  if ((reaction.f & 16384) !== 0) {
    abort_signal(abort_controller);
    return;
  }
  if (abort_controller !== null && (reaction.f & 134217728) === 0 && (abort_signal(abort_controller), abort_controller = null), (flags & 2) !== 0 && is_destroying_effect) {
    for (var wv = 0, j = 0; j < length; j++) {
      var dep = j < skipped_deps ? (
        /** @type {Value[]} */
        deps[j]
      ) : (
        /** @type {Value[]} */
        new_deps[j - skipped_deps]
      );
      wv = Math.max(wv, teardown_version_of(dep));
    }
    teardown_run = { v: value, wv, e: error, assigned: (flags & 67108864) !== 0 };
    return;
  }
  if ((flags & 2) !== 0 && batch_values === null && first_batch !== null && first_batch.next !== null && keep_global_run(
    /** @type {Derived} */
    reaction,
    flags
  ), new_deps === null && batch_values === null && dv !== null && length === deps?.length && reaction.edges === deps && (flags & 4194320) === 0) {
    for (var i = 0; i < length; i++) dv[i] = deps[i].wv;
    dv.length !== length && (dv.length = length), (flags & 2) !== 0 && set_derived_value(
      /** @type {Derived} */
      reaction,
      value,
      error
    ), (reaction.ac || abort_controller) && set_global_signal(reaction);
    return;
  }
  update_record(reaction, flags, length, value, error);
}
function set_derived_value(derived2, value, error) {
  var flags = derived2.f & -67108865;
  (error || (flags & 8388608) !== 0 || !derived2.equals(value)) && (derived2.v = value, derived2.wv = increment_write_version(), flags = error ? flags | 8388608 : flags & -8388609), derived2.f = flags, update_derived_status(derived2);
}
function update_record(reaction, flags, length, value, error) {
  var should_record = is_recorded_effect(flags);
  if (batch_values !== null && (flags & 2) !== 0) {
    var written = batch_values.get(
      /** @type {Derived} */
      reaction
    );
    written !== void 0 && written.wv !== /** @type {Derived} */
    reaction.wv && (should_record = !0);
  }
  var deps = reaction.deps, read = deps;
  (new_deps !== null || skipped_deps !== deps?.length) && (read = skipped_deps === 0 ? new_deps : (
    /** @type {Value[]} */
    deps.slice(0, skipped_deps).concat(new_deps ?? [])
  ));
  for (var dv = !should_record && batch_values === null && reaction.dv !== null ? reaction.dv : [], i = 0; i < length; i++) {
    var dep = (
      /** @type {Value[]} */
      read[i]
    );
    dv[i] = seen_version(dep), batch_values !== null && (flags & 2) !== 0 && (dv[i] !== dep.wv || (dep.f & 1026) === 2) && (should_record = !0);
  }
  dv.length !== length && (dv.length = length), !should_record && batch_values !== null && (flags & 2) !== 0 && keep_global_run(
    /** @type {Derived} */
    reaction,
    flags
  );
  var keep = should_record || has_batch_record(reaction), tracking = effect_tracking() && (reaction.f & 512) !== 0, edges = reaction.edges;
  if (keep && edges !== null) {
    if (new_deps !== null) {
      reaction.edges = edges = edges.slice();
      for (dep of new_deps)
        includes.call(edges, dep) || (edges.push(dep), tracking && (dep.reactions ??= []).push(reaction));
    }
  } else {
    if (edges === deps) {
      if (edges !== null)
        for (i = skipped_deps; i < edges.length; i++) remove_reaction(reaction, edges[i]);
      if (tracking && new_deps !== null)
        for (dep of new_deps) (dep.reactions ??= []).push(reaction);
    } else {
      var added = new Set(read);
      for (
        dep of
        /** @type {Value[]} */
        edges
      )
        added.delete(dep) || remove_reaction(reaction, dep);
      if (tracking)
        for (dep of added) (dep.reactions ??= []).push(reaction);
    }
    reaction.edges = read;
  }
  if (should_record) {
    var batch = (
      /** @type {Batch} */
      view_batch
    );
    (batch.records ??= /* @__PURE__ */ new Map()).set(
      reaction,
      (flags & 2) !== 0 ? {
        deps: read,
        dv,
        v: value,
        e: error,
        wv: derived_version(
          /** @type {Derived} */
          reaction,
          batch,
          value,
          error
        ),
        ac: abort_controller
      } : { deps: read, dv, ac: abort_controller }
    );
  } else
    reaction.deps = read, reaction.dv = dv, (reaction.ac || abort_controller) && set_global_signal(reaction), (flags & 2) !== 0 && (set_derived_value(
      /** @type {Derived} */
      reaction,
      value,
      error
    ), view_batch?.records?.delete(reaction));
}
function keep_global_run(derived2, flags) {
  var dv = derived2.dv;
  if (
    // no valid run, or one that doesn't depend on anything — which is only run again if it was invalidated
    !(dv === null || dv.length === 0 || // an assigned value isn't a run, see `get_view`
    (flags & 67108864) !== 0)
  ) {
    for (var record, ac = derived2.ac, live = ac !== null && current_batch !== null && current_batch.is_fork && !ac.signal.aborted, batch = first_batch; batch !== null; batch = batch.next)
      batch !== view_batch && !batch.records?.has(derived2) && // otherwise, the run's signal was aborted when the derived ran again, or is about to be (see `set_global_signal`)
      (ac === null || live || supersedes(current_batch, batch)) && (record ??= {
        deps: derived2.deps,
        dv: dv.slice(),
        v: derived2.v,
        wv: derived2.wv,
        e: (flags & 8388608) !== 0
      }, (batch.records ??= /* @__PURE__ */ new Map()).set(derived2, record));
    live && record !== void 0 && (record.ac = ac, derived2.ac = null);
  }
}
function seen_version(signal) {
  if (batch_values === null) return signal.wv;
  if ((signal.f & 2) !== 0) {
    var cache = (
      /** @type {Map<Derived, Derived | DerivedRecord | BatchValue>} */
      view_cache
    );
    return cache.get(
      /** @type {Derived} */
      signal
    )?.wv ?? -1;
  }
  return (batch_values.get(signal) ?? signal).wv;
}
function derived_version(derived2, batch, value, error) {
  if (!error) {
    var previous = (
      /** @type {DerivedRecord | undefined} */
      batch.get_record(derived2)
    );
    if (previous !== void 0 && !previous.e && derived2.equals.call(previous, value))
      return previous.wv;
    if ((derived2.f & 8388608) === 0 && derived2.equals(value)) return derived2.wv;
  }
  return increment_write_version();
}
function promote_record(reaction, record) {
  if (reaction.deps = record.deps, reaction.dv = (reaction.f & 2) !== 0 ? record.dv.slice() : record.dv, (reaction.f & 2) !== 0) {
    var derived2 = (
      /** @type {Derived} */
      reaction
    );
    derived2.v = record.v, derived2.wv = /** @type {number} */
    record.wv, derived2.f = (record.e ? derived2.f | 8388608 : derived2.f & -8388609) & -67108865, update_derived_status(derived2);
  }
  abort_signal(reaction.ac), reaction.ac = record.ac ?? null, record.ac = null;
}
function has_batch_record(reaction) {
  for (var batch = first_batch; batch !== null; batch = batch.next)
    if (batch.records?.has(reaction)) return !0;
  return !1;
}
function prune_edges(effect2) {
  var edges = effect2.edges, deps = effect2.deps;
  if (!(edges === deps || has_batch_record(effect2))) {
    for (
      var dep of
      /** @type {Value[]} */
      edges
    )
      (deps === null || !includes.call(deps, dep)) && remove_reaction(effect2, dep);
    effect2.edges = deps;
  }
}
function add_reaction(signal, reaction) {
  var reactions = signal.reactions ??= [];
  includes.call(reactions, reaction) || reactions.push(reaction);
}
function remove_reaction(signal, dependency) {
  let reactions = dependency.reactions;
  if (reactions !== null) {
    var index = index_of.call(reactions, signal);
    if (index !== -1) {
      var new_length = reactions.length - 1;
      new_length === 0 ? reactions = dependency.reactions = null : (reactions[index] = reactions[new_length], reactions.pop());
    }
  }
  if (reactions === null && (dependency.f & 2) !== 0 && // Destroying a child effect while updating a parent effect can cause a dependency to appear
  // to be unused, when in fact it is used by the currently-updating parent. Checking `new_deps`
  // allows us to skip the expensive work of disconnecting and immediately reconnecting it
  (new_deps === null || !includes.call(new_deps, dependency))) {
    var derived2 = (
      /** @type {Derived} */
      dependency
    );
    if ((derived2.f & 512) !== 0 && (derived2.f ^= 512), derived2.v !== UNINITIALIZED && update_derived_status(derived2), abort_all(derived2)) {
      set_signal_status(derived2, 2048), derived2.dv = null;
      for (var b = first_batch; b !== null; b = b.next) b.records?.delete(derived2);
      view_cache?.delete(derived2);
    }
    freeze_derived_effects(derived2), remove_reactions(derived2);
  }
}
function remove_reactions(signal) {
  var dependencies = signal.edges;
  if (dependencies !== null)
    for (var i = 0; i < dependencies.length; i++)
      remove_reaction(signal, dependencies[i]);
}
function update_effect(effect2) {
  var flags = effect2.f;
  if ((flags & 16384) === 0) {
    set_signal_status(effect2, 1024);
    var previous_effect = active_effect;
    active_effect = effect2;
    var previous_renderer = custom_renderers_flag ? push_renderer(effect2.r) : null;
    if (false_default) {
      var previous_component_fn = dev_current_component_function;
      set_dev_current_component_function(effect2.component_function);
      var previous_stack = (
        /** @type {any} */
        dev_stack
      );
      set_dev_stack(effect2.dev_stack ?? dev_stack);
    }
    try {
      (flags & 16777232) !== 0 ? destroy_block_effect_children(effect2) : destroy_effect_children(effect2), execute_effect_teardown(effect2);
      var teardown2 = update_reaction(effect2);
      if (effect2.teardown = typeof teardown2 == "function" ? teardown2 : null, effect2.wv = write_version, false_default && tracing_mode_flag && (effect2.f & 2048) !== 0 && effect2.edges !== null)
        for (var dep of effect2.edges)
          dep.set_during_effect && (dep.wv = increment_write_version(), dep.set_during_effect = !1);
    } finally {
      active_effect = previous_effect, custom_renderers_flag && pop_renderer(previous_renderer), false_default && (set_dev_current_component_function(previous_component_fn), set_dev_stack(previous_stack));
    }
  }
}
async function tick() {
  if (async_mode_flag)
    return new Promise((f) => {
      requestAnimationFrame(() => f()), setTimeout(() => f());
    });
  await Promise.resolve(), flushSync();
}
function settled() {
  return Batch.ensure().settled();
}
function get(signal) {
  var flags = signal.f, is_derived = (flags & 2) !== 0;
  captured_signals?.add(signal);
  var late_reader = null;
  if (active_reaction !== null && !untracking) {
    var destroyed = active_effect !== null && (active_effect.f & 16384) !== 0;
    if (!destroyed && (current_sources === null || !current_sources.has(signal))) {
      var deps = active_reaction.deps;
      if ((active_reaction.f & 2097152) !== 0)
        signal.rv < read_version && (signal.rv = read_version, new_deps === null && deps !== null && deps[skipped_deps] === signal ? skipped_deps++ : new_deps === null ? new_deps = [signal] : new_deps.push(signal));
      else {
        late_reader = active_reaction;
        var edges = active_reaction.edges;
        (edges === null || !includes.call(edges, signal)) && (active_reaction.edges = edges === null ? [signal] : [...edges, signal]), add_reaction(signal, active_reaction);
      }
    }
  }
  if (false_default) {
    if (!untracking && // By checking that no batch is active we filter out false positives.
    // reactivity_loss_tracker is only reset after a microtask, so if a flush happens
    // before that, we get warnings for things we shouldn't warn on.
    view_batch === null && reactivity_loss_tracker && !reactivity_loss_tracker.warned && (reactivity_loss_tracker.effect.f & 2097152) === 0 && !reactivity_loss_tracker.effect_deps.has(signal)) {
      reactivity_loss_tracker.warned = !0, await_reactivity_loss(
        /** @type {string} */
        signal.label
      );
      var trace2 = get_error("traced at");
      trace2 && console.warn(trace2);
    }
    if (recent_async_deriveds.delete(signal), tracing_mode_flag && !untracking && tracing_expressions !== null && active_reaction !== null && tracing_expressions.reaction === active_reaction) {
      if (signal.trace)
        signal.trace();
      else if (trace2 = get_error("traced at"), trace2) {
        var entry = tracing_expressions.entries.get(signal);
        entry === void 0 && (entry = { traces: [] }, tracing_expressions.entries.set(signal, entry));
        var last = entry.traces[entry.traces.length - 1];
        trace2.stack !== last?.stack && entry.traces.push(trace2);
      }
    }
  }
  if (is_destroying_effect) {
    var held = teardown_view(signal);
    if (held !== void 0) {
      if (held.e) throw held.v;
      return held.v;
    }
  }
  var view;
  if (is_derived) {
    var derived2 = (
      /** @type {Derived} */
      signal
    ), should_connect = (derived2.f & 512) === 0 && !untracking && active_reaction !== null && (active_reaction.f & 512) !== 0, is_new = (derived2.f & 32768) === 0;
    should_connect && (derived2.f |= 512), batch_values !== null ? view = get_view(derived2) : update_derived(derived2), should_connect && !is_new && (unfreeze_derived_effects(derived2), reconnect(derived2));
  } else batch_values !== null && (view = batch_values.get(signal));
  if (late_reader !== null && record_late_read(late_reader, signal, view === void 0 ? signal.wv : view.wv), view !== void 0 && view !== signal) {
    if (view.e) throw view.v;
    return view.v;
  }
  if ((signal.f & 8388608) !== 0)
    throw signal.v;
  return signal.v;
}
function record_late_read(reaction, signal, version) {
  var run3 = run_of(reaction), deps = run3.deps;
  run3.dv !== null && (deps === null || !includes.call(deps, signal)) && (run3.deps = deps === null ? [signal] : [...deps, signal], run3.dv.push(version));
}
function reconnect(derived2) {
  if (derived2.f |= 512, derived2.edges !== null)
    for (let dep of derived2.edges)
      add_reaction(dep, derived2), (dep.f & 2) !== 0 && (dep.f & 512) === 0 && (unfreeze_derived_effects(
        /** @type {Derived} */
        dep
      ), reconnect(
        /** @type {Derived} */
        dep
      ));
}
function teardown_view(signal) {
  var written = signal.wv > teardown_version ? void 0 : teardown_context?.reverted_values.get(signal);
  if ((signal.f & 2) === 0) return written;
  var derived2 = (
    /** @type {Derived} */
    signal
  ), cache = (teardown_context ?? push_teardown_context()).derived_cache, record = cache.get(derived2);
  if (record !== void 0) return record;
  var held = written ?? batch_values?.get(derived2), assigned = held === void 0 ? (derived2.f & 67108864) !== 0 : held.assigned, dv = held === void 0 && !assigned ? derived2.dv : null, deps = derived2.deps ?? [], holds = assigned || dv !== null, length = dv?.length ?? deps.length;
  held ??= {
    v: derived2.v,
    wv: derived2.wv,
    e: (derived2.f & 8388608) !== 0,
    assigned: (derived2.f & 67108864) !== 0
  };
  for (var i = 0; holds && i < length; i++) {
    var version = teardown_version_of(deps[i]);
    holds = dv === null ? version <= held.wv : version === dv[i];
  }
  return holds ? record = held : (execute_derived(derived2), record = teardown_run, held.wv < record.wv && !held.e && !record.e && derived2.equals.call(held, record.v) && (record.wv = held.wv)), cache.set(derived2, record), record;
}
function teardown_version_of(signal) {
  return (teardown_view(signal) ?? batch_values?.get(signal) ?? signal).wv;
}
function safe_get(signal) {
  return signal && get(signal);
}
function untrack(fn) {
  var previous_untracking = untracking;
  try {
    return untracking = !0, fn();
  } finally {
    untracking = previous_untracking;
  }
}
function deep_read_state(value) {
  if (!(typeof value != "object" || !value || value instanceof EventTarget)) {
    if (STATE_SYMBOL in value)
      deep_read(value);
    else if (!Array.isArray(value))
      for (let key in value) {
        let prop = value[key];
        typeof prop == "object" && prop && STATE_SYMBOL in prop && deep_read(prop);
      }
  }
}
function deep_read(value, visited = /* @__PURE__ */ new Set()) {
  if (typeof value == "object" && value !== null && // We don't want to traverse DOM elements
  !(value instanceof EventTarget) && !visited.has(value)) {
    visited.add(value), value instanceof Date && value.getTime();
    for (let key in value)
      try {
        deep_read(value[key], visited);
      } catch {
      }
    let proto = get_prototype_of(value);
    if (proto !== Object.prototype && proto !== Array.prototype && proto !== Map.prototype && proto !== Set.prototype && proto !== Date.prototype) {
      let descriptors = get_descriptors(proto);
      for (let key in descriptors) {
        let get2 = descriptors[key].get;
        if (get2)
          try {
            get2.call(value);
          } catch {
          }
      }
    }
  }
}

// node_modules/svelte/src/internal/client/proxy.js
var regex_is_valid_identifier = /^[a-zA-Z_$][a-zA-Z_$0-9]*$/;
function proxy(value) {
  if (typeof value != "object" || value === null || STATE_SYMBOL in value || COMPONENT_SYMBOL in value)
    return value;
  let prototype = get_prototype_of(value);
  if (prototype !== object_prototype && prototype !== array_prototype)
    return value;
  var sources = /* @__PURE__ */ new Map(), is_proxied_array = is_array(value), version = state(0), stack2 = false_default && tracing_mode_flag ? get_error("created at") : null, parent_version = update_version, with_parent = (fn) => {
    if (update_version === parent_version)
      return fn();
    var reaction = active_reaction, version2 = update_version;
    set_active_reaction(null), set_update_version(parent_version);
    var result = fn();
    return set_active_reaction(reaction), set_update_version(version2), result;
  };
  is_proxied_array && (sources.set("length", state(
    /** @type {any[]} */
    value.length,
    stack2
  )), false_default && (value = /** @type {any} */
  inspectable_array(
    /** @type {any[]} */
    value
  )));
  var path = "";
  let updating = !1;
  function update_path(new_path) {
    if (!updating) {
      updating = !0, path = new_path, tag(version, `${path} version`);
      for (let [prop, source2] of sources)
        tag(source2, get_label(path, prop));
      updating = !1;
    }
  }
  return new Proxy(
    /** @type {any} */
    value,
    {
      defineProperty(_, prop, descriptor) {
        (!("value" in descriptor) || descriptor.configurable === !1 || descriptor.enumerable === !1 || descriptor.writable === !1) && state_descriptors_fixed();
        var s = sources.get(prop);
        return s === void 0 ? with_parent(() => {
          var s2 = state(descriptor.value, stack2);
          return sources.set(prop, s2), false_default && typeof prop == "string" && tag(s2, get_label(path, prop)), s2;
        }) : set(s, descriptor.value, !0), !0;
      },
      deleteProperty(target, prop) {
        var s = sources.get(prop);
        if (s === void 0) {
          if (prop in target) {
            let s2 = with_parent(() => state(UNINITIALIZED, stack2));
            sources.set(prop, s2), increment(version), false_default && tag(s2, get_label(path, prop));
          }
        } else
          set(s, UNINITIALIZED), increment(version);
        return !0;
      },
      get(target, prop, receiver) {
        if (prop === STATE_SYMBOL)
          return value;
        if (false_default && prop === PROXY_PATH_SYMBOL)
          return update_path;
        var s = sources.get(prop), exists = prop in target;
        if (s === void 0 && (!exists || get_descriptor(target, prop)?.writable) && (s = with_parent(() => {
          var p = proxy(exists ? target[prop] : UNINITIALIZED), s2 = state(p, stack2);
          return false_default && tag(s2, get_label(path, prop)), s2;
        }), sources.set(prop, s)), s !== void 0) {
          var v = get(s);
          return v === UNINITIALIZED ? void 0 : v;
        }
        return Reflect.get(target, prop, receiver);
      },
      getOwnPropertyDescriptor(target, prop) {
        this.has?.(target, prop);
        var descriptor = Reflect.getOwnPropertyDescriptor(target, prop), s = sources.get(prop);
        if (s !== void 0) {
          var value2 = get(s);
          if (value2 === UNINITIALIZED)
            return;
          if (descriptor && "value" in descriptor)
            descriptor.value = value2;
          else
            return {
              enumerable: !0,
              configurable: !0,
              value: value2,
              writable: !0
            };
        }
        return descriptor;
      },
      has(target, prop) {
        if (prop === STATE_SYMBOL)
          return !0;
        var s = sources.get(prop), has = s !== void 0 && s.v !== UNINITIALIZED || Reflect.has(target, prop);
        if (s !== void 0 || (active_effect !== null || active_reaction !== null) && (!has || get_descriptor(target, prop)?.writable)) {
          s === void 0 && (s = with_parent(() => {
            var p = has ? proxy(target[prop]) : UNINITIALIZED, s2 = state(p, stack2);
            return false_default && tag(s2, get_label(path, prop)), s2;
          }), sources.set(prop, s));
          var value2 = get(s);
          if (value2 === UNINITIALIZED)
            return !1;
        }
        return has;
      },
      set(target, prop, value2, receiver) {
        var s = sources.get(prop), has = prop in target;
        if (is_proxied_array && prop === "length")
          for (var i = value2; i < /** @type {Source<number>} */
          s.v; i += 1) {
            var other_s = sources.get(i + "");
            other_s !== void 0 ? set(other_s, UNINITIALIZED) : i in target && (other_s = with_parent(() => state(UNINITIALIZED, stack2)), sources.set(i + "", other_s), false_default && tag(other_s, get_label(path, i)));
          }
        if (s === void 0)
          (!has || get_descriptor(target, prop)?.writable) && (s = with_parent(() => state(void 0, stack2)), false_default && tag(s, get_label(path, prop)), set(s, proxy(value2)), sources.set(prop, s));
        else {
          has = s.v !== UNINITIALIZED;
          var p = with_parent(() => proxy(value2));
          set(s, p);
        }
        var descriptor = Reflect.getOwnPropertyDescriptor(target, prop);
        if (descriptor?.set && descriptor.set.call(receiver, value2), !has) {
          if (is_proxied_array && typeof prop == "string") {
            var ls = (
              /** @type {Source<number>} */
              sources.get("length")
            ), n = Number(prop);
            Number.isInteger(n) && n >= ls.v && set(ls, n + 1);
          }
          increment(version);
        }
        return !0;
      },
      ownKeys(target) {
        get(version);
        var own_keys = Reflect.ownKeys(target).filter((key2) => {
          var source3 = sources.get(key2);
          return source3 === void 0 || source3.v !== UNINITIALIZED;
        });
        for (var [key, source2] of sources)
          source2.v !== UNINITIALIZED && !(key in target) && own_keys.push(key);
        return own_keys;
      },
      setPrototypeOf() {
        state_prototype_fixed();
      }
    }
  );
}
function get_label(path, prop) {
  return typeof prop == "symbol" ? `${path}[Symbol(${prop.description ?? ""})]` : regex_is_valid_identifier.test(prop) ? `${path}.${prop}` : /^\d+$/.test(prop) ? `${path}[${prop}]` : `${path}['${prop}']`;
}
function get_proxied_value(value) {
  try {
    if (value !== null && typeof value == "object" && STATE_SYMBOL in value)
      return value[STATE_SYMBOL];
  } catch {
  }
  return value;
}
function is(a, b) {
  return Object.is(get_proxied_value(a), get_proxied_value(b));
}
var ARRAY_MUTATING_METHODS = /* @__PURE__ */ new Set([
  "copyWithin",
  "fill",
  "pop",
  "push",
  "reverse",
  "shift",
  "sort",
  "splice",
  "unshift"
]);
function inspectable_array(array) {
  return new Proxy(array, {
    get(target, prop, receiver) {
      var value = Reflect.get(target, prop, receiver);
      return ARRAY_MUTATING_METHODS.has(
        /** @type {string} */
        prop
      ) ? function(...args) {
        set_eager_effects_deferred();
        var result = value.apply(this, args);
        return flush_eager_effects(), result;
      } : value;
    }
  });
}

// node_modules/svelte/src/internal/client/dev/equality.js
function init_array_prototype_warnings() {
  let array_prototype2 = Array.prototype, cleanup = Array.__svelte_cleanup;
  cleanup && cleanup();
  let { indexOf, lastIndexOf, includes: includes2 } = array_prototype2;
  array_prototype2.indexOf = function(item, from_index) {
    let index = indexOf.call(this, item, from_index);
    if (index === -1) {
      for (let i = from_index ?? 0; i < this.length; i += 1)
        if (get_proxied_value(this[i]) === item) {
          state_proxy_equality_mismatch("array.indexOf(...)");
          break;
        }
    }
    return index;
  }, array_prototype2.lastIndexOf = function(item, from_index) {
    let index = lastIndexOf.call(this, item, from_index ?? this.length - 1);
    if (index === -1) {
      for (let i = 0; i <= (from_index ?? this.length - 1); i += 1)
        if (get_proxied_value(this[i]) === item) {
          state_proxy_equality_mismatch("array.lastIndexOf(...)");
          break;
        }
    }
    return index;
  }, array_prototype2.includes = function(item, from_index) {
    let has = includes2.call(this, item, from_index);
    if (!has) {
      for (let i = 0; i < this.length; i += 1)
        if (get_proxied_value(this[i]) === item) {
          state_proxy_equality_mismatch("array.includes(...)");
          break;
        }
    }
    return has;
  }, Array.__svelte_cleanup = () => {
    array_prototype2.indexOf = indexOf, array_prototype2.lastIndexOf = lastIndexOf, array_prototype2.includes = includes2;
  };
}
function strict_equals(a, b, equal = !0) {
  try {
    a === b != (get_proxied_value(a) === get_proxied_value(b)) && state_proxy_equality_mismatch(equal ? "===" : "!==");
  } catch {
  }
  return a === b === equal;
}
function equals2(a, b, equal = !0) {
  return a == b != (get_proxied_value(a) == get_proxied_value(b)) && state_proxy_equality_mismatch(equal ? "==" : "!="), a == b === equal;
}

// node_modules/svelte/src/internal/client/dom/operations.js
var $window, $document, is_firefox, first_child_getter, next_sibling_getter;
function init_operations() {
  if ($window === void 0) {
    $window = window, $document = document, is_firefox = /Firefox/.test(navigator.userAgent);
    var element_prototype = Element.prototype, node_prototype = Node.prototype, text_prototype = Text.prototype;
    first_child_getter = get_descriptor(node_prototype, "firstChild").get, next_sibling_getter = get_descriptor(node_prototype, "nextSibling").get, is_extensible(element_prototype) && (element_prototype[CLASS_CACHE] = void 0, element_prototype[ATTRIBUTES_CACHE] = null, element_prototype[STYLE_CACHE] = void 0, element_prototype.__e = void 0), is_extensible(text_prototype) && (text_prototype[TEXT_CACHE] = void 0), false_default && (element_prototype.__svelte_meta = null, init_array_prototype_warnings());
  }
}
function create_text(value = "") {
  return current_renderer ? (
    /** @type {Text} */
    current_renderer.createText(value)
  ) : document.createTextNode(value);
}
// @__NO_SIDE_EFFECTS__
function get_first_child(node) {
  return current_renderer ? (
    /** @type {TemplateNode | null} */
    current_renderer.getFirstChild(node)
  ) : (
    /** @type {TemplateNode | null} */
    first_child_getter.call(node)
  );
}
// @__NO_SIDE_EFFECTS__
function get_next_sibling(node) {
  return current_renderer ? (
    /** @type {TemplateNode | null} */
    current_renderer.getNextSibling(node)
  ) : (
    /** @type {TemplateNode | null} */
    next_sibling_getter.call(node)
  );
}
function child(node, is_text) {
  if (!hydrating)
    return /* @__PURE__ */ get_first_child(node);
  var child2 = /* @__PURE__ */ get_first_child(hydrate_node);
  if (child2 === null)
    child2 = /** @type {TemplateNode} */
    append_child(hydrate_node, create_text());
  else if (is_text && node_type(child2) !== TEXT_NODE) {
    var text = create_text();
    return insert_before(child2, text), set_hydrate_node(text), text;
  }
  return is_text && merge_text_nodes(
    /** @type {Text} */
    child2
  ), set_hydrate_node(child2), child2;
}
function first_child(node, is_text = !1) {
  if (!hydrating) {
    var first = /* @__PURE__ */ get_first_child(node);
    return is_comment(first) && get_node_value(first) === "" ? /* @__PURE__ */ get_next_sibling(first) : first;
  }
  if (is_text) {
    if (node_type(hydrate_node) !== TEXT_NODE) {
      var text = create_text();
      return hydrate_node && insert_before(hydrate_node, text), set_hydrate_node(text), text;
    }
    merge_text_nodes(
      /** @type {Text} */
      hydrate_node
    );
  }
  return hydrate_node;
}
function only_child(node, is_text = !1) {
  if (!hydrating)
    return /* @__PURE__ */ get_first_child(node);
  var first = child(node, is_text);
  return reset(node), first;
}
function sibling(node, count = 1, is_text = !1) {
  let next_sibling = hydrating ? hydrate_node : node;
  for (var last_sibling; count--; )
    last_sibling = next_sibling, next_sibling = /** @type {TemplateNode} */
    /* @__PURE__ */ get_next_sibling(next_sibling);
  if (!hydrating)
    return next_sibling;
  if (is_text) {
    if (node_type(next_sibling) !== TEXT_NODE) {
      var text = create_text();
      return next_sibling === null ? last_sibling && insert_after(last_sibling, text) : insert_before(next_sibling, text), set_hydrate_node(text), text;
    }
    merge_text_nodes(
      /** @type {Text} */
      next_sibling
    );
  }
  return set_hydrate_node(next_sibling), next_sibling;
}
function clear_text_content(node) {
  if (current_renderer) {
    for (var child2 = current_renderer.getFirstChild(node); child2 !== null; ) {
      var next2 = current_renderer.getNextSibling(child2);
      current_renderer.remove(child2), child2 = next2;
    }
    return;
  }
  node.textContent = "";
}
function should_defer_append() {
  if (!async_mode_flag && eager_block_writes === null) return !1;
  var flags = (
    /** @type {Effect} */
    active_effect.f
  );
  return (flags & 32768) !== 0;
}
function create_element(tag2, namespace, is2) {
  return current_renderer ? current_renderer.createElement(tag2) : namespace == null || namespace === NAMESPACE_HTML ? (
    /** @type {T extends keyof HTMLElementTagNameMap ? HTMLElementTagNameMap[T] : Element} */
    is2 ? document.createElement(tag2, { is: is2 }) : document.createElement(tag2)
  ) : (
    /** @type {T extends keyof HTMLElementTagNameMap ? HTMLElementTagNameMap[T] : Element} */
    is2 ? document.createElementNS(namespace, tag2, { is: is2 }) : document.createElementNS(namespace, tag2)
  );
}
function create_fragment() {
  return current_renderer ? (
    /** @type {DocumentFragment} */
    current_renderer.createFragment()
  ) : document.createDocumentFragment();
}
function create_comment(data = "") {
  return current_renderer ? (
    /** @type {Comment} */
    current_renderer.createComment(data)
  ) : document.createComment(data);
}
function set_attribute(element, key, value = "") {
  if (current_renderer) {
    current_renderer.setAttribute(element, key, value);
    return;
  }
  if (key.startsWith("xlink:")) {
    element.setAttributeNS("http://www.w3.org/1999/xlink", key, value);
    return;
  }
  return element.setAttribute(key, value);
}
function merge_text_nodes(text) {
  if (current_renderer || /** @type {string} */
  text.nodeValue.length < 65536)
    return;
  let next2 = text.nextSibling;
  for (; next2 !== null && node_type(next2) === TEXT_NODE; )
    next2.remove(), text.nodeValue += /** @type {string} */
    next2.nodeValue, next2 = text.nextSibling;
}
function is_comment(node) {
  return current_renderer ? !!node && node_type(node) === COMMENT_NODE : node instanceof Comment;
}
function node_type(node) {
  if (node != null) {
    if (current_renderer) {
      let type = current_renderer.getNodeType(node);
      return CUSTOM_RENDERER_NODE_TYPE_MAP[type];
    }
    return node.nodeType;
  }
}
function node_name(node) {
  return current_renderer ? "" : node.nodeName;
}
function get_last_child(node) {
  return current_renderer ? (
    /** @type {TemplateNode | null} */
    current_renderer.getLastChild(node)
  ) : (
    /** @type {TemplateNode | null} */
    node.lastChild
  );
}
function get_parent_node(node) {
  return current_renderer ? (
    /** @type {TemplateNode | null} */
    current_renderer.getParent(node)
  ) : (
    /** @type {TemplateNode | null} */
    node.parentNode
  );
}
function append_child(parent, child2) {
  return current_renderer ? (current_renderer.insert(parent, child2, null), child2) : parent.appendChild(child2);
}
function insert_before(ref_node, new_node) {
  if (current_renderer) {
    var parent = current_renderer.getParent(ref_node);
    current_renderer.insert(parent, new_node, ref_node);
    return;
  }
  ref_node.before(new_node);
}
function insert_after(ref_node, new_node) {
  if (current_renderer) {
    var parent = current_renderer.getParent(ref_node), next2 = current_renderer.getNextSibling(ref_node);
    current_renderer.insert(parent, new_node, next2);
    return;
  }
  ref_node.after(new_node);
}
function remove_node(node) {
  if (current_renderer) {
    current_renderer.remove(node);
    return;
  }
  node.remove();
}
function remove_child(parent, child2) {
  return current_renderer ? (current_renderer.remove(child2), child2) : (
    /** @type {ChildNode} */
    parent.removeChild(child2)
  );
}
function replace_with(old_node, new_node) {
  if (current_renderer) {
    var parent = current_renderer.getParent(old_node);
    current_renderer.insert(parent, new_node, old_node), current_renderer.remove(old_node);
    return;
  }
  old_node.replaceWith(new_node);
}
function set_text_content(node, value) {
  current_renderer && custom_renderer_unsupported_operation("textContent"), node.textContent = value;
}
function set_node_value(node, value) {
  if (current_renderer) {
    current_renderer.setText(node, value);
    return;
  }
  node.nodeValue = value;
}
function get_node_value(node) {
  return current_renderer ? current_renderer.getText(node) : node.nodeValue;
}
function set_element_value(element, value) {
  if (current_renderer) {
    current_renderer.setAttribute(element, "value", value ?? "");
    return;
  }
  element.value = value ?? "";
}
function set_element_checked(element, checked) {
  if (current_renderer) {
    checked ? current_renderer.setAttribute(element, "checked", "") : current_renderer.removeAttribute(element, "checked");
    return;
  }
  element.checked = checked;
}
function set_element_default_value(element, value) {
  if (current_renderer) {
    current_renderer.setAttribute(element, "defaultValue", value);
    return;
  }
  let existing_value = element.value;
  element.defaultValue = value, element.value = existing_value;
}
function set_element_default_checked(element, checked) {
  if (current_renderer) {
    checked ? current_renderer.setAttribute(element, "defaultChecked", "") : current_renderer.removeAttribute(element, "defaultChecked");
    return;
  }
  let existing_value = element.checked;
  element.defaultChecked = checked, element.checked = existing_value;
}
function get_attribute(element, name) {
  return current_renderer ? current_renderer.getAttribute(element, name) : element.getAttribute(name);
}
function remove_attribute(element, name) {
  if (current_renderer) {
    current_renderer.removeAttribute(element, name);
    return;
  }
  element.removeAttribute(name);
}
function has_attribute(element, name) {
  return current_renderer ? current_renderer.hasAttribute(element, name) : element.hasAttribute(name);
}
function set_inner_html(element, value) {
  current_renderer && custom_renderer_unsupported_operation("innerHTML"), element.innerHTML = value;
}
function clone_node(node, deep) {
  return current_renderer && custom_renderer_unsupported_operation("cloneNode"), node.cloneNode(deep);
}
function import_node(node, deep) {
  return current_renderer && custom_renderer_unsupported_operation("importNode"), document.importNode(node, deep);
}
function add_event_listener(target, type, handler, options) {
  if (current_renderer) {
    current_renderer.addEventListener(target, type, handler, options);
    return;
  }
  target.addEventListener(type, handler, options);
}
function remove_event_listener(target, type, handler, options) {
  if (current_renderer) {
    current_renderer.removeEventListener(target, type, handler, options);
    return;
  }
  target.removeEventListener(type, handler, options);
}
function dispatch_event(target, event) {
  return current_renderer && custom_renderer_unsupported_operation("dispatchEvent"), target.dispatchEvent(event);
}
function style_set_property(element, property, value, priority) {
  element.style.setProperty(property, value, priority);
}
function style_remove_property(element, property) {
  element.style.removeProperty(property);
}
function set_css_text(element, value) {
  if (current_renderer) {
    current_renderer.setAttribute(element, "style", value);
    return;
  }
  element.style.cssText = value;
}
function class_list_toggle(element, name, force) {
  element.classList.toggle(name, force);
}

// node_modules/svelte/src/internal/client/dom/hydration.js
var hydrating = !1;
function set_hydrating(value) {
  hydrating = value;
}
var hydrate_node;
function set_hydrate_node(node) {
  if (node === null)
    throw hydration_mismatch(), HYDRATION_ERROR;
  return hydrate_node = node;
}
function hydrate_next() {
  return set_hydrate_node(get_next_sibling(hydrate_node));
}
function reset(node) {
  if (hydrating) {
    if (get_next_sibling(hydrate_node) !== null)
      throw hydration_mismatch(), HYDRATION_ERROR;
    hydrate_node = node;
  }
}
function hydrate_template(template) {
  hydrating && (hydrate_node = template.content);
}
function next(count = 1) {
  if (hydrating) {
    for (var i = count, node = hydrate_node; i--; )
      node = /** @type {TemplateNode} */
      get_next_sibling(node);
    hydrate_node = node;
  }
}
function skip_nodes(remove = !0) {
  for (var depth2 = 0, node = hydrate_node; ; ) {
    if (node_type(node) === COMMENT_NODE) {
      var data = get_node_value(node) ?? "";
      if (data === "]") {
        if (depth2 === 0) return node;
        depth2 -= 1;
      } else (data === "[" || data === "[!" || // "[?" followed by the serialized error, for failed boundaries
      data.startsWith("[?") || // "[1", "[2", etc. for if blocks
      data[0] === "[" && !isNaN(Number(data.slice(1)))) && (depth2 += 1);
    }
    var next2 = (
      /** @type {TemplateNode} */
      get_next_sibling(node)
    );
    remove && remove_node(node), node = next2;
  }
}
function read_hydration_instruction(node) {
  if (!node || node_type(node) !== COMMENT_NODE)
    throw hydration_mismatch(), HYDRATION_ERROR;
  return get_node_value(node) ?? "";
}

export {
  is_array,
  array_from,
  object_keys,
  define_property,
  get_descriptor,
  get_descriptors,
  get_prototype_of,
  has_own_property,
  is_function,
  noop,
  is_promise,
  run,
  run_all,
  deferred,
  fallback,
  to_array,
  exclude_from_object,
  STATE_SYMBOL,
  LEGACY_PROPS,
  LOADING_ATTR_SYMBOL,
  ATTRIBUTES_CACHE,
  CLASS_CACHE,
  STYLE_CACHE,
  TEXT_CACHE,
  FORM_RESET_HANDLER,
  HMR_ANCHOR,
  IS_XHTML,
  ELEMENT_NODE,
  TEXT_NODE,
  COMMENT_NODE,
  DOCUMENT_FRAGMENT_NODE,
  HYDRATION_ERROR,
  UNINITIALIZED,
  FILENAME,
  HMR,
  NAMESPACE_HTML,
  NAMESPACE_SVG,
  NAMESPACE_MATHML,
  ATTACHMENT_KEY,
  hydrating,
  set_hydrating,
  hydrate_node,
  set_hydrate_node,
  hydrate_next,
  reset,
  hydrate_template,
  next,
  skip_nodes,
  read_hydration_instruction,
  safe_not_equal,
  experimental_async_required,
  invalid_default_snippet,
  invalid_snippet_arguments,
  lifecycle_outside_component,
  snippet_without_render_tag,
  store_invalid_shape,
  svelte_element_invalid_this_value,
  bind_invalid_checkbox_value,
  component_api_changed,
  component_api_invalid_new,
  each_key_duplicate,
  each_key_volatile,
  effect_pending_outside_reaction,
  get_abort_signal_outside_reaction,
  hydratable_missing_but_required,
  hydration_failed,
  invalid_snippet,
  invalid_snippet_in_custom_renderer,
  lifecycle_legacy_only,
  props_invalid_value,
  props_rest_readonly,
  renderer_cannot_cross,
  rune_outside_svelte,
  snippet_renderer_mismatch,
  svelte_boundary_reset_onerror,
  async_mode_flag,
  legacy_mode_flag,
  custom_renderers_flag,
  dynamic_void_element_content,
  snapshot,
  queue_micro_task,
  set_signal_status,
  get_error,
  component_context,
  set_component_context,
  dev_stack,
  add_svelte_meta,
  dev_current_component_function,
  set_dev_current_component_function,
  createContext,
  getContext,
  setContext,
  hasContext,
  getAllContexts,
  push,
  pop,
  mark_as_component,
  is_runes,
  current_renderer,
  push_renderer,
  pop_renderer,
  flatten,
  run_after_blockers,
  capture,
  save,
  unsave,
  track_reactivity_loss,
  for_await_track_reactivity_loss,
  unset_context,
  run2,
  wait,
  merge_target,
  derived,
  async_derived,
  user_derived,
  derived_safe_equal,
  current_batch,
  view_batch,
  is_flushing_sync,
  Batch,
  set_before_effects,
  flushSync,
  eager,
  fork,
  trace,
  tag,
  tag_proxy,
  label,
  source,
  state,
  mutable_source,
  mutate,
  set,
  internal_set,
  update,
  update_pre,
  increment,
  proxy,
  is,
  strict_equals,
  equals2 as equals,
  $window,
  $document,
  is_firefox,
  init_operations,
  create_text,
  get_first_child,
  get_next_sibling,
  child,
  first_child,
  only_child,
  sibling,
  clear_text_content,
  should_defer_append,
  create_element,
  create_fragment,
  create_comment,
  set_attribute,
  merge_text_nodes,
  node_type,
  node_name,
  get_last_child,
  get_parent_node,
  append_child,
  insert_before,
  remove_node,
  remove_child,
  replace_with,
  set_text_content,
  set_node_value,
  get_node_value,
  set_element_value,
  set_element_checked,
  set_element_default_value,
  set_element_default_checked,
  get_attribute,
  remove_attribute,
  has_attribute,
  set_inner_html,
  clone_node,
  import_node,
  add_event_listener,
  remove_event_listener,
  dispatch_event,
  style_set_property,
  style_remove_property,
  set_css_text,
  class_list_toggle,
  invoke_error_boundary,
  validate_effect,
  effect_tracking,
  teardown,
  user_effect,
  user_pre_effect,
  eager_effect,
  effect_root,
  component_root,
  effect,
  legacy_pre_effect,
  legacy_pre_effect_reset,
  render_effect,
  template_effect,
  deferred_template_effect,
  block,
  managed,
  branch,
  destroy_effect,
  remove_effect_dom,
  pause_effect,
  resume_effect,
  move_effect,
  invalidate_inner_signals,
  autofocus,
  remove_textarea_child,
  add_form_reset_listener,
  listen,
  without_reactive_context,
  listen_to_event_and_reset_event,
  is_destroying_effect,
  active_reaction,
  set_active_reaction,
  active_effect,
  set_active_effect,
  update_version,
  get_abort_signal,
  tick,
  settled,
  get,
  safe_get,
  untrack,
  deep_read_state,
  deep_read
};
