package run

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/epistemer/genx/internal/domain"
	"github.com/epistemer/genx/internal/manifest"
	"github.com/epistemer/genx/internal/paths"
	"github.com/epistemer/genx/internal/query"
)

// produce runs a producer's command, then reads every file it declared. It
// returns the domains the producer wrote.
func (s *state) produce(p manifest.Producer) ([]domain.Domain, error) {
	if err := s.command(p, p.Run); err != nil {
		return nil, err
	}
	return s.readEmits(p)
}

// checkProducer is produce for genx check: it runs the producer's check
// command, if it has one, and records a failure as a problem; a producer
// without one is trusted, and says so. Either way its declared files must
// be there for the generators after it.
func (s *state) checkProducer(p manifest.Producer) ([]domain.Domain, error) {
	if len(p.Check) == 0 {
		fmt.Fprintf(s.opts.Stdout, "%s: not checked (no check command)\n", p.Name)
	} else if err := s.command(p, p.Check); err != nil {
		if s.ctx.Err() != nil {
			return nil, err
		}
		s.problem("producer  %s: %s", p.Name, strings.TrimPrefix(err.Error(), "producer "+p.Name+": "))
	}
	// A file that's missing or broken is a problem to report with the
	// rest; the steps after it read what's there.
	var wrote []domain.Domain
	for _, e := range p.Emits {
		v, d, err := s.readDomainFile(e)
		if err != nil {
			s.problem("producer  %s: %v", p.Name, err)
			continue
		}
		s.docs[d.String()] = append(s.docs[d.String()], document{path: e.Path, value: v})
		wrote = appendDomain(wrote, d)
	}
	return wrote, nil
}

// command runs argv for producer p in its directory, under its timeout,
// with its output prefixed by its name.
func (s *state) command(p manifest.Producer, argv []string) error {
	step := "producer " + p.Name
	ctx, cancel := context.WithTimeout(s.ctx, p.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = filepath.Join(s.rootDir, filepath.FromSlash(p.Dir))
	var mu sync.Mutex
	stdout := &prefixWriter{w: s.opts.Stdout, prefix: p.Name + ": ", mu: &mu}
	stderr := &prefixWriter{w: s.opts.Stderr, prefix: p.Name + ": ", mu: &mu}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	err := cmd.Run()
	stdout.flush()
	stderr.flush()
	if s.ctx.Err() != nil {
		return s.ctx.Err()
	}
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("%s: timed out after %s", step, p.Timeout)
	}
	if err != nil {
		return fmt.Errorf("%s: %s: %w", step, strings.Join(argv, " "), err)
	}
	return nil
}

// readEmits reads every file a producer declared, and returns their domains.
func (s *state) readEmits(p manifest.Producer) ([]domain.Domain, error) {
	var wrote []domain.Domain
	for _, e := range p.Emits {
		v, d, err := s.readDomainFile(e)
		if err != nil {
			return nil, fmt.Errorf("producer %s: %w", p.Name, err)
		}
		s.docs[d.String()] = append(s.docs[d.String()], document{path: e.Path, value: v})
		wrote = appendDomain(wrote, d)
	}
	return wrote, nil
}

// readDomainFile decodes a declared file and works out its domain: the one
// the manifest gives a foreign file, else the file's own domain field.
func (s *state) readDomainFile(e manifest.Emit) (any, domain.Domain, error) {
	b, err := s.root.ReadFile(filepath.FromSlash(e.Path))
	if errors.Is(err, os.ErrNotExist) {
		return nil, domain.Domain{}, fmt.Errorf("%s: declared in emits, but the producer didn't write it", e.Path)
	}
	if err != nil {
		return nil, domain.Domain{}, err
	}
	v, err := query.Decode(b, isYAML(e.Path))
	if err != nil {
		return nil, domain.Domain{}, fmt.Errorf("%s: %w", e.Path, err)
	}
	if e.Foreign {
		return v, e.Domain, nil
	}
	d, err := domainOf(v)
	if err != nil {
		return nil, domain.Domain{}, fmt.Errorf("%s: %w", e.Path, err)
	}
	return v, d, nil
}

// domainOf reads a document's top-level domain field.
func domainOf(v any) (domain.Domain, error) {
	obj, ok := v.(map[string]any)
	if !ok {
		return domain.Domain{}, errors.New(`not an object, so it can't name its domain (a foreign file needs "domain" in the manifest)`)
	}
	raw, ok := obj["domain"].(string)
	if !ok {
		return domain.Domain{}, errors.New(`no "domain" field (a foreign file needs "domain" in the manifest)`)
	}
	return domain.Parse(raw)
}

// claim records that step writes p; a second step writing the same path,
// compared without case, is an error.
func (s *state) claim(p, step string) error {
	key := paths.Fold(p)
	if other, ok := s.owners[key]; ok {
		return fmt.Errorf("%s writes %s, which %s writes too", step, p, other)
	}
	s.owners[key] = step
	return nil
}

func isYAML(p string) bool {
	return strings.HasSuffix(p, ".yaml") || strings.HasSuffix(p, ".yml")
}

// prefixWriter writes each line with a prefix, holding a partial line until
// it ends.
type prefixWriter struct {
	w      io.Writer
	prefix string
	mu     *sync.Mutex
	buf    bytes.Buffer
}

func (p *prefixWriter) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.buf.Write(b)
	for {
		i := bytes.IndexByte(p.buf.Bytes(), '\n')
		if i < 0 {
			return len(b), nil
		}
		line := p.buf.Next(i + 1)
		if _, err := fmt.Fprintf(p.w, "%s%s", p.prefix, line); err != nil {
			return 0, err
		}
	}
}

func (p *prefixWriter) flush() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.buf.Len() > 0 {
		fmt.Fprintf(p.w, "%s%s\n", p.prefix, p.buf.Bytes())
		p.buf.Reset()
	}
}
