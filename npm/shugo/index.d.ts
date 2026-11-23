// The types of the module shugo, for the editor and for shugo check: the
// package shugo gives them to a project that installs it. The module itself
// is a part of the shugo command, internal/jsrt/runtime.js and search.js;
// a test of the repository checks that both name the same things.

/** A row of a table: its columns by their names. */
export type Row = Record<string, any>;

/** How where compares a column with a value. */
export type Comparison = "=" | "!=" | "<" | "<=" | ">" | ">=" | "like" | "in";

/**
 * A question to one table of the data database of the site. Every step
 * returns a new query; all, first, get and count ask it, at once, without
 * waiting. What a page reads of the rows decides when it is computed
 * again.
 */
export interface Query<T extends Row = Row> {
  /** The columns to take: select("id", "title"). Without it, all of them. */
  select(...columns: (string | string[])[]): Query<T>;
  /** The rows whose columns hold these values: where({ category: "c9", available: true }). */
  where(values: Record<string, unknown>): Query<T>;
  /**
   * The rows whose column compares so with a value: where("price", "<", 1000),
   * where("category", "in", ["c1", "c2"]). Only = and != compare with null.
   */
  where(column: string, comparison: Comparison, value: unknown): Query<T>;
  /** Sorts the rows by a column: order("price", "desc"). */
  order(column: string, direction?: "asc" | "desc"): Query<T>;
  limit(count: number): Query<T>;
  offset(count: number): Query<T>;
  /**
   * Puts into every row the row of another table that its column points
   * at, or null: join("categories", "category") gives row.categories.
   * select names the columns to take, as the property to put the row
   * under, on the column of the other table to find it by when that is
   * not its key.
   */
  join(table: string, column: string, options?: { select?: string[]; as?: string; on?: string }): Query<T>;
  /** The rows. */
  all(): T[];
  /** The first row, or null. */
  first(): T | null;
  /** The row with this key, or null. */
  get(key: string | number): T | null;
  /** How many rows there are, whatever the order, limit and offset. */
  count(): number;
}

/**
 * Starts a question to a table of the data database named in shugo.json.
 * Ask it in a page, a component, getStaticPaths or GET, or in a function
 * they call: at the top of a module nothing is computed yet.
 */
export function query<T extends Row = Row>(table: string): Query<T>;

/** A part of a list cut into pages, as paginate gives it to a page. */
export interface Paged<T> {
  /** The items of this page. */
  data: T[];
  /** The place of the first and of the last item of this page in the whole list, from 0. */
  start: number;
  end: number;
  /** How many items a page holds. */
  size: number;
  /** How many items there are in all. */
  total: number;
  currentPage: number;
  lastPage: number;
  /** The addresses of pages of the list; undefined where there is none. */
  url: { current?: string; prev?: string; next?: string; first?: string; last?: string };
}

/**
 * In getStaticPaths: the pages of a list of so many items, for a page file
 * named with [page] or [...page]: paginate(query("posts").count(), { pageSize: 10 }).
 */
export function paginate(total: number, options?: { pageSize?: number; params?: Record<string, string> }): { params: Record<string, string | undefined> }[];
/**
 * In a page: the part of a list whose number stands in the address of the
 * page. A query asks the data database for that part alone.
 */
export function paginate<T>(list: Query<T> | T[], options?: { pageSize?: number }): Paged<T>;

/** The page being computed, as page of $app/state is in SvelteKit. */
export const page: {
  /** The full address of the page, on the site of shugo.json. */
  readonly url: URL;
  /** The parameters of the address, as text: { slug: "jug" } for /products/[slug]. */
  readonly params: Record<string, string>;
  /** The route: { id: "/products/[slug]" }. */
  readonly route: { readonly id: string };
  /** 200, and 404 for the page 404.svelte. */
  readonly status: number;
};

/** The kinds of filters of a search: by the values of a column, by a range of numbers, by yes or no. */
export type FilterType = "multi" | "range" | "boolean";

/** An order of a search: by columns, { price: "asc" }, or by a list with a label. */
export type SearchSort = Record<string, "asc" | "desc"> | { by: { column: string; direction: "asc" | "desc" }[]; label?: string };

/** What a search of a table finds and shows. Every value is JSON. */
export interface SearchOptions {
  /** The columns of the full-text search by q. */
  text?: string[];
  /** More columns to show in hits. */
  fields?: string[];
  /** The filters of the address: { category: "multi", price: "range", available: "boolean" }. */
  filters?: Record<string, FilterType | { column?: string; type: FilterType; label?: string }>;
  /** The orders that sort may name: { cheap: { price: "asc" } }. */
  sorts?: Record<string, SearchSort>;
  /** The order when the address names none. */
  defaultSort?: string;
  /** How many hits a page of results holds; 24 if not given. */
  pageSize?: number;
  /** A part of the table that every search keeps to: { published: true }. */
  where?: Record<string, unknown>;
}

export interface FacetOption {
  value: string;
  label: string;
  count: number;
  selected: boolean;
}

export interface Facet {
  type: FilterType;
  label: string;
  options: FacetOption[];
  min: number | null;
  max: number | null;
  count: number;
}

export interface SortChoice {
  value: string;
  label: string;
  selected: boolean;
}

export interface PageLink {
  gap: boolean;
  current: boolean;
  label: string;
  href: string;
}

/** What a search found: hits, facets, pages and the address that asks again. */
export interface SearchResult<T extends Row = Row> {
  /** The address an island asks for the next results. */
  endpoint: string;
  hits: T[];
  query: { q: string; sort: string; page: number; filters: Record<string, unknown> };
  facets: Record<string, Facet>;
  sorts: SortChoice[];
  total: number;
  page: number;
  pages: number;
  hasHits: boolean;
  hasPages: boolean;
  hasPrev: boolean;
  hasNext: boolean;
  prevHref: string;
  nextHref: string;
  pageLinks: PageLink[];
}

/**
 * A search of a table of the data database, for an island: Shugo builds
 * its index and names it. Bind it to the request with
 * request:initial={search("products", { … })}.
 */
export function search<T extends Row = Row>(table: string, options?: SearchOptions): SearchResult<T>;
