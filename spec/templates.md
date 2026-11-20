# Page Templates

The document describes the `.shugo` file format: what parts the file is made of, how expressions are written, and control tags.

## Decisions

- The framework has its own file format - `.shugo`. In the first version, it is similar to Astro.
- The file consists of two parts:

  ```
  ---
  TS code that runs on the server and receives data
  ---
  page, layout or component layout
  ```

- So the file immediately divides the data acquisition code and the template code.
- The data code can return a list of dynamic pages `getStaticPaths`, as in Next.js and Astro.
- Expressions in markup are JS. Conditions, cycles and other control structures are their own tags.
- Differences from Astro:
  - Components are connected in the template itself, not in the data code: a component connection is not a data stream.
  - The parameters (props) are not readable in the JS code, that is, there are no entries of the type `const { name } = Astro.props`. They are declared structurally: the root element or system tags at the beginning of the template.

## From the prototype, pending approval

- Expression is any single expression of JS, including method calls and arrow functions. The compiler checks that the expression is one and finished.
- Expression depends only on its data and does not change them.
- A condition with a branch "other"; a cycle on the list with the element name, element number and key.
- The values are screened when inserted into the page. Addresses in links are checked separately.
- An error in expression shows a row and column in the template.

## Open questions

- Expressions: `{ }` or `{{ }}`. Attributes with expression.
- Names and type of control tags, write "otherwise" and "otherwise if".
- Data code form: top-level code, as in Astro, or a regular TS module with exports.
- How the pattern addresses the data: by name or through a common root.
- How components and function-helpers for expressions are connected.
- Output of the finished HTML without shielding.
- Output rules: `null`, `undefined`, `false`, numbers; attributes that are i.e., not.
- Checking types in templates.
