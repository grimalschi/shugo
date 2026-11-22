import {
  derived,
  get,
  readable,
  readonly,
  writable
} from "./chunk-J4FMNBVD.js";
import "./chunk-UC3IQMGR.js";
import "./chunk-GFOPKZ2A.js";
import "./chunk-VCLKK7XB.js";

// node_modules/svelte/src/store/index-server.js
function toStore(get2, set) {
  let store = writable(get2());
  return set ? {
    set,
    update: (fn) => set(fn(get2())),
    subscribe: store.subscribe
  } : {
    subscribe: store.subscribe
  };
}
function fromStore(store) {
  return "set" in store ? {
    get current() {
      return get(store);
    },
    set current(v) {
      store.set(v);
    }
  } : {
    get current() {
      return get(store);
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
