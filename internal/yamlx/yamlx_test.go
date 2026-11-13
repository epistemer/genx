package yamlx

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr string
	}{
		{
			name:    "empty string",
			data:    "",
			wantErr: "genx.yaml:1: empty document",
		},
		{
			name:    "comment only",
			data:    "# just a comment\n",
			wantErr: "genx.yaml:1: empty document",
		},
		{
			name:    "null document",
			data:    "---\n",
			wantErr: "genx.yaml:1: empty document",
		},
		{
			name:    "second document",
			data:    "a: 1\n---\nb: 2\n",
			wantErr: "genx.yaml:2: unexpected second document",
		},
		{
			name:    "duplicate mapping keys",
			data:    "foo: 1\nfoo: 2\n",
			wantErr: `genx.yaml:2: foo: duplicate key "foo"`,
		},
		{
			name:    "nested duplicate mapping keys",
			data:    "parent:\n  child: 1\n  child: 2\n",
			wantErr: `genx.yaml:3: parent.child: duplicate key "child"`,
		},
		{
			name:    "sequence duplicate mapping keys",
			data:    "items:\n  - a: 1\n    a: 2\n",
			wantErr: `genx.yaml:3: items[0].a: duplicate key "a"`,
		},
		{
			name:    "non-scalar mapping key",
			data:    "[1, 2]: val\n",
			wantErr: "genx.yaml:1: mapping key must be scalar",
		},
		{
			name:    "syntax error",
			data:    "a: : bad\n",
			wantErr: "genx.yaml:1: mapping values are not allowed in this context",
		},
		{
			name:    "alias expansion",
			data:    "base: &b hello\nref: *b\n",
			wantErr: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			node, err := Parse("genx.yaml", []byte(tc.data))
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Parse unexpected error: %v", err)
				}
				if node == nil {
					t.Fatal("expected non-nil node")
				}
			} else {
				if err == nil {
					t.Fatalf("Parse expected error %q, got nil", tc.wantErr)
				}
				if err.Error() != tc.wantErr {
					t.Fatalf("Parse error = %q, want %q", err.Error(), tc.wantErr)
				}
			}
		})
	}
}

