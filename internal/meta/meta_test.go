package meta

import (
	"strings"
	"testing"

	"github.com/epistemer/genx/internal/domain"
)

func TestParseValid(t *testing.T) {
	metaYAML := `version: 1

defs: |
  def pascal: [splits("[^\\p{L}\\p{N}]+") | select(length > 0)
               | (.[0:1] | ascii_upcase) + .[1:]] | join("");

generators:
  labels:
    doc: i18n keys for every label, one file per package
    needs: [labels/v1]
    units: |
      .["labels/v1"] | group_by(.package.dir)[]
      | {package: .[0].package, labels: [.[].labels[]]}
    out: '"\(.package.dir)/zz_\(.package.name)_i18n.go"'
    template: labels.go.tmpl

  enum-labels:
    doc: a labels file for the enums in sqlc's request
    needs: [sqlc.request/v1]
    emits: labels/v1
    units: |
      .["sqlc.request/v1"][0]
    out: '"models/zz_enums.labels.json"'
`
	m, err := Parse("genx.meta.yaml", []byte(metaYAML))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.Version != 1 {
		t.Errorf("version = %d, want 1", m.Version)
	}
	if !strings.Contains(m.Defs, "def pascal") {
		t.Errorf("defs = %q", m.Defs)
	}
	if len(m.Generators) != 2 {
		t.Fatalf("len(Generators) = %d, want 2", len(m.Generators))
	}

	g0 := m.Generators[0]
	if g0.Name != "labels" {
		t.Errorf("g0.Name = %q, want labels", g0.Name)
	}
	if g0.Doc != "i18n keys for every label, one file per package" {
		t.Errorf("g0.Doc = %q", g0.Doc)
	}
	if len(g0.Needs) != 1 || g0.Needs[0] != (domain.Domain{Name: "labels", Major: 1}) {
		t.Errorf("g0.Needs = %v", g0.Needs)
	}
	if !strings.Contains(g0.Units, "group_by") {
		t.Errorf("g0.Units = %q", g0.Units)
	}
	if g0.UnitsLine == 0 {
		t.Error("expected non-zero UnitsLine")
	}
	if !strings.Contains(g0.Out, "zz_") {
		t.Errorf("g0.Out = %q", g0.Out)
	}
	if g0.OutLine == 0 {
		t.Error("expected non-zero OutLine")
	}
	if g0.Template != "labels.go.tmpl" || g0.HasEmits {
		t.Errorf("g0 template/emits = %q, %v", g0.Template, g0.HasEmits)
	}

	g1 := m.Generators[1]
	if g1.Name != "enum-labels" {
		t.Errorf("g1.Name = %q, want enum-labels", g1.Name)
	}
	if !g1.HasEmits || g1.Emits != (domain.Domain{Name: "labels", Major: 1}) {
		t.Errorf("g1 emits = %+v, hasEmits = %v", g1.Emits, g1.HasEmits)
	}
	if g1.Template != "" {
		t.Errorf("g1.Template = %q, want empty", g1.Template)
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr string
	}{
		{
			name:    "missing version",
			data:    "generators:\n  g:\n    doc: d\n    needs: [l/v1]\n    units: u\n    out: o\n    template: t.tmpl\n",
			wantErr: "genx.meta.yaml:1: version: required",
		},
		{
			name:    "version newer than supported",
			data:    "version: 2\ngenerators:\n  g:\n    doc: d\n    needs: [l/v1]\n    units: u\n    out: o\n    template: t.tmpl\n",
			wantErr: "genx.meta.yaml:1: version: version 2 is newer than this genx supports; upgrade genx",
		},
		{
			name:    "version 0",
			data:    "version: 0\ngenerators:\n  g:\n    doc: d\n    needs: [l/v1]\n    units: u\n    out: o\n    template: t.tmpl\n",
			wantErr: "genx.meta.yaml:1: version: must be 1",
		},
		{
			name:    "unknown top-level field",
			data:    "version: 1\nextra: val\ngenerators:\n  g:\n    doc: d\n    needs: [l/v1]\n    units: u\n    out: o\n    template: t.tmpl\n",
			wantErr: "genx.meta.yaml:2: extra: unknown field",
		},
		{
			name:    "missing generators",
			data:    "version: 1\n",
			wantErr: "genx.meta.yaml:1: generators: required",
		},
		{
			name:    "generators not mapping",
			data:    "version: 1\ngenerators: []\n",
			wantErr: "genx.meta.yaml:2: generators: must be a mapping",
		},
		{
			name:    "generators empty mapping",
			data:    "version: 1\ngenerators: {}\n",
			wantErr: "genx.meta.yaml:2: generators: must not be empty",
		},
		{
			name:    "invalid generator name",
			data:    "version: 1\ngenerators:\n  BadName:\n    doc: d\n    needs: [l/v1]\n    units: u\n    out: o\n    template: t.tmpl\n",
			wantErr: `genx.meta.yaml:3: generators.BadName: "BadName" is not a valid generator name (a lowercase letter, then lowercase letters, digits and -)`,
		},
		{
			name:    "generator unknown field",
			data:    "version: 1\ngenerators:\n  g:\n    doc: d\n    needs: [l/v1]\n    units: u\n    out: o\n    template: t.tmpl\n    extra: 1\n",
			wantErr: "genx.meta.yaml:9: generators.g.extra: unknown field",
		},
		{
			name:    "missing doc",
			data:    "version: 1\ngenerators:\n  g:\n    needs: [l/v1]\n    units: u\n    out: o\n    template: t.tmpl\n",
			wantErr: "genx.meta.yaml:4: generators.g.doc: required",
		},
		{
			name:    "empty doc",
			data:    "version: 1\ngenerators:\n  g:\n    doc: \"\"\n    needs: [l/v1]\n    units: u\n    out: o\n    template: t.tmpl\n",
			wantErr: "genx.meta.yaml:4: generators.g.doc: must not be empty",
		},
		{
			name:    "multiline doc",
			data:    "version: 1\ngenerators:\n  g:\n    doc: |\n      line 1\n      line 2\n    needs: [l/v1]\n    units: u\n    out: o\n    template: t.tmpl\n",
			wantErr: "genx.meta.yaml:4: generators.g.doc: must be one line",
		},
		{
			name:    "missing needs",
			data:    "version: 1\ngenerators:\n  g:\n    doc: d\n    units: u\n    out: o\n    template: t.tmpl\n",
			wantErr: "genx.meta.yaml:4: generators.g.needs: required",
		},
		{
			name:    "needs not sequence",
			data:    "version: 1\ngenerators:\n  g:\n    doc: d\n    needs: invalid\n    units: u\n    out: o\n    template: t.tmpl\n",
			wantErr: "genx.meta.yaml:5: generators.g.needs: must be a sequence",
		},
		{
			name:    "needs empty",
			data:    "version: 1\ngenerators:\n  g:\n    doc: d\n    needs: []\n    units: u\n    out: o\n    template: t.tmpl\n",
			wantErr: "genx.meta.yaml:5: generators.g.needs: must not be empty",
		},
		{
			name:    "needs invalid domain",
			data:    "version: 1\ngenerators:\n  g:\n    doc: d\n    needs: [bad]\n    units: u\n    out: o\n    template: t.tmpl\n",
			wantErr: `genx.meta.yaml:5: generators.g.needs[0]: "bad": invalid domain (want name/vN, such as labels/v1)`,
		},
		{
			name:    "missing units",
			data:    "version: 1\ngenerators:\n  g:\n    doc: d\n    needs: [l/v1]\n    out: o\n    template: t.tmpl\n",
			wantErr: "genx.meta.yaml:4: generators.g.units: required",
		},
		{
			name:    "empty units",
			data:    "version: 1\ngenerators:\n  g:\n    doc: d\n    needs: [l/v1]\n    units: \"\"\n    out: o\n    template: t.tmpl\n",
			wantErr: "genx.meta.yaml:6: generators.g.units: must not be empty",
		},
		{
			name:    "missing out",
			data:    "version: 1\ngenerators:\n  g:\n    doc: d\n    needs: [l/v1]\n    units: u\n    template: t.tmpl\n",
			wantErr: "genx.meta.yaml:4: generators.g.out: required",
		},
		{
			name:    "empty out",
			data:    "version: 1\ngenerators:\n  g:\n    doc: d\n    needs: [l/v1]\n    units: u\n    out: \"\"\n    template: t.tmpl\n",
			wantErr: "genx.meta.yaml:7: generators.g.out: must not be empty",
		},
		{
			name:    "neither template nor emits",
			data:    "version: 1\ngenerators:\n  g:\n    doc: d\n    needs: [l/v1]\n    units: u\n    out: o\n",
			wantErr: "genx.meta.yaml:3: generators.g: exactly one of template and emits",
		},
		{
			name:    "both template and emits",
			data:    "version: 1\ngenerators:\n  g:\n    doc: d\n    needs: [l/v1]\n    units: u\n    out: o\n    template: t.tmpl\n    emits: l/v1\n",
			wantErr: "genx.meta.yaml:3: generators.g: exactly one of template and emits",
		},
		{
			name:    "template empty",
			data:    "version: 1\ngenerators:\n  g:\n    doc: d\n    needs: [l/v1]\n    units: u\n    out: o\n    template: \"\"\n",
			wantErr: "genx.meta.yaml:8: generators.g.template: must not be empty",
		},
		{
			name:    "template contains slash",
			data:    "version: 1\ngenerators:\n  g:\n    doc: d\n    needs: [l/v1]\n    units: u\n    out: o\n    template: a/b.tmpl\n",
			wantErr: "genx.meta.yaml:8: generators.g.template: must not contain '/' or '\\'",
		},
		{
			name:    "template contains backslash",
			data:    "version: 1\ngenerators:\n  g:\n    doc: d\n    needs: [l/v1]\n    units: u\n    out: o\n    template: \"a\\\\b.tmpl\"\n",
			wantErr: "genx.meta.yaml:8: generators.g.template: must not contain '/' or '\\'",
		},
		{
			name:    "template is dot",
			data:    "version: 1\ngenerators:\n  g:\n    doc: d\n    needs: [l/v1]\n    units: u\n    out: o\n    template: \".\"\n",
			wantErr: "genx.meta.yaml:8: generators.g.template: must not be '.' or '..'",
		},
		{
			name:    "template starts with dot",
			data:    "version: 1\ngenerators:\n  g:\n    doc: d\n    needs: [l/v1]\n    units: u\n    out: o\n    template: \".hidden.tmpl\"\n",
			wantErr: "genx.meta.yaml:8: generators.g.template: must not start with '.'",
		},
		{
			name:    "emits invalid domain",
			data:    "version: 1\ngenerators:\n  g:\n    doc: d\n    needs: [l/v1]\n    units: u\n    out: o\n    emits: bad_domain\n",
			wantErr: `genx.meta.yaml:8: generators.g.emits: "bad_domain": invalid domain (want name/vN, such as labels/v1)`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse("genx.meta.yaml", []byte(tc.data))
			if err == nil {
				t.Fatalf("expected error %q, got nil", tc.wantErr)
			}
			if err.Error() != tc.wantErr {
				t.Fatalf("error = %q, want %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestMultipleMetaErrorsJoined(t *testing.T) {
	metaYAML := `version: 1
generators:
  g1:
    doc: ""
    needs: []
    units: ""
    out: ""
`
	_, err := Parse("genx.meta.yaml", []byte(metaYAML))
	if err == nil {
		t.Fatal("expected errors, got nil")
	}
	lines := strings.Split(err.Error(), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 errors, got %d: %v", len(lines), lines)
	}
	expected := []string{
		"genx.meta.yaml:3: generators.g1: exactly one of template and emits",
		"genx.meta.yaml:4: generators.g1.doc: must not be empty",
		"genx.meta.yaml:5: generators.g1.needs: must not be empty",
		"genx.meta.yaml:6: generators.g1.units: must not be empty",
		"genx.meta.yaml:7: generators.g1.out: must not be empty",
	}
	for i, exp := range expected {
		if lines[i] != exp {
			t.Errorf("line %d:\ngot:  %q\nwant: %q", i, lines[i], exp)
		}
	}
}
