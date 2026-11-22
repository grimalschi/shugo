import {
  await_reactivity_loss,
  derived_inert,
  false_default,
  hydration_mismatch
} from "./chunk-VCLKK7XB.js";

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

// node_modules/svelte/src/internal/shared/utils.js
var is_array = Array.isArray, index_of = Array.prototype.indexOf, includes = Array.prototype.includes, array_from = Array.from, object_keys = Object.keys, define_property = Object.defineProperty, get_descriptor = Object.getOwnPropertyDescriptor;
var object_prototype = Object.prototype, array_prototype = Array.prototype, get_prototype_of = Object.getPrototypeOf;
var has_own_property = Object.prototype.hasOwnProperty;
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
function push_array(array, items) {
  for (var i = 0; i < items.length; i++)
    array.push(items[i]);
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
var UNINITIALIZED = /* @__PURE__ */ Symbol("uninitialized"), FILENAME = /* @__PURE__ */ Symbol("filename"), HMR = /* @__PURE__ */ Symbol("hmr"), NAMESPACE_HTML = "http://www.w3.org/1999/xhtml";
var ATTACHMENT_KEY = "@attach";

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

// node_modules/svelte/src/internal/client/constants.js
var STATE_SYMBOL = /* @__PURE__ */ Symbol("$state"), COMPONENT_SYMBOL = /* @__PURE__ */ Symbol("component"), LEGACY_PROPS = /* @__PURE__ */ Symbol("legacy props");
var PROXY_PATH_SYMBOL = /* @__PURE__ */ Symbol("proxy path");
var STALE_REACTION = new class extends Error {
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

// node_modules/svelte/src/internal/client/errors.js
function derived_references_self() {
  if (false_default) {
    let error = new Error(`derived_references_self
A derived value cannot reference itself recursively
https://svelte.dev/e/derived_references_self`);
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/derived_references_self");
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
function hydration_failed() {
  if (false_default) {
    let error = new Error(`hydration_failed
Failed to hydrate the application
https://svelte.dev/e/hydration_failed`);
    throw error.name = "Svelte error", error;
  } else
    throw new Error("https://svelte.dev/e/hydration_failed");
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

// node_modules/svelte/src/internal/shared/dev.js
function get_error(label) {
  let error = new Error(), stack2 = get_stack();
  return stack2.length === 0 ? null : (stack2.unshift(`
`), define_property(error, "stack", {
    value: stack2.join(`
`)
  }), define_property(error, "name", {
    value: label
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
var dev_current_component_function = null;
function set_dev_current_component_function(fn) {
  dev_current_component_function = fn;
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

// node_modules/svelte/src/internal/client/custom-renderer/state.js
var current_renderer = null;
var depth = 0, suspended_hydration_at = [];
function push_renderer(value) {
  var previous = current_renderer;
  return current_renderer = value, depth += 1, hydrating && value !== null && (set_hydrating(!1), suspended_hydration_at.push(depth)), previous;
}
function pop_renderer(previous) {
  suspended_hydration_at[suspended_hydration_at.length - 1] === depth && (suspended_hydration_at.pop(), set_hydrating(!0)), depth -= 1, current_renderer = previous;
}

// node_modules/svelte/src/internal/client/dom/operations.js
var is_firefox, first_child_getter, next_sibling_getter;
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
function clear_text_content(node) {
  if (current_renderer) {
    for (var child = current_renderer.getFirstChild(node); child !== null; ) {
      var next2 = current_renderer.getNextSibling(child);
      current_renderer.remove(child), child = next2;
    }
    return;
  }
  node.textContent = "";
}
function create_element(tag2, namespace, is) {
  return current_renderer ? current_renderer.createElement(tag2) : namespace == null || namespace === NAMESPACE_HTML ? (
    /** @type {T extends keyof HTMLElementTagNameMap ? HTMLElementTagNameMap[T] : Element} */
    is ? document.createElement(tag2, { is }) : document.createElement(tag2)
  ) : (
    /** @type {T extends keyof HTMLElementTagNameMap ? HTMLElementTagNameMap[T] : Element} */
    is ? document.createElementNS(namespace, tag2, { is }) : document.createElementNS(namespace, tag2)
  );
}
function create_fragment() {
  return current_renderer ? (
    /** @type {DocumentFragment} */
    current_renderer.createFragment()
  ) : document.createDocumentFragment();
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
function get_parent_node(node) {
  return current_renderer ? (
    /** @type {TemplateNode | null} */
    current_renderer.getParent(node)
  ) : (
    /** @type {TemplateNode | null} */
    node.parentNode
  );
}
function append_child(parent, child) {
  return current_renderer ? (current_renderer.insert(parent, child, null), child) : parent.appendChild(child);
}
function insert_before(ref_node, new_node) {
  if (current_renderer) {
    var parent = current_renderer.getParent(ref_node);
    current_renderer.insert(parent, new_node, ref_node);
    return;
  }
  ref_node.before(new_node);
}
function remove_node(node) {
  if (current_renderer) {
    current_renderer.remove(node);
    return;
  }
  node.remove();
}
function remove_child(parent, child) {
  return current_renderer ? (current_renderer.remove(child), child) : (
    /** @type {ChildNode} */
    parent.removeChild(child)
  );
}
function get_node_value(node) {
  return current_renderer ? current_renderer.getText(node) : node.nodeValue;
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

// node_modules/svelte/src/internal/client/error-handling.js
var adjustments = /* @__PURE__ */ new WeakMap();
function handle_error(error) {
  var effect = active_effect;
  if (effect === null)
    return error;
  if (false_default && error instanceof Error && !adjustments.has(error) && adjustments.set(error, get_adjustments(error, effect)), (effect.f & 32768) === 0 && (effect.f & 4) === 0)
    throw false_default && !effect.parent && error instanceof Error && apply_adjustments(error), error;
  invoke_error_boundary(error, effect);
}
function invoke_error_boundary(error, effect) {
  if (error === HYDRATION_ERROR)
    throw error;
  if (!(effect !== null && (effect.f & 16384) !== 0)) {
    for (; effect !== null; ) {
      if ((effect.f & 128) !== 0 && (effect.f & 33570816) === 0) {
        if ((effect.f & 32768) === 0)
          throw error;
        try {
          effect.b.error(error);
          return;
        } catch (e) {
          error = e;
        }
      }
      effect = effect.parent;
    }
    throw false_default && error instanceof Error && apply_adjustments(error), error;
  }
}
function get_adjustments(error, effect) {
  let message_descriptor = get_descriptor(error, "message");
  if (!(message_descriptor && !message_descriptor.configurable)) {
    for (var indent = is_firefox ? "  " : "	", component_stack = `
${indent}in ${effect.fn?.name || "<unknown>"}`, context = effect.ctx; context !== null; )
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

// node_modules/svelte/src/internal/client/reactivity/status.js
var STATUS_MASK = -7169;
function set_signal_status(signal, status) {
  signal.f = signal.f & STATUS_MASK | status;
}
function update_derived_status(derived2) {
  (derived2.f & 512) !== 0 || derived2.edges === null ? set_signal_status(derived2, 1024) : set_signal_status(derived2, 4096);
}

// node_modules/svelte/src/internal/client/reactivity/async.js
function merge_target(batch) {
  for (; batch.merged !== null; ) batch = batch.merged;
  return batch;
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
    for (let effect of this.#pending.keys())
      if (this.is_blocked_by(effect))
        return !0;
    return !1;
  }
  /**
   * Whether `effect` is a branch that this batch skips, or that only exists offscreen for other batches
   * @param {Effect} effect
   */
  #skips(effect) {
    return this.#skipped_branches.has(effect) || (effect.f & 33554432) !== 0 && !this.#unskipped_branches.has(effect);
  }
  /**
   * Whether `effect` is inside a branch that this batch skips, i.e. doesn't exist in its view of the world
   * @param {Effect} effect
   */
  is_skipped(effect) {
    for (var e = (
      /** @type {Effect | null} */
      effect
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
  is_blocked_by(effect, root = null) {
    for (var e = (
      /** @type {Effect | null} */
      effect
    ); e !== null && e !== root; e = e.parent)
      if (this.#skips(e) || (e.f & 16512) === 128 && /** @type {Boundary} */
      e.b.is_pending)
        return !1;
    return !0;
  }
  /**
   * @param {Effect} effect
   */
  skip_effect(effect) {
    this.#skipped_branches.add(effect), this.#unskipped_branches.delete(effect);
  }
  /**
   * @param {Effect} effect
   */
  unskip_effect(effect) {
    (this.#skipped_branches.delete(effect) || !this.#unskipped_branches.has(effect) && ((effect.f & 33554432) !== 0 || this.folded.some((batch) => batch.#skipped_branches.has(effect)))) && this.revalidate_branch(effect), this.#unskipped_branches.add(effect);
  }
  /**
   * Like `unskip_effect`, for a branch that was just created offscreen while this batch is processed.
   * Its effects ran in this batch's view of the world when they were created, so there's nothing to re-validate
   * TODO can we skip revalidation in `unskip_effect` in more cases, by detecting whether things changed
   * inside a branch while it was skipped?
   * @param {Effect} effect
   */
  unskip_new_effect(effect) {
    this.#unskipped_branches.add(effect);
  }
  /**
   * @param {Effect} effect
   */
  revalidate_branch(effect) {
    for (var e = effect.first; e !== null; e = e.next) {
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
    for (let effect of this.#scheduled)
      if (!((effect.f & 16384) !== 0 || (effect.f & 6144) === 0)) {
        for (var e = effect, covered = !1; e.parent !== null; ) {
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
        for (let effect of render_effects) this.#scheduled.push(effect);
        for (let effect of effects) this.#scheduled.push(effect);
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
          var effect = (
            /** @type {Effect} */
            reaction
          );
          effects.has(effect) || (effects.add(effect), (flags & 155648) === 0 && // branches that this batch skips don't exist in its view of the world
          !this.is_skipped(effect) && this.schedule(effect, 4096));
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
    for (var effect = root.first; effect !== null; ) {
      var flags = effect.f, is_branch = (flags & 96) !== 0, is_skippable_branch = is_branch && (flags & 1024) !== 0, skip = is_skippable_branch || (flags & 139264) !== 0 || this.#skips(effect);
      if (!skip && effect.fn !== null) {
        is_branch ? effect.f ^= 1024 : (flags & 4) !== 0 ? effects.push(effect) : async_mode_flag && (flags & 16777224) !== 0 ? render_effects.push(effect) : is_dirty(effect) && update_effect(effect);
        var child = effect.first;
        if (child !== null) {
          effect = child;
          continue;
        }
      }
      for (; effect !== null; ) {
        if ((effect.f & 16512) === 128) {
          var boundary = (
            /** @type {Boundary} */
            effect.b
          );
          boundary.is_pending && boundary.is_ready(this) && revealed_boundaries.push(boundary);
        }
        var next2 = effect.next;
        if (next2 !== null) {
          effect = next2;
          break;
        }
        effect = effect.parent;
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
    for (let [effect, deferred2] of batch.async_deriveds) {
      let d = this.async_deriveds.get(effect);
      d && d !== deferred2 && deferred2.promise.then(d.resolve, d.reject), this.async_deriveds.set(effect, deferred2);
    }
    for (let [reaction, record] of batch.records ?? [])
      if ((reaction.f & 16) === 0) {
        var replaced = this.records?.get(reaction);
        (this.records ??= /* @__PURE__ */ new Map()).set(reaction, record), batch.records?.delete(reaction), abort_record(reaction, replaced);
      }
    if (batch.async_deriveds.clear(), batch.#pending.size > 0) {
      for (let [effect, count] of batch.#pending)
        this.#pending.set(effect, (this.#pending.get(effect) ?? 0) + count);
      batch.#pending.clear();
    }
    if (batch.#unprocessed.size > 0) {
      for (let [effect, status] of batch.#unprocessed)
        this.#add_dirty_effect(effect, status);
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
  #add_dirty_effect(effect, status) {
    (status === 2048 || this.#unprocessed.get(effect) !== 2048) && this.#unprocessed.set(effect, status);
  }
  /**
   * @param {Effect[]} effects
   */
  #defer_effects(effects) {
    for (var i = 0; i < effects.length; i += 1) {
      var effect = effects[i], status = effect.f & 6144;
      status !== 0 && (this.#add_dirty_effect(effect, effect.dv === null ? 4096 : status), set_signal_status(effect, 1024));
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
  increment(effect) {
    var count = this.#pending.get(effect) ?? 0;
    this.#pending.set(effect, count + 1);
  }
  /**
   * @param {Effect} effect
   */
  decrement(effect) {
    var count = this.#pending.get(effect) ?? 0;
    count === 1 ? this.#pending.delete(effect) : this.#pending.set(effect, count - 1), !this.#decrement_queued && (this.#decrement_queued = !0, queue_micro_task(() => {
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
            var effect = (
              /** @type {Effect} */
              reaction
            );
            (flags & 16) !== 0 && eager_block_effects !== null && (set_signal_status(effect, status), eager_block_effects.add(effect)), updated_during_traversal !== null ? (set_signal_status(effect, status), updated_during_traversal.push(effect)) : this.schedule(effect, status);
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
  schedule(effect, status) {
    if (last_scheduled_effect = effect, view_batch !== null && view_batch !== this && effect !== active_reaction) {
      this.#add_dirty_effect(
        effect,
        (effect.f & 2097152) !== 0 ? status : 4096
      );
      return;
    }
    if (processing_batch !== this && active_reaction === null) {
      if (this.prev !== null || this.next !== null || this.is_fork) {
        this.#add_dirty_effect(effect, 4096);
        return;
      }
      status = 4096;
    }
    this.#add(effect, status);
  }
  /**
   * @param {Effect} effect
   * @param {number} status
   */
  #add(effect, status) {
    (effect.f & 2048) === 0 && set_signal_status(effect, status), this.#scheduled.push(effect);
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
function defer_if_pending(effect) {
  if ((effect.f & 32772) !== 4) return !1;
  for (var boundary = effect.b; boundary !== null; boundary = boundary.parent)
    if (boundary.is_pending)
      return boundary.defer_effect(effect), !0;
  return !1;
}
function flush_queued_effects(effects, earlier_writes = null) {
  var length = effects.length;
  if (length === 0) return earlier_writes;
  for (var i = 0; i < length; ) {
    var effect = effects[i++];
    if ((effect.f & 24576) === 0 && // don't run if already marked for destruction in the next batch
    // (this can happen if an effect writes a value which e.g. makes an if block falsy)
    !current_batch?.is_skipped(effect) && !defer_if_pending(effect) && is_dirty(effect)) {
      eager_block_effects = /* @__PURE__ */ new Set();
      var version = write_version;
      if (update_effect(effect), effect.edges === null && effect.first === null && effect.nodes === null && effect.teardown === null && effect.ac === null && unlink_effect(effect), eager_block_effects?.size > 0) {
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
function in_real_world(fn) {
  var previous_batch_values = batch_values, previous_view_cache = view_cache, previous_view_batch = view_batch;
  batch_values = view_cache = view_batch = null;
  try {
    fn();
  } finally {
    batch_values = previous_batch_values, view_cache = previous_view_cache, view_batch = previous_view_batch;
  }
}
function reset_branch(effect, force = !1) {
  if (!(!force && (effect.f & 32) !== 0 && (effect.f & 1024) !== 0)) {
    set_signal_status(effect, 1024);
    for (var e = effect.first; e !== null; )
      reset_branch(e, force), e = e.next;
  }
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
  for (let effect of eager_effects) {
    (effect.f & 1024) !== 0 && set_signal_status(effect, 4096);
    let dirty;
    try {
      dirty = is_dirty(effect);
    } catch {
      dirty = !0;
    }
    dirty && update_effect(effect);
  }
  eager_effects.clear();
}
function increment(source2) {
  set(source2, source2.v + 1);
}

// node_modules/svelte/src/internal/client/reactivity/deriveds.js
var reactivity_loss_tracker = null;
var recent_async_deriveds = /* @__PURE__ */ new Set();
var OBSOLETE = /* @__PURE__ */ Symbol("obsolete");
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

// node_modules/svelte/src/internal/client/legacy.js
var captured_signals = null;

// node_modules/svelte/src/internal/client/dom/elements/bindings/shared.js
function without_reactive_context(fn) {
  var previous_reaction = active_reaction, previous_effect = active_effect;
  set_active_reaction(null), set_active_effect(null);
  try {
    return fn();
  } finally {
    set_active_reaction(previous_reaction), set_active_effect(previous_effect);
  }
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
function set_active_effect(effect) {
  active_effect = effect;
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
      var run2 = (
        /** @type {Derived} */
        dep
      );
      if (batch_values === null ? update_derived(
        /** @type {Derived} */
        dep
      ) : run2 = get_view(
        /** @type {Derived} */
        dep
      ), run2.wv > wv) {
        var assigned = run2 === dep ? (dep.f & 67108864) !== 0 : !("dv" in run2);
        if (assigned) return !0;
        if (unchanged === null)
          unchanged = /* @__PURE__ */ new Set();
        else if (unchanged.has(run2))
          continue;
        if (changed_since(
          /** @type {DerivedRecord} */
          run2.deps,
          wv,
          unchanged
        ))
          return !0;
        unchanged.add(run2);
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
    var run2 = run_of(reaction), deps = run2.deps, dv = run2.dv;
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
function schedule_possible_effect_self_invalidation(signal, effect, root = !0) {
  var reactions = signal.reactions;
  if (reactions !== null && !(!async_mode_flag && current_sources !== null && current_sources.has(signal)))
    for (var i = 0; i < reactions.length; i++) {
      var reaction = reactions[i];
      (reaction.f & 2) !== 0 ? schedule_possible_effect_self_invalidation(
        /** @type {Derived} */
        reaction,
        effect,
        !1
      ) : effect === reaction && current_batch.schedule(
        /** @type {Effect} */
        reaction,
        root ? 2048 : 4096
      );
    }
}
var abort_controller = null;
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
function prune_edges(effect) {
  var edges = effect.edges, deps = effect.deps;
  if (!(edges === deps || has_batch_record(effect))) {
    for (
      var dep of
      /** @type {Value[]} */
      edges
    )
      (deps === null || !includes.call(deps, dep)) && remove_reaction(effect, dep);
    effect.edges = deps;
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
function update_effect(effect) {
  var flags = effect.f;
  if ((flags & 16384) === 0) {
    set_signal_status(effect, 1024);
    var previous_effect = active_effect;
    active_effect = effect;
    var previous_renderer = custom_renderers_flag ? push_renderer(effect.r) : null;
    if (false_default) {
      var previous_component_fn = dev_current_component_function;
      set_dev_current_component_function(effect.component_function);
      var previous_stack = (
        /** @type {any} */
        dev_stack
      );
      set_dev_stack(effect.dev_stack ?? dev_stack);
    }
    try {
      (flags & 16777232) !== 0 ? destroy_block_effect_children(effect) : destroy_effect_children(effect), execute_effect_teardown(effect);
      var teardown2 = update_reaction(effect);
      if (effect.teardown = typeof teardown2 == "function" ? teardown2 : null, effect.wv = write_version, false_default && tracing_mode_flag && (effect.f & 2048) !== 0 && effect.edges !== null)
        for (var dep of effect.edges)
          dep.set_during_effect && (dep.wv = increment_write_version(), dep.set_during_effect = !1);
    } finally {
      active_effect = previous_effect, custom_renderers_flag && pop_renderer(previous_renderer), false_default && (set_dev_current_component_function(previous_component_fn), set_dev_stack(previous_stack));
    }
  }
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
      var trace = get_error("traced at");
      trace && console.warn(trace);
    }
    if (recent_async_deriveds.delete(signal), tracing_mode_flag && !untracking && tracing_expressions !== null && active_reaction !== null && tracing_expressions.reaction === active_reaction) {
      if (signal.trace)
        signal.trace();
      else if (trace = get_error("traced at"), trace) {
        var entry = tracing_expressions.entries.get(signal);
        entry === void 0 && (entry = { traces: [] }, tracing_expressions.entries.set(signal, entry));
        var last = entry.traces[entry.traces.length - 1];
        trace.stack !== last?.stack && entry.traces.push(trace);
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
  var run2 = run_of(reaction), deps = run2.deps;
  run2.dv !== null && (deps === null || !includes.call(deps, signal)) && (run2.deps = deps === null ? [signal] : [...deps, signal], run2.dv.push(version));
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
function untrack(fn) {
  var previous_untracking = untracking;
  try {
    return untracking = !0, fn();
  } finally {
    untracking = previous_untracking;
  }
}

// node_modules/svelte/src/internal/client/reactivity/effects.js
function push_effect(effect, parent_effect) {
  var parent_last = parent_effect.last;
  parent_last === null ? parent_effect.last = parent_effect.first = effect : (parent_last.next = effect, effect.prev = parent_last, parent_effect.last = effect);
}
function create_effect(type, fn) {
  var parent = active_effect;
  if (false_default)
    for (; parent !== null && (parent.f & 131072) !== 0; )
      parent = parent.parent;
  parent !== null && (parent.f & 8192) !== 0 && (type |= 8192);
  var effect = {
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
  false_default && (effect.component_function = dev_current_component_function);
  var e = effect;
  if ((type & 4) !== 0)
    collected_effects !== null ? collected_effects.push(effect) : Batch.ensure().schedule(effect, 2048);
  else if (fn !== null) {
    try {
      update_effect(effect);
    } catch (e2) {
      throw destroy_effect(effect), e2;
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
  return effect;
}
function effect_tracking() {
  return active_reaction !== null && !untracking;
}
function teardown(fn) {
  let effect = create_effect(8, null);
  return set_signal_status(effect, 1024), effect.teardown = fn, effect;
}
function create_user_effect(fn) {
  return create_effect(1048580, fn);
}
function effect_root(fn) {
  Batch.ensure();
  let effect = create_effect(524352, fn);
  return () => {
    destroy_effect(effect);
  };
}
function component_root(fn) {
  Batch.ensure();
  let effect = create_effect(524352, fn);
  return (options = {}) => new Promise((fulfil) => {
    options.outro ? pause_effect(effect, () => {
      destroy_effect(effect), fulfil(void 0);
    }) : (destroy_effect(effect), fulfil(void 0));
  });
}
function render_effect(fn, flags = 0) {
  return create_effect(8 | flags, fn);
}
function block(fn, flags = 0) {
  var effect = create_effect(16 | flags, fn);
  return false_default && (effect.dev_stack = dev_stack), effect;
}
function branch(fn) {
  return create_effect(524320, fn);
}
function execute_effect_teardown(effect) {
  var teardown2 = effect.teardown;
  if (teardown2 !== null) {
    let previously_destroying_effect = is_destroying_effect, previous_teardown_version = teardown_version, previous_teardown_context = teardown_context, previous_reaction = active_reaction;
    set_is_destroying_effect(!0), set_active_reaction(null);
    try {
      teardown2.call(null);
    } catch (error) {
      invoke_error_boundary(error, effect.parent);
    } finally {
      set_is_destroying_effect(previously_destroying_effect, previous_teardown_version), set_teardown_context(previous_teardown_context), set_active_reaction(previous_reaction);
    }
  }
}
function destroy_effect_children(signal, remove_dom = !1) {
  var effect = signal.first;
  for (signal.first = signal.last = null; effect !== null; ) {
    abort_all(effect);
    var next2 = effect.next;
    (effect.f & 64) !== 0 ? effect.parent = null : destroy_effect(effect, remove_dom), effect = next2;
  }
}
function destroy_block_effect_children(signal) {
  for (var effect = signal.first; effect !== null; ) {
    var next2 = effect.next;
    (effect.f & 32) === 0 && destroy_effect(effect), effect = next2;
  }
}
function destroy_effect(effect, remove_dom = !0) {
  var removed = !1, previous_renderer = custom_renderers_flag ? push_renderer(effect.r) : null;
  try {
    (remove_dom || (effect.f & 262144) !== 0) && effect.nodes !== null && effect.nodes.end !== null && (remove_effect_dom(
      effect.nodes.start,
      /** @type {TemplateNode} */
      effect.nodes.end
    ), removed = !0), effect.f |= 33554432, destroy_effect_children(effect, remove_dom && !removed), remove_reactions(effect);
    var transitions = effect.nodes && effect.nodes.t;
    if (transitions !== null)
      for (let transition of transitions)
        transition.stop();
    execute_effect_teardown(effect);
  } finally {
    custom_renderers_flag && pop_renderer(previous_renderer);
  }
  effect.f ^= 33554432, effect.f |= 16384;
  var parent = effect.parent;
  parent !== null && parent.first !== null && unlink_effect(effect), false_default && (effect.component_function = null), effect.next = effect.prev = effect.teardown = effect.ctx = effect.deps = effect.dv = effect.edges = effect.fn = effect.nodes = effect.ac = effect.b = effect.r = null;
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
function unlink_effect(effect) {
  var parent = effect.parent, prev = effect.prev, next2 = effect.next;
  prev !== null && (prev.next = next2), next2 !== null && (next2.prev = prev), parent !== null && (parent.first === effect && (parent.first = next2), parent.last === effect && (parent.last = prev));
}
function pause_effect(effect, callback, destroy = !0) {
  var transitions = [];
  effect.f |= 256, pause_children(effect, transitions, !0);
  var fn = () => {
    destroy && destroy_effect(effect), callback && callback();
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
function pause_children(effect, transitions, local) {
  if ((effect.f & 8192) === 0) {
    effect.f ^= 8192;
    var t = effect.nodes && effect.nodes.t;
    if (t !== null)
      for (let transition of t)
        (transition.is_global || local) && transitions.push(transition);
    for (var child = effect.first; child !== null; ) {
      var sibling = child.next;
      if ((child.f & 64) === 0) {
        var transparent = (child.f & 65536) !== 0 || // If this is a branch effect without a block effect parent,
        // it means the parent block effect was pruned. In that case,
        // transparency information was transferred to the branch effect.
        (child.f & 32) !== 0 && (effect.f & 16) !== 0;
        pause_children(child, transitions, transparent ? local : !1);
      }
      child = sibling;
    }
  }
}
function move_effect(effect, fragment) {
  if (effect.nodes) {
    var previous_renderer = custom_renderers_flag ? push_renderer(effect.r) : null;
    try {
      for (var node = effect.nodes.start, end = effect.nodes.end; node !== null; ) {
        var next2 = node === end ? null : get_next_sibling(node);
        append_child(fragment, node), node = next2;
      }
    } finally {
      custom_renderers_flag && pop_renderer(previous_renderer);
    }
  }
}

// node_modules/svelte/src/internal/client/dev/tracing.js
var tracing_expressions = null;
function tag(source2, label) {
  return source2.label = label, tag_proxy(source2.v, label), source2;
}
function tag_proxy(value, label) {
  return value?.[PROXY_PATH_SYMBOL]?.(label), value;
}

export {
  experimental_async_required,
  invalid_default_snippet,
  invalid_snippet_arguments,
  set_context_after_init,
  snippet_without_render_tag,
  store_invalid_shape,
  svelte_element_invalid_this_value,
  create_context,
  get_or_init_context_map,
  is_array,
  array_from,
  object_keys,
  define_property,
  has_own_property,
  noop,
  is_promise,
  run,
  run_all,
  push_array,
  deferred,
  fallback,
  to_array,
  exclude_from_object,
  async_mode_flag,
  custom_renderers_flag,
  HYDRATION_ERROR,
  UNINITIALIZED,
  FILENAME,
  HMR,
  ATTACHMENT_KEY,
  get_stack,
  LEGACY_PROPS,
  STALE_REACTION,
  COMMENT_NODE,
  hydrating,
  set_hydrating,
  hydrate_node,
  set_hydrate_node,
  hydrate_next,
  next,
  skip_nodes,
  read_hydration_instruction,
  safe_not_equal,
  hydration_failed,
  rune_outside_svelte,
  svelte_boundary_reset_onerror,
  dynamic_void_element_content,
  snapshot,
  queue_micro_task,
  set_signal_status,
  component_context,
  set_component_context,
  push,
  pop,
  mark_as_component,
  current_renderer,
  push_renderer,
  pop_renderer,
  current_batch,
  Batch,
  flushSync,
  tag,
  source,
  state,
  mutable_source,
  set,
  internal_set,
  increment,
  create_text,
  get_first_child,
  get_next_sibling,
  clear_text_content,
  create_element,
  create_fragment,
  node_type,
  get_parent_node,
  append_child,
  insert_before,
  remove_child,
  get_node_value,
  add_event_listener,
  remove_event_listener,
  invoke_error_boundary,
  effect_tracking,
  teardown,
  effect_root,
  component_root,
  render_effect,
  block,
  branch,
  destroy_effect,
  pause_effect,
  move_effect,
  without_reactive_context,
  active_reaction,
  set_active_reaction,
  active_effect,
  set_active_effect,
  get,
  untrack
};
