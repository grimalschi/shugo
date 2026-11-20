// Tests of the library of the load side. They are run by jsrt_test.go on
// every JavaScript runtime there is, next to a copy of load.js.
import { isDeepStrictEqual } from "node:util";
import { Hasher, hex, reader } from "./load.js";

let failures = 0;
function fail(...what) {
  console.log("FAIL", ...what);
  failures++;
}

// --- The fingerprint is MurmurHash3 of the words, under two seeds -------------

// The same published values the Go test checks its own MurmurHash3 against.
// The first half of a fingerprint has the seed 0, the second 0x9747b28c.
const halves = (words) => {
  const h = new Hasher();
  for (const word of words) h.word(word);
  h.finish();
  return [h.a >>> 0, h.b >>> 0];
};
for (const [words, half, want] of [
  [[], 0, 0],
  [[0xffffffff], 0, 0x76293b50],
  [[0x87654321], 0, 0xf55b516b],
  [[0], 0, 0x2362f9de],
  [[0x74736574], 0, 0xba6bd213], // "test"
  [[0x61616161], 1, 0x5a97808a], // "aaaa"
]) {
  const got = halves(words)[half];
  if (got !== want) fail("MurmurHash3 of", words, "half", half, "is", got.toString(16), "and must be", want.toString(16));
}
if (hex(0x01020304, 0xfffefdfc | 0) !== "01020304fffefdfc") fail("hex", hex(0x01020304, 0xfffefdfc | 0));

// Text goes in as its length and two characters in a word, the same as if
// the words were fed one by one.
for (const text of ["", "a", "ab", "abc", "Café Cup 9", "😀 astral"]) {
  const h = new Hasher();
  h.string(text);
  h.finish();
  const words = [text.length];
  for (let i = 0; i + 1 < text.length; i += 2) words.push(text.charCodeAt(i) | (text.charCodeAt(i + 1) << 16));
  if (text.length % 2) words.push(text.charCodeAt(text.length - 1));
  if (!isDeepStrictEqual([h.a >>> 0, h.b >>> 0], halves(words))) fail("the text", JSON.stringify(text), "is not fed as its words");
}

// --- A fingerprint covers exactly what the template is given ------------------

// A small generator of numbers that repeats itself.
let seed = 20261120;
const random = () => ((seed = (Math.imul(seed, 1664525) + 1013904223) >>> 0) / 2 ** 32);
const below = (n) => Math.floor(random() * n);
const pick = (list) => list[below(list.length)];
const keys = ["a", "b", "c", "length"];

function value(depth) {
  switch (below(depth > 0 ? 13 : 8)) {
    case 0: return null;
    case 1: return undefined;
    case 2: return random() < 0.5;
    case 3: return below(5);
    case 4: return below(1000) / 8 + 0.1;
    case 5: return pick(["", "a", "ab", "Clay", "x".repeat(below(9))]);
    case 6: return BigInt(below(3));
    case 7: return 0;
    case 8: return Array.from({ length: below(4) }, () => value(depth - 1));
    case 9: return Object.fromEntries(keys.filter(() => random() < 0.6).map((key) => [key, value(depth - 1)]));
    case 10: return new Date(below(3) * 86400000);
    case 11: return new Map(Array.from({ length: below(3) }, (_, i) => [pick(["k", i]), value(depth - 1)]));
    default: return new Set(Array.from({ length: below(3) }, () => value(0)));
  }
}

function shape(depth) {
  switch (below(depth > 0 ? 5 : 2)) {
    case 0: return 1;
    case 1: return 0;
    case 2: return [shape(depth - 1)];
    default: {
      const names = keys.filter(() => random() < 0.5);
      return Object.fromEntries((names.length ? names : ["a"]).map((key) => [key, shape(depth - 1)]));
    }
  }
}

// changed returns a copy of a value with one thing in it changed.
function changed(v) {
  if (Array.isArray(v) && v.length > 0 && random() < 0.8) {
    const at = below(v.length);
    return v.map((item, i) => (i === at ? changed(item) : item));
  }
  if (v !== null && typeof v === "object" && Object.getPrototypeOf(v) === Object.prototype && Object.keys(v).length > 0 && random() < 0.8) {
    const key = pick(Object.keys(v));
    return { ...v, [key]: changed(v[key]) };
  }
  return value(1);
}

const print = (r, v) => {
  const h = new Hasher();
  r.print(h, v);
  h.finish();
  return hex(h.a, h.b);
};

let differing = 0;
for (let i = 0; i < 20000; i++) {
  const [s, v] = [shape(3), value(3)];
  const r = reader(s);
  const w = random() < 0.2 ? structuredClone(v) : changed(v);
  const [partV, partW] = [r.part(v), r.part(w)];
  const [printV, printW] = [print(r, v), print(r, w)];
  // The same is given to the template exactly when the fingerprint is the same.
  if (isDeepStrictEqual(partV, partW) !== (printV === printW)) {
    fail("shape", JSON.stringify(s), "\n  values", v, w, "\n  parts", partV, partW, "\n  fingerprints", printV, printW);
    if (failures > 5) break;
  }
  if (printV !== printW) differing++;
  // The part is a copy: it shares nothing with the value that can change.
  if (partV !== null && typeof partV === "object" && partV === v) fail("the part is the value itself", s, v);
}
if (differing < 3000) fail("the test changes too little:", differing, "pairs differ");

// What a template is given of some values, as examples.
for (const [s, v, want] of [
  [1, { a: [1, { b: 2 }], d: new Date(5) }, { a: [1, { b: 2 }], d: new Date(5) }],
  [0, { a: 1 }, true],
  [0, "", ""],
  [0, 0, 0],
  [0, [], true],
  [{ title: 1 }, { title: "Jug", price: 5 }, { title: "Jug" }],
  [{ title: 1 }, null, null],
  [{ length: 1 }, [1, 2, 3], { length: 3 }],
  [{ length: 1 }, "abc", "abc"],
  [{ discount: { percent: 1 } }, { discount: null }, { discount: null }],
  [[{ id: 1 }], [{ id: 1, x: 2 }, { id: 3 }], [{ id: 1 }, { id: 3 }]],
  [[1], new Set(["a", "b"]), ["a", "b"]],
  [[[1]], new Map([["k", 1]]), [["k", 1]]],
  [[1], null, null],
  [{ size: 1 }, new Map([["k", 1]]), { size: 1 }],
]) {
  const got = reader(s).part(v);
  if (!isDeepStrictEqual(got, want)) fail("the part of", v, "by", JSON.stringify(s), "is", got, "and must be", want);
}

// What is not data is refused, by the fingerprint and by the copy alike.
class Money {}
const circle = { name: "x" };
circle.self = circle;
for (const [s, v] of [[1, () => 1], [1, { f() {} }], [1, new Money()], [1, [Symbol("s")]], [{ a: 1 }, { a: () => 1 }], [[1], [() => 1]], [{ a: 1 }, () => 1], [1, circle]]) {
  for (const way of ["print", "part"]) {
    if (way === "part" && v === circle) continue; // a fingerprint is always made first
    try {
      way === "print" ? print(reader(s), v) : reader(s).part(v);
      fail(way, "takes what is not data:", v);
    } catch (error) {
      if (error.message !== "not data") fail(way, "fails in another way:", error);
    }
  }
}
// Only what is read counts: a function next to it is no matter.
try {
  print(reader({ a: 1 }), { a: 1, f() {} });
} catch (error) {
  fail("a function that the template does not read is refused:", error);
}

if (failures === 0) console.log("ok");
else process.exitCode = 1;
