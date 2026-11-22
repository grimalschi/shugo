// The data database of a project, as its pages read it.
//
// A query is built step by step and run by all(), first(), get() or count():
//
//   Shugo.query("products").where({ category: "c9" }).order("position").limit(24).all()
//
// Every query that runs leaves readings with the page or shared block that
// made it: which rows of which table it looked for, and which of their
// columns were read. The Go side keeps the readings, and when a row of the
// database changes it computes again exactly those who read it.
//
// Which columns were read is not what the query took but what was done with
// the rows afterwards. A row notes every column that is read of it, by
// whoever reads it while the page or shared block that asked for it is
// computed: its script, its template, the functions they call, the
// components the template gives the row to. So a reader depends on those
// columns and on no others, and a query need not name the columns it takes.
//
// This is sound for the reason signals are: a computation that reads the
// same values takes the same path and gives the same result. While nothing
// that was read has changed, computing again would read the same and show
// the same; and whatever makes it take another path — into the other branch
// of a condition, once more round a loop — is something it has read.
//
// A reading is the rows of one table that hold given values in given
// columns. Readings come in shapes: a shape names the table, those columns,
// and the columns a change of which matters to the reader. The shapes of a
// project are few, so a shape is told to the Go side once, and a reading is
// its number and the values.
//
// Runs on Bun and on Node: each has SQLite built in.


// open opens the database at path for reading. host is the Go side as the
// queries need it: declare tells it a new shape of readings, index asks it
// for an index and tells whether one had to be made.
//
// Everything is read inside one transaction, so that a run sees one state
// of the data however long it takes — until an index is made: see serve.
export async function open(path, host) {
  let db, prepare;
  if (typeof Bun !== "undefined") {
    const { Database } = await import("bun:sqlite");
    db = new Database(path, { readonly: true });
    prepare = (sql) => db.query(sql);
  } else {
    const { DatabaseSync } = await import("node:sqlite");
    db = new DatabaseSync(path, { readOnly: true });
    // Node gives rows without the methods every object has; a script or a
    // template may count on them, so rows are given them, as on Bun.
    const usual = (row) => Object.setPrototypeOf(row, Object.prototype);
    prepare = (sql) => {
      const statement = db.prepare(sql);
      return {
        all(...values) {
          const rows = statement.all(...values);
          rows.forEach(usual);
          return rows;
        },
        get(...values) {
          const row = statement.get(...values);
          return row === undefined ? undefined : usual(row);
        },
      };
    };
  }
  db.exec("BEGIN");
  const again = () => {
    db.exec("COMMIT");
    db.exec("BEGIN");
  };
  return new Data(prepare, again, host);
}

// Data is the open database with what was learned about it so far.
class Data {
  constructor(prepare, again, host) {
    this.prepare = prepare;
    this.again = again; // starts reading anew, from the state the database is in now
    this.host = host;
    this.tables = new Map(); // name → what table() returns
    this.shapes = new Map(); // what a shape is → its number
    this.nextShape = 0; // the number the next shape met for the first time gets
    // A project asks the same few questions with other values over and over.
    // How to ask one and what it reads is worked out once: see Query.#plan.
    this.plans = new Map();
    this.served = new Set(); // the questions that have their index, as the Go side writes them
  }

