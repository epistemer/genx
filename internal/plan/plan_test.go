package plan_test

import (
	"reflect"
	"testing"

	"github.com/epistemer/genx/internal/domain"
	"github.com/epistemer/genx/internal/plan"
)

func d(name string, major int) domain.Domain {
	return domain.Domain{Name: name, Major: major}
}

func TestNew_DuplicateNames(t *testing.T) {
	tests := []struct {
		name    string
		steps   []plan.Step
		wantErr string
	}{
		{
			name: "duplicate step names",
			steps: []plan.Step{
				{Name: "settings"},
				{Name: "models"},
				{Name: "settings"},
			},
			wantErr: "two steps are named settings",
		},
		{
			name: "unique step names",
			steps: []plan.Step{
				{Name: "settings"},
				{Name: "models"},
			},
			wantErr: "",
		},
		{
			name: "unknown steps with missing domain allowed in New",
			steps: []plan.Step{
				{Name: "gen", Needs: []domain.Domain{d("labels", 1)}, Known: true},
				{Name: "prod", Known: false},
			},
			wantErr: "",
		},
		{
			name: "known steps with missing domain allowed in New",
			steps: []plan.Step{
				{Name: "gen", Needs: []domain.Domain{d("labels", 1)}, Known: true},
			},
			wantErr: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := plan.New(tc.steps)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tc.wantErr)
				}
				if err.Error() != tc.wantErr {
					t.Fatalf("error = %q, want %q", err.Error(), tc.wantErr)
				}
			}
		})
	}
}

