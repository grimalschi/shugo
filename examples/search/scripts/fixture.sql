CREATE TABLE products (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  category TEXT NOT NULL,
  price INTEGER NOT NULL,
  available INTEGER NOT NULL,
  internal_code TEXT NOT NULL
);
CREATE TABLE articles (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  section TEXT NOT NULL,
  minutes INTEGER NOT NULL,
  published INTEGER NOT NULL,
  editor_note TEXT NOT NULL
);

INSERT INTO products VALUES
  ('p1', 'Salt Cup', 'ceramics', 120, 1, 'private-product-one'),
  ('p2', 'Flax Cup', 'ceramics', 260, 0, 'private-product-two'),
  ('p3', 'Moss Teapot', 'ceramics', 360, 1, 'private-product-three'),
  ('p4', 'Shore Tray', 'wood', 90, 1, 'private-product-four'),
  ('p5', 'Forest Cup', 'wood', 170, 1, 'private-product-five');

INSERT INTO articles VALUES
  ('a1', 'Clay cups', 'guides', 8, 1, 'private-article-one'),
  ('a2', 'How to make a tray', 'notes', 3, 1, 'private-article-two'),
  ('a3', 'A teapot at home', 'guides', 4, 0, 'private-article-three'),
  ('a4', 'Handmade mugs', 'guides', 12, 1, 'private-article-four');
