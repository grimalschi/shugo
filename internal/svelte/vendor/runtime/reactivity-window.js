import "./chunk-IKPOX6JB.js";
import {
  createSubscriber
} from "./chunk-D5SIUDWU.js";
import {
  on
} from "./chunk-FHKBVURM.js";
import "./chunk-SZUTS3VD.js";
import "./chunk-6ZQVU4YW.js";
import "./chunk-J4FMNBVD.js";
import "./chunk-UC3IQMGR.js";
import {
  get,
  set,
  source,
  tag
} from "./chunk-GFOPKZ2A.js";
import {
  false_default
} from "./chunk-VCLKK7XB.js";

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

// node_modules/svelte/src/reactivity/window/index.js
var scrollX = new ReactiveValue(
  false_default ? () => window.scrollX : () => {
  },
  (update) => on(window, "scroll", update)
), scrollY = new ReactiveValue(
  false_default ? () => window.scrollY : () => {
  },
  (update) => on(window, "scroll", update)
), innerWidth = new ReactiveValue(
  false_default ? () => window.innerWidth : () => {
  },
  (update) => on(window, "resize", update)
), innerHeight = new ReactiveValue(
  false_default ? () => window.innerHeight : () => {
  },
  (update) => on(window, "resize", update)
), outerWidth = new ReactiveValue(
  false_default ? () => window.outerWidth : () => {
  },
  (update) => on(window, "resize", update)
), outerHeight = new ReactiveValue(
  false_default ? () => window.outerHeight : () => {
  },
  (update) => on(window, "resize", update)
), screenLeft = new ReactiveValue(
  false_default ? () => window.screenLeft : () => {
  },
  (update) => {
    let value = window.screenLeft, frame = requestAnimationFrame(function check() {
      frame = requestAnimationFrame(check), value !== (value = window.screenLeft) && update();
    });
    return () => {
      cancelAnimationFrame(frame);
    };
  }
), screenTop = new ReactiveValue(
  false_default ? () => window.screenTop : () => {
  },
  (update) => {
    let value = window.screenTop, frame = requestAnimationFrame(function check() {
      frame = requestAnimationFrame(check), value !== (value = window.screenTop) && update();
    });
    return () => {
      cancelAnimationFrame(frame);
    };
  }
), online = new ReactiveValue(
  false_default ? () => navigator.onLine : () => {
  },
  (update) => {
    let unsub_online = on(window, "online", update), unsub_offline = on(window, "offline", update);
    return () => {
      unsub_online(), unsub_offline();
    };
  }
), devicePixelRatio = /* @__PURE__ */ new class {
  #dpr = source(false_default ? window.devicePixelRatio : void 0);
  #update() {
    let off = on(
      window.matchMedia(`(resolution: ${window.devicePixelRatio}dppx)`),
      "change",
      () => {
        set(this.#dpr, window.devicePixelRatio), off(), this.#update();
      }
    );
  }
  constructor() {
    false_default && this.#update(), false_default && tag(this.#dpr, "window.devicePixelRatio");
  }
  get current() {
    return get(this.#dpr), false_default ? window.devicePixelRatio : void 0;
  }
}();
export {
  devicePixelRatio,
  innerHeight,
  innerWidth,
  online,
  outerHeight,
  outerWidth,
  screenLeft,
  screenTop,
  scrollX,
  scrollY
};
