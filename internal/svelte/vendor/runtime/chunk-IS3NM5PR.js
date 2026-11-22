// node_modules/svelte/src/reactivity/index-server.js
var SvelteDate = globalThis.Date, SvelteSet = globalThis.Set, SvelteMap = globalThis.Map, SvelteURL = globalThis.URL, SvelteURLSearchParams = globalThis.URLSearchParams, MediaQuery = class {
  current;
  /**
   * @param {string} query
   * @param {boolean} [matches]
   */
  constructor(query, matches = !1) {
    this.current = matches;
  }
};
function createSubscriber(_) {
  return () => {
  };
}

export {
  SvelteDate,
  SvelteSet,
  SvelteMap,
  SvelteURL,
  SvelteURLSearchParams,
  MediaQuery,
  createSubscriber
};
