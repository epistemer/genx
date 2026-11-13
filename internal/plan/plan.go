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
	Writes []domain.Domain // what the step is known, or guessed, to write
	Known  bool            // false: a producer whose writes are learned when it has run
	// Peeked marks Writes as a guess: a producer's domains read from files
	// a previous run left. A guess orders steps, but no error rests on it.
	Peeked bool
}

// Plan is a scheduler that orders producers and generators by domain dependencies.
type Plan struct {
	steps        []Step
	finished     []bool
	actualWrites [][]domain.Domain
	lenient      bool // Order: report the order, not missing domains
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
		copied[i] = s
		copied[i].Needs = append([]domain.Domain(nil), s.Needs...)
		copied[i].Writes = append([]domain.Domain(nil), s.Writes...)
	}
	return &Plan{
		steps:        copied,
		finished:     make([]bool, len(steps)),
		actualWrites: make([][]domain.Domain, len(steps)),
	}, nil
}

// Next returns the index of the next step to run: the lowest-indexed pending
// step none of whose needed domains has another pending writer that is
// Known, and, if it needs anything, with no pending unknown step that needs
// nothing. ok is false when no step is pending; then Next checks that every
// needed domain was written. An error means the remaining steps can't run
// (a cycle), or a step needs a domain nobody writes or writes in another
// major version.
func (p *Plan) Next() (i int, ok bool, err error) {
	if !p.pending() {
		if p.lenient {
			return 0, false, nil
		}
		return 0, false, p.checkAllWritten()
	}
	for cand := range p.steps {
		if !p.finished[cand] && p.isReady(cand) {
			if err := p.validateChosen(cand); err != nil {
				return 0, false, err
			}
			return cand, true, nil
		}
	}
	return 0, false, p.cycleError()
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
		for j := range p.steps {
			if j != i && p.finished[j] && containsDomain(p.steps[j].Needs, d) {
				return fmt.Errorf("%s needed %s, but ran before %s, which writes it; run genx again: the second run reads the domain from the file",
					p.steps[j].Name, d, p.steps[i].Name)
			}
		}
	}
	p.finished[i] = true
	p.actualWrites[i] = append([]domain.Domain(nil), wrote...)
	return nil
}

// Order returns the order a run would take, for genx list, without running
// anything: each step is taken to write what it is known or guessed to
// write. It reports cycles, not missing domains: a step whose producer
// hasn't run yet may need a domain nothing is known to write.
func Order(steps []Step) ([]int, error) {
	p, err := New(steps)
	if err != nil {
		return nil, err
	}
	p.lenient = true
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
		if err := p.Done(i, p.steps[i].Writes); err != nil {
			return nil, err
		}
	}
}

func (p *Plan) pending() bool {
	for _, f := range p.finished {
		if !f {
			return true
		}
	}
	return false
}

func (p *Plan) isReady(cand int) bool {
	for _, d := range p.steps[cand].Needs {
		if p.pendingKnownWriter(d, cand) >= 0 {
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

// pendingKnownWriter returns the lowest-indexed pending step other than
// except that is known (or guessed) to write d's name, in any major
// version, or -1. Ordering goes by name, so a stale guess of the major
// still orders a producer before its readers; validation compares majors.
func (p *Plan) pendingKnownWriter(d domain.Domain, except int) int {
	for w := range p.steps {
		if w == except || p.finished[w] || !p.steps[w].Known {
			continue
		}
		for _, x := range p.steps[w].Writes {
			if x.Name == d.Name {
				return w
			}
		}
	}
	return -1
}

// cycleError names the steps that wait on each other: those on a cycle of
// "needs a domain that a pending step writes", not the steps merely
// waiting behind one.
func (p *Plan) cycleError() error {
	next := func(cand int) []int {
		var ws []int
		for _, d := range p.steps[cand].Needs {
			if w := p.pendingKnownWriter(d, cand); w >= 0 {
				ws = append(ws, w)
			}
		}
		return ws
	}
	onCycle := func(start int) bool {
		seen := make([]bool, len(p.steps))
		stack := next(start)
		for len(stack) > 0 {
			w := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if w == start {
				return true
			}
			if !seen[w] {
				seen[w] = true
				stack = append(stack, next(w)...)
			}
		}
		return false
	}
	var parts []string
	for cand := range p.steps {
		if p.finished[cand] || !onCycle(cand) {
			continue
		}
		for _, d := range p.steps[cand].Needs {
			if w := p.pendingKnownWriter(d, cand); w >= 0 {
				parts = append(parts, fmt.Sprintf("%s (needs %s, written by %s)", p.steps[cand].Name, d, p.steps[w].Name))
				break
			}
		}
	}
	return fmt.Errorf("these steps need each other's domains: %s", strings.Join(parts, "; "))
}

// validateChosen checks a step's needs when it is about to run. A need is
// settled by a finished step that wrote it, or a pending step known for
// sure to write it. While a pending step's writes are unknown or guessed,
// nothing is reported: checkAllWritten has the last word.
func (p *Plan) validateChosen(chosen int) error {
	if p.lenient {
		return nil
	}
	for _, d := range p.steps[chosen].Needs {
		if p.surelyWritten(d) || p.uncertainPending() {
			continue
		}
		return p.missing(chosen, d)
	}
	return nil
}

// checkAllWritten runs once every step has: each need must have been
// written by some step.
func (p *Plan) checkAllWritten() error {
	for i, s := range p.steps {
		for _, d := range s.Needs {
			if !p.surelyWritten(d) {
				return p.missing(i, d)
			}
		}
	}
	return nil
}

// missing reports a need nothing writes, naming a writer of another major
// version when there is one.
func (p *Plan) missing(i int, d domain.Domain) error {
	name := p.steps[i].Name
	for w := range p.steps {
		writes := p.actualWrites[w]
		if !p.finished[w] {
			if !p.steps[w].Known || p.steps[w].Peeked {
				continue
			}
			writes = p.steps[w].Writes
		}
		for _, x := range writes {
			if x.Name == d.Name && x.Major != d.Major {
				return fmt.Errorf("%s needs %s, but %s writes %s", name, d, p.steps[w].Name, x)
			}
		}
	}
	return fmt.Errorf("%s needs %s, which no step writes", name, d)
}

// surelyWritten reports whether a finished step wrote d, or a pending step
// whose writes aren't a guess will.
func (p *Plan) surelyWritten(d domain.Domain) bool {
	for w := range p.steps {
		if p.finished[w] {
			if containsDomain(p.actualWrites[w], d) {
				return true
			}
		} else if p.steps[w].Known && !p.steps[w].Peeked && containsDomain(p.steps[w].Writes, d) {
			return true
		}
	}
	return false
}

// uncertainPending reports whether a pending step's writes are unknown or
// guessed.
func (p *Plan) uncertainPending() bool {
	for w := range p.steps {
		if !p.finished[w] && (!p.steps[w].Known || p.steps[w].Peeked) {
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
