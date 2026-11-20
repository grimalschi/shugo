# Pictures

The document describes how the images are reduced and compressed.

## Decisions

- The images are compressed on the fly, the result is cached.

## From the prototype, pending approval

So far, nothing: there were no pictures in the prototype.

## Open questions

- What to compress: code inside the runtime or a separate service like imgproxy.
- Output formats: WebP, AVIF, JPEG.
- How the template sets the size and options of the picture.
- Where the cache is located and how the image addresses are arranged.
- What happens to images when you generate the entire site into files.
- Images whose address comes from data.
