# dst

Deterministic simulation tools for single-threaded Go tests: a seeded random stream, a virtual-time event heap, an FNV fingerprint, record-and-continue assertions, cooperative fault points, and process/engine contracts on top.

Module: `github.com/arunlakshman/dst`

## Install

```bash
go get github.com/arunlakshman/dst@latest
```

## Mechanism packages

| Package | Role |
|---|---|
| `rng` | SplitMix64 stream (`Uint64`, `Intn`, `Float64`, `Chance`, `Between`, `Calls`, `Bytes`) |
| `sched` | Virtual mono clock and `(at, tie, seq)` event heap |
| `trace` | Running FNV-1a `Hash`, optional `Ring[T]` |
| `assert` | Always / Sometimes / Reachable / Unreachable with coverage |
| `fault` | Maybe / MaybeP / Fault with per-site latch and Quiet |

## Contract packages

| Package | Role |
|---|---|
| `eng` | `Engine` + `Context` (`Deliver`, `Self`). Returns `ErrNotMine` for foreign actions. Does not import `sim`. |
| `sim` | `Process`, `Runner`, control actions (`Send`, `SetTimer`, `CancelTimer`, `Crash`). Owns one `rng`, `sched`, `fault.Injector`, `assert.Log`, and `trace.Hash` per run. |

## How to wire a Process and Engine

1. Implement `sim.Process` (`ID`, `StateHash`, `Step`).
2. Implement `eng.Engine` (`ID`, `StateHash`, `Handle`, `OnSelf`).
3. `r := sim.New(sim.Config{Seed: seed})`, then `r.Register(engine)`, `r.Spawn(process)`, `r.Inject(...)`, `r.Run()`.

`Step` returns control actions or domain actions. The runner handles control types first, then tries engines in register order until one does not return `eng.ErrNotMine`.

A packaged form lives under `examples/kv`: `Store` is an `eng.Engine`, `Node` and `Client` are `sim.Process` values. Same seed must yield the same `Result.Hash` and `Calls`.

## Mechanism quickstart

```go
package main

import (
	"fmt"

	"github.com/arunlakshman/dst/assert"
	"github.com/arunlakshman/dst/fault"
	"github.com/arunlakshman/dst/rng"
	"github.com/arunlakshman/dst/sched"
	"github.com/arunlakshman/dst/trace"
)

func main() {
	r := rng.New(42)
	s := sched.New(r)
	h := trace.New()
	a := assert.New(func(property, detail string) {
		fmt.Printf("FAIL %s: %s\n", property, detail)
	})
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

	put("a", "1")
	put("a", "2")
	for s.Next() {
	}
	h.Uint64(r.Calls())
	fmt.Printf("hash=%016x steps=%d\n", h.Sum(), s.Steps())
}
```

## Stay out (v1)

These stay out of this module on purpose:

- Spec / rapid property engines
- Runtime union or domain event vocabularies
- Campaign runners and corpus orchestration
- Kafka / S3 or other I/O fakes
- Process, Workload, Engine, or network partition simulators

Wall clocks, product fault policy, and domain IDs belong in the importing project.

## License

Apache-2.0. See [LICENSE](LICENSE).
