import {
  listen_to_delegated_events
} from "./chunk-FHKBVURM.js";
import {
  ATTACHMENT_KEY,
  Batch,
  COMMENT_NODE,
  HYDRATION_ERROR,
  LEGACY_PROPS,
  active_effect,
  active_reaction,
  append_child,
  async_mode_flag,
  block,
  branch,
  clear_text_content,
  component_context,
  component_root,
  create_element,
  create_fragment,
  create_text,
  current_batch,
  custom_renderers_flag,
  define_property,
  destroy_effect,
  effect_root,
  effect_tracking,
  flushSync,
  get,
  get_first_child,
  get_next_sibling,
  get_node_value,
  get_parent_node,
  hydrate_next,
  hydrate_node,
  hydrating,
  hydration_failed,
  increment,
  insert_before,
  internal_set,
  invoke_error_boundary,
  mark_as_component,
  move_effect,
  mutable_source,
  next,
  node_type,
  noop,
  object_keys,
  pause_effect,
  pop,
  pop_renderer,
  push,
  push_renderer,
  queue_micro_task,
  read_hydration_instruction,
  remove_child,
  render_effect,
  rune_outside_svelte,
  set,
  set_active_effect,
  set_active_reaction,
  set_component_context,
  set_hydrate_node,
  set_hydrating,
  set_signal_status,
  skip_nodes,
  source,
  svelte_boundary_reset_onerror,
  tag,
  teardown,
  untrack
} from "./chunk-GFOPKZ2A.js";
import {
  false_default,
  hydration_mismatch,
  lifecycle_double_unmount,
  svelte_boundary_reset_noop
} from "./chunk-VCLKK7XB.js";

// node_modules/svelte/src/internal/client/dom/reconciler.js
var policy = (
  // We gotta write it like this because after downleveling the pure comment may end up in the wrong location
  globalThis?.window?.trustedTypes && /* @__PURE__ */ globalThis.window.trustedTypes.createPolicy("svelte-trusted-html", {
    /** @param {string} html */
    createHTML: (html2) => html2
  })
);

// node_modules/svelte/src/internal/client/dom/template.js
function assign_nodes(start, end) {
  var effect2 = (
    /** @type {Effect} */
    active_effect
  );
  effect2.nodes === null && (effect2.nodes = { start, end, a: null, t: null });
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

// node_modules/svelte/src/reactivity/create-subscriber.js
function createSubscriber(start) {
  let subscribers = 0, version = source(0), stop;
  return false_default && tag(version, "createSubscriber version"), () => {
    effect_tracking() && (get(version), render_effect(() => (subscribers === 0 && (stop = untrack(() => start(() => increment(version)))), subscribers += 1, () => {
      queue_micro_task(() => {
        subscribers -= 1, subscribers === 0 && (stop?.(), stop = void 0, increment(version));
      });
    })));
  };
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
    let pending2 = this.#props.pending;
    pending2 && (this.is_pending = !0, this.#pending_effect = branch(() => pending2(this.#anchor)), queue_micro_task(() => {
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
        let pending2 = this.#props.pending;
        if (pending2) {
          var fragment = this.#offscreen_fragment = create_fragment();
          move_effect(this.#main_effect, fragment), this.#pending_effect = branch(() => pending2(this.#anchor));
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
    for (let [owner, pending2] of this.#pending)
      if (!owner.is_fork) {
        for (let effect2 of pending2.keys())
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
    var pending2 = this.#pending.get(batch);
    pending2 === void 0 && this.#pending.set(batch, pending2 = /* @__PURE__ */ new Map());
    var count = pending2.get(effect2) ?? 0;
    pending2.set(effect2, count + 1);
  }
  /**
   * @param {Effect} effect
   * @param {Batch} batch
   */
  decrement(effect2, batch) {
    var pending2 = (
      /** @type {Map<Effect, number>} */
      this.#pending.get(batch)
    ), count = (
      /** @type {number} */
      pending2.get(effect2)
    );
    count === 1 ? (pending2.delete(effect2), pending2.size === 0 && this.#pending.delete(batch), queue_micro_task(() => {
      this.is_pending && !this.#is_destroyed() && Batch.ensure().schedule(this.#effect, 4096);
    })) : pending2.set(effect2, count - 1);
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

// node_modules/svelte/src/internal/client/render.js
var should_intro = !0;
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

// node_modules/svelte/src/internal/client/timing.js
var now = false_default ? () => performance.now() : () => Date.now(), raf = {
  // don't access requestAnimationFrame eagerly outside method
  // this allows basic testing of user code without JSDOM
  // bunder will eval and remove ternary when the user's app is built
  tick: (
    /** @param {any} _ */
    (_) => (false_default ? requestAnimationFrame : noop)(_)
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
  createSubscriber,
  createAttachmentKey,
  fromAction,
  raf,
  loop
};
