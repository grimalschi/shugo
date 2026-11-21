-- The data of a small shop: three products in two categories.
CREATE TABLE shop (id INTEGER PRIMARY KEY, name TEXT NOT NULL, phone TEXT NOT NULL);
CREATE TABLE categories (id TEXT PRIMARY KEY, title TEXT NOT NULL, position INTEGER NOT NULL);
CREATE TABLE products (
  id TEXT PRIMARY KEY, slug TEXT NOT NULL UNIQUE, title TEXT NOT NULL,
  price INTEGER NOT NULL, category TEXT NOT NULL, note TEXT NOT NULL DEFAULT ''
);
CREATE TABLE variants (sku TEXT PRIMARY KEY, product TEXT NOT NULL, title TEXT NOT NULL, stock INTEGER NOT NULL, position INTEGER NOT NULL);

INSERT INTO shop VALUES (1, 'Pottery Shop', '+1 202-555-0142');
INSERT INTO categories VALUES ('c1', 'Pitchers', 1), ('c2', 'Platters', 2);
INSERT INTO products VALUES
  ('p1', 'jug', 'North Pitcher', 2400, 'c1', ''),
  ('p2', 'pitcher', 'Morning Pitcher', 1900, 'c1', ''),
  ('p3', 'dish', 'Salt Platter', 3100, 'c2', '');
INSERT INTO variants VALUES
  ('p1-s', 'p1', 'small', 3, 1), ('p1-l', 'p1', 'large', 0, 2),
  ('p2-s', 'p2', 'small', 5, 1),
  ('p3-s', 'p3', 'one size', 2, 1);
