import {
  can_delegate_event,
  create_event,
  delegate,
  delegated,
  hash,
  is_capture_event,
  is_raw_text_element,
  is_void,
  listen_to_delegated_events,
  normalize_attribute,
  sanitize_location
} from "./chunk-ZMX3RH2Y.js";
import {
  createSubscriber,
  get as get2,
  subscribe_to_store
} from "./chunk-R2NOJYQV.js";
import {
  ATTACHMENT_KEY,
  ATTRIBUTES_CACHE,
  Batch,
  CLASS_CACHE,
  COMMENT_NODE,
  DOCUMENT_FRAGMENT_NODE,
  ELEMENT_NODE,
  FILENAME,
  FORM_RESET_HANDLER,
  HMR,
  HMR_ANCHOR,
  HYDRATION_ERROR,
  IS_XHTML,
  LEGACY_PROPS,
  LOADING_ATTR_SYMBOL,
  NAMESPACE_HTML,
  NAMESPACE_MATHML,
  NAMESPACE_SVG,
  STATE_SYMBOL,
  STYLE_CACHE,
  TEXT_CACHE,
  TEXT_NODE,
  UNINITIALIZED,
  active_effect,
  active_reaction,
  add_form_reset_listener,
  append_child,
  array_from,
  async_mode_flag,
  autofocus,
  bind_invalid_checkbox_value,
  block,
  branch,
  capture,
  class_list_toggle,
  clear_text_content,
  clone_node,
  component_api_changed,
  component_api_invalid_new,
  component_context,
  component_root,
  create_comment,
  create_element,
  create_fragment,
  create_text,
  current_batch,
  current_renderer,
  custom_renderers_flag,
  deep_read_state,
  define_property,
  derived,
  derived_safe_equal,
  destroy_effect,
  dev_current_component_function,
  dev_stack,
  dynamic_void_element_content,
  each_key_duplicate,
  each_key_volatile,
  eager_effect,
  effect,
  effect_pending_outside_reaction,
  effect_root,
  experimental_async_required,
  flatten,
  flushSync,
  get,
  get_abort_signal,
  get_abort_signal_outside_reaction,
  get_attribute,
  get_descriptor,
  get_descriptors,
  get_error,
  get_first_child,
  get_last_child,
  get_next_sibling,
  get_node_value,
  get_parent_node,
  get_prototype_of,
  has_attribute,
  has_own_property,
  hydratable_missing_but_required,
  hydrate_next,
  hydrate_node,
  hydrating,
  hydration_failed,
  import_node,
  insert_before,
  internal_set,
  invalid_snippet,
  invalid_snippet_arguments,
  invalid_snippet_in_custom_renderer,
  invoke_error_boundary,
  is,
  is_array,
  is_destroying_effect,
  is_firefox,
  is_flushing_sync,
  is_function,
  is_promise,
  is_runes,
  legacy_mode_flag,
  lifecycle_legacy_only,
  lifecycle_outside_component,
  listen,
  listen_to_event_and_reset_event,
  managed,
  mark_as_component,
  merge_target,
  merge_text_nodes,
  move_effect,
  mutable_source,
  next,
  node_name,
  node_type,
  noop,
  object_keys,
  pause_effect,
  pop,
  pop_renderer,
  props_invalid_value,
  props_rest_readonly,
  proxy,
  push,
  push_renderer,
  queue_micro_task,
  read_hydration_instruction,
  remove_attribute,
  remove_child,
  remove_effect_dom,
  remove_event_listener,
  remove_node,
  render_effect,
  renderer_cannot_cross,
  replace_with,
  reset,
  resume_effect,
  run,
  run_after_blockers,
  run_all,
  rune_outside_svelte,
  safe_not_equal,
  set,
  set_active_effect,
  set_active_reaction,
  set_attribute,
  set_before_effects,
  set_component_context,
  set_css_text,
  set_dev_current_component_function,
  set_element_checked,
  set_element_default_checked,
  set_element_default_value,
  set_element_value,
  set_hydrate_node,
  set_hydrating,
  set_inner_html,
  set_node_value,
  set_signal_status,
  set_text_content,
  should_defer_append,
  skip_nodes,
  snapshot,
  snippet_renderer_mismatch,
  snippet_without_render_tag,
  source,
  store_invalid_shape,
  style_remove_property,
  style_set_property,
  svelte_boundary_reset_onerror,
  svelte_element_invalid_this_value,
  tag,
  teardown,
  template_effect,
  tick,
  unset_context,
  untrack,
  update,
  user_effect,
  user_pre_effect,
  validate_effect,
  view_batch,
  without_reactive_context
} from "./chunk-SJURP6G5.js";
import {
  assignment_value_stale,
  binding_property_non_reactive,
  console_log_state,
  false_default,
  hydratable_missing_but_expected,
  hydration_attribute_changed,
  hydration_html_changed,
  hydration_mismatch,
  invalid_raw_snippet_render,
  lifecycle_double_unmount,
  ownership_invalid_binding,
  ownership_invalid_mutation,
  select_multiple_invalid_value,
  svelte_boundary_reset_noop,
  true_default
} from "./chunk-CG7SAXEK.js";

// node_modules/svelte/src/internal/client/dev/assign.js
function compare(a, b, property, location) {
  return a !== b && typeof b == "object" && STATE_SYMBOL in b && assignment_value_stale(
    property,
    /** @type {string} */
    sanitize_location(location)
  ), a;
}
function assign(object, property, operator, rhs, location) {
  return compare(
    operator === "=" ? object[property] = rhs : operator === "&&=" ? object[property] &&= rhs() : operator === "||=" ? object[property] ||= rhs() : operator === "??=" ? object[property] ??= rhs() : null,
    untrack(() => object[property]),
    property,
    location
  );
}
async function assign_async(object, property, operator, rhs, location) {
  return compare(
    operator === "=" ? object[property] = await rhs : operator === "&&=" ? object[property] &&= await rhs() : operator === "||=" ? object[property] ||= await rhs() : operator === "??=" ? object[property] ??= await rhs() : null,
    untrack(() => object[property]),
    property,
    location
  );
}

// node_modules/svelte/src/internal/client/dev/css.js
var all_styles = /* @__PURE__ */ new Map();
function register_style(hash2, style) {
  var styles = all_styles.get(hash2);
  styles || (styles = /* @__PURE__ */ new Set(), all_styles.set(hash2, styles)), styles.add(style);
}
function cleanup_styles(hash2) {
  var styles = all_styles.get(hash2);
  if (styles) {
    for (let style of styles)
      remove_node(style);
    all_styles.delete(hash2);
  }
}

// node_modules/svelte/src/internal/client/dev/elements.js
function add_locations(fn, filename, locations) {
  var create = () => {
    var dom = fn();
    return hydrating || assign_locations(
      node_type(dom) === DOCUMENT_FRAGMENT_NODE ? get_first_child(dom) : dom,
      filename,
      locations
    ), dom;
  };
  return create.hydrate_assign_locations = (node) => {
    hydrating && assign_locations(node, filename, locations);
  }, create;
}
function assign_location(element2, filename, location) {
  element2.__svelte_meta = {
    parent: dev_stack,
    loc: { file: filename, line: location[0], column: location[1] },
    component: component_context
  }, location[2] && assign_locations(get_first_child(element2), filename, location[2]);
}
function assign_locations(node, filename, locations) {
  for (var i = 0, depth = 0; node && i < locations.length; ) {
    if (hydrating && node_type(node) === COMMENT_NODE) {
      var comment2 = (
        /** @type {Comment} */
        node
      );
      let data = get_node_value(comment2) ?? "";
      data[0] === "[" ? depth += 1 : data[0] === "]" && (depth -= 1);
    }
    depth === 0 && node_type(node) === ELEMENT_NODE && !node.__svelte_meta && assign_location(
      /** @type {Element} */
      node,
      filename,
      locations[i++]
    ), node = get_next_sibling(node);
  }
}

// node_modules/svelte/src/internal/client/dom/reconciler.js
var policy = (
  // We gotta write it like this because after downleveling the pure comment may end up in the wrong location
  globalThis?.window?.trustedTypes && /* @__PURE__ */ globalThis.window.trustedTypes.createPolicy("svelte-trusted-html", {
    /** @param {string} html */
    createHTML: (html2) => html2
  })
);
function create_trusted_html(html2) {
  return (
    /** @type {string} */
    policy?.createHTML(html2) ?? html2
  );
}
function create_fragment_from_html(html2) {
  var elem = create_element("template");
  return set_inner_html(elem, create_trusted_html(html2.replaceAll("<!>", "<!---->"))), elem.content;
}

// node_modules/svelte/src/internal/client/dom/template.js
var TEMPLATE_TAG = IS_XHTML ? "template" : "TEMPLATE", SCRIPT_TAG = IS_XHTML ? "script" : "SCRIPT";
function assign_nodes(start, end) {
  var effect2 = (
    /** @type {Effect} */
    active_effect
  );
  effect2.nodes === null && (effect2.nodes = { start, end, a: null, t: null });
}
// @__NO_SIDE_EFFECTS__
function from_html(content, flags2) {
  var is_fragment = (flags2 & 1) !== 0, use_import_node = (flags2 & 2) !== 0, node, has_start = !content.startsWith("<!>");
  return () => {
    if (hydrating)
      return assign_nodes(hydrate_node, null), hydrate_node;
    node === void 0 && (node = create_fragment_from_html(has_start ? content : "<!>" + content), is_fragment || (node = /** @type {TemplateNode} */
    get_first_child(node)));
    var clone = (
      /** @type {TemplateNode} */
      use_import_node || is_firefox ? import_node(node, !0) : clone_node(node, !0)
    );
    if (is_fragment) {
      var start = (
        /** @type {TemplateNode} */
        get_first_child(clone)
      ), end = (
        /** @type {TemplateNode} */
        get_last_child(clone)
      );
      assign_nodes(start, end);
    } else
      assign_nodes(clone, clone);
    return clone;
  };
}
// @__NO_SIDE_EFFECTS__
function from_namespace(content, flags2, ns = "svg") {
  var has_start = !content.startsWith("<!>"), is_fragment = (flags2 & 1) !== 0, wrapped = `<${ns}>${has_start ? content : "<!>" + content}</${ns}>`, node;
  return () => {
    if (hydrating)
      return assign_nodes(hydrate_node, null), hydrate_node;
    if (!node) {
      var fragment = (
        /** @type {DocumentFragment} */
        create_fragment_from_html(wrapped)
      ), root = (
        /** @type {Element} */
        get_first_child(fragment)
      );
      if (is_fragment)
        for (node = create_fragment(); get_first_child(root); )
          append_child(
            node,
            /** @type {TemplateNode} */
            get_first_child(root)
          );
      else
        node = /** @type {Element} */
        get_first_child(root);
    }
    var clone = (
      /** @type {TemplateNode} */
      clone_node(node, !0)
    );
    if (is_fragment) {
      var start = (
        /** @type {TemplateNode} */
        get_first_child(clone)
      ), end = (
        /** @type {TemplateNode} */
        get_last_child(clone)
      );
      assign_nodes(start, end);
    } else
      assign_nodes(clone, clone);
    return clone;
  };
}
// @__NO_SIDE_EFFECTS__
function from_svg(content, flags2) {
  return /* @__PURE__ */ from_namespace(content, flags2, "svg");
}
// @__NO_SIDE_EFFECTS__
function from_mathml(content, flags2) {
  return /* @__PURE__ */ from_namespace(content, flags2, "math");
}
function fragment_from_tree(structure, ns) {
  var fragment = create_fragment();
  for (var item of structure) {
    if (typeof item == "string") {
      append_child(fragment, create_text(item));
      continue;
    }
    if (item === void 0 || item[0][0] === "/") {
      append_child(fragment, create_comment(item ? item[0].slice(3) : ""));
      continue;
    }
    let [name, attributes, ...children] = item, namespace = name === "svg" ? NAMESPACE_SVG : name === "math" ? NAMESPACE_MATHML : ns;
    var element2 = create_element(name, namespace, attributes?.is);
    for (var key2 in attributes)
      set_attribute(element2, key2, attributes[key2]);
    if (children.length > 0) {
      var target = node_name(element2) === TEMPLATE_TAG ? (
        // this is safe for custom renderers because the name will never be `template` due to how `node_name` works
        /** @type {HTMLTemplateElement} */
        element2.content
      ) : element2;
      append_child(
        target,
        fragment_from_tree(children, node_name(element2) === "foreignObject" ? void 0 : namespace)
      );
    }
    append_child(fragment, element2);
  }
  return fragment;
}
// @__NO_SIDE_EFFECTS__
function from_tree(structure, flags2) {
  var is_fragment = (flags2 & 1) !== 0, use_import_node = (flags2 & 2) !== 0;
  let ns = (flags2 & 4) !== 0 ? NAMESPACE_SVG : (flags2 & 8) !== 0 ? NAMESPACE_MATHML : void 0;
  var node;
  function create() {
    var fragment = fragment_from_tree(structure, ns);
    return is_fragment ? fragment : (
      /** @type {TemplateNode} */
      get_first_child(fragment)
    );
  }
  return () => {
    if (hydrating)
      return assign_nodes(hydrate_node, null), hydrate_node;
    var clone;
    if (current_renderer !== null ? clone = /** @type {TemplateNode} */
    create() : (node ??= create(), clone = /** @type {TemplateNode} */
    use_import_node || is_firefox ? import_node(node, !0) : clone_node(node, !0)), is_fragment) {
      var start = (
        /** @type {TemplateNode} */
        get_first_child(clone)
      ), end = (
        /** @type {TemplateNode} */
        get_last_child(clone)
      );
      assign_nodes(start, end);
    } else
      assign_nodes(clone, clone);
    return clone;
  };
}
function with_script(fn) {
  return () => run_scripts(fn());
}
function run_scripts(node) {
  if (hydrating) return node;
  let is_fragment = node_type(node) === DOCUMENT_FRAGMENT_NODE, scripts = node_name(node) === SCRIPT_TAG ? [
    /** @type {HTMLScriptElement} */
    node
  ] : node.querySelectorAll("script"), effect2 = (
    /** @type {Effect & { nodes: EffectNodes }} */
    active_effect
  );
  for (let script of scripts) {
    let clone = create_element("script");
    for (var attribute of script.attributes)
      set_attribute(clone, attribute.name, attribute.value);
    set_text_content(clone, script.textContent ?? ""), (is_fragment ? get_first_child(node) === script : node === script) && (effect2.nodes.start = clone), (is_fragment ? get_last_child(node) === script : node === script) && (effect2.nodes.end = clone), replace_with(script, clone);
  }
  return node;
}
function text(value = "") {
  if (!hydrating) {
    var t = create_text(value + "");
    return assign_nodes(t, t), t;
  }
  var node = hydrate_node;
  return node_type(node) !== TEXT_NODE ? (insert_before(node, node = create_text()), set_hydrate_node(node)) : merge_text_nodes(
    /** @type {Text} */
    node
  ), assign_nodes(node, node), node;
}
function comment() {
  if (hydrating)
    return assign_nodes(hydrate_node, null), hydrate_node;
  var frag = create_fragment(), start = create_comment(""), anchor = create_text();
  return append_child(frag, start), append_child(frag, anchor), assign_nodes(start, anchor), frag;
}
function append(anchor, dom) {
  if (hydrating) {
    var effect2 = (
      /** @type {Effect & { nodes: EffectNodes }} */
      active_effect
    );
    ((effect2.f & 32768) === 0 || effect2.nodes.end === null) && (effect2.nodes.end = hydrate_node), hydrate_next();
    return;
  }
  anchor !== null && insert_before(
    anchor,
    /** @type {Node} */
    dom
  );
}
function props_id() {
  let node_value;
  if (hydrating && hydrate_node && node_type(hydrate_node) === COMMENT_NODE && (node_value = get_node_value(hydrate_node))?.startsWith("$")) {
    let id = node_value.substring(1);
    return hydrate_next(), id;
  }
  return (globalThis.__svelte ??= {}).uid ??= 1, `c${globalThis.__svelte.uid++}`;
}

