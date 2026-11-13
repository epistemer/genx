// Package labels validates labels/v1 documents according to docs/interface.md §11
// and schema/labels.v1.json.
package labels

import (
	"errors"
	"fmt"
	"go/token"

	"github.com/epistemer/genx/internal/paths"
)

// Check validates one decoded labels/v1 document against the shape
// in schema/labels.v1.json and the semantic rules in docs/interface.md §11.
// It returns all problems joined via errors.Join.
func Check(v any) error {
	obj, ok := v.(map[string]any)
	if !ok {
		return errors.New("document: must be an object")
	}

	var errs []error
	add := func(loc, msg string) {
		errs = append(errs, fmt.Errorf("%s: %s", loc, msg))
	}

	// domain: required, const "labels/v1"
	if raw, ok := obj["domain"]; !ok {
		add("domain", "required")
	} else if s, ok := raw.(string); !ok {
		add("domain", "must be a string")
	} else if s != "labels/v1" {
		add("domain", fmt.Sprintf("want %q, got %q", "labels/v1", s))
	}

	// producer: required non-empty string
	if raw, ok := obj["producer"]; !ok {
		add("producer", "required")
	} else if s, ok := raw.(string); !ok {
		add("producer", "must be a string")
	} else if s == "" {
		add("producer", "must not be empty")
	}

	// package: required object
	if raw, ok := obj["package"]; !ok {
		add("package", "required")
	} else if pkg, ok := raw.(map[string]any); !ok {
		add("package", "must be an object")
	} else {
		checkPackage(pkg, add)
	}

	// labels: required array of objects
	if raw, ok := obj["labels"]; !ok {
		add("labels", "required")
	} else if items, ok := raw.([]any); !ok {
		add("labels", "must be an array")
	} else {
		checkLabels(items, add)
	}

	return errors.Join(errs...)
}

func checkPackage(pkg map[string]any, add func(loc, msg string)) {
	if raw, ok := pkg["name"]; !ok {
		add("package.name", "required")
	} else if s, ok := raw.(string); !ok {
		add("package.name", "must be a string")
	} else if s == "" {
		add("package.name", "must not be empty")
	} else if token.IsKeyword(s) {
		add("package.name", fmt.Sprintf("%q is a Go keyword", s))
	} else if !token.IsIdentifier(s) {
		add("package.name", fmt.Sprintf("%q is not a Go identifier", s))
	} else if s == "_" {
		add("package.name", `"_"`+" is the blank identifier")
	}

	if raw, ok := pkg["dir"]; !ok {
		add("package.dir", "required")
	} else if s, ok := raw.(string); !ok {
		add("package.dir", "must be a string")
	} else if s == "" {
		add("package.dir", "must not be empty")
	} else if err := paths.Dir(s); err != nil {
		add("package.dir", err.Error())
	}
}

func checkLabels(items []any, add func(loc, msg string)) {
	seenVar := make(map[string]int)
	seenPath := make(map[string]int)

	for i, item := range items {
		loc := fmt.Sprintf("labels[%d]", i)
		lbl, ok := item.(map[string]any)
		if !ok {
			add(loc, "must be an object")
			continue
		}

		if raw, ok := lbl["key"]; !ok {
			add(loc+".key", "required")
		} else if s, ok := raw.(string); !ok {
			add(loc+".key", "must be a string")
		} else if s == "" {
			add(loc+".key", "must not be empty")
		}

		if raw, ok := lbl["text"]; !ok {
			add(loc+".text", "required")
		} else if _, ok := raw.(string); !ok {
			add(loc+".text", "must be a string")
		}

		if raw, ok := lbl["path"]; ok {
			if s, ok := raw.(string); !ok {
				add(loc+".path", "must be a string")
			} else if s == "" {
				add(loc+".path", "must not be empty")
			} else if first, ok := seenPath[s]; ok {
				add(loc+".path", fmt.Sprintf("%q is declared twice (also labels[%d])", s, first))
			} else {
				seenPath[s] = i
			}
		}

		if raw, ok := lbl["var"]; ok {
			if s, ok := raw.(string); !ok {
				add(loc+".var", "must be a string")
			} else if s == "" {
				add(loc+".var", "must not be empty")
			} else if token.IsKeyword(s) {
				add(loc+".var", fmt.Sprintf("%q is a Go keyword", s))
			} else if !token.IsIdentifier(s) {
				add(loc+".var", fmt.Sprintf("%q is not a Go identifier", s))
			} else if s == "_" {
				add(loc+".var", `"_"`+" is the blank identifier")
			} else if s == "L10n" || s == "l10n" {
				add(loc+".var", fmt.Sprintf("%q is reserved", s))
			} else if first, ok := seenVar[s]; ok {
				add(loc+".var", fmt.Sprintf("%q is declared twice (also labels[%d])", s, first))
			} else {
				seenVar[s] = i
			}
		}
	}
}
