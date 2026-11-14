// Package genxtest helps the authors of producers and trivial generators
// test them with genx.
//
// An app checks from its tests that its generated files are current, as
// `genx check` does in CI:
//
//	func TestGenerated(t *testing.T) { genxtest.AssertGenerated(t) }
//
// A package that ships a trivial generator renders it on example domain
// files, with no manifest, and compares the result with golden files:
//
//	files := genxtest.Render(t, ".", "labels", genxtest.Input{Path: "testdata/settings.labels.json"})
//
// Render keeps every parameter the metafile declares at its default;
// RenderParams sets some, as an app's use entry would:
//
//	files := genxtest.RenderParams(t, ".", "labels", map[string]any{"dir": "internal/db"}, in)
package genxtest