func TestOrder_Table(t *testing.T) {
	tests := []struct {
		name      string
		steps     []plan.Step
		wantOrder []int
		wantErr   string
	}{
		{
			name:      "empty steps",
			steps:     []plan.Step{},
			wantOrder: []int{},
		},
		{
			name: "linear chain",
			steps: []plan.Step{
				{Name: "s0", Writes: []domain.Domain{d("a", 1)}, Known: true},
				{Name: "s1", Needs: []domain.Domain{d("a", 1)}, Writes: []domain.Domain{d("b", 1)}, Known: true},
				{Name: "s2", Needs: []domain.Domain{d("b", 1)}, Known: true},
			},
			wantOrder: []int{0, 1, 2},
		},
		{
			name: "manifest-order tie-breaking",
			steps: []plan.Step{
				{Name: "s0", Known: true},
				{Name: "s1", Known: true},
				{Name: "s2", Known: true},
			},
			wantOrder: []int{0, 1, 2},
		},
		{
			name: "generator declared before producer it needs",
			steps: []plan.Step{
				{Name: "gen", Needs: []domain.Domain{d("labels", 1)}, Known: true},
				{Name: "prod", Writes: []domain.Domain{d("labels", 1)}, Known: true},
			},
			wantOrder: []int{1, 0},
		},
		{
			name: "several writers of one domain",
			steps: []plan.Step{
				{Name: "prod1", Writes: []domain.Domain{d("labels", 1)}, Known: true},
				{Name: "gen", Needs: []domain.Domain{d("labels", 1)}, Known: true},
				{Name: "prod2", Writes: []domain.Domain{d("labels", 1)}, Known: true},
			},
			wantOrder: []int{0, 2, 1},
		},
		{
			name: "pilot order with reverse generator declaration",
			steps: []plan.Step{
				{Name: "settings", Writes: []domain.Domain{d("labels", 1)}, Known: true},
				{Name: "models", Writes: []domain.Domain{d("sqlc.request", 1)}, Known: true},
				{Name: "i18n:labels", Needs: []domain.Domain{d("labels", 1)}, Known: true},
				{Name: "models:enum-labels", Needs: []domain.Domain{d("sqlc.request", 1)}, Writes: []domain.Domain{d("labels", 1)}, Known: true},
			},
			wantOrder: []int{0, 1, 3, 2},
		},
		{
			name: "cycle two steps",
			steps: []plan.Step{
				{Name: "stepA", Needs: []domain.Domain{d("y", 1)}, Writes: []domain.Domain{d("x", 1)}, Known: true},
				{Name: "stepB", Needs: []domain.Domain{d("x", 1)}, Writes: []domain.Domain{d("y", 1)}, Known: true},
			},
			wantErr: "these steps need each other's domains: stepA (needs y/v1, written by stepB); stepB (needs x/v1, written by stepA)",
		},
		{
			name: "cycle three steps",
			steps: []plan.Step{
				{Name: "s0", Needs: []domain.Domain{d("d1", 1)}, Writes: []domain.Domain{d("d0", 1)}, Known: true},
				{Name: "s1", Needs: []domain.Domain{d("d2", 1)}, Writes: []domain.Domain{d("d1", 1)}, Known: true},
				{Name: "s2", Needs: []domain.Domain{d("d0", 1)}, Writes: []domain.Domain{d("d2", 1)}, Known: true},
			},
			wantErr: "these steps need each other's domains: s0 (needs d1/v1, written by s1); s1 (needs d2/v1, written by s2); s2 (needs d0/v1, written by s0)",
		},
		{
			name: "cycle self dependency",
			steps: []plan.Step{
				{Name: "self", Needs: []domain.Domain{d("x", 1)}, Writes: []domain.Domain{d("x", 1)}, Known: true},
			},
			wantErr: "these steps need each other's domains: self (needs x/v1, written by self)",
		},
		{
			name: "cycle first blocking need and lowest indexed writer chosen",
			steps: []plan.Step{
				{
					Name:   "s0",
					Needs:  []domain.Domain{d("d1", 1), d("d2", 1)},
					Writes: []domain.Domain{d("d0", 1)},
					Known:  true,
				},
				{
					Name:   "s1",
					Needs:  []domain.Domain{d("d0", 1)},
					Writes: []domain.Domain{d("d1", 1)},
					Known:  true,
				},
				{
					Name:   "s2",
					Needs:  []domain.Domain{d("d0", 1)},
					Writes: []domain.Domain{d("d1", 1)},
					Known:  true,
				},
			},
			wantErr: "these steps need each other's domains: s0 (needs d1/v1, written by s1); s1 (needs d0/v1, written by s0); s2 (needs d0/v1, written by s0)",
		},
		{
			name: "missing domain no unknown steps",
			steps: []plan.Step{
				{Name: "github.com/epistemer/i18n:labels", Needs: []domain.Domain{d("labels", 1)}, Known: true},
			},
			wantErr: "github.com/epistemer/i18n:labels needs labels/v1, which no step writes",
		},
		{
			name: "missing second domain in order",
			steps: []plan.Step{
				{Name: "prod", Writes: []domain.Domain{d("a", 1)}, Known: true},
				{Name: "gen", Needs: []domain.Domain{d("a", 1), d("missing", 1)}, Known: true},
			},
			wantErr: "gen needs missing/v1, which no step writes",
		},
		{
			name: "major mismatch message",
			steps: []plan.Step{
				{Name: "gen", Needs: []domain.Domain{d("labels", 1)}, Known: true},
				{Name: "prod", Writes: []domain.Domain{d("labels", 2)}, Known: true},
			},
			wantErr: "gen needs labels/v1, but prod writes labels/v2",
		},
		{
			name: "major mismatch lowest indexed writer chosen",
			steps: []plan.Step{
				{Name: "gen", Needs: []domain.Domain{d("labels", 1)}, Known: true},
				{Name: "prodB", Writes: []domain.Domain{d("labels", 3)}, Known: true},
				{Name: "prodA", Writes: []domain.Domain{d("labels", 2)}, Known: true},
			},
			wantErr: "gen needs labels/v1, but prodB writes labels/v3",
		},
		{
			name: "exact writer prevents major mismatch error",
			steps: []plan.Step{
				{Name: "prodBad", Writes: []domain.Domain{d("labels", 2)}, Known: true},
				{Name: "prodGood", Writes: []domain.Domain{d("labels", 1)}, Known: true},
				{Name: "gen", Needs: []domain.Domain{d("labels", 1)}, Known: true},
			},
			wantOrder: []int{0, 1, 2},
		},
		{
			name: "unknown step in Order writes nothing",
			steps: []plan.Step{
				{Name: "gen", Needs: []domain.Domain{d("labels", 1)}, Known: true},
				{Name: "prod", Writes: []domain.Domain{d("labels", 1)}, Known: false},
			},
			wantErr: "gen needs labels/v1, which no step writes",
		},
		{
			name: "duplicate step names in Order",
			steps: []plan.Step{
				{Name: "dup", Known: true},
				{Name: "dup", Known: true},
			},
			wantErr: "two steps are named dup",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := plan.Order(tc.steps)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !reflect.DeepEqual(got, tc.wantOrder) {
					t.Fatalf("Order() = %v, want %v", got, tc.wantOrder)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tc.wantErr)
				}
				if err.Error() != tc.wantErr {
					t.Fatalf("error = %q, want %q", err.Error(), tc.wantErr)
				}
			}
		})
	}
}

