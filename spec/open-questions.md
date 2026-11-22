# Open questions

Questions are numbered: answer the numbers. Where there is advice, yes - to accept it. Decisions taken without confirmation are collected in [Decisions awaiting approval](decisions.md); proposals that do not yet exist are in [docs/proposals](../docs/proposals/README.md).

### 1. 36–38 solutions and warning thresholds

The `src/site.html` shell, files from modules with `GET` and their export are made according to your answers. The details I chose are in the [36](decisions.md#36-the-srcsitehtml-shell), [37](decisions.md#37-files-not-in-html-module-with-get) and [38](decisions.md#38-exporting-module-files) solutions. Main: no `src/site.html` — takes the framework shell; `<html>`, `<head>` or `<body>` in the template is an assembly error; `GET` returns only a string, without `Response`; module without file type in name (`data.ts`) is an error.

Warning about a common component that shares almost nothing is also done. Thresholds I chose ([3](decisions.md#3-which-blocks-are-shared)): 100 shared blocks or more, and each is shown on average less than one and a half places.

**The Council:** approve.
