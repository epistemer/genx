package main

import (
	"testing"

	"github.com/epistemer/genx/genxtest"
)

// TestGenerated fails when a generated file is stale, as genx check does.
func TestGenerated(t *testing.T) {
	genxtest.AssertGenerated(t)
}
