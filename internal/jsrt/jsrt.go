// Package jsrt holds the JavaScript side of the framework: the library that
// shows the components Svelte compiled from the templates, cuts the pages
// they render into blocks and hands them to the Go side, and the queries to
// the data database of the project. It is bundled into every project and
// runs on Bun or on Node.
package jsrt

import _ "embed"

// Source is the library as an ECMAScript module, known to the bundle as
// "shugo:runtime".
//
//go:embed runtime.js
var Source string

// Query is the module the library reads the data database with:
// "shugo:query".
//
//go:embed query.js
var Query string

// Search is the module of searches of tables: "shugo:search", which the
// module shugo gives search from.
//
//go:embed search.js
var Search string

//go:embed wire.js
var Wire string
