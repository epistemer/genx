package query_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/epistemer/genx/internal/query"
)

func TestRefusedItems(t *testing.T) {
	tests := []struct {
		name    string
		item    string
		src     string
		defs    string
		wantErr string
	}{
		// Clock and time zone (14 items)
		{
			name:    "units",
			item:    "now",
			src:     "now",
			wantErr: `units: "now" is not allowed in a query (it reads the clock)`,
		},
		{
			name:    "units",
			item:    "mktime",
			src:     "mktime",
			wantErr: `units: "mktime" is not allowed in a query (it reads the clock)`,
		},
		{
			name:    "units",
			item:    "gmtime",
			src:     "gmtime",
			wantErr: `units: "gmtime" is not allowed in a query (it reads the clock)`,
		},
		{
			name:    "units",
			item:    "localtime",
			src:     "localtime",
			wantErr: `units: "localtime" is not allowed in a query (it reads the clock)`,
		},
		{
			name:    "units",
			item:    "strftime",
			src:     `strftime("%Y")`,
			wantErr: `units: "strftime" is not allowed in a query (it reads the clock)`,
		},
		{
			name:    "units",
			item:    "strflocaltime",
			src:     `strflocaltime("%Y")`,
			wantErr: `units: "strflocaltime" is not allowed in a query (it reads the clock)`,
		},
		{
			name:    "units",
			item:    "strptime",
			src:     `strptime("%Y")`,
			wantErr: `units: "strptime" is not allowed in a query (it reads the clock)`,
		},
		{
			name:    "units",
			item:    "todate",
			src:     "todate",
			wantErr: `units: "todate" is not allowed in a query (it reads the clock)`,
		},
		{
			name:    "units",
			item:    "fromdate",
			src:     "fromdate",
			wantErr: `units: "fromdate" is not allowed in a query (it reads the clock)`,
		},
		{
			name:    "units",
			item:    "date",
			src:     "date",
			wantErr: `units: "date" is not allowed in a query (it reads the clock)`,
		},
		{
			name:    "units",
			item:    "dateadd",
			src:     `dateadd(1; "days")`,
			wantErr: `units: "dateadd" is not allowed in a query (it reads the clock)`,
		},
		{
			name:    "units",
			item:    "datesub",
			src:     `datesub(1; "days")`,
			wantErr: `units: "datesub" is not allowed in a query (it reads the clock)`,
		},
		{
			name:    "units",
			item:    "todateiso8601",
			src:     "todateiso8601",
			wantErr: `units: "todateiso8601" is not allowed in a query (it reads the clock)`,
		},
		{
			name:    "units",
			item:    "fromdateiso8601",
			src:     "fromdateiso8601",
			wantErr: `units: "fromdateiso8601" is not allowed in a query (it reads the clock)`,
		},
		// Environment (2 items)
		{
			name:    "units",
			item:    "env",
			src:     "env",
			wantErr: `units: "env" is not allowed in a query (it reads the environment)`,
		},
		{
			name:    "units",
			item:    "$ENV",
			src:     "$ENV",
			wantErr: `units: "$ENV" is not allowed in a query (it reads the environment)`,
		},
		// Other input (4 items)
		{
			name:    "units",
			item:    "input",
			src:     "input",
			wantErr: `units: "input" is not allowed in a query (it reads other input)`,
		},
		{
			name:    "units",
			item:    "inputs",
			src:     "inputs",
			wantErr: `units: "inputs" is not allowed in a query (it reads other input)`,
		},
		{
			name:    "units",
			item:    "input_filename",
			src:     "input_filename",
			wantErr: `units: "input_filename" is not allowed in a query (it reads other input)`,
		},
		{
			name:    "units",
			item:    "input_line_number",
			src:     "input_line_number",
			wantErr: `units: "input_line_number" is not allowed in a query (it reads other input)`,
		},
		// Side channels (4 items)
		{
			name:    "units",
			item:    "debug",
			src:     "debug",
			wantErr: `units: "debug" is not allowed in a query (it is a side channel)`,
		},
		{
			name:    "units",
			item:    "stderr",
			src:     "stderr",
			wantErr: `units: "stderr" is not allowed in a query (it is a side channel)`,
		},
		{
			name:    "units",
			item:    "halt",
			src:     "halt",
			wantErr: `units: "halt" is not allowed in a query (it is a side channel)`,
		},
		{
			name:    "units",
			item:    "halt_error",
			src:     "halt_error",
			wantErr: `units: "halt_error" is not allowed in a query (it is a side channel)`,
		},
		// Modules (5 items)
		{
			name:    "units",
			item:    "import",
			src:     `import "foo" as f; .`,
			wantErr: `units: "import" is not allowed in a query (modules are not allowed)`,
		},
		{
			name:    "units",
			item:    "include",
			src:     `include "foo"; .`,
			wantErr: `units: "include" is not allowed in a query (modules are not allowed)`,
		},
		{
			name:    "units",
			item:    "get_search_list",
			src:     "get_search_list",
			wantErr: `units: "get_search_list" is not allowed in a query (modules are not allowed)`,
		},
		{
			name:    "units",
			item:    "modulemeta",
			src:     "modulemeta",
			wantErr: `units: "modulemeta" is not allowed in a query (modules are not allowed)`,
		},
		{
			name:    "units",
			item:    "$__loc__",
			src:     "$__loc__",
			wantErr: `units: "$__loc__" is not allowed in a query (modules are not allowed)`,
		},
		// Forbidden item in defs
		{
			name:    "out",
			item:    "now in defs",
			defs:    "def get_time: now;",
			src:     "get_time",
			wantErr: `out: "now" is not allowed in a query (it reads the clock)`,
		},
		// Forbidden item nested in function definition arg
		{
			name:    "gen",
			item:    "$ENV in def arg",
			src:     "def f($ENV): 1; f(2)",
			wantErr: `gen: "$ENV" is not allowed in a query (it reads the environment)`,
		},
		// Forbidden item in pattern
		{
			name:    "gen",
			item:    "$ENV in pattern",
			src:     ". as $ENV | 1",
			wantErr: `gen: "$ENV" is not allowed in a query (it reads the environment)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.item, func(t *testing.T) {
			_, err := query.Compile(tt.name, tt.defs, tt.src)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tt.item)
			}
			if err.Error() != tt.wantErr {
				t.Errorf("error mismatch for %s:\ngot:  %q\nwant: %q", tt.item, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestParseError(t *testing.T) {
	_, err := query.Compile("myquery", "", "1 + * 2")
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
	expected := `myquery: unexpected token "*"`
	if err.Error() != expected {
		t.Errorf("got %q, want %q", err.Error(), expected)
	}
}

func TestErrorValuePassthrough(t *testing.T) {
	ctx := context.Background()

	t.Run("string error", func(t *testing.T) {
		q, err := query.Compile("test", "", `error("custom error message")`)
		if err != nil {
			t.Fatalf("compile error: %v", err)
		}
		_, err = q.Run(ctx, nil)
		if err == nil {
			t.Fatal("expected run error, got nil")
		}
		var qErr *query.Error
		if !errors.As(err, &qErr) {
			t.Fatalf("expected *query.Error, got %T: %v", err, err)
		}
		if qErr.Error() != "custom error message" {
			t.Errorf("got %q, want %q", qErr.Error(), "custom error message")
		}
		if qErr.Value != "custom error message" {
			t.Errorf("Value got %v, want %q", qErr.Value, "custom error message")
		}
	})

	t.Run("json object error", func(t *testing.T) {
		q, err := query.Compile("test", "", `error({code: 42, reason: "bad"})`)
		if err != nil {
			t.Fatalf("compile error: %v", err)
		}
		_, err = q.Run(ctx, nil)
		if err == nil {
			t.Fatal("expected run error, got nil")
		}
		var qErr *query.Error
		if !errors.As(err, &qErr) {
			t.Fatalf("expected *query.Error, got %T: %v", err, err)
		}
		expected := `{"code":42,"reason":"bad"}`
		if qErr.Error() != expected {
			t.Errorf("got %q, want %q", qErr.Error(), expected)
		}
	})
}

func TestTimeout(t *testing.T) {
	q, err := query.Compile("timeout_test", "", "def f: f; f")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = q.Run(ctx, nil)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded in error chain, got: %v", err)
	}
	expectedMsg := "query timed out: context deadline exceeded"
	if err.Error() != expectedMsg {
		t.Errorf("got %q, want %q", err.Error(), expectedMsg)
	}
}

func TestPrecisionThroughPipeline(t *testing.T) {
	cases := []string{
		"9007199254740993",
		"12345678901234567890",
	}

	for _, s := range cases {
		t.Run("JSON_"+s, func(t *testing.T) {
			input := []byte(s)
			decoded, err := query.Decode(input, false)
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}

			q, err := query.Compile("add1", "", ". + 1")
			if err != nil {
				t.Fatalf("compile error: %v", err)
			}

			vals, err := q.Run(context.Background(), decoded)
			if err != nil {
				t.Fatalf("run error: %v", err)
			}
			if len(vals) != 1 {
				t.Fatalf("expected 1 output value, got %d", len(vals))
			}

			encoded, err := query.Encode(vals[0])
			if err != nil {
				t.Fatalf("encode error: %v", err)
			}

			var expected string
			switch s {
			case "9007199254740993":
				expected = "9007199254740994\n"
			case "12345678901234567890":
				expected = "12345678901234567891\n"
			}

			if string(encoded) != expected {
				t.Errorf("precision loss:\ngot:  %q\nwant: %q", string(encoded), expected)
			}
		})

		t.Run("YAML_"+s, func(t *testing.T) {
			input := []byte(s + "\n")
			decoded, err := query.Decode(input, true)
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}

			q, err := query.Compile("add1", "", ". + 1")
			if err != nil {
				t.Fatalf("compile error: %v", err)
			}

			vals, err := q.Run(context.Background(), decoded)
			if err != nil {
				t.Fatalf("run error: %v", err)
			}
			if len(vals) != 1 {
				t.Fatalf("expected 1 output value, got %d", len(vals))
			}

			encoded, err := query.Encode(vals[0])
			if err != nil {
				t.Fatalf("encode error: %v", err)
			}

			var expected string
			switch s {
			case "9007199254740993":
				expected = "9007199254740994\n"
			case "12345678901234567890":
				expected = "12345678901234567891\n"
			}

			if string(encoded) != expected {
				t.Errorf("precision loss:\ngot:  %q\nwant: %q", string(encoded), expected)
			}
		})
	}
}

func TestKeyOrderDeterministic50Runs(t *testing.T) {
	data := map[string]any{
		"zebra": 1,
		"apple": 2,
		"mango": 3,
		"banana": map[string]any{
			"yellow": true,
			"green":  false,
		},
		"orange": []any{"round", "sweet"},
	}

	var first []byte
	for i := 0; i < 50; i++ {
		out, err := query.Encode(data)
		if err != nil {
			t.Fatalf("encode error on run %d: %v", i, err)
		}
		if i == 0 {
			first = out
		} else if string(out) != string(first) {
			t.Fatalf("nondeterministic key order on run %d:\nfirst: %s\ngot:   %s", i, string(first), string(out))
		}
	}
}

func TestGroupByStability(t *testing.T) {
	input := []any{
		map[string]any{"group": "B", "val": 1},
		map[string]any{"group": "A", "val": 2},
		map[string]any{"group": "B", "val": 3},
		map[string]any{"group": "A", "val": 4},
		map[string]any{"group": "B", "val": 5},
	}

	q, err := query.Compile("test_group", "", "group_by(.group)")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	vals, err := q.Run(context.Background(), input)
	if err != nil {
		t.Fatalf("run error: %v", err)
	}

	groups, ok := vals[0].([]any)
	if !ok || len(groups) != 2 {
		t.Fatalf("unexpected group structure: %#v", vals)
	}

	// Group "A" should be first because "A" < "B"
	groupA := groups[0].([]any)
	if len(groupA) != 2 {
		t.Fatalf("group A length mismatch: %d", len(groupA))
	}
	if groupA[0].(map[string]any)["val"] != 2 || groupA[1].(map[string]any)["val"] != 4 {
		t.Errorf("group A stability violated: %#v", groupA)
	}

	groupB := groups[1].([]any)
	if len(groupB) != 3 {
		t.Fatalf("group B length mismatch: %d", len(groupB))
	}
	if groupB[0].(map[string]any)["val"] != 1 || groupB[1].(map[string]any)["val"] != 3 || groupB[2].(map[string]any)["val"] != 5 {
		t.Errorf("group B stability violated: %#v", groupB)
	}
}

func TestLabelsQuery(t *testing.T) {
	src, err := os.ReadFile("testdata/labels.jq")
	if err != nil {
		t.Fatalf("failed to read testdata/labels.jq: %v", err)
	}

	q, err := query.Compile("labels", "", string(src))
	if err != nil {
		t.Fatalf("failed to compile labels.jq: %v", err)
	}

	doc1 := map[string]any{
		"domain":   "labels/v1",
		"producer": "settings",
		"package": map[string]any{
			"name": "settings",
			"dir":  "internal/settings",
		},
		"labels": []any{
			map[string]any{
				"key":  "title",
				"text": "Settings Title",
				"var":  "Title",
				"path": "title",
			},
			map[string]any{
				"key":  "hidden_label",
				"text": "Hidden Text",
			},
		},
	}

	t.Run("valid input produces units", func(t *testing.T) {
		units, err := q.Run(context.Background(), []any{doc1})
		if err != nil {
			t.Fatalf("run error: %v", err)
		}
		if len(units) != 1 {
			t.Fatalf("expected 1 unit, got %d", len(units))
		}

		unit, ok := units[0].(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any unit, got %T", units[0])
		}

		if unit["out"] != "internal/settings/zz_settings_i18n.go" {
			t.Errorf("out mismatch: got %v", unit["out"])
		}

		pkg := unit["package"].(map[string]any)
		if pkg["name"] != "settings" || pkg["dir"] != "internal/settings" {
			t.Errorf("package mismatch: got %#v", pkg)
		}

		vars := unit["vars"].([]any)
		if len(vars) != 1 {
			t.Errorf("expected 1 var, got %d", len(vars))
		}

		hidden := unit["hidden"].([]any)
		if len(hidden) != 1 {
			t.Errorf("expected 1 hidden label, got %d", len(hidden))
		}

		paths := unit["paths"].([]any)
		if len(paths) != 1 {
			t.Errorf("expected 1 path label, got %d", len(paths))
		}
	})

	t.Run("one key two texts clash error", func(t *testing.T) {
		doc2 := map[string]any{
			"domain":   "labels/v1",
			"producer": "models",
			"package": map[string]any{
				"name": "models",
				"dir":  "internal/models",
			},
			"labels": []any{
				map[string]any{
					"key":  "title",
					"text": "Models Different Title",
					"var":  "ModelsTitle",
				},
			},
		}

		_, err := q.Run(context.Background(), []any{doc1, doc2})
		if err == nil {
			t.Fatal("expected clash error, got nil")
		}

		var qErr *query.Error
		if !errors.As(err, &qErr) {
			t.Fatalf("expected *query.Error, got %T: %v", err, err)
		}

		expectedMsg := "labels: one key, two texts: title (models, settings)"
		if qErr.Error() != expectedMsg {
			t.Errorf("clash error message mismatch:\ngot:  %q\nwant: %q", qErr.Error(), expectedMsg)
		}
	})
}

func TestDecodeErrorsAndEdgeCases(t *testing.T) {
	t.Run("JSON trailing data", func(t *testing.T) {
		_, err := query.Decode([]byte(`{"a": 1} trailing`), false)
		if err == nil {
			t.Fatal("expected error on trailing JSON data, got nil")
		}
	})

	t.Run("JSON multiple values", func(t *testing.T) {
		_, err := query.Decode([]byte(`{"a": 1} {"b": 2}`), false)
		if err == nil {
			t.Fatal("expected error on multiple JSON values, got nil")
		}
		if err.Error() != "unexpected trailing data after JSON value" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("YAML multiple documents", func(t *testing.T) {
		yamlData := []byte("a: 1\n---\nb: 2\n")
		_, err := query.Decode(yamlData, true)
		if err == nil {
			t.Fatal("expected error on multiple YAML documents, got nil")
		}
		if err.Error() != "multiple YAML documents in data" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("YAML .nan refused", func(t *testing.T) {
		_, err := query.Decode([]byte("val: .nan\n"), true)
		if err == nil {
			t.Fatal("expected error on .nan, got nil")
		}
		if err.Error() != ".nan is not allowed in YAML" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("YAML .inf refused", func(t *testing.T) {
		_, err := query.Decode([]byte("val: .inf\n"), true)
		if err == nil {
			t.Fatal("expected error on .inf, got nil")
		}
		if err.Error() != ".inf is not allowed in YAML" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("YAML -.inf refused", func(t *testing.T) {
		_, err := query.Decode([]byte("val: -.inf\n"), true)
		if err == nil {
			t.Fatal("expected error on -.inf, got nil")
		}
		if err.Error() != ".inf is not allowed in YAML" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("YAML timestamp stays string", func(t *testing.T) {
		v, err := query.Decode([]byte("ts: 2026-11-13T05:39:23Z\n"), true)
		if err != nil {
			t.Fatalf("unexpected decode error: %v", err)
		}
		m, ok := v.(map[string]any)
		if !ok {
			t.Fatalf("expected map, got %T", v)
		}
		ts, ok := m["ts"].(string)
		if !ok {
			t.Fatalf("expected timestamp to be string, got %T: %#v", m["ts"], m["ts"])
		}
		if ts != "2026-11-13T05:39:23Z" {
			t.Errorf("got %q, want %q", ts, "2026-11-13T05:39:23Z")
		}
	})

	t.Run("YAML non-string key refused", func(t *testing.T) {
		_, err := query.Decode([]byte("123: value\n"), true)
		if err == nil {
			t.Fatal("expected error for non-string key, got nil")
		}
		if err.Error() != "mapping keys must be strings: found tag !!int" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("YAML aliases and merge keys expanded", func(t *testing.T) {
		src := `
base: &base
  x: 10
  y: 20
child:
  <<: *base
  y: 99
  z: 30
alias_ref: *base
`
		v, err := query.Decode([]byte(src), true)
		if err != nil {
			t.Fatalf("unexpected decode error: %v", err)
		}
		m := v.(map[string]any)
		child := m["child"].(map[string]any)
		if !reflect.DeepEqual(child["x"], 10) || !reflect.DeepEqual(child["y"], 99) || !reflect.DeepEqual(child["z"], 30) {
			t.Errorf("merge key expansion failed: %#v", child)
		}
		aliasRef := m["alias_ref"].(map[string]any)
		if !reflect.DeepEqual(aliasRef["x"], 10) || !reflect.DeepEqual(aliasRef["y"], 20) {
			t.Errorf("alias expansion failed: %#v", aliasRef)
		}
	})

	t.Run("YAML alias reached on two paths", func(t *testing.T) {
		v, err := query.Decode([]byte("a: &a 1\nb: &b [*a]\nc: *b\nd: *b\n"), true)
		if err != nil {
			t.Fatalf("unexpected decode error: %v", err)
		}
		want := map[string]any{"a": 1, "b": []any{1}, "c": []any{1}, "d": []any{1}}
		if !reflect.DeepEqual(v, want) {
			t.Errorf("got %#v, want %#v", v, want)
		}
	})

	t.Run("YAML merge sequence: earlier mappings win", func(t *testing.T) {
		v, err := query.Decode([]byte("p: &p {x: 1, y: 1}\nq: &q {x: 2, z: 2}\nr:\n  <<: [*p, *q]\n  y: 3\n"), true)
		if err != nil {
			t.Fatalf("unexpected decode error: %v", err)
		}
		want := map[string]any{"x": 1, "y": 3, "z": 2}
		if got := v.(map[string]any)["r"]; !reflect.DeepEqual(got, want) {
			t.Errorf("got %#v, want %#v", got, want)
		}
	})

	t.Run("YAML alias cycle refused", func(t *testing.T) {
		if _, err := query.Decode([]byte("a: &a [*a]\n"), true); err == nil {
			t.Fatal("expected an error for a cycle, got nil")
		}
	})

	t.Run("YAML alias bomb refused", func(t *testing.T) {
		src := "a: &a [x, x, x, x, x, x, x, x, x, x]\n"
		for i, prev := 1, "a"; i < 9; i++ {
			name := string(rune('a' + i))
			src += fmt.Sprintf("%s: &%s [*%s, *%s, *%s, *%s, *%s, *%s, *%s, *%s, *%s, *%s]\n", name, name, prev, prev, prev, prev, prev, prev, prev, prev, prev, prev)
			prev = name
		}
		_, err := query.Decode([]byte(src), true)
		if err == nil || err.Error() != "the YAML aliases expand to too many values" {
			t.Fatalf("err = %v, want the expansion limit", err)
		}
	})
}
