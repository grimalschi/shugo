package render

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/grimalschi/shugo/internal/program"
)

// IslandBody walks a precompiled markup program with the same JSON props the
// browser will hydrate with. Only the root of a Svelte SSR render has the
// outer markers; the emitter has already supplied those inside the template.
func IslandBody(site *program.Site, name string, props json.RawMessage) (string, error) {
	var root map[string]any
	decoder := json.NewDecoder(bytes.NewReader(props))
	decoder.UseNumber()
	if err := decoder.Decode(&root); err != nil {
		return "", fmt.Errorf("island %s props: %w", name, err)
	}
	r := renderer{site: site, out: []byte("<!--[-->")}
	if err := r.islandComponent(name, root, 0); err != nil {
		return "", err
	}
	return string(append(r.out, "<!--]-->"...)), nil
}

func (r *renderer) islandComponent(name string, props map[string]any, depth int) error {
	ops, ok := r.site.IslandPrograms[name]
	if !ok {
		return fmt.Errorf("no Go markup program for Svelte island %s", name)
	}
	if err := r.islandOps(ops, map[string]any{"$": props}, depth); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func (r *renderer) islandOps(ops []program.IslandOp, scope map[string]any, depth int) error {
	if depth >= maxDepth {
		return errors.New("island template is nested too deep")
	}
	for _, op := range ops {
		if err := r.islandOp(op, scope, depth); err != nil {
			return err
		}
		if len(r.out) > maxOutput {
			return errors.New("the island is too large")
		}
	}
	return nil
}

func (r *renderer) islandOp(op program.IslandOp, scope map[string]any, depth int) error {
	switch op.Kind {
	case "literal":
		r.out = append(r.out, op.Text...)
		return nil
	case "component":
		props := make(map[string]any, len(op.Props))
		for name, binding := range op.Props {
			value, err := islandValue(&binding, scope)
			if err != nil {
				return err
			}
			props[name] = value
		}
		return r.islandComponent(op.Text, props, depth+1)
	}
	value, err := islandValue(op.Value, scope)
	if err != nil {
		return err
	}
	switch op.Kind {
	case "text":
		text, err := islandText(value)
		if err != nil {
			return err
		}
		r.out = appendEscaped(r.out, []byte(text), false)
	case "attr":
		return r.islandAttribute(op.Text, value, op.Boolean)
	case "class":
		if value == nil || value == "" {
			return nil
		}
		class, ok := value.(string)
		if !ok {
			return errors.New("an island class needs a JSON string")
		}
		return r.islandAttribute("class", class, false)
	case "if":
		condition, ok := value.(bool)
		if !ok {
			return errors.New("an island condition needs a JSON boolean")
		}
		if condition {
			return r.islandOps(op.Body, scope, depth+1)
		}
		return r.islandOps(op.Alt, scope, depth+1)
	case "each":
		items, ok := value.([]any)
		if !ok {
			return errors.New("an island repetition needs a JSON array")
		}
		if _, exists := scope[op.Text]; exists || op.Text == "" {
			return fmt.Errorf("invalid island loop binding %q", op.Text)
		}
		defer delete(scope, op.Text)
		for _, item := range items {
			scope[op.Text] = item
			if err := r.islandOps(op.Body, scope, depth+1); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unknown island operation %q", op.Kind)
	}
	return nil
}

func islandValue(binding *program.IslandValue, scope map[string]any) (any, error) {
	if binding == nil {
		return nil, errors.New("island operation is missing its input")
	}
	if len(binding.Const) != 0 {
		if len(binding.Path) != 0 {
			return nil, errors.New("island input has both a path and a constant")
		}
		decoder := json.NewDecoder(bytes.NewReader(binding.Const))
		decoder.UseNumber()
		var value any
		err := decoder.Decode(&value)
		return value, err
	}
	if len(binding.Path) == 0 {
		return nil, errors.New("island input is missing its path")
	}
	var value any = scope
	for _, key := range binding.Path {
		if fields, ok := value.(map[string]any); ok {
			if next, present := fields[key]; present {
				value = next
				continue
			}
		}
		return nil, fmt.Errorf("missing island input %v", binding.Path)
	}
	return value, nil
}

func islandText(value any) (string, error) {
	switch value := value.(type) {
	case nil:
		return "", nil
	case string:
		return value, nil
	case bool:
		if value {
			return "true", nil
		}
		return "false", nil
	case json.Number:
		return string(value), nil
	default:
		return "", errors.New("an island text or attribute needs a JSON scalar")
	}
}

func (r *renderer) islandAttribute(name string, value any, boolean bool) error {
	if !attributeName([]byte(name)) {
		return fmt.Errorf("invalid island attribute %q", name)
	}
	if value == nil {
		return nil
	}
	if boolean {
		switch value {
		case false:
			return nil
		case true, "":
			return r.attribute(name, []byte("+"), false)
		default:
			return fmt.Errorf("island attribute %q needs a boolean or an empty string", name)
		}
	}
	text, err := islandText(value)
	if err != nil {
		return err
	}
	return r.attribute(name, append([]byte{'='}, text...), program.IsAddress(name))
}
