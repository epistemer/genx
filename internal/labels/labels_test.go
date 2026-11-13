package labels_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/epistemer/genx/internal/labels"
	"github.com/epistemer/genx/internal/query"
)

func TestConformanceValid(t *testing.T) {
	files, err := filepath.Glob("../../testdata/labels/valid/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no valid conformance files found")
	}

	for _, file := range files {
		name := filepath.Base(file)
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("ReadFile(%s): %v", file, err)
			}
			v, err := query.Decode(data, false)
			if err != nil {
				t.Fatalf("query.Decode(%s): %v", file, err)
			}
			if err := labels.Check(v); err != nil {
				t.Fatalf("labels.Check(%s) unexpected error: %v", file, err)
			}
		})
	}
}

func TestConformanceInvalid(t *testing.T) {
	jsonFiles, err := filepath.Glob("../../testdata/labels/invalid/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(jsonFiles) == 0 {
		t.Fatal("no invalid conformance files found")
	}

	errFiles, err := filepath.Glob("../../testdata/labels/invalid/*.err")
	if err != nil {
		t.Fatal(err)
	}
	if len(jsonFiles) != len(errFiles) {
		t.Fatalf("mismatched count: %d .json files vs %d .err files", len(jsonFiles), len(errFiles))
	}

	for _, file := range jsonFiles {
		name := filepath.Base(file)
		testName := strings.TrimSuffix(name, ".json")
		errFile := filepath.Join("../../testdata/labels/invalid", testName+".err")

		t.Run(testName, func(t *testing.T) {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("ReadFile(%s): %v", file, err)
			}
			wantErrBytes, err := os.ReadFile(errFile)
			if err != nil {
				t.Fatalf("ReadFile(%s): %v", errFile, err)
			}
			wantErr := strings.TrimRight(string(wantErrBytes), "\r\n")

			v, err := query.Decode(data, false)
			if err != nil {
				t.Fatalf("query.Decode(%s): %v", file, err)
			}

			checkErr := labels.Check(v)
			if checkErr == nil {
				t.Fatalf("labels.Check(%s) succeeded, want error %q", file, wantErr)
			}

			if checkErr.Error() != wantErr {
				t.Fatalf("labels.Check(%s):\ngot:  %q\nwant: %q", file, checkErr.Error(), wantErr)
			}
		})
	}
}

func TestCheckMultipleErrors(t *testing.T) {
	doc := map[string]any{
		"domain":   "labels/v2",
		"producer": "",
		"package": map[string]any{
			"name": "func",
			"dir":  "../x",
		},
		"labels": []any{
			map[string]any{
				"key":  "",
				"text": "Hello",
				"var":  "l10n",
			},
		},
	}

	err := labels.Check(doc)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	expectedErrors := []string{
		`domain: want "labels/v1", got "labels/v2"`,
		`producer: must not be empty`,
		`package.name: "func" is a Go keyword`,
		`package.dir: "../x": must not contain ".."`,
		`labels[0].key: must not be empty`,
		`labels[0].var: "l10n" is reserved`,
	}

	got := err.Error()
	for _, expected := range expectedErrors {
		if !strings.Contains(got, expected) {
			t.Errorf("error missing %q; got:\n%s", expected, got)
		}
	}
}
