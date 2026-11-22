// Package compile turns the sources of a project into what the next stages
// run: one JavaScript bundle in which the code Svelte compiled from the
// templates renders the pages, reading the data database of the project
// through query from the module shugo, and the skeleton of the site.
package compile

import (
	"github.com/grimalschi/shugo/internal/svelte"
)

// role is what a template is to the site.
type role int

const (
	component role = iota
	page
	layout
)

// unit is one regular .svelte file, compiled by Svelte for a server.
type unit struct {
	path, source string
	role         role
	*svelte.Template
}

// warnings are what Svelte warns about in the file, each with its place.
func (u *unit) warnings() []string {
	out := make([]string, len(u.Warnings))
	for i, w := range u.Warnings {
		out[i] = (&Error{Path: u.path, Line: w.Line, Column: w.Column, Message: w.Text}).Error()
	}
	return out
}

// own tells that the file itself cannot be computed once for all pages: it
// shows what differs from page to page.
func (u *unit) own() bool {
	return u.Facts.Page
}

// placed tells that what the file shows depends on the place it stands at:
// it uses the context of Svelte, which the components around it give; calls
// $props.id(), whose ids count the components rendered before it; or shows
// a snippet inside a <select>, whose value marks the <option> the snippet
// shows. Such a component is rendered where it stands, and so is any that
// shows it.
func (u *unit) placed() bool {
	return u.Facts.Context || u.Facts.ID || u.Facts.Select
}
