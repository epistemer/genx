package meta

import (
	"reflect"
	"testing"
)

const paramsGen = "generators:\n  g:\n    doc: d\n    needs: [l/v1]\n    units: u\n    out: o\n    template: t.tmpl\n"

func TestParseParams(t *testing.T) {
	data := `version: 1
params:
  dir: {doc: the directory, default: models}
  package:
    doc: the name; null means the last element of dir
    default: null
  table_prefix: {doc: a prefix, default: ""}
  n2: {doc: a number, default: 3}
  big: {doc: an integer past 2^53, default: 12345678901234567890}
  list: {doc: a list, default: [a, 1]}
  obj: {doc: an object, default: {b: true}}
` + paramsGen
	m, err := Parse("genx.meta.yaml", []byte(data))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, p := range m.Params {
		names = append(names, p.Name)
	}
	// The metafile's order.
	if want := []string{"dir", "package", "table_prefix", "n2", "big", "list", "obj"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
	byName := map[string]Param{}
	for _, p := range m.Params {
		byName[p.Name] = p
	}
	if p := byName["dir"]; p.Default != "models" || p.Doc != "the directory" || p.Line != 3 {
		t.Errorf("dir = %+v", p)
	}
	if p := byName["package"]; p.Default != nil || p.Line != 4 {
		t.Errorf("package = %+v", p)
	}
	if p := byName["table_prefix"]; p.Default != "" {
		t.Errorf("table_prefix = %+v", p)
	}
	if p := byName["n2"]; p.Default != 3 {
		t.Errorf("n2 = %#v", p.Default)
	}
	if p := byName["big"]; p.Default == nil || reflect.TypeOf(p.Default).String() != "*big.Int" {
		t.Errorf("big = %#v", p.Default)
	}
	if p := byName["list"]; !reflect.DeepEqual(p.Default, []any{"a", 1}) {
		t.Errorf("list = %#v", p.Default)
	}
	if p := byName["obj"]; !reflect.DeepEqual(p.Default, map[string]any{"b": true}) {
		t.Errorf("obj = %#v", p.Default)
	}

	// No params: none.
	m, err = Parse("genx.meta.yaml", []byte("version: 1\n"+paramsGen))
	if err != nil || m.Params != nil {
		t.Errorf("no params: %v, %v", m.Params, err)
	}
	// An empty mapping declares none.
	m, err = Parse("genx.meta.yaml", []byte("version: 1\nparams: {}\n"+paramsGen))
	if err != nil || len(m.Params) != 0 {
		t.Errorf("empty params: %v, %v", m.Params, err)
	}
}

func TestParseParamsErrors(t *testing.T) {
	for _, tt := range []struct {
		name, params, want string
	}{
		{"not a mapping", "params: [dir]\n",
			"genx.meta.yaml:2: params: must be a mapping of parameter names to {doc, default}"},
		{"an invalid name", "params:\n  table-prefix: {doc: d, default: x}\n",
			`genx.meta.yaml:3: params.table-prefix: "table-prefix" is not a valid parameter name (a lowercase letter, then lowercase letters, digits and _)`},
		{"a name starting with _", "params:\n  _x: {doc: d, default: x}\n",
			`genx.meta.yaml:3: params._x: "_x" is not a valid parameter name (a lowercase letter, then lowercase letters, digits and _)`},
		{"a scalar declaration", "params:\n  dir: models\n",
			"genx.meta.yaml:3: params.dir: must be a mapping {doc, default}"},
		{"no doc", "params:\n  dir: {default: x}\n",
			"genx.meta.yaml:3: params.dir.doc: required"},
		{"an empty doc", "params:\n  dir: {doc: \"\", default: x}\n",
			"genx.meta.yaml:3: params.dir.doc: must not be empty"},
		{"a doc of two lines", "params:\n  dir: {doc: \"a\\nb\", default: x}\n",
			"genx.meta.yaml:3: params.dir.doc: must be one line"},
		{"a doc that isn't a string", "params:\n  dir: {doc: [a], default: x}\n",
			"genx.meta.yaml:3: params.dir.doc: must be a string"},
		{"no default", "params:\n  dir: {doc: d}\n",
			"genx.meta.yaml:3: params.dir.default: required (write default: null for a parameter with no default value)"},
		{"an unknown field", "params:\n  dir: {doc: d, default: x, type: string}\n",
			"genx.meta.yaml:3: params.dir.type: unknown field"},
		{"a .nan default", "params:\n  dir: {doc: d, default: .nan}\n",
			"genx.meta.yaml:3: params.dir.default: .nan is not allowed in YAML"},
		{"a nested .nan, at its own line", "params:\n  dir:\n    doc: d\n    default:\n      - 1\n      - .nan\n",
			"genx.meta.yaml:7: params.dir.default: .nan is not allowed in YAML"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse("genx.meta.yaml", []byte("version: 1\n"+tt.params+paramsGen))
			if err == nil || err.Error() != tt.want {
				t.Errorf("got  %v\nwant %s", err, tt.want)
			}
		})
	}
}
