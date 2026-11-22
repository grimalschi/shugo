# Pictures

Proposal on how pictures are reduced and compressed. While this is not: pictures from `public/` are given as is.

## Decisions

- The images are compressed on the fly, the result is cached.
- The compression lives inside the runtime and is built on libvips or another modern supported library.
- The output format is WebP, AVIF or JPEG, depending on what the browser supports.
- When generating the entire site into files, all images are compressed in advance and saved.
- **Two modes** (Decision of the author 22 November 2026 years): built-in compression, where WebP is the recommended format, and, if desired, libvips, if it is installed in the system or associated with ShugoIt can be specified and used for acceleration.
- **Format** (22 November 2026 solution): defaults to WebP only, in normal `<img>`; `<picture>` with AVIF - when libvips is enabled. No selection for the `Accept` header.

## Measurements

Photo 4000×3000 (12 megapixels) in JPEG is a difficult case: smaller images are often put on sites, and time grows with the number of pixels. Widths at the output — 400, 800 and 1600.

### Built-in compression: what can be done without cgo

One read, reduce, encode operation, one kernel, median:

| Step | Than | 400 | 800 | 1600 |
| --- | --- | ---: | ---: | ---: |
| Read JPEG | Go Standard Library | 187 ms | 187 ms | 187 ms |
| Zoom Out | `rez`, Lanczos-3 | 18 ms | 21 ms | 28 ms |
| WebP, quality 80 | `deepteams/webp`, pure Go | 33 ms | 132 ms | 484 ms |
| WebP, quality 80 | `gen2brain/webp`: libwebp, assembled in WASM | 68 ms | 266 ms | 1,012 ms |
| JPEG, quality 80 | Go Standard Library | 3 ms | 14 ms | 53 ms |
| AVIF, quality 63 | `gen2brain/avif`: libavif in WASM, speed 8 | 181 ms | 619 ms | 2,253 ms |

The whole operation in the new process: JPEG  `rez` → WebP `deepteams` — 261 / 405 / 905 ms and 47–63 MB memory. With JPEG output — 215 / 227 / 273 ms.

- **WebP without CGO is a working option.** `deepteams/webp` gives a file no larger than libwebp (19.6 KB vs 20.5 KB for width 400), at a quality slightly lower (SSIM 0.963 vs 0.972). It is twice as fast as libwebp in WASM.
- **Reduction is not the main expense**, if you take `rez`: 18–28 ms. Standard `x/image/draw` spends on the same 300–440 ms.
- **The main expense on small widths is JPEG reading**: standard library can not reduce when reading and reads all 12 megapixels. libvips can, so on the width of 400 it is fastest.
- **AVIF without libvips slow**: seconds on a large width. AVIF on the pure Go is not: the only library found, `goavif`, gives a corrupted picture.

### libvips

| One operation, one core | 400 | 800 | 1600 |
| --- | ---: | ---: | ---: |
| WebP, libvips within the process (cgo) | 59 ms | 108 ms | 278 ms |
| WebP, `vipsthumbnail` team | 126 ms | 168 ms | 358 ms |
| AVIF, libvips within the process | 77 ms | 162 ms | 396 ms |
| JPEG, libvips within the process | 49 ms | 57 ms | 93 ms |

Starting the `vipsthumbnail` team alone costs around 54 ms.

### Compression in advance, 8 streams

Picture per second; each three widths:

| Than | WebP | JPEG | AVIF, WebP and JPEG together |
| --- | ---: | ---: | ---: |
| Built-in: `deepteams/webp`, standard JPEG, AVIF in WASM | 3.7 | 13.3 | 0.6 |
| `vipsthumbnail` Team | 7.0 | 11.5 | 2.3 |
| libvips within the process | 9.5 | 20.8 | 3.3 |
| libwebp system loaded without cgo | 7.9 | — | — |

