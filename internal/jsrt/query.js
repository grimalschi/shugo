// The data database of a project, as its pages read it.
//
// A query is built step by step and run by all(), first(), get() or count():
//
//   Shugo.query("products").select("title", "price").where({ category: "c9" }).order("position").limit(24).all()
//
// Every query that runs leaves readings with the page or shared block that
// made it: which rows of which table it looked for, and which of their
// columns it took. The Go side keeps the readings, and when a row of the
// database changes it computes again exactly those who read it.
//
// A reading is the rows of one table that hold given values in given
// columns. Readings come in shapes: a shape names the table, those columns,
// and the columns a change of which matters to the reader. The shapes of a
// project are few, so a shape is told to the Go side once, and a reading is
// its number and the values.
//
// Runs on Bun and on Node: each has SQLite built in.

// open opens the database at path for reading. Everything read through it
// belongs to one state of the data, however long the run takes.
export async function open(path, declare) {
  let prepare;
  if (typeof Bun !== "undefined") {
    const { Database } = await import("bun:sqlite");
    const db = new Database(path, { readonly: true });
    db.exec("BEGIN");
    prepare = (sql) => db.query(sql);
  } else {
    const { DatabaseSync } = await import("node:sqlite");
    const db = new DatabaseSync(path, { readOnly: true });
    db.exec("BEGIN");
    prepare = (sql) => db.prepare(sql);
  }
  return new Data(prepare, declare);
}

// Data is the open database with what was learned about it so far.
class Data {
  constructor(prepare, declare) {
    this.prepare = prepare;
    this.declare = declare; // tells the Go side a new shape: its number and what it is
    this.tables = new Map(); // name → what table() returns
    this.shapes = new Map(); // what a shape is → its number
    // A project asks the same few questions with other values over and over.
    // How to ask one and what it reads is worked out once: see Query.#plan.
    this.plans = new Map();
  }

  // table returns the columns of a table: their names, what kind of values
  // each holds, which one is the key and which ones never repeat a value.
  table(name) {
    let table = this.tables.get(name);
    if (table !== undefined) return table;
    if (typeof name !== "string") throw new TypeError(`the name of a table is a text, got ${describe(name)}`);
    const columns = name.startsWith("_shugo_") ? [] : this.prepare("SELECT name, type, pk FROM pragma_table_info(?) ORDER BY cid").all(name);
    const kind = this.prepare("SELECT type FROM sqlite_master WHERE name = ?").get(name)?.type;
    if (kind === "view") {
      throw new Error(`${name} is a view. Shugo cannot tell which records a view is made of, so it cannot know when to compute a page again: query the tables themselves`);
    }
    if (kind !== "table" || columns.length === 0) {
      const names = this.prepare("SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite\\_%' ESCAPE '\\' AND name NOT LIKE '\\_shugo\\_%' ESCAPE '\\' ORDER BY name").all().map((row) => row.name);
      throw new Error(`there is no table ${JSON.stringify(name)} in the data database; it has: ${names.join(", ") || "no tables"}`);
    }
    table = { name, columns: new Map(columns.map((column) => [column.name, kindOf(column.type)])), key: columns.find((column) => column.pk > 0).name, unique: new Set() };
    table.unique.add(table.key);
    for (const index of this.prepare(`SELECT name FROM pragma_index_list(?) WHERE "unique" = 1`).all(name)) {
      const parts = this.prepare("SELECT name FROM pragma_index_info(?)").all(index.name);
      if (parts.length === 1 && parts[0].name !== null) table.unique.add(parts[0].name);
    }
    this.tables.set(name, table);
    return table;
  }

