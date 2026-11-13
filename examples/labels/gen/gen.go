// Package gen ships the example's trivial generator, labels, which turns
// every labels/v1 file into Go: L10n, which gives a label's text by key,
// and a constant for each label's key that has a var. It holds no code: genx.meta.yaml and labels.go.tmpl are all
// of it. The Go file is here so that the package can be imported, which
// keeps it in go.mod and in vendor/ (genx ADR 0003).
package gen
