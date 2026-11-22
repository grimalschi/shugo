# Page shell: `app.html`

Proposal to take `<html>`, `<head>` and `<body>` from the root layout to a separate file, as `src/app.html` in SvelteKit. The author of the project asked 22 November 2026: "what about app.html? do site.html? or how." Nothing is done.

## Decisions

- The official name of the layout remains `+layout.shugo` (22 solution of November 2026 year).

## Like now

The root layout itself writes the entire document:

```svelte
<!-- src/pages/+layout.shugo -->
<script>
  let { children } = $props()
</script>

<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
  </head>
  <body>
    <main>{@render children()}</main>
  </body>
</html>
```

At the end of `<head>`, the server puts what page templates gave in `<svelte:head>`, and a link to styles.

## As in SvelteKit

Document is a regular HTML in `src/app.html` with places where SvelteKit inserts its:

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    %sveltekit.head%
  </head>
  <body>
    <div style="display: contents">%sveltekit.body%</div>
  </body>
</html>
```

The root `+layout.svelte` begins immediately with the contents of `<body>`. `<html>` in the SvelteKit component is not written.

## Proposal

`src/app.html` file is a shell of all pages:

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    %shugo.head%
  </head>
  <body>
    %shugo.body%
  </body>
</html>
```

- `%shugo.head%` is what page templates gave in `<svelte:head>`, in order of document and with the latest `<title>`, then a link to styles.
- `%shugo.body%` is a page in its layouts.
- The root layout is a common component, like `+layout.svelte` in SvelteKit: without `<html>`, `<head>` and `<body>`. The layout from SvelteKit is transferred to Shugo without editing, and back again.
- No file - takes the default shell, such as above.
- A layout or page that writes `<html>`, `<head>` or `<body>` is an assembly error with a hint about `app.html`.
- The shell is not a template: there are no expressions and no code in it. The framework stores it once on the site.

### `app.html` or `site.html`

| Name | For | Against |
| --- | --- | --- |
| `src/app.html` | same name and same role as in SvelteKit: agent will find and write it out of habit | “app” is an app, and Shugo makes websites |
| `src/site.html` | more honest in meaning | agent does not know about it and, getting used to SvelteKit, will write `app.html` |

**The Council:** `src/app.html` with `%shugo.head%` and `%shugo.body%`: for a name that the agent knows, it is worth paying with word inaccuracy.

### What else is needed

- Page language. In the shell of `<html lang="en">` one for the entire site. For a site in several languages, you need a place `%shugo.lang%` - it will fill [internationalization](i18n.md).
- `<body>` attributes for a separate page, such as a class. In SvelteKit, only the code in hooks is available for this; in Shugo, you can do nothing until you need it.

## Open questions

- Make the shell a separate file.
- Name: `app.html` or `site.html`.