- libvips faster built-in compression of WebP in 2–2.5 times, full set with AVIF — in 4–6 times.
- The `vipsthumbnail` command is almost as fast as libvips inside the process, when there are many pictures and they are compressed in parallel.
- One operation takes 47–63 MB memory in built-in mode and up to 400 MB for AVIF with width 1600. The number of simultaneous operations should be limited.
- AVIF with quality 63 gives about the same quality as JPEG 80 and WebP 79, with smaller size: 16, 25 and 20 KB for width 400.

## Proposal

### Two modes

| | Built-in | libvips |
| --- | --- | --- |
| What you need | nothing: everything inside `shugo` | libvips in the system |
| Formats | WebP Recommended, JPEG | AVIF, WebP, JPEG |
| WebP, 8 streams | 3.7 pictures per second | 7–9.5 |
| Reading the Big JPEGs | read all pixels | decreased Reading |

Built-in mode always works. `shugo` remains one file without dependencies and without cgo: JPEG and PNG reading is the standard library of Go, reduction is `rez`, WebP is `deepteams/webp`, JPEG is the standard library.

libvips is enabled by setting. `shugo` itself does not require it.

### How `shugo` gets libvips without cgo

1. **`vipsthumbnail` Team** it is put together with libvips (`apt install libvips-tools`, `brew install vips`). `shugo` launches it on every picture or on a pack of pictures. The price is about 54 ms per run; when compressed in advance, it is almost invisible (7.0 pictures per second vs. 9.5 inside the process). I suggest you start with it: it is simple and measured.
2. **Loading libvips during operation**, without cgo, as `gen2brain/webp` loads system libwebp. There is no process start. But libvips functions accept a variable number of arguments; they will have to be called through the common libvips operation interface. Not checked and not measured.
3. **`shugo` build with libvips via cgo** is for those who collect `shugo` themselves. The fastest way, but this is another `shugo` file, not the one that is being shared by everyone.

The mode is chosen explicitly, without guessing. The two modes encode differently, and the bytes of the pictures are different. If `shugo` had taken libvips itself when it found it, the same project on two machines would have given different files, and export by changes would have rewritten pictures at each change of machine.

### Configuration

```json
{
  "images": {
    "engine": "vips",
    "formats": ["avif", "webp"],
    "quality": 80
  }
}
```

| Field | Default | What does it mean? |
| --- | --- | --- |
| `engine` | `"builtin"` | `"builtin"` — built-in compression, `"vips"` — libvips |
| `vips` | is looking for `PATH` | where is `vipsthumbnail` if it is not in `PATH` |
| `formats` | `["webp"]` | image formats in order of preference; `"avif"` — only with libvips |
| `quality` | 80 | quality WebP and JPEG; AVIF gets equal in appearance — 63 |

`"engine": "vips"` without libvips on the machine is a startup error, not a quiet transition to built-in mode.

### Image Address

```svelte
<img src="/_img/w800/photos/jug.jpg.webp" alt="Pitcher" />
```

- `/_img/` is a processing feature, `w800` is a parameter, then the file path in `public/`, and lastly, the format: `.webp`, `.avif` or `.jpg`.
- Options: `w` and `h` are size, `q` is quality.
- The format is in the address itself, so the address works equally on the `shugo serve` server and on any static hosting: the hosting gives the file with the type of the latest extension.
- The result is cached under the name from the source file hash, parameters and mode. The repeated request gives the finished file. Changing the mode or its version gives other names - the pictures are compressed again.
- In both modes, the addresses and file set are the same: only bytes change.

The format "depending on what the browser supports" is chosen by the browser itself according to `<picture>`:

```svelte
<picture>
  <source srcset="/_img/w800/photos/jug.jpg.avif" type="image/avif" />
  <img src="/_img/w800/photos/jug.jpg.webp" alt="Pitcher" width="800" height="600" />
</picture>
```

The choice of the header `Accept` is not needed: on static hosting it is not, and caches in front of the server it is often confused. WebP understand all browsers that are currently in use, so in built-in mode, one `<img>` is enough.

