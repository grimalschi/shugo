# Comparisons and windows in readings

Proposal on how to make accurate queries with comparisons (`<`, `<=`, `>`, `>=`) and with the `limit` window. Now they are true, but after changing the data, they calculate more than necessary.

## Decisions

- **Deferred** is the words of the author of the project, 21 November 2026: "We postpone while we work with direct equality."

## Like now

Each query through `Shugo.query` leaves a reading: a table, columns for which strings are found by equality, their values and columns, the change of which is important to the reader. When a string changes, the update takes its old and new values of these columns and finds out who read it. How it works in [Analysis of data changes](../change-tracking.md).

Equality of reading remembers exactly. `where({ category: "c1" })` only affects the string that was or became a `c1` category string.

Comparisons and window reading do not fall. The comparison column is written among the important ones, and its boundaries are not. Therefore, the request affects any row of the table that has changed this column, even if it did not fit under the condition either before or after the change. Same with `limit`: a string that was not in the window and will not get into it, still affects the request.

The result is not spoiled: the page can not remain outdated. The only other calculation: the page is considered anew and goes back.

Example: A movie with its neighbors:

```svelte
<!-- src/pages/films/[slug].svelte -->
<script module>
  export function getStaticPaths({ query }) {
    return query("films").select("slug").all().map(({ slug }) => ({ params: { slug } }))
  }
</script>

<script>
  const film = Shugo.query("films").where({ slug: Shugo.params.slug }).first()
  const near = Shugo.query("films")
    .where("rating", ">=", film.rating - 0.5)
    .where("rating", "<=", film.rating + 0.5)
    .where("id", "!=", film.id)
    .order("rating", "desc")
    .limit(5)
    .all()
</script>

<h1>{film.title}</h1>
<ul>
  {#each near as other}<li>{other.title}: {other.rating}</li>{/each}
</ul>
```

Measurement 21 November 2026 Years: 1,000 movies, on the page of each - five neighbors with a rating from r  0.5 up to r + 0.5. Changed the rating of one film. Update calculated all 1,000 pages, and HTML changed to 21.

Other tables and columns that the request does not use do not affect it. Changing the title of the film only calculates the pages where this title is read.

## Proposal

### Limits of Reading Conditions

Reading not only remembers the values of the equations, but also the boundaries of each comparison: the column, the lower and upper bounds, and whether they are included. `>= 4` is the segment from 4 inclusive to infinity, `< 1` is from minus infinity to 1, not including it. Two comparisons of one column, as in the example, give one segment.

The changed line will only affect reading if:

1. the column that is important to the reader has changed, as it is now;
2. and the old line fit all reading conditions, or the new one fits. Equality is checked as it is now, comparisons are checked by borders.

A line that did not fit before the change and does not fit after, the reader does not hurt: he did not see it and will not see it.

`count()` is even less affected: the number of rows changes only when a row enters or exits under conditions. Changing the columns of a row that was and remains under conditions does not affect the number.

### How to find a damaged reading quickly

Readings of one kind can be as many as pages: each film has its own boundaries. You can not go through them all for each changed line. You need an index that finds all the segments in which it falls by value.

In SQLite, there is such an index - the R*Tree module. It works in our driver on pure Go. Each reading with comparisons becomes a segment (or rectangle if two columns are compared); equality remains the key to reading, as it is now. According to the old and new value of the string, R*Tree gives the readings whose segments contain them, and Go checks the remaining conditions.

Details to be considered:

- R*Tree stores 32-bit boundaries and rounds them outwards. It can return extra reading, but will not miss the necessary one. Therefore, the found readings are checked by exact boundaries.
- R*Tree knows only numbers. Text comparisons — and dates in the database are stored with text in the form of ISO 8601 — need to be translated into numbers: date — into seconds. Comparison of text that is not translated into a number remains rough, as it is now.

### Window `limit`

The query with `order` and `limit` shows the first rows in order. The rows are in order of the query, and equals in it are in the table key ([Change tracking](../change-tracking.md)). Therefore, the window has an exact border: the values of the order columns and the key of the last row of the window.

Reading remembers that boundary. A changed line only affects a window if it fits the conditions and its old or new location is not beyond the boundary: it was in the window or falls into it. The line that is before and after the change is abroad, the window does not affect. If a line leaves the window, the following one comes in its place this change also affects reading, because the old place of the line was in the window.

With `offset`, the window does not start with the first line. A line that stands in front of a window or leaves under it shifts the entire window. Therefore, for a window with `offset`, only the final border is important: it is affected by any changed string that fits the conditions and stands no further than the end of the window.

### What will remain rude

- `!=` and `like`: the first is almost the entire line, the second is not a segment of numbers.
- Compare text that is not translated into a number.

### What is it worth?

- Reading with comparisons becomes more: borders are stored with it.
- For each changed line, search R*Tree for each type of reading with comparisons of its table, for old and new values.
- How much it costs on a large site, you need to measure: changing thousands of lines at once is the main case where additional searches are noticeable.

## Open questions

- When to return to this: when this site will need lists by ranges - neighbors, prices "from and to", events by dates.
- Whether the accuracy of the `limit` window is needed separately from comparisons. The most common case is "last 10 records": a new entry enters the window, and changing the entry far beyond the window now calculates all pages with the list.
- One segment per read or any number of comparison columns.

## Sources

- Measured by 21 November 2026 year, neighborly conversation: 1,000 movies, five neighbors ranked.
- Module R*Tree: [sqlite.org/rtree.html](https://sqlite.org/rtree.html).
