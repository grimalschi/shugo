// The data database of a project, as its pages read it.
//
// A query is built step by step and run by all(), first(), get() or count():
//
//   Shugo.query("products").select("id", "title").where({ category: "c9" }).order("position").limit(24).all()
//
// Every query that runs leaves readings with the page or shared block that
// made it: which rows of which table it looked for, and which of their
// columns it took. The Go side keeps the readings, and when a row of the
// database changes it computes again exactly those who read it.
//
// A reading is the rows of one table that hold given values in given
// columns. Readings come in shapes: a shape names the table, those columns,
// and the columns a change of which matters to the reader. The shapes of a
// project are few, so a shape is told to the Go side once and a reading is
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
    this.statements = new Map(); // SQL → prepared statement
    this.tables = new Map(); // name → what table() returns
    this.shapes = new Map(); // what a shape is → its number
  }

  statement(sql) {
    let statement = this.statements.get(sql);
    if (statement === undefined) this.statements.set(sql, (statement = this.prepare(sql)));
    return statement;
  }

  // table returns the columns of a table: their names, what kind of values
  // each holds, which one is the key and which ones never repeat a value.
  table(name) {
    let table = this.tables.get(name);
    if (table !== undefined) return table;
    if (typeof name !== "string") throw new TypeError(`the name of a table is a text, got ${describe(name)}`);
    const columns = name.startsWith("_shugo_") ? [] : this.statement("SELECT name, type, pk FROM pragma_table_info(?) ORDER BY cid").all(name);
    const kind = this.statement("SELECT type FROM sqlite_master WHERE name = ?").get(name)?.type;
    if (kind === "view") {
      throw new Error(`${name} is a view. Shugo cannot tell which records a view is made of, so it cannot know when to compute a page again: query the tables themselves`);
    }
    if (kind !== "table" || columns.length === 0) {
      const names = this.statement("SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite\\_%' ESCAPE '\\' AND name NOT LIKE '\\_shugo\\_%' ESCAPE '\\' ORDER BY name").all().map((row) => row.name);
      throw new Error(`there is no table ${JSON.stringify(name)} in the data database; it has: ${names.join(", ") || "no tables"}`);
    }
    table = { name, columns: new Map(columns.map((column) => [column.name, kindOf(column.type)])), key: columns.find((column) => column.pk > 0).name, unique: new Set() };
    table.unique.add(table.key);
    for (const index of this.statement(`SELECT name FROM pragma_index_list(?) WHERE "unique" = 1`).all(name)) {
      const parts = this.statement("SELECT name FROM pragma_index_info(?)").all(index.name);
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
  if (type === "boolean") value = value ? 1 : 0;
  else if (type === "number" && Number.isFinite(value)) {
    if (kind === "text") throw new TypeError(`${table.name}.${column} holds text, and it is compared with the number ${value}: write ${JSON.stringify(String(value))}`);
  } else if (type === "string") {
    // A column of numbers holds a text only if the text is not a number.
    if (kind === "number" && numeral.test(value)) throw new TypeError(`${table.name}.${column} holds numbers, and it is compared with the text ${JSON.stringify(value)}: write ${Number(value)}`);
  } else {
    throw new TypeError(`${table.name}.${column} is compared with ${describe(value)}: a value in a query is a text, a number, true or false`);
  }
  return value;
}

function describe(value) {
  if (value === null) return "null";
  if (Array.isArray(value)) return "a list";
  return typeof value === "string" ? `the text ${JSON.stringify(value)}` : typeof value === "object" ? "an object" : typeof value === "undefined" ? "undefined" : `${typeof value} ${String(value)}`;
}

const quote = (name) => `"${name.replaceAll('"', '""')}"`;

const comparisons = new Set(["=", "!=", "<", "<=", ">", ">=", "like", "in"]);

// Reads collects the readings of one page or shared block while it is computed.
export class Reads {
  constructor() {
    this.seen = new Set();
  }

  note(shape, key) {
    this.seen.add(JSON.stringify([shape, ...key]));
  }

  // text is what goes to the Go side: a JSON list of readings, each the
  // number of its shape and the values of its key.
  text() {
    return this.seen.size === 0 ? "" : "[" + [...this.seen].join(",") + "]";
  }
}

// Query is a question to one table. Every step returns a new query, so a
// query can be kept and asked in several ways.
export class Query {
  #data;
  #reads;
  #table;
  #parts;

  constructor(data, reads, name, parts) {
    this.#data = data;
    this.#reads = reads;
    this.#table = data.table(name);
    this.#parts = parts ?? { select: null, equal: [], among: null, other: [], order: [], limit: null, offset: null, joins: [] };
  }

  #with(change) {
    return new Query(this.#data, this.#reads, this.#table.name, { ...this.#parts, ...change });
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
    if (names.length === 0) throw new Error("select() takes the names of columns: select(\"id\", \"title\")");
    for (const name of names) this.#column(name, "select");
    return this.#with({ select: [...new Set(names)] });
  }

  // where narrows the rows: where({ category: "c9", available: true }) for
  // equal values, where("price", "<", 1000) for another comparison. Several
  // of them all hold at once.
  where(column, comparison, value) {
    if (column !== null && typeof column === "object" && arguments.length === 1) {
      let next = this;
      for (const [name, wanted] of Object.entries(column)) next = wanted === null ? next.where(name, "=", null) : next.where(name, "=", wanted);
      return next;
    }
    if (arguments.length !== 3) throw new Error('where() takes an object of equal values, where({ category: "c9" }), or a column, a comparison and a value: where("price", "<", 1000)');
    this.#column(column, "where");
    if (typeof comparison !== "string" || !comparisons.has(comparison.toLowerCase())) {
      throw new Error(`where("${column}", ${JSON.stringify(comparison)}, …): the comparisons are ${[...comparisons].join(", ")}`);
    }
    comparison = comparison.toLowerCase();
    if (value === null || value === undefined) {
      if (comparison !== "=" && comparison !== "!=") throw new Error(`where("${column}", "${comparison}", ${value}): only = and != compare with null`);
      return this.#with({ other: [...this.#parts.other, [column, comparison === "=" ? "IS NULL" : "IS NOT NULL"]] });
    }
    if (comparison === "in") {
      if (!Array.isArray(value)) throw new TypeError(`where("${column}", "in", …) takes a list of values, got ${describe(value)}`);
      const values = [...new Set(value.map((one) => fit(this.#table, column, one)))];
      // The rows with one of several values are found by each value, like
      // rows with one value. One such list is enough for most queries.
      if (this.#parts.among === null) return this.#with({ among: [column, values] });
      return this.#with({ other: [...this.#parts.other, [column, `IN (${values.map(() => "?").join(", ")})`, ...values]] });
    }
    value = fit(this.#table, column, value);
    if (comparison === "=") return this.#with({ equal: [...this.#parts.equal.filter(([name]) => name !== column), [column, value]] });
    return this.#with({ other: [...this.#parts.other, [column, `${comparison === "like" ? "LIKE" : comparison} ?`, value]] });
  }

  // order sorts the rows by a column; order("price", "desc") from the
  // greatest. Rows that are equal there come in the order of their keys.
  order(column, direction = "asc") {
    this.#column(column, "order");
    if (direction !== "asc" && direction !== "desc") throw new Error(`order("${column}", ${JSON.stringify(direction)}): the direction is "asc" or "desc"`);
    return this.#with({ order: [...this.#parts.order, [column, direction]] });
  }

  limit(count) {
    if (!Number.isInteger(count) || count < 0) throw new TypeError(`limit() takes a whole number that is not negative, got ${describe(count)}`);
    return this.#with({ limit: count });
  }

  offset(count) {
    if (!Number.isInteger(count) || count < 0) throw new TypeError(`offset() takes a whole number that is not negative, got ${describe(count)}`);
    return this.#with({ offset: count });
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
    return this.#with({ joins: [...this.#parts.joins, { table: other, column, on, select: select ?? [...other.columns.keys()], as }] });
  }

  // all returns the rows as objects. The key of the table is among the
  // columns of every row, named or not.
  all() {
    return this.#run(this.#parts.limit, this.#parts.offset);
  }

  // first returns the first row, or null if there is none.
  first() {
    return this.#run(1, this.#parts.offset)[0] ?? null;
  }

  // get returns the row with the given key, or null.
  get(key) {
    return this.where(this.#table.key, "=", key).first();
  }

  // count tells how many rows there are, whatever the order, limit and offset.
  count() {
    const filter = this.#filter();
    const found = this.#data.statement(`SELECT count(*) AS count FROM ${quote(this.#table.name)}${filter.sql}`).get(...filter.values);
    this.#found(filter, filter.matters, null, false);
    return found.count;
  }

  // The filter as SQL with its values, and what a reading of it holds: the
  // columns that rows are found by with a key of their values (one key for
  // every value of a list), and the columns that decide which rows it finds.
  #filter() {
    const { equal, among, other } = this.#parts;
    const sorted = [...equal].sort(([a], [b]) => (a < b ? -1 : 1));
    const terms = sorted.map(([column]) => `${quote(column)} COLLATE BINARY = ?`);
    const values = sorted.map(([, value]) => value);
    let by = sorted.map(([column]) => column);
    let keys = [values.slice()];
    if (among !== null) {
      const [column, list] = among;
      terms.push(`${quote(column)} COLLATE BINARY IN (${list.map(() => "?").join(", ")})`);
      // The column of the list takes its place among the columns of the key.
      const at = by.filter((one) => one < column).length;
      by = by.toSpliced(at, 0, column);
      keys = list.map((one) => keys[0].toSpliced(at, 0, one));
      values.push(...list);
    }
    for (const [column, comparison, ...rest] of other) {
      terms.push(`${quote(column)} ${comparison}`);
      values.push(...rest);
    }
    const matters = new Set([...by, ...other.map(([column]) => column)]);
    return { sql: terms.length === 0 ? "" : " WHERE " + terms.join(" AND "), values, by, keys, matters };
  }

  // The columns a query takes from its table: the ones it names or all of
  // them, the key, and the ones its joins find rows by.
  #taken() {
    const table = this.#table;
    const named = this.#parts.select ?? [...table.columns.keys()];
    return [...new Set([...named, table.key, ...this.#parts.joins.map((join) => join.column)])];
  }

  #run(limit, offset) {
    const table = this.#table;
    const { order, joins } = this.#parts;
    const filter = this.#filter();
    const values = filter.values;
    // Rows come in one order every time: by the key where nothing else tells.
    const sorted = order.map(([column, direction]) => `${quote(column)}${direction === "desc" ? " DESC" : ""}`);
    if (!order.some(([column]) => column === table.key)) sorted.push(quote(table.key));
    let text = `SELECT ${this.#taken().map(quote).join(", ")} FROM ${quote(table.name)}${filter.sql} ORDER BY ${sorted.join(", ")}`;
    const cut = limit !== null || offset !== null;
    if (cut) {
      text += " LIMIT ? OFFSET ?";
      values.push(limit ?? -1, offset ?? 0);
    }
    const rows = this.#data.statement(text).all(...values);

    for (const join of joins) {
      const find = this.#data.statement(`SELECT ${[...new Set([...join.select, join.on])].map(quote).join(", ")} FROM ${quote(join.table.name)} WHERE ${quote(join.on)} COLLATE BINARY = ?`);
      const found = new Map();
      for (const row of rows) {
        const value = row[join.column];
        if (value == null) {
          row[join.as] = null;
          continue;
        }
        if (!found.has(value)) found.set(value, find.get(fit(join.table, join.on, value)) ?? null);
        row[join.as] = found.get(value);
      }
    }

    const matters = filter.matters;
    for (const [column] of order) matters.add(column);
    this.#found(filter, matters, rows, cut);
    return rows;
  }

  // found leaves the readings of a query that has run. matters are the
  // columns that decide which rows it finds and in which order; rows are
  // the rows it returned, or null for a query that takes no columns; cut
  // tells that a limit or an offset left some of the rows out.
  #found(filter, matters, rows, cut) {
    const data = this.#data;
    const table = this.#table;
    let taken = rows === null ? [] : this.#taken().filter((column) => column !== table.key && !matters.has(column));
    // A query that returns every row its values find reads those rows whole:
    // the columns it took belong to the same reading. So does one that looks
    // for a value no two rows share. Otherwise rows with those values may
    // change unseen behind the limit, and every row returned is a reading of
    // its own, found by its key.
    if (!cut || filter.by.some((column) => table.unique.has(column))) {
      for (const column of taken) matters.add(column);
      taken = [];
    }
    const shape = data.shape(table.name, filter.by, [...matters].sort());
    for (const key of filter.keys) this.#reads.note(shape, key);
    if (taken.length > 0) {
      const one = data.shape(table.name, [table.key], taken.sort());
      for (const row of rows) this.#reads.note(one, [row[table.key]]);
    }
    if (rows === null) return;
    for (const join of this.#parts.joins) {
      const joined = data.shape(join.table.name, [join.on], join.select.filter((column) => column !== join.on).sort());
      for (const row of rows) if (row[join.column] != null) this.#reads.note(joined, [row[join.column]]);
    }
  }
}
