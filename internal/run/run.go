// Package run runs an app's manifest: producers, then trivial generators,
// writing only the files whose bytes changed (interface.md §9).
package run

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/epistemer/genx/internal/domain"
	"github.com/epistemer/genx/internal/manifest"
	"github.com/epistemer/genx/internal/plan"
)

// ManifestFile is the manifest's name at the module root.
const ManifestFile = "genx.yaml"

// Options configures a run.
type Options struct {
	// Dir is the working directory; the module root is the nearest
	// directory at or above it that holds a go.mod.
	Dir string
	// Stdout receives progress and producers' output; Stderr, producers'
	// errors.
	Stdout, Stderr io.Writer
}

// Run runs every step of the manifest at the module root above opts.Dir.
func Run(ctx context.Context, opts Options) error {
	s, m, gens, err := load(ctx, opts)
	if err != nil {
		return err
	}
	defer s.root.Close()
	steps := s.steps(m.Producers, gens)
	pl, err := plan.New(steps)
	if err != nil {
		return err
	}
	for {
		i, ok, err := pl.Next()
		if err != nil {
			return err
		}
		if !ok {
			break
		}
		var wrote []domain.Domain
		if i < len(m.Producers) {
			wrote, err = s.produce(m.Producers[i])
		} else {
			g := gens[i-len(m.Producers)]
			wrote, err = g.writes(), s.generate(g)
		}
		if err != nil {
			return err
		}
		if err := pl.Done(i, wrote); err != nil {
			return err
		}
	}
	return s.write()
}

// load finds the module root, reads and checks the manifest, claims every
// path the producers declare, and compiles the generators in use. It runs
// nothing. The caller closes s.root.
func load(ctx context.Context, opts Options) (*state, *manifest.Manifest, []generator, error) {
	if opts.Stdout == nil {
		opts.Stdout = io.Discard
	}
	if opts.Stderr == nil {
		opts.Stderr = io.Discard
	}
	rootDir, err := FindRoot(opts.Dir)
	if err != nil {
		return nil, nil, nil, err
	}
	root, err := os.OpenRoot(rootDir)
	if err != nil {
		return nil, nil, nil, err
	}
	s := &state{
		ctx:     ctx,
		opts:    opts,
		rootDir: rootDir,
		root:    root,
		docs:    map[string][]document{},
		owners:  map[string]string{},
	}
	m, gens, err := s.load()
	if err != nil {
		root.Close()
		return nil, nil, nil, err
	}
	return s, m, gens, nil
}

func (s *state) load() (*manifest.Manifest, []generator, error) {
	data, err := s.root.ReadFile(ManifestFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, fmt.Errorf("no %s in the module root %s", ManifestFile, s.rootDir)
	}
	if err != nil {
		return nil, nil, err
	}
	m, err := manifest.Parse(ManifestFile, data)
	if err != nil {
		return nil, nil, err
	}
	// Every path a producer declares is known before anything runs.
	for _, p := range m.Producers {
		for _, e := range p.Emits {
			if err := s.claim(e.Path, "producer "+p.Name); err != nil {
				return nil, nil, err
			}
		}
	}
	gens, err := s.resolve(m.Use)
	if err != nil {
		return nil, nil, err
	}
	return m, gens, nil
}

// FindRoot returns the nearest directory at or above dir that holds a
// go.mod.
func FindRoot(dir string) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for d := dir; ; {
		if fi, err := os.Stat(filepath.Join(d, "go.mod")); err == nil && !fi.IsDir() {
			return d, nil
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", fmt.Errorf("no go.mod in %s or any directory above it", dir)
		}
		d = parent
	}
}

// state is one run in progress.
type state struct {
	ctx     context.Context
	opts    Options
	rootDir string
	root    *os.Root

	// docs holds every domain document read or rendered so far, by domain,
	// in the order they arrived; generate sorts them by path.
	docs map[string][]document
	// owners maps each folded path to the step that writes it.
	owners map[string]string
	// outputs are the generators' files, in the order they were rendered.
	outputs []output
}

// document is one domain file's decoded value.
type document struct {
	path  string
	value any
}

// output is one file a generator rendered.
type output struct {
	path string
	data []byte
	step string
}

// steps describes producers, then generators in use order, for the plan. A
// producer's domains are known before it runs when the manifest gives them
// (a foreign file) or its declared files are already on disk; otherwise the
// plan learns them when it has run (internal/plan).
func (s *state) steps(producers []manifest.Producer, gens []generator) []plan.Step {
	var steps []plan.Step
	for _, p := range producers {
		st := plan.Step{Name: "producer " + p.Name, Needs: p.Needs, Known: true}
		for _, e := range p.Emits {
			d, ok := s.peekDomain(e)
			if !ok {
				st.Known = false
				continue
			}
			st.Writes = appendDomain(st.Writes, d)
		}
		steps = append(steps, st)
	}
	for _, g := range gens {
		steps = append(steps, plan.Step{Name: g.step, Needs: g.needs, Writes: g.writes(), Known: true})
	}
	return steps
}

// peekDomain reads the domain of a declared file before its producer runs:
// the manifest's, for a foreign file, else the file's own, if it exists.
func (s *state) peekDomain(e manifest.Emit) (domain.Domain, bool) {
	if e.Foreign {
		return e.Domain, true
	}
	_, d, err := s.readDomainFile(e)
	return d, err == nil
}

func appendDomain(ds []domain.Domain, d domain.Domain) []domain.Domain {
	for _, x := range ds {
		if x == d {
			return ds
		}
	}
	return append(ds, d)
}