  // table returns the columns of a table: their names, what kind of values
  // each holds, which one is the key and which ones never repeat a value.
  // The columns of distinct tell any two rows apart: beside never repeating
  // a value, they always hold one.
  table(name) {
    let table = this.tables.get(name);
    if (table !== undefined) return table;
    if (typeof name !== "string") throw new TypeError(`the name of a table is a text, got ${describe(name)}`);
    const columns = name.startsWith("_shugo_") ? [] : this.prepare(`SELECT name, type, pk, "notnull" AS filled FROM pragma_table_info(?) ORDER BY cid`).all(name);
    const kind = this.prepare("SELECT type FROM sqlite_master WHERE name = ?").get(name)?.type;
    if (kind === "view") {
      throw new Error(`${name} is a view. Shugo cannot tell which records a view is made of, so it cannot know when to compute a page again: query the tables themselves`);
    }
    if (kind !== "table" || columns.length === 0) {
      const names = this.prepare("SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite\\_%' ESCAPE '\\' AND name NOT LIKE '\\_shugo\\_%' ESCAPE '\\' ORDER BY name").all().map((row) => row.name);
      throw new Error(`there is no table ${JSON.stringify(name)} in the data database; it has: ${names.join(", ") || "no tables"}`);
    }
    table = { name, columns: new Map(columns.map((column) => [column.name, kindOf(column.type)])), key: columns.find((column) => column.pk > 0).name, unique: new Set(), distinct: new Set() };
    table.unique.add(table.key);
    table.distinct.add(table.key);
    for (const index of this.prepare(`SELECT name FROM pragma_index_list(?) WHERE "unique" = 1 AND partial = 0`).all(name)) {
      const parts = this.prepare("SELECT name FROM pragma_index_info(?)").all(index.name);
      if (parts.length !== 1 || parts[0].name === null) continue;
      table.unique.add(parts[0].name);
      if (columns.find((column) => column.name === parts[0].name).filled) table.distinct.add(parts[0].name);
    }
    this.tables.set(name, table);
    return table;
  }

  // know takes what the Go side knows from earlier runs: the questions that
  // have their index, and the shapes of readings with their numbers in the
  // store. Neither is told to the Go side again.
  know({ plans, shapes }) {
    for (const question of plans) this.served.add(question);
    for (const [number, spec] of shapes) {
      this.shapes.set(spec, number);
      this.nextShape = Math.max(this.nextShape, number + 1);
    }
  }

  // shape returns the number of a shape of readings. A shape met for the
  // first time gets a number of its own, past those the Go side told, and
  // is told to the Go side.
  shape(table, by, columns) {
    const spec = JSON.stringify({ table, by, columns });
    let number = this.shapes.get(spec);
    if (number === undefined) {
      this.shapes.set(spec, (number = this.nextShape++));
      this.host.declare(number, spec);
    }
    return number;
  }

  // serve makes sure that the rows of a table with given values in the
  // columns of equal can be found without reading the whole table: counted
  // only, or, if rows is set, taken in the order of the columns of order
  // with nothing left to sort. The Go side owns the database: it is asked
  // for the index, and makes one unless an index that is there serves.
  //
  // A new index is seen only by who starts reading after it was made, so
  // reading starts anew then. What was read before belongs to an earlier
  // state of the data; whatever has changed in between is in the journal
  // past the place the update read it at, and the next update computes
  // again whoever read it.
  serve(table, equal, order, rows) {
    const question = [table.name, rows ? 1 : 0, equal.length, ...equal, ...order].join("\x02");
    if (this.served.has(question)) return;
    this.served.add(question);
    if (this.host.index(question)) this.again();
  }
}

// What kind of values a column holds decides how SQLite compares them: it
// reads the text "5" as a number for a column of numbers, and the number 5
// as text for a column of text. A reading must find the same rows the query
// did, so a value has to be of the kind of its column.
function kindOf(type) {
  const declared = type.toUpperCase();
  if (declared.includes("INT")) return "number";
  if (declared.includes("CHAR") || declared.includes("CLOB") || declared.includes("TEXT")) return "text";
  if (declared === "" || declared.includes("BLOB")) return "any";
  return "number";
}

const numeral = /^\s*[-+]?(\d+\.?\d*|\.\d+)([eE][-+]?\d+)?\s*$/;

