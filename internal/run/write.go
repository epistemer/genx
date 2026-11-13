package run

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/epistemer/genx/internal/render"
)

// write checks every rule of interface.md §7–8 against the disk, then
// deletes leftover output, writes every output whose bytes changed, and
// writes the lock. Nothing is touched unless every check passes.
func (s *state) write(prev lock) error {
	leftovers, err := s.preflight(prev)
	if err != nil {
		return err
	}
	for _, p := range leftovers {
		if err := s.root.Remove(filepath.FromSlash(p)); err != nil {
			return fmt.Errorf("deleting %s, which %s wrote: %w", p, prev.steps[p], err)
		}
		fmt.Fprintf(s.opts.Stdout, "deleted %s\n", p)
	}
	for _, o := range s.outputs {
		if old, err := s.root.ReadFile(filepath.FromSlash(o.path)); err == nil && bytes.Equal(old, o.data) {
			continue
		}
		if err := s.writeFile(o.path, o.data); err != nil {
			return fmt.Errorf("%s: writing %s: %w", o.step, o.path, err)
		}
		fmt.Fprintf(s.opts.Stdout, "wrote %s\n", o.path)
	}
	next := s.newLock()
	if !prev.exists && len(next.steps) == 0 {
		return nil
	}
	data := next.encode()
	if old, err := s.root.ReadFile(LockFile); err == nil && bytes.Equal(old, data) {
		return nil
	}
	if err := s.writeFile(LockFile, data); err != nil {
		return fmt.Errorf("writing %s: %w", LockFile, err)
	}
	return nil
}

// preflight returns the leftovers to delete: files in the previous lock
// that this run doesn't produce, still on disk. It fails, naming every
// file, when an output would replace a file genx didn't write, or a file
// genx wrote has lost its generated-code header.
func (s *state) preflight(prev lock) ([]string, error) {
	var problems []string
	produced := map[string]bool{}
	for _, o := range s.outputs {
		produced[o.path] = true
		b, err := s.root.ReadFile(filepath.FromSlash(o.path))
		if err != nil {
			continue // nothing there to replace
		}
		switch {
		case !prev.has(o.path):
			problems = append(problems, fmt.Sprintf("%s: %s exists and isn't in %s, so genx didn't write it; move it away, then run genx again", o.step, o.path, LockFile))
		case isGo(o.path) && !render.HasHeader(b):
			problems = append(problems, fmt.Sprintf("%s: %s has lost its generated-code header, so it may hold edits; restore the header or delete the file, then run genx again", o.step, o.path))
		}
	}
	var leftovers []string
	for _, p := range prev.sorted() {
		if produced[p] {
			continue
		}
		b, err := s.root.ReadFile(filepath.FromSlash(p))
		if errors.Is(err, os.ErrNotExist) {
			continue // gone already; the new lock drops it
		}
		if err != nil {
			return nil, err
		}
		if isGo(p) && !render.HasHeader(b) {
			problems = append(problems, fmt.Sprintf("%s, which %s wrote, is no longer generated, but it has lost its generated-code header, so it may hold edits; delete it or move it, then run genx again", p, prev.steps[p]))
			continue
		}
		leftovers = append(leftovers, p)
	}
	if len(problems) > 0 {
		return nil, errors.New(strings.Join(problems, "\n"))
	}
	return leftovers, nil
}

// compare is genx check's end: every output, leftover and the lock against
// the disk. It reports every problem, with what to run.
func (s *state) compare(prev lock) error {
	produced := map[string]bool{}
	for _, o := range s.outputs {
		produced[o.path] = true
		b, err := s.root.ReadFile(filepath.FromSlash(o.path))
		switch {
		case errors.Is(err, os.ErrNotExist):
			s.problem("missing   %s (%s)", o.path, o.step)
		case err != nil:
			return err
		case !bytes.Equal(b, o.data):
			s.problem("stale     %s (%s)", o.path, o.step)
		}
	}
	for _, p := range prev.sorted() {
		if produced[p] {
			continue
		}
		if _, err := s.root.Stat(filepath.FromSlash(p)); err == nil {
			s.problem("leftover  %s (%s wrote it; nothing does now)", p, prev.steps[p])
		}
	}
	next := s.newLock()
	if prev.exists || len(next.steps) > 0 {
		old, err := s.root.ReadFile(LockFile)
		if err != nil || !bytes.Equal(old, next.encode()) {
			s.problem("stale     %s", LockFile)
		}
	}
	if len(s.problems) == 0 {
		return nil
	}
	n := "problems"
	if len(s.problems) == 1 {
		n = "problem"
	}
	return fmt.Errorf("check found %d %s; run genx:\n  %s", len(s.problems), n, strings.Join(s.problems, "\n  "))
}

func (s *state) problem(format string, args ...any) {
	s.problems = append(s.problems, fmt.Sprintf(format, args...))
}

func isGo(p string) bool { return strings.HasSuffix(p, ".go") }

// writeFile replaces p atomically.
func (s *state) writeFile(p string, data []byte) error {
	dir := path.Dir(p)
	if dir != "." {
		if err := s.root.MkdirAll(filepath.FromSlash(dir), 0o755); err != nil {
			return err
		}
	}
	var r [6]byte
	if _, err := rand.Read(r[:]); err != nil {
		return err
	}
	tmp := filepath.FromSlash(path.Join(dir, ".genx-"+hex.EncodeToString(r[:])+"-"+path.Base(p)))
	f, err := s.root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, werr := f.Write(data)
	cerr := f.Close()
	if werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = s.root.Rename(tmp, filepath.FromSlash(p))
	}
	if werr != nil {
		if rerr := s.root.Remove(tmp); rerr != nil && !errors.Is(rerr, os.ErrNotExist) {
			werr = errors.Join(werr, fmt.Errorf("removing the temporary file %s: %w", filepath.ToSlash(tmp), rerr))
		}
	}
	return werr
}
