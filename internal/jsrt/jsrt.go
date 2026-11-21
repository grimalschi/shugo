// Package jsrt holds the JavaScript side of the framework: the small library
// that the code generated from templates calls while it computes the values
// of pages, and the queries to the data database of the project. It is
// bundled into every project and runs on Bun or on Node.
package jsrt

import _ "embed"

// Source is the library as an ECMAScript module, known to the generated
// code as "shugo:runtime".
//
//go:embed runtime.js
var Source string

// Query is the module the library reads the data database with:
// "shugo:query".
//
//go:embed query.js
var Query string

//go:embed catalog.js
var Catalog string