// node_modules/svelte/src/internal/client/dom/blocks/boundary.js
var flags = 589824;
function boundary(node, props, children, transform_error) {
  new Boundary(node, props, children, transform_error);
}
var Boundary = class {
  /** @type {Boundary | null} */
  parent;
  is_pending = !1;
  /**
   * API-level transformError transform function. Transforms errors before they reach the `failed` snippet.
   * Inherited from parent boundary, or defaults to identity.
   * @type {(error: unknown) => unknown}
   */
  transform_error;
  /** @type {TemplateNode} */
  #anchor;
  /** @type {TemplateNode | null} */
  #hydrate_open = hydrating ? hydrate_node : null;
  /** @type {BoundaryProps} */
  #props;
  /** @type {((anchor: Node) => void)} */
  #children;
  /** @type {Effect} */
  #effect;
  /** @type {Effect | null} */
  #main_effect = null;
  /** @type {Effect | null} */
  #pending_effect = null;
  /** @type {Effect | null} */
  #failed_effect = null;
  /** @type {DocumentFragment | null} */
  #offscreen_fragment = null;
  #local_pending_count = 0;
  #pending_count_update_queued = !1;
  /**
   * User effects created inside this boundary. They will run when the boundary is revealed
   * @type {Effect[]}
   */
  #effects = [];
  /**
   * Async effects that are currently in flight, grouped by the batch that started them.
   * A speculative fork's work must not hold up the real world's boundary.
   * @type {Map<Batch, Map<Effect, number>>}
   */
  #pending = /* @__PURE__ */ new Map();
  /**
   * A source containing the number of pending async deriveds/expressions.
   * Only created if `$effect.pending()` is used inside the boundary,
   * otherwise updating the source results in needless `Batch.ensure()`
   * calls followed by no-op flushes
   * @type {Source<number> | null}
   */
  #effect_pending = null;
  #effect_pending_subscriber = createSubscriber(() => (this.#effect_pending = source(this.#local_pending_count), false_default && tag(this.#effect_pending, "$effect.pending()"), () => {
    this.#effect_pending = null;
  }));
  /**
   * @param {TemplateNode} node
   * @param {BoundaryProps} props
   * @param {((anchor: Node) => void)} children
   * @param {((error: unknown) => unknown) | undefined} [transform_error]
   */
  constructor(node, props, children, transform_error) {
    this.#anchor = node, this.#props = props, this.#children = (anchor) => {
      var effect2 = (
        /** @type {Effect} */
        active_effect
      );
      effect2.b = this, effect2.f |= 128, children(anchor);
    }, this.parent = /** @type {Effect} */
    active_effect.b, this.transform_error = transform_error ?? this.parent?.transform_error ?? ((e) => e), this.#effect = block(() => {
      if (hydrating) {
        let comment_data = read_hydration_instruction(
          /** @type {TemplateNode} */
          this.#hydrate_open
        );
        hydrate_next();
        let server_rendered_pending = comment_data === "[!";
        if (comment_data.startsWith("[?")) {
          let serialized_error = JSON.parse(comment_data.slice("[?".length));
          this.#hydrate_failed_content(serialized_error);
        } else server_rendered_pending ? this.#hydrate_pending_content() : this.#hydrate_resolved_content();
      } else
        this.#render();
    }, flags), hydrating && (this.#anchor = hydrate_node);
  }
  #hydrate_resolved_content() {
    try {
      this.#main_effect = branch(() => this.#children(this.#anchor));
    } catch (error) {
      this.error(error);
    }
  }
  /**
   * @param {unknown} error The deserialized error from the server's hydration comment
   */
  #hydrate_failed_content(error) {
    let failed = this.#props.failed, { reset: reset2, invoke_onerror } = this.#create_reset(error);
    queue_micro_task(invoke_onerror), failed && (this.#failed_effect = branch(() => {
      failed(
        this.#anchor,
        () => error,
        () => reset2
      );
    }));
  }
  /**
   * Creates the `reset` function for a failed boundary, along with a function
   * that invokes `onerror` with it (if provided)
   * @param {unknown} error
   * @returns {{ reset: () => void, invoke_onerror: () => void }}
   */
  #create_reset(error) {
    var did_reset = !1, calling_on_error = !1;
    let reset2 = () => {
      if (!this.#is_destroyed()) {
        if (did_reset) {
          svelte_boundary_reset_noop();
          return;
        }
        did_reset = !0, calling_on_error && svelte_boundary_reset_onerror(), this.#failed_effect !== null && pause_effect(this.#failed_effect, () => {
          this.#failed_effect = null;
        }), this.#run(() => {
          this.#render();
        });
      }
    };
    return { reset: reset2, invoke_onerror: () => {
      if (!this.#is_destroyed())
        try {
          calling_on_error = !0, this.#props.onerror?.(error, reset2), calling_on_error = !1;
        } catch (err) {
          invoke_error_boundary(err, this.#effect && this.#effect.parent);
        }
    } };
  }
  #is_destroyed() {
    return (this.#effect.f & 33570816) !== 0;
  }
  #hydrate_pending_content() {
    let pending3 = this.#props.pending;
    pending3 && (this.is_pending = !0, this.#pending_effect = branch(() => pending3(this.#anchor)), queue_micro_task(() => {
      if (!this.#is_destroyed()) {
        var previous_renderer = custom_renderers_flag ? push_renderer(this.#effect.r) : null;
        try {
          var fragment = this.#offscreen_fragment = create_fragment(), anchor = create_text();
          append_child(fragment, anchor), this.#main_effect = this.#run(() => {
            try {
              return branch(() => this.#children(anchor));
            } catch (error) {
              try {
                this.error(error);
              } catch (error2) {
                invoke_error_boundary(error2, this.#effect.parent);
              }
              return null;
            }
          }), this.#main_effect !== null && this.#pending.size === 0 && (this.is_pending = !1, insert_before(this.#anchor, fragment), this.#offscreen_fragment = null, pause_effect(
            /** @type {Effect} */
            this.#pending_effect,
            () => {
              this.#pending_effect = null;
            }
          ));
        } finally {
          custom_renderers_flag && pop_renderer(previous_renderer);
        }
      }
    }));
  }
  #render() {
    try {
      if (this.is_pending = this.has_pending_snippet(), this.#local_pending_count = 0, this.#main_effect = branch(() => {
        this.#children(this.#anchor);
      }), this.#pending.size === 0)
        this.is_pending = !1;
      else {
        let pending3 = this.#props.pending;
        if (pending3) {
          var fragment = this.#offscreen_fragment = create_fragment();
          move_effect(this.#main_effect, fragment), this.#pending_effect = branch(() => pending3(this.#anchor));
        }
      }
    } catch (error) {
      this.error(error);
    }
  }
  /**
   * Defer an effect inside a pending boundary until the boundary resolves
   * @param {Effect} effect
   */
  defer_effect(effect2) {
    this.#effects.push(effect2);
  }
  has_pending_snippet() {
    return !!this.#props.pending;
  }
  /**
   * @template T
   * @param {() => T} fn
   */
  #run(fn) {
    var previous_effect = active_effect, previous_reaction = active_reaction, previous_ctx = component_context;
    set_active_effect(this.#effect), set_active_reaction(this.#effect), set_component_context(this.#effect.ctx);
    var previous_renderer = custom_renderers_flag ? push_renderer(this.#effect.r) : null;
    try {
      return Batch.ensure(), fn();
    } finally {
      set_active_effect(previous_effect), set_active_reaction(previous_reaction), set_component_context(previous_ctx), custom_renderers_flag && pop_renderer(previous_renderer);
    }
  }
  /**
   * @param {Batch} batch
   */
  is_ready(batch) {
    if (batch.is_fork && this.#pending.size > 0)
      return !1;
    for (let [owner, pending3] of this.#pending)
      if (!owner.is_fork) {
        for (let effect2 of pending3.keys())
          if (batch.is_blocked_by(effect2, this.#main_effect))
            return !1;
      }
    return !0;
  }
  /**
   * @param {Effect[]} effects
   */
  reveal(effects) {
    this.is_pending = !1;
    for (let effect2 of this.#effects)
      set_signal_status(effect2, 2048), effects.push(effect2);
    if (this.#effects.length = 0, this.#pending_effect && pause_effect(this.#pending_effect, () => {
      this.#pending_effect = null;
    }), this.#offscreen_fragment) {
      var previous_renderer = custom_renderers_flag ? push_renderer(this.#effect.r) : null;
      try {
        insert_before(this.#anchor, this.#offscreen_fragment), this.#offscreen_fragment = null;
      } finally {
        custom_renderers_flag && pop_renderer(previous_renderer);
      }
    }
  }
  /**
   * @param {Effect} effect
   * @param {Batch} batch
   */
  increment(effect2, batch) {
    var pending3 = this.#pending.get(batch);
    pending3 === void 0 && this.#pending.set(batch, pending3 = /* @__PURE__ */ new Map());
    var count = pending3.get(effect2) ?? 0;
    pending3.set(effect2, count + 1);
  }
  /**
   * @param {Effect} effect
   * @param {Batch} batch
   */
  decrement(effect2, batch) {
    var pending3 = (
      /** @type {Map<Effect, number>} */
      this.#pending.get(batch)
    ), count = (
      /** @type {number} */
      pending3.get(effect2)
    );
    count === 1 ? (pending3.delete(effect2), pending3.size === 0 && this.#pending.delete(batch), queue_micro_task(() => {
      this.is_pending && !this.#is_destroyed() && Batch.ensure().schedule(this.#effect, 4096);
    })) : pending3.set(effect2, count - 1);
  }
  /**
   * Update the source that powers `$effect.pending()` inside this boundary.
   * Do not call from inside the class
   * @param {1 | -1} d
   */
  update_effect_pending(d) {
    this.#local_pending_count += d, !(!this.#effect_pending || this.#pending_count_update_queued) && (this.#pending_count_update_queued = !0, queue_micro_task(() => {
      this.#pending_count_update_queued = !1, this.#effect_pending && internal_set(this.#effect_pending, this.#local_pending_count);
    }));
  }
  get_effect_pending() {
    return this.#effect_pending_subscriber(), get(
      /** @type {Source<number>} */
      this.#effect_pending
    );
  }
  /** @param {unknown} error */
  error(error) {
    if (error === HYDRATION_ERROR || !this.#props.onerror && !this.#props.failed)
      throw error;
    current_batch?.is_fork ? (this.#main_effect && current_batch.skip_effect(this.#main_effect), this.#pending_effect && current_batch.skip_effect(this.#pending_effect), this.#failed_effect && current_batch.skip_effect(this.#failed_effect), current_batch.oncommit(() => {
      this.#is_destroyed() || this.#handle_error(error);
    })) : this.#handle_error(error);
  }
  /**
   * @param {unknown} error
   */
  #handle_error(error) {
    this.#main_effect && (destroy_effect(this.#main_effect), this.#main_effect = null), this.#pending_effect && (destroy_effect(this.#pending_effect), this.#pending_effect = null), this.#failed_effect && (destroy_effect(this.#failed_effect), this.#failed_effect = null), hydrating && (set_hydrate_node(
      /** @type {TemplateNode} */
      this.#hydrate_open
    ), next(), set_hydrate_node(skip_nodes()));
    let failed = this.#props.failed, handle_error_result = (transformed_error) => {
      if (this.#is_destroyed()) return;
      let { reset: reset2, invoke_onerror } = this.#create_reset(transformed_error);
      invoke_onerror(), failed && !this.#is_destroyed() && (this.#failed_effect = this.#run(() => {
        try {
          return branch(() => {
            var effect2 = (
              /** @type {Effect} */
              active_effect
            );
            effect2.b = this, effect2.f |= 128, failed(
              this.#anchor,
              () => transformed_error,
              () => reset2
            );
          });
        } catch (error2) {
          return invoke_error_boundary(
            error2,
            /** @type {Effect} */
            this.#effect.parent
          ), null;
        }
      }));
    };
    queue_micro_task(() => {
      if (!this.#is_destroyed()) {
        var result;
        try {
          result = this.transform_error(error);
        } catch (e) {
          invoke_error_boundary(e, this.#effect && this.#effect.parent);
          return;
        }
        result !== null && typeof result == "object" && typeof /** @type {any} */
        result.then == "function" ? result.then(
          handle_error_result,
          /** @param {unknown} e */
          (e) => invoke_error_boundary(e, this.#effect && this.#effect.parent)
        ) : handle_error_result(result);
      }
    });
  }
};
function pending() {
  active_effect === null && effect_pending_outside_reaction();
  var boundary2 = active_effect.b;
  return boundary2 === null ? 0 : boundary2.get_effect_pending();
}

// node_modules/svelte/src/internal/client/render.js
var should_intro = !0;
function set_should_intro(value) {
  should_intro = value;
}
function set_text(text2, value) {
  var str = value == null ? "" : typeof value == "object" ? `${value}` : value;
  str !== /** @type {any} */
  (text2[TEXT_CACHE] ??= get_node_value(text2)) && (text2[TEXT_CACHE] = str, set_node_value(text2, `${str}`));
}
function mount(component2, options) {
  return _mount(component2, options);
}
function hydrate(component2, options) {
  options.intro = options.intro ?? !1;
  let target = options.target, was_hydrating = hydrating, previous_hydrate_node = hydrate_node;
  try {
    for (var anchor = get_first_child(target); anchor && (node_type(anchor) !== COMMENT_NODE || get_node_value(anchor) !== "["); )
      anchor = get_next_sibling(anchor);
    if (!anchor)
      throw HYDRATION_ERROR;
    set_hydrating(!0), set_hydrate_node(
      /** @type {Comment} */
      anchor
    );
    let instance = _mount(component2, { ...options, anchor });
    return async_mode_flag && current_batch !== null && current_batch.commit_writes(), set_hydrating(!1), /**  @type {Exports} */
    instance;
  } catch (error) {
    if (error instanceof Error && error.message.split(`
`).some((line) => line.startsWith("https://svelte.dev/e/")))
      throw error;
    return error !== HYDRATION_ERROR && console.warn("Failed to hydrate: ", error), options.recover === !1 && hydration_failed(), clear_text_content(target), set_hydrating(!1), mount(component2, options);
  } finally {
    set_hydrating(was_hydrating), set_hydrate_node(previous_hydrate_node);
  }
}
function _mount(Component, options) {
  if (custom_renderers_flag && options.renderer) {
    var previous_renderer = push_renderer(options.renderer);
    try {
      return _mount_inner(Component, options);
    } finally {
      pop_renderer(previous_renderer);
    }
  }
  return _mount_inner(Component, options);
}
function _mount_inner(Component, { target, anchor, props = {}, events, context, intro = !0, transformError, renderer }) {
  var component2 = void 0, unmount2 = component_root(() => {
    var anchor_node = anchor ?? /** @type {Text} */
    append_child(
      /** @type {Node} */
      target,
      create_text()
    );
    boundary(
      /** @type {TemplateNode} */
      anchor_node,
      {
        pending: () => {
        }
      },
      (anchor_node2) => {
        push({});
        var ctx = (
          /** @type {ComponentContext} */
          component_context
        );
        if (context && (ctx.c = context), events && (props.$$events = events), hydrating && assign_nodes(
          /** @type {TemplateNode} */
          anchor_node2,
          null
        ), should_intro = intro, component2 = Component(anchor_node2, props) || mark_as_component(), should_intro = !0, hydrating && (active_effect.nodes.end = hydrate_node, hydrate_node === null || node_type(hydrate_node) !== COMMENT_NODE || get_node_value(hydrate_node) !== "]"))
          throw hydration_mismatch(), HYDRATION_ERROR;
        pop();
      },
      transformError
    );
    var stop_listening = renderer ? null : listen_to_delegated_events(
      /** @type {EventTarget} */
      target
    );
    return () => {
      if (stop_listening?.(), anchor_node !== anchor) {
        var parent = get_parent_node(
          /** @type {Node} */
          anchor_node
        );
        parent && remove_child(
          parent,
          /** @type {ChildNode} */
          anchor_node
        );
      }
    };
  });
  return mounted_components.set(component2, unmount2), component2;
}
var mounted_components = /* @__PURE__ */ new WeakMap();
function unmount(component2, options) {
  let fn = mounted_components.get(component2);
  return fn ? (mounted_components.delete(component2), fn(options)) : (false_default && lifecycle_double_unmount(), Promise.resolve());
}

// node_modules/svelte/src/internal/client/dev/hmr.js
function hmr(fn) {
  let current = source(fn);
  function wrapper(initial_anchor, props) {
    let component2 = {}, instance = {}, effect2, ran = !1, anchor = initial_anchor;
    return block(() => {
      if (component2 !== (component2 = get(current))) {
        if (effect2) {
          for (var k in instance) delete instance[k];
          destroy_effect(effect2);
        }
        effect2 = branch(() => {
          anchor = /** @type {any} */
          anchor[HMR_ANCHOR] ?? anchor, ran && set_should_intro(!1);
          var result = (
            // @ts-expect-error
            new.target ? new component2(anchor, props) : component2(anchor, props)
          );
          result && Object.defineProperties(instance, Object.getOwnPropertyDescriptors(result)), ran && set_should_intro(!0);
        });
        var inner_nodes = effect2.nodes;
        if (inner_nodes) {
          var ae = (
            /** @type {Effect} */
            active_effect
          );
          ae.nodes ? (ae.nodes.start = inner_nodes.start, ae.nodes.end = inner_nodes.end) : ae.nodes = { start: inner_nodes.start, end: inner_nodes.end, a: null, t: null };
        }
      }
    }, 65536), ran = !0, hydrating && (anchor = hydrate_node), instance;
  }
  return wrapper[FILENAME] = fn[FILENAME], wrapper[HMR] = {
    fn,
    current,
    update: (incoming) => {
      set(wrapper[HMR].current, incoming[HMR].fn), incoming[HMR].current = wrapper[HMR].current;
    }
  }, wrapper;
}

// node_modules/svelte/src/internal/client/dev/ownership.js
function create_ownership_validator(props) {
  let component2 = component_context?.function, parent = component_context?.p?.function;
  return {
    /**
     * @param {string} prop
     * @param {any[]} path
     * @param {any} result
     * @param {number} line
     * @param {number} column
     */
    mutation: (prop2, path2, result, line, column) => {
      let name = path2[0];
      if (is_bound_or_unset(props, name) || !parent)
        return result;
      let value = props;
      for (let i = 0; i < path2.length - 1; i++)
        if (value = value[path2[i]], !value?.[STATE_SYMBOL])
          return result;
      let location = sanitize_location(`${component2[FILENAME]}:${line}:${column}`);
      return ownership_invalid_mutation(name, location, prop2, parent[FILENAME]), result;
    },
    /**
     * @param {any} key
     * @param {any} child_component
     * @param {() => any} value
     */
    binding: (key2, child_component, value) => {
      !is_bound_or_unset(props, key2) && parent && value()?.[STATE_SYMBOL] && ownership_invalid_binding(
        component2[FILENAME],
        key2,
        child_component[FILENAME],
        parent[FILENAME]
      );
    }
  };
}
function is_bound_or_unset(props, prop_name) {
  let is_entry_props = STATE_SYMBOL in props || LEGACY_PROPS in props;
  return !!get_descriptor(props, prop_name)?.set || is_entry_props && prop_name in props || !(prop_name in props);
}

// node_modules/svelte/src/internal/client/dev/legacy.js
function check_target(target) {
  target && component_api_invalid_new(target[FILENAME] ?? "a component", target.name);
}
function legacy_api() {
  let component2 = component_context?.function;
  function error(method) {
    component_api_changed(method, component2[FILENAME]);
  }
  return {
    $destroy: () => error("$destroy()"),
    $on: () => error("$on(...)"),
    $set: () => error("$set(...)")
  };
}

// node_modules/svelte/src/internal/client/dev/inspect.js
function inspect(get_value, inspector, show_stack = !1) {
  validate_effect("$inspect");
  let initial = !0, error = (
    /** @type {any} */
    UNINITIALIZED
  );
  eager_effect(() => {
    error = UNINITIALIZED;
    try {
      var value = get_value();
    } catch (e) {
      error = e;
      return;
    }
    var snap = snapshot(value, !0, !0);
    untrack(() => {
      if (show_stack) {
        if (inspector(...snap), !initial) {
          let stack = get_error("$inspect(...)");
          stack && (console.groupCollapsed("stack trace"), console.log(stack), console.groupEnd());
        }
      } else
        inspector(initial ? "init" : "update", ...snap);
    }), initial = !1;
  }), render_effect(() => {
    try {
      get_value();
    } catch {
    }
    error !== UNINITIALIZED && (console.error(error), error = UNINITIALIZED);
  });
}

// node_modules/svelte/src/internal/client/dom/blocks/async.js
function async(node, blockers = [], expressions = [], fn) {
  var was_hydrating = hydrating, end = null;
  if (was_hydrating && (hydrate_next(), end = skip_nodes(!1), assign_nodes(node, end)), expressions.length === 0 && blockers.every((b) => b.settled)) {
    fn(node), was_hydrating && set_hydrate_node(end);
    return;
  }
  if (was_hydrating) {
    var previous_hydrate_node = hydrate_node;
    set_hydrate_node(end);
  }
  flatten(blockers, [], expressions, (values) => {
    was_hydrating && (set_hydrating(!0), set_hydrate_node(previous_hydrate_node));
    try {
      for (let d of values) get(d);
      fn(node, ...values);
    } finally {
      was_hydrating && set_hydrating(!1);
    }
  });
}

// node_modules/svelte/src/internal/client/dev/validation.js
function validate_snippet_args(anchor, ...args) {
  !current_renderer && (typeof anchor != "object" || !(anchor instanceof Node)) && invalid_snippet_arguments();
  for (let arg of args)
    typeof arg != "function" && invalid_snippet_arguments();
}

