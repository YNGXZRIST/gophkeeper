package main

import "testing"

func TestGetAnalyzers_returnsList(t *testing.T) {
	a := getAnalyzers()
	if len(a) < 10 {
		t.Fatalf("expected many analyzers, got %d", len(a))
	}
}
