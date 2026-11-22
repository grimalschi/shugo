import { readSync, writeSync } from "node:fs";

// A frame starts with the size of its kind and payload in four bytes.
const maxFrame = 1 << 30;

// The same build wire is available while modules are loading and while
// run() is answering requests. Reads are synchronous and never read ahead.
export function buildChannel() {
  const input = Number(process.env.SHUGO_IN);
  const output = Number(process.env.SHUGO_OUT);
  if (!Number.isInteger(input) || !Number.isInteger(output) || input < 3 || output < 3 || input === output) return null;

  function send(kind, text) {
    const body = Buffer.from(kind + text);
    if (body.length > maxFrame) throw new Error("Shugo message is too large");
    const head = Buffer.allocUnsafe(4);
    head.writeUInt32LE(body.length, 0);
    for (const buffer of [head, body]) {
      for (let sent = 0; sent < buffer.length; ) {
        const count = writeSync(output, buffer, sent, buffer.length - sent);
        if (count === 0) throw new Error("the Shugo command stopped accepting messages");
        sent += count;
      }
    }
  }

  function fill(buffer) {
    for (let got = 0; got < buffer.length; ) {
      const count = readSync(input, buffer, got, buffer.length - got, null);
      if (count === 0) return false;
      got += count;
    }
    return true;
  }

  function receive() {
    const head = Buffer.allocUnsafe(4);
    if (!fill(head)) return null;
    const size = head.readUInt32LE(0);
    if (size < 1 || size > maxFrame) throw new Error("invalid Shugo message size");
    const body = Buffer.allocUnsafe(size);
    return fill(body) ? { kind: String.fromCharCode(body[0]), text: body.toString("utf8", 1) } : null;
  }

  return { send, receive };
}
