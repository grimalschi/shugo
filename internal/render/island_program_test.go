package render

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/grimalschi/shugo/internal/program"
)

func TestIslandProgram(t *testing.T) {
	path := func(fields ...string) *program.IslandValue { return &program.IslandValue{Path: fields} }
	site := &program.Site{IslandPrograms: map[string][]program.IslandOp{
		"catalog": {
			{Kind: "literal", Text: "<ul>"},
			{Kind: "each", Text: "row", Value: path("$", "items"), Body: []program.IslandOp{
				{Kind: "literal", Text: "<li>"},
				{Kind: "component", Text: "card", Props: map[string]program.IslandValue{"item": *path("row")}},
				{Kind: "literal", Text: "</li>"},
			}},
			{Kind: "literal", Text: "</ul>"},
		},
		"card": {
			{Kind: "literal", Text: "<a"},
			{Kind: "attr", Text: "href", Value: path("$", "item", "url")},
			{Kind: "attr", Text: "aria-current", Value: path("$", "item", "selected")},
			{Kind: "class", Value: path("$", "item", "class")},
			{Kind: "literal", Text: ">"},
			{Kind: "if", Value: path("$", "item", "selected"), Body: []program.IslandOp{
				{Kind: "literal", Text: "<!--[0-->"},
				{Kind: "text", Value: path("$", "item", "title")},
			}, Alt: []program.IslandOp{{Kind: "literal", Text: "<!--[-1-->Not selected"}}},
			{Kind: "literal", Text: "<!--]--></a>"},
		},
	}}
	props := json.RawMessage(`{"items":[{"url":"/?a=1&b=2","selected":true,"title":"a < b","class":"card"},{"url":"javascript:bad()","selected":false,"class":null}]}`)
	body, err := IslandBody(site, "catalog", props)
	want := `<!--[--><ul><li><a href="/?a=1&amp;b=2" aria-current="true" class="card"><!--[0-->a &lt; b<!--]--></a></li><li><a href="about:invalid#blocked" aria-current="false"><!--[-1-->Not selected<!--]--></a></li></ul><!--]-->`
	if err != nil || body != want {
		t.Fatalf("got %q, %v\nwant %q", body, err, want)
	}
	if _, err := IslandBody(site, "catalog", json.RawMessage(`{"items":[{"url":"/","class":""}]}`)); err == nil || !strings.Contains(err.Error(), "missing island input") {
		t.Fatalf("the program must report a missing input: %v", err)
	}
}
