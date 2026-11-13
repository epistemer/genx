// Package plan orders producers and generators by the domains they write and need.
//
// A run has steps: producers (external programs) then generators, in manifest order.
// Each step needs some domains and writes some domains. A step that needs domain D
// must run after every step that writes D; among ready steps the lowest index goes first.
// Generators' writes are known up front. A producer's writes are known up front only
// when genx could read them before the run; otherwise they are unknown until it has run.
// Plan is a scheduler that learns as it goes.
package plan
