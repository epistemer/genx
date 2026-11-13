package plan

import (
	"fmt"
	"strings"

	"github.com/epistemer/genx/internal/domain"
)

// Step is one producer or generator, as the plan sees it.
type Step struct {
	Name   string // for messages, e.g. "producer settings" or "github.com/epistemer/i18n:labels"
	Needs  []domain.Domain
	Writes []domain.Domain // what the step is known to write
	Known  bool            // false: a producer whose writes are learned when it has run
}

// Plan is a scheduler that orders producers and generators by domain dependencies.
type Plan struct {
	steps        []Step
	finished     []bool
	actualWrites [][]domain.Domain
}

// New checks what can be checked up front and returns a plan.
func New(steps []Step) (*Plan, error) {
	seen := make(map[string]bool, len(steps))
	copied := make([]Step, len(steps))
	for i, s := range steps {
		if seen[s.Name] {
			return nil, fmt.Errorf("two steps are named %s", s.Name)
		}
		seen[s.Name] = true
		copied[i] = Step{
			Name:   s.Name,
			Needs:  append([]domain.Domain(nil), s.Needs...),
			Writes: append([]domain.Domain(nil), s.Writes...),
			Known:  s.Known,
		}
	}
	return &Plan{
		steps:        copied,
		finished:     make([]bool, len(steps)),
		actualWrites: make([][]domain.Domain, len(steps)),
	}, nil
}

// Next returns the index of the next step to run: the lowest-indexed pending
// step none of whose needed domains has a pending writer that is Known, and,
// if it needs anything, with no pending unknown step that needs nothing.
// ok is false when no step is pending. An error means the remaining steps
// can't run (a cycle) or the chosen step needs a domain nobody writes or
// writes in another major version (checked when the step is chosen; see below).
func (p *Plan) Next() (i int, ok bool, err error) {
	hasPending := false
	for _, f := range p.finished {
		if !f {
			hasPending = true
			break
		}
	}
	if !hasPending {
		return 0, false, nil
	}

	chosen := -1
	for cand := 0; cand < len(p.steps); cand++ {
		if p.finished[cand] {
			continue
		}
		if p.isReady(cand) {
			chosen = cand
			break
		}
	}

	if chosen == -1 {
		return 0, false, p.cycleError()
	}

	if err := p.validateChosen(chosen); err != nil {
		return 0, false, err
	}

	return chosen, true, nil
}

// Done records that step i ran and what it actually wrote (for a Known step,
// what it declared). It fails when an already-finished step needed one of these
// domains: that step ran too early.
func (p *Plan) Done(i int, wrote []domain.Domain) error {
	if i < 0 || i >= len(p.steps) {
		return fmt.Errorf("step index %d out of range", i)
	}
	if p.finished[i] {
		return fmt.Errorf("step %d is already finished", i)
	}

	for _, d := range wrote {
		for j := 0; j < len(p.steps); j++ {
			if j == i || !p.finished[j] {
				continue
			}
			if containsDomain(p.steps[j].Needs, d) {
				return fmt.Errorf("%s needed %s, but ran before %s, which writes it; run genx again: the second run reads the domain from the file",
					p.steps[j].Name, d, p.steps[i].Name)
			}
		}
	}

	p.finished[i] = true
	p.actualWrites[i] = append([]domain.Domain(nil), wrote...)
	return nil
}

// Order runs the scheduler with every step treated as Known (an unknown step
// writes nothing) and returns the full order, for `genx list`.
func Order(steps []Step) ([]int, error) {
	simSteps := make([]Step, len(steps))
	for i, s := range steps {
		simSteps[i] = Step{
			Name:   s.Name,
			Needs:  append([]domain.Domain(nil), s.Needs...),
			Writes: append([]domain.Domain(nil), s.Writes...),
			Known:  true,
		}
		if !s.Known {
			simSteps[i].Writes = nil
		}
	}

	p, err := New(simSteps)
	if err != nil {
		return nil, err
	}

	order := make([]int, 0, len(steps))
	for {
		i, ok, err := p.Next()
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		order = append(order, i)
		if err := p.Done(i, simSteps[i].Writes); err != nil {
			return nil, err
		}
	}
	return order, nil
}

