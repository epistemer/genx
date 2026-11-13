package run

import (
	"context"
	"testing"

	"github.com/epistemer/genx/internal/domain"
)

func TestEmitLabelsValidation(t *testing.T) {
	s := &state{
		ctx:  context.Background(),
		docs: map[string][]document{},
	}
	d, _ := domain.Parse("labels/v1")
	g := generator{
		step:  "example.com/app/gen:enum-labels",
		emits: d,
	}

	invalidUnit := map[string]any{
		"domain":   "labels/v1",
		"producer": "example",
		"package": map[string]any{
			"name": "settings",
			"dir":  "internal/settings",
		},
		"labels": []any{
			map[string]any{
				"key":  "port",
				"text": "Port",
				"var":  "func",
			},
		},
	}

	_, err := s.emit(g, "models/zz_enums.labels.json", invalidUnit)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	want := `example.com/app/gen:enum-labels: models/zz_enums.labels.json: labels[0].var: "func" is a Go keyword`
	if err.Error() != want {
		t.Fatalf("got:  %q\nwant: %q", err.Error(), want)
	}

	validUnit := map[string]any{
		"domain":   "labels/v1",
		"producer": "example",
		"package": map[string]any{
			"name": "settings",
			"dir":  "internal/settings",
		},
		"labels": []any{
			map[string]any{
				"key":  "port",
				"text": "Port",
				"var":  "Port",
			},
		},
	}

	data, err := s.emit(g, "models/zz_enums.labels.json", validUnit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected data, got empty")
	}
	if len(s.docs["labels/v1"]) != 1 {
		t.Fatalf("expected 1 doc recorded, got %d", len(s.docs["labels/v1"]))
	}
}
