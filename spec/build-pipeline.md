# Build pipeline

The project is being built in four stages. Each stage has its own entrance and its own result. The next stage uses only the results of the previous ones.

## Decisions

| Stage | What does | Result |
| --- | --- | --- |
| 1. Compilation | Reads the project source | JS files for downloading data, JS files for performing all expressions of templates, Go files for rantime |
| 2. Downloading data | Performs JS-files of download: they receive data at once for the whole project. Compares new data with past | Project data and what has changed in them |
| 3. Execution of expressions | Performs JS expression templates on data. If there is a cache or you know which blocks need to be updated, only they are counted | Expression values saved in files |
| 4. Rantime | Go server collects HTML from the results of the stages 1 and 3 | Pages on request or the entire site in the form of files |

- Comparison of new data with the past is done at the stage of data loading, most likely on Go.
- The dependency tree ("what part of the page what data is read") is built when compiling a project.
- After the first complete assembly, the project is updated on the dependency tree: only what is associated with the changed data is recalculated.
- Rantime always takes meanings from the saved and doesn't know how or when they were updated.

## From the prototype, pending approval

- Expression values are written in a flat list in the order of the template: value of each substitution, `1` or `0` for each condition, number of elements for each cycle. Unselected branch condition is not met.
- Expressions are performed by one long-lived process JS: starting a process is more expensive than calculating a page.
- The failed update does not change the saved state.

## Open questions

- The result of the compilation for runtime: Go-code pages (you need to build Go for each project) or a page program that performs one common runtime.
- Where to store the results of the stages: JSON-files or SQLite.
- Which JS runtime performs the 2 and 3 steps: Node, Bun or both.
- Where exactly is the boundary between 2 and 3: what exactly does the 2 stage transmit next.
- Teams for each stage and for the entire assembly.
- What lies in the build folder between stages.
