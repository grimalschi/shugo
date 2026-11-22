import {
  FILENAME,
  active_effect,
  active_reaction,
  add_event_listener,
  array_from,
  current_renderer,
  define_property,
  dispatch_event,
  hydrating,
  queue_micro_task,
  remove_attribute,
  remove_event_listener,
  set_active_effect,
  set_active_reaction,
  teardown,
  without_reactive_context
} from "./chunk-SJURP6G5.js";
import {
  event_handler_invalid
} from "./chunk-CG7SAXEK.js";

// node_modules/svelte/src/utils.js
var regex_return_characters = /\r/g;
function hash(str) {
  str = str.replace(regex_return_characters, "");
  let hash2 = 5381, i = str.length;
  for (; i--; ) hash2 = (hash2 << 5) - hash2 ^ str.charCodeAt(i);
  return (hash2 >>> 0).toString(36);
}
var VOID_ELEMENT_NAMES = [
  "area",
  "base",
  "br",
  "col",
  "command",
  "embed",
  "hr",
  "img",
  "input",
  "keygen",
  "link",
  "meta",
  "param",
  "source",
  "track",
  "wbr"
];
function is_void(name) {
  return VOID_ELEMENT_NAMES.includes(name) || name.toLowerCase() === "!doctype";
}
function is_capture_event(name) {
  return name.endsWith("capture") && name !== "gotpointercapture" && name !== "lostpointercapture";
}
var DELEGATED_EVENTS = [
  "beforeinput",
  "click",
  "change",
  "dblclick",
  "contextmenu",
  "focusin",
  "focusout",
  "input",
  "keydown",
  "keyup",
  "mousedown",
  "mousemove",
  "mouseout",
  "mouseover",
  "mouseup",
  "pointerdown",
  "pointermove",
  "pointerout",
  "pointerover",
  "pointerup",
  "touchend",
  "touchmove",
  "touchstart"
];
function can_delegate_event(event_name) {
  return DELEGATED_EVENTS.includes(event_name);
}
var DOM_BOOLEAN_ATTRIBUTES = [
  "allowfullscreen",
  "async",
  "autofocus",
  "autoplay",
  "checked",
  "controls",
  "default",
  "disabled",
  "formnovalidate",
  "indeterminate",
  "inert",
  "ismap",
  "loop",
  "multiple",
  "muted",
  "nomodule",
  "novalidate",
  "open",
  "playsinline",
  "readonly",
  "required",
  "reversed",
  "seamless",
  "selected",
  "webkitdirectory",
  "defer",
  "disablepictureinpicture",
  "disableremoteplayback"
];
var ATTRIBUTE_ALIASES = {
  // no `class: 'className'` because we handle that separately
  formnovalidate: "formNoValidate",
  ismap: "isMap",
  nomodule: "noModule",
  playsinline: "playsInline",
  readonly: "readOnly",
  defaultvalue: "defaultValue",
  defaultchecked: "defaultChecked",
  srcobject: "srcObject",
  novalidate: "noValidate",
  allowfullscreen: "allowFullscreen",
  disablepictureinpicture: "disablePictureInPicture",
  disableremoteplayback: "disableRemotePlayback"
};
function normalize_attribute(name) {
  return name = name.toLowerCase(), ATTRIBUTE_ALIASES[name] ?? name;
}
var DOM_PROPERTIES = [
  ...DOM_BOOLEAN_ATTRIBUTES,
  "formNoValidate",
  "isMap",
  "noModule",
  "playsInline",
  "readOnly",
  "value",
  "volume",
  "defaultValue",
  "defaultChecked",
  "srcObject",
  "noValidate",
  "allowFullscreen",
  "disablePictureInPicture",
  "disableRemotePlayback"
];
var PASSIVE_EVENTS = ["touchstart", "touchmove"];
function is_passive_event(name) {
  return PASSIVE_EVENTS.includes(name);
}
var STATE_CREATION_RUNES = (
  /** @type {const} */
  [
    "$state",
    "$state.raw",
    "$derived",
    "$derived.by"
  ]
), RUNES = (
  /** @type {const} */
  [
    ...STATE_CREATION_RUNES,
    "$state.eager",
    "$state.snapshot",
    "$props",
    "$props.id",
    "$bindable",
    "$effect",
    "$effect.pre",
    "$effect.tracking",
    "$effect.root",
    "$effect.pending",
    "$inspect",
    "$inspect().with",
    "$inspect.trace",
    "$host"
  ]
);
var RAW_TEXT_ELEMENTS = (
  /** @type {const} */
  ["textarea", "script", "style", "title"]
);
function is_raw_text_element(name) {
  return RAW_TEXT_ELEMENTS.includes(
    /** @type {typeof RAW_TEXT_ELEMENTS[number]} */
    name
  );
}
function sanitize_location(location) {
  return (
    /** @type {T} */
    location?.replace(/\//g, "/​")
  );
}

// node_modules/svelte/src/internal/client/dom/elements/events.js
var event_symbol = /* @__PURE__ */ Symbol("events"), propagation_stopped_at = /* @__PURE__ */ Symbol("propagation stopped at"), all_registered_events = /* @__PURE__ */ new Set(), root_event_handles = /* @__PURE__ */ new Set();
function replay_events(dom) {
  if (!hydrating) return;
  remove_attribute(dom, "onload"), remove_attribute(dom, "onerror");
  let event2 = dom.__e;
  event2 !== void 0 && (dom.__e = void 0, queueMicrotask(() => {
    dom.isConnected && dispatch_event(dom, event2);
  }));
}
function create_event(event_name, dom, handler, options = {}) {
  var is_custom_renderer = current_renderer != null;
  function target_handler(...args) {
    if (is_custom_renderer)
      return without_reactive_context(() => handler?.apply(
        this,
        /** @type {any} */
        args
      ));
    var event2 = (
      /** @type {Event} */
      args[0]
    );
    if (options.capture || handle_event_propagation.call(dom, event2), options.capture || event2[propagation_stopped_at] !== dom)
      return without_reactive_context(() => handler?.call(this, event2));
  }
  return !is_custom_renderer && (event_name.startsWith("pointer") || event_name.startsWith("touch") || event_name === "wheel") ? (target_handler.__removed = !1, queue_micro_task(() => {
    target_handler.__removed || add_event_listener(dom, event_name, target_handler, options);
  })) : add_event_listener(dom, event_name, target_handler, options), target_handler;
}
function on(element, type, handler, options = {}) {
  var target_handler = create_event(type, element, handler, options);
  return () => {
    target_handler.__removed = !0, remove_event_listener(element, type, target_handler, options);
  };
}
function event(event_name, dom, handler, capture, passive) {
  var options = { capture, passive }, target_handler = create_event(event_name, dom, handler, options);
  // if there's a renderer we will never add to body, window or document.
  current_renderer == null && (dom === document.body || // @ts-ignore
  dom === window || // @ts-ignore
  dom === document || // Firefox has quirky behavior, it can happen that we still get "canplay" events when the element is already removed
  dom instanceof HTMLMediaElement) && teardown(() => {
    target_handler.__removed = !0, remove_event_listener(dom, event_name, target_handler, options);
  });
}
function delegated(event_name, element, handler) {
  (element[event_symbol] ??= {})[event_name] = handler;
}
function delegate(events) {
  for (var i = 0; i < events.length; i++)
    all_registered_events.add(events[i]);
  for (var fn of root_event_handles)
    fn(events);
}
var delegated_listeners = /* @__PURE__ */ new Map();
function listen_to_delegated_events(target) {
  var registered_events = /* @__PURE__ */ new Set(), event_handle = (events) => {
    for (var i = 0; i < events.length; i++) {
      var event_name = events[i];
      if (!registered_events.has(event_name)) {
        registered_events.add(event_name);
        var passive = is_passive_event(event_name);
        for (let node of [target, document]) {
          var counts = delegated_listeners.get(node);
          counts === void 0 && (counts = /* @__PURE__ */ new Map(), delegated_listeners.set(node, counts));
          var count = counts.get(event_name);
          count === void 0 ? (node.addEventListener(event_name, handle_event_propagation, { passive }), counts.set(event_name, 1)) : counts.set(event_name, count + 1);
        }
      }
    }
  };
  return event_handle(array_from(all_registered_events)), root_event_handles.add(event_handle), () => {
    for (var event_name of registered_events)
      for (let node of [target, document]) {
        var counts = (
          /** @type {Map<string, number>} */
          delegated_listeners.get(node)
        ), count = (
          /** @type {number} */
          counts.get(event_name)
        );
        --count == 0 ? (node.removeEventListener(event_name, handle_event_propagation), counts.delete(event_name), counts.size === 0 && delegated_listeners.delete(node)) : counts.set(event_name, count);
      }
    root_event_handles.delete(event_handle);
  };
}
var last_propagated_event = null, last_propagated_event_clear_scheduled = !1;
function handle_event_propagation(event2) {
  var handler_element = this, was_stopped = event2.cancelBubble;
  if (!(was_stopped && event2[propagation_stopped_at] === handler_element)) {
    was_stopped || delete event2[propagation_stopped_at];
    var owner_document = (
      /** @type {Node} */
      handler_element.ownerDocument
    ), event_name = event2.type, path = event2.composedPath?.() || [], current_target = (
      /** @type {null | Element} */
      path[0] || event2.target
    );
    last_propagated_event = event2, last_propagated_event_clear_scheduled || (last_propagated_event_clear_scheduled = !0, setTimeout(() => {
      last_propagated_event_clear_scheduled = !1, last_propagated_event = null;
    }));
    var path_idx = 0, handled_at = last_propagated_event === event2 && event2[event_symbol];
    if (handled_at) {
      var at_idx = path.indexOf(handled_at);
      if (at_idx !== -1 && (handler_element === document || handler_element === /** @type {any} */
      window)) {
        event2[event_symbol] = handler_element;
        return;
      }
      var handler_idx = path.indexOf(handler_element);
      if (handler_idx === -1)
        return;
      at_idx <= handler_idx && (path_idx = at_idx);
    }
    if (current_target = /** @type {Element} */
    path[path_idx] || event2.target, current_target !== handler_element) {
      define_property(event2, "currentTarget", {
        configurable: !0,
        get() {
          return current_target || owner_document;
        }
      });
      var previous_reaction = active_reaction, previous_effect = active_effect;
      set_active_reaction(null), set_active_effect(null);
      try {
        for (var throw_error, other_errors = []; current_target !== null && current_target !== handler_element; ) {
          try {
            var delegated2 = current_target[event_symbol]?.[event_name];
            delegated2 != null && (!/** @type {any} */
            current_target.disabled || // DOM could've been updated already by the time this is reached, so we check this as well
            // -> the target could not have been disabled because it emits the event in the first place
            event2.target === current_target) && delegated2.call(current_target, event2);
          } catch (error) {
            throw_error ? other_errors.push(error) : throw_error = error;
          }
          if (event2.cancelBubble) {
            was_stopped || (event2[propagation_stopped_at] = handler_element);
            break;
          }
          path_idx++, current_target = path_idx < path.length ? (
            /** @type {Element} */
            path[path_idx]
          ) : null;
        }
        if (throw_error) {
          for (let error of other_errors)
            queueMicrotask(() => {
              throw error;
            });
          throw throw_error;
        }
      } finally {
        event2[event_symbol] = handler_element, delete event2.currentTarget, set_active_reaction(previous_reaction), set_active_effect(previous_effect);
      }
    }
  }
}
function apply(thunk, element, args, component, loc, has_side_effects = !1, remove_parens = !1) {
  let handler, error;
  try {
    handler = thunk();
  } catch (e) {
    error = e;
  }
  if (typeof handler != "function" && (has_side_effects || handler != null || error)) {
    let filename = component?.[FILENAME], location = loc ? ` at ${filename}:${loc[0]}:${loc[1]}` : ` in ${filename}`, phase = args[0]?.eventPhase < Event.BUBBLING_PHASE ? "capture" : "", description = `\`${args[0]?.type + phase}\` handler${location}`;
    if (event_handler_invalid(description, remove_parens ? "remove the trailing `()`" : "add a leading `() =>`"), error)
      throw error;
  }
  handler?.apply(element, args);
}

export {
  hash,
  is_void,
  is_capture_event,
  can_delegate_event,
  normalize_attribute,
  is_raw_text_element,
  sanitize_location,
  replay_events,
  create_event,
  on,
  event,
  delegated,
  delegate,
  listen_to_delegated_events,
  apply
};
