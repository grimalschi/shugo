// Package jsrt holds the JavaScript side of the framework: small libraries
// that are bundled into every project and run on Bun or on Node. One runs
// the scripts of pages and components and tells what their templates read;
// with it goes the one that reads the data database of the project. The
// third computes the values of templates.
package jsrt

import _ "embed"

// Load is the library of the load side, as an ECMAScript module: it runs
// the scripts with the data of the project and leads the computation.
//
//go:embed load.js
var Load string

// Render is the library of the render side, as an ECMAScript module: the
// code generated from templates calls it while it computes their values.
//
//go:embed render.js
var Render string

// Query is the module the load side reads the data database with:
// "shugo:query".
//
//go:embed query.js
var Query string
