package run

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/epistemer/genx/internal/paths"
	"github.com/epistemer/genx/internal/query"
	"github.com/epistemer/genx/internal/render"
)

// GeneratorTimeout is the time a generator's queries share (interface.md §5).
const GeneratorTimeout = 30 * time.Second

// generate runs one generator: its units query over the input, then, for
// each unit, out and the template (or the JSON encoding of an emitted
// domain). Nothing is written here.
func (s *state) generate(g generator) error {
	ctx, cancel := context.WithTimeout(s.ctx, GeneratorTimeout)
	defer cancel()

	input, files := s.input(g)
	units, err := g.units.Run(ctx, input, g.vars)
	if err != nil {
		return s.queryError(ctx, g, "units", err, files)
	}
	for _, unit := range units {
		if err := s.timedOut(ctx, g); err != nil {
			return err
		}
		vals, err := g.out.Run(ctx, unit, g.vars)
		if err != nil {
			return s.queryError(ctx, g, "out", err, files)
		}
		if len(vals) != 1 {
			return fmt.Errorf("%s: out gave %d values for a unit; it must give one path", g.step, len(vals))
		}
		raw, ok := vals[0].(string)
		if !ok {
			return fmt.Errorf("%s: out gave %s; it must give a path as a string", g.step, describe(vals[0]))
		}
		p, err := paths.Out(raw)
		if err != nil {
			return fmt.Errorf("%s: out: %w", g.step, err)
		}

		var data []byte
		if g.template == nil {
			if err := s.claim(p, g.step); err != nil {
				return err
			}
			if data, err = s.emit(g, p, unit); err != nil {
				return err
			}
		} else {
			b, err := g.template.Execute(unit)
			if err != nil {
				return fmt.Errorf("%s: %w", g.step, err)
			}
			b, empty, err := render.Finish(p, b)
			if err != nil {
				return fmt.Errorf("%s: %w", g.step, err)
			}
			if empty {
				continue
			}
			if err := s.claim(p, g.step); err != nil {
				return err
			}
			data = b
		}
		s.outputs = append(s.outputs, output{path: p, data: data, step: g.step})
	}
	return nil
}

// input builds the object a generator's units query reads: each needed
// domain's documents, in the byte order of their paths (interface.md §5).
// It also returns the input files' paths, for messages.
func (s *state) input(g generator) (map[string]any, []string) {
	in := map[string]any{}
	var files []string
	for _, d := range g.needs {
		docs := append([]document(nil), s.docs[d.String()]...)
		sort.Slice(docs, func(i, j int) bool { return docs[i].path < docs[j].path })
		vals := make([]any, 0, len(docs))
		for _, doc := range docs {
			vals = append(vals, doc.value)
			files = append(files, doc.path)
		}
		in[d.String()] = vals
	}
	return in, files
}

// emit checks a unit of a generator that writes a domain, encodes it, and
// makes it available to later generators.
func (s *state) emit(g generator, p string, unit any) ([]byte, error) {
	d, err := domainOf(unit)
	if err != nil {
		return nil, fmt.Errorf("%s: a unit for %s: %w (each unit must be an object with \"domain\": %q)", g.step, p, err, g.emits.String())
	}
	if d != g.emits {
		return nil, fmt.Errorf("%s: a unit for %s says domain %s; the generator emits %s", g.step, p, d, g.emits)
	}
	data, err := query.Encode(unit)
	if err != nil {
		return nil, fmt.Errorf("%s: %s: %w", g.step, p, err)
	}
	// Later generators read what this one emits, as they would read it from
	// the file: decoded from the bytes genx writes.
	v, err := query.Decode(data, false)
	if err != nil {
		return nil, fmt.Errorf("%s: %s: %w", g.step, p, err)
	}
	if err := s.record(d, p, v); err != nil {
		return nil, fmt.Errorf("%s: %w", g.step, err)
	}
	return data, nil
}

// timedOut reports the run's own cancellation, or the generator's timeout.
func (s *state) timedOut(ctx context.Context, g generator) error {
	if s.ctx.Err() != nil {
		return s.ctx.Err()
	}
	if ctx.Err() != nil {
		return fmt.Errorf("%s: timed out after %s", g.step, GeneratorTimeout)
	}
	return nil
}

// queryError names the generator and its input files with a query's error.
func (s *state) queryError(ctx context.Context, g generator, which string, err error, files []string) error {
	if terr := s.timedOut(ctx, g); terr != nil {
		return terr
	}
	var qe *query.Error
	if errors.As(err, &qe) {
		in := "no input files"
		if len(files) > 0 {
			in = "input: " + strings.Join(files, ", ")
		}
		return fmt.Errorf("%s: %s: %s\n(%s)", g.step, which, qe.Error(), in)
	}
	return fmt.Errorf("%s: %s: %w", g.step, which, err)
}

func describe(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case map[string]any:
		return "an object"
	case []any:
		return "an array"
	case bool:
		return "a boolean"
	default:
		return "a number"
	}
}
