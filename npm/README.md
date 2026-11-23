# npm Packages

Shugo is put in the project package npm, as esbuild:

- `shugo` What the project puts forward: `bun add shugo`. Command `shugo` (`bin/shugo.js`Types of Modules `shugo` (`index.d.ts`), settings TypeScript the Project (`tsconfig.json`He inherits `tsconfig.json` Project and Dependency `svelte` The same version as the program: editor and `svelte-check` Take Types Svelte Rune and Rune`ambient.d.ts`).
- `@shugo/<system>-<cPU>` is a ready program of `shugo` for one system: `linux-x64`, `linux-arm64`, `darwin-x64`, `darwin-arm64`, `win32-x64`, `win32-arm64`. They are optional dependencies of the `shugo` package with `os` and `cpu` fields, so the package manager puts only the one that suits the machine. The `shugo` package command finds it and runs it with the same arguments. Each package contains a Svelte license and information about the code it is made of: the program carries Svelte inside.

Here lies the `shugo` package with the version of `0.0.0`; packages for systems make `make.go`.

## Collect

From the Repository Root:

```sh
go run npm/make.go -version 0.1.0                    # all systems, about one and a half minutes
go run npm/make.go -version 0.1.0 -only linux-x64    # one
```

Packages are placed in `npm/dist/`, each with its own folder: `npm/dist/shugo` with the affixed version and `npm/dist/@shugo/linux-x64` and others. The program is assembled without cgo, so it is suitable for any system with any.

## Publish

First packages under the system, then `shugo`, which refers to them:

```sh
for package in npm/dist/@shugo/*; do (cd "$package" && npm publish --access public); done
(cd npm/dist/shugo && npm publish)
```

Prior to the first publication, `shugo` is listed in npm for `@shugo/…` names and Shugo license instead of `UNLICENSED`.

## Check without posting

Packages compressed into archives are put in the project as published; packages under the system place in `overrides`:

```sh
for package in npm/dist/shugo npm/dist/@shugo/*; do (cd "$package" && bun pm pack --destination /tmp/packs); done
```

```json
{
  "overrides": {
    "@shugo/linux-x64": "file:/tmp/packs/shugo-linux-x64-0.1.0.tgz"
  }
}
```

```sh
bun add shugo@file:/tmp/packs/shugo-0.1.0.tgz
bunx shugo build
```

So checked 23 November 2026 Years on Linux: `bun add` We have a team, program and Svelte, `bunx shugo build` I have a website, and TypeScript 5.9 and 7.0 and `svelte-check` S `tsconfig.json` In one line found errors in `query`, `page` and `svelte/store`, and only them. Repository test `TestTypesOfTheModuleShugo` Verify `index.d.ts` Exporting the module `shugo`, and `TestPackageCommand` Checks the command from the package for Node and Bun.
