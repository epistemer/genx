package run

import (
	"reflect"
	"testing"

	"github.com/epistemer/genx/internal/meta"
)

func TestBindParams(t *testing.T) {
	declared := []meta.Param{
		{Name: "table_prefix", Default: ""},
		{Name: "dir", Default: "models"},
		{Name: "package", Default: nil},
		{Name: "n", Default: 1},
	}
	got, err := bindParams("example.com/m", declared, []setting{
		{name: "dir", value: "db", where: "w"},
		{name: "package", value: []any{"any", "type"}, where: "w"},
		{name: "n", value: 2.5, where: "w"}, // an integer and a float are both numbers
	})
	if err != nil {
		t.Fatal(err)
	}
	// By name, the app's value or else the default.
	want := []param{
		{name: "dir", value: "db", byApp: true},
		{name: "n", value: 2.5, byApp: true},
		{name: "package", value: []any{"any", "type"}, byApp: true},
		{name: "table_prefix", value: ""},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
	obj := paramsObject(got)
	if wantObj := map[string]any{"dir": "db", "n": 2.5, "package": []any{"any", "type"}, "table_prefix": ""}; !reflect.DeepEqual(obj, wantObj) {
		t.Errorf("$params = %v", obj)
	}

	// Nothing declared, nothing set: $params is {}.
	got, err = bindParams("example.com/m", nil, nil)
	if err != nil || len(got) != 0 || len(paramsObject(got)) != 0 {
		t.Errorf("none: %v, %v", got, err)
	}

	for _, tt := range []struct {
		name string
		set  []setting
		want string
	}{
		{"unknown", []setting{{name: "colour", value: "x", where: "w"}},
			`w: example.com/m declares no parameter "colour" (it declares: dir, n, package, table_prefix)`},
		{"a string for a number", []setting{{name: "n", value: "1", where: "w"}},
			"w: must be a number, as its default 1 is (got a string)"},
		{"an object for a string", []setting{{name: "dir", value: map[string]any{}, where: "w"}},
			`w: must be a string, as its default "models" is (got an object)`},
		{"null for a string", []setting{{name: "table_prefix", value: nil, where: "w"}},
			`w: must be a string, as its default "" is (got null)`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := bindParams("example.com/m", declared, tt.set)
			if err == nil || err.Error() != tt.want {
				t.Errorf("got  %v\nwant %s", err, tt.want)
			}
		})
	}
}
