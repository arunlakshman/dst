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

// Event is an opaque process inbox value.
type Event = any

// Action is an opaque value from Process.Step: a control action or an engine action.
type Action = any

// Process is one actor the runner steps.
type Process interface {
	ID() ProcessID
	StateHash() uint64
	Step(cx Context, ev Event) []Action
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
	Body Event
}

// SetTimer arms a timer on the issuing process. Body is delivered when it fires.
type SetTimer struct {
	ID    uint64
	After sched.TimeNs
	Body  Event
}

// CancelTimer cancels a pending timer on the issuing process.
type CancelTimer struct {
	ID uint64
}

// Crash drops the issuing process. Deliveries to it are ignored until Spawn again.
type Crash struct{}

// Config configures one Runner.
type Config struct {
	Seed      uint64
	Steps     int // stop after this many delivered process steps (0 = drain until empty / MaxEvents)
	MaxEvents int // hard cap on scheduler pops; default 1_000_000
}

// Failure is one assert onFail record collected during Run.
type Failure struct {
	Property string
	Detail   string
}

// Result is the fingerprint and counters of one Run.
type Result struct {
	Hash     uint64
	Steps    int
	Events   int
	Calls    uint64
	Failures []Failure
}

const defaultMaxEvents = 1_000_000
