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

	"github.com/epistemer/genx/internal/manifest"
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
	rootDir, err := FindRoot(opts.Dir)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(rootDir)
	if err != nil {
		return err
	}
	defer root.Close()

	data, err := root.ReadFile(ManifestFile)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("no %s in the module root %s", ManifestFile, rootDir)
	}
	if err != nil {
		return err
	}
	m, err := manifest.Parse(ManifestFile, data)
	if err != nil {
		return err
	}

	s := &state{
		ctx:     ctx,
		opts:    opts,
		rootDir: rootDir,
		root:    root,
		docs:    map[string][]document{},
		owners:  map[string]string{},
	}
	for _, p := range m.Producers {
		if err := s.produce(p); err != nil {
			return err
		}
	}
	gens, err := s.resolve(m.Use)
	if err != nil {
		return err
	}
	for _, g := range gens {
		if err := s.generate(g); err != nil {
			return err
		}
	}
	return s.write()
}

// FindRoot returns the nearest directory at or above dir that holds a
// go.mod.
func FindRoot(dir string) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for d := dir; ; {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
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
