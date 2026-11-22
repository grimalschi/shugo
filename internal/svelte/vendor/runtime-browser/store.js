import {
  createSubscriber,
  derived,
  get,
  readable,
  readonly,
  writable
} from "./chunk-R2NOJYQV.js";
import {
  active_effect,
  active_reaction,
  effect_root,
  effect_tracking,
  render_effect,
  set_active_effect,
  set_active_reaction
} from "./chunk-SJURP6G5.js";
import "./chunk-CG7SAXEK.js";

// node_modules/svelte/src/store/index-client.js
function toStore(get2, set) {
  var effect = active_effect, reaction = active_reaction, init_value = get2();
  let store = writable(init_value, (set2) => {
    var ran = init_value !== get2(), teardown, previous_reaction = active_reaction, previous_effect = active_effect;
    set_active_reaction(reaction), set_active_effect(effect);
    try {
      teardown = effect_root(() => {
        render_effect(() => {
          let value = get2();
          ran && set2(value);
        });
      });
    } finally {
      set_active_reaction(previous_reaction), set_active_effect(previous_effect);
    }
    return ran = !0, teardown;
  });
  return set ? {
    set,
    update: (fn) => set(fn(get2())),
    subscribe: store.subscribe
  } : {
    subscribe: store.subscribe
  };
}
function fromStore(store) {
  let value, subscribe = createSubscriber((update) => {
    let ran = !1, unsubscribe = store.subscribe((v) => {
      value = v, ran && update();
    });
    return ran = !0, unsubscribe;
  });
  function current() {
    return effect_tracking() ? (subscribe(), value) : get(store);
  }
  return "set" in store ? {
    get current() {
      return current();
    },
    set current(v) {
      store.set(v);
    }
  } : {
    get current() {
      return current();
    }
  };
}
export {
  derived,
  fromStore,
  get,
  readable,
  readonly,
  toStore,
  writable
};
