# `query` and `page` from the `shugo` module

Proposal to replace the global object `Shugo` import from the module `shugo`, and address and page parameters give the object `page`, as in SvelteKit. None of this has yet been done; the author of the project asked to write down the proposal 22 November 2026 year.

## Decisions

Nothing yet.

## Like now

In `<script>` and in the layout, you can see the object `Shugo`, which no one imports:

```svelte
<script>
  const product = Shugo.query("products").where({ slug: Shugo.params.slug }).first()
  const page = Shugo.paginate(Shugo.query("reviews").where({ product: product.id }), { pageSize: 10 })
</script>

<p>{Shugo.url.pathname}: {product.title}</p>
```

| Field | What is it |
| --- | --- |
| `Shugo.query(table)` | question to the project database |
| `Shugo.params` | page Address Settings |
| `Shugo.url` | address of the page, `URL` |
| `Shugo.paginate(request or array, ...)` | part of the list for the page with the number |

`<script module>` does not have it: `getStaticPaths` gets `query` and `paginate` by argument.

## As in SvelteKit

The address and page parameters are given by `page` of `$app/state` ([documentation](https://svelte.dev/docs/kit/$app-state#page)):

```svelte
<script>
  import { page } from '$app/state';
</script>

<p>Currently at {page.url.pathname}</p>
```

| Field `page` | What is it |
| --- | --- |
| `url` | address of the page, `URL` |
| `params` | route parameters: `{ slug: "jug" }` for `/blog/[slug]` |
| `route.id` | route: `/blog/[slug]` |
| `status` | response Code |
| `error` | error if it is an error page |
| `data` | that returned the functions of `load` pages and layouts |
| `form` | data after sending the form |
| `state` | the state that the page passed when you clicked |

You can read `page` on the server only while the page is showing, and not in `load`. Data in SvelteKit does not come with a request from the template, but from `load`, so there is no `query` of its own.

## Proposal

The `shugo` module gives what now lies in the `Shugo`, and the `page` with SvelteKit fields that make sense for a static page:

```svelte
<script>
  import { query, paginate, page } from "shugo"

  const product = query("products").where({ slug: page.params.slug }).first()
  const reviews = paginate(query("reviews").where({ product: product.id }), { pageSize: 10 })
</script>

<p>{page.url.pathname}: {product.title}</p>
```

| Name | What is it |
| --- | --- |
| `query(table)` | question to the project database |
| `paginate(request or array, { pageSize })` | part of the list for the page with the number |
| `page.url` | address of the page, `URL` |
| `page.params` | address Settings |
| `page.route.id` | route: `/products/[slug]` |
| `page.status` | 200, and the page `404.shugo` — 404 |

`data`, `form`, `state` and `error` are not needed: `load`, no forms and transitions in the browser.

- In `getStaticPaths`, the same `query` and `paginate` are from import, not argument: one record everywhere.
- Import the same in any file: in a page, layout, component, in a module from `src/lib/`. The request, as now, belongs to the page or component that is being calculated; outside of calculation, it is an error with an explanation.
- `Shugo.*` after the transition — error with a hint of what import to write.

### For and Against

| For | Against |
| --- | --- |
| Explicit import, like everything else: you can see where it came from; the solution of “no magic” already has | One more line in each file that asks the database |
| The editor knows the types from the module without its global name ads | The `shugo` module should be visible to the editor: package or `tsconfig.json`, which will make `shugo init` |
| Agent who knows SvelteKit writes `page.params` out of habit | `page` in SvelteKit comes from `$app/state`, and here from `shugo`: import is still different |

### How to do it

The compiler already collects all the project code by one assembler. The `shugo` module is a built-in assembly module, as is now `shugo:runtime`: `query`, `paginate` and `page` take what is being calculated now from the runtime. There is no need for a separate npm package to work; for types in the editor, the ad file that `shugo init` puts in.

## Open questions

- Moving to Import.
- Whether to leave `Shugo` next to the import at the time of transition or immediately make it a mistake. Tip - immediately: it is easier for the agent when the record is one.
