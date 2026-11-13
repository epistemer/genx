package main

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// block is a code block in a guide that shows a file of this example:
// <!-- file: path --> followed by a fenced block.
var block = regexp.MustCompile("(?s)<!-- file: (\\S+) -->\\n```[a-z]*\\n(.*?)```\\n")

// TestGuides keeps the guides' code blocks the same as the files they show,
// so the guides' examples are the ones CI runs.
func TestGuides(t *testing.T) {
	guides, err := filepath.Glob("../../docs/guides/*.md")
	if err != nil || len(guides) == 0 {
		t.Fatalf("no guides found: %v", err)
	}
	n := 0
	for _, g := range guides {
		b, err := os.ReadFile(g)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range block.FindAllSubmatch(b, -1) {
			n++
			want, err := os.ReadFile(string(m[1]))
			if err != nil {
				t.Errorf("%s: %v", g, err)
				continue
			}
			if string(m[2]) != string(want) {
				t.Errorf("%s: the block for %s differs from the file", g, m[1])
			}
		}
	}
	if n < 6 {
		t.Errorf("found %d file blocks in the guides, want at least 6", n)
	}
}