To not collect addresses with your hands, you need a function `image("/photos/jug.jpg", { width: 800 })` from module `shugo` The finished component that it produces `<picture>` with formats from settings, `srcset` for several widths, `width` and `height` against page jumps and `loading="lazy"`B. SvelteKit He does the same `<enhanced:img>`. It is not a component but a tag that plugin `@sveltejs/enhanced-img` rewrites in source to compiler Sveltewithout such step Svelte Detox `<enhanced:img>` On the page as is.

### How to choose a format: options

| Option | How | For | Against |
| --- | --- | --- | --- |
| WebP only | one address `…/jug.jpg.webp` in regular `<img>` | the easiest; equally on the server and on any hosting; WebP understand all browsers that are currently in use | AVIF on 15–20% is smaller with the same quality (measurements above) and it is not |
| `<picture>` with AVIF and WebP | the browser takes the first format that understands | the lightest pictures | AVIF only with libvips; marking longer; component needed |
| by the name `Accept` | one address, the server chooses the format | the markup does not change | does not work on static hosting; caches before the server confuse answers |

Solved as follows: only WebP by default - in both modes; `<picture>` with AVIF - when the settings have `"avif"`, and it is only with libvips. The choice for `Accept` - no.

### Where is the cache: options

Compression of photos is tenths of a second, thousands of photos are minutes. The cache stores the finished files under the name from the source file hash, parameters and mode.

| Option | For | Against |
| --- | --- | --- |
| `_build/images/` | next to everything Shugo writes | disappears with `_build/`, and everything is compressed again |
| Own project folder outside of `_build/`, such as `.shugo-cache/images/`, not in git | is undergoing cleaning of `_build/`; you can see how much space it takes | each copy of the project has its own cache |
| Shared user folder, `~/.cache/shugo/images/` | common to all projects and copies on the machine | grows without a host; on the build server it must be stored separately |
| In Storage SQLite | one file | storage is inflated with pictures that the page server does not need |

**The Council:** has its own project folder outside of `_build/`, and setting `"cache"` to `shugo.json` takes it anywhere — for example, in the build server cache. Details — in a separate [the Cash Offer](image-cache.md).

### Export

- Export finds `/_img/…` addresses in HTML pages that it writes: in `src`, `srcset` and `href` attributes. Know them in advance of the runtime is nowhere to be found: with the 22 November 2026 year, the records are the finished HTML by Svelte, not the values that the runtime itself put into the attributes it knows.
- For each address, take a ready file from the cache or compress the picture and write it next to the pages.
- Exporting by changes only writes pictures that are not yet in the folder.

### On the fly

`shugo serve` compresses the picture at the first request of its address and puts it in a cache. At the same time, there are no more operations than cores: each takes tens and hundreds of megabytes of memory. While the operation is underway, other requests of the same address are waiting for it, not starting their own.

### Images whose address comes from data

Rantime compresses only project files and does not go to the network: otherwise any page could make the server download other people's addresses. Foreign pictures download [data Adapter](data-adapter.md) along with data, to the project folder; then they are the same as their own.

### imgproxy

The third way remains possible: then Shugo does not compress the pictures, but only builds signed addresses. In the free version of imgproxy there is no cache of results, it is needed outside.

## Open questions

- How `shugo` gets libvips: start with the `vipsthumbnail` command as suggested, or download the library immediately while running.
- Address type: `/_img/w800/…` prefix as here, or file name ending; format last extension (`jug.jpg.webp`) or instead of original (`jug.webp`).
- The finished image component: its Svelte component, which gives Shugo, or a SvelteKit record `<enhanced:img>` — then Shugo would rewrite it in the source to the Svelte compiler, as does the SvelteKit plugin.
- The cache is [separate offer](image-cache.md).

## Sources

Measurements are made by 20 November 2026 on this machine: Xeon E3-1270 v6, 4 core and 8 threads, Go 1.27.4, libvips 8.16.1. Libraries: `deepteams/webp` 1.2.9, `gen2brain/webp` 0.6.4, `gen2brain/avif` 0.6.0, `gen2brain/jpegli` 0.4.2, `rez`, `goavif` 0.1.0. The measurement programs and raw results are in the topology repository, folder `research/images`; summary tables are `results/pooled-tables.txt`.
