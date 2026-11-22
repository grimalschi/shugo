import {
  createSubscriber
} from "./chunk-R2NOJYQV.js";

// node_modules/svelte/src/reactivity/reactive-value.js
var ReactiveValue = class {
  #fn;
  #subscribe;
  /**
   *
   * @param {() => T} fn
   * @param {(update: () => void) => void} onsubscribe
   */
  constructor(fn, onsubscribe) {
    this.#fn = fn, this.#subscribe = createSubscriber(onsubscribe);
  }
  get current() {
    return this.#subscribe(), this.#fn();
  }
};

export {
  ReactiveValue
};
