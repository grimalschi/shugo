import { catalogView } from "shugo:runtime";

// A fresh copy of the default result from the same catalog snapshot used by Go.
export function search() {
  return catalogView();
}

export { search as catalog };
