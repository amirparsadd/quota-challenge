package buildinfo

import "testing"

func TestStringUsesInjectedMetadata(t *testing.T) {
	got := String()
	if got != Version+" "+Commit {
		t.Fatalf("String() = %q, want %q + space + %q", got, Version, Commit)
	}
}