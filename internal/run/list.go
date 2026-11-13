package run

import (
	"context"
	"fmt"
	"strings"

	"github.com/epistemer/genx/internal/plan"
)

// List writes the steps in the order a run would take them, with what each
// reads and writes (interface.md §9). It runs no producer and no query.
func List(ctx context.Context, opts Options) error {
	s, m, gens, err := load(ctx, opts)
	if err != nil {
		return err
	}
	defer s.root.Close()

	steps := s.steps(m.Producers, gens)
	order, err := plan.Order(steps)
	if err != nil {
		return err
	}
	w := s.opts.Stdout
	for n, i := range order {
		fmt.Fprintf(w, "%d. %s\n", n+1, steps[i].Name)
		if i < len(m.Producers) {
			p := m.Producers[i]
			run := strings.Join(p.Run, " ")
			if p.Dir != "." {
				run += " (in " + p.Dir + ")"
			}
			fmt.Fprintf(w, "   runs    %s\n", run)
			if len(p.Needs) > 0 {
				fmt.Fprintf(w, "   needs   %s\n", joinDomains(p.Needs))
			}
			for _, e := range p.Emits {
				d, ok := s.peekDomain(e)
				switch {
				case e.Foreign:
					fmt.Fprintf(w, "   writes  %s (%s, foreign)\n", e.Path, d)
				case ok:
					fmt.Fprintf(w, "   writes  %s (%s)\n", e.Path, d)
				default:
					fmt.Fprintf(w, "   writes  %s (its domain is known once it exists)\n", e.Path)
				}
			}
			continue
		}
		g := gens[i-len(m.Producers)]
		fmt.Fprintf(w, "   doc     %s\n", g.doc)
		fmt.Fprintf(w, "   needs   %s\n", joinDomains(g.needs))
		if g.template == nil {
			fmt.Fprintf(w, "   emits   %s\n", g.emits)
		}
	}
	return nil
}
