import {
  noop,
  untrack
} from "./chunk-GFOPKZ2A.js";

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

export {
  subscribe_to_store
};