  // shape returns the number of a shape of readings.
  shape(table, by, columns) {
    const spec = JSON.stringify({ table, by, columns });
    let number = this.shapes.get(spec);
    if (number === undefined) {
      this.shapes.set(spec, (number = this.shapes.size));
      this.declare(number, spec);
    }
    return number;
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

// Reads collects the readings of one page or shared block while it is
// computed: the number of a shape, then the values of its key, one reading
// after another. The shape tells how many values a key has.
export class Reads {
  constructor() {
    this.list = [];
  }

  // text is what goes to the Go side: the readings as a JSON list.
  text() {
    return this.list.length === 0 ? "" : JSON.stringify(this.list);
  }
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

  // select names the columns to take. Without it a query takes all of them;
  // a page is computed again only when a column it took changes, so naming
  // the ones it shows keeps it from being computed for nothing.
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
    const found = plan.statement.get(...plan.places.map((at) => values[at]));
    this.#read(plan, null);
    return found.count;
  }

  // rows asks the question and leaves the readings.
  #rows(plan) {
    const values = this.#values;
    const rows = plan.statement.all(...plan.places.map((at) => values[at]));
    for (const join of plan.joins) {
      const found = new Map();
      for (const row of rows) {
        const value = row[join.column];
        if (value == null) {
          row[join.as] = null;
          continue;
        }
        if (!found.has(value)) found.set(value, join.statement.get(fit(join.table, join.on, value)) ?? null);
        row[join.as] = found.get(value);
      }
    }
    this.#read(plan, rows);
    return rows;
  }

  // read leaves the readings of a question that was asked: the rows that
  // hold its values, each row returned where the plan says so, and the rows
  // its joins looked for.
  #read(plan, rows) {
    const values = this.#values;
    const list = this.#reads.list;
    if (plan.among === null) {
      list.push(plan.shape);
      for (const at of plan.key) list.push(values[at]);
    } else {
      // One reading for every value of the list, the value at its place
      // among the others.
      for (const one of plan.among.values) {
        list.push(plan.shape);
        for (const at of plan.key) list.push(values[at === -1 ? one : at]);
      }
    }
    if (rows === null) return;
    if (plan.each !== -1) for (const row of rows) list.push(plan.each, row[plan.id]);
    for (const join of plan.joins) for (const row of rows) if (row[join.column] != null) list.push(join.shape, row[join.column]);
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

    plan = { statement: null, places, key: found.map((one) => one.value), among, shape: -1, each: -1, id: table.key, joins: [] };
    if (way === "count") {
      plan.statement = data.prepare(`SELECT count(*) AS count FROM ${quote(table.name)}${filter}`);
      plan.shape = data.shape(table.name, by, [...matters].sort());
      data.plans.set(name, plan);
      return plan;
    }

    // The columns to take: the ones named or all, the key, and the ones
    // the joins find rows by.
    const taken = [...new Set([...(select ?? table.columns.keys()), table.key, ...joins.map((join) => join.column)])];
    const sorted = order.map((one) => `${quote(one.column)}${one.direction === "desc" ? " DESC" : ""}`);
    for (const one of order) matters.add(one.column);
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
    // whole: the columns it took belong to the same reading. So does one that
    // looks for a value no two rows share. Otherwise rows with those values
    // may change unseen behind the limit, and every row returned is a reading
    // of its own, found by its key.
    let each = taken.filter((column) => column !== table.key && !matters.has(column));
    if (!cut || by.some((column) => table.unique.has(column))) {
      for (const column of each) matters.add(column);
      each = [];
    }
    plan.shape = data.shape(table.name, by, [...matters].sort());
    if (each.length > 0) plan.each = data.shape(table.name, [table.key], each.sort());
    for (const join of joins) {
      plan.joins.push({
        ...join,
        statement: data.prepare(`SELECT ${[...new Set([...join.select, join.on])].map(quote).join(", ")} FROM ${quote(join.table.name)} WHERE ${quote(join.on)} COLLATE BINARY = ?`),
        shape: data.shape(join.table.name, [join.on], join.select.filter((column) => column !== join.on).sort()),
      });
    }
    data.plans.set(name, plan);
    return plan;
  }
}
