import { createInterface } from 'node:readline';
import { createRequire } from 'node:module';

const { compile, compileModule, VERSION } = await import('svelte/compiler').catch((error) => {
  throw new Error(`install svelte@${wanted} in this Shugo project to compile .island.svelte files`, { cause: error });
});
if (VERSION !== wanted) throw new Error(`Shugo islands use Svelte ${wanted}: this project has ${VERSION}`);

// Resolve the parser from Svelte so projects with isolated node_modules
// layouts use the same parser dependency as the pinned compiler.
const requireFromSvelte = createRequire(import.meta.resolve('svelte/compiler'));
const parseGenerated = (source) => requireFromSvelte('acorn').parse(source, {
  ecmaVersion: 'latest', sourceType: 'module', locations: true,
});
const serverMaps = new Map();

for await (const line of createInterface({ input: process.stdin })) {
  const { filename, source, lower, module } = JSON.parse(line);
  let result;
  try {
    if (lower) {
      result = { program: lowerIsland(source, parseGenerated) };
    } else if (module) {
      // A module with runes, .svelte.js, or .svelte.ts without its types.
      const client = compileModule(source, { filename, generate: 'client' });
      const server = compileModule(source, { filename, generate: 'server' });
      result = { client: client.js.code, server: server.js.code, css: '', warnings: client.warnings.map(diagnostic) };
    } else {
      const options = { filename, css: 'external' };
      const client = compile(source, { ...options, generate: 'client' });
      const server = compile(source, { ...options, generate: 'server' });
      serverMaps.set(filename, server.js.map);
      result = {
        client: client.js.code,
        server: server.js.code,
        css: client.css?.code ?? '',
        warnings: client.warnings.map(diagnostic),
      };
    }
  } catch (error) {
    result = { error: diagnostic(error), sourceMap: lower ? serverMaps.get(filename) : undefined };
  }
  process.stdout.write(JSON.stringify(result) + '\n');
}

function diagnostic({ message, start }) {
  return { message, line: start?.line, column: start?.column };
}
