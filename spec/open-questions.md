# Open questions

The list is compiled during the first version. Questions are arranged according to importance: the first change the device of the framework, the last - details. Details of each - in the document by reference.

## Change the device

1. **Final version.** Collected from the fifth option: data in the database SQLite, lines remember what you read, log on triggers, pages without waiting, tapes inside records, server without a copy, export by changes. Approve it and the decisions that are made in it without confirmation. [Change tracking](change-tracking.md#final-version-waiting-for-approval)
   - **`props` from `getStaticPaths`.** With the data in the database, the page quickly gets its line itself, and `props` is forced to calculate all the route pages after changing the line it read. Do I leave them as in Astro. — [Dynamic pages](dynamic-pages.md#open-questions)
2. **Import components in code like Astro.** This is a departure from the first wish to keep the components connected in a template. — [6](decisions.md#6-components-and-functions-are-imported-in-data-code-as-in-astro)
3. **The rule of common blocks.** The component is general if it does not read the page and is called with simple parameters. Is this enough, or an explicit mark is needed. — [3](decisions.md#3-which-blocks-are-shared)
4. **Key in `<g-for>`.** Not entered: it does not give anything in the new scheme. — [4](decisions.md#4-the-key-in-g-for-is-not-entered)
5. **The program instead of the Go-code.** Whether the Go-code generation of pages is needed as acceleration. — [5](decisions.md#5-the-program-of-the-whole-project-is-one-file-the-rantime-is-common)

## Next steps

6. **Data adapter.** How data from any source gets into the project database: full load with comparison and point changes. Discussed, not done. — [Data adapter and data copy](data-adapter.md)
7. **Internationalization.** One page for all languages and strings in JSON. — [proposal](i18n.md)
8. **Pictures.** libvips, address type, other people's pictures. — [proposal](images.md)
9. **The scripts.** TypeScript and assembly of `<script>`. — [Page scripts](scripts.md)
10. **Styles inside the component.** Now `<style>` is valid for the whole site. — [Styles](styles.md)
11. **Checking Types in Templates** by `tsc` in the generated files. — [Page Templates](templates.md)
12. **Pages not in HTML:** `sitemap.xml`, `robots.txt`, RSS. — [File-based routing](routing.md)

## Work arrangement

13. **Several projects in one runtime process.** Provided by the device: only the storage file is needed in the runtime. How the server selects the project on request is not decided. — [Runtime](runtime.md)
14. **Sandbox for stage JS.** Who sets it up and what it is allowed to do.
15. **Failed update.** The question is postponed; while the storage remains the same. — [20](decisions.md#20-storage)
16. **Installation.** How `shugo` gets on the machine; the path of the Go module when published.
17. **SQLite driver:** without cgo as it is now, or faster on cgo. With libvips for cgo pictures you will need anyway. — [Storage](storage.md)

## Details

18. Reset the layout chain for a separate page. — [Layouts](layouts.md)
19. `/about` or `/about/`; `about/index.html` or `about.html` when exporting. — [10](decisions.md#10-routes)
20. `true`, `false`, lists and objects rules; attribute rules. — [12](decisions.md#12-what-outputs-expression-and-attributeexpression)
21. `<g-html>` or a record like `set:html`. — [13](decisions.md#13-ready-html--tag-g-html)
22. Slots recording. — [8](decisions.md#8-slots)
23. The name of the object `Shugo` and a set of its fields. — [7](decisions.md#7-object-shugo)
24. `src` folder, `tsconfig.json` file, `Shugo` types for editor. — [Project structure](project-structure.md)
25. Compression of answers and `ETag`. — [Runtime](runtime.md)
26. The language of the compiler messages. — [22](decisions.md#22-message-language)
27. `shugo init` team with project example and agent memo. — [Command line](cli.md)
28. Check how agents write on `.shugo`. Done once, see [24](decisions.md#24-verification-on-agent). Need other tasks and other models.
