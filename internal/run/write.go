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
)

// write writes every output whose bytes differ from the file on disk,
// through a temporary file in the same directory and a rename, all inside
// the module root.
func (s *state) write() error {
	for _, o := range s.outputs {
		name := filepath.FromSlash(o.path)
		if old, err := s.root.ReadFile(name); err == nil && bytes.Equal(old, o.data) {
			continue
		}
		if err := s.writeFile(o.path, o.data); err != nil {
			return fmt.Errorf("%s: writing %s: %w", o.step, o.path, err)
		}
		fmt.Fprintf(s.opts.Stdout, "wrote %s\n", o.path)
	}
	return nil
}

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
