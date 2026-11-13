package manifest

import (
	"strings"
	"testing"
	"time"

	"github.com/epistemer/genx/internal/domain"
)

func TestParseValid(t *testing.T) {
	manifestYAML := `version: 1

producers:
  - name: settings
    dir: internal/settings
    run:   [go, tool, cfgx, gen, ./schema]
    check: [go, tool, cfgx, gen, -check, ./schema]
    emits:
      - internal/settings/zz_settings.labels.json

  - name: models
    run: [go, tool, sqlc, generate, -f, config/sqlc.yaml]
    check: [go, tool, sqlc, diff, -f, config/sqlc.yaml]
    timeout: 5m
    needs:
      - labels/v1
    emits:
      - path: models/codegen_request.json
        domain: sqlc.request/v1

use:
  - github.com/epistemer/i18n: [labels]
  - example.com/app/internal/gen: [enum-labels]
`
	m, err := Parse("genx.yaml", []byte(manifestYAML))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.Version != 1 {
		t.Errorf("version = %d, want 1", m.Version)
	}
	if len(m.Producers) != 2 {
		t.Fatalf("len(Producers) = %d, want 2", len(m.Producers))
	}

	p0 := m.Producers[0]
	if p0.Name != "settings" || p0.Dir != "internal/settings" {
		t.Errorf("p0 name/dir = %q, %q", p0.Name, p0.Dir)
	}
	if len(p0.Run) != 5 || p0.Run[0] != "go" {
		t.Errorf("p0 run = %v", p0.Run)
	}
	if len(p0.Check) != 6 {
		t.Errorf("p0 check = %v", p0.Check)
	}
	if p0.Timeout != 10*time.Minute {
		t.Errorf("p0 default timeout = %v, want 10m", p0.Timeout)
	}
	if len(p0.Emits) != 1 || p0.Emits[0].Path != "internal/settings/zz_settings.labels.json" || p0.Emits[0].Foreign {
		t.Errorf("p0 emits = %+v", p0.Emits)
	}

	p1 := m.Producers[1]
	if p1.Name != "models" || p1.Dir != "." {
		t.Errorf("p1 name/dir = %q, %q", p1.Name, p1.Dir)
	}
	if p1.Timeout != 5*time.Minute {
		t.Errorf("p1 timeout = %v, want 5m", p1.Timeout)
	}
	if len(p1.Needs) != 1 || p1.Needs[0].String() != "labels/v1" {
		t.Errorf("p1 needs = %v", p1.Needs)
	}
	if len(p1.Emits) != 1 {
		t.Fatalf("p1 len(emits) = %d", len(p1.Emits))
	}
	e1 := p1.Emits[0]
	if e1.Path != "models/codegen_request.json" || !e1.Foreign || e1.Domain != (domain.Domain{Name: "sqlc.request", Major: 1}) {
		t.Errorf("p1 emits[0] = %+v", e1)
	}

	if len(m.Use) != 2 {
		t.Fatalf("len(Use) = %d, want 2", len(m.Use))
	}
	if m.Use[0].Package != "github.com/epistemer/i18n" || len(m.Use[0].Generators) != 1 || m.Use[0].Generators[0] != "labels" {
		t.Errorf("m.Use[0] = %+v", m.Use[0])
	}
	if m.Use[1].Package != "example.com/app/internal/gen" || len(m.Use[1].Generators) != 1 || m.Use[1].Generators[0] != "enum-labels" {
		t.Errorf("m.Use[1] = %+v", m.Use[1])
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
			data:    "producers: []\n",
			wantErr: "genx.yaml:1: version: required",
		},
		{
			name:    "version not integer",
			data:    "version: \"1\"\n",
			wantErr: "genx.yaml:1: version: must be an integer",
		},
		{
			name:    "version 2 newer than supported",
			data:    "version: 2\n",
			wantErr: "genx.yaml:1: version: version 2 is newer than this genx supports; upgrade genx",
		},
		{
			name:    "version 0 must be 1",
			data:    "version: 0\n",
			wantErr: "genx.yaml:1: version: must be 1",
		},
		{
			name:    "unknown top-level field",
			data:    "version: 1\nextra: true\n",
			wantErr: "genx.yaml:2: extra: unknown field",
		},
		{
			name:    "producers not sequence",
			data:    "version: 1\nproducers: invalid\n",
			wantErr: "genx.yaml:2: producers: must be a sequence",
		},
		{
			name:    "producer unknown field",
			data:    "version: 1\nproducers:\n  - name: p1\n    run: [go]\n    emits: [zz_a.json]\n    bogus: true\n",
			wantErr: "genx.yaml:6: producers[0].bogus: unknown field",
		},
		{
			name:    "producer missing name",
			data:    "version: 1\nproducers:\n  - run: [go]\n    emits: [zz_a.json]\n",
			wantErr: "genx.yaml:3: producers[0].name: required",
		},
		{
			name:    "producer invalid name pattern",
			data:    "version: 1\nproducers:\n  - name: Bad_Name\n    run: [go]\n    emits: [zz_a.json]\n",
			wantErr: `genx.yaml:3: producers[0].name: "Bad_Name" is not a valid producer name`,
		},
		{
			name:    "duplicate producer name",
			data:    "version: 1\nproducers:\n  - name: p1\n    run: [go]\n    emits: [zz_a.json]\n  - name: p1\n    run: [go]\n    emits: [zz_b.json]\n",
			wantErr: `genx.yaml:6: producers[1].name: duplicate producer name "p1"`,
		},
		{
			name:    "producer invalid dir",
			data:    "version: 1\nproducers:\n  - name: p1\n    dir: a/../b\n    run: [go]\n    emits: [zz_a.json]\n",
			wantErr: `genx.yaml:4: producers[0].dir: "a/../b": not clean (use "b")`,
		},
		{
			name:    "producer missing run",
			data:    "version: 1\nproducers:\n  - name: p1\n    emits: [zz_a.json]\n",
			wantErr: "genx.yaml:3: producers[0].run: required",
		},
		{
			name:    "producer run empty",
			data:    "version: 1\nproducers:\n  - name: p1\n    run: []\n    emits: [zz_a.json]\n",
			wantErr: "genx.yaml:4: producers[0].run: must not be empty",
		},
		{
			name:    "producer run empty string element",
			data:    "version: 1\nproducers:\n  - name: p1\n    run: [go, \"\"]\n    emits: [zz_a.json]\n",
			wantErr: "genx.yaml:4: producers[0].run[1]: must not be empty",
		},
		{
			name:    "producer check empty",
			data:    "version: 1\nproducers:\n  - name: p1\n    run: [go]\n    check: []\n    emits: [zz_a.json]\n",
			wantErr: "genx.yaml:5: producers[0].check: must not be empty",
		},
		{
			name:    "producer invalid timeout",
			data:    "version: 1\nproducers:\n  - name: p1\n    run: [go]\n    timeout: invalid\n    emits: [zz_a.json]\n",
			wantErr: `genx.yaml:5: producers[0].timeout: invalid duration "invalid"`,
		},
		{
			name:    "producer timeout non-positive",
			data:    "version: 1\nproducers:\n  - name: p1\n    run: [go]\n    timeout: 0s\n    emits: [zz_a.json]\n",
			wantErr: "genx.yaml:5: producers[0].timeout: must be positive",
		},
		{
			name:    "producer invalid needs domain",
			data:    "version: 1\nproducers:\n  - name: p1\n    run: [go]\n    needs: [bad_domain]\n    emits: [zz_a.json]\n",
			wantErr: `genx.yaml:5: producers[0].needs[0]: "bad_domain": invalid domain`,
		},
		{
			name:    "producer missing emits",
			data:    "version: 1\nproducers:\n  - name: p1\n    run: [go]\n",
			wantErr: "genx.yaml:3: producers[0].emits: required",
		},
		{
			name:    "producer emits empty",
			data:    "version: 1\nproducers:\n  - name: p1\n    run: [go]\n    emits: []\n",
			wantErr: "genx.yaml:5: producers[0].emits: must not be empty",
		},
		{
			name:    "producer emit not clean path",
			data:    "version: 1\nproducers:\n  - name: p1\n    run: [go]\n    emits: [a/../b]\n",
			wantErr: `genx.yaml:5: producers[0].emits[0]: "a/../b": not clean (use "b")`,
		},
		{
			name:    "producer foreign emit missing path",
			data:    "version: 1\nproducers:\n  - name: p1\n    run: [go]\n    emits:\n      - domain: labels/v1\n",
			wantErr: "genx.yaml:6: producers[0].emits[0].path: required",
		},
		{
			name:    "producer foreign emit missing domain",
			data:    "version: 1\nproducers:\n  - name: p1\n    run: [go]\n    emits:\n      - path: models/req.json\n",
			wantErr: "genx.yaml:6: producers[0].emits[0].domain: required",
		},
		{
			name:    "producer foreign emit invalid domain",
			data:    "version: 1\nproducers:\n  - name: p1\n    run: [go]\n    emits:\n      - path: models/req.json\n        domain: invalid\n",
			wantErr: `genx.yaml:7: producers[0].emits[0].domain: "invalid": invalid domain`,
		},
		{
			name:    "duplicate emit path across producers",
			data:    "version: 1\nproducers:\n  - name: p1\n    run: [go]\n    emits: [zz_a.json]\n  - name: p2\n    run: [go]\n    emits: [ZZ_A.JSON]\n",
			wantErr: `genx.yaml:8: producers[1].emits[0]: duplicate emit path "ZZ_A.JSON"`,
		},
		{
			name:    "use not sequence",
			data:    "version: 1\nuse: invalid\n",
			wantErr: "genx.yaml:2: use: must be a sequence",
		},
		{
			name:    "use item not mapping",
			data:    "version: 1\nuse:\n  - not_a_map\n",
			wantErr: "genx.yaml:3: use[0]: must be a one-key mapping",
		},
		{
			name:    "use item multiple keys",
			data:    "version: 1\nuse:\n  - a: [x]\n    b: [y]\n",
			wantErr: "genx.yaml:3: use[0]: must be a one-key mapping",
		},
		{
			name:    "use empty package path",
			data:    "version: 1\nuse:\n  - \"\": [labels]\n",
			wantErr: "genx.yaml:3: use[0]: package path must not be empty",
		},
		{
			name:    "use package path with spaces",
			data:    "version: 1\nuse:\n  - \"pkg with space\": [labels]\n",
			wantErr: "genx.yaml:3: use[0]: package path must not contain spaces",
		},
		{
			name:    "use generators empty list",
			data:    "version: 1\nuse:\n  - example.com/pkg: []\n",
			wantErr: "genx.yaml:3: use[0].example.com/pkg: must not be empty",
		},
		{
			name:    "use invalid generator name",
			data:    "version: 1\nuse:\n  - example.com/pkg: [Bad_Gen]\n",
			wantErr: `genx.yaml:3: use[0].example.com/pkg: "Bad_Gen" is not a valid generator name`,
		},
		{
			name:    "use duplicate package+generator",
			data:    "version: 1\nuse:\n  - example.com/pkg: [gen1, gen2]\n  - example.com/pkg: [gen1]\n",
			wantErr: `genx.yaml:4: use[1].example.com/pkg: duplicate generator "gen1" for package "example.com/pkg"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse("genx.yaml", []byte(tc.data))
			if err == nil {
				t.Fatalf("expected error %q, got nil", tc.wantErr)
			}
			if err.Error() != tc.wantErr {
				t.Fatalf("error = %q, want %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestMultipleManifestErrorsJoined(t *testing.T) {
	manifestYAML := `version: 1
bogus_key: 123
producers:
  - name: BadName
    run: []
    emits:
      - bad/../path
`
	_, err := Parse("genx.yaml", []byte(manifestYAML))
	if err == nil {
		t.Fatal("expected errors, got nil")
	}
	lines := strings.Split(err.Error(), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 errors, got %d: %v", len(lines), lines)
	}
	expected := []string{
		"genx.yaml:2: bogus_key: unknown field",
		`genx.yaml:4: producers[0].name: "BadName" is not a valid producer name`,
		"genx.yaml:5: producers[0].run: must not be empty",
		`genx.yaml:7: producers[0].emits[0]: "bad/../path": not clean (use "path")`,
	}
	for i, exp := range expected {
		if lines[i] != exp {
			t.Errorf("line %d:\ngot:  %q\nwant: %q", i, lines[i], exp)
		}
	}
}
