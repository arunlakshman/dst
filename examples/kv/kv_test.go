package kv_test

import (
	"testing"

	"github.com/manifesterio/dst/examples/kv"
)

func TestDeterministic(t *testing.T) {
	a := kv.Run(42)
	b := kv.Run(42)
	if a.Hash != b.Hash {
		t.Fatalf("hash %016x != %016x", a.Hash, b.Hash)
	}
	if a.Calls != b.Calls {
		t.Fatalf("Calls %d != %d", a.Calls, b.Calls)
	}
	if a.Steps != b.Steps {
		t.Fatalf("Steps %d != %d", a.Steps, b.Steps)
	}
	if a.Steps == 0 {
		t.Fatal("no steps")
	}
}

func TestDifferentSeedsDiverge(t *testing.T) {
	a := kv.Run(1)
	b := kv.Run(2)
	if a.Hash == b.Hash && a.Calls == b.Calls {
		t.Fatal("distinct seeds produced identical Result")
	}
}
