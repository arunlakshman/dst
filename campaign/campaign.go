// Package campaign runs many independent seeds in parallel and folds the results.
package campaign

import (
	goruntime "runtime"
	"sync"
	"time"

	"github.com/arunlakshman/dst/trace"
)

// Coverage counts evaluations of one property across seeds.
type Coverage struct {
	Kind   string
	Checks uint64
	Passes uint64
}

// Result is the outcome of one seed.
type Result struct {
	Seed      uint64
	Hash      uint64
	OK        bool
	Reproduce string
	Coverage  map[string]Coverage
}

// Counts summarizes how many seeds ran and how many failed.
type Counts struct {
	Seeds  int
	OK     int
	Failed int
}

// Summary is the outcome of many seeds of one scenario.
type Summary struct {
	Failed   []Result
	Hash     uint64
	Coverage map[string]Coverage
	Elapsed  time.Duration
	Counts   Counts
}

// Run executes seeds first, first+1, … for seeds count, using workers goroutines.
// When workers <= 0, GOMAXPROCS is used. Each run is independent; results are
// folded in seed order. Coverage maps are merged by summing Checks and Passes.
func Run(first uint64, seeds int, workers int, run func(seed uint64) Result) Summary {
	if seeds <= 0 {
		return Summary{Coverage: map[string]Coverage{}}
	}
	if workers <= 0 {
		workers = goruntime.GOMAXPROCS(0)
	}
	workers = min(workers, seeds)

	start := time.Now()
	results := make([]Result, seeds)
	var wg sync.WaitGroup
	jobs := make(chan int)
	for range workers {
		wg.Go(func() {
			for i := range jobs {
				seed := first + uint64(i)
				r := run(seed)
				r.Seed = seed
				results[i] = r
			}
		})
	}
	for i := range seeds {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	out := Summary{
		Coverage: map[string]Coverage{},
		Counts:   Counts{Seeds: seeds},
	}
	h := trace.New()
	for _, r := range results {
		h.Uint64(r.Hash)
		addCoverage(out.Coverage, r.Coverage)
		if r.OK {
			out.Counts.OK++
			continue
		}
		out.Failed = append(out.Failed, r)
		out.Counts.Failed++
	}
	out.Hash = h.Sum()
	out.Elapsed = time.Since(start)
	return out
}

func addCoverage(into, from map[string]Coverage) {
	for k, c := range from {
		t := into[k]
		t.Kind = c.Kind
		t.Checks += c.Checks
		t.Passes += c.Passes
		into[k] = t
	}
}
