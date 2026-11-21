package compile

import (
	"fmt"
	"slices"
	"strings"
)

// Some elements cannot stand inside others: a browser that reads such HTML
// repairs it, by moving, removing or adding elements, and the page shows
// otherwise than written. Svelte refuses such a template, or warns where a
// block stands between the two; so does Shugo. The rules are those of
// Svelte 5.59 (html-tree-validation.js, MIT license).

// placement is what an element does not allow inside itself.
type placement struct {
	direct     []string // not right inside it
	descendant []string // not anywhere inside it
	resetBy    []string // unless one of these stands between
	only       []string // nothing but these right inside it
}

var autoclosingChildren = map[string]placement{
	"li":       {direct: []string{"li"}},
	"dt":       {descendant: []string{"dt", "dd"}, resetBy: []string{"dl"}},
	"dd":       {descendant: []string{"dt", "dd"}, resetBy: []string{"dl"}},
	"p":        {descendant: []string{"address", "article", "aside", "blockquote", "div", "dl", "fieldset", "footer", "form", "h1", "h2", "h3", "h4", "h5", "h6", "header", "hgroup", "hr", "main", "menu", "nav", "ol", "p", "pre", "section", "table", "ul"}},
	"rt":       {descendant: []string{"rt", "rp"}},
	"rp":       {descendant: []string{"rt", "rp"}},
	"optgroup": {descendant: []string{"optgroup"}},
	"option":   {descendant: []string{"option", "optgroup"}},
	"thead":    {direct: []string{"tbody", "tfoot"}},
	"tbody":    {direct: []string{"tbody", "tfoot"}},
	"tfoot":    {direct: []string{"tbody"}},
	"tr":       {direct: []string{"tr", "tbody"}},
	"td":       {direct: []string{"td", "th", "tr"}},
	"th":       {direct: []string{"td", "th", "tr"}},
}

var headings = []string{"h1", "h2", "h3", "h4", "h5", "h6"}

var disallowedChildren = func() map[string]placement {
	out := map[string]placement{}
	for name, rule := range autoclosingChildren {
		out[name] = rule
	}
	out["form"] = placement{descendant: []string{"form"}}
	out["a"] = placement{descendant: []string{"a"}}
	out["button"] = placement{descendant: []string{"button"}}
	for _, h := range headings {
		out[h] = placement{descendant: headings}
	}
	out["tr"] = placement{only: []string{"th", "td", "style", "script", "template"}}
	for _, name := range []string{"tbody", "thead", "tfoot"} {
		out[name] = placement{only: []string{"tr", "style", "script", "template"}}
	}
	out["colgroup"] = placement{only: []string{"col", "template"}}
	out["table"] = placement{only: []string{"caption", "colgroup", "tbody", "thead", "tfoot", "style", "script", "template"}}
	out["head"] = placement{only: []string{"base", "basefont", "bgsound", "link", "meta", "title", "noscript", "noframes", "style", "script", "template"}}
	out["html"] = placement{only: []string{"head", "body", "frameset"}}
	out["frameset"] = placement{only: []string{"frame"}}
	return out
}()

// validWithParent tells why child cannot stand right inside parent; "" if it can.
func validWithParent(child, parent string) string {
	if strings.Contains(child, "-") || strings.Contains(parent, "-") || parent == "template" {
		return ""
	}
	if rule, ok := disallowedChildren[parent]; ok {
		switch {
		case slices.Contains(rule.direct, child):
			return fmt.Sprintf("<%s> cannot be a direct child of <%s>", child, parent)
		case slices.Contains(rule.descendant, child):
			return fmt.Sprintf("<%s> cannot be a child of <%s>", child, parent)
		case rule.only != nil:
			if slices.Contains(rule.only, child) {
				return ""
			}
			return fmt.Sprintf("<%s> cannot be a child of <%s>: <%s> only allows <%s>", child, parent, parent, strings.Join(rule.only, ">, <"))
		}
	}
	switch child {
	case "body", "caption", "col", "colgroup", "frameset", "frame", "head", "html":
		return fmt.Sprintf("<%s> cannot be a child of <%s>", child, parent)
	case "thead", "tbody", "tfoot":
		return fmt.Sprintf("<%s> must be the child of a <table>, not of a <%s>", child, parent)
	case "td", "th":
		return fmt.Sprintf("<%s> must be the child of a <tr>, not of a <%s>", child, parent)
	case "tr":
		return fmt.Sprintf("<tr> must be the child of a <thead>, <tbody> or <tfoot>, not of a <%s>", parent)
	}
	return ""
}

// validWithAncestor tells why child cannot stand anywhere inside the last of
// ancestors, which lists the elements from the parent outwards; "" if it can.
func validWithAncestor(child string, ancestors []string) string {
	if strings.Contains(child, "-") {
		return ""
	}
	ancestor := ancestors[len(ancestors)-1]
	rule, ok := disallowedChildren[ancestor]
	if !ok {
		return ""
	}
	for i := len(ancestors) - 2; i >= 0; i-- {
		if strings.Contains(ancestors[i], "-") || slices.Contains(rule.resetBy, ancestors[i]) {
			return ""
		}
	}
	if slices.Contains(rule.descendant, child) {
		return fmt.Sprintf("<%s> cannot be a descendant of <%s>", child, ancestor)
	}
	return ""
}

// ancestor is an element around a place in a template, as far as the
// component or the snippet it is written in. blocked tells that a block
// stands between: then the browser does not repair the page, and Svelte
// only warns.
type ancestor struct {
	name    string
	blocked bool
}

// placed checks where an element of HTML stands: what the browser would
// repair is an error, or a warning across a block.
func (l *lowerer) placed(name string, at int, ancestors []ancestor) {
	if len(ancestors) == 0 {
		return
	}
	report := func(blocked bool, message string) {
		message += ": the browser would repair the HTML, and the page would show otherwise than written"
		if blocked {
			l.warnf(at, "%s", message)
		} else {
			l.errorf(at, "%s", message)
		}
	}
	if message := validWithParent(name, ancestors[0].name); message != "" {
		report(ancestors[0].blocked, message)
		return
	}
	names := []string{ancestors[0].name}
	for _, a := range ancestors[1:] {
		names = append(names, a.name)
		if message := validWithAncestor(name, names); message != "" {
			report(a.blocked, message)
			return
		}
	}
}
