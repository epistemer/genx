// Package domain implements parsing and validation of domain names according to docs/interface.md §10.
package domain

import (
	"fmt"
	"strconv"
	"strings"
)

// Domain represents a domain identifier of the form name/vN.
type Domain struct {
	Name  string
	Major int
}

// String returns the formatted domain name/vN, or "" for a zero Domain.
func (d Domain) String() string {
	if d.Name == "" && d.Major == 0 {
		return ""
	}
	return fmt.Sprintf("%s/v%d", d.Name, d.Major)
}

// Parse parses a domain string of the form name/vN.
// name is one or more parts matching [a-z][a-z0-9-]* joined by '.';
// N matches [1-9][0-9]*.
func Parse(s string) (Domain, error) {
	idx := strings.IndexByte(s, '/')
	if idx == -1 {
		return Domain{}, fmt.Errorf("%q: invalid domain", s)
	}
	name := s[:idx]
	ver := s[idx+1:]

	// Validate name: one or more parts joined by '.'
	if name == "" {
		return Domain{}, fmt.Errorf("%q: invalid domain", s)
	}
	parts := strings.Split(name, ".")
	for _, part := range parts {
		if !isValidPart(part) {
			return Domain{}, fmt.Errorf("%q: invalid domain", s)
		}
	}

	// Validate version: "v" + [1-9][0-9]*
	if !strings.HasPrefix(ver, "v") {
		return Domain{}, fmt.Errorf("%q: invalid domain", s)
	}
	numStr := ver[1:]
	if len(numStr) == 0 {
		return Domain{}, fmt.Errorf("%q: invalid domain", s)
	}
	if numStr[0] < '1' || numStr[0] > '9' {
		return Domain{}, fmt.Errorf("%q: invalid domain", s)
	}
	for i := 1; i < len(numStr); i++ {
		if numStr[i] < '0' || numStr[i] > '9' {
			return Domain{}, fmt.Errorf("%q: invalid domain", s)
		}
	}
	major, err := strconv.Atoi(numStr)
	if err != nil {
		return Domain{}, fmt.Errorf("%q: invalid domain", s)
	}

	return Domain{Name: name, Major: major}, nil
}

func isValidPart(p string) bool {
	if len(p) == 0 {
		return false
	}
	if p[0] < 'a' || p[0] > 'z' {
		return false
	}
	for i := 1; i < len(p); i++ {
		c := p[i]
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}
	return true
}