func TestAliasExpansion(t *testing.T) {
	data := `
common: &common
  run: [go, test]
producers:
  - name: p1
    run: *common
`
	node, err := Parse("genx.yaml", []byte(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := MappingEntries("genx.yaml", node, "", nil, NewErrors("genx.yaml"))
	prodsEntry := FindEntry(entries, "producers")
	if prodsEntry == nil {
		t.Fatal("missing producers")
	}
	if prodsEntry.ValNode.Kind != yaml.SequenceNode {
		t.Fatalf("expected sequence, got %v", prodsEntry.ValNode.Kind)
	}
	p1 := prodsEntry.ValNode.Content[0]
	p1Entries := MappingEntries("genx.yaml", p1, "producers[0]", nil, NewErrors("genx.yaml"))
	runEntry := FindEntry(p1Entries, "run")
	if runEntry == nil {
		t.Fatal("missing run entry")
	}
	// Verify that run was expanded from *common to a mapping with run: [go, test]
	if runEntry.ValNode.Kind != yaml.MappingNode {
		t.Fatalf("expected run to be expanded to MappingNode, got kind %v", runEntry.ValNode.Kind)
	}
}

func TestAliasCycle(t *testing.T) {
	data := `
a: &a
  b: *a
`
	_, err := Parse("genx.yaml", []byte(data))
	if err == nil {
		t.Fatal("expected error on alias cycle")
	}
	if !strings.Contains(err.Error(), "alias cycle detected") {
		t.Fatalf("expected alias cycle error, got: %v", err)
	}
}

func TestMappingEntries(t *testing.T) {
	errs := NewErrors("genx.yaml")
	node, err := Parse("genx.yaml", []byte("a: 1\nb: 2\n"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	entries := MappingEntries("genx.yaml", node, "", []string{"a"}, errs)
	if len(entries) != 1 {
		t.Fatalf("expected 1 allowed entry, got %d", len(entries))
	}
	if err := errs.Err(); err == nil {
		t.Fatal("expected unknown field error")
	} else if err.Error() != `genx.yaml:2: b: unknown field` {
		t.Fatalf("got %q, want %q", err.Error(), `genx.yaml:2: b: unknown field`)
	}
}

func TestReaders(t *testing.T) {
	yamlStr := `
str: hello
int: 42
seq:
  - one
  - two
bad_str: 123
bad_int: "not an int"
bad_seq: not_a_seq
`
	node, err := Parse("test.yaml", []byte(yamlStr))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	errs := NewErrors("test.yaml")
	entries := MappingEntries("test.yaml", node, "", nil, errs)

	// String
	s, ok := ReadString(FindEntry(entries, "str").ValNode, "str", errs)
	if !ok || s != "hello" {
		t.Fatalf("ReadString = (%q, %v), want (hello, true)", s, ok)
	}
	_, ok = ReadString(FindEntry(entries, "bad_str").ValNode, "bad_str", errs)
	if ok {
		t.Fatal("expected ReadString to fail on bad_str")
	}

	// Int
	n, ok := ReadInt(FindEntry(entries, "int").ValNode, "int", errs)
	if !ok || n != 42 {
		t.Fatalf("ReadInt = (%d, %v), want (42, true)", n, ok)
	}
	_, ok = ReadInt(FindEntry(entries, "bad_int").ValNode, "bad_int", errs)
	if ok {
		t.Fatal("expected ReadInt to fail on bad_int")
	}

	// Seq
	seq, ok := ReadStringSlice(FindEntry(entries, "seq").ValNode, "seq", errs)
	if !ok || len(seq) != 2 || seq[0] != "one" || seq[1] != "two" {
		t.Fatalf("ReadStringSlice = (%v, %v)", seq, ok)
	}
	_, ok = ReadStringSlice(FindEntry(entries, "bad_seq").ValNode, "bad_seq", errs)
	if ok {
		t.Fatal("expected ReadStringSlice to fail on bad_seq")
	}

	allErr := errs.Err()
	if allErr == nil {
		t.Fatal("expected collected errors")
	}
	lines := strings.Split(allErr.Error(), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 errors, got %d: %v", len(lines), lines)
	}
	if lines[0] != "test.yaml:7: bad_str: must be a string" {
		t.Errorf("line 0: got %q", lines[0])
	}
	if lines[1] != "test.yaml:8: bad_int: must be an integer" {
		t.Errorf("line 1: got %q", lines[1])
	}
	if lines[2] != "test.yaml:9: bad_seq: must be a sequence" {
		t.Errorf("line 2: got %q", lines[2])
	}
}

func TestErrorsSorting(t *testing.T) {
	errs := NewErrors("test.yaml")
	errs.Add(10, "z", "msg 10")
	errs.Add(2, "b", "msg 2")
	errs.Add(5, "a", "msg 5")

	err := errs.Err()
	want := "test.yaml:2: b: msg 2\ntest.yaml:5: a: msg 5\ntest.yaml:10: z: msg 10"
	if err.Error() != want {
		t.Fatalf("got %q, want %q", err.Error(), want)
	}
}

func TestParseRefusals(t *testing.T) {
	bomb := "a: &a [x, x, x, x, x, x, x, x, x, x]\n"
	for i, prev := 1, "a"; i < 9; i++ {
		name := string(rune('a' + i))
		bomb += name + ": &" + name + " [" + strings.Repeat("*"+prev+", ", 9) + "*" + prev + "]\n"
		prev = name
	}
	for _, tt := range []struct{ name, src, want string }{
		{"a broken second document", "version: 1\n---\n: : [\n", "genx.yaml:4: did not find expected node content"},
		{"an alias bomb", bomb, "genx.yaml:1: the YAML aliases expand to too many values"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse("genx.yaml", []byte(tt.src))
			if err == nil || !strings.HasPrefix(err.Error(), tt.want) {
				t.Fatalf("err = %v, want it to start with %q", err, tt.want)
			}
		})
	}
	// An anchor used on two paths is fine.
	if _, err := Parse("genx.yaml", []byte("a: &a 1\nb: &b [*a]\nc: *b\nd: *b\n")); err != nil {
		t.Fatalf("shared alias: %v", err)
	}
}
