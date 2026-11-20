# Tailwind styles

Install Tailwind in the project's `node_modules` before building:

```sh
cd examples/tailwind
npm install            # or bun install
../../bin/shugo dev .
```

The example includes the following:

- `styles/main.css` loads Tailwind, defines a custom color with `@theme`, and defines a custom rule with `@apply`.
- The markup and the strings in `components/Button.shugo` both contain classes. Tailwind finds them by scanning the project files as text.
- The generated stylesheet goes into `_build/assets/` and loads automatically on every page.
