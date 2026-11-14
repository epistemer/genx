package query

import (
	"context"
	"math/big"
	"reflect"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestVariables(t *testing.T) {
	// A variable Compile names is bound in the defs and in the query.
	q, err := Compile("q", `def pkg: $params.package // ($params.dir | split("/") | last);`,
		`[pkg, $params.dir, .x]`, "$params")
	if err != nil {
		t.Fatal(err)
	}
	got, err := q.Run(context.Background(), map[string]any{"x": 1},
		map[string]any{"dir": "internal/db", "package": nil})
	if err != nil {
		t.Fatal(err)
	}
	if want := []any{[]any{"db", "internal/db", 1}}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// A query that reads a variable Compile doesn't name doesn't compile.
	if _, err := Compile("q", "", "$params"); err == nil {
		t.Error("an unbound $params compiled")
	}

	// A query may still bind the name itself; its own binding wins inside.
	q, err = Compile("q", "", `1 as $params | $params`, "$params")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := q.Run(context.Background(), nil, map[string]any{}); err != nil || !reflect.DeepEqual(got, []any{1}) {
		t.Errorf("shadowed: %v, %v", got, err)
	}
}

func TestKindCompactNormalize(t *testing.T) {
	bi, _ := new(big.Int).SetString("12345678901234567890", 10)
	for _, tt := range []struct {
		v             any
		kind, compact string
	}{
		{nil, "null", "null"},
		{true, "boolean", "true"},
		{1, "number", "1"},
		{1.5, "number", "1.5"},
		{bi, "number", "12345678901234567890"},
		{"a<b", "string", `"a<b"`},
		{[]any{"a", 1}, "array", `["a",1]`},
		{map[string]any{"b": 1, "a": nil}, "object", `{"a":null,"b":1}`},
	} {
		if got := Kind(tt.v); got != tt.kind {
			t.Errorf("Kind(%v) = %s, want %s", tt.v, got, tt.kind)
		}
		if got := Compact(tt.v); got != tt.compact {
			t.Errorf("Compact(%v) = %s, want %s", tt.v, got, tt.compact)
		}
	}

	// A value that isn't decoded isn't taken for a number.
	if got := Kind([]string{"x"}); got != "Go []string" {
		t.Errorf("Kind([]string) = %s", got)
	}

	// Normalize gives what a query reads from JSON.
	got, err := Normalize(map[string]any{"s": []string{"x"}, "n": int64(3), "f": float32(0.5), "big": uint64(1 << 63)})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"s": []any{"x"}, "n": 3, "f": 0.5, "big": new(big.Int).SetUint64(1 << 63)}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Normalize = %#v\nwant        %#v", got, want)
	}
	if _, err := Normalize(func() {}); err == nil {
		t.Error("Normalize accepted a func")
	}
}

func TestDecodeNode(t *testing.T) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte("{a: [1, x, null], b: {c: true}, d: 2026-11-14}"), &doc); err != nil {
		t.Fatal(err)
	}
	got, err := DecodeNode(doc.Content[0])
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"a": []any{1, "x", nil}, "b": map[string]any{"c": true}, "d": "2026-11-14"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DecodeNode = %#v", got)
	}
}
