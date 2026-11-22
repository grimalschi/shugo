// Adapt the public Svelte AST to the fields the Shugo lowerer consumes.
// Source slices keep identifiers, expressions, and markup in the form the
// author wrote. JavaScript positions are UTF-16, while Go indexes UTF-8.
function parseTemplate(compiler, filename, input) {
  const bom = input.charCodeAt(0) === 0xfeff ? 1 : 0;
  const source = input.slice(bom);
  const offsets = new Uint32Array(source.length + 1);
  let bytes = bom * 3;
  for (let i = 0; i < source.length; i++) {
    offsets[i] = bytes;
    const point = source.codePointAt(i);
    if (point > 0xffff) {
      offsets[++i] = bytes;
      bytes += 4;
    } else if (point > 0x7ff) {
      bytes += 3;
    } else {
      bytes += point > 0x7f ? 2 : 1;
    }
  }
  offsets[source.length] = bytes;

  function problem(message, start) {
    throw Object.assign(new Error(message), { start: start + bom });
  }

  const start = source.trimStart();
  if (start.startsWith('---')) {
    problem('the code of a component goes into <script>…</script>, as in Svelte, not between "---" lines', source.length - start.length);
  }
  let ast;
  try {
    ast = compiler.parse(source, { modern: true, filename });
  } catch (error) {
    problem(error.message.split('\n')[0], error.position?.[0] ?? 0);
  }

  const at = (node) => offsets[node.start];
  const raw = (node) => source.slice(node.start, node.end);
  const code = (node) => node && { Text: raw(node), Start: at(node) };
  const nodes = (fragment) => fragment?.nodes.map(node) ?? null;
  const part = (item) => item.type === 'Text'
    ? { Start: at(item), Text: item.raw }
    : { Start: at(item), Expr: code(item.expression) };

  function attrValue(value) {
    if (value === true || !value) return null;
    return (Array.isArray(value) ? value : [value]).map(part);
  }

  function quoted(value) {
    return Array.isArray(value) && ['"', "'"].includes(source[value[0]?.start - 1]);
  }

  function attribute(attr) {
    const base = { Start: at(attr), Raw: raw(attr) };
    switch (attr.type) {
      case 'Attribute': {
        if (source[attr.start] === '{') {
          return { ...base, Kind: 1, Name: attr.name, Value: attrValue(attr.value) };
        }
        const { name, value } = attr;
        if (name.startsWith('request:')) {
          const [Name, ...Modifiers] = name.slice('request:'.length).split('|');
          if (!Name) problem('request: needs a prop name after the colon', attr.start);
          return { ...base, Kind: 3, Name, Directive: 'request', Modifiers, Value: attrValue(value), Quoted: quoted(value) };
        }
        return { ...base, Kind: 0, Name: name, Value: attrValue(value), Quoted: quoted(value) };
      }
      case 'SpreadAttribute':
        return { ...base, Kind: 2, Value: [{ Start: at(attr.expression), Expr: code(attr.expression) }] };
      case 'AttachTag':
        return { ...base, Kind: 4 };
      case 'StyleDirective':
        return { ...base, Kind: 3, Name: attr.name, Directive: 'style', Modifiers: attr.modifiers, Value: attrValue(attr.value), Quoted: quoted(attr.value) };
      default: {
        if (!attr.type.endsWith('Directive')) {
          problem(`unsupported Svelte attribute ${attr.type}`, attr.start);
        }
        const [directive] = raw(attr).split(':', 1);
        const expression = attr.expression;
        const explicit = raw(attr).includes('=');
        return {
          ...base, Kind: 3, Name: attr.name, Directive: directive, Modifiers: attr.modifiers,
          Value: explicit && expression ? [{ Start: at(expression), Expr: code(expression) }] : null,
        };
      }
    }
  }

  function section(item) {
    if (!item) return null;
    return {
      Start: at(item), Code: { Start: at(item.content), End: offsets[item.content.end] },
      Attrs: item.attributes.map(attribute),
    };
  }

  function element(item) {
    const attributes = item.attributes.map(attribute);
    const target = item.type === 'SvelteElement' ? item.tag : item.type === 'SvelteComponent' ? item.expression : null;
    if (target) {
      // Svelte moves "this" out of the attributes array. Static target text
      // appears as a synthetic string literal without quotes in the source.
      let expression = code(target);
      if (target.type === 'Literal' && typeof target.value === 'string' && expression.Text !== target.raw) {
        expression = { ...expression, Text: JSON.stringify(target.value) };
      }
      attributes.push({ Start: at(target), Kind: 0, Name: 'this', Value: [{ Start: at(target), Expr: expression }] });
    }
    const children = nodes(item.fragment);
    return {
      Type: 'Element', Start: at(item), Name: item.name, Attrs: attributes, Children: children,
      SelfClosing: children.length === 0 && source.slice(item.end - 2, item.end) === '/>',
    };
  }

  function ifBranches(item) {
    const branches = [];
    while (item) {
      branches.push({ Start: at(item), Test: code(item.test), Body: nodes(item.consequent) });
      if (!item.alternate) break;
      const first = item.alternate.nodes[0];
      if (first?.type !== 'IfBlock' || !first.elseif) {
        branches.push({ Start: first ? at(first) : offsets[item.end], Body: nodes(item.alternate) });
        break;
      }
      item = first;
    }
    return branches;
  }

  function awaitBranch(fragment, argument, item) {
    if (!fragment) return null;
    const first = fragment.nodes[0] ?? item;
    return { Start: at(first), Test: code(argument), Body: nodes(fragment) };
  }

  function renderTag(item) {
    const optional = item.expression.type === 'ChainExpression';
    const call = optional ? item.expression.expression : item.expression;
    const args = call.arguments.map((arg) => {
      if (arg.type === 'SpreadElement') {
        problem('spread arguments are not allowed in {@render}', arg.start);
      }
      return code(arg);
    });
    return {
      Type: 'Tag', Start: at(item), Kind: 3, Code: code(item.expression),
      Call: { Callee: code(call.callee), Arguments: args, Optional: optional },
    };
  }

  function node(item) {
    const base = { Type: item.type, Start: at(item) };
    switch (item.type) {
      case 'Text':
        return { ...base, Raw: item.raw };
      case 'Comment':
        return { ...base, Type: 'HTMLComment', Data: item.data };
      case 'ExpressionTag':
        return { ...base, Type: 'Tag', Kind: 0, Code: code(item.expression) };
      case 'HtmlTag':
        return { ...base, Type: 'Tag', Kind: 1, Code: code(item.expression) };
      case 'ConstTag':
        return { ...base, Type: 'Tag', Kind: 2, Code: code(item.declaration.declarations[0]) };
      case 'DeclarationTag':
        problem('use {@const …} for template declarations in Shugo', item.start);
      case 'RenderTag':
        return renderTag(item);
      case 'DebugTag': {
        const first = item.identifiers[0];
        const last = item.identifiers.at(-1);
        return { ...base, Type: 'Tag', Kind: 4, Code: first ? code({ start: first.start, end: last.end }) : { Start: at(item), Text: '' } };
      }
      case 'IfBlock':
        return { ...base, Branches: ifBranches(item) };
      case 'EachBlock':
        return {
          ...base, List: code(item.expression), Item: code(item.context), Index: item.index, Key: code(item.key),
          Body: nodes(item.body), Else: nodes(item.fallback),
        };
      case 'AwaitBlock':
        return {
          ...base, Value: code(item.expression), Pending: nodes(item.pending), ThenFirst: !item.pending,
          Then: awaitBranch(item.then, item.value, item), Catch: awaitBranch(item.catch, item.error, item),
        };
      case 'KeyBlock':
        return { ...base, Value: code(item.expression), Body: nodes(item.fragment) };
      case 'SnippetBlock': {
        const first = item.parameters[0];
        const last = item.parameters.at(-1);
        const params = first ? code({ start: first.start, end: last.end }) : { Start: offsets[item.expression.end], Text: '' };
        return { ...base, Name: item.expression.name, Params: params, Body: nodes(item.body) };
      }
      default:
        if (item.name?.toLowerCase() === '!doctype') {
          return { ...base, Type: 'Doctype', Raw: raw(item) };
        }
        if (item.name && item.attributes && item.fragment) return element(item);
        problem(`unsupported Svelte syntax ${item.type}`, item.start);
    }
  }

  if (ast.options) {
    problem('<svelte:options> is not supported in Shugo templates', ast.options.start);
  }
  return {
    Module: section(ast.module), Script: section(ast.instance), Style: section(ast.css),
    Body: nodes(ast.fragment),
  };
}