// fit returns a value as it is compared with a column, or throws if SQLite
// would turn it into something else first.
function fit(table, column, value) {
  const kind = table.columns.get(column);
  const type = typeof value;
  if (type === "string") {
    // A column of numbers holds a text only if the text is not a number.
    if (kind === "number" && numeral.test(value)) throw new TypeError(`${table.name}.${column} holds numbers, and it is compared with the text ${JSON.stringify(value)}: write ${Number(value)}`);
    return value;
  }
  if (type === "number" && Number.isFinite(value)) {
    if (kind === "text") throw new TypeError(`${table.name}.${column} holds text, and it is compared with the number ${value}: write ${JSON.stringify(String(value))}`);
    return value;
  }
  if (type === "boolean") return value ? 1 : 0;
  throw new TypeError(`${table.name}.${column} is compared with ${describe(value)}: a value in a query is a text, a number, true or false`);
}

function describe(value) {
  if (value === null) return "null";
  if (Array.isArray(value)) return "a list";
  return typeof value === "string" ? `the text ${JSON.stringify(value)}` : typeof value === "object" ? "an object" : typeof value === "undefined" ? "undefined" : `${typeof value} ${String(value)}`;
}

const quote = (name) => `"${name.replaceAll('"', '""')}"`;

const comparisons = new Set(["=", "!=", "<", "<=", ">", ">=", "like", "in"]);

// Reads collects what one page, shared block or list of pages asked of the
// data while it was computed.
//
// It belongs to that computation alone. A question asked, or a column read,
// by another computation or after this one is over would leave nobody
// depending on the answer; so both are refused, with what to do instead.
let catalogAtModuleScope = false;
let searchesAtModuleScope;

export class Reads {
  constructor() {
    this.asked = [];
    this.catalog = false;
    this.searches = undefined;
    // settled tells that the readings are told: nothing more may be asked
    // or read. handed tells that the rows went on to others with every
    // column counted as read, and stay readable.
    this.settled = false;
    this.handed = false;
  }

  // text settles the readings and writes them for the Go side: a JSON list
  // of the number of a shape and the values of its key, one reading after
  // another; the shape tells how many values a key has.
  //
  // It is asked for when the page or shared block is fully computed. whole
  // tells that the rows go on to others who may read anything of them — a
  // list of pages hands them to its pages — and counts every column as read.
  text(whole = false) {
    this.settled = true;
    this.handed = whole;
    const catalog = this.catalog || catalogAtModuleScope;
    if (this.asked.length === 0 && !catalog && !this.searches && !searchesAtModuleScope) return "";
    // Wire shape -1 reads the default search result and has no key.
    const list = catalog ? [-1] : [];
    // Wire shape -2 names one generic search by its declaration ID.
    if (searchesAtModuleScope) {
      for (const id of searchesAtModuleScope) list.push(-2, id);
    }
    if (this.searches) {
      for (const id of this.searches) if (!searchesAtModuleScope?.has(id)) list.push(-2, id);
    }
    for (const one of this.asked) one.settle(list, whole);
    return JSON.stringify(list);
  }

  // mine tells whether the computation running now is the one these
  // readings are of, and is not over.
  mine() {
    return !this.settled && computing === this;
  }

  // late says why a question or a read is refused.
  late(what) {
    return (
      `${what} ${this.settled ? "when the page, shared block or list of pages it belongs to is already computed" : "by another page, shared block or list of pages than the one it belongs to"}, ` +
      "so whoever reads it now would not be computed again when the data changes. "
    );
  }
}

// Who is being computed right now: the Reads of the page, shared block or
// list of pages. Nothing waits while a page is computed, so one computation
// runs at a time and is all there is to know.
let computing = null;

// The default search result is a build input of the computation that asks
// for it, including a list of pages or a shared block.
export function readCatalog() {
  if (computing?.mine()) {
    computing.catalog = true;
  } else {
    // Module-level values can be shared by any consumer of the bundle.
    catalogAtModuleScope = true;
  }
}

export function readSearch(id) {
  if (computing?.mine()) {
    (computing.searches ??= new Set()).add(id);
  } else {
    (searchesAtModuleScope ??= new Set()).add(id);
  }
}