func TestPlan_StepByStep(t *testing.T) {
	t.Run("linear chain Next and Done", func(t *testing.T) {
		steps := []plan.Step{
			{Name: "s0", Writes: []domain.Domain{d("a", 1)}, Known: true},
			{Name: "s1", Needs: []domain.Domain{d("a", 1)}, Writes: []domain.Domain{d("b", 1)}, Known: true},
			{Name: "s2", Needs: []domain.Domain{d("b", 1)}, Known: true},
		}
		p, err := plan.New(steps)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}

		i, ok, err := p.Next()
		if err != nil || !ok || i != 0 {
			t.Fatalf("Next() 1 = (%d, %v, %v), want (0, true, nil)", i, ok, err)
		}
		if err := p.Done(i, steps[i].Writes); err != nil {
			t.Fatalf("Done(0) failed: %v", err)
		}

		i, ok, err = p.Next()
		if err != nil || !ok || i != 1 {
			t.Fatalf("Next() 2 = (%d, %v, %v), want (1, true, nil)", i, ok, err)
		}
		if err := p.Done(i, steps[i].Writes); err != nil {
			t.Fatalf("Done(1) failed: %v", err)
		}

		i, ok, err = p.Next()
		if err != nil || !ok || i != 2 {
			t.Fatalf("Next() 3 = (%d, %v, %v), want (2, true, nil)", i, ok, err)
		}
		if err := p.Done(i, steps[i].Writes); err != nil {
			t.Fatalf("Done(2) failed: %v", err)
		}

		_, ok, err = p.Next()
		if err != nil || ok {
			t.Fatalf("Next() after finished = (_, %v, %v), want (false, nil)", ok, err)
		}
	})

	t.Run("unknown producer pending allows dependent generator to run then Done catches it", func(t *testing.T) {
		steps := []plan.Step{
			{Name: "github.com/epistemer/i18n:labels", Needs: []domain.Domain{d("labels", 1)}, Known: true},
			{Name: "producer settings", Known: false},
		}
		p, err := plan.New(steps)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}

		// Generator runs because unknown producer is pending
		i, ok, err := p.Next()
		if err != nil || !ok || i != 0 {
			t.Fatalf("Next() 1 = (%d, %v, %v), want (0, true, nil)", i, ok, err)
		}
		if err := p.Done(0, nil); err != nil {
			t.Fatalf("Done(0) failed: %v", err)
		}

		// Now producer runs
		i, ok, err = p.Next()
		if err != nil || !ok || i != 1 {
			t.Fatalf("Next() 2 = (%d, %v, %v), want (1, true, nil)", i, ok, err)
		}

		// Producer wrote labels/v1, which step 0 needed!
		wantErr := "github.com/epistemer/i18n:labels needed labels/v1, but ran before producer settings, which writes it; run genx again: the second run reads the domain from the file"
		err = p.Done(1, []domain.Domain{d("labels", 1)})
		if err == nil {
			t.Fatalf("expected error %q, got nil", wantErr)
		}
		if err.Error() != wantErr {
			t.Fatalf("Done(1) error = %q, want %q", err.Error(), wantErr)
		}
	})

	t.Run("Done late writer chooses lowest indexed finished step", func(t *testing.T) {
		steps := []plan.Step{
			{Name: "gen1", Needs: []domain.Domain{d("labels", 1)}, Known: true},
			{Name: "gen2", Needs: []domain.Domain{d("labels", 1)}, Known: true},
			{Name: "prod", Known: false},
		}
		p, err := plan.New(steps)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}

		// Both gen1 and gen2 run while prod is pending unknown
		i, _, err := p.Next()
		if err != nil || i != 0 {
			t.Fatalf("Next() = %d, want 0", i)
		}
		_ = p.Done(0, nil)

		i, _, err = p.Next()
		if err != nil || i != 1 {
			t.Fatalf("Next() = %d, want 1", i)
		}
		_ = p.Done(1, nil)

		// Now prod runs
		i, _, err = p.Next()
		if err != nil || i != 2 {
			t.Fatalf("Next() = %d, want 2", i)
		}

		// Prod writes labels/v1. Both gen1 and gen2 needed it.
		// Error must name gen1 (first such finished step by index).
		wantErr := "gen1 needed labels/v1, but ran before prod, which writes it; run genx again: the second run reads the domain from the file"
		err = p.Done(2, []domain.Domain{d("labels", 1)})
		if err == nil {
			t.Fatalf("expected error %q, got nil", wantErr)
		}
		if err.Error() != wantErr {
			t.Fatalf("Done(2) error = %q, want %q", err.Error(), wantErr)
		}
	})

	t.Run("unknown producer writes nothing anyone needs is fine", func(t *testing.T) {
		steps := []plan.Step{
			{Name: "github.com/epistemer/i18n:labels", Needs: []domain.Domain{d("labels", 1)}, Known: true},
			{Name: "producer settings", Known: false},
		}
		p, err := plan.New(steps)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}

		i, ok, err := p.Next()
		if err != nil || !ok || i != 0 {
			t.Fatalf("Next() 1 = (%d, %v, %v), want (0, true, nil)", i, ok, err)
		}
		if err := p.Done(0, nil); err != nil {
			t.Fatalf("Done(0) failed: %v", err)
		}

		i, ok, err = p.Next()
		if err != nil || !ok || i != 1 {
			t.Fatalf("Next() 2 = (%d, %v, %v), want (1, true, nil)", i, ok, err)
		}
		// Producer writes nothing or writes an unrelated domain
		if err := p.Done(1, []domain.Domain{d("other", 1)}); err != nil {
			t.Fatalf("Done(1) unexpected error: %v", err)
		}

		_, ok, err = p.Next()
		if err != nil || ok {
			t.Fatalf("Next() end = (_, %v, %v), want (false, nil)", ok, err)
		}
	})

	t.Run("major mismatch with finished writer", func(t *testing.T) {
		steps := []plan.Step{
			{Name: "prod", Writes: []domain.Domain{d("labels", 2)}, Known: true},
			{Name: "gen", Needs: []domain.Domain{d("labels", 1)}, Known: true},
		}
		p, err := plan.New(steps)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}

		i, ok, err := p.Next()
		if err != nil || !ok || i != 0 {
			t.Fatalf("Next() 1 = (%d, %v, %v), want (0, true, nil)", i, ok, err)
		}
		if err := p.Done(0, steps[0].Writes); err != nil {
			t.Fatalf("Done(0) failed: %v", err)
		}

		// When gen is chosen, prod already finished and wrote labels/v2
		wantErr := "gen needs labels/v1, but prod writes labels/v2"
		_, _, err = p.Next()
		if err == nil {
			t.Fatalf("expected error %q, got nil", wantErr)
		}
		if err.Error() != wantErr {
			t.Fatalf("Next() 2 error = %q, want %q", err.Error(), wantErr)
		}
	})

	t.Run("Done on Known step with different writes allowed", func(t *testing.T) {
		steps := []plan.Step{
			{Name: "s0", Writes: []domain.Domain{d("a", 1)}, Known: true},
			{Name: "s1", Needs: []domain.Domain{d("b", 1)}, Known: true},
		}
		p, err := plan.New(steps)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}

		i, ok, err := p.Next()
		if err != nil || !ok || i != 0 {
			t.Fatalf("Next() 1 = (%d, %v, %v), want (0, true, nil)", i, ok, err)
		}
		// Done with b/v1 instead of a/v1
		if err := p.Done(0, []domain.Domain{d("b", 1)}); err != nil {
			t.Fatalf("Done(0) failed: %v", err)
		}

		// Now s1 can run because s0 wrote b/v1!
		i, ok, err = p.Next()
		if err != nil || !ok || i != 1 {
			t.Fatalf("Next() 2 = (%d, %v, %v), want (1, true, nil)", i, ok, err)
		}
		if err := p.Done(1, nil); err != nil {
			t.Fatalf("Done(1) failed: %v", err)
		}
	})

	t.Run("Done invalid index or already finished", func(t *testing.T) {
		steps := []plan.Step{
			{Name: "s0", Known: true},
		}
		p, err := plan.New(steps)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}

		if err := p.Done(-1, nil); err == nil {
			t.Fatal("Done(-1) expected error, got nil")
		}
		if err := p.Done(1, nil); err == nil {
			t.Fatal("Done(1) expected error, got nil")
		}

		if err := p.Done(0, nil); err != nil {
			t.Fatalf("Done(0) unexpected error: %v", err)
		}
		if err := p.Done(0, nil); err == nil {
			t.Fatal("Done(0) already finished expected error, got nil")
		}
	})
}
