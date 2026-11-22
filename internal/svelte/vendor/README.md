# Svelte

Shugo ships Svelte 5.59.1, so a project has pages and islands without installing it. See the [Svelte license](SVELTE-LICENSE.md) and [dependency notices](THIRD-PARTY.md).

- `compiler-5.59.1.cjs` is the unmodified `package/compiler/index.js` of [svelte@5.59.1 on npm](https://www.npmjs.com/package/svelte/v/5.59.1). A build runs it on Bun or Node to compile the templates of a project for the server, and islands for the browser and the server.
- `runtime/` is the server runtime of the same package: the modules a page may import (`svelte`, `svelte/internal/server`, `svelte/store`, …), bundled by esbuild from the package's `src/` for a server in production. Each module is one file named after it, `svelte/internal/server` is `internal-server.js`; the code they share is in `chunk-*.js`.
- `runtime-browser/` is the browser runtime, made the same way: the modules an island may import in the browser (`svelte`, `svelte/internal/client`, `svelte/store`, …).
- `acorn.cjs` is the unmodified `dist/acorn.js` of acorn 8.19.0, the parser of JavaScript that Svelte depends on. Shugo reads with it the server code of an island bound to a request.

When upgrading the pinned version, replace all of them from the upstream distribution, update the version in `../svelte.go`, and run the tests.

```sh
npm pack svelte@5.59.1
tar -xOf svelte-5.59.1.tgz package/compiler/index.js > compiler-5.59.1.cjs
sha256sum compiler-5.59.1.cjs

npm install --prefix /tmp/svelte-vendor svelte@5.59.1
go run internal/svelte/vendor/make.go /tmp/svelte-vendor   # from the root of the repository: runtime/, runtime-browser/ and acorn.cjs
```

SHA-256 of the compiler: `dcdddbb651c802dfdc3e80b884793c96f653663b4091473b05ec4ddc89f90763`
