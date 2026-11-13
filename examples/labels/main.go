// Command labels is genx's example app: a producer written in Go
// (cmd/settings) and a trivial generator (gen), run by genx from genx.yaml.
//
// Regenerate with:
//
//	go run github.com/epistemer/genx/cmd/genx
package main

import (
	"fmt"

	"github.com/epistemer/genx/examples/labels/internal/settings"
)

func main() {
	fmt.Println(settings.Texts[settings.L10nPort])
}
