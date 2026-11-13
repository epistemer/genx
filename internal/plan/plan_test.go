package plan_test

import (
	"errors"
	"fmt"
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
			wantOrder: nil,
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
			// A step may read a domain it also writes: it reads what the
			// other writers wrote. Alone, it reads nothing.
			wantOrder: []int{0},
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
			wantErr: "these steps need each other's domains: s0 (needs d1/v1, written by s1); s1 (needs d0/v1, written by s0)",
		},
		{
			name: "missing domain no unknown steps",
			steps: []plan.Step{
				{Name: "github.com/epistemer/i18n:labels", Needs: []domain.Domain{d("labels", 1)}, Known: true},
			},
			wantErr: "github.com/epistemer/i18n:labels needs labels/v1, which no step writes; add what writes it to genx.yaml, or remove what needs it",
		},
		{
			name: "missing second domain in order",
			steps: []plan.Step{
				{Name: "prod", Writes: []domain.Domain{d("a", 1)}, Known: true},
				{Name: "gen", Needs: []domain.Domain{d("a", 1), d("missing", 1)}, Known: true},
			},
			wantErr: "gen needs missing/v1, which no step writes; add what writes it to genx.yaml, or remove what needs it",
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
			wantErr: "gen needs labels/v1, which no step writes; add what writes it to genx.yaml, or remove what needs it",
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
			got, err := strictOrder(tc.steps)
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

	// Only an unknown producer that has needs of its own can turn out late:
	// one without needs runs before any step that needs something.
	t.Run("unknown producer with needs is caught by Done when late", func(t *testing.T) {
		steps := []plan.Step{
			{Name: "github.com/epistemer/i18n:labels", Needs: []domain.Domain{d("labels", 1)}, Known: true},
			{Name: "producer consumer", Needs: []domain.Domain{d("other", 1)}},
			{Name: "producer base", Writes: []domain.Domain{d("other", 1)}, Known: true},
		}
		p, err := plan.New(steps)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		var order []int
		var doneErr error
		for doneErr == nil {
			i, ok, err := p.Next()
			if err != nil {
				t.Fatalf("Next() failed: %v", err)
			}
			if !ok {
				break
			}
			order = append(order, i)
			var wrote []domain.Domain
			switch i {
			case 1:
				wrote = []domain.Domain{d("labels", 1)}
			case 2:
				wrote = []domain.Domain{d("other", 1)}
			}
			doneErr = p.Done(i, wrote)
		}
		if !reflect.DeepEqual(order, []int{0, 2, 1}) {
			t.Fatalf("order = %v, want [0 2 1]", order)
		}
		want := "github.com/epistemer/i18n:labels needed labels/v1, but ran before producer consumer, which writes it; run genx again: the second run reads the domain from the file"
		if doneErr == nil || doneErr.Error() != want {
			t.Fatalf("Done error = %v, want %q", doneErr, want)
		}
	})

	t.Run("Done late writer chooses lowest indexed finished step", func(t *testing.T) {
		steps := []plan.Step{
			{Name: "gen1", Needs: []domain.Domain{d("labels", 1)}, Known: true},
			{Name: "gen2", Needs: []domain.Domain{d("labels", 1)}, Known: true},
			{Name: "prod", Needs: []domain.Domain{d("other", 1)}},
			{Name: "base", Writes: []domain.Domain{d("other", 1)}, Known: true},
		}
		p, err := plan.New(steps)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		for _, want := range []int{0, 1, 3} {
			i, _, err := p.Next()
			if err != nil || i != want {
				t.Fatalf("Next() = %d, %v; want %d", i, err, want)
			}
			var wrote []domain.Domain
			if i == 3 {
				wrote = []domain.Domain{d("other", 1)}
			}
			if err := p.Done(i, wrote); err != nil {
				t.Fatal(err)
			}
		}
		i, _, err := p.Next()
		if err != nil || i != 2 {
			t.Fatalf("Next() = %d, %v; want 2", i, err)
		}
		wantErr := "gen1 needed labels/v1, but ran before prod, which writes it; run genx again: the second run reads the domain from the file"
		err = p.Done(2, []domain.Domain{d("labels", 1)})
		if err == nil || err.Error() != wantErr {
			t.Fatalf("Done(2) error = %v, want %q", err, wantErr)
		}
	})

	t.Run("unknown producer without needs runs first; if it writes nothing needed, the need is missing", func(t *testing.T) {
		steps := []plan.Step{
			{Name: "github.com/epistemer/i18n:labels", Needs: []domain.Domain{d("labels", 1)}, Known: true},
			{Name: "producer settings", Known: false},
		}
		p, err := plan.New(steps)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		i, ok, err := p.Next()
		if err != nil || !ok || i != 1 {
			t.Fatalf("Next() 1 = (%d, %v, %v), want (1, true, nil)", i, ok, err)
		}
		if err := p.Done(1, []domain.Domain{d("other", 1)}); err != nil {
			t.Fatalf("Done(1) unexpected error: %v", err)
		}
		_, _, err = p.Next()
		want := "github.com/epistemer/i18n:labels needs labels/v1, which no step writes; add what writes it to genx.yaml, or remove what needs it"
		if err == nil || err.Error() != want {
			t.Fatalf("Next() 2 error = %v, want %q", err, want)
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

// On a first run no producer's files exist yet. A producer that needs a
// domain must still wait for producers that need nothing.
func TestUnknownProducersWithoutNeedsGoFirst(t *testing.T) {
	labels := domain.Domain{Name: "labels", Major: 1}
	steps := []plan.Step{
		{Name: "producer consumer", Needs: []domain.Domain{labels}},
		{Name: "producer settings"},
		{Name: "gen:labels", Needs: []domain.Domain{labels}, Known: true},
	}
	p, err := plan.New(steps)
	if err != nil {
		t.Fatal(err)
	}
	var order []int
	for {
		i, ok, err := p.Next()
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			break
		}
		order = append(order, i)
		var wrote []domain.Domain
		if i == 1 {
			wrote = []domain.Domain{labels}
		}
		if err := p.Done(i, wrote); err != nil {
			t.Fatal(err)
		}
	}
	if fmt.Sprint(order) != "[1 0 2]" {
		t.Errorf("order = %v, want [1 0 2]", order)
	}
}

// strictOrder runs the scheduler as genx run does, each step writing what
// it declared (an unknown step, nothing).
func strictOrder(steps []plan.Step) ([]int, error) {
	p, err := plan.New(steps)
	if err != nil {
		return nil, err
	}
	var order []int
	for {
		i, ok, err := p.Next()
		if err != nil {
			return nil, err
		}
		if !ok {
			return order, nil
		}
		order = append(order, i)
		var wrote []domain.Domain
		if steps[i].Known {
			wrote = steps[i].Writes
		}
		if err := p.Done(i, wrote); err != nil {
			return nil, err
		}
	}
}

// Order, for genx list, reports the order without failing on domains that
// unknown or not-yet-run producers will write.
func TestOrderIsLenient(t *testing.T) {
	steps := []plan.Step{
		{Name: "gen", Needs: []domain.Domain{d("labels", 1)}, Known: true},
		{Name: "producer settings"},
	}
	got, err := plan.Order(steps)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []int{1, 0}) {
		t.Fatalf("Order() = %v, want [1 0]", got)
	}
}

// A guessed write orders steps, but a stale guess raises no error: the
// producer runs and writes the new major.
func TestPeekedWritesAreGuesses(t *testing.T) {
	steps := []plan.Step{
		{Name: "gen", Needs: []domain.Domain{d("labels", 2)}, Known: true},
		{Name: "producer settings", Writes: []domain.Domain{d("labels", 1)}, Known: true, Peeked: true},
	}
	p, err := plan.New(steps)
	if err != nil {
		t.Fatal(err)
	}
	i, _, err := p.Next()
	if err != nil || i != 1 {
		t.Fatalf("Next() = %d, %v; want 1, nil", i, err)
	}
	if err := p.Done(1, []domain.Domain{d("labels", 2)}); err != nil {
		t.Fatal(err)
	}
	i, _, err = p.Next()
	if err != nil || i != 0 {
		t.Fatalf("Next() = %d, %v; want 0, nil", i, err)
	}
}

// A need that nothing turns out to write fails at the end, even when an
// unknown producer let the step run.
func TestNeedNeverWrittenFailsAtTheEnd(t *testing.T) {
	steps := []plan.Step{
		{Name: "producer other", Needs: []domain.Domain{d("x", 1)}},
		{Name: "gen", Needs: []domain.Domain{d("labels", 1)}, Known: true},
		{Name: "producer x", Writes: []domain.Domain{d("x", 1)}, Known: true},
	}
	_, err := strictOrder(steps)
	want := "gen needs labels/v1, which no step writes; add what writes it to genx.yaml, or remove what needs it"
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %q", err, want)
	}
}

func TestMismatchErrorType(t *testing.T) {
	steps := []plan.Step{
		{Name: "prod", Writes: []domain.Domain{d("labels", 2)}, Known: true},
		{Name: "gen", Needs: []domain.Domain{d("labels", 1)}, Known: true},
	}
	p, err := plan.New(steps)
	if err != nil {
		t.Fatal(err)
	}
	i, ok, err := p.Next()
	if err != nil || !ok || i != 0 {
		t.Fatalf("Next = %d, %v, %v", i, ok, err)
	}
	if err := p.Done(i, steps[i].Writes); err != nil {
		t.Fatal(err)
	}
	_, _, err = p.Next()
	var mm *plan.MismatchError
	if !errors.As(err, &mm) {
		t.Fatalf("err is not *plan.MismatchError: %v", err)
	}
	if mm.Step != "gen" || mm.Writer != "prod" || mm.Need != d("labels", 1) || mm.Wrote != d("labels", 2) {
		t.Fatalf("unexpected MismatchError: %+v", mm)
	}
	want := "gen needs labels/v1, but prod writes labels/v2"
	if mm.Error() != want {
		t.Fatalf("mm.Error() = %q, want %q", mm.Error(), want)
	}
}
