# Open questions

Questions are numbered: answer the numbers. Where there is advice, yes - to accept it. Decisions taken without confirmation are collected in [Decisions awaiting approval](decisions.md); proposals that do not yet exist are in [docs/proposals](../docs/proposals/README.md).

### 1. Which component is common

This is a rule about how many times Shugo computes and stores a component. The normal component is calculated on each page where it stands: the cap on 100,000 pages is 100,000 times. The total is calculated once per site for each set of parameters, and pages store only a link to it.

The component becomes common itself, without markings, if both conditions are met:

1. It does not depend on the page: neither he nor the modules he imports take `page` or `paginate` from the `shugo` module. This is visible from the imports, the code does not need to run.
2. It was called without parameters or with simple values: strings, numbers, `true`, `false`, `null`.

| Challenge | How many times is calculated |
| --- | --- |
| `<Header />` | once for the entire site |
| `<ProductCard id="p5" />` | once per item, no matter how many pages it shows |
| `<ProductCard item={product} />` | once per page because the prop is an object |
| `<Menu />`, which highlights the current item on `page.url` | on each page again |

Snippets passed inside do not interfere. Therefore, the layout is common, although the page it receives with the snippet `children`.

Profit: the product has changed - one card is calculated, not all the pages where it stands. Price: A component that is called with very different values, for example `<Price value={n} />`, gives a record for each number, and almost every record shows one page - there is no need to store it separately.

**The Council:** approve the rule ([3](decisions.md#3-which-blocks-are-shared)) and warn when assembling when the common component shares almost nothing: it has many records, and each shows one page on average.

### 2. Solutions 36–38

The `src/site.html` shell, files from modules with `GET` and their export are made according to your answers. The details I chose are in the [36](decisions.md#36-the-srcsitehtml-shell), [37](decisions.md#37-files-not-in-html-module-with-get) and [38](decisions.md#38-exporting-module-files) solutions. Main: no `src/site.html` — takes the framework shell; `<html>`, `<head>` or `<body>` in the template is an assembly error; `GET` returns only a string, without `Response`; module without file type in name (`data.ts`) is an error.

**The Council:** approve.
