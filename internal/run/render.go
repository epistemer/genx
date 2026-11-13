package run

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/epistemer/genx/internal/domain"
	"github.com/epistemer/genx/internal/query"
)

// Input is a domain file for Render. Domain is empty unless the file is
// foreign and doesn't name its domain itself.
type Input struct {
	Path   string
	Domain string
}

// Render runs one generator of the metafile in dir on the given domain
// files, without a manifest, and returns the files it would write, by
// slash-separated path. It writes nothing. genxtest.Render uses it.
func Render(ctx context.Context, dir, generator string, inputs []Input) (map[string][]byte, error) {
	s := &state{
		ctx:    ctx,
		docs:   map[string][]document{},
		owners: map[string]string{},
	}
	mf, root, err := readMetafile(dir, dir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	g, err := compile(dir, mf, root, generator)
	if err != nil {
		return nil, err
	}
	for _, in := range inputs {
		b, err := os.ReadFile(in.Path)
		if err != nil {
			return nil, err
		}
		name := filepath.ToSlash(in.Path)
		v, err := query.Decode(b, isYAML(name))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		var d domain.Domain
		if in.Domain != "" {
			d, err = domain.Parse(in.Domain)
		} else if d, err = domainOf(v); err != nil {
			err = fmt.Errorf("%w (give a foreign file's Input a Domain)", err)
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		for _, need := range g.needs {
			if need.Name == d.Name && need.Major != d.Major {
				return nil, fmt.Errorf("%s needs %s, but %s says %s", g.step, need, name, d)
			}
		}
		if err := s.record(d, name, v); err != nil {
			return nil, err
		}
	}
	if err := s.generate(g); err != nil {
		return nil, err
	}
	out := map[string][]byte{}
	for _, o := range s.outputs {
		out[o.path] = o.data
	}
	return out, nil
}
