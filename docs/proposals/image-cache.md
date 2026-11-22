# Image cache

Proposal about where and how compressed pictures are stored to compress each once. Part of the proposal about [pictures](images.md); the author of the project asked to write it separately 22 November 2026 year. None of this has yet been done.

## Decisions

Nothing yet.

## Why

Compression is expensive: the photo of 12 megapixels in built-in mode is compressed in WebP for 0.26–0.9 s width, with libvips for tenths of a second. A thousand photos in three widths is a minute. The cache stores the finished files, and the picture is compressed again only when it has changed itself, its parameters or the method of compression.

## Proposal

### Where it lies

Its project folder is outside of `_build/`: `.shugo-cache/images/`. It is not in git like `_build/`, but is experiencing its cleaning and complete reassembly.

```json
{ "images": { "cache": "/var/cache/shugo/shop" } }
```

Setting up `cache` in `shugo.json` takes the folder anywhere: for example, to the build server cache that is saved between builds, or to one folder for multiple copies of the project on the machine.

Why not other places:

| Option | Why not? |
| --- | --- |
| `_build/images/` | disappears with `_build/`, and everything is compressed again |
| Shared user folder, `~/.cache/shugo/` | grows without a host: it is unclear what project is needed; on the build server, it still needs to be stored separately |
| Storage SQLite | storage is inflated with pictures that the page server does not need |

### What are the files called

The file name is a hash of what the result depends on: the content of the original image, parameters (`w`, `h`, `q`, format), compression library mode and versions.

```
.shugo-cache/images/3f/3fa9c1…e2.webp
```

- Changed the picture, settings or mode - got another hash and another file. There are no outdated files in the cache: there are unnecessary.
- Renamed the image — hash content the former, no need to compress.
- The first two hash letters are a subfolder so that hundreds of thousands of files do not lie in one folder.

### How to write

The file is first written under a temporary name and then renamed. So `shugo serve`, `shugo update` and `shugo export`, which work simultaneously, never see the file half. If two processes compress one image at a time, renaming the latter will simply replace the same file.

Within one process, one operation compresses one image: other requests of the same address are waiting for it. At the same time, there are no more operations than cores: each takes tens and hundreds of megabytes of memory.

### How to clean

Unnecessary files remain after the picture has been replaced or removed from the pages. The `shugo images clean` command removes files from the cache that no page on the site links to: `/_img/…` addresses in pages know export, and they show which files are needed. The build itself does not remove anything: the cache can be shared by several copies of the project.

### How much space

The photo is in three widths, 400, 800 and 1600, in WebP about 20, 70 and 250 KB, that is, about 340 KB per original photo; with AVIF about 280 KB.

## Open questions

- Default folder name: `.shugo-cache/` or other.
- Whether the cleanup command is needed or if it is enough that the cache can simply be removed.
