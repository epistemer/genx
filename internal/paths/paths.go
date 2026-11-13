// Package paths validates and normalizes output paths and producer paths
// according to the rules in docs/interface.md §7.
package paths

import (
	"fmt"
	"path"
	"strings"
)

var windowsReserved = map[string]bool{
	"CON":  true,
	"PRN":  true,
	"AUX":  true,
	"NUL":  true,
	"COM1": true,
	"COM2": true,
	"COM3": true,
	"COM4": true,
	"COM5": true,
	"COM6": true,
	"COM7": true,
	"COM8": true,
	"COM9": true,
	"LPT1": true,
	"LPT2": true,
	"LPT3": true,
	"LPT4": true,
	"LPT5": true,
	"LPT6": true,
	"LPT7": true,
	"LPT8": true,
	"LPT9": true,
}

// Fold returns the lowercased path for case-insensitive comparison.
func Fold(p string) string {
	return strings.ToLower(p)
}

// Declared validates a declared path (such as a producer's emit path).
// The path must equal path.Clean(p), must not be ".", must not be absolute,
// must not contain ".." elements, each element must contain only [A-Za-z0-9._-],
// must not start or end with '.', and no element's stem before the first '.' may be
// a Windows reserved device name in any case.
func Declared(p string) error {
	cleaned := path.Clean(p)
	if p != cleaned {
		return fmt.Errorf("%q: not clean (use %q)", p, cleaned)
	}
	return validateElements(p, cleaned)
}

// Dir validates a directory path (such as producers[].dir or labels package.dir).
// "." is allowed; otherwise the rules of Declared apply.
func Dir(p string) error {
	if p == "." {
		return nil
	}
	return Declared(p)
}

// Out cleans p and validates that it is a valid generator output path.
// In addition to the Declared rules, the base name must start with "zz_".
// It returns the cleaned path.
func Out(p string) (string, error) {
	cleaned := path.Clean(p)
	if err := validateElements(p, cleaned); err != nil {
		return "", err
	}
	base := path.Base(cleaned)
	if !strings.HasPrefix(base, "zz_") {
		return "", fmt.Errorf("%q: the base name must start with zz_", p)
	}
	return cleaned, nil
}

func validateElements(raw, cleaned string) error {
	if cleaned == "." {
		return fmt.Errorf("%q: must not be \".\"", raw)
	}
	if path.IsAbs(cleaned) || strings.HasPrefix(raw, "/") {
		return fmt.Errorf("%q: must not be absolute", raw)
	}
	elements := strings.Split(cleaned, "/")
	for _, elem := range elements {
		if elem == ".." {
			return fmt.Errorf("%q: must not contain \"..\"", raw)
		}
		if elem == "" {
			return fmt.Errorf("%q: contains empty element", raw)
		}
		if strings.HasPrefix(elem, ".") {
			return fmt.Errorf("%q: element %q must not start with '.'", raw, elem)
		}
		if strings.HasSuffix(elem, ".") {
			return fmt.Errorf("%q: element %q must not end with '.'", raw, elem)
		}
		for i := 0; i < len(elem); i++ {
			c := elem[i]
			if !isAllowedChar(c) {
				return fmt.Errorf("%q: element %q contains invalid character %q", raw, elem, c)
			}
		}
		stem := elem
		if idx := strings.IndexByte(elem, '.'); idx != -1 {
			stem = elem[:idx]
		}
		if windowsReserved[strings.ToUpper(stem)] {
			return fmt.Errorf("%q: %q is a reserved name on Windows", raw, stem)
		}
	}
	return nil
}

func isAllowedChar(c byte) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '.' || c == '_' || c == '-'
}
