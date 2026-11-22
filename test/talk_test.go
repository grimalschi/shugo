package test

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// talk runs the JavaScript of the site through a wrapper that keeps a copy
// of what it says to shugo, and returns the letters of those messages.
func (s *site) talk(run func()) string {
	s.t.Helper()
	runtime := s.runtime
	if runtime == "" {
		runtime = "bun"
		if _, err := exec.LookPath("bun"); err != nil {
			runtime = "node"
		}
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		s.t.Skip("the wrapper that keeps the conversation needs bash")
	}
	copy := filepath.Join(s.t.TempDir(), "said.bin")
	wrapper := filepath.Join(s.t.TempDir(), "tap.sh")
	script := fmt.Sprintf("#!%s\nexec %s \"$@\" 3> >(tee -a %q >&3)\n", bash, runtime, copy)
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		s.t.Fatal(err)
	}
	was := s.runtime
	s.runtime = wrapper
	defer func() { s.runtime = was }()
	run()
	data, err := os.ReadFile(copy)
	if os.IsNotExist(err) {
		return ""
	} else if err != nil {
		s.t.Fatal(err)
	}
	var letters strings.Builder
	for len(data) >= 5 {
		size := int(binary.LittleEndian.Uint32(data))
		if size < 1 || 4+size > len(data) {
			s.t.Fatalf("a damaged copy of the conversation")
		}
		letters.WriteByte(data[4])
		data = data[4+size:]
	}
	return letters.String()
}

// After the first build an update asks nothing it was asked before: shugo
// tells at the start which questions have their indexes and which shapes of
// readings it knows, so a change that reaches one page is one request and
// one answer.
func TestUpdateAsksNothingKnown(t *testing.T) {
	s := newSite(t, indexSite)
	s.sql(readRows)
	built := s.talk(func() { s.build() })
	if !strings.Contains(built, "I") || !strings.Contains(built, "S") {
		t.Fatalf("the first build asked for no index or told no shape: %q", built)
	}

	s.sql(`UPDATE products SET title = 'Big jug' WHERE id = 'p1'`)
	var computed int
	said := s.talk(func() { computed = s.update().Computed.Pages })
	if computed == 0 {
		t.Fatal("the update computed nothing")
	}
	// Records, then done: no question about an index, no shape told again.
	if strings.Trim(said, "R") != "D" || !strings.HasPrefix(said, "R") {
		t.Errorf("the project's code said %q in the update, want records and done", said)
	}

	// A question asked for the first time is still asked, and answered.
	s.write(`
-- src/pages/new.shugo --
<script>
import { query } from "shugo"
</script>
<p>{query("products").where({ title: "Cup" }).order("price").all().length}</p>
`)
	s.build()
	if got := s.indexes(); !contains(got, "_shugo_products_title_price_id") {
		t.Errorf("no index for the new question: %q", got)
	}
	s.sameAsBuilt()
}

func contains(list []string, item string) bool {
	for _, one := range list {
		if one == item {
			return true
		}
	}
	return false
}
