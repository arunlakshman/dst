package campaign_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/arunlakshman/dst/campaign"
	"github.com/arunlakshman/dst/trace"
)

func TestRunFoldsHashesInSeedOrder(t *testing.T) {
	const first, seeds = 10, 5
	want := trace.New()
	for i := range seeds {
		seed := first + uint64(i)
		want.Uint64(seed * 1000)
	}

	got := campaign.Run(first, seeds, 4, func(seed uint64) campaign.Result {
		// Finish later seeds first so a buggy fold would scramble order.
		time.Sleep(time.Duration(seeds-int(seed-first)) * time.Millisecond)
		return campaign.Result{Hash: seed * 1000, OK: true}
	})

	if got.Hash != want.Sum() {
		t.Fatalf("fold order: got %#x want %#x", got.Hash, want.Sum())
	}
	if got.Counts != (campaign.Counts{Seeds: seeds, OK: seeds}) {
		t.Fatalf("counts: %+v", got.Counts)
	}
	if len(got.Failed) != 0 {
		t.Fatalf("failed: %+v", got.Failed)
	}
}

func TestRunCollectsFailuresAndMergesCoverage(t *testing.T) {
	got := campaign.Run(1, 4, 2, func(seed uint64) campaign.Result {
		ok := seed%2 == 0
		return campaign.Result{
			Hash: seed,
			OK:   ok,
			Coverage: map[string]campaign.Coverage{
				"path": {Kind: "reachable", Checks: 1, Passes: uint64(seed)},
			},
			Reproduce: "repro",
		}
	})

	if got.Counts.Failed != 2 || got.Counts.OK != 2 {
		t.Fatalf("counts: %+v", got.Counts)
	}
	if len(got.Failed) != 2 {
		t.Fatalf("failed len=%d", len(got.Failed))
	}
	if got.Failed[0].Seed != 1 || got.Failed[1].Seed != 3 {
		t.Fatalf("failed seed order: %+v", got.Failed)
	}
	c := got.Coverage["path"]
	if c.Kind != "reachable" || c.Checks != 4 || c.Passes != 1+2+3+4 {
		t.Fatalf("coverage: %+v", c)
	}
}

func TestRunZeroSeeds(t *testing.T) {
	var calls atomic.Int32
	got := campaign.Run(1, 0, 2, func(uint64) campaign.Result {
		calls.Add(1)
		return campaign.Result{OK: true}
	})
	if calls.Load() != 0 {
		t.Fatalf("ran %d times", calls.Load())
	}
	if got.Counts.Seeds != 0 || got.Hash != 0 {
		t.Fatalf("summary: %+v", got)
	}
}

func TestRunStampsSeedOnFailedResults(t *testing.T) {
	got := campaign.Run(7, 3, 1, func(seed uint64) campaign.Result {
		return campaign.Result{Hash: seed, OK: seed != 8}
	})
	if len(got.Failed) != 1 || got.Failed[0].Seed != 8 {
		t.Fatalf("seed stamp: %+v", got.Failed)
	}
}
