package manifest

import (
	"reflect"
	"testing"
)

func TestUseMapForm(t *testing.T) {
	data := `version: 1
use:
  - example.com/a: [x]
  - example.com/b:
      generators: [y, z]
      params:
        dir: db
        package: null
        tags: [a, b]
        n: 2
  - example.com/c: {generators: [w]}
  - example.com/d:
      generators: [v]
      params:
        base: &b {x: 1}
        merged: {<<: *b, y: 2}
`
	m, err := Parse("genx.yaml", []byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Use) != 4 {
		t.Fatalf("len(Use) = %d", len(m.Use))
	}
	if u := m.Use[0]; u.Package != "example.com/a" || !reflect.DeepEqual(u.Generators, []string{"x"}) || u.Params != nil || u.Path != "use[0].example.com/a" {
		t.Errorf("list form = %+v", u)
	}
	u := m.Use[1]
	if u.Package != "example.com/b" || !reflect.DeepEqual(u.Generators, []string{"y", "z"}) || u.Path != "use[1].example.com/b" {
		t.Errorf("map form = %+v", u)
	}
	want := []Param{
		{Name: "dir", Value: "db", Line: 7},
		{Name: "package", Value: nil, Line: 8},
		{Name: "tags", Value: []any{"a", "b"}, Line: 9},
		{Name: "n", Value: 2, Line: 10},
	}
	if !reflect.DeepEqual(u.Params, want) {
		t.Errorf("params = %#v\nwant     %#v", u.Params, want)
	}
	if u := m.Use[2]; !reflect.DeepEqual(u.Generators, []string{"w"}) || u.Params != nil {
		t.Errorf("map form without params = %+v", u)
	}
	// Anchors, aliases and merge keys in values.
	wantD := []Param{
		{Name: "base", Value: map[string]any{"x": 1}, Line: 15},
		{Name: "merged", Value: map[string]any{"x": 1, "y": 2}, Line: 16},
	}
	if u := m.Use[3]; !reflect.DeepEqual(u.Params, wantD) {
		t.Errorf("aliases = %#v", u.Params)
	}

	// params: {} sets nothing.
	m, err = Parse("genx.yaml", []byte("version: 1\nuse:\n  - example.com/a: {generators: [x], params: {}}\n"))
	if err != nil || len(m.Use[0].Params) != 0 {
		t.Errorf("empty params: %+v, %v", m.Use, err)
	}
}

func TestUseMapFormErrors(t *testing.T) {
	for _, tt := range []struct {
		name, use, want string
	}{
		{"no generators", "  - example.com/a: {params: {dir: db}}\n",
			"genx.yaml:3: use[0].example.com/a.generators: required"},
		{"no generators, block style", "  - example.com/a:\n      params: {dir: db}\n",
			"genx.yaml:3: use[0].example.com/a.generators: required"},
		{"generators not a sequence", "  - example.com/a: {generators: x}\n",
			"genx.yaml:3: use[0].example.com/a.generators: must be a sequence of generator names"},
		{"empty generators", "  - example.com/a: {generators: []}\n",
			"genx.yaml:3: use[0].example.com/a.generators: must not be empty"},
		{"an unknown field", "  - example.com/a: {generators: [x], param: {}}\n",
			"genx.yaml:3: use[0].example.com/a.param: unknown field"},
		{"params not a mapping", "  - example.com/a: {generators: [x], params: [dir]}\n",
			"genx.yaml:3: use[0].example.com/a.params: must be a mapping of parameter names to values"},
		{"a scalar entry", "  - example.com/a: x\n",
			"genx.yaml:3: use[0].example.com/a: must be a sequence of generator names, or a mapping {generators, params}"},
		{"a duplicate parameter", "  - example.com/a:\n      generators: [x]\n      params:\n        dir: a\n        dir: b\n",
			`genx.yaml:7: use[0].example.com/a.params.dir: duplicate key "dir"`},
		{"a .inf value", "  - example.com/a: {generators: [x], params: {n: .inf}}\n",
			"genx.yaml:3: use[0].example.com/a.params.n: .inf is not allowed in YAML"},
		{"a generator twice across forms", "  - example.com/a: [x]\n  - example.com/a: {generators: [x]}\n",
			`genx.yaml:4: use[1].example.com/a.generators: duplicate generator "x" for package "example.com/a"`},
		{"a nested .inf, at its own line", "  - example.com/a:\n      generators: [x]\n      params:\n        n:\n          a: 1\n          b: .inf\n",
			"genx.yaml:8: use[0].example.com/a.params.n: .inf is not allowed in YAML"},
		{"params checked without generators", "  - example.com/a: {params: [dir]}\n",
			"genx.yaml:3: use[0].example.com/a.generators: required\ngenx.yaml:3: use[0].example.com/a.params: must be a mapping of parameter names to values"},
		{"an empty list form", "  - example.com/a: []\n",
			"genx.yaml:3: use[0].example.com/a: must not be empty"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse("genx.yaml", []byte("version: 1\nuse:\n"+tt.use))
			if err == nil || err.Error() != tt.want {
				t.Errorf("got  %v\nwant %s", err, tt.want)
			}
		})
	}
}
