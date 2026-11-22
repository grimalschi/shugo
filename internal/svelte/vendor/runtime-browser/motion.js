import {
  MediaQuery
} from "./chunk-IQJ2MIWA.js";
import "./chunk-W2HXMHQ3.js";
import "./chunk-IKPOX6JB.js";
import {
  loop,
  raf
} from "./chunk-YFE7YRUS.js";
import "./chunk-ZMX3RH2Y.js";
import {
  writable
} from "./chunk-R2NOJYQV.js";
import {
  deferred,
  get,
  noop,
  render_effect,
  set,
  state,
  tag
} from "./chunk-SJURP6G5.js";
import {
  linear
} from "./chunk-Q4LEMBIN.js";
import {
  false_default
} from "./chunk-CG7SAXEK.js";

// node_modules/svelte/src/motion/utils.js
function is_date(obj) {
  return Object.prototype.toString.call(obj) === "[object Date]";
}

// node_modules/svelte/src/motion/spring.js
function tick_spring(ctx, last_value, current_value, target_value) {
  if (typeof current_value == "number" || is_date(current_value)) {
    let delta = target_value - current_value, velocity = (current_value - last_value) / (ctx.dt || 1 / 60), spring2 = ctx.opts.stiffness * delta, damper = ctx.opts.damping * velocity, acceleration = (spring2 - damper) * ctx.inv_mass, d = (velocity + acceleration) * ctx.dt;
    return Math.abs(d) < ctx.opts.precision && Math.abs(delta) < ctx.opts.precision ? target_value : (ctx.settled = !1, is_date(current_value) ? new Date(current_value.getTime() + d) : current_value + d);
  } else {
    if (Array.isArray(current_value))
      return current_value.map(
        (_, i) => (
          // @ts-ignore
          tick_spring(ctx, last_value[i], current_value[i], target_value[i])
        )
      );
    if (typeof current_value == "object") {
      let next_value = {};
      for (let k in current_value)
        next_value[k] = tick_spring(ctx, last_value[k], current_value[k], target_value[k]);
      return next_value;
    } else
      throw new Error(`Cannot spring ${typeof current_value} values`);
  }
}
function spring(value, opts = {}) {
  let store = writable(value), { stiffness = 0.15, damping = 0.8, precision = 0.01 } = opts, last_time, task, current_token, last_value = (
    /** @type {T} */
    value
  ), target_value = (
    /** @type {T | undefined} */
    value
  ), inv_mass = 1, inv_mass_recovery_rate = 0, cancel_task = !1;
  function set2(new_value, opts2 = {}) {
    target_value = new_value;
    let token = current_token = {};
    return value == null || opts2.hard || spring2.stiffness >= 1 && spring2.damping >= 1 ? (cancel_task = !0, last_time = raf.now(), last_value = new_value, store.set(value = target_value), Promise.resolve()) : (opts2.soft && (inv_mass_recovery_rate = 1 / ((opts2.soft === !0 ? 0.5 : +opts2.soft) * 60), inv_mass = 0), task || (last_time = raf.now(), cancel_task = !1, task = loop((now) => {
      if (cancel_task)
        return cancel_task = !1, task = null, !1;
      inv_mass = Math.min(inv_mass + inv_mass_recovery_rate, 1);
      let elapsed = Math.min(now - last_time, 1e3 / 30), ctx = {
        inv_mass,
        opts: spring2,
        settled: !0,
        dt: elapsed * 60 / 1e3
      }, next_value = tick_spring(ctx, last_value, value, target_value);
      return last_time = now, last_value = /** @type {T} */
      value, store.set(value = /** @type {T} */
      next_value), ctx.settled && (task = null), !ctx.settled;
    })), new Promise((fulfil) => {
      task.promise.then(() => {
        token === current_token && fulfil();
      });
    }));
  }
  let spring2 = {
    set: set2,
    update: (fn, opts2) => set2(fn(
      /** @type {T} */
      target_value,
      /** @type {T} */
      value
    ), opts2),
    subscribe: store.subscribe,
    stiffness,
    damping,
    precision
  };
  return spring2;
}
var Spring = class _Spring {
  #stiffness = state(0.15);
  #damping = state(0.8);
  #precision = state(0.01);
  #current;
  #target;
  #last_value = (
    /** @type {T} */
    void 0
  );
  #last_time = 0;
  #inverse_mass = 1;
  #momentum = 0;
  /** @type {import('../internal/client/types').Task | null} */
  #task = null;
  /** @type {ReturnType<typeof deferred> | null} */
  #deferred = null;
  /**
   * @param {T} value
   * @param {SpringOptions} [options]
   */
  constructor(value, options = {}) {
    this.#current = false_default ? tag(state(value), "Spring.current") : state(value), this.#target = false_default ? tag(state(value), "Spring.target") : state(value), typeof options.stiffness == "number" && (this.#stiffness.v = clamp(options.stiffness, 0, 1)), typeof options.damping == "number" && (this.#damping.v = clamp(options.damping, 0, 1)), typeof options.precision == "number" && (this.#precision.v = options.precision), false_default && (tag(this.#stiffness, "Spring.stiffness"), tag(this.#damping, "Spring.damping"), tag(this.#precision, "Spring.precision"), tag(this.#current, "Spring.current"), tag(this.#target, "Spring.target"));
  }
  /**
   * Create a spring whose value is bound to the return value of `fn`. This must be called
   * inside an effect root (for example, during component initialisation).
   *
   * ```svelte
   * <script>
   * 	import { Spring } from 'svelte/motion';
   *
   * 	let { number } = $props();
   *
   * 	const spring = Spring.of(() => number);
   * <\/script>
   * ```
   * @template U
   * @param {() => U} fn
   * @param {SpringOptions} [options]
   */
  static of(fn, options) {
    let spring2 = new _Spring(fn(), options);
    return render_effect(() => {
      spring2.set(fn());
    }), spring2;
  }
  /** @param {T} value */
  #update(value) {
    if (set(this.#target, value), this.#current.v ??= value, this.#last_value ??= this.#current.v, !this.#task) {
      this.#last_time = raf.now();
      var inv_mass_recovery_rate = 1e3 / (this.#momentum * 60);
      this.#task ??= loop((now) => {
        this.#inverse_mass = Math.min(this.#inverse_mass + inv_mass_recovery_rate, 1);
        let elapsed = Math.min(now - this.#last_time, 1e3 / 30), ctx = {
          inv_mass: this.#inverse_mass,
          opts: {
            stiffness: this.#stiffness.v,
            damping: this.#damping.v,
            precision: this.#precision.v
          },
          settled: !0,
          dt: elapsed * 60 / 1e3
        };
        var next = tick_spring(ctx, this.#last_value, this.#current.v, this.#target.v);
        return this.#last_value = this.#current.v, this.#last_time = now, set(this.#current, next), ctx.settled && (this.#task = null), !ctx.settled;
      });
    }
    return this.#task.promise;
  }
  /**
   * Sets `spring.target` to `value` and returns a `Promise` that resolves if and when `spring.current` catches up to it.
   *
   * If `options.instant` is `true`, `spring.current` immediately matches `spring.target`.
   *
   * If `options.preserveMomentum` is provided, the spring will continue on its current trajectory for
   * the specified number of milliseconds. This is useful for things like 'fling' gestures.
   *
   * @param {T} value
   * @param {SpringUpdateOptions} [options]
   */
  set(value, options) {
    if (this.#deferred?.reject(new Error("Aborted")), options?.instant || this.#current.v === void 0)
      return this.#task?.abort(), this.#task = null, set(this.#current, set(this.#target, value)), this.#last_value = value, Promise.resolve();
    options?.preserveMomentum && (this.#inverse_mass = 0, this.#momentum = options.preserveMomentum);
    var d = this.#deferred = deferred();
    return d.promise.catch(noop), this.#update(value).then(() => {
      d === this.#deferred && d.resolve(void 0);
    }), d.promise;
  }
  get current() {
    return get(this.#current);
  }
  get damping() {
    return get(this.#damping);
  }
  set damping(v) {
    set(this.#damping, clamp(v, 0, 1));
  }
  get precision() {
    return get(this.#precision);
  }
  set precision(v) {
    set(this.#precision, v);
  }
  get stiffness() {
    return get(this.#stiffness);
  }
  set stiffness(v) {
    set(this.#stiffness, clamp(v, 0, 1));
  }
  get target() {
    return get(this.#target);
  }
  set target(v) {
    this.set(v);
  }
};
function clamp(n, min, max) {
  return Math.max(min, Math.min(max, n));
}

// node_modules/svelte/src/motion/tweened.js
function get_interpolator(a, b) {
  if (a === b || a !== a) return () => a;
  let type = typeof a;
  if (type !== typeof b || Array.isArray(a) !== Array.isArray(b))
    throw new Error("Cannot interpolate values of different type");
  if (Array.isArray(a)) {
    let arr = (
      /** @type {Array<any>} */
      b.map((bi, i) => get_interpolator(
        /** @type {Array<any>} */
        a[i],
        bi
      ))
    );
    return (t) => arr.map((fn) => fn(t));
  }
  if (type === "object") {
    if (!a || !b)
      throw new Error("Object cannot be null");
    if (is_date(a) && is_date(b)) {
      let an = a.getTime(), delta = b.getTime() - an;
      return (t) => new Date(an + t * delta);
    }
    let keys = Object.keys(b), interpolators = {};
    return keys.forEach((key) => {
      interpolators[key] = get_interpolator(a[key], b[key]);
    }), (t) => {
      let result = {};
      return keys.forEach((key) => {
        result[key] = interpolators[key](t);
      }), result;
    };
  }
  if (type === "number") {
    let delta = (
      /** @type {number} */
      b - /** @type {number} */
      a
    );
    return (t) => a + t * delta;
  }
  return () => b;
}
function tweened(value, defaults = {}) {
  let store = writable(value), task, target_value = value;
  function set2(new_value, opts) {
    if (target_value = new_value, value == null)
      return store.set(value = new_value), Promise.resolve();
    let previous_task = task, started = !1, {
      delay = 0,
      duration = 400,
      easing = linear,
      interpolate = get_interpolator
    } = { ...defaults, ...opts };
    if (duration === 0)
      return previous_task && (previous_task.abort(), previous_task = null), store.set(value = target_value), Promise.resolve();
    let start = raf.now() + delay, fn;
    return task = loop((now) => {
      if (now < start) return !0;
      started || (fn = interpolate(
        /** @type {any} */
        value,
        new_value
      ), typeof duration == "function" && (duration = duration(
        /** @type {any} */
        value,
        new_value
      )), started = !0), previous_task && (previous_task.abort(), previous_task = null);
      let elapsed = now - start;
      return elapsed > /** @type {number} */
      duration ? (store.set(value = new_value), !1) : (store.set(value = fn(easing(elapsed / duration))), !0);
    }), task.promise;
  }
  return {
    set: set2,
    update: (fn, opts) => set2(fn(
      /** @type {any} */
      target_value,
      /** @type {any} */
      value
    ), opts),
    subscribe: store.subscribe
  };
}
var Tween = class _Tween {
  #current;
  #target;
  /** @type {TweenOptions<T>} */
  #defaults;
  /** @type {import('../internal/client/types').Task | null} */
  #task = null;
  /**
   * @param {T} value
   * @param {TweenOptions<T>} options
   */
  constructor(value, options = {}) {
    this.#current = state(value), this.#target = state(value), this.#defaults = options, false_default && (tag(this.#current, "Tween.current"), tag(this.#target, "Tween.target"));
  }
  /**
   * Create a tween whose value is bound to the return value of `fn`. This must be called
   * inside an effect root (for example, during component initialisation).
   *
   * ```svelte
   * <script>
   * 	import { Tween } from 'svelte/motion';
   *
   * 	let { number } = $props();
   *
   * 	const tween = Tween.of(() => number);
   * <\/script>
   * ```
   * @template U
   * @param {() => U} fn
   * @param {TweenOptions<U>} [options]
   */
  static of(fn, options) {
    let tween = new _Tween(fn(), options);
    return render_effect(() => {
      tween.set(fn());
    }), tween;
  }
  /**
   * Sets `tween.target` to `value` and returns a `Promise` that resolves if and when `tween.current` catches up to it.
   *
   * If `options` are provided, they will override the tween's defaults.
   * @param {T} value
   * @param {TweenOptions<T>} [options]
   * @returns
   */
  set(value, options) {
    set(this.#target, value);
    let {
      delay = 0,
      duration = 400,
      easing = linear,
      interpolate = get_interpolator
    } = { ...this.#defaults, ...options };
    if (duration === 0)
      return this.#task?.abort(), set(this.#current, value), Promise.resolve();
    let start = raf.now() + delay, fn, started = !1, previous_task = this.#task;
    return this.#task = loop((now) => {
      if (now < start)
        return !0;
      if (!started) {
        started = !0;
        let prev = this.#current.v;
        fn = interpolate(prev, value), typeof duration == "function" && (duration = duration(prev, value)), previous_task?.abort(), previous_task = null;
      }
      let elapsed = now - start;
      return elapsed > /** @type {number} */
      duration ? (set(this.#current, value), !1) : (set(this.#current, fn(easing(elapsed / /** @type {number} */
      duration))), !0);
    }), this.#task.promise;
  }
  get current() {
    return get(this.#current);
  }
  get target() {
    return get(this.#target);
  }
  set target(v) {
    this.set(v);
  }
};

// node_modules/svelte/src/motion/index.js
var prefersReducedMotion = /* @__PURE__ */ new MediaQuery(
  "(prefers-reduced-motion: reduce)"
);
export {
  Spring,
  Tween,
  prefersReducedMotion,
  spring,
  tweened
};
