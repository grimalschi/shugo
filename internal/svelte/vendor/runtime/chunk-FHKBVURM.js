import {
  is_passive_event
} from "./chunk-6ZQVU4YW.js";
import {
  active_effect,
  active_reaction,
  add_event_listener,
  array_from,
  current_renderer,
  define_property,
  queue_micro_task,
  remove_event_listener,
  set_active_effect,
  set_active_reaction,
  without_reactive_context
} from "./chunk-GFOPKZ2A.js";

// node_modules/svelte/src/internal/client/dom/elements/events.js
var event_symbol = /* @__PURE__ */ Symbol("events"), propagation_stopped_at = /* @__PURE__ */ Symbol("propagation stopped at"), all_registered_events = /* @__PURE__ */ new Set(), root_event_handles = /* @__PURE__ */ new Set();
function create_event(event_name, dom, handler, options = {}) {
  var is_custom_renderer = current_renderer != null;
  function target_handler(...args) {
    if (is_custom_renderer)
      return without_reactive_context(() => handler?.apply(
        this,
        /** @type {any} */
        args
      ));
    var event = (
      /** @type {Event} */
      args[0]
    );
    if (options.capture || handle_event_propagation.call(dom, event), options.capture || event[propagation_stopped_at] !== dom)
      return without_reactive_context(() => handler?.call(this, event));
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
function handle_event_propagation(event) {
  var handler_element = this, was_stopped = event.cancelBubble;
  if (!(was_stopped && event[propagation_stopped_at] === handler_element)) {
    was_stopped || delete event[propagation_stopped_at];
    var owner_document = (
      /** @type {Node} */
      handler_element.ownerDocument
    ), event_name = event.type, path = event.composedPath?.() || [], current_target = (
      /** @type {null | Element} */
      path[0] || event.target
    );
    last_propagated_event = event, last_propagated_event_clear_scheduled || (last_propagated_event_clear_scheduled = !0, setTimeout(() => {
      last_propagated_event_clear_scheduled = !1, last_propagated_event = null;
    }));
    var path_idx = 0, handled_at = last_propagated_event === event && event[event_symbol];
    if (handled_at) {
      var at_idx = path.indexOf(handled_at);
      if (at_idx !== -1 && (handler_element === document || handler_element === /** @type {any} */
      window)) {
        event[event_symbol] = handler_element;
        return;
      }
      var handler_idx = path.indexOf(handler_element);
      if (handler_idx === -1)
        return;
      at_idx <= handler_idx && (path_idx = at_idx);
    }
    if (current_target = /** @type {Element} */
    path[path_idx] || event.target, current_target !== handler_element) {
      define_property(event, "currentTarget", {
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
            var delegated = current_target[event_symbol]?.[event_name];
            delegated != null && (!/** @type {any} */
            current_target.disabled || // DOM could've been updated already by the time this is reached, so we check this as well
            // -> the target could not have been disabled because it emits the event in the first place
            event.target === current_target) && delegated.call(current_target, event);
          } catch (error) {
            throw_error ? other_errors.push(error) : throw_error = error;
          }
          if (event.cancelBubble) {
            was_stopped || (event[propagation_stopped_at] = handler_element);
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
        event[event_symbol] = handler_element, delete event.currentTarget, set_active_reaction(previous_reaction), set_active_effect(previous_effect);
      }
    }
  }
}

export {
  on,
  listen_to_delegated_events
};
