// Package jsrt holds the JavaScript side of the framework: the small library
// that the code generated from templates calls while it computes the values
// of pages. It is bundled into every project and runs on Bun or on Node.
package jsrt

import _ "embed"

// Source is the library as an ECMAScript module.
//
//go:embed runtime.js
var Source string
