# npm packages

Shugo uses the same package layout as esbuild.

- `shugo` provides the `shugo` command, the `shugo` module types in `index.d.ts`, and the `tsconfig.json` projects can extend. It pins Svelte, `svelte-check`, and TypeScript. Builds use Svelte's compiler and runtime. The `shugo check` command uses the paired `svelte-check` and TypeScript versions, and editors load the Svelte and rune types through `ambient.d.ts`. The command passes the dependency paths to the executable through `SHUGO_SVELTE` and `SHUGO_SVELTE_CHECK`.
- `@shugo/<platform>-<arch>` provides the executable for each target. Targets are `linux-x64`, `linux-arm64`, `darwin-x64`, `darwin-arm64`, `win32-x64`, and `win32-arm64`. The platform packages are optional dependencies with `os` and `cpu` fields. The package manager installs the matching binary, and the command forwards its arguments to it.

The source `shugo` package uses version `0.0.0`. The `make.go` script creates distribution packages with the specified version.

## Build

Run from the repository root.

```sh
go run npm/make.go -version 0.1.0                    # all targets, about 90 seconds
go run npm/make.go -version 0.1.0 -only linux-x64    # one target
```

Each package is written under `npm/dist/`, including `npm/dist/shugo` and directories such as `npm/dist/@shugo/linux-x64`. The executable builds with cgo disabled.

## Publish

Publish the platform packages first, then the `shugo` package that references them.

```sh
for package in npm/dist/@shugo/*; do (cd "$package" && npm publish --access public); done
(cd npm/dist/shugo && npm publish)
```

The first release needs an npm organization for the `@shugo` scope and a project license to replace `UNLICENSED`.

## Verify locally

Pack the distributions and use an override for the local platform tarball.

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

A Linux test on November 23, 2026 installed the CLI, executable, and Svelte and built a site. TypeScript 5.9 and 7.0 and `svelte-check` caught the deliberately invalid uses of `query`, `page`, and `svelte/store` with a one-line `tsconfig.json`. `TestTypesOfTheModuleShugo` compares `index.d.ts` with the exports of the `shugo` module. `TestPackageCommand` checks the CLI wrapper on Node and Bun.
