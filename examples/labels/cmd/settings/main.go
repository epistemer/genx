// Command settings is the example's producer: it writes the labels of the
// app's settings as a labels/v1 file. A real producer, such as cfgx, reads
// them from a schema; this one keeps them in a table.
//
// With -check it writes nothing, and fails if the file isn't what it would
// write, so genx check can run it.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

// out is where the labels go, relative to the module root, where genx runs
// this producer.
const out = "internal/settings/zz_settings.labels.json"

// setting is one setting the app has.
type setting struct {
	path, text, name string
}

var settings = []setting{
	{"server.port", "Listen port", "Port"},
	{"server.host", "Listen address", "Host"},
	{"log.level", "Minimum log level", "LogLevel"},
}

func main() {
	check := flag.Bool("check", false, "write nothing; fail if the file is stale")
	flag.Parse()
	if err := run(*check); err != nil {
		fmt.Fprintln(os.Stderr, "settings:", err)
		os.Exit(1)
	}
}

func run(check bool) error {
	labels := []map[string]string{}
	for _, s := range settings {
		labels = append(labels, map[string]string{
			"key":  "settings." + s.path,
			"text": s.text,
			"path": s.path,
			"var":  "L10n" + s.name,
		})
	}
	doc := map[string]any{
		"domain":   "labels/v1",
		"producer": "settings",
		"package":  map[string]string{"name": "settings", "dir": "internal/settings"},
		"labels":   labels,
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if check {
		old, err := os.ReadFile(out)
		if err != nil || !bytes.Equal(old, b) {
			return fmt.Errorf("%s is stale; run genx", out)
		}
		return nil
	}
	return os.WriteFile(out, b, 0o644)
}
