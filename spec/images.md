# Pictures

The document describes how the pictures are reduced and compressed. In the first version, this is not: pictures from `public/` are given as is. Below is what is decided, measurements and sentence.

## Decisions

- The images are compressed on the fly, the result is cached.
- The compression lives inside the runtime and is built on libvips or another modern supported library.
- The output format is WebP, AVIF or JPEG, depending on what the browser supports.
- When generating the entire site into files, all images are compressed in advance and saved.

## Measurements

Photo 4000×3000 in JPEG, one core, time per operation "read, reduce, encode" for width 400, 800 and 1600:

| Method | JPEG | WebP | AVIF |
| --- | ---: | ---: | ---: |
| Go without cgo | 523 / 501 / 690 ms | 551 / 622 / 1,204 ms | 694 / 1,051 / 2,668 ms |
| libvips | 49 / 57 / 93 ms | 59 / 108 / 278 ms | 77 / 162 / 396 ms |

Compression in advance, 8 streams, source pictures per second (each three widths):

| Method | JPEG | WebP | AVIF | All three formats |
| --- | ---: | ---: | ---: | ---: |
| Go without cgo | 2.9 | 2.1 | 0.67 | 0.54 |
| libvips | 20.8 | 9.5 | 6.45 | 3.32 |

- libvips is faster in 3–10 once per operation and in 6 once per full set. This confirms the choice.
- The standard JPEG decoder in Go does not know how to reduce when reading: even for previews, it spends 187 ms on all 12 megapixels.
- One operation takes 75–245 MB memory, AVIF width 1600 - about 390 MB. The number of simultaneous operations should be limited.
- AVIF with quality 63 gives about the same quality as JPEG 80 and WebP 79, with smaller size: 16, 25 and 20 KB for width 400.

## Proposal

### Library

libvips connects via cgo. Supported bindings for Go two: `govips` (version 2.20, October 2026) and `vipsgen` (2.1, November 2026). `bimg` not updated with 2022 years.

Consequence: runtime with pictures can not be collected in one file without dependencies - libvips is needed on the machine. I suggest collecting two variants of `shugo`: with pictures and without. Without pictures, everything that is now remains.

### Image Address

Your idea is to recognize the prefix or the end of the address. I suggest the prefix:

```
<img src="/_img/w800/photos/jug.jpg" alt="Pitcher">
```

- `/_img/` is a processing feature, `w800` is parameters, then the file path in `public/`.
- Options: `w` and `h` are size, `q` is quality.
- The format selects the rank by the header of the `Accept` browser: AVIF, otherwise WebP, otherwise JPEG.
- The result is put in `_build/images/` under the name from the hash of the source file and parameters. The repeated request gives the finished file.

To not collect the address with your hands, you need a function: `image("/photos/jug.jpg", { width: 800 })`.

### Export

Static hosting does not choose the format by title. Therefore, when exporting:

- rantime finds `/_img/…` addresses in the collected pages: it knows which attributes contain addresses;
- aVIF, WebP and JPEG are prepared for everyone;
- in the page you need to output `<picture>` with three sources - this will require a ready component.

### Images whose address comes from data

| | For | Against |
| --- | --- | --- |
| Compress them too | one way for all pictures | rantime should download other people's files: this is a delay in the first display and the ability to make the server walk to other people's addresses |
| Download at the calculation stage to the project folder | rantime remains without access to the network | the calculation becomes longer, files need to be stored |

I propose a second way: the runtime processes only project files, and other people’s pictures the data code downloads itself, according to the list of allowed domains.

### imgproxy

The second way remains possible: then rantime only builds signed addresses. In the free version of imgproxy there is no cache of results, it is needed outside.

## Open questions

- Two assembly options `shugo`, with libvips and without, or libvips is required always.
- Address type: `/_img/w800/…` prefix as here, or end of file name.
- Do you need a ready-made image component that displays `<picture>`, dimensions and `loading="lazy"` itself?
- Foreign pictures: download when calculating or processing in rantime.
- Where the cache is located: `_build/images/` is deleted along with `_build`. Perhaps the image cache should be kept separately.

## Sources

Measurements are made 20 November 2026 years on this machine (Xeon E3-1270 v6, 4 Cores), libvips 8.16.1, Go 1.27.4. The measurement programs and raw results are in the repository of the top, in the folder `research/images`.