func (p *Plan) isReady(cand int) bool {
	for _, d := range p.steps[cand].Needs {
		if p.hasPendingKnownWriter(d) {
			return false
		}
	}
	// A step that needs anything also waits for every pending step whose
	// writes are unknown and that needs nothing itself: such a step can
	// always run, and it may write what this one needs. (One that has
	// needs of its own doesn't hold others back, so it can't cause a
	// deadlock; Done catches it if it turns out to write a needed domain.)
	if len(p.steps[cand].Needs) > 0 {
		for w := range p.steps {
			if w != cand && !p.finished[w] && !p.steps[w].Known && len(p.steps[w].Needs) == 0 {
				return false
			}
		}
	}
	return true
}

func (p *Plan) hasPendingKnownWriter(d domain.Domain) bool {
	for w := 0; w < len(p.steps); w++ {
		if !p.finished[w] && p.steps[w].Known && containsDomain(p.steps[w].Writes, d) {
			return true
		}
	}
	return false
}

func (p *Plan) cycleError() error {
	var parts []string
	for cand := 0; cand < len(p.steps); cand++ {
		if p.finished[cand] {
			continue
		}
		for _, d := range p.steps[cand].Needs {
			writerIdx := -1
			for w := 0; w < len(p.steps); w++ {
				if !p.finished[w] && p.steps[w].Known && containsDomain(p.steps[w].Writes, d) {
					writerIdx = w
					break
				}
			}
			if writerIdx != -1 {
				parts = append(parts, fmt.Sprintf("%s (needs %s, written by %s)",
					p.steps[cand].Name, d, p.steps[writerIdx].Name))
				break
			}
		}
	}
	return fmt.Errorf("these steps need each other's domains: %s", strings.Join(parts, "; "))
}

func (p *Plan) validateChosen(chosen int) error {
	step := p.steps[chosen]
	for _, d := range step.Needs {
		hasExact := p.hasExactWriter(d)
		if !hasExact {
			if mismatchWriter, mismatchDomain, ok := p.findMajorMismatchWriter(d); ok {
				return fmt.Errorf("%s needs %s, but %s writes %s",
					step.Name, d, mismatchWriter.Name, mismatchDomain)
			}
			if !p.hasPendingUnknown() {
				return fmt.Errorf("%s needs %s, which no step writes", step.Name, d)
			}
		}
	}
	return nil
}

func (p *Plan) hasExactWriter(d domain.Domain) bool {
	for i := 0; i < len(p.steps); i++ {
		if p.finished[i] {
			if containsDomain(p.actualWrites[i], d) {
				return true
			}
		} else if p.steps[i].Known {
			if containsDomain(p.steps[i].Writes, d) {
				return true
			}
		}
	}
	return false
}

func (p *Plan) findMajorMismatchWriter(d domain.Domain) (Step, domain.Domain, bool) {
	for i := 0; i < len(p.steps); i++ {
		var writes []domain.Domain
		if p.finished[i] {
			writes = p.actualWrites[i]
		} else if p.steps[i].Known {
			writes = p.steps[i].Writes
		}
		for _, w := range writes {
			if w.Name == d.Name && w.Major != d.Major {
				return p.steps[i], w, true
			}
		}
	}
	return Step{}, domain.Domain{}, false
}

func (p *Plan) hasPendingUnknown() bool {
	for i := 0; i < len(p.steps); i++ {
		if !p.finished[i] && !p.steps[i].Known {
			return true
		}
	}
	return false
}

func containsDomain(list []domain.Domain, d domain.Domain) bool {
	for _, x := range list {
		if x == d {
			return true
		}
	}
	return false
}
