package skeleton

import "encoding/json"

// IslandValue binds one insertion, branch or repetition to an input value.
// A path starts at "$" (the props) or at a row in an enclosing repetition.
// A constant is already a JSON value, including null.
type IslandValue struct {
	Path  []string        `json:"path,omitempty"`
	Const json.RawMessage `json:"const,omitempty"`
}

// IslandOp is static markup or an instruction to place JSON data in it.
// Kind determines whether Text is markup, an attribute name, a loop's
// binding name, or the source path of a component. Body and Alt are
// subordinate programs. No operation evaluates an expression.
type IslandOp struct {
	Kind    string                 `json:"kind"`
	Text    string                 `json:"text,omitempty"`
	Value   *IslandValue           `json:"value,omitempty"`
	Boolean bool                   `json:"boolean,omitempty"`
	Body    []IslandOp             `json:"body,omitempty"`
	Alt     []IslandOp             `json:"alt,omitempty"`
	Props   map[string]IslandValue `json:"props,omitempty"`
}
