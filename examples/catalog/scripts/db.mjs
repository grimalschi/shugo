// One way to open a SQLite database on Bun and on Node, for the scripts
// that fill the data of the shop and change it.
export async function open(path) {
  if (typeof Bun !== "undefined") {
    const { Database } = await import("bun:sqlite");
    const db = new Database(path);
    return {
      exec: (sql) => db.exec(sql),
      prepare: (sql) => db.query(sql),
      close: () => db.close(),
    };
  }
  const { DatabaseSync } = await import("node:sqlite");
  const db = new DatabaseSync(path);
  return { exec: (sql) => db.exec(sql), prepare: (sql) => db.prepare(sql), close: () => db.close() };
}
