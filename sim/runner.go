package sim

import (
	"errors"
	"fmt"

	"github.com/arunlakshman/dst/assert"
	"github.com/arunlakshman/dst/eng"
	"github.com/arunlakshman/dst/fault"
	"github.com/arunlakshman/dst/rng"
	"github.com/arunlakshman/dst/sched"
	"github.com/arunlakshman/dst/trace"
)

// ErrProtocol means an action matched no control type and no registered engine.
var ErrProtocol = errors.New("sim: unhandled action")

// Runner owns one rng, sched, fault.Injector, assert.Log, and trace.Hash for a run.
type Runner struct {
	cfg      Config
	rand     *rng.Rand
	sched    *sched.Scheduler
	faults   *fault.Injector
	asserts  *assert.Log
	hash     trace.Hash
	engines  []eng.Engine
	procs    map[ProcessID]Process
	crashed  map[ProcessID]bool
	timers   map[timerKey]uint64
	failures []Failure
	steps    int
	events   int
	protoErr error
}

type timerKey struct {
	pid ProcessID
	id  uint64
}

// New builds a Runner for cfg. MaxEvents defaults to 1_000_000 when zero.
func New(cfg Config) *Runner {
	if cfg.MaxEvents <= 0 {
		cfg.MaxEvents = defaultMaxEvents
	}
	r := &Runner{
		cfg:     cfg,
		rand:    rng.New(cfg.Seed),
		procs:   map[ProcessID]Process{},
		crashed: map[ProcessID]bool{},
		timers:  map[timerKey]uint64{},
		hash:    trace.New(),
	}
	r.sched = sched.New(r.rand)
	r.faults = fault.New(r.rand)
	r.asserts = assert.New(func(property, detail string) {
		r.failures = append(r.failures, Failure{Property: property, Detail: detail})
	})
	return r
}

// Register adds an engine. Dispatch tries engines in registration order.
func (r *Runner) Register(e eng.Engine) {
	r.engines = append(r.engines, e)
}

// Spawn adds or replaces a process and clears a prior crash for that ID.
func (r *Runner) Spawn(p Process) {
	id := p.ID()
	r.procs[id] = p
	delete(r.crashed, id)
}

// Inject schedules a workload event for process to after delay.
func (r *Runner) Inject(to ProcessID, after sched.TimeNs, ev Event) {
	r.deliver(to, after, ev)
}

// Run drains the scheduler under cfg limits and returns the run fingerprint.
func (r *Runner) Run() (Result, error) {
	for r.sched.Len() > 0 && r.events < r.cfg.MaxEvents {
		if r.cfg.Steps > 0 && r.steps >= r.cfg.Steps {
			break
		}
		if !r.sched.Next() {
			break
		}
		r.events++
	}
	for _, e := range r.engines {
		r.hash.Uint64(e.StateHash())
	}
	res := Result{
		Hash:     r.hash.Sum(),
		Steps:    r.steps,
		Events:   r.events,
		Calls:    r.rand.Calls(),
		Failures: append([]Failure(nil), r.failures...),
	}
	return res, r.protoErr
}

func (r *Runner) deliver(to ProcessID, after sched.TimeNs, ev Event) {
	r.sched.After(after, "deliver:"+string(to), func() {
		r.handleProcessEvent(to, ev)
	})
}

func (r *Runner) handleProcessEvent(to ProcessID, ev Event) {
	if r.crashed[to] {
		return
	}
	p, ok := r.procs[to]
	if !ok {
		r.asserts.Unreachable("sim: unknown process", assert.Pair{Key: "to", Value: string(to)})
		return
	}
	cx := procContext{r: r}
	acts := p.Step(cx, ev)
	r.steps++
	r.hash.Int64(int64(r.sched.Mono()))
	r.hash.String(string(to))
	r.hash.Uint64(p.StateHash())
	r.hash.Uint64(uint64(len(acts)))
	for _, act := range acts {
		r.route(to, act)
	}
}

func (r *Runner) route(from ProcessID, act Action) {
	switch a := act.(type) {
	case Send:
		r.deliver(a.To, 0, a.Body)
		return
	case SetTimer:
		r.setTimer(from, a)
		return
	case CancelTimer:
		r.cancelTimer(from, a.ID)
		return
	case Crash:
		r.crashed[from] = true
		return
	}

	ecx := engContext{r: r, from: eng.ActorID(from), engine: nil}
	for _, e := range r.engines {
		ecx.engine = e
		err := e.Handle(ecx, act)
		if errors.Is(err, eng.ErrNotMine) {
			continue
		}
		if err != nil {
			r.asserts.Always(false, "sim: engine handle", assert.Pair{Key: "err", Value: err.Error()})
			r.protoErr = errors.Join(r.protoErr, err)
		}
		return
	}
	r.asserts.Unreachable("sim: unhandled action", assert.Pair{Key: "type", Value: fmt.Sprintf("%T", act)})
	r.protoErr = errors.Join(r.protoErr, fmt.Errorf("%w: %T", ErrProtocol, act))
}

func (r *Runner) setTimer(pid ProcessID, t SetTimer) {
	key := timerKey{pid: pid, id: t.ID}
	r.timers[key]++
	gen := r.timers[key]
	body := t.Body
	r.sched.After(t.After, "timer", func() {
		if r.timers[key] != gen {
			return
		}
		delete(r.timers, key)
		r.handleProcessEvent(pid, body)
	})
}

func (r *Runner) cancelTimer(pid ProcessID, id uint64) {
	key := timerKey{pid: pid, id: id}
	r.timers[key]++
}

type procContext struct{ r *Runner }

func (c procContext) Now() sched.TimeNs       { return c.r.sched.Mono() }
func (c procContext) Rand() *rng.Rand         { return c.r.rand }
func (c procContext) Faults() *fault.Injector { return c.r.faults }
func (c procContext) Assert() *assert.Log     { return c.r.asserts }

type engContext struct {
	r      *Runner
	from   eng.ActorID
	engine eng.Engine
}

func (c engContext) Now() sched.TimeNs       { return c.r.sched.Mono() }
func (c engContext) Rand() *rng.Rand         { return c.r.rand }
func (c engContext) Faults() *fault.Injector { return c.r.faults }
func (c engContext) Assert() *assert.Log     { return c.r.asserts }
func (c engContext) From() eng.ActorID       { return c.from }

func (c engContext) Deliver(to eng.ActorID, after sched.TimeNs, event any) {
	c.r.deliver(ProcessID(to), after, event)
}

func (c engContext) Self(after sched.TimeNs, body any) {
	e := c.engine
	c.r.sched.After(after, "self:"+string(e.ID()), func() {
		ecx := engContext{r: c.r, from: c.from, engine: e}
		e.OnSelf(ecx, body)
	})
}
