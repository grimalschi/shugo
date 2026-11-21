import { createInterface } from "node:readline";
import { createWriteStream } from "node:fs";

// fd 3 belongs to Go. Component logs go to the normal output streams.
export async function serve(islands) {
  const output = createWriteStream(null, { fd: 3 });
  for await (const line of createInterface({ input: process.stdin, crlfDelay: Infinity })) {
    let response;
    try {
      const { id, props } = JSON.parse(line);
      const island = islands.get(id);
      if (!island) throw new Error(`unknown island ${id}`);
      const { body, head } = island.render(props);
      response = { body, head };
    } catch (error) {
      response = { error: String(error) };
    }
    await new Promise((resolve, reject) => {
      output.write(JSON.stringify(response) + "\n", (error) => error ? reject(error) : resolve());
    });
  }
  output.end();
}