// node_modules/svelte/src/internal/client/dom/blocks/branches.js
var BranchManager = class {
  /** @type {TemplateNode} */
  anchor;
  /** @type {Map<Batch, Key>} */
  #batches = /* @__PURE__ */ new Map();
  /**
   * Map of keys to effects that are currently rendered in the DOM.
   * These effects are visible and actively part of the document tree.
   * Example:
   * ```
   * {#if condition}
   * 	foo
   * {:else}
   * 	bar
   * {/if}
   * ```
   * Can result in the entries `true->Effect` and `false->Effect`
   * @type {Map<Key, Effect>}
   */
  #onscreen = /* @__PURE__ */ new Map();
  /**
   * Similar to #onscreen with respect to the keys, but contains branches that are not yet
   * in the DOM, because their insertion is deferred.
   * @type {Map<Key, Branch>}
   */
  #offscreen = /* @__PURE__ */ new Map();
  /**
   * Keys of effects that are currently outroing
   * @type {Set<Key>}
   */
  #outroing = /* @__PURE__ */ new Set();
  /**
   * Whether to pause (i.e. outro) on change, or destroy immediately.
   * This is necessary for `<svelte:element>`
   */
  #transition = !0;
  /**
   * The renderer that was active when this BranchManager was created.
   * Needed so that #commit can push the correct renderer when doing DOM operations
   * outside of an effect context (e.g. as a batch commit callback).
   * @type {Renderer | null}
   */
  #renderer = null;
  /**
   * @param {TemplateNode} anchor
   * @param {boolean} transition
   */
  constructor(anchor, transition2 = !0) {
    this.anchor = anchor, this.#transition = transition2, this.#renderer = current_renderer;
  }
  /**
   * @param {Batch} batch
   */
  #commit = (batch) => {
    if (this.#batches.has(batch)) {
      var previous_renderer = custom_renderers_flag ? push_renderer(this.#renderer) : null;
      try {
        var key2 = (
          /** @type {Key} */
          this.#batches.get(batch)
        ), onscreen = this.#onscreen.get(key2);
        if (onscreen)
          resume_effect(onscreen), this.#outroing.delete(key2);
        else {
          var offscreen = this.#offscreen.get(key2);
          offscreen && (resume_effect(offscreen.effect), this.#onscreen.set(key2, offscreen.effect), this.#offscreen.delete(key2), false_default && (get_last_child(offscreen.fragment)[HMR_ANCHOR] = this.anchor), remove_node(
            /** @type {ChildNode} */
            get_last_child(offscreen.fragment)
          ), insert_before(this.anchor, offscreen.fragment), onscreen = offscreen.effect, onscreen.f &= -33554433);
        }
        this.#discard(batch);
        for (let [k, effect2] of this.#onscreen) {
          if (k === key2 || this.#outroing.has(k)) continue;
          let on_destroy = () => {
            if (Array.from(this.#batches.values()).includes(k)) {
              var fragment = create_fragment();
              move_effect(effect2, fragment), append_child(fragment, create_text()), this.#offscreen.set(k, { effect: effect2, fragment });
            } else
              destroy_effect(effect2);
            this.#outroing.delete(k), this.#onscreen.delete(k);
          };
          this.#transition || !onscreen ? (this.#outroing.add(k), pause_effect(effect2, on_destroy, !1)) : on_destroy();
        }
      } finally {
        custom_renderers_flag && pop_renderer(previous_renderer);
      }
    }
  };
  /**
   * @param {Batch} batch
   */
  #discard = (batch) => {
    for (let b of this.#batches.keys())
      (b === batch || !b.linked) && this.#batches.delete(b);
    let keys = Array.from(this.#batches.values());
    for (let [k, branch2] of this.#offscreen)
      keys.includes(k) || (destroy_effect(branch2.effect), this.#offscreen.delete(k));
  };
  /**
   *
   * @param {any} key
   * @param {null | ((target: TemplateNode) => void)} fn
   */
  ensure(key2, fn) {
    var batch = (
      /** @type {Batch} */
      current_batch
    ), defer = should_defer_append();
    if (fn && !this.#onscreen.has(key2) && !this.#offscreen.has(key2))
      if (defer) {
        var fragment = create_fragment(), target = create_text();
        append_child(fragment, target);
        var effect2 = branch(() => fn(target));
        effect2.f |= 33554432, batch.unskip_new_effect(effect2), this.#offscreen.set(key2, { effect: effect2, fragment });
      } else
        this.#onscreen.set(
          key2,
          branch(() => fn(this.anchor))
        );
    if (this.#batches.set(batch, key2), defer) {
      for (let [k, effect3] of this.#onscreen)
        k === key2 ? batch.unskip_effect(effect3) : batch.skip_effect(effect3);
      for (let [k, branch2] of this.#offscreen)
        k === key2 ? batch.unskip_effect(branch2.effect) : batch.skip_effect(branch2.effect);
      batch.oncommit(this.#commit), batch.ondiscard(this.#discard);
    } else
      hydrating && (this.anchor = hydrate_node), this.#commit(batch);
  }
};

// node_modules/svelte/src/internal/client/dom/blocks/await.js
var PENDING = 0, THEN = 1, CATCH = 2;
function await_block(node, get_input, pending_fn, then_fn, catch_fn) {
  hydrating && hydrate_next();
  var runes = is_runes(), v = (
    /** @type {V} */
    UNINITIALIZED
  ), value = runes ? source(v) : mutable_source(v, !1, !1), error = runes ? source(v) : mutable_source(v, !1, !1);
  value.f |= 4194304, error.f |= 4194304, false_default && (value.label = "{#await ...} value", error.label = "{#await ...} error");
  var branches = new BranchManager(node), current = {}, forks = /* @__PURE__ */ new Map(), commit = (batch) => {
    var run3 = forks.get(batch);
    run3 && (current = run3), forks.delete(batch);
  }, discard = (batch) => {
    forks.delete(batch);
  }, is_latest = (run3, batch) => {
    if (forks.has(batch)) return forks.get(batch) === run3;
    if (run3 !== current) return !1;
    for (let [b, r2] of forks)
      if (!b.is_fork && r2 !== null) return !1;
    return !0;
  }, owner = (batch) => {
    var target = merge_target(batch);
    return forks.has(batch) && target.linked ? target : null;
  };
  block(() => {
    var effect2 = (
      /** @type {Effect} */
      active_effect
    ), batch = (
      /** @type {Batch} */
      current_batch
    ), input = get_input(), run3 = {}, fork2 = batch.is_fork && should_defer_append();
    if ((fork2 || forks.has(batch)) && (forks.set(batch, run3), batch.oncommit(commit), batch.ondiscard(discard)), !fork2) {
      for (let b of forks.keys())
        !b.is_fork && b !== batch && forks.set(b, null);
      current = run3;
    }
    let mismatch = hydrating && is_promise(input) === (get_node_value(node) === "[!");
    if (mismatch && (set_hydrate_node(skip_nodes()), set_hydrating(!1)), is_promise(input)) {
      var restore = capture(), resolved = !1;
      let resolve = (fn) => {
        if (!((effect2.f & 16384) !== 0 || !is_latest(run3, batch))) {
          resolved = !0;
          var target = owner(batch);
          if (target !== null) {
            var previous = current_batch;
            restore(target);
            try {
              fn((source2, v2) => (
                /** @type {Batch} */
                target.capture(source2, v2, source2.equals(v2))
              ));
            } finally {
              unset_context(!1), target.flush(), previous?.linked && previous.activate();
            }
            return;
          }
          restore(null), current_batch === batch && batch.deactivate(), Batch.ensure();
          try {
            fn(internal_set);
          } finally {
            unset_context(!1), is_flushing_sync || flushSync();
          }
        }
      };
      input.then(
        (v2) => {
          resolve((set2) => {
            set2(value, v2), branches.ensure(THEN, then_fn && ((target) => then_fn(target, value)));
          });
        },
        (e) => {
          resolve((set2) => {
            if (set2(error, e), branches.ensure(CATCH, catch_fn && ((target) => catch_fn(target, error))), !catch_fn)
              throw e;
          });
        }
      ), hydrating ? branches.ensure(PENDING, pending_fn) : queue_micro_task(() => {
        resolved || resolve(() => {
          branches.ensure(PENDING, pending_fn);
        });
      });
    } else
      owner(batch) === batch ? batch.capture(value, input, value.equals(input)) : internal_set(value, input), branches.ensure(THEN, then_fn && ((target) => then_fn(target, value)));
    mismatch && set_hydrating(!0);
  }, 524288);
}

// node_modules/svelte/src/internal/client/dom/blocks/if.js
function if_block(node, fn, elseif = !1) {
  var marker;
  hydrating && (marker = hydrate_node, hydrate_next());
  var branches = new BranchManager(node), flags2 = elseif ? 65536 : 0;
  function update_branch(key2, fn2) {
    if (hydrating) {
      var data = read_hydration_instruction(
        /** @type {TemplateNode} */
        marker
      );
      if (key2 !== parseInt(data.substring(1))) {
        var anchor = skip_nodes();
        set_hydrate_node(anchor), branches.anchor = anchor, set_hydrating(!1), branches.ensure(key2, fn2), set_hydrating(!0);
        return;
      }
    }
    branches.ensure(key2, fn2);
  }
  block(() => {
    var has_branch = !1;
    fn((fn2, key2 = 0) => {
      has_branch = !0, update_branch(key2, fn2);
    }), has_branch || update_branch(-1, null);
  }, flags2);
}

// node_modules/svelte/src/internal/client/dom/blocks/key.js
var NAN = /* @__PURE__ */ Symbol("NaN");
function key(node, get_key, render_fn) {
  hydrating && hydrate_next();
  var branches = new BranchManager(node), legacy = !is_runes();
  block(() => {
    var key2 = get_key();
    key2 !== key2 && (key2 = /** @type {any} */
    NAN), legacy && key2 !== null && typeof key2 == "object" && (key2 = /** @type {V} */
    {}), branches.ensure(key2, render_fn);
  });
}

// node_modules/svelte/src/escaping.js
var ATTR_REGEX = /[&"<]/g, CONTENT_REGEX = /[&<]/g;
function escape_html(value, is_attr) {
  let str = String(value ?? ""), pattern = is_attr ? ATTR_REGEX : CONTENT_REGEX;
  pattern.lastIndex = 0;
  let escaped = "", last = 0;
  for (; pattern.test(str); ) {
    let i = pattern.lastIndex - 1, ch = str[i];
    escaped += str.substring(last, i) + (ch === "&" ? "&amp;" : ch === '"' ? "&quot;" : "&lt;"), last = i + 1;
  }
  return escaped + str.substring(last);
}

// node_modules/clsx/dist/clsx.mjs
function r(e) {
  var t, f, n = "";
  if (typeof e == "string" || typeof e == "number") n += e;
  else if (typeof e == "object") if (Array.isArray(e)) {
    var o = e.length;
    for (t = 0; t < o; t++) e[t] && (f = r(e[t])) && (n && (n += " "), n += f);
  } else for (f in e) e[f] && (n && (n += " "), n += f);
  return n;
}
function clsx() {
  for (var e, t, f = 0, n = "", o = arguments.length; f < o; f++) (e = arguments[f]) && (t = r(e)) && (n && (n += " "), n += t);
  return n;
}

// node_modules/svelte/src/internal/shared/attributes.js
var replacements = {
  translate: /* @__PURE__ */ new Map([
    [!0, "yes"],
    [!1, "no"]
  ])
};
function attr(name, value, is_boolean = !1) {
  if (name === "hidden" && value !== "until-found" && (is_boolean = !0), value == null || is_boolean && !value && value !== "") return "";
  let normalized = has_own_property.call(replacements, name) && replacements[name].get(value) || value, assignment = is_boolean ? '=""' : `="${escape_html(normalized, !0)}"`;
  return ` ${name}${assignment}`;
}
function clsx2(value) {
  return typeof value == "object" ? clsx(value) : value ?? "";
}
var whitespace = [...` 	
\r\f \v\uFEFF`];
function to_class(value, hash2, directives) {
  var classname = value == null ? "" : "" + value;
  if (hash2 && (classname = classname ? classname + " " + hash2 : hash2), directives) {
    for (var key2 of Object.keys(directives))
      if (directives[key2])
        classname = classname ? classname + " " + key2 : key2;
      else if (classname.length)
        for (var len = key2.length, a = 0; (a = classname.indexOf(key2, a)) >= 0; ) {
          var b = a + len;
          (a === 0 || whitespace.includes(classname[a - 1])) && (b === classname.length || whitespace.includes(classname[b])) ? classname = (a === 0 ? "" : classname.substring(0, a)) + classname.substring(b + 1) : a = b;
        }
  }
  return classname === "" ? null : classname;
}
function append_styles(styles, important = !1) {
  var separator = important ? " !important;" : ";", css = "";
  for (var key2 of Object.keys(styles)) {
    var value = styles[key2];
    value != null && value !== "" && (css += " " + key2 + ": " + value + separator);
  }
  return css;
}
function to_css_name(name) {
  return name[0] !== "-" || name[1] !== "-" ? name.toLowerCase() : name;
}
function to_style(value, styles) {
  if (styles) {
    var new_style = "", normal_styles, important_styles;
    if (Array.isArray(styles) ? (normal_styles = styles[0], important_styles = styles[1]) : normal_styles = styles, value) {
      value = String(value).replaceAll(/\/\*.*?\*\//g, "").trim();
      var in_str = !1, in_apo = 0, in_comment = !1, reserved_names = [];
      normal_styles && reserved_names.push(...Object.keys(normal_styles).map(to_css_name)), important_styles && reserved_names.push(...Object.keys(important_styles).map(to_css_name));
      var start_index = 0, name_index = -1;
      let len = value.length;
      for (var i = 0; i < len; i++) {
        var c = value[i];
        if (in_comment ? c === "/" && value[i - 1] === "*" && (in_comment = !1) : in_str ? in_str === c && (in_str = !1) : c === "/" && value[i + 1] === "*" ? in_comment = !0 : c === '"' || c === "'" ? in_str = c : c === "(" ? in_apo++ : c === ")" && in_apo--, !in_comment && in_str === !1 && in_apo === 0) {
          if (c === ":" && name_index === -1)
            name_index = i;
          else if (c === ";" || i === len - 1) {
            if (name_index !== -1) {
              var name = to_css_name(value.substring(start_index, name_index).trim());
              if (!reserved_names.includes(name)) {
                c !== ";" && i++;
                var property = value.substring(start_index, i).trim();
                new_style += " " + property + ";";
              }
            }
            start_index = i + 1, name_index = -1;
          }
        }
      }
    }
    return normal_styles && (new_style += append_styles(normal_styles)), important_styles && (new_style += append_styles(important_styles, !0)), new_style = new_style.trim(), new_style === "" ? null : new_style;
  }
  return value == null ? null : String(value);
}

// node_modules/svelte/src/internal/client/dom/blocks/css-props.js
function css_props(element2, get_styles) {
  if (hydrating && set_hydrate_node(get_first_child(element2)), custom_renderers_flag && current_renderer !== null) {
    var base = get_attribute(element2, "style");
    render_effect(() => {
      var style = to_style(base, get_styles());
      style === null ? remove_attribute(element2, "style") : set_css_text(
        /** @type {HTMLElement} */
        element2,
        style
      );
    });
    return;
  }
  render_effect(() => {
    var styles = get_styles();
    for (var key2 in styles) {
      var value = styles[key2];
      value == null || value === "" ? style_remove_property(
        /** @type {HTMLElement} */
        element2,
        key2
      ) : style_set_property(
        /** @type {HTMLElement} */
        element2,
        key2,
        value
      );
    }
  });
}

// node_modules/svelte/src/internal/client/dom/blocks/each.js
function index(_, i) {
  return i;
}
function pause_effects(state2, to_destroy, controlled_anchor) {
  for (var transitions = [], length = to_destroy.length, group, remaining = to_destroy.length, i = 0; i < length; i++) {
    let effect2 = to_destroy[i];
    pause_effect(
      effect2,
      () => {
        if (group) {
          if (group.pending.delete(effect2), group.done.add(effect2), group.pending.size === 0) {
            var groups = (
              /** @type {Set<EachOutroGroup>} */
              state2.outrogroups
            );
            destroy_effects(state2, array_from(group.done)), groups.delete(group), groups.size === 0 && (state2.outrogroups = null);
          }
        } else
          remaining -= 1;
      },
      !1
    );
  }
  if (remaining === 0) {
    var fast_path = transitions.length === 0 && controlled_anchor !== null && state2.pending.size === 0;
    if (fast_path) {
      var anchor = (
        /** @type {Element} */
        controlled_anchor
      ), parent_node = (
        /** @type {Element} */
        get_parent_node(anchor)
      );
      clear_text_content(parent_node), append_child(parent_node, anchor), state2.items.clear();
    }
    destroy_effects(state2, to_destroy, !fast_path);
  } else
    group = {
      pending: new Set(to_destroy),
      done: /* @__PURE__ */ new Set()
    }, (state2.outrogroups ??= /* @__PURE__ */ new Set()).add(group);
}
function selected_effects(state2) {
  var selected = /* @__PURE__ */ new Set();
  for (let keys of state2.pending.values())
    for (let key2 of keys)
      selected.add(
        /** @type {EachItem} */
        state2.items.get(key2).e
      );
  return selected;
}
function destroy_effects(state2, to_destroy, remove_dom = !0) {
  for (var preserved_effects = state2.pending.size > 0 ? selected_effects(state2) : void 0, i = 0; i < to_destroy.length; i++) {
    var e = to_destroy[i];
    if (preserved_effects?.has(e)) {
      e.f |= 33554432;
      let fragment = create_fragment();
      move_effect(e, fragment), e !== state2.effect.last && (link(state2, e.prev, e.next), link(state2, state2.effect.last, e), link(state2, e, null));
    } else
      destroy_effect(to_destroy[i], remove_dom);
  }
}
function is_outroing(state2, effect2) {
  if (state2.outrogroups !== null) {
    for (let group of state2.outrogroups)
      if (group.pending.has(effect2) || group.done.has(effect2)) return !0;
  }
  return !1;
}
var offscreen_anchors = /* @__PURE__ */ new WeakMap(), dom_weak_key = {};
function get_offscreen_anchor() {
  var cached = offscreen_anchors.get(current_renderer ?? dom_weak_key);
  if (cached) return cached;
  var offscreen_anchor = create_text();
  if (current_renderer) {
    var fragment = create_fragment();
    append_child(fragment, offscreen_anchor);
  }
  return offscreen_anchors.set(current_renderer ?? dom_weak_key, offscreen_anchor), offscreen_anchor;
}
function each(node, flags2, get_collection, get_key, render_fn, fallback_fn = null) {
  var anchor = node, items = /* @__PURE__ */ new Map(), is_controlled = (flags2 & 4) !== 0, renderer = current_renderer;
  if (is_controlled) {
    var parent_node = (
      /** @type {Element} */
      node
    );
    anchor = hydrating ? set_hydrate_node(get_first_child(parent_node)) : (
      /** @type {Text} */
      append_child(parent_node, create_text())
    );
  }
  hydrating && hydrate_next();
  var fallback2 = null, each_array = derived_safe_equal(() => {
    var collection = get_collection();
    return (
      /** @type {V[]} */
      is_array(collection) ? collection : collection == null ? [] : array_from(collection)
    );
  });
  false_default && tag(each_array, "{#each ...}");
  var pending3 = /* @__PURE__ */ new Map(), first_run = !0;
  function commit(batch) {
    if ((state2.effect.f & 16384) === 0) {
      var previous_renderer = custom_renderers_flag ? push_renderer(renderer) : null;
      try {
        state2.pending.delete(batch);
        var array = get(each_array);
        state2.fallback = fallback2, reconcile(state2, array, anchor, flags2, get_key), fallback2 !== null && (array.length === 0 ? (fallback2.f & 33554432) === 0 ? resume_effect(fallback2) : (fallback2.f ^= 33554432, move(fallback2, null, anchor)) : pause_effect(fallback2, () => {
          fallback2 = null;
        }));
      } finally {
        custom_renderers_flag && pop_renderer(previous_renderer);
      }
      destroy_unselected();
    }
  }
  function discard(batch) {
    state2.pending.delete(batch), (state2.effect.f & 16384) === 0 && destroy_unselected();
  }
  function destroy_unselected() {
    for (var selected, effect3 = state2.effect.last; effect3 !== null; ) {
      var prev = effect3.prev;
      if ((effect3.f & 32) !== 0 && effect3 !== fallback2) {
        if ((effect3.f & 33554432) === 0) break;
        selected ??= selected_effects(state2), !selected.has(effect3) && !is_outroing(state2, effect3) && destroy_effect(effect3);
      }
      effect3 = prev;
    }
  }
  var effect2 = block(() => {
    var array = (
      /** @type {V[]} */
      get(each_array)
    ), length = array.length;
    let mismatch = !1;
    if (hydrating) {
      var is_else = read_hydration_instruction(anchor) === "[!";
      is_else !== (length === 0) && (anchor = skip_nodes(), set_hydrate_node(anchor), set_hydrating(!1), mismatch = !0);
    }
    for (var keys = /* @__PURE__ */ new Set(), batch = (
      /** @type {Batch} */
      current_batch
    ), defer = should_defer_append(), index2 = 0; index2 < length; index2 += 1) {
      hydrating && node_type(hydrate_node) === COMMENT_NODE && get_node_value(hydrate_node) === "]" && (anchor = /** @type {Comment} */
      hydrate_node, mismatch = !0, set_hydrating(!1));
      var value = array[index2], key2 = get_key(value, index2);
      if (false_default) {
        var key_again = get_key(value, index2);
        key2 !== key_again && each_key_volatile(String(index2), String(key2), String(key_again));
      }
      var item = first_run ? null : items.get(key2);
      item ? (item.v && internal_set(item.v, value), item.i && internal_set(item.i, index2), defer && batch.unskip_effect(item.e)) : (item = create_item(
        items,
        first_run ? anchor : get_offscreen_anchor(),
        value,
        key2,
        index2,
        render_fn,
        flags2,
        get_collection
      ), first_run || (item.e.f |= 33554432, defer && batch.unskip_new_effect(item.e)), items.set(key2, item)), keys.add(key2);
    }
    if (length === 0 && fallback_fn && !fallback2 && (first_run ? fallback2 = branch(() => fallback_fn(anchor)) : (fallback2 = branch(() => fallback_fn(get_offscreen_anchor())), fallback2.f |= 33554432, defer && batch.unskip_new_effect(fallback2))), defer && fallback2 !== null && (length === 0 ? batch.unskip_effect(fallback2) : batch.skip_effect(fallback2)), length > keys.size && (false_default ? validate_each_keys(array, get_key) : each_key_duplicate("", "", "")), hydrating && length > 0 && set_hydrate_node(skip_nodes()), !first_run)
      if (pending3.set(batch, keys), defer) {
        for (let [key3, item2] of items)
          keys.has(key3) || batch.skip_effect(item2.e);
        batch.oncommit(commit), batch.ondiscard(discard);
      } else
        commit(batch);
    mismatch && set_hydrating(!0), get(each_array);
  }), state2 = { effect: effect2, flags: flags2, items, pending: pending3, outrogroups: null, fallback: fallback2 };
  first_run = !1, hydrating && (anchor = hydrate_node);
}
function skip_to_branch(effect2) {
  for (; effect2 !== null && (effect2.f & 32) === 0; )
    effect2 = effect2.next;
  return effect2;
}
function reconcile(state2, array, anchor, flags2, get_key) {
  var is_animated = (flags2 & 8) !== 0, length = array.length, items = state2.items, current = skip_to_branch(state2.effect.first), seen, prev = null, to_animate, matched = [], stashed = [], value, key2, effect2, i;
  if (is_animated)
    for (i = 0; i < length; i += 1)
      value = array[i], key2 = get_key(value, i), effect2 = /** @type {EachItem} */
      items.get(key2).e, (effect2.f & 33554432) === 0 && (effect2.nodes?.a?.measure(), (to_animate ??= /* @__PURE__ */ new Set()).add(effect2));
  for (i = 0; i < length; i += 1) {
    if (value = array[i], key2 = get_key(value, i), effect2 = /** @type {EachItem} */
    items.get(key2).e, state2.outrogroups !== null)
      for (let group of state2.outrogroups)
        group.pending.delete(effect2), group.done.delete(effect2);
    if ((effect2.f & 8192) !== 0 && (resume_effect(effect2), is_animated && (effect2.nodes?.a?.unfix(), (to_animate ??= /* @__PURE__ */ new Set()).delete(effect2))), (effect2.f & 33554432) !== 0)
      if (effect2.f ^= 33554432, effect2 === current)
        move(effect2, null, anchor);
      else {
        var next2 = prev ? prev.next : current;
        effect2 === state2.effect.last && (state2.effect.last = effect2.prev), effect2.prev && (effect2.prev.next = effect2.next), effect2.next && (effect2.next.prev = effect2.prev), link(state2, prev, effect2), link(state2, effect2, next2), move(effect2, next2, anchor), prev = effect2, matched = [], stashed = [], current = skip_to_branch(prev.next);
        continue;
      }
    if (effect2 !== current) {
      if (seen !== void 0 && seen.has(effect2)) {
        if (matched.length < stashed.length) {
          var start = stashed[0], j;
          prev = start.prev;
          var a = matched[0], b = matched[matched.length - 1];
          for (j = 0; j < matched.length; j += 1)
            move(matched[j], start, anchor);
          for (j = 0; j < stashed.length; j += 1)
            seen.delete(stashed[j]);
          link(state2, a.prev, b.next), link(state2, prev, a), link(state2, b, start), current = start, prev = b, i -= 1, matched = [], stashed = [];
        } else
          seen.delete(effect2), move(effect2, current, anchor), link(state2, effect2.prev, effect2.next), link(state2, effect2, prev === null ? state2.effect.first : prev.next), link(state2, prev, effect2), prev = effect2;
        continue;
      }
      for (matched = [], stashed = []; current !== null && current !== effect2; )
        (seen ??= /* @__PURE__ */ new Set()).add(current), stashed.push(current), current = skip_to_branch(current.next);
      if (current === null)
        continue;
    }
    (effect2.f & 33554432) === 0 && matched.push(effect2), prev = effect2, current = skip_to_branch(effect2.next);
  }
  if (state2.outrogroups !== null) {
    for (let group of state2.outrogroups)
      group.pending.size === 0 && (destroy_effects(state2, array_from(group.done)), state2.outrogroups?.delete(group));
    state2.outrogroups.size === 0 && (state2.outrogroups = null);
  }
  if (current !== null || seen !== void 0) {
    var to_destroy = [];
    if (seen !== void 0)
      for (effect2 of seen)
        (effect2.f & 8192) === 0 && to_destroy.push(effect2);
    for (; current !== null; )
      (current.f & 8192) === 0 && current !== state2.fallback && to_destroy.push(current), current = skip_to_branch(current.next);
    var destroy_length = to_destroy.length;
    if (destroy_length > 0) {
      var controlled_anchor = (flags2 & 4) !== 0 && length === 0 ? anchor : null;
      if (is_animated) {
        for (i = 0; i < destroy_length; i += 1)
          to_destroy[i].nodes?.a?.measure();
        for (i = 0; i < destroy_length; i += 1)
          to_destroy[i].nodes?.a?.fix();
      }
      pause_effects(state2, to_destroy, controlled_anchor);
    }
  }
  is_animated && queue_micro_task(() => {
    if (to_animate !== void 0)
      for (effect2 of to_animate)
        effect2.nodes?.a?.apply();
  });
}
function create_item(items, anchor, value, key2, index2, render_fn, flags2, get_collection) {
  var v = (flags2 & 1) !== 0 ? (flags2 & 16) === 0 ? mutable_source(value, !1, !1) : source(value) : null, i = (flags2 & 2) !== 0 ? source(index2) : null;
  return false_default && v && (v.trace = () => {
    get_collection()[i?.v ?? index2];
  }), {
    v,
    i,
    e: branch(() => (render_fn(anchor, v ?? value, i ?? index2, get_collection), () => {
      items.delete(key2);
    }))
  };
}
function move(effect2, next2, anchor) {
  if (effect2.nodes)
    for (var node = effect2.nodes.start, end = effect2.nodes.end, dest = next2 && (next2.f & 33554432) === 0 ? (
      /** @type {EffectNodes} */
      next2.nodes.start
    ) : anchor; node !== null; ) {
      var next_node = (
        /** @type {TemplateNode} */
        get_next_sibling(node)
      );
      if (insert_before(dest, node), node === end)
        return;
      node = next_node;
    }
}
function link(state2, prev, next2) {
  prev === null ? state2.effect.first = next2 : prev.next = next2, next2 === null ? state2.effect.last = prev : next2.prev = prev;
}
function validate_each_keys(array, key_fn) {
  let keys = /* @__PURE__ */ new Map(), length = array.length;
  for (let i = 0; i < length; i++) {
    let key2 = key_fn(array[i], i);
    if (keys.has(key2)) {
      let a = String(keys.get(key2)), b = String(i), k = String(key2);
      k.startsWith("[object ") && (k = null), each_key_duplicate(a, b, k);
    }
    keys.set(key2, i);
  }
}

// node_modules/svelte/src/internal/client/dom/blocks/html.js
function check_hash(server_hash, value, loc) {
  if (!server_hash || server_hash === hash(String(value ?? ""))) return;
  let location;
  loc ? location = `near ${loc[0]}:${loc[1]}:${loc[2]}` : dev_current_component_function?.[FILENAME] && (location = `in ${dev_current_component_function[FILENAME]}`), hydration_html_changed(sanitize_location(location));
}
function assign_locations2(effect2, parent, loc) {
  if (effect2.nodes !== null)
    for (var node = effect2.nodes.start; node !== null && (node_type(node) === ELEMENT_NODE && (node.__svelte_meta = {
      parent,
      loc: { file: loc[0], line: loc[1], column: loc[2] },
      component: component_context
    }), node !== effect2.nodes.end); )
      node = get_next_sibling(node);
}
function html(node, get_value, is_controlled = !1, svg = !1, mathml = !1, skip_warning = !1, location) {
  var anchor = node, parent = false_default ? dev_stack : null, value = "";
  if (is_controlled) {
    var parent_node = (
      /** @type {Element} */
      node
    );
    hydrating && (anchor = set_hydrate_node(get_first_child(parent_node)));
  }
  template_effect(() => {
    var effect2 = (
      /** @type {Effect} */
      active_effect
    );
    if (value === (value = get_value() ?? "")) {
      hydrating && hydrate_next();
      return;
    }
    if (is_controlled && !hydrating) {
      effect2.nodes = null, set_inner_html(
        parent_node,
        /** @type {string} */
        value
      ), value !== "" && (assign_nodes(
        /** @type {TemplateNode} */
        get_first_child(parent_node),
        /** @type {TemplateNode} */
        get_last_child(parent_node)
      ), false_default && location && assign_locations2(effect2, parent, location));
      return;
    }
    if (effect2.nodes !== null && (remove_effect_dom(
      effect2.nodes.start,
      /** @type {TemplateNode} */
      effect2.nodes.end
    ), effect2.nodes = null), value !== "") {
      if (hydrating) {
        for (var hash2 = get_node_value(hydrate_node), next2 = hydrate_next(), last = next2; next2 !== null && (node_type(next2) !== COMMENT_NODE || get_node_value(next2) !== ""); )
          last = next2, next2 = get_next_sibling(next2);
        if (next2 === null)
          throw hydration_mismatch(), HYDRATION_ERROR;
        false_default && !skip_warning && check_hash(hash2, value, location), assign_nodes(hydrate_node, last), false_default && location && assign_locations2(effect2, parent, location), anchor = set_hydrate_node(next2);
        return;
      }
      var ns = svg ? NAMESPACE_SVG : mathml ? NAMESPACE_MATHML : void 0, wrapper = (
        /** @type {HTMLTemplateElement | SVGElement | MathMLElement} */
        create_element(svg ? "svg" : mathml ? "math" : "template", ns)
      );
      set_inner_html(
        wrapper,
        /** @type {any} */
        value
      );
      var node2 = svg || mathml ? wrapper : (
        /** @type {HTMLTemplateElement} */
        wrapper.content
      );
      if (assign_nodes(
        /** @type {TemplateNode} */
        get_first_child(node2),
        /** @type {TemplateNode} */
        get_last_child(node2)
      ), false_default && location && assign_locations2(effect2, parent, location), svg || mathml)
        for (; get_first_child(node2); )
          insert_before(
            anchor,
            /** @type {TemplateNode} */
            get_first_child(node2)
          );
      else
        insert_before(anchor, node2);
    }
  });
}

// node_modules/svelte/src/internal/client/dom/blocks/slot.js
function slot(anchor, $$props, name, slot_props, fallback_fn, location) {
  if (hydrating && hydrate_next(), $$props.$$host?.$$shadowRoot) {
    let element2 = create_element("slot");
    if (name !== "default" && set_attribute(element2, "name", name), false_default && location && (element2.__svelte_meta = {
      parent: dev_stack,
      loc: {
        file: location[0],
        line: location[1],
        column: location[2]
      },
      component: component_context
    }), append(anchor, element2), fallback_fn !== null) {
      let fallback_anchor = create_text();
      append_child(element2, fallback_anchor), fallback_fn(fallback_anchor);
    }
    return;
  }
  var slot_fn = $$props.$$slots?.[name], is_interop = !1;
  slot_fn === !0 && (slot_fn = $$props[name === "default" ? "children" : name], is_interop = !0), slot_fn === void 0 ? fallback_fn !== null && fallback_fn(anchor) : slot_fn(anchor, is_interop ? () => slot_props : slot_props);
}
function sanitize_slots(props) {
  let sanitized = {};
  props.children && (sanitized.default = !0);
  for (let key2 in props.$$slots)
    sanitized[key2] = !0;
  return sanitized;
}

// node_modules/svelte/src/internal/shared/validate.js
function validate_void_dynamic_element(tag_fn) {
  let tag2 = tag_fn();
  tag2 && is_void(tag2) && dynamic_void_element_content(tag2);
}
function validate_dynamic_element_tag(tag_fn) {
  let tag2 = tag_fn();
  tag2 && !(typeof tag2 == "string") && svelte_element_invalid_this_value();
}
function validate_store(store, name) {
  store != null && typeof store.subscribe != "function" && store_invalid_shape(name);
}
function prevent_snippet_stringification(fn) {
  return fn.toString = () => (snippet_without_render_tag(), ""), fn;
}

// node_modules/svelte/src/internal/client/custom-renderer/containers.js
var containers = /* @__PURE__ */ new Map(), shared_containers = 0, queued = !1;
function add_crossing(container, crossing) {
  var crossings = containers.get(container);
  if (crossings === void 0) {
    containers.set(container, [crossing]);
    return;
  }
  crossings.push(crossing), crossings.length === 2 && shared_containers++ === 0 && set_before_effects(sync_shared_containers), queued || (queued = !0, queue_micro_task(() => {
    queued = !1, sync_shared_containers();
  }));
}
function remove_crossing(container, crossing) {
  var crossings = (
    /** @type {Crossing[]} */
    containers.get(container)
  );
  crossings.splice(crossings.indexOf(crossing), 1), crossings.length === 1 && --shared_containers === 0 && set_before_effects(null), crossings.length === 0 && containers.delete(container);
}
function sync_shared_containers() {
  do {
    var moved = !1;
    for (var crossings of containers.values())
      crossings.length > 1 && sync(crossings) && (moved = !0);
  } while (moved && shared_containers > 1);
}
function sync(crossings) {
  var host = crossings[0].host;
  for (var crossing of crossings)
    if (crossing.host !== host) return !1;
  var previous_renderer = push_renderer(host);
  try {
    var paths = /* @__PURE__ */ new Map(), trees = /* @__PURE__ */ new Map();
    for (crossing of crossings) {
      var nodes = path(crossing.host_node), tree = trees.get(nodes[0]);
      tree === void 0 && trees.set(nodes[0], tree = []), tree.push(crossing), paths.set(crossing, nodes);
    }
    for (tree of trees.values())
      tree.sort(
        (a, b) => compare2(
          /** @type {TemplateNode[]} */
          paths.get(a),
          /** @type {TemplateNode[]} */
          paths.get(b)
        )
      );
    var order = crossings.map((crossing2) => {
      var root = (
        /** @type {TemplateNode[]} */
        paths.get(crossing2)[0]
      );
      return (
        /** @type {Crossing} */
        /** @type {Crossing[]} */
        trees.get(root).shift()
      );
    });
    if (order.every((crossing2, index2) => crossing2 === crossings[index2])) return !1;
  } finally {
    pop_renderer(previous_renderer);
  }
  previous_renderer = push_renderer(order[0].guest);
  try {
    for (var k = order.length - 2; k >= 0; k--) {
      var next2 = order[k + 1].start;
      if (get_next_sibling(order[k].end) !== next2)
        for (var node = (
          /** @type {TemplateNode | null} */
          order[k].start
        ); node !== null; ) {
          var sibling2 = node === order[k].end ? null : get_next_sibling(node);
          insert_before(next2, node), node = sibling2;
        }
    }
  } finally {
    pop_renderer(previous_renderer);
  }
  return crossings.splice(0, crossings.length, ...order), !0;
}
function path(node) {
  for (var nodes = [], n = (
    /** @type {TemplateNode | null} */
    node
  ); n !== null && (nodes.push(n), node_type(n) !== DOCUMENT_FRAGMENT_NODE); n = get_parent_node(n))
    ;
  return nodes.reverse();
}
function compare2(a, b) {
  for (var i = 0; i < a.length && i < b.length && a[i] === b[i]; ) i++;
  if (i === a.length || i === b.length) return 0;
  for (var node = get_next_sibling(a[i]); node !== null; node = get_next_sibling(node))
    if (node === b[i]) return -1;
  return 1;
}

// node_modules/svelte/src/internal/client/custom-renderer/cross.js
function is_crossing(renderer) {
  return current_renderer !== renderer;
}
function cross(renderer, anchor, render) {
  var host = current_renderer, parent = get_parent_node(anchor), connection;
  renderer?.embed ? connection = renderer.embed({ host, parent, anchor }) : host?.host ? connection = host.host({ guest: renderer, parent, anchor }) : renderer_cannot_cross(
    renderer === null ? "the DOM" : "a custom renderer",
    host === null ? "the DOM" : render !== null ? "a different custom renderer" : "a custom renderer"
  );
  var host_node = (
    /** @type {TemplateNode} */
    connection.node ?? create_text()
  );
  insert_before(anchor, host_node), assign_nodes(host_node, host_node);
  var start = null, end = null, stop_listening = null, result, previous_renderer = push_renderer(renderer);
  try {
    var first = create_text(), last = create_text(), container = (
      /** @type {Node} */
      connection.container
    );
    append_child(container, first), append_child(container, last), start = first, end = last;
    var crossing = { host, guest: renderer, host_node, start, end };
    add_crossing(container, crossing), renderer === null && (stop_listening = listen_to_delegated_events(container)), branch(() => {
      result = render(last);
    });
  } finally {
    pop_renderer(previous_renderer), teardown(() => {
      crossing !== void 0 && remove_crossing(container, crossing);
      var previous_renderer2 = push_renderer(renderer);
      try {
        for (var node = start; node !== null; ) {
          var next2 = node === end ? null : get_next_sibling(node);
          remove_node(
            /** @type {ChildNode} */
            node
          ), node = next2;
        }
      } finally {
        pop_renderer(previous_renderer2);
      }
      stop_listening?.(), connection.destroy?.();
    });
  }
  return result;
}

// node_modules/svelte/src/internal/client/dom/blocks/snippet.js
function snippet(node, get_snippet, ...args) {
  var branches = new BranchManager(node);
  block(() => {
    let snippet2 = get_snippet() ?? null;
    false_default && snippet2 == null && invalid_snippet(), branches.ensure(snippet2, snippet2 && ((anchor) => snippet2(anchor, ...args)));
  }, 65536);
}
function wrap_snippet(component2, fn) {
  let snippet2 = (node, ...args) => {
    var previous_component_function = dev_current_component_function;
    set_dev_current_component_function(component2);
    try {
      return fn(node, ...args);
    } finally {
      set_dev_current_component_function(previous_component_function);
    }
  };
  return prevent_snippet_stringification(snippet2), snippet2;
}
function renderer_snippet(renderer, fn) {
  var wrapped = (
    /** @type {T} */
    ((anchor, ...args) => is_crossing(renderer) ? cross(renderer, anchor, (anchor2) => fn(anchor2, ...args)) : fn(anchor, ...args))
  );
  return wrapped.__renderer = renderer, wrapped;
}
function validate_snippet_renderer(fn) {
  return fn != null && !has_own_property.call(fn, "__renderer") && snippet_renderer_mismatch(), fn;
}
function createRawSnippet(fn) {
  return (anchor, ...params) => {
    current_renderer != null && invalid_snippet_in_custom_renderer();
    var snippet2 = fn(...params), element2;
    if (hydrating)
      element2 = /** @type {Element} */
      hydrate_node, hydrate_next();
    else {
      var html2 = snippet2.render().trim(), fragment = create_fragment_from_html(html2);
      element2 = /** @type {Element} */
      get_first_child(fragment), false_default && (get_next_sibling(element2) !== null || node_type(element2) !== ELEMENT_NODE) && invalid_raw_snippet_render(), insert_before(anchor, element2);
    }
    false_default && dev_stack !== null && (element2.__svelte_meta = {
      parent: dev_stack,
      loc: { file: dev_stack.file, line: dev_stack.line, column: dev_stack.column },
      component: component_context
    });
    let result = snippet2.setup?.(element2);
    assign_nodes(element2, element2), typeof result == "function" && teardown(result);
  };
}

// node_modules/svelte/src/internal/client/dom/blocks/svelte-component.js
function component(node, get_component, render_fn) {
  var hydration_start_node;
  hydrating && (hydration_start_node = hydrate_node, hydrate_next());
  var branches = new BranchManager(node);
  block(() => {
    var component2 = get_component() ?? null;
    if (hydrating) {
      var data = read_hydration_instruction(
        /** @type {TemplateNode} */
        hydration_start_node
      ), server_had_component = data === "[", client_has_component = component2 !== null;
      if (server_had_component !== client_has_component) {
        var anchor = skip_nodes();
        set_hydrate_node(anchor), branches.anchor = anchor, set_hydrating(!1), branches.ensure(component2, component2 && ((target) => render_fn(target, component2))), set_hydrating(!0);
        return;
      }
    }
    branches.ensure(component2, component2 && ((target) => render_fn(target, component2)));
  }, 65536);
}

// node_modules/svelte/src/internal/client/timing.js
var now = true_default ? () => performance.now() : () => Date.now(), raf = {
  // don't access requestAnimationFrame eagerly outside method
  // this allows basic testing of user code without JSDOM
  // bunder will eval and remove ternary when the user's app is built
  tick: (
    /** @param {any} _ */
    (_) => (true_default ? requestAnimationFrame : noop)(_)
  ),
  now: () => now(),
  tasks: /* @__PURE__ */ new Set()
};

// node_modules/svelte/src/internal/client/loop.js
function run_tasks() {
  let now2 = raf.now();
  raf.tasks.forEach((task) => {
    task.c(now2) || (raf.tasks.delete(task), task.f());
  }), raf.tasks.size !== 0 && raf.tick(run_tasks);
}
function loop(callback) {
  let task;
  return raf.tasks.size === 0 && raf.tick(run_tasks), {
    promise: new Promise((fulfill) => {
      raf.tasks.add(task = { c: callback, f: fulfill });
    }),
    abort() {
      raf.tasks.delete(task);
    }
  };
}

// node_modules/svelte/src/internal/client/dom/elements/transitions.js
function dispatch_event(element2, type) {
  without_reactive_context(() => {
    element2.dispatchEvent(new CustomEvent(type));
  });
}
function css_property_to_camelcase(style) {
  if (style === "float") return "cssFloat";
  if (style === "offset") return "cssOffset";
  if (style.startsWith("--")) return style;
  let parts = style.split("-");
  return parts.length === 1 ? parts[0] : parts[0] + parts.slice(1).map(
    /** @param {any} word */
    (word) => word[0].toUpperCase() + word.slice(1)
  ).join("");
}
function css_to_keyframe(css) {
  let keyframe = {}, parts = css.split(";");
  for (let part of parts) {
    let [property, value] = part.split(":");
    if (!property || value === void 0) break;
    let formatted_property = css_property_to_camelcase(property.trim());
    keyframe[formatted_property] = value.trim();
  }
  return keyframe;
}
var linear = (t) => t, animation_effect_override = null;
function set_animation_effect_override(v) {
  animation_effect_override = v;
}
function animation(element2, get_fn, get_params) {
  var effect2 = animation_effect_override ?? /** @type {Effect} */
  active_effect, nodes = (
    /** @type {EffectNodes} */
    effect2.nodes
  ), from, to, animation2, original_styles = null;
  nodes.a ??= {
    element: element2,
    measure() {
      from = this.element.getBoundingClientRect();
    },
    apply() {
      if (animation2?.abort(), to = this.element.getBoundingClientRect(), from.left !== to.left || from.right !== to.right || from.top !== to.top || from.bottom !== to.bottom) {
        let options = get_fn()(this.element, { from, to }, get_params?.());
        animation2 = animate(
          this.element,
          options,
          void 0,
          1,
          () => {
          },
          () => {
            animation2?.abort(), animation2 = void 0;
          }
        );
      }
    },
    fix() {
      if (!element2.getAnimations().length) {
        var { position, width, height } = getComputedStyle(element2);
        if (position !== "absolute" && position !== "fixed") {
          var style = (
            /** @type {HTMLElement | SVGElement} */
            element2.style
          );
          original_styles = {
            position: style.position,
            width: style.width,
            height: style.height,
            transform: style.transform
          }, style.position = "absolute", style.width = width, style.height = height;
          var to2 = element2.getBoundingClientRect();
          if (from.left !== to2.left || from.top !== to2.top) {
            var transform = `translate(${from.left - to2.left}px, ${from.top - to2.top}px)`;
            style.transform = style.transform ? `${style.transform} ${transform}` : transform;
          }
        }
      }
    },
    unfix() {
      if (original_styles) {
        var style = (
          /** @type {HTMLElement | SVGElement} */
          element2.style
        );
        style.position = original_styles.position, style.width = original_styles.width, style.height = original_styles.height, style.transform = original_styles.transform;
      }
    }
  }, nodes.a.element = element2;
}
function transition(flags2, element2, get_fn, get_params) {
  var is_intro = (flags2 & 1) !== 0, is_outro = (flags2 & 2) !== 0, is_both = is_intro && is_outro, is_global = (flags2 & 4) !== 0, direction = is_both ? "both" : is_intro ? "in" : "out", current_options, inert = element2.inert, overflow = element2.style.overflow, intro, outro;
  function get_options() {
    return without_reactive_context(() => current_options ??= get_fn()(element2, get_params?.() ?? /** @type {P} */
    {}, {
      direction
    }));
  }
  var transition2 = {
    is_global,
    in() {
      if (element2.inert = inert, !is_intro) {
        outro?.abort(), outro?.reset?.();
        return;
      }
      is_outro || intro?.abort(), intro = animate(
        element2,
        get_options(),
        outro,
        1,
        () => {
          dispatch_event(element2, "introstart");
        },
        () => {
          dispatch_event(element2, "introend"), intro?.abort(), intro = current_options = void 0, element2.style.overflow = overflow;
        }
      );
    },
    out(fn) {
      if (!is_outro) {
        fn?.(), current_options = void 0;
        return;
      }
      element2.inert = !0, outro = animate(
        element2,
        get_options(),
        intro,
        0,
        () => {
          dispatch_event(element2, "outrostart");
        },
        () => {
          dispatch_event(element2, "outroend"), fn?.();
        }
      );
    },
    stop: () => {
      intro?.abort(), outro?.abort();
    }
  }, e = (
    /** @type {Effect & { nodes: EffectNodes }} */
    active_effect
  );
  if ((e.nodes.t ??= []).push(transition2), is_intro && should_intro) {
    var run3 = is_global;
    if (!run3) {
      for (var block2 = (
        /** @type {Effect | null} */
        e.parent
      ); block2 && (block2.f & 65536) !== 0; )
        for (; (block2 = block2.parent) && (block2.f & 16) === 0; )
          ;
      run3 = !block2 || (block2.f & 32768) !== 0;
    }
    run3 && effect(() => {
      untrack(() => transition2.in());
    });
  }
}
function animate(element2, options, counterpart, t2, on_begin, on_finish) {
  var is_intro = t2 === 1, aborted = !1;
  if (is_function(options)) {
    var a;
    return queue_micro_task(() => {
      if (!aborted) {
        var o = options({ direction: is_intro ? "in" : "out" });
        a = animate(element2, o, counterpart, t2, on_begin, on_finish);
      }
    }), {
      abort: () => {
        aborted = !0, a?.abort();
      },
      deactivate: () => a?.deactivate(),
      reset: () => a?.reset(),
      t: () => a?.t() ?? 1 - t2
    };
  }
  if (counterpart?.deactivate(), !options?.duration && !options?.delay)
    return on_begin(), on_finish(), {
      abort: noop,
      deactivate: noop,
      reset: noop,
      t: () => t2
    };
  let { delay = 0, css, tick: tick2, easing = linear } = options;
  var animation2, get_t = () => 1 - t2;
  return queue_micro_task(() => {
    if (!aborted) {
      var keyframes = [];
      if (is_intro && counterpart === void 0 && (tick2 && tick2(0, 1), css)) {
        var styles = css_to_keyframe(css(0, 1));
        keyframes.push(styles, styles);
      }
      animation2 = element2.animate(keyframes, { duration: delay, fill: "forwards" }), animation2.onfinish = () => {
        animation2.cancel(), on_begin();
        var t1 = counterpart?.t() ?? 1 - t2;
        counterpart?.abort();
        var delta = t2 - t1, duration = (
          /** @type {number} */
          options.duration * Math.abs(delta)
        ), keyframes2 = [];
        if (duration > 0) {
          var needs_overflow_hidden = !1;
          if (css)
            for (var n = Math.ceil(duration / 16.666666666666668), i = 0; i <= n; i += 1) {
              var t = t1 + delta * easing(i / n), styles2 = css_to_keyframe(css(t, 1 - t));
              keyframes2.push(styles2), needs_overflow_hidden ||= styles2.overflow === "hidden";
            }
          needs_overflow_hidden && (element2.style.overflow = "hidden"), get_t = () => {
            var time = (
              /** @type {number} */
              /** @type {globalThis.Animation} */
              animation2.currentTime
            );
            return t1 + delta * easing(time / duration);
          }, tick2 && loop(() => {
            if (animation2.playState !== "running") return !1;
            var t3 = get_t();
            return tick2(t3, 1 - t3), !0;
          });
        }
        animation2 = element2.animate(keyframes2, { duration, fill: "forwards" }), animation2.onfinish = () => {
          get_t = () => t2, tick2?.(t2, 1 - t2), on_finish();
        };
      };
    }
  }), {
    abort: () => {
      aborted = !0, animation2 && (animation2.cancel(), animation2.effect = null, animation2.onfinish = noop);
    },
    deactivate: () => {
      on_finish = noop;
    },
    reset: () => {
      t2 === 0 && tick2?.(1, 0);
    },
    t: () => get_t()
  };
}

// node_modules/svelte/src/internal/client/dom/blocks/svelte-element.js
function element(node, get_tag, is_svg, render_fn, get_namespace, location) {
  let was_hydrating = hydrating;
  hydrating && hydrate_next();
  var filename = false_default && location && component_context?.function[FILENAME], element2 = null;
  hydrating && node_type(hydrate_node) === ELEMENT_NODE && (element2 = /** @type {Element} */
  hydrate_node, hydrate_next());
  var anchor = (
    /** @type {TemplateNode} */
    hydrating ? hydrate_node : node
  ), parent_effect = (
    /** @type {Effect} */
    active_effect
  ), branches = new BranchManager(anchor, !1);
  block(() => {
    let next_tag = get_tag() || null;
    var ns = get_namespace ? get_namespace() : is_svg || next_tag === "svg" ? NAMESPACE_SVG : void 0;
    if (next_tag === null) {
      branches.ensure(null, null), set_should_intro(!0);
      return;
    }
    return branches.ensure(next_tag, (anchor2) => {
      if (next_tag) {
        var is_hydrating = hydrating;
        if (element2 = hydrating ? (
          /** @type {Element} */
          element2
        ) : create_element(next_tag, ns), false_default && location && (element2.__svelte_meta = {
          parent: dev_stack,
          loc: {
            file: filename,
            line: location[0],
            column: location[1]
          },
          component: component_context
        }), assign_nodes(element2, element2), render_fn) {
          var tmp_comment = null;
          hydrating && is_raw_text_element(next_tag) && append_child(element2, tmp_comment = create_comment(""));
          var child_anchor = hydrating ? get_first_child(element2) : (
            /** @type {Text} */
            append_child(element2, create_text())
          );
          hydrating && (child_anchor === null ? set_hydrating(!1) : set_hydrate_node(child_anchor)), set_animation_effect_override(parent_effect), render_fn(element2, child_anchor), tmp_comment && remove_child(element2, tmp_comment), set_animation_effect_override(null);
        }
        active_effect.nodes.end = element2, is_hydrating || insert_before(anchor2, element2);
      }
      hydrating && set_hydrate_node(anchor2);
    }), set_should_intro(!0), () => {
      next_tag && set_should_intro(!1);
    };
  }, 65536), teardown(() => {
    set_should_intro(!0);
  }), was_hydrating && (set_hydrating(!0), set_hydrate_node(anchor));
}

// node_modules/svelte/src/internal/client/dom/blocks/svelte-head.js
function head(hash2, render_fn) {
  let previous_hydrate_node = null, was_hydrating = hydrating;
  var anchor;
  if (hydrating) {
    previous_hydrate_node = hydrate_node;
    for (var head_anchor = get_first_child(document.head); head_anchor !== null && (node_type(head_anchor) !== COMMENT_NODE || get_node_value(head_anchor) !== hash2); )
      head_anchor = get_next_sibling(head_anchor);
    if (head_anchor === null)
      set_hydrating(!1);
    else {
      var start = (
        /** @type {TemplateNode} */
        get_next_sibling(head_anchor)
      );
      remove_node(
        /** @type {ChildNode} */
        head_anchor
      ), set_hydrate_node(start);
    }
  }
  hydrating || (anchor = /** @type {Comment | Text} */
  append_child(document.head, create_text()));
  try {
    block(() => {
      var e = branch(() => render_fn(anchor));
      e.f |= 262144, hydrating || (e.nodes === null ? e.nodes = { start: anchor, end: anchor, a: null, t: null } : e.nodes.end = anchor);
    });
  } finally {
    was_hydrating && (set_hydrating(!0), set_hydrate_node(
      /** @type {TemplateNode} */
      previous_hydrate_node
    ));
  }
}

// node_modules/svelte/src/internal/client/dom/css.js
function append_styles2(anchor, css) {
  effect(() => {
    anchor = active_effect?.parent?.nodes?.start ?? anchor;
    var root = anchor.getRootNode(), target = (
      /** @type {ShadowRoot} */
      root.host ? (
        /** @type {ShadowRoot} */
        root
      ) : (
        /** @type {Document} */
        root.head ?? /** @type {Document} */
        root.ownerDocument.head
      )
    );
    if (!/** @type {Element} */
    target.querySelector("#" + css.hash)) {
      let style = create_element("style");
      style.id = css.hash, set_text_content(style, css.code), append_child(target, style), false_default && register_style(css.hash, style);
    }
  });
}

// node_modules/svelte/src/internal/client/dom/elements/actions.js
function action(dom, action2, get_value) {
  effect(() => {
    var payload = untrack(() => action2(dom, get_value?.()) || {});
    if (get_value && payload?.update) {
      var inited = !1, prev = (
        /** @type {any} */
        {}
      );
      render_effect(() => {
        var value = get_value();
        deep_read_state(value), inited && safe_not_equal(prev, value) && (prev = value, payload.update(value));
      }), inited = !0;
    }
    if (payload?.destroy)
      return () => (
        /** @type {Function} */
        payload.destroy()
      );
  });
}

// node_modules/svelte/src/internal/client/dom/elements/attachments.js
function attach(node, get_fn) {
  var fn = void 0, e;
  managed(() => {
    fn !== (fn = get_fn()) && (e && (destroy_effect(e), e = null), fn && (e = branch(() => {
      effect(() => (
        /** @type {(node: Element) => void} */
        fn(node)
      ));
    })));
  });
}

// node_modules/svelte/src/internal/client/dom/elements/class.js
function set_class(dom, is_html, value, hash2, prev_classes, next_classes) {
  var prev = (
    /** @type {any} */
    dom[CLASS_CACHE]
  );
  if (hydrating || prev !== value || prev === void 0 || // for edge case of `class={undefined}`
  // custom renderers only deal in attributes, so rather than toggling individual classes
  // (which would mean editing the `class` attribute), always set the whole thing
  custom_renderers_flag && current_renderer !== null) {
    var next_class_name = to_class(value, hash2, next_classes);
    (!hydrating || next_class_name !== get_attribute(dom, "class")) && (next_class_name == null ? remove_attribute(dom, "class") : is_html ? dom.className = next_class_name : set_attribute(dom, "class", next_class_name)), dom[CLASS_CACHE] = value;
  } else if (next_classes && prev_classes !== next_classes)
    for (var key2 in next_classes) {
      var is_present = !!next_classes[key2];
      (prev_classes == null || is_present !== !!prev_classes[key2]) && class_list_toggle(
        /** @type {HTMLElement} */
        dom,
        key2,
        is_present
      );
    }
  return next_classes;
}

// node_modules/svelte/src/internal/client/dom/elements/style.js
function update_styles(dom, prev = {}, next2, priority) {
  for (var key2 in next2) {
    var value = next2[key2];
    prev[key2] !== value && (next2[key2] == null ? style_remove_property(
      /** @type {HTMLElement} */
      dom,
      key2
    ) : style_set_property(
      /** @type {HTMLElement} */
      dom,
      key2,
      value,
      priority
    ));
  }
}
function set_style(dom, value, prev_styles, next_styles) {
  var prev = (
    /** @type {any} */
    dom[STYLE_CACHE]
  );
  if (hydrating || prev !== value || custom_renderers_flag && current_renderer !== null) {
    var next_style_attr = to_style(value, next_styles);
    (!hydrating || next_style_attr !== get_attribute(dom, "style")) && (next_style_attr == null ? remove_attribute(dom, "style") : set_css_text(
      /** @type {HTMLElement} */
      dom,
      next_style_attr
    )), dom[STYLE_CACHE] = value;
  } else next_styles && (Array.isArray(next_styles) ? (update_styles(dom, prev_styles?.[0], next_styles[0]), update_styles(dom, prev_styles?.[1], next_styles[1], "important")) : update_styles(dom, prev_styles, next_styles));
  return next_styles;
}

// node_modules/svelte/src/internal/client/dom/elements/bindings/select.js
function set_selected(option, selected) {
  selected ? has_attribute(option, "selected") || set_attribute(option, "selected", "") : remove_attribute(option, "selected");
}
function set_default_select_value(select, value) {
  var mounting = !("__defaultValue" in select);
  !mounting && select.__defaultValue === value || (select.__defaultValue = value, apply_default_select_value(select, !mounting || "__value" in select));
}
function apply_default_select_value(select, preserve) {
  var value = select.__defaultValue, multiple = select.multiple, values = multiple ? value ?? [] : null;
  if (!(multiple && !is_array(values))) {
    var index2 = select.selectedIndex, selected = preserve && multiple ? new Set(select.selectedOptions) : null;
    for (var option of select.options) {
      var option_value = get_option_value(option);
      set_selected(
        option,
        multiple ? (
          /** @type {any[]} */
          values.includes(option_value)
        ) : is(option_value, value)
      );
    }
    if (preserve)
      if (selected !== null)
        for (option of select.options) {
          var was_selected = selected.has(option);
          option.selected !== was_selected && (option.selected = was_selected);
        }
      else select.selectedIndex !== index2 && (select.selectedIndex = index2);
  }
}
function select_option(select, value, mounting = !1) {
  if (select.multiple) {
    if (value == null)
      return;
    if (!is_array(value))
      return select_multiple_invalid_value();
    for (var option of select.options)
      option.selected = value.includes(get_option_value(option));
    return;
  }
  for (option of select.options) {
    var option_value = get_option_value(option);
    if (is(option_value, value)) {
      option.selected = !0;
      return;
    }
  }
  (!mounting || value !== void 0) && (select.selectedIndex = -1);
}
function init_select(select) {
  var observer = new MutationObserver((entries) => {
    entries.every(is_selectedcontent_mutation) || ("__defaultValue" in select && apply_default_select_value(select, !1), "__value" in select && select_option(select, select.__value));
  });
  observer.observe(select, {
    // Listen to option element changes
    childList: !0,
    subtree: !0,
    // because of <optgroup>
    // Listen to option element value attribute changes
    // (doesn't get notified of select value changes,
    // because that property is not reflected as an attribute)
    attributes: !0,
    attributeFilter: ["value"]
  }), teardown(() => {
    observer.disconnect();
  });
}
function bind_select_value(select, get3, set2 = get3) {
  var batches = /* @__PURE__ */ new WeakSet(), mounting = !0;
  listen_to_event_and_reset_event(select, "change", (is_reset) => {
    var query = is_reset ? "[selected]" : ":checked", value;
    if (select.multiple)
      value = [].map.call(select.querySelectorAll(query), get_option_value);
    else {
      var selected_option = select.querySelector(query) ?? // will fall back to first non-disabled option if no option is selected
      select.querySelector("option:not([disabled])");
      value = selected_option && get_option_value(selected_option);
    }
    set2(value), select.__value = value, current_batch !== null && batches.add(current_batch);
  }), effect(() => {
    var value = get3();
    if (!(select === document.activeElement && batches.has(
      /** @type {Batch} */
      view_batch
    ))) {
      if (select_option(select, value, mounting), mounting && value === void 0) {
        var selected_option = select.querySelector(":checked");
        selected_option !== null && (value = get_option_value(selected_option), set2(value));
      }
      select.__value = value, mounting = !1;
    }
  });
}
function get_option_value(option) {
  return "__value" in option ? option.__value : option.value;
}
function is_selectedcontent_mutation(entry) {
  if (
    /** @type {Element} */
    entry.target.closest("selectedcontent") !== null
  )
    return !0;
  if (entry.type === "childList") {
    var nodes = [...entry.addedNodes, ...entry.removedNodes];
    return nodes.length > 0 && nodes.every((node) => node.nodeName === "SELECTEDCONTENT");
  }
  return !1;
}

// node_modules/svelte/src/internal/client/dom/elements/attributes.js
var CLASS = /* @__PURE__ */ Symbol("class"), STYLE = /* @__PURE__ */ Symbol("style"), IS_CUSTOM_ELEMENT = /* @__PURE__ */ Symbol("is custom element"), IS_HTML = /* @__PURE__ */ Symbol("is html"), LINK_TAG = IS_XHTML ? "link" : "LINK", INPUT_TAG = IS_XHTML ? "input" : "INPUT", OPTION_TAG = IS_XHTML ? "option" : "OPTION", SELECT_TAG = IS_XHTML ? "select" : "SELECT", PROGRESS_TAG = IS_XHTML ? "progress" : "PROGRESS";
function remove_input_defaults(input) {
  if (hydrating) {
    var already_removed = !1, remove_defaults = () => {
      if (!already_removed) {
        if (already_removed = !0, has_attribute(input, "value")) {
          var value = input.value;
          set_attribute2(input, "value", null), input.value = value;
        }
        if (has_attribute(input, "checked")) {
          var checked = input.checked;
          set_attribute2(input, "checked", null), input.checked = checked;
        }
      }
    };
    input[FORM_RESET_HANDLER] = remove_defaults, queue_micro_task(remove_defaults), add_form_reset_listener();
  }
}
function set_value(element2, value) {
  var attributes = get_attributes(element2);
  attributes.value === (attributes.value = // treat null and undefined the same for the initial value
  value ?? void 0) || // `progress` elements always need their value set when it's `0`
  current_renderer == null && /** @type {any} */
  element2.value === value && (value !== 0 || node_name(element2) !== PROGRESS_TAG) || set_element_value(element2, value);
}
function set_checked(element2, checked) {
  var attributes = get_attributes(element2);
  attributes.checked !== (attributes.checked = // treat null and undefined the same for the initial value
  checked ?? void 0) && set_element_checked(element2, checked);
}
function set_default_checked(element2, checked) {
  set_element_default_checked(element2, checked);
}
function set_default_value(element2, value) {
  set_element_default_value(element2, value);
}
function set_attribute2(element2, attribute, value, skip_warning) {
  var attributes = get_attributes(element2);
  if (hydrating && (attributes[attribute] = get_attribute(element2, attribute), attribute === "src" || attribute === "srcset" || attribute === "href" && node_name(element2) === LINK_TAG)) {
    skip_warning || check_src_in_dev_hydration(element2, attribute, value ?? "");
    return;
  }
  var previous = attributes[attribute];
  previous !== (attributes[attribute] = value) && (attribute === "loading" && (element2[LOADING_ATTR_SYMBOL] = value), value == null ? remove_attribute(element2, attribute) : typeof value != "string" && get_setters(element2).has(attribute) ? element2[attribute] = value : hydrating && (typeof value == "number" || typeof value == "boolean") && previous === String(value) || set_attribute(element2, attribute, value));
}
function set_xlink_attribute(dom, attribute, value) {
  dom.setAttributeNS("http://www.w3.org/1999/xlink", attribute, value);
}
function set_custom_element_data(node, prop2, value) {
  var previous_reaction = active_reaction, previous_effect = active_effect;
  let was_hydrating = hydrating;
  hydrating && set_hydrating(!1), set_active_reaction(null), set_active_effect(null);
  try {
    // `style` should use `set_attribute` rather than the setter
    prop2 !== "style" && // Don't compute setters for custom elements while they aren't registered yet,
    // because during their upgrade/instantiation they might add more setters.
    // Instead, fall back to a simple "an object, then set as property" heuristic.
    (setters_cache.has(get_attribute(node, "is") || node_name(node)) || // customElements may not be available in browser extension contexts
    !customElements || customElements.get(get_attribute(node, "is") || node_name(node).toLowerCase()) ? get_setters(node).has(prop2) : value && typeof value == "object") ? node[prop2] = value : set_attribute2(node, prop2, value == null ? value : String(value));
  } finally {
    set_active_reaction(previous_reaction), set_active_effect(previous_effect), was_hydrating && set_hydrating(!0);
  }
}
function set_attributes(element2, prev, next2, css_hash, should_remove_defaults = !1, skip_warning = !1) {
  hydrating && should_remove_defaults && node_name(element2) === INPUT_TAG && ("defaultValue" in next2 || "defaultChecked" in next2 || remove_input_defaults(
    /** @type {HTMLInputElement} */
    element2
  ));
  var attributes = get_attributes(element2), is_custom_element = attributes[IS_CUSTOM_ELEMENT], preserve_attribute_case = !attributes[IS_HTML];
  let is_hydrating_custom_element = hydrating && is_custom_element;
  is_hydrating_custom_element && set_hydrating(!1);
  var current = prev || {}, is_option_element = node_name(element2) === OPTION_TAG, is_select_element = node_name(element2) === SELECT_TAG;
  try {
    for (var key2 in prev)
      !(key2 in next2) && key2[0] + key2[1] !== "$$" && (next2[key2] = null);
    next2.class ? next2.class = clsx2(next2.class) : (css_hash || next2[CLASS]) && (next2.class = null), next2[STYLE] && (next2.style ??= null);
    var setters = get_setters(element2);
    if (
      // the following logic only makes sense in a DOM environment
      current_renderer == null && node_name(element2) === INPUT_TAG && "type" in next2 && ("value" in next2 || "__value" in next2)
    ) {
      var type = next2.type;
      (type !== current.type || type === void 0 && element2.hasAttribute("type")) && (current.type = type, set_attribute2(element2, "type", type, skip_warning));
    }
    for (let key3 in next2) {
      let value = next2[key3];
      if (is_option_element && key3 === "value" && value == null) {
        element2.value = element2.__value = "", current[key3] = value;
        continue;
      }
      if (key3 === "class") {
        var is_html = current_renderer == null && element2.namespaceURI === NAMESPACE_HTML;
        set_class(element2, is_html, value, css_hash, prev?.[CLASS], next2[CLASS]), current[key3] = value, current[CLASS] = next2[CLASS];
        continue;
      }
      if (key3 === "style") {
        set_style(element2, value, prev?.[STYLE], next2[STYLE]), current[key3] = value, current[STYLE] = next2[STYLE];
        continue;
      }
      var prev_value = current[key3];
      if (!(value === prev_value && !(value === void 0 && has_attribute(element2, key3)))) {
        current[key3] = value;
        var prefix = key3[0] + key3[1];
        if (prefix !== "$$")
          if (prefix === "on") {
            let opts = {}, event_handle_key = "$$" + key3, event_name = key3.slice(2);
            var is_delegated = current_renderer == null && can_delegate_event(event_name);
            if (is_capture_event(event_name) && (event_name = event_name.slice(0, -7), opts.capture = !0), !is_delegated && prev_value) {
              if (value != null) continue;
              remove_event_listener(element2, event_name, current[event_handle_key], opts), current[event_handle_key] = null;
            }
            if (is_delegated)
              delegated(event_name, element2, value), delegate([event_name]);
            else if (value != null) {
              let handle = function(...args) {
                current[key3].apply(this, args);
              };
              current[event_handle_key] = create_event(event_name, element2, handle, opts);
            }
          } else if (key3 === "style")
            set_attribute2(element2, key3, value);
          else if (key3 === "autofocus")
            current_renderer == null ? autofocus(
              /** @type {HTMLElement} */
              element2,
              !!value
            ) : value ? set_attribute(element2, key3, value) : remove_attribute(element2, key3);
          else if (!is_custom_element && (key3 === "__value" || key3 === "value" && value != null))
            element2.__value = value, (prev_value == null || // @ts-ignore
            element2.value !== value || value === 0 && element2.nodeName === PROGRESS_TAG) && set_element_value(element2, value);
          else if (key3 === "selected" && is_option_element)
            set_selected(
              /** @type {HTMLOptionElement} */
              element2,
              value
            );
          else {
            var name = key3;
            preserve_attribute_case || (name = normalize_attribute(name));
            var is_default = name === "defaultValue" || name === "defaultChecked";
            if (is_select_element && name === "defaultValue") continue;
            if (value == null && !is_custom_element && !is_default)
              if (attributes[key3] = null, (name === "value" || name === "checked") && current_renderer == null) {
                let input = (
                  /** @type {HTMLInputElement} */
                  element2
                ), use_default = prev === void 0;
                if (name === "value") {
                  let previous = input.defaultValue;
                  remove_attribute(input, name), set_element_default_value(input, previous), set_element_value(input, input.__value = use_default ? previous : null);
                } else {
                  let previous = input.defaultChecked;
                  remove_attribute(input, name), set_element_default_checked(input, previous), set_element_checked(input, use_default ? previous : !1);
                }
              } else
                remove_attribute(element2, key3), name === "value" && (element2.__value = null);
            else is_default && current_renderer != null ? (name === "defaultValue" ? set_element_default_value(element2, value) : set_element_default_checked(element2, value), name in attributes && (attributes[name] = UNINITIALIZED)) : is_default || (is_custom_element || typeof value != "string") && setters.has(name) ? (element2[name] = value, name in attributes && (attributes[name] = UNINITIALIZED)) : typeof value != "function" && set_attribute2(element2, name, value, skip_warning);
          }
      }
    }
  } finally {
    is_hydrating_custom_element && set_hydrating(!0);
  }
  return current;
}
function attribute_effect(element2, fn, sync2 = [], async2 = [], blockers = [], css_hash, should_remove_defaults = !1, skip_warning = !1) {
  flatten(blockers, sync2, async2, (values) => {
    var prev = void 0, effects = {}, is_select = node_name(element2) === SELECT_TAG, inited = !1;
    if (managed(() => {
      var next2 = fn(...values.map(get)), current = set_attributes(
        element2,
        prev,
        next2,
        css_hash,
        should_remove_defaults,
        skip_warning
      );
      if (inited && is_select) {
        var select2 = (
          /** @type {HTMLSelectElement} */
          element2
        );
        "defaultValue" in next2 && set_default_select_value(select2, next2.defaultValue), "value" in next2 && select_option(select2, next2.value);
      }
      for (let symbol of Object.getOwnPropertySymbols(effects))
        next2[symbol] || destroy_effect(effects[symbol]);
      for (let symbol of Object.getOwnPropertySymbols(next2)) {
        var n = next2[symbol];
        symbol.description === ATTACHMENT_KEY && (!prev || n !== prev[symbol]) && (effects[symbol] && destroy_effect(effects[symbol]), effects[symbol] = branch(() => attach(element2, () => n))), current[symbol] = n;
      }
      prev = current;
    }), is_select) {
      var select = (
        /** @type {HTMLSelectElement} */
        element2
      );
      effect(() => {
        var attrs = (
          /** @type {Record<string | symbol, any>} */
          prev
        );
        "defaultValue" in attrs && set_default_select_value(select, attrs.defaultValue), select_option(select, attrs.value, !0), init_select(select);
      });
    }
    inited = !0;
  });
}
function get_attributes(element2) {
  return (
    /** @type {Record<string | symbol, unknown>} **/
    /** @type {any} */
    element2[ATTRIBUTES_CACHE] ??= {
      [IS_CUSTOM_ELEMENT]: node_name(element2).includes("-"),
      [IS_HTML]: current_renderer == null && element2.namespaceURI === NAMESPACE_HTML
    }
  );
}
var setters_cache = /* @__PURE__ */ new Map(), empty_setters = /* @__PURE__ */ new Set();
function get_setters(element2) {
  if (current_renderer) return empty_setters;
  var cache_key = get_attribute(element2, "is") || node_name(element2), setters = setters_cache.get(cache_key);
  if (setters) return setters;
  setters_cache.set(cache_key, setters = /* @__PURE__ */ new Set());
  for (var descriptors, proto = element2, element_proto = Element.prototype; element_proto !== proto; ) {
    descriptors = get_descriptors(proto);
    for (var key2 in descriptors)
      descriptors[key2].set && // better safe than sorry, we don't want spread attributes to mess with HTML content
      key2 !== "innerHTML" && key2 !== "textContent" && key2 !== "innerText" && setters.add(key2);
    proto = get_prototype_of(proto);
  }
  return setters;
}
function check_src_in_dev_hydration(element2, attribute, value) {
  !false_default || current_renderer != null || attribute === "srcset" && srcset_url_equal(element2, value) || src_url_equal(get_attribute(element2, attribute) ?? "", value) || hydration_attribute_changed(
    attribute,
    element2.outerHTML.replace(element2.innerHTML, element2.innerHTML && "..."),
    String(value)
  );
}
function src_url_equal(element_src, url) {
  return element_src === url || current_renderer != null ? !0 : new URL(element_src, document.baseURI).href === new URL(url, document.baseURI).href;
}
function split_srcset(srcset) {
  return srcset.split(",").map((src) => src.trim().split(" ").filter(Boolean));
}
function srcset_url_equal(element2, srcset) {
  if (current_renderer != null) return !0;
  var element_urls = split_srcset(element2.srcset), urls = split_srcset(srcset);
  return urls.length === element_urls.length && urls.every(
    ([url, width], i) => width === element_urls[i][1] && // We need to test both ways because Vite will create an a full URL with
    // `new URL(asset, import.meta.url).href` for the client when `base: './'`, and the
    // relative URLs inside srcset are not automatically resolved to absolute URLs by
    // browsers (in contrast to img.src). This means both SSR and DOM code could
    // contain relative or absolute URLs.
    (src_url_equal(element_urls[i][0], url) || src_url_equal(url, element_urls[i][0]))
  );
}

// node_modules/svelte/src/internal/client/dom/elements/customizable-select.js
var supported = null;
function is_supported() {
  if (supported === null) {
    var select = create_element("select");
    select.innerHTML = create_trusted_html("<option><span>t</span></option>"), supported = /** @type {Element} */
    select.firstChild?.firstChild?.nodeType === 1;
  }
  return supported;
}
function selectedcontent(element2, update_element) {
  is_supported() && attach(element2, () => () => {
    let select = element2.closest("select");
    if (!select) return;
    let observer = new MutationObserver((entries) => {
      var selected = !1;
      for (let entry of entries) {
        if (entry.target === element2)
          return;
        selected ||= !!entry.target.parentElement?.closest("option")?.selected;
      }
      selected && (element2.replaceWith(element2 = /** @type {HTMLElement} */
      element2.cloneNode(!0)), update_element(element2));
    });
    return observer.observe(select, {
      childList: !0,
      characterData: !0,
      subtree: !0
    }), () => {
      observer.disconnect();
    };
  });
}
function customizable_select(element2, rich_fn) {
  var was_hydrating = hydrating;
  is_supported() || (set_hydrating(!1), element2.textContent = "", element2.append(create_comment("")));
  try {
    rich_fn();
  } finally {
    was_hydrating && (hydrating ? reset(element2) : (set_hydrating(!0), set_hydrate_node(element2)));
  }
}

// node_modules/svelte/src/internal/client/dom/elements/bindings/document.js
function bind_active_element(update2) {
  listen(document, ["focusin", "focusout"], (event2) => {
    event2 && event2.type === "focusout" && /** @type {FocusEvent} */
    event2.relatedTarget || without_reactive_context(() => update2(document.activeElement));
  });
}

// node_modules/svelte/src/internal/client/dom/elements/bindings/input.js
function bind_value(input, get3, set2 = get3) {
  var batches = /* @__PURE__ */ new WeakSet();
  listen_to_event_and_reset_event(input, "input", async (is_reset) => {
    false_default && input.type === "checkbox" && bind_invalid_checkbox_value();
    var value = is_reset ? input.defaultValue : input.value;
    if (value = is_numberlike_input(input) ? to_number(value) : value, set2(value), current_batch !== null && batches.add(current_batch), await tick(), value !== (value = get3())) {
      var start = input.selectionStart, end = input.selectionEnd, length = input.value.length;
      if (input.value = value ?? "", end !== null) {
        var new_length = input.value.length;
        start === end && end === length && new_length > length ? (input.selectionStart = new_length, input.selectionEnd = new_length) : (input.selectionStart = start, input.selectionEnd = Math.min(end, new_length));
      }
    }
  }), // If we are hydrating and the value has since changed,
  // then use the updated value from the input instead.
  (hydrating && input.defaultValue !== input.value || // If defaultValue is set, then value == defaultValue
  // TODO Svelte 6: remove input.value check and set to empty string?
  untrack(get3) == null && input.value) && (set2(is_numberlike_input(input) ? to_number(input.value) : input.value), current_batch !== null && batches.add(current_batch)), render_effect(() => {
    false_default && input.type === "checkbox" && bind_invalid_checkbox_value();
    var value = get3();
    input === document.activeElement && batches.has(
      /** @type {Batch} */
      view_batch
    ) || is_numberlike_input(input) && value === to_number(input.value) || input.type === "date" && !value && !input.value || value !== input.value && (input.value = value ?? "");
  });
}
var pending2 = /* @__PURE__ */ new Set();
function bind_group(inputs, group_index, input, get3, set2 = get3) {
  var is_checkbox = input.getAttribute("type") === "checkbox", binding_group = inputs;
  let hydration_mismatch2 = !1;
  if (group_index !== null)
    for (var index2 of group_index)
      binding_group = binding_group[index2] ??= [];
  binding_group.push(input), listen_to_event_and_reset_event(
    input,
    "change",
    () => {
      var value = input.__value;
      is_checkbox && (value = get_binding_group_value(binding_group, value, input.checked)), set2(value);
    },
    // TODO better default value handling
    () => set2(is_checkbox ? [] : null)
  ), render_effect(() => {
    var value = get3();
    if (hydrating && input.defaultChecked !== input.checked) {
      hydration_mismatch2 = !0;
      return;
    }
    is_checkbox ? (value = value || [], input.checked = value.includes(input.__value)) : input.checked = is(input.__value, value);
  }), teardown(() => {
    var index3 = binding_group.indexOf(input);
    index3 !== -1 && binding_group.splice(index3, 1);
  }), pending2.has(binding_group) || (pending2.add(binding_group), queue_micro_task(() => {
    binding_group.sort((a, b) => a.compareDocumentPosition(b) === 4 ? -1 : 1), pending2.delete(binding_group);
  })), queue_micro_task(() => {
    if (hydration_mismatch2) {
      var value;
      if (is_checkbox)
        value = get_binding_group_value(binding_group, value, input.checked);
      else {
        var hydration_input = binding_group.find((input2) => input2.checked);
        value = hydration_input?.__value;
      }
      set2(value);
    }
  });
}
function bind_checked(input, get3, set2 = get3) {
  listen_to_event_and_reset_event(input, "change", (is_reset) => {
    var value = is_reset ? input.defaultChecked : input.checked;
    set2(value);
  }), // If we are hydrating and the value has since changed,
  // then use the update value from the input instead.
  (hydrating && input.defaultChecked !== input.checked || // If defaultChecked is set, then checked == defaultChecked
  untrack(get3) == null) && set2(input.checked), render_effect(() => {
    var value = get3();
    input.checked = !!value;
  });
}
function get_binding_group_value(group, __value, checked) {
  for (var value = /* @__PURE__ */ new Set(), i = 0; i < group.length; i += 1)
    group[i].checked && value.add(group[i].__value);
  return checked || value.delete(__value), Array.from(value);
}
function is_numberlike_input(input) {
  var type = input.type;
  return type === "number" || type === "range";
}
function to_number(value) {
  return value === "" ? null : +value;
}
function bind_files(input, get3, set2 = get3) {
  listen_to_event_and_reset_event(
    input,
    "change",
    () => {
      set2(input.files);
    },
    () => {
      typeof DataTransfer == "function" && (input.files = new DataTransfer().files), set2(input.files);
    }
  ), // If we are hydrating and the value has since changed,
  // then use the updated value from the input instead.
  hydrating && input.files && set2(input.files), render_effect(() => {
    input.files = get3();
  });
}

// node_modules/svelte/src/internal/client/dom/elements/bindings/media.js
function time_ranges_to_array(ranges) {
  for (var array = [], i = 0; i < ranges.length; i += 1)
    array.push({ start: ranges.start(i), end: ranges.end(i) });
  return array;
}
function bind_current_time(media, get3, set2 = get3) {
  var raf_id, value, callback = () => {
    cancelAnimationFrame(raf_id), media.paused || (raf_id = requestAnimationFrame(callback));
    var next_value = media.currentTime;
    value !== next_value && set2(value = next_value);
  };
  raf_id = requestAnimationFrame(callback), media.addEventListener("timeupdate", callback), render_effect(() => {
    var next_value = Number(get3());
    value !== next_value && !isNaN(
      /** @type {any} */
      next_value
    ) && (media.currentTime = value = next_value);
  }), teardown(() => {
    cancelAnimationFrame(raf_id), media.removeEventListener("timeupdate", callback);
  });
}
function bind_buffered(media, set2) {
  var current;
  listen(media, ["loadedmetadata", "progress", "timeupdate", "seeking"], () => {
    var ranges = media.buffered;
    (!current || current.length !== ranges.length || current.some((range, i) => ranges.start(i) !== range.start || ranges.end(i) !== range.end)) && (current = time_ranges_to_array(ranges), set2(current));
  });
}
function bind_seekable(media, set2) {
  listen(media, ["loadedmetadata"], () => set2(time_ranges_to_array(media.seekable)));
}
function bind_played(media, set2) {
  listen(media, ["timeupdate"], () => set2(time_ranges_to_array(media.played)));
}
function bind_seeking(media, set2) {
  listen(media, ["seeking", "seeked"], () => set2(media.seeking));
}
function bind_ended(media, set2) {
  listen(media, ["timeupdate", "ended"], () => set2(media.ended));
}
function bind_ready_state(media, set2) {
  listen(
    media,
    ["loadedmetadata", "loadeddata", "canplay", "canplaythrough", "playing", "waiting", "emptied"],
    () => set2(media.readyState)
  );
}
function bind_playback_rate(media, get3, set2 = get3) {
  effect(() => {
    var value = Number(get3());
    value !== media.playbackRate && !isNaN(value) && (media.playbackRate = value);
  }), effect(() => {
    listen(media, ["ratechange"], () => {
      set2(media.playbackRate);
    });
  });
}
function bind_paused(media, get3, set2 = get3) {
  var paused = get3(), update2 = () => {
    paused !== media.paused && set2(paused = media.paused);
  };
  listen(media, ["play", "pause", "canplay"], update2, paused == null), effect(() => {
    (paused = !!get3()) !== media.paused && (paused ? media.pause() : media.play().catch((error) => {
      throw set2(paused = !0), error;
    }));
  });
}
function bind_volume(media, get3, set2 = get3) {
  var callback = () => {
    set2(media.volume);
  };
  get3() == null && callback(), listen(media, ["volumechange"], callback, !1), render_effect(() => {
    var value = Number(get3());
    value !== media.volume && !isNaN(value) && (media.volume = value);
  });
}
function bind_muted(media, get3, set2 = get3) {
  var callback = () => {
    set2(media.muted);
  };
  get3() == null && callback(), listen(media, ["volumechange"], callback, !1), render_effect(() => {
    var value = !!get3();
    media.muted !== value && (media.muted = value);
  });
}

// node_modules/svelte/src/internal/client/dom/elements/bindings/navigator.js
function bind_online(update2) {
  listen(window, ["online", "offline"], () => {
    update2(navigator.onLine);
  });
}

// node_modules/svelte/src/internal/client/dom/elements/bindings/props.js
function bind_prop(props, prop2, value) {
  var desc = get_descriptor(props, prop2);
  desc && desc.set && (props[prop2] = value, teardown(() => {
    props[prop2] = null;
  }));
}

// node_modules/svelte/src/internal/client/dom/elements/bindings/size.js
var ResizeObserverSingleton = class _ResizeObserverSingleton {
  /** */
  #listeners = /* @__PURE__ */ new WeakMap();
  /** @type {ResizeObserver | undefined} */
  #observer;
  /** @type {ResizeObserverOptions} */
  #options;
  /** @static */
  static entries = /* @__PURE__ */ new WeakMap();
  /** @param {ResizeObserverOptions} options */
  constructor(options) {
    this.#options = options;
  }
  /**
   * @param {Element} element
   * @param {(entry: ResizeObserverEntry) => any} listener
   */
  observe(element2, listener) {
    var listeners = this.#listeners.get(element2) || /* @__PURE__ */ new Set();
    return listeners.add(listener), this.#listeners.set(element2, listeners), this.#getObserver().observe(element2, this.#options), () => {
      var listeners2 = this.#listeners.get(element2);
      listeners2.delete(listener), listeners2.size === 0 && (this.#listeners.delete(element2), this.#observer.unobserve(element2));
    };
  }
  #getObserver() {
    return this.#observer ?? (this.#observer = new ResizeObserver(
      /** @param {any} entries */
      (entries) => {
        for (var entry of entries) {
          _ResizeObserverSingleton.entries.set(entry.target, entry);
          for (var listener of this.#listeners.get(entry.target) || [])
            listener(entry);
        }
      }
    ));
  }
}, resize_observer_content_box = /* @__PURE__ */ new ResizeObserverSingleton({
  box: "content-box"
}), resize_observer_border_box = /* @__PURE__ */ new ResizeObserverSingleton({
  box: "border-box"
}), resize_observer_device_pixel_content_box = /* @__PURE__ */ new ResizeObserverSingleton({
  box: "device-pixel-content-box"
});
function bind_resize_observer(element2, type, set2) {
  var observer = type === "contentRect" || type === "contentBoxSize" ? resize_observer_content_box : type === "borderBoxSize" ? resize_observer_border_box : resize_observer_device_pixel_content_box, unsub = observer.observe(
    element2,
    /** @param {any} entry */
    (entry) => set2(entry[type])
  );
  teardown(unsub);
}
function bind_element_size(element2, type, set2) {
  var unsub = resize_observer_border_box.observe(element2, () => set2(element2[type]));
  effect(() => (untrack(() => set2(element2[type])), unsub));
}

// node_modules/svelte/src/internal/client/dom/elements/bindings/this.js
function is_bound_this(bound_value, element_or_component) {
  return bound_value === element_or_component || bound_value?.[STATE_SYMBOL] === element_or_component;
}
function bind_this(element_or_component = mark_as_component(), update2, get_value, get_parts) {
  var component_effect = (
    /** @type {ComponentContext} */
    component_context.r
  ), parent = (
    /** @type {Effect} */
    active_effect
  );
  return effect(() => {
    var old_parts, parts;
    return render_effect(() => {
      old_parts = parts, parts = get_parts?.() || [], untrack(() => {
        is_bound_this(get_value(...parts), element_or_component) || (update2(element_or_component, ...parts), old_parts && is_bound_this(get_value(...old_parts), element_or_component) && update2(null, ...old_parts));
      });
    }), () => {
      let p = parent;
      for (; p !== component_effect && p.parent !== null && p.parent.f & 33554432; )
        p = p.parent;
      let teardown2 = () => {
        parts && is_bound_this(get_value(...parts), element_or_component) && update2(null, ...parts);
      }, original_teardown = p.teardown;
      p.teardown = () => {
        teardown2(), original_teardown?.();
      };
    };
  }), element_or_component;
}

// node_modules/svelte/src/internal/client/dom/elements/bindings/universal.js
function bind_content_editable(property, element2, get3, set2 = get3) {
  element2.addEventListener("input", () => {
    set2(element2[property]);
  }), render_effect(() => {
    var value = get3();
    if (element2[property] !== value)
      if (value == null) {
        var non_null_value = element2[property];
        set2(non_null_value);
      } else
        element2[property] = value + "";
  });
}
function bind_property(property, event_name, element2, set2, get3) {
  var handler = () => {
    set2(element2[property]);
  };
  element2.addEventListener(event_name, handler), get3 ? render_effect(() => {
    element2[property] = get3();
  }) : handler(), (element2 === document.body || element2 === window || element2 === document) && teardown(() => {
    element2.removeEventListener(event_name, handler);
  });
}
function bind_focused(element2, set2) {
  listen(element2, ["focus", "blur"], () => {
    set2(element2 === document.activeElement);
  });
}

// node_modules/svelte/src/internal/client/dom/elements/bindings/window.js
function bind_window_scroll(type, get3, set2 = get3) {
  var is_scrolling_x = type === "x", target_handler = () => without_reactive_context(() => {
    scrolling = !0, clearTimeout(timeout), timeout = setTimeout(clear, 100), set2(window[is_scrolling_x ? "scrollX" : "scrollY"]);
  });
  addEventListener("scroll", target_handler, {
    passive: !0
  });
  var scrolling = !1, timeout, clear = () => {
    scrolling = !1;
  }, first = !0;
  render_effect(() => {
    var latest_value = get3();
    first ? first = !1 : !scrolling && latest_value != null && (scrolling = !0, clearTimeout(timeout), is_scrolling_x ? scrollTo(latest_value, window.scrollY) : scrollTo(window.scrollX, latest_value), timeout = setTimeout(clear, 100));
  }), effect(target_handler), teardown(() => {
    removeEventListener("scroll", target_handler);
  });
}
function bind_window_size(type, set2) {
  listen(window, ["resize"], () => without_reactive_context(() => set2(window[type])));
}

// node_modules/svelte/src/internal/client/dom/legacy/event-modifiers.js
function trusted(fn) {
  return function(...args) {
    var event2 = (
      /** @type {Event} */
      args[0]
    );
    event2.isTrusted && fn?.apply(this, args);
  };
}
function self(fn) {
  return function(...args) {
    var event2 = (
      /** @type {Event} */
      args[0]
    );
    event2.target === this && fn?.apply(this, args);
  };
}
function stopPropagation(fn) {
  return function(...args) {
    var event2 = (
      /** @type {Event} */
      args[0]
    );
    return event2.stopPropagation(), fn?.apply(this, args);
  };
}
function once(fn) {
  var ran = !1;
  return function(...args) {
    if (!ran)
      return ran = !0, fn?.apply(this, args);
  };
}
function stopImmediatePropagation(fn) {
  return function(...args) {
    var event2 = (
      /** @type {Event} */
      args[0]
    );
    return event2.stopImmediatePropagation(), fn?.apply(this, args);
  };
}
function preventDefault(fn) {
  return function(...args) {
    var event2 = (
      /** @type {Event} */
      args[0]
    );
    return event2.preventDefault(), fn?.apply(this, args);
  };
}

// node_modules/svelte/src/internal/client/dom/legacy/lifecycle.js
function init(immutable = !1) {
  let context = (
    /** @type {ComponentContextLegacy} */
    component_context
  ), callbacks = context.l.u;
  if (!callbacks) return;
  let props = () => deep_read_state(context.s);
  if (immutable) {
    let version = 0, prev = (
      /** @type {Record<string, any>} */
      {}
    ), d = derived(() => {
      let changed = !1, props2 = context.s;
      for (let key2 in props2)
        props2[key2] !== prev[key2] && (prev[key2] = props2[key2], changed = !0);
      return changed && version++, version;
    });
    props = () => get(d);
  }
  callbacks.b.length && user_pre_effect(() => {
    observe_all(context, props), run_all(callbacks.b);
  }), user_effect(() => {
    let fns = untrack(() => callbacks.m.map(run));
    return () => {
      for (let fn of fns)
        typeof fn == "function" && fn();
    };
  }), callbacks.a.length && user_effect(() => {
    observe_all(context, props), run_all(callbacks.a);
  });
}
function observe_all(context, props) {
  if (context.l.s)
    for (let signal of context.l.s) get(signal);
  props();
}

// node_modules/svelte/src/internal/client/dom/legacy/misc.js
function reactive_import(fn) {
  var s = source(0);
  return function() {
    return arguments.length === 1 ? (set(s, get(s) + 1), arguments[0]) : (get(s), fn());
  };
}
function bubble_event($$props, event2) {
  var events = (
    /** @type {Record<string, Function[] | Function>} */
    $$props.$$events?.[event2.type]
  ), callbacks = is_array(events) ? events.slice() : events == null ? [] : [events];
  for (var fn of callbacks)
    fn.call(this, event2);
}
function add_legacy_event_listener($$props, event_name, event_callback) {
  $$props.$$events ||= {}, $$props.$$events[event_name] ||= [], $$props.$$events[event_name].push(event_callback);
}
function update_legacy_props($$new_props) {
  for (var key2 in $$new_props)
    key2 in this && (this[key2] = $$new_props[key2]);
}

// node_modules/svelte/src/internal/client/reactivity/store.js
var legacy_is_updating_store = !1, is_store_binding = !1, IS_UNMOUNTED = /* @__PURE__ */ Symbol("unmounted");
function store_get(store, store_name, stores) {
  let entry = stores[store_name] ??= {
    store: null,
    source: mutable_source(void 0),
    unsubscribe: noop
  };
  if (false_default && (entry.source.label = store_name), entry.store !== store && !(IS_UNMOUNTED in stores))
    if (entry.unsubscribe(), entry.store = store ?? null, store == null)
      entry.source.v = void 0, entry.unsubscribe = noop;
    else {
      var is_synchronous_callback = !0;
      entry.unsubscribe = subscribe_to_store(store, (v) => {
        is_synchronous_callback ? entry.source.v = v : set(entry.source, v);
      }), is_synchronous_callback = !1;
    }
  return store && IS_UNMOUNTED in stores ? get2(store) : get(entry.source);
}
function store_unsub(store, store_name, stores) {
  let entry = stores[store_name];
  return entry && entry.store !== store && (entry.unsubscribe(), entry.unsubscribe = noop), store;
}
function store_set(store, value) {
  return update_with_flag(store, value), value;
}
function invalidate_store(stores, store_name) {
  var entry = stores[store_name];
  entry.store !== null && store_set(entry.store, entry.source.v);
}
function setup_stores() {
  let stores = {};
  function cleanup() {
    teardown(() => {
      for (var store_name in stores)
        stores[store_name].unsubscribe();
      define_property(stores, IS_UNMOUNTED, {
        enumerable: !1,
        value: !0
      });
    });
  }
  return [stores, cleanup];
}
function update_with_flag(store, value) {
  legacy_is_updating_store = !0;
  try {
    store.set(value);
  } finally {
    legacy_is_updating_store = !1;
  }
}
function store_mutate(store, expression, new_value) {
  return update_with_flag(store, new_value), expression;
}
function update_store(store, store_value, d = 1) {
  return update_with_flag(store, store_value + d), store_value;
}
function update_pre_store(store, store_value, d = 1) {
  let value = store_value + d;
  return update_with_flag(store, value), value;
}
function mark_store_binding() {
  is_store_binding = !0;
}
function capture_store_binding(fn) {
  var previous_is_store_binding = is_store_binding;
  try {
    return is_store_binding = !1, [fn(), is_store_binding];
  } finally {
    is_store_binding = previous_is_store_binding;
  }
}

// node_modules/svelte/src/internal/client/reactivity/props.js
function update_prop(fn, d = 1) {
  let value = fn();
  return fn(value + d), value;
}
function update_pre_prop(fn, d = 1) {
  let value = fn() + d;
  return fn(value), value;
}
var rest_props_handler = {
  get(target, key2) {
    if (!target.exclude.has(key2))
      return target.props[key2];
  },
  set(target, key2) {
    return false_default && props_rest_readonly(`${target.name}.${String(key2)}`), !1;
  },
  getOwnPropertyDescriptor(target, key2) {
    if (!target.exclude.has(key2) && key2 in target.props)
      return {
        enumerable: !0,
        configurable: !0,
        value: target.props[key2]
      };
  },
  has(target, key2) {
    return target.exclude.has(key2) ? !1 : key2 in target.props;
  },
  ownKeys(target) {
    return Reflect.ownKeys(target.props).filter((key2) => !target.exclude.has(key2));
  }
};
// @__NO_SIDE_EFFECTS__
function rest_props(props, exclude, name) {
  return new Proxy(false_default ? { props, exclude, name } : { props, exclude }, rest_props_handler);
}
var legacy_rest_props_handler = {
  get(target, key2) {
    if (!target.exclude.includes(key2))
      return get(target.version), key2 in target.special ? target.special[key2]() : target.props[key2];
  },
  set(target, key2, value) {
    if (!(key2 in target.special)) {
      var previous_effect = active_effect;
      try {
        set_active_effect(target.parent_effect), target.special[key2] = prop(
          {
            get [key2]() {
              return target.props[key2];
            }
          },
          /** @type {string} */
          key2,
          4
        );
      } finally {
        set_active_effect(previous_effect);
      }
    }
    return target.special[key2](value), update(target.version), !0;
  },
  getOwnPropertyDescriptor(target, key2) {
    if (!target.exclude.includes(key2) && key2 in target.props)
      return {
        enumerable: !0,
        configurable: !0,
        value: target.props[key2]
      };
  },
  deleteProperty(target, key2) {
    return target.exclude.includes(key2) || (target.exclude.push(key2), update(target.version)), !0;
  },
  has(target, key2) {
    return target.exclude.includes(key2) ? !1 : key2 in target.props;
  },
  ownKeys(target) {
    return Reflect.ownKeys(target.props).filter((key2) => !target.exclude.includes(key2));
  }
};
function legacy_rest_props(props, exclude) {
  return new Proxy(
    {
      props,
      exclude,
      special: {},
      version: source(0),
      // TODO this is only necessary because we need to track component
      // destruction inside `prop`, because of `bind:this`, but it
      // seems likely that we can simplify `bind:this` instead
      parent_effect: (
        /** @type {Effect} */
        active_effect
      )
    },
    legacy_rest_props_handler
  );
}
var spread_props_handler = {
  get(target, key2) {
    let i = target.props.length;
    for (; i--; ) {
      let p = target.props[i];
      if (is_function(p) && (p = p()), typeof p == "object" && p !== null && key2 in p) return p[key2];
    }
  },
  set(target, key2, value) {
    let i = target.props.length;
    for (; i--; ) {
      let p = target.props[i];
      is_function(p) && (p = p());
      let desc = get_descriptor(p, key2);
      if (desc && desc.set)
        return desc.set(value), !0;
    }
    return !1;
  },
  getOwnPropertyDescriptor(target, key2) {
    let i = target.props.length;
    for (; i--; ) {
      let p = target.props[i];
      if (is_function(p) && (p = p()), typeof p == "object" && p !== null && key2 in p) {
        let descriptor = get_descriptor(p, key2);
        return descriptor && !descriptor.configurable && (descriptor.configurable = !0), descriptor;
      }
    }
  },
  has(target, key2) {
    if (key2 === STATE_SYMBOL || key2 === LEGACY_PROPS) return !1;
    for (let p of target.props)
      if (is_function(p) && (p = p()), p != null && key2 in p) return !0;
    return !1;
  },
  ownKeys(target) {
    let keys = [];
    for (let p of target.props)
      if (is_function(p) && (p = p()), !!p) {
        for (let key2 in p)
          keys.includes(key2) || keys.push(key2);
        for (let key2 of Object.getOwnPropertySymbols(p))
          keys.includes(key2) || keys.push(key2);
      }
    return keys;
  }
};
function spread_props(...props) {
  return new Proxy({ props }, spread_props_handler);
}
function prop(props, key2, flags2, fallback2) {
  var runes = !legacy_mode_flag || (flags2 & 2) !== 0, bindable = (flags2 & 8) !== 0, lazy = (flags2 & 16) !== 0, fallback_value = (
    /** @type {V} */
    fallback2
  ), fallback_dirty = !0, fallback_signal = (
    /** @type {Derived<V> | undefined} */
    void 0
  ), get_fallback = () => lazy && runes ? (fallback_signal ??= derived(
    /** @type {() => V} */
    fallback2
  ), get(fallback_signal)) : (fallback_dirty && (fallback_dirty = !1, fallback_value = lazy ? untrack(
    /** @type {() => V} */
    fallback2
  ) : (
    /** @type {V} */
    fallback2
  )), fallback_value);
  let setter;
  if (bindable) {
    var is_entry_props = STATE_SYMBOL in props || LEGACY_PROPS in props;
    setter = get_descriptor(props, key2)?.set ?? (is_entry_props && key2 in props ? (v) => props[key2] = v : void 0);
  }
  var initial_value, is_store_sub = !1;
  bindable ? [initial_value, is_store_sub] = capture_store_binding(() => (
    /** @type {V} */
    props[key2]
  )) : initial_value = /** @type {V} */
  props[key2], initial_value === void 0 && fallback2 !== void 0 && (initial_value = get_fallback(), setter && (runes && props_invalid_value(key2), setter(initial_value)));
  var getter;
  if (runes ? getter = () => {
    var value = (
      /** @type {V} */
      props[key2]
    );
    return value === void 0 ? get_fallback() : (fallback_dirty = !0, value);
  } : getter = () => {
    var value = (
      /** @type {V} */
      props[key2]
    );
    return value !== void 0 && (fallback_value = /** @type {V} */
    void 0), value === void 0 ? fallback_value : value;
  }, runes && (flags2 & 4) === 0)
    return getter;
  if (setter) {
    var legacy_parent = props.$$legacy;
    return (
      /** @type {() => V} */
      (function(value, mutation) {
        return arguments.length > 0 ? ((!runes || !mutation || legacy_parent || is_store_sub) && setter(mutation ? getter() : value), value) : getter();
      })
    );
  }
  var overridden = !1, d = ((flags2 & 1) !== 0 ? derived : derived_safe_equal)(() => (overridden = !1, getter()));
  false_default && (d.label = key2), bindable && get(d);
  var parent_effect = (
    /** @type {Effect} */
    active_effect
  );
  return (
    /** @type {() => V} */
    (function(value, mutation) {
      if (arguments.length > 0) {
        let new_value = mutation ? get(d) : runes && bindable ? proxy(value) : value;
        return set(d, new_value), overridden = !0, fallback_value !== void 0 && (fallback_value = new_value), value;
      }
      return is_destroying_effect && overridden || (parent_effect.f & 16384) !== 0 ? d.v : get(d);
    })
  );
}

// node_modules/svelte/src/internal/client/validate.js
function validate_binding(binding, blockers, get_object, get_property, line, column) {
  run_after_blockers(blockers, () => {
    var warned = !1, filename = dev_current_component_function?.[FILENAME];
    render_effect(() => {
      if (!warned) {
        var [object, is_store_sub] = capture_store_binding(get_object);
        if (!is_store_sub) {
          var property = get_property(), ran = !1, effect2 = render_effect(() => {
            ran || object[property];
          });
          if (ran = !0, effect2.edges === null) {
            var location = `${filename}:${line}:${column}`;
            binding_property_non_reactive(binding, location), warned = !0;
          }
        }
      }
    });
  });
}

// node_modules/svelte/src/legacy/legacy-client.js
function createClassComponent(options) {
  return new Svelte4Component(options);
}
var Svelte4Component = class {
  /** @type {any} */
  #events;
  /** @type {Record<string, any>} */
  #instance;
  /**
   * @param {ComponentConstructorOptions & {
   *  component: any;
   * }} options
   */
  constructor(options) {
    var sources = /* @__PURE__ */ new Map(), add_source = (key2, value) => {
      var s = mutable_source(value, !1, !1);
      return sources.set(key2, s), s;
    };
    let props = new Proxy(
      { ...options.props || {}, $$events: {} },
      {
        get(target, prop2) {
          return get(sources.get(prop2) ?? add_source(prop2, Reflect.get(target, prop2)));
        },
        has(target, prop2) {
          return prop2 === LEGACY_PROPS ? !0 : (get(sources.get(prop2) ?? add_source(prop2, Reflect.get(target, prop2))), Reflect.has(target, prop2));
        },
        set(target, prop2, value) {
          return set(sources.get(prop2) ?? add_source(prop2, value), value), Reflect.set(target, prop2, value);
        }
      }
    ), render_options = {
      target: options.target,
      anchor: options.anchor,
      props,
      context: options.context,
      intro: options.intro ?? !1,
      recover: options.recover,
      transformError: options.transformError
    };
    this.#instance = options.hydrate ? hydrate(options.component, render_options) : mount(options.component, render_options), !async_mode_flag && (!options?.props?.$$host || options.sync === !1) && flushSync(), this.#events = props.$$events;
    for (let key2 of Object.keys(this.#instance))
      key2 === "$set" || key2 === "$destroy" || key2 === "$on" || define_property(this, key2, {
        get() {
          return this.#instance[key2];
        },
        /** @param {any} value */
        set(value) {
          this.#instance[key2] = value;
        },
        enumerable: !0
      });
    this.#instance.$set = /** @param {Record<string, any>} next */
    (next2) => {
      Object.assign(props, next2);
    }, this.#instance.$destroy = () => {
      unmount(this.#instance);
    };
  }
  /** @param {Record<string, any>} props */
  $set(props) {
    this.#instance.$set(props);
  }
  /**
   * @param {string} event
   * @param {(...args: any[]) => any} callback
   * @returns {any}
   */
  $on(event2, callback) {
    this.#events[event2] = this.#events[event2] || [];
    let cb = (...args) => callback.call(this, ...args);
    return this.#events[event2].push(cb), () => {
      this.#events[event2] = this.#events[event2].filter(
        /** @param {any} fn */
        (fn) => fn !== cb
      );
    };
  }
  $destroy() {
    this.#instance.$destroy();
  }
};

// node_modules/svelte/src/internal/client/dom/elements/custom-element.js
var SvelteElement;
typeof HTMLElement == "function" && (SvelteElement = class extends HTMLElement {
  /** The Svelte component constructor */
  $$ctor;
  /** Slots */
  $$s;
  /** @type {any} The Svelte component instance */
  $$c;
  /** Whether or not the custom element is connected */
  $$cn = !1;
  /** @type {Record<string, any>} Component props data */
  $$d = {};
  /** `true` if currently in the process of reflecting component props back to attributes */
  $$r = !1;
  /** @type {Record<string, CustomElementPropDefinition>} Props definition (name, reflected, type etc) */
  $$p_d = {};
  /** @type {Record<string, EventListenerOrEventListenerObject[]>} Event listeners */
  $$l = {};
  /** @type {Map<EventListenerOrEventListenerObject, Function>} Event listener unsubscribe functions */
  $$l_u = /* @__PURE__ */ new Map();
  /** @type {any} The managed render effect for reflecting attributes */
  $$me;
  /** @type {ShadowRoot | null} The ShadowRoot of the custom element */
  $$shadowRoot = null;
  /**
   * @param {*} $$componentCtor
   * @param {*} $$slots
   * @param {ShadowRootInit | undefined} shadow_root_init
   */
  constructor($$componentCtor, $$slots, shadow_root_init) {
    super(), this.$$ctor = $$componentCtor, this.$$s = $$slots, shadow_root_init && (this.$$shadowRoot = this.attachShadow(shadow_root_init));
  }
  /**
   * @param {string} type
   * @param {EventListenerOrEventListenerObject} listener
   * @param {boolean | AddEventListenerOptions} [options]
   */
  addEventListener(type, listener, options) {
    if (this.$$l[type] = this.$$l[type] || [], this.$$l[type].push(listener), this.$$c) {
      let unsub = this.$$c.$on(type, listener);
      this.$$l_u.set(listener, unsub);
    }
    super.addEventListener(type, listener, options);
  }
  /**
   * @param {string} type
   * @param {EventListenerOrEventListenerObject} listener
   * @param {boolean | AddEventListenerOptions} [options]
   */
  removeEventListener(type, listener, options) {
    if (super.removeEventListener(type, listener, options), this.$$c) {
      let unsub = this.$$l_u.get(listener);
      unsub && (unsub(), this.$$l_u.delete(listener));
    }
  }
  async connectedCallback() {
    if (this.$$cn = !0, !this.$$c) {
      let create_slot = function(name) {
        return (anchor) => {
          let slot2 = create_element("slot");
          name !== "default" && (slot2.name = name), append(anchor, slot2);
        };
      };
      if (await Promise.resolve(), !this.$$cn || this.$$c)
        return;
      let $$slots = {}, existing_slots = get_custom_elements_slots(this);
      for (let name of this.$$s)
        name in existing_slots && (name === "default" && !this.$$d.children ? (this.$$d.children = create_slot(name), $$slots.default = !0) : $$slots[name] = create_slot(name));
      for (let attribute of this.attributes) {
        let name = this.$$g_p(attribute.name);
        name in this.$$d || (this.$$d[name] = get_custom_element_value(name, attribute.value, this.$$p_d, "toProp"));
      }
      for (let key2 in this.$$p_d)
        !(key2 in this.$$d) && this[key2] !== void 0 && (this.$$d[key2] = this[key2], delete this[key2]);
      this.$$c = createClassComponent({
        component: this.$$ctor,
        target: this.$$shadowRoot || this,
        props: {
          ...this.$$d,
          $$slots,
          $$host: this
        }
      }), this.$$me = effect_root(() => {
        render_effect(() => {
          this.$$r = !0;
          for (let key2 of object_keys(this.$$c)) {
            if (!this.$$p_d[key2]?.reflect) continue;
            this.$$d[key2] = this.$$c[key2];
            let attribute_value = get_custom_element_value(
              key2,
              this.$$d[key2],
              this.$$p_d,
              "toAttribute"
            );
            attribute_value == null ? this.removeAttribute(this.$$p_d[key2].attribute || key2) : this.setAttribute(this.$$p_d[key2].attribute || key2, attribute_value);
          }
          this.$$r = !1;
        });
      });
      for (let type in this.$$l)
        for (let listener of this.$$l[type]) {
          let unsub = this.$$c.$on(type, listener);
          this.$$l_u.set(listener, unsub);
        }
      this.$$l = {};
    }
  }
  // We don't need this when working within Svelte code, but for compatibility of people using this outside of Svelte
  // and setting attributes through setAttribute etc, this is helpful
  /**
   * @param {string} attr
   * @param {string} _oldValue
   * @param {string} newValue
   */
  attributeChangedCallback(attr2, _oldValue, newValue) {
    this.$$r || (attr2 = this.$$g_p(attr2), this.$$d[attr2] = get_custom_element_value(attr2, newValue, this.$$p_d, "toProp"), this.$$c?.$set({ [attr2]: this.$$d[attr2] }));
  }
  disconnectedCallback() {
    this.$$cn = !1, Promise.resolve().then(() => {
      !this.$$cn && this.$$c && (this.$$c.$destroy(), this.$$me(), this.$$c = void 0);
    });
  }
  /**
   * @param {string} attribute_name
   */
  $$g_p(attribute_name) {
    return object_keys(this.$$p_d).find(
      (key2) => this.$$p_d[key2].attribute === attribute_name || !this.$$p_d[key2].attribute && key2.toLowerCase() === attribute_name
    ) || attribute_name;
  }
});
function get_custom_element_value(prop2, value, props_definition, transform) {
  let type = props_definition[prop2]?.type;
  if (value = type === "Boolean" && typeof value != "boolean" ? value != null : value, !transform || !props_definition[prop2])
    return value;
  if (transform === "toAttribute")
    switch (type) {
      case "Object":
      case "Array":
        return value == null ? null : JSON.stringify(value);
      case "Boolean":
        return value ? "" : null;
      case "Number":
        return value ?? null;
      default:
        return value;
    }
  else
    switch (type) {
      case "Object":
      case "Array":
        return value && JSON.parse(value);
      case "Boolean":
        return value;
      // conversion already handled above
      case "Number":
        return value != null ? +value : value;
      default:
        return value;
    }
}
function get_custom_elements_slots(element2) {
  let result = {};
  return element2.childNodes.forEach((node) => {
    result[
      /** @type {Element} node */
      node.slot || "default"
    ] = !0;
  }), result;
}
function create_custom_element(Component, props_definition, slots, exports, shadow_root_init, extend) {
  let Class = class extends SvelteElement {
    constructor() {
      super(Component, slots, shadow_root_init), this.$$p_d = props_definition;
    }
    static get observedAttributes() {
      return object_keys(props_definition).map(
        (key2) => (props_definition[key2].attribute || key2).toLowerCase()
      );
    }
  };
  return object_keys(props_definition).forEach((prop2) => {
    define_property(Class.prototype, prop2, {
      get() {
        return this.$$c && prop2 in this.$$c ? this.$$c[prop2] : this.$$d[prop2];
      },
      set(value) {
        value = get_custom_element_value(prop2, value, props_definition), this.$$d[prop2] = value;
        var component2 = this.$$c;
        if (component2) {
          var setter = get_descriptor(component2, prop2)?.get;
          setter ? component2[prop2] = value : component2.$set({ [prop2]: value });
        }
      }
    });
  }), exports.forEach((property) => {
    define_property(Class.prototype, property, {
      get() {
        return this.$$c?.[property];
      }
    });
  }), extend && (Class = extend(Class)), Component.element = /** @type {any} */
  Class, Class;
}

// node_modules/svelte/src/internal/client/dev/console-log.js
function log_if_contains_state(method, ...objects) {
  return untrack(() => {
    try {
      let has_state = !1, transformed = [];
      for (let obj of objects)
        obj && typeof obj == "object" && STATE_SYMBOL in obj ? (transformed.push(snapshot(obj, !0)), has_state = !0) : transformed.push(obj);
      has_state && (console_log_state(method), console.log("%c[snapshot]", "color: grey", ...transformed));
    } catch {
    }
  }), objects;
}

// node_modules/svelte/src/internal/client/hydratable.js
function hydratable(key2, fn) {
  if (async_mode_flag || experimental_async_required("hydratable"), hydrating) {
    let store = window.__svelte?.h;
    if (store?.has(key2))
      return (
        /** @type {T} */
        store.get(key2)
      );
    false_default ? hydratable_missing_but_required(key2) : hydratable_missing_but_expected(key2);
  }
  return fn();
}

// node_modules/svelte/src/index-client.js
if (false_default) {
  let throw_rune_error = function(rune) {
    if (!(rune in globalThis)) {
      let value;
      Object.defineProperty(globalThis, rune, {
        configurable: !0,
        // eslint-disable-next-line getter-return
        get: () => {
          if (value !== void 0)
            return value;
          rune_outside_svelte(rune);
        },
        set: (v) => {
          value = v;
        }
      });
    }
  };
  throw_rune_error("$state"), throw_rune_error("$effect"), throw_rune_error("$derived"), throw_rune_error("$inspect"), throw_rune_error("$props"), throw_rune_error("$bindable");
}
function getAbortSignal() {
  return active_reaction === null && get_abort_signal_outside_reaction(), get_abort_signal(active_reaction);
}
function onMount(fn) {
  component_context === null && lifecycle_outside_component("onMount"), legacy_mode_flag && component_context.l !== null ? init_update_callbacks(component_context).m.push(fn) : user_effect(() => {
    let cleanup = untrack(fn);
    if (typeof cleanup == "function") return (
      /** @type {() => void} */
      cleanup
    );
  });
}
function onDestroy(fn) {
  component_context === null && lifecycle_outside_component("onDestroy"), onMount(() => () => untrack(fn));
}
function create_custom_event(type, detail, { bubbles = !1, cancelable = !1 } = {}) {
  return new CustomEvent(type, { detail, bubbles, cancelable });
}
function createEventDispatcher() {
  let active_component_context = component_context;
  return active_component_context === null && lifecycle_outside_component("createEventDispatcher"), (type, detail, options) => {
    let events = (
      /** @type {Record<string, Function | Function[]>} */
      active_component_context.s.$$events?.[
        /** @type {string} */
        type
      ]
    );
    if (events) {
      let callbacks = is_array(events) ? events.slice() : [events], event2 = create_custom_event(
        /** @type {string} */
        type,
        detail,
        options
      );
      for (let fn of callbacks)
        fn.call(active_component_context.x, event2);
      return !event2.defaultPrevented;
    }
    return !0;
  };
}
function beforeUpdate(fn) {
  component_context === null && lifecycle_outside_component("beforeUpdate"), component_context.l === null && lifecycle_legacy_only("beforeUpdate"), init_update_callbacks(component_context).b.push(fn);
}
function afterUpdate(fn) {
  component_context === null && lifecycle_outside_component("afterUpdate"), component_context.l === null && lifecycle_legacy_only("afterUpdate"), init_update_callbacks(component_context).a.push(fn);
}
function init_update_callbacks(context) {
  var l = (
    /** @type {ComponentContextLegacy} */
    context.l
  );
  return l.u ??= { a: [], b: [], m: [] };
}

// node_modules/svelte/src/attachments/index.js
function createAttachmentKey() {
  return Symbol(ATTACHMENT_KEY);
}
function fromAction(action2, fn = (
  /** @type {() => T} */
  noop
)) {
  return (element2) => {
    let { update: update2, destroy } = untrack(() => action2(element2, fn()) ?? {});
    if (update2) {
      var ran = !1;
      render_effect(() => {
        let arg = fn();
        ran && update2(arg);
      }), ran = !0;
    }
    destroy && teardown(destroy);
  };
}

export {
  createAttachmentKey,
  fromAction,
  assign,
  assign_async,
  cleanup_styles,
  add_locations,
  from_html,
  from_svg,
  from_mathml,
  from_tree,
  with_script,
  text,
  comment,
  append,
  props_id,
  boundary,
  pending,
  set_text,
  mount,
  hydrate,
  unmount,
  hmr,
  create_ownership_validator,
  check_target,
  legacy_api,
  inspect,
  async,
  validate_snippet_args,
  await_block,
  if_block,
  key,
  attr,
  clsx2 as clsx,
  css_props,
  index,
  each,
  html,
  slot,
  sanitize_slots,
  validate_void_dynamic_element,
  validate_dynamic_element_tag,
  validate_store,
  prevent_snippet_stringification,
  is_crossing,
  cross,
  snippet,
  wrap_snippet,
  renderer_snippet,
  validate_snippet_renderer,
  createRawSnippet,
  component,
  raf,
  loop,
  animation,
  transition,
  element,
  head,
  append_styles2 as append_styles,
  action,
  attach,
  set_class,
  set_style,
  set_selected,
  set_default_select_value,
  select_option,
  init_select,
  bind_select_value,
  CLASS,
  STYLE,
  remove_input_defaults,
  set_value,
  set_checked,
  set_default_checked,
  set_default_value,
  set_attribute2 as set_attribute,
  set_xlink_attribute,
  set_custom_element_data,
  attribute_effect,
  selectedcontent,
  customizable_select,
  bind_active_element,
  bind_value,
  bind_group,
  bind_checked,
  bind_files,
  bind_current_time,
  bind_buffered,
  bind_seekable,
  bind_played,
  bind_seeking,
  bind_ended,
  bind_ready_state,
  bind_playback_rate,
  bind_paused,
  bind_volume,
  bind_muted,
  bind_online,
  bind_prop,
  bind_resize_observer,
  bind_element_size,
  bind_this,
  bind_content_editable,
  bind_property,
  bind_focused,
  bind_window_scroll,
  bind_window_size,
  trusted,
  self,
  stopPropagation,
  once,
  stopImmediatePropagation,
  preventDefault,
  init,
  reactive_import,
  bubble_event,
  add_legacy_event_listener,
  update_legacy_props,
  store_get,
  store_unsub,
  store_set,
  invalidate_store,
  setup_stores,
  store_mutate,
  update_store,
  update_pre_store,
  mark_store_binding,
  update_prop,
  update_pre_prop,
  rest_props,
  legacy_rest_props,
  spread_props,
  prop,
  validate_binding,
  create_custom_element,
  log_if_contains_state,
  hydratable,
  getAbortSignal,
  onMount,
  onDestroy,
  createEventDispatcher,
  beforeUpdate,
  afterUpdate
};
