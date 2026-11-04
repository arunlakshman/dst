// Package sim is the process runner that wires eng engines to rng/sched/trace/assert/fault.
package sim

import (
	"github.com/arunlakshman/dst/assert"
	"github.com/arunlakshman/dst/eng"
	"github.com/arunlakshman/dst/fault"
	"github.com/arunlakshman/dst/rng"
	"github.com/arunlakshman/dst/sched"
)

// ProcessID names a process in one run.
type ProcessID = eng.ActorID

// Process is one actor the runner steps.
//
// Step takes one inbox value and returns zero or more outbound values.
// Outbound values are either a control struct in this package (Send, SetTimer,
// CancelTimer, Crash, Restart) or a domain value an eng.Engine understands.
// There is no separate Event/Action type: those names collide with "simulated
// action" in everyday English.
type Process interface {
	ID() ProcessID
	StateHash() uint64
	Step(cx Context, inbox any) (out []any)
}

// Context is lent to Process.Step.
type Context interface {
	Now() sched.TimeNs
	Rand() *rng.Rand
	Faults() *fault.Injector
	Assert() *assert.Log
}

// Send delivers Body to To at the next turn (delay 0).
type Send struct {
	To   ProcessID
	Body any
}

// SetTimer arms a timer on the issuing process. Body is delivered when it fires.
type SetTimer struct {
	ID    uint64
	After sched.TimeNs
	Body  any
}

// CancelTimer cancels a pending timer on the issuing process.
type CancelTimer struct {
	ID uint64
}

// Crash marks the issuing process crashed. Deliveries to it are ignored until
// Restart (or Spawn) clears the crash. Pending timers still fire; Step is a no-op
// while crashed. Crash does not remove the process from the runner.
type Crash struct{}

// Restart clears Crash for Who. If Boot is non-nil, it is delivered to Who at
// delay 0 after the clear. Any process may return Restart; Who need not be the issuer.
type Restart struct {
	Who  ProcessID
	Boot any
}

// Config configures one Runner.
type Config struct {
	Seed    uint64
	Steps   int // stop after this many Process.Step calls (0 = drain until empty / MaxPops)
	MaxPops int // hard cap on scheduler pops; default 1_000_000
}

// Failure is one assert onFail record collected during Run.
type Failure struct {
	Property string
	Detail   string
}

// Result is the fingerprint and counters of one Run.
type Result struct {
	Hash     uint64
	Steps    int // Process.Step calls
	Pops     int // scheduler pops
	Calls    uint64
	Failures []Failure
}

const defaultMaxPops = 1_000_000