// during runs compute as the computation that reads belongs to.
export function during(reads, compute) {
  const outer = computing;
  computing = reads;
  try {
    return compute();
  } finally {
    computing = outer;
  }
}

// everything is noted among the columns read of a row when the row was
// taken as a whole: its keys were listed, as spreading it or writing it as
// JSON does.
const everything = "*";

// Watcher stands between a row of a table and whoever reads it, and notes in
// seen the columns that are read. The row itself is a plain object and
// stays one. reads is what the row was asked for.
//
// Only columns count. Whatever else is asked of a row — a method every
// object has, then, a symbol — tells nothing of the data.
class Watcher {
  constructor(reads, table, seen) {
    this.reads = reads;
    this.table = table;
    this.seen = seen;
  }

  // note notes a column that is read. Rows handed on to others have every
  // column counted already.
  note(name) {
    const reads = this.reads;
    if (reads.handed) return;
    if (!reads.mine()) {
      throw new Error(
        reads.late(`a row of the table ${this.table.name} is read`) +
          "A row is for the script that asked for it: do not keep it in a variable of a module for others, let every script ask for the rows it uses",
      );
    }
    this.seen.add(name);
  }

  get(row, name) {
    if (this.table.columns.has(name)) this.note(name);
    return row[name];
  }

  has(row, name) {
    if (this.table.columns.has(name)) this.note(name);
    return name in row;
  }

  ownKeys(row) {
    this.note(everything);
    return Reflect.ownKeys(row);
  }

  getOwnPropertyDescriptor(row, name) {
    if (this.table.columns.has(name)) this.note(name);
    return Reflect.getOwnPropertyDescriptor(row, name);
  }
}

// Asked is one question that was asked, with what was read of its answer.
class Asked {
  constructor(plan, values) {
    this.plan = plan;
    this.values = values;
    this.seen = null; // the columns read of the rows, where they count together
    this.rows = null; // [key, columns read] of every row, where each row counts alone
    this.joined = null; // [join, value, columns read] of every row a join looked for
  }

  // settle adds the readings of the question to list: the rows that hold
  // its values, each row returned where rows count alone, and the rows its
  // joins looked for.
  settle(list, whole) {
    const { plan, values } = this;
    const shape = this.seen === null ? plan.shape : plan.together(this.seen, whole);
    if (plan.among === null) {
      list.push(shape);
      for (const at of plan.key) list.push(values[at]);
    } else {
      // One reading for every value of the list, the value at its place
      // among the others.
      for (const one of plan.among.values) {
        list.push(shape);
        for (const at of plan.key) list.push(values[at === -1 ? one : at]);
      }
    }
    if (this.rows !== null) {
      for (const [key, seen] of this.rows) {
        const each = plan.each(seen, whole);
        if (each !== -1) list.push(each, key);
      }
    }
    if (this.joined !== null) {
      for (const [join, value, seen] of this.joined) list.push(join.shapeOf(seen, whole), value);
    }
  }
}

// shaper returns what tells the shape of a reading from the columns that
// were read of its rows. The rows are found by the columns of by; taken are
// the columns they were asked with. always are the columns that count
// whatever was read; without them (null) a reading of nothing has no shape,
// -1. covered are columns that another reading of the same question
// accounts for already.
//
// A script reads the same few sets of columns of row after row, so the
// shape of each set is worked out once.
function shaper(data, table, by, taken, always, covered = []) {
  const counted = taken.filter((column) => column !== table.key && !by.includes(column) && !covered.includes(column));
  const known = new Map();
  return (seen, whole) => {
    const all = whole || seen.has(everything);
    let read = "";
    for (let i = 0; i < counted.length; i++) if (all || seen.has(counted[i])) read += i + ",";
    let shape = known.get(read);
    if (shape === undefined) {
      const columns = new Set(always ?? []);
      for (const column of counted) if (all || seen.has(column)) columns.add(column);
      shape = columns.size === 0 && always === null ? -1 : data.shape(table.name, by, [...columns].sort());
      known.set(read, shape);
    }
    return shape;
  };
}

