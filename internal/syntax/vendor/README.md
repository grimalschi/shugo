# Svelte compiler

`svelte-5.59.1.cjs` is the unmodified `package/compiler/index.js` distributed in [svelte@5.59.1 on npm](https://www.npmjs.com/package/svelte/v/5.59.1). Shugo embeds it to parse ordinary templates during builds on Bun or Node. See the [Svelte license](SVELTE-LICENSE.md) and [dependency notices](THIRD-PARTY.md).

When upgrading the pinned version, replace the bundle and notices from the upstream distribution, update the version expected by the parser, and run the template and SSR parity tests with the new version.

```sh
npm pack svelte@5.59.1
tar -xOf svelte-5.59.1.tgz package/compiler/index.js > svelte-5.59.1.cjs
sha256sum svelte-5.59.1.cjs
```

SHA-256 `dcdddbb651c802dfdc3e80b884793c96f653663b4091473b05ec4ddc89f90763`
