import {
  ReactiveValue
} from "./chunk-W2HXMHQ3.js";
import "./chunk-IKPOX6JB.js";
import "./chunk-YFE7YRUS.js";
import {
  on
} from "./chunk-ZMX3RH2Y.js";
import "./chunk-R2NOJYQV.js";
import {
  get,
  set,
  source,
  tag
} from "./chunk-SJURP6G5.js";
import {
  false_default,
  true_default
} from "./chunk-CG7SAXEK.js";

// node_modules/svelte/src/reactivity/window/index.js
var scrollX = new ReactiveValue(
  true_default ? () => window.scrollX : () => {
  },
  (update) => on(window, "scroll", update)
), scrollY = new ReactiveValue(
  true_default ? () => window.scrollY : () => {
  },
  (update) => on(window, "scroll", update)
), innerWidth = new ReactiveValue(
  true_default ? () => window.innerWidth : () => {
  },
  (update) => on(window, "resize", update)
), innerHeight = new ReactiveValue(
  true_default ? () => window.innerHeight : () => {
  },
  (update) => on(window, "resize", update)
), outerWidth = new ReactiveValue(
  true_default ? () => window.outerWidth : () => {
  },
  (update) => on(window, "resize", update)
), outerHeight = new ReactiveValue(
  true_default ? () => window.outerHeight : () => {
  },
  (update) => on(window, "resize", update)
), screenLeft = new ReactiveValue(
  true_default ? () => window.screenLeft : () => {
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
  true_default ? () => window.screenTop : () => {
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
  true_default ? () => navigator.onLine : () => {
  },
  (update) => {
    let unsub_online = on(window, "online", update), unsub_offline = on(window, "offline", update);
    return () => {
      unsub_online(), unsub_offline();
    };
  }
), devicePixelRatio = /* @__PURE__ */ new class {
  #dpr = source(true_default ? window.devicePixelRatio : void 0);
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
    true_default && this.#update(), false_default && tag(this.#dpr, "window.devicePixelRatio");
  }
  get current() {
    return get(this.#dpr), true_default ? window.devicePixelRatio : void 0;
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