const blank = { select: null, equal: [], among: null, other: [], order: [], limit: -1, offset: -1, joins: [] };

// Query is a question to one table. Every step returns a new query, so a
// query can be kept and asked in several ways.
//
// A query knows its steps twice: as what they said, and as a text that
// names them without their values. A project asks few different questions,
// so how to ask a question — its SQL, the order of its values and the
// shapes of what it reads — is worked out when it is first asked, and found
// by that text ever after.
export class Query {
  #data;
  #reads;
  #table;
  #steps;
  #values;
  #parts;

  constructor(data, reads, name, steps = name, values = [], parts = blank) {
    this.#data = data;
    this.#reads = reads;
    this.#table = data.table(name);
    this.#steps = steps;
    this.#values = values;
    this.#parts = parts;
  }

  // A step adds its name to the steps, its values to the values, and what
  // it said to the parts.
  #step(name, values, change) {
    return new Query(this.#data, this.#reads, this.#table.name, this.#steps + "\n" + name, values.length === 0 ? this.#values : [...this.#values, ...values], { ...this.#parts, ...change });
  }

  #column(name, where) {
    if (!this.#table.columns.has(name)) {
      throw new Error(`there is no column ${JSON.stringify(name)} in the table ${this.#table.name} (${where}); it has: ${[...this.#table.columns.keys()].join(", ")}`);
    }
    return name;
  }

  // select names the columns to take. Without it a query takes all of them.
  // It is a way to fetch less and no more than that: what a reader depends
  // on is the columns that are read of the rows, named here or not.
  select(...columns) {
    const names = columns.flat();
    if (names.length === 0) throw new Error('select() takes the names of columns: select("id", "title")');
    for (const name of names) this.#column(name, "select");
    return this.#step("select " + names.join(","), [], { select: [...new Set(names)] });
  }

  // where narrows the rows: where({ category: "c9", available: true }) for
  // equal values, where("price", "<", 1000) for another comparison. Several
  // of them all hold at once.
  where(column, comparison, value) {
    if (column !== null && typeof column === "object" && arguments.length === 1) {
      let next = this;
      for (const name in column) next = next.where(name, "=", column[name]);
      return next;
    }
    if (arguments.length !== 3) throw new Error('where() takes an object of equal values, where({ category: "c9" }), or a column, a comparison and a value: where("price", "<", 1000)');
    this.#column(column, "where");
    if (typeof comparison !== "string" || !comparisons.has((comparison = comparison.toLowerCase()))) {
      throw new Error(`where("${column}", ${JSON.stringify(comparison)}, …): the comparisons are ${[...comparisons].join(", ")}`);
    }
    const parts = this.#parts;
    const at = this.#values.length;
    if (value === null || value === undefined) {
      if (comparison !== "=" && comparison !== "!=") throw new Error(`where("${column}", "${comparison}", ${value}): only = and != compare with null`);
      const test = comparison === "=" ? "IS NULL" : "IS NOT NULL";
      return this.#step(`where ${column} ${test}`, [], { other: [...parts.other, { column, test, values: [] }] });
    }
    if (comparison === "in") {
      if (!Array.isArray(value)) throw new TypeError(`where("${column}", "in", …) takes a list of values, got ${describe(value)}`);
      const list = [...new Set(value.map((one) => fit(this.#table, column, one)))];
      const places = list.map((_, i) => at + i);
      // The rows with one of several values are found by each value, like
      // rows with one value. One such list is enough for most queries; the
      // next one only narrows what the first finds.
      if (parts.among === null && !parts.equal.some((one) => one.column === column)) {
        return this.#step(`where ${column} in ${list.length}`, list, { among: { column, values: places } });
      }
      return this.#step(`where ${column} in ${list.length} more`, list, { other: [...parts.other, { column, test: `IN (${list.map(() => "?").join(", ")})`, values: places }] });
    }
    value = fit(this.#table, column, value);
    if (comparison === "=") {
      if (parts.equal.some((one) => one.column === column) || parts.among?.column === column) {
        throw new Error(`where: ${this.#table.name}.${column} is already compared with a value in this query`);
      }
      return this.#step(`where ${column} =`, [value], { equal: [...parts.equal, { column, value: at }] });
    }
    return this.#step(`where ${column} ${comparison}`, [value], { other: [...parts.other, { column, test: `${comparison === "like" ? "LIKE" : comparison} ?`, values: [at] }] });
  }

  // order sorts the rows by a column; order("price", "desc") from the
  // greatest. Rows that are equal there, and all rows without an order, come
  // as the database finds them: name what tells them apart where it matters.
  order(column, direction = "asc") {
    this.#column(column, "order");
    if (direction !== "asc" && direction !== "desc") throw new Error(`order("${column}", ${JSON.stringify(direction)}): the direction is "asc" or "desc"`);
    return this.#step(`order ${column} ${direction}`, [], { order: [...this.#parts.order, { column, direction }] });
  }

  limit(count) {
    if (!Number.isInteger(count) || count < 0) throw new TypeError(`limit() takes a whole number that is not negative, got ${describe(count)}`);
    return this.#step("limit", [count], { limit: this.#values.length });
  }

  offset(count) {
    if (!Number.isInteger(count) || count < 0) throw new TypeError(`offset() takes a whole number that is not negative, got ${describe(count)}`);
    return this.#step("offset", [count], { offset: this.#values.length });
  }

  // join puts into every row the row of another table that its column
  // points at: join("categories", "category") gives row.categories, the
  // category whose key is row.category, or null. The options name the columns
  // to take, the property to put the row under, and the column of the other
  // table to find it by when that is not its key:
  //   join("categories", "category", { select: ["title"], as: "group", on: "id" })
  join(name, column, { select = null, as = name, on } = {}) {
    this.#column(column, "join");
    const other = this.#data.table(name);
    on ??= other.key;
    const missing = [on, ...(select ?? [])].find((one) => !other.columns.has(one));
    if (missing !== undefined) throw new Error(`there is no column ${JSON.stringify(missing)} in the table ${name} (join); it has: ${[...other.columns.keys()].join(", ")}`);
    if (!other.unique.has(on)) {
      throw new Error(`join("${name}", "${column}"): ${name}.${on} may hold the same value in several rows. A join finds one row by its key or by a column that is unique; for many rows, make a query of its own: query("${name}").where({ ${on}: … })`);
    }
    if (this.#table.columns.has(as) || this.#parts.joins.some((join) => join.as === as)) {
      throw new Error(`join("${name}", "${column}"): a row already has ${JSON.stringify(as)}; give the joined row another name: { as: "…" }`);
    }
    select ??= [...other.columns.keys()];
    return this.#step(`join ${name} ${column} ${on} ${as} ${select.join(",")}`, [], { joins: [...this.#parts.joins, { table: other, column, on, select, as }] });
  }

  // all returns the rows as objects. The key of the table is among the
  // columns of every row, named or not.
  all() {
    return this.#rows(this.#plan("all"));
  }

  // first returns the first row, or null if there is none.
  first() {
    return this.#rows(this.#plan("first"))[0] ?? null;
  }

  // get returns the row with the given key, or null.
  get(key) {
    return this.where(this.#table.key, "=", key).first();
  }

  // count tells how many rows there are, whatever the order, limit and offset.
  count() {
    const plan = this.#plan("count");
    const values = this.#values;
    this.#mine();
    const found = plan.statement.get(...plan.places.map((at) => values[at]));
    this.#reads.asked.push(new Asked(plan, values));
    return found.count;
  }

  // mine makes sure that the question is asked by the computation it was
  // made for: its answer is noted there and nowhere else.
  #mine() {
    if (this.#reads.mine()) return;
    throw new Error(
      this.#reads.late(`the table ${this.#table.name} is asked`) +
        "Shugo.query belongs to the script it is called in: do not keep Shugo, or a query made of it, in a variable of a module for others",
    );
  }

  // rows asks the question and returns its rows, each of them noting what
  // is read of it from now on.
  #rows(plan) {
    const values = this.#values;
    const reads = this.#reads;
    this.#mine();
    const rows = plan.statement.all(...plan.places.map((at) => values[at]));
    const asked = new Asked(plan, values);
    for (const join of plan.joins) {
      // Rows that point at the same row get the same one.
      const found = new Map();
      asked.joined ??= [];
      for (const row of rows) {
        const value = row[join.column];
        if (value == null) {
          row[join.as] = null;
          continue;
        }
        if (!found.has(value)) {
          const other = join.statement.get(fit(join.table, join.on, value));
          const seen = new Set();
          // A row that is not there is read too: it is missed until it comes.
          asked.joined.push([join, value, seen]);
          found.set(value, other == null ? null : new Proxy(other, new Watcher(reads, join.table, seen)));
        }
        row[join.as] = found.get(value);
      }
    }
    // The column a join finds its row by is read of every row: by the join.
    const table = this.#table;
    if (plan.alone) {
      asked.rows = [];
      for (let i = 0; i < rows.length; i++) {
        const seen = new Set(plan.pointers);
        asked.rows.push([rows[i][table.key], seen]);
        rows[i] = new Proxy(rows[i], new Watcher(reads, table, seen));
      }
    } else {
      asked.seen = new Set(plan.pointers);
      const watcher = new Watcher(reads, table, asked.seen);
      for (let i = 0; i < rows.length; i++) rows[i] = new Proxy(rows[i], watcher);
    }
    reads.asked.push(asked);
    return rows;
  }

  // plan works out how to ask the question in one of its ways, and what it
  // reads, once for all questions with these steps.
  #plan(way) {
    const name = this.#steps + "\n" + way;
    let plan = this.#data.plans.get(name);
    if (plan !== undefined) return plan;

    const data = this.#data;
    const table = this.#table;
    const { select, equal, among, other, order, limit, offset, joins } = this.#parts;
    // The rows are found by the columns that are compared with one value or
    // with a list of them, in the order of their names: that is the key of
    // the reading. The value of a list stands at -1.
    const found = [...equal, ...(among === null ? [] : [{ column: among.column, value: -1 }])].sort((a, b) => (a.column < b.column ? -1 : 1));
    const by = found.map((one) => one.column);
    const terms = [];
    const places = [];
    for (const one of found) {
      if (one.value === -1) {
        terms.push(`${quote(one.column)} COLLATE BINARY IN (${among.values.map(() => "?").join(", ")})`);
        places.push(...among.values);
      } else {
        terms.push(`${quote(one.column)} COLLATE BINARY = ?`);
        places.push(one.value);
      }
    }
    for (const one of other) {
      terms.push(`${quote(one.column)} ${one.test}`);
      places.push(...one.values);
    }
    const filter = terms.length === 0 ? "" : " WHERE " + terms.join(" AND ");
    // What decides which rows are found, and in which order.
    const matters = new Set([...by, ...other.map((one) => one.column)]);

    // The rows come in the order asked for, and rows that it does not tell
    // apart in the order of their keys — unless one row at most is found,
    // or the order tells any two rows apart already. Left unsaid, the order
    // of such rows would be that of whichever index found them, and of the
    // places they happen to lie at in the table: a page would show another
    // one after an index was made, or after a row was removed and put back
    // as it was.
    const told = order.findIndex((one) => table.distinct.has(one.column));
    const ties = way !== "count" && told === -1 && !by.some((column) => table.unique.has(column));

    // The index that serves the question: the columns the rows are found by
    // with given values, then the columns of its order, down to the one
    // that tells any two rows apart. Rows come from it in their order, and
    // nothing is sorted. A question that neither narrows nor orders needs
    // none. Other comparisons add nothing: the rows they leave out are read
    // all the same.
    const through = way === "count" ? [] : [...order.slice(0, told === -1 ? order.length : told + 1).map((one) => one.column), ...(ties ? [table.key] : [])];
    const ordered = [...new Set(through)].filter((column) => !by.includes(column));
    if (by.length + ordered.length > 0) data.serve(table, by, ordered, way !== "count");

    // A plan of a question is what stays the same whatever its values: how
    // to ask it, where its values stand, and how to tell the shapes of its
    // readings from the columns that are read of its rows.
    plan = { statement: null, places, key: found.map((one) => one.value), among, shape: -1, alone: false, together: null, each: null, joins: [], pointers: joins.map((join) => join.column) };
    const decides = (columns) => [...columns].filter((column) => !by.includes(column)).sort();
    if (way === "count") {
      plan.statement = data.prepare(`SELECT count(*) AS count FROM ${quote(table.name)}${filter}`);
      plan.shape = data.shape(table.name, by, decides(matters));
      data.plans.set(name, plan);
      return plan;
    }

    // The columns to take: the ones named or all, the key, and the ones
    // the joins find rows by.
    const taken = [...new Set([...(select ?? table.columns.keys()), table.key, ...joins.map((join) => join.column)])];
    const sorted = order.map((one) => `${quote(one.column)}${one.direction === "desc" ? " DESC" : ""}`);
    for (const one of order) matters.add(one.column);
    // The key is compared exactly, as the index holds it, and runs the way
    // the order before it runs, for the index to be read from one end.
    if (ties) sorted.push(quote(table.key) + " COLLATE BINARY" + (order.at(-1)?.direction === "desc" ? " DESC" : ""));
    let text = `SELECT ${taken.map(quote).join(", ")} FROM ${quote(table.name)}${filter}${sorted.length === 0 ? "" : " ORDER BY " + sorted.join(", ")}`;
    const cut = way === "first" || limit !== -1 || offset !== -1;
    if (cut) {
      text += way === "first" ? " LIMIT 1" : limit === -1 ? " LIMIT -1" : " LIMIT ?";
      if (way !== "first" && limit !== -1) places.push(limit);
      if (offset !== -1) {
        text += " OFFSET ?";
        places.push(offset);
      }
    }
    plan.statement = data.prepare(text);

    // A question that returns every row its values find reads those rows
    // together: what is read of any of them belongs to the one reading. So
    // does one that looks for a value no two rows share. Otherwise rows with
    // those values may change unseen behind the limit, and every row returned
    // counts alone: it is a reading of its own, found by its key, of the
    // columns read of it that do not already decide which rows are found.
    plan.alone = cut && !by.some((column) => table.unique.has(column));
    if (plan.alone) {
      plan.shape = data.shape(table.name, by, decides(matters));
      plan.each = shaper(data, table, [table.key], taken, null, [...matters]);
    } else {
      plan.together = shaper(data, table, by, taken, decides(matters));
    }
    for (const join of joins) {
      // A row that a join looks for is a reading whatever is read of it: it
      // may come, or go.
      const columns = [...new Set([...join.select, join.on])];
      plan.joins.push({
        ...join,
        statement: data.prepare(`SELECT ${columns.map(quote).join(", ")} FROM ${quote(join.table.name)} WHERE ${quote(join.on)} COLLATE BINARY = ?`),
        shapeOf: shaper(data, join.table, [join.on], columns, []),
      });
    }
    data.plans.set(name, plan);
    return plan;
  }
}
