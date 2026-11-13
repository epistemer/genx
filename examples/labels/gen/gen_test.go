package gen

import (
	"bytes"
	"flag"
	"os"
	"testing"

	"github.com/epistemer/genx/genxtest"
)

var update = flag.Bool("update", false, "rewrite the golden file")

// TestLabels renders the generator on a labels file, with no manifest, and
// compares the result with a golden file.
func TestLabels(t *testing.T) {
	files := genxtest.Render(t, ".", "labels", genxtest.Input{Path: "testdata/settings.labels.json"})
	got, ok := files["internal/settings/zz_settings_labels.go"]
	if !ok || len(files) != 1 {
		t.Fatalf("got files %v, want internal/settings/zz_settings_labels.go only", keys(files))
	}
	const golden = "testdata/zz_settings_labels.go.golden"
	if *update {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func keys(m map[string][]byte) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
