package run

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/epistemer/genx/internal/manifest"
	"github.com/epistemer/genx/internal/meta"
	"github.com/epistemer/genx/internal/query"
)

// ParamsVar is the variable every query of a metafile reads its parameters
// from (interface amendment A4).
const ParamsVar = "$params"

// param is one parameter as a generator runs with it.
type param struct {
	name  string
	value any
	byApp bool // set by the app, not left at its default
}

// setting is a value given to a parameter: from a use entry, or from
// genxtest.RenderParams. where locates it at the start of a message.
type setting struct {
	name  string
	value any
	where string
}

// manifestSettings locates the values a use entry sets in genx.yaml.
func manifestSettings(u manifest.Use) []setting {
	var set []setting
	for _, p := range u.Params {
		set = append(set, setting{
			name:  p.Name,
			value: p.Value,
			where: fmt.Sprintf("%s:%d: %s.params.%s", ManifestFile, p.Line, u.Path, p.Name),
		})
	}
	return set
}

// bindParams checks the values given to owner's declared parameters and
// returns every parameter, by name, with the value its generators see:
// the one given, or else the default. An unknown name is an error, and so
// is a value whose JSON type differs from a non-null default's. All the
// problems are reported at once.
func bindParams(owner string, declared []meta.Param, set []setting) ([]param, error) {
	byName := map[string]meta.Param{}
	var names []string
	for _, d := range declared {
		byName[d.Name] = d
		names = append(names, d.Name)
	}
	sort.Strings(names)
	given := map[string]setting{}
	var errs []error
	for _, s := range set {
		d, ok := byName[s.name]
		if !ok {
			has := "it declares none"
			if len(names) > 0 {
				has = "it declares: " + strings.Join(names, ", ")
			}
			errs = append(errs, fmt.Errorf("%s: %s declares no parameter %q (%s)", s.where, owner, s.name, has))
			continue
		}
		if d.Default != nil && query.Kind(s.value) != query.Kind(d.Default) {
			errs = append(errs, fmt.Errorf("%s: must be %s, as its default %s is (got %s)",
				s.where, article(query.Kind(d.Default)), query.Compact(d.Default), article(query.Kind(s.value))))
			continue
		}
		given[s.name] = s
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	params := make([]param, 0, len(names))
	for _, n := range names {
		p := param{name: n, value: byName[n].Default}
		if s, ok := given[n]; ok {
			p.value, p.byApp = s.value, true
		}
		params = append(params, p)
	}
	return params, nil
}

// paramsObject is the value of $params: every parameter by name.
func paramsObject(params []param) map[string]any {
	obj := make(map[string]any, len(params))
	for _, p := range params {
		obj[p.name] = p.value
	}
	return obj
}

func article(kind string) string {
	switch kind {
	case "null":
		return "null"
	case "array", "object":
		return "an " + kind
	default:
		return "a " + kind
	}
}
