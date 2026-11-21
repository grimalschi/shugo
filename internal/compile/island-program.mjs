// Lower the render operations of generated Svelte 5 SSR code. This is a
// checked subset of its server output: the Go side reads a JSON view model
// and writes markup and Svelte's own hydration markers.
export function lowerIsland(code, parse) {
  const imports = new Map();
  const scope = new Map();
  let component;
  let loopID = 0;

  for (const node of parse(code).body) {
    if (node.type === 'ImportDeclaration') {
      for (const name of node.specifiers) imports.set(name.local.name, [node.source.value, name.imported?.name ?? name.type]);
    } else if (node.type === 'ExportDefaultDeclaration') {
      component = node.declaration;
    } else if (node.type === 'ExportNamedDeclaration' && isFlag(node.declaration)) {
      scope.set('shugoSSR', { const: true });
    } else {
      reject(node, 'module code runs outside the island; move request-time data into props');
    }
  }
  if (component?.type !== 'FunctionDeclaration' || component.async || component.generator || component.params.length > 2) {
    reject(component, 'expected a synchronous Svelte server component');
  }
  const [renderer, props] = component.params;
  if (renderer?.type !== 'Identifier' || (props && props.type !== 'Identifier')) {
    reject(component, 'expected Svelte renderer and props parameters');
  }
  if (props) scope.set(props.name, { path: ['$'] });
  return statements(component.body.body, scope, renderer.name);

  function statements(nodes, env, renderer) {
    const ops = [];
    for (const node of nodes) {
      switch (node.type) {
        case 'VariableDeclaration':
          if (node.kind === 'var') reject(node, 'use block-scoped let or const declarations in request-rendered islands');
          for (const d of node.declarations) declaration(d, env);
          break;
        case 'FunctionDeclaration':
          // Functions may be browser event handlers. Rendering can neither
          // invoke them nor read them as values.
          env.set(node.id.name, null);
          break;
        case 'ExpressionStatement':
          ops.push(...output(node.expression, env, renderer));
          break;
        case 'IfStatement': {
          const value = reference(node.test, env);
          if (isConstant(value)) {
            ops.push(...branch(value.const ? node.consequent : node.alternate, env, renderer));
          } else {
            ops.push({ kind: 'if', value, body: branch(node.consequent, env, renderer), alt: branch(node.alternate, env, renderer) });
          }
          break;
        }
        case 'ForStatement':
          ops.push(each(node, env, renderer));
          break;
        default:
          reject(node, 'request markup supports Svelte conditionals and each blocks with JSON values');
      }
    }
    return ops;
  }

  function declaration(node, env) {
    const { id, init } = node;
    if (isSvelteCall(init, 'ensure_array_like', env) && init.arguments.length === 1 && id.type === 'Identifier') {
      env.set(id.name, { array: reference(init.arguments[0], env) });
      return;
    }
    if (id.type === 'Identifier' && (init?.type === 'ArrowFunctionExpression' || init?.type === 'FunctionExpression')) {
      env.set(id.name, null);
      return;
    }
    const value = init ? reference(init, env) : { const: null };
    if (id.type === 'Identifier') {
      env.set(id.name, value);
    } else if (id.type === 'ObjectPattern' && value.path) {
      for (const prop of id.properties) {
        if (prop.type !== 'Property' || prop.computed || prop.value.type !== 'Identifier') {
          reject(prop, 'props may only be destructured by name');
        }
        env.set(prop.value.name, { path: [...value.path, staticKey(prop.key)] });
      }
    } else {
      reject(node, 'use direct assignments from JSON props');
    }
  }

  function reference(node, env) {
    switch (node.type) {
      case 'Literal':
        if (!node.regex && !node.bigint && (typeof node.value !== 'number' || Number.isFinite(node.value))) return { const: node.value };
        break;
      case 'Identifier': {
        const value = env.get(node.name);
        if (value && !value.array) return value;
        if (node.name === 'undefined' && !env.has(node.name) && !imports.has(node.name)) return { const: null };
        break;
      }
      case 'MemberExpression': {
        const owner = reference(node.object, env);
        if (!node.optional && owner.path && (!node.computed || node.property.type === 'Literal')) {
          const key = staticKey(node.property);
          if (key !== '__proto__' && key !== 'constructor' && key !== 'prototype') return { path: [...owner.path, key] };
        }
        break;
      }
      case 'ConditionalExpression': {
        const test = reference(node.test, env);
        if (isConstant(test)) return reference(test.const ? node.consequent : node.alternate, env);
        break;
      }
      case 'UnaryExpression':
        if (node.operator === 'void' && node.argument.type === 'Literal') return { const: null };
        break;
    }
    reject(node, 'request markup needs a JSON path or a literal; compute display values in the data projection');
  }

  function branch(node, env, renderer) {
    if (!node) return [];
    return statements(node.type === 'BlockStatement' ? node.body : [node], new Map(env), renderer);
  }

  function each(node, env, renderer) {
    // In this Svelte version an each block iterates ensure_array_like(value)
    // with one index and one length. Read the element into the loop scope.
    const [index, length, extra] = node.init?.declarations ?? [];
    const array = length?.init?.object?.name;
    const [item, more] = node.body.body?.[0]?.declarations ?? [];
    const source = env.get(array)?.array;
    if (extra || node.init?.kind !== 'let' || index?.id.type !== 'Identifier' || index.init?.value !== 0 ||
        length?.id.type !== 'Identifier' || length.init?.type !== 'MemberExpression' || length.init.computed ||
        length.init.property.name !== 'length' || !source || node.test?.type !== 'BinaryExpression' ||
        node.test.operator !== '<' || node.test.left.name !== index.id.name || node.test.right.name !== length.id.name ||
        node.update?.type !== 'UpdateExpression' || node.update.operator !== '++' || node.update.argument.name !== index.id.name ||
        more || item?.id.type !== 'Identifier' ||
        item.init?.type !== 'MemberExpression' || !item.init.computed || item.init.object.name !== array ||
        item.init.property.name !== index.id.name) {
      reject(node, 'unsupported Svelte each loop');
    }
    const local = String(++loopID);
    const inner = new Map(env);
    inner.set(index.id.name, null);
    inner.set(length.id.name, null);
    inner.set(item.id.name, { path: [local] });
    return { kind: 'each', text: local, value: source, body: statements(node.body.body.slice(1), inner, renderer) };
  }

  function output(node, env, renderer) {
    if (node.type !== 'CallExpression' || node.optional) reject(node, 'only direct Svelte renderer calls are supported');
    const fn = node.callee;
    const args = node.arguments;
    if (fn.type === 'MemberExpression' && !fn.computed && fn.object.name === renderer) {
      if (fn.property.name === 'push' && args.length === 1) return push(args[0], env);
      if (fn.property.name === 'component' && args.length === 1) return callback(args[0], env);
      if (fn.property.name === 'option' && args.length === 2) return option(args, env);
      reject(node, `unsupported Svelte server renderer operation ${fn.property.name}`);
    }
    if (fn.type === 'Identifier' && !env.has(fn.name)) {
      const [from, imported] = imports.get(fn.name) ?? [];
      // Svelte's onMount is a no-op in SSR. Its callback body cannot run here.
      if (from === 'svelte' && imported === 'onMount' && args.length === 1 &&
          (args[0].type === 'ArrowFunctionExpression' || args[0].type === 'FunctionExpression')) return [];
      if (from && imported === 'ImportDefaultSpecifier' && /\.(svelte|shugo)$/.test(from) &&
          args.length === 2 && args[0].type === 'Identifier' && args[0].name === renderer) {
        return [{ kind: 'component', text: from, props: namedValues(args[1], env) }];
      }
    }
    reject(node, 'request rendering cannot run script statements or call JavaScript helpers');
  }

  function callback(node, env) {
    if (node.type !== 'ArrowFunctionExpression' || node.async || node.body.type !== 'BlockStatement' ||
        node.params.length !== 1 || node.params[0].type !== 'Identifier') reject(node, 'expected Svelte markup callback');
    return statements(node.body.body, new Map(env), node.params[0].name);
  }

  function option(args, env) {
    // The supported select has a form value and explicit selected flags in
    // its options. A bound <select> uses renderer.select, rejected above.
    const attrs = namedValues(args[0], env);
    if (!attrs.value || Object.keys(attrs).some((name) => name !== 'value' && name !== 'selected')) {
      reject(args[0], 'each option needs a string value and may set selected explicitly');
    }
    const body = callback(args[1], env);
    if (body.some((op) => op.kind !== 'text' && (op.kind !== 'literal' || op.text.includes('<!---->')))) {
      reject(args[1], 'option labels may contain only text and direct text expressions');
    }
    return [
      { kind: 'literal', text: '<option' },
      ...Object.entries(attrs).map(([name, value]) => ({ kind: 'attr', text: name, value, boolean: name === 'selected' })),
      { kind: 'literal', text: '>' },
      ...body,
      { kind: 'literal', text: '</option>' },
    ];
  }

  function push(node, env) {
    if (node.type === 'Literal' && typeof node.value === 'string') return [{ kind: 'literal', text: node.value }];
    if (node.type !== 'TemplateLiteral') reject(node, 'expected Svelte markup template');
    const ops = [];
    for (let index = 0; index < node.quasis.length; index++) {
      const text = node.quasis[index].value.cooked;
      if (text) ops.push({ kind: 'literal', text });
      if (index < node.expressions.length) ops.push(interpolation(node.expressions[index], env));
    }
    return ops;
  }

  function interpolation(node, env) {
    if (isSvelteCall(node, 'escape', env) && node.arguments.length === 1) {
      return { kind: 'text', value: reference(node.arguments[0], env) };
    }
    if (isSvelteCall(node, 'attr', env)) {
      const [name, value, boolean, extra] = node.arguments;
      if (!extra && name?.type === 'Literal' && typeof name.value === 'string' && value &&
          name.value !== 'hidden' && name.value !== 'translate' && (!boolean || boolean.type === 'Literal' && typeof boolean.value === 'boolean')) {
        return { kind: 'attr', text: name.value, value: reference(value, env), boolean: boolean?.value === true };
      }
    }
    if (isSvelteCall(node, 'attr_class', env) && node.arguments.length === 1 && isSvelteCall(node.arguments[0], 'clsx', env) &&
        node.arguments[0].arguments.length === 1) {
      return { kind: 'class', value: reference(node.arguments[0].arguments[0], env) };
    }
    reject(node, 'request markup only supports plain escaped text, simple attributes and string classes');
  }

  function namedValues(node, env) {
    if (node.type !== 'ObjectExpression') reject(node, 'pass plain JSON props to a child component');
    return Object.fromEntries(node.properties.map((prop) => {
      if (prop.type !== 'Property' || prop.computed || prop.kind !== 'init' || prop.method) reject(prop, 'pass object properties by name');
      return [staticKey(prop.key), reference(prop.value, env)];
    }));
  }

  function isSvelteCall(node, name, env) {
    const fn = node?.callee;
    if (node?.type !== 'CallExpression' || node.optional || fn.type !== 'MemberExpression' || fn.computed ||
        fn.object.type !== 'Identifier' || env.has(fn.object.name) || fn.property.name !== name) return false;
    const [from, specifier] = imports.get(fn.object.name) ?? [];
    return from === 'svelte/internal/server' && specifier === 'ImportNamespaceSpecifier';
  }
}

function staticKey(node) {
  if (node.type === 'Identifier') return node.name;
  if (node.type === 'Literal' && typeof node.value === 'string') return node.value;
  reject(node, 'expected a named field');
}

function isFlag(node) {
  if (node?.type !== 'VariableDeclaration' || node.kind !== 'const' || node.declarations.length !== 1) return false;
  const [flag] = node.declarations;
  return flag.id.type === 'Identifier' && flag.id.name === 'shugoSSR' && flag.init?.type === 'Literal' && flag.init.value === true;
}

function isConstant(value) {
  return Object.hasOwn(value, 'const');
}

function reject(node, why) {
  const error = new Error(why);
  error.start = node?.loc?.start;
  throw error;
}
