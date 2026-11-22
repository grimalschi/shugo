import { catalogView } from "shugo:runtime";

// A fresh copy of the default result from the same catalog snapshot used by Go.
export function search(...args) {
  if (args.length) throw new TypeError('import { search } from "shugo:search" to search a named table');
  return catalogView();
}

export { search as catalog };
