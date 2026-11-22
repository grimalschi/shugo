import {
  effect_tracking,
  get,
  increment,
  noop,
  queue_micro_task,
  render_effect,
  run_all,
  safe_not_equal,
  source,
  tag,
  untrack
} from "./chunk-SJURP6G5.js";
import {
  false_default
} from "./chunk-CG7SAXEK.js";

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

// node_modules/svelte/src/store/utils.js
function subscribe_to_store(store, run, invalidate) {
  if (store == null)
    return run(void 0), invalidate && invalidate(void 0), noop;
  let unsub = untrack(
    () => store.subscribe(
      run,
      // @ts-expect-error
      invalidate
    )
  );
  return unsub.unsubscribe ? () => unsub.unsubscribe() : unsub;
}

// node_modules/svelte/src/store/shared/index.js
var subscriber_queue = [];
function readable(value, start) {
  return {
    subscribe: writable(value, start).subscribe
  };
}
function writable(value, start = noop) {
  let stop = null, subscribers = /* @__PURE__ */ new Set();
  function set(new_value) {
    if (safe_not_equal(value, new_value) && (value = new_value, stop)) {
      let run_queue = !subscriber_queue.length;
      for (let subscriber of subscribers)
        subscriber[1](), subscriber_queue.push(subscriber, value);
      if (run_queue) {
        for (let i = 0; i < subscriber_queue.length; i += 2)
          subscriber_queue[i][0](subscriber_queue[i + 1]);
        subscriber_queue.length = 0;
      }
    }
  }
  function update(fn) {
    set(fn(
      /** @type {T} */
      value
    ));
  }
  function subscribe(run, invalidate = noop) {
    let subscriber = [run, invalidate];
    return subscribers.add(subscriber), subscribers.size === 1 && (stop = start(set, update) || noop), run(
      /** @type {T} */
      value
    ), () => {
      subscribers.delete(subscriber), subscribers.size === 0 && stop && (stop(), stop = null);
    };
  }
  return { set, update, subscribe };
}
function derived(stores, fn, initial_value) {
  let single = !Array.isArray(stores), stores_array = single ? [stores] : stores;
  if (!stores_array.every(Boolean))
    throw new Error("derived() expects stores as input, got a falsy value");
  let auto = fn.length < 2;
  return readable(initial_value, (set, update) => {
    let started = !1, values = [], pending = 0, cleanup = noop, sync = () => {
      if (pending)
        return;
      cleanup();
      let result = fn(single ? values[0] : values, set, update);
      auto ? set(result) : cleanup = typeof result == "function" ? result : noop;
    }, unsubscribers = stores_array.map(
      (store, i) => subscribe_to_store(
        store,
        (value) => {
          values[i] = value, pending &= ~(1 << i), started && sync();
        },
        () => {
          pending |= 1 << i;
        }
      )
    );
    return started = !0, sync(), function() {
      run_all(unsubscribers), cleanup(), started = !1;
    };
  });
}
function readonly(store) {
  return {
    // @ts-expect-error TODO i suspect the bind is unnecessary
    subscribe: store.subscribe.bind(store)
  };
}
function get2(store) {
  let value;
  return subscribe_to_store(store, (_) => value = _)(), value;
}

export {
  createSubscriber,
  subscribe_to_store,
  readable,
  writable,
  derived,
  readonly,
  get2 as get
};
