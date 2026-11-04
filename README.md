# dst

Deterministic simulation tools for single-threaded Go tests. A product implements `sim.Process` and `eng.Engine`; the runner owns one seeded `rng`, virtual-time `sched`, cooperative `fault` injector, record-and-continue `assert` log, and `trace` fingerprint per run.

Module: `github.com/arunlakshman/dst`

## Install

```bash
go get github.com/arunlakshman/dst@latest
```

## Contract packages

Wire these first.

| Package | Role |
|---|---|
| `eng` | `Engine` + `Context` (`Deliver`, `Self`). Returns `ErrNotMine` for foreign actions. Does not import `sim`. |
| `sim` | `Process`, `Runner`, control values (`Send`, `SetTimer`, `CancelTimer`, `Crash`, `Restart`). Owns one `rng`, `sched`, `fault.Injector`, `assert.Log`, and `trace.Hash` per run. |

### How to wire a Process and Engine

1. Implement `sim.Process` (`ID`, `StateHash`, `Step`).
2. Implement `eng.Engine` (`ID`, `StateHash`, `Handle`, `OnSelf`).
3. `r := sim.New(sim.Config{Seed: seed})`, then `r.Register(engine)`, `r.Spawn(process)`, `r.Inject(...)`, `r.Run()`.

`Process.Step(cx, inbox any) []any` takes one inbox value and returns outbound values. Those are either control structs in package `sim` or domain values an `eng.Engine` understands. There is no `sim.Event` / `sim.Action` alias.

The runner handles control types first, then tries engines in register order until one does not return `eng.ErrNotMine`.

**Crash / Restart.** `Crash` marks the issuing process crashed; deliveries to it are ignored until `Restart` (or `Spawn`) clears the flag. `Restart{Who, Boot}` clears crash for `Who` and, when `Boot` is non-nil, delivers that inbox at delay 0. Pending timers still fire while crashed; `Step` is a no-op until restart.

`examples/kv` is a happy-path store. `examples/kvchaos` adds write drops (`Fault` + MaybeApplied client retries) and Crash+Restart on some seeds.

## Mechanism packages

| Package | Role |
|---|---|
| `rng` | SplitMix64 stream (`Uint64`, `Intn`, `Float64`, `Chance`, `Between`, `Calls`, `Bytes`) |
| `sched` | Virtual mono clock and `(at, tie, seq)` event heap |
| `trace` | Running FNV-1a `Hash`, optional `Ring[T]` |
| `assert` | Always / Sometimes / Reachable / Unreachable with coverage |
| `fault` | Maybe / MaybeP / Fault with per-site latch and Quiet |

## How to run campaigns

A campaign is a seed loop in an ordinary Go test. Same seed must replay to the same `Result.Hash` / `Calls` / `Steps`. Fail the test on `Run` errors (protocol) or non-empty `Failures` (asserts). Cross-seed hash collisions are normal and not signal.

```bash
go test ./examples/kv -run TestCampaign -count=1
go test ./examples/kvchaos -run TestCampaign -count=1
go test ./...
```

## Stay out

These stay out of this module on purpose:

- Spec / rapid property engines
- Closed domain message vocabularies (Kafka, S3, product RPCs)
- Campaign runners and corpus orchestration
- Product I/O fakes beyond the tiny KV examples
- Full network-partition simulators

Wall clocks, product fault policy, and domain IDs belong in the importing project.

## License

Apache-2.0. See [LICENSE](LICENSE).
