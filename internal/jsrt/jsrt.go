// Package jsrt holds the JavaScript side of the framework: two small
// libraries that are bundled into every project and run on Bun or on Node.
// One runs the scripts of pages and components and tells what has changed
// in what their templates read; the other computes the values of templates.
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
