package jsrt

import (
	"encoding/binary"
	"math/bits"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// murmur3 is MurmurHash3 (x86, 32 bits) as its author describes it, for
// input whose length is a multiple of four.
func murmur3(data []byte, seed uint32) uint32 {
	h := seed
	for i := 0; i+4 <= len(data); i += 4 {
		k := binary.LittleEndian.Uint32(data[i:])
		k *= 0xcc9e2d51
		k = bits.RotateLeft32(k, 15)
		k *= 0x1b873593
		h ^= k
		h = bits.RotateLeft32(h, 13)
		h = h*5 + 0xe6546b64
	}
	h ^= uint32(len(data))
	h ^= h >> 16
	h *= 0x85ebca6b
	h ^= h >> 13
	h *= 0xc2b2ae35
	return h ^ h>>16
}

// The values that everyone checks an implementation of MurmurHash3 against.
func TestMurmurHash3(t *testing.T) {
	for _, c := range []struct {
		data string
		seed uint32
		want uint32
	}{
		{"", 0, 0},
		{"", 1, 0x514e28b7},
		{"", 0xffffffff, 0x81f16f39},
		{"\xff\xff\xff\xff", 0, 0x76293b50},
		{"\x21\x43\x65\x87", 0, 0xf55b516b},
		{"\x21\x43\x65\x87", 0x5082edee, 0x2362f9de},
		{"\x00\x00\x00\x00", 0, 0x2362f9de},
		{"aaaa", 0x9747b28c, 0x5a97808a},
		{"test", 0, 0xba6bd213},
	} {
		if got := murmur3([]byte(c.data), c.seed); got != c.want {
			t.Errorf("murmur3(%q, %#x) = %#x, want %#x", c.data, c.seed, got, c.want)
		}
	}
}

// The library of the load side is tested where it runs: on Bun and on Node.
// The script prints what is wrong, one line for each thing.
func TestLoadSide(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "load.js"), []byte(Load), 0o644); err != nil {
		t.Fatal(err)
	}
	script, err := os.ReadFile("load_test.mjs")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "test.mjs"), script, 0o644); err != nil {
		t.Fatal(err)
	}
	ran := false
	for _, runtime := range []string{"bun", "node"} {
		if _, err := exec.LookPath(runtime); err != nil {
			continue
		}
		ran = true
		out, err := exec.Command(runtime, filepath.Join(dir, "test.mjs")).CombinedOutput()
		if err != nil || strings.TrimSpace(string(out)) != "ok" {
			t.Errorf("%s: %v\n%s", runtime, err, out)
		}
	}
	if !ran {
		t.Skip("neither bun nor node is installed")
	}
}
