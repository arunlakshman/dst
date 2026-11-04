package fault_test

import (
	"testing"

	"github.com/arunlakshman/dst/fault"
	"github.com/arunlakshman/dst/rng"
)

func TestQuietSilencesFaultOnly(t *testing.T) {
	r := rng.New(42)
	inj := fault.New(r)
	inj.SetQuiet(true)
	if !inj.Quiet() {
		t.Fatal("Quiet not set")
	}
	before := r.Calls()
	if inj.Fault("drop", 1.0) {
		t.Fatal("Fault fired while quiet")
	}
	if r.Calls() != before {
		t.Fatal("quiet Fault must not draw")
	}
	// Maybe still draws (activate 0.5, then fire 0.05).
	_ = inj.Maybe("site")
	if r.Calls() <= before {
		t.Fatal("Maybe drew nothing while quiet")
	}
}

func TestMaybeLatchesPerSite(t *testing.T) {
	// First call draws Chance(0.5) to latch, then Chance(0.05) only if active.
	// Find a seed where site A activates (2 draws), then confirm a second call draws once.
	for seed := uint64(0); seed < 10_000; seed++ {
		r := rng.New(seed)
		inj := fault.New(r)
		c0 := r.Calls()
		_ = inj.Maybe("A")
		if r.Calls()-c0 != 2 {
			continue
		}
		c1 := r.Calls()
		_ = inj.Maybe("A")
		if r.Calls()-c1 != 1 {
			t.Fatalf("seed %d: second Maybe drew %d, want 1 (latch)", seed, r.Calls()-c1)
		}
		c2 := r.Calls()
		_ = inj.Maybe("B")
		delta := r.Calls() - c2
		if delta != 1 && delta != 2 {
			t.Fatalf("seed %d: new site drew %d", seed, delta)
		}
		return
	}
	t.Fatal("no seed found for latch check")
}

func TestMaybePIgnoresQuiet(t *testing.T) {
	r := rng.New(11)
	inj := fault.New(r)
	inj.SetQuiet(true)
	before := r.Calls()
	_ = inj.MaybeP("x", 0.5)
	if r.Calls() == before {
		t.Fatal("MaybeP must draw while quiet")
	}
}

func TestMaybePLatchesPerSite(t *testing.T) {
	for seed := uint64(0); seed < 10_000; seed++ {
		r := rng.New(seed)
		inj := fault.New(r)
		c0 := r.Calls()
		_ = inj.MaybeP("A", 0.05)
		if r.Calls()-c0 != 2 {
			continue
		}
		c1 := r.Calls()
		_ = inj.MaybeP("A", 0.05)
		if r.Calls()-c1 != 1 {
			t.Fatalf("seed %d: second MaybeP drew %d, want 1 (latch)", seed, r.Calls()-c1)
		}
		return
	}
	t.Fatal("no seed found for MaybeP latch check")
}
