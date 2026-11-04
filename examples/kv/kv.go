package kv

import (
	"github.com/manifesterio/dst/assert"
	"github.com/manifesterio/dst/fault"
	"github.com/manifesterio/dst/rng"
	"github.com/manifesterio/dst/sched"
	"github.com/manifesterio/dst/trace"
)

// Result is the fingerprint of one toy KV run.
type Result struct {
	Hash  uint64
	Calls uint64
	Steps int
}

// Run executes a small map store under one seeded stream.
func Run(seed uint64) Result {
	r := rng.New(seed)
	s := sched.New(r)
	h := trace.New()
	a := assert.New(nil)
	f := fault.New(r)

	kv := map[string]string{}

	put := func(k, v string) {
		s.After(sched.TimeNs(r.Between(1, 5)*1_000_000), "put", func() {
			if f.Fault("kv.write", 0.1) {
				return
			}
			kv[k] = v
			h.String(k)
			h.String(v)
			a.Always(kv[k] == v, "put sticks", assert.Pair{Key: "k", Value: k})
		})
	}

	get := func(k, want string) {
		s.After(sched.TimeNs(r.Between(1, 5)*1_000_000), "get", func() {
			got := kv[k]
			a.Always(got == want, "read matches", assert.Pair{Key: "k", Value: k})
			h.String(k)
			h.String(got)
		})
	}

	put("a", "1")
	put("a", "2")
	get("a", "2")

	for s.Next() {
	}
	h.Uint64(r.Calls())
	return Result{Hash: h.Sum(), Calls: r.Calls(), Steps: s.Steps()}
}
