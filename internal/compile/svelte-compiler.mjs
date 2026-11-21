import { createInterface } from 'node:readline';

const wanted = '5.59.1';
const { compile, VERSION } = await import('svelte/compiler').catch((error) => {
  throw new Error(`install svelte@${wanted} in this Shugo project to compile .svelte islands`, { cause: error });
});
if (VERSION !== wanted) throw new Error(`Shugo islands use Svelte ${wanted}: this project has ${VERSION}`);

for await (const line of createInterface({ input: process.stdin })) {
  const { filename, source } = JSON.parse(line);
  let result;
  try {
    const options = { filename, css: 'external' };
    const client = compile(source, { ...options, generate: 'client' });
    const server = compile(source, { ...options, generate: 'server' });
    result = {
      client: client.js.code,
      server: server.js.code,
      css: client.css?.code ?? '',
      warnings: client.warnings.map(diagnostic),
    };
  } catch (error) {
    result = { error: diagnostic(error) };
  }
  process.stdout.write(JSON.stringify(result) + '\n');
}

function diagnostic({ message, start }) {
  return { message, line: start?.line, column: start?.column };
}
