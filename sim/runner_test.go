package sim_test

import (
	"errors"
	"testing"

	"github.com/arunlakshman/dst/eng"
	"github.com/arunlakshman/dst/sched"
	"github.com/arunlakshman/dst/sim"
)

func TestSameSeedSameHash(t *testing.T) {
	run := func(seed uint64) uint64 {
		r := sim.New(sim.Config{Seed: seed})
		r.Spawn(&echoProc{id: "p"})
		r.Inject("p", 0, "ping")
		res, err := r.Run()
		if err != nil {
			t.Fatal(err)
		}
		return res.Hash
	}
	if run(7) != run(7) {
		t.Fatal("hash drift")
	}
}

type echoProc struct{ id sim.ProcessID }

func (p *echoProc) ID() sim.ProcessID { return p.id }
func (p *echoProc) StateHash() uint64 { return 1 }
func (p *echoProc) Step(_ sim.Context, inbox any) []any {
	if inbox == "ping" {
		return []any{sim.Send{To: p.id, Body: "pong"}}
	}
	return nil
}

func TestUnhandledOutbound(t *testing.T) {
	r := sim.New(sim.Config{Seed: 1})
	r.Spawn(&outProc{id: "p", out: []any{struct{ X int }{1}}})
	r.Inject("p", 0, true)
	_, err := r.Run()
	if !errors.Is(err, sim.ErrProtocol) {
		t.Fatalf("got %v, want ErrProtocol", err)
	}
}

type outProc struct {
	id  sim.ProcessID
	out []any
}

func (p *outProc) ID() sim.ProcessID           { return p.id }
func (p *outProc) StateHash() uint64           { return 0 }
func (p *outProc) Step(sim.Context, any) []any { return p.out }

func TestCrashAndRestart(t *testing.T) {
	r := sim.New(sim.Config{Seed: 3})
	n := &countProc{id: "n"}
	sup := &outProc{id: "sup", out: nil}
	r.Spawn(n)
	r.Spawn(sup)

	// Crash the node via a kill message, then restart with a boot inbox.
	r.Spawn(&outProc{id: "killer", out: []any{sim.Send{To: "n", Body: "kill"}}})
	r.Inject("killer", 0, true)

	// After kill is processed, inject restart from supervisor.
	r.Inject("sup", sched.TimeNs(1), true)
	sup.out = []any{sim.Restart{Who: "n", Boot: "boot"}}

	res, err := r.Run()
	if err != nil {
		t.Fatal(err)
	}
	if n.kills != 1 {
		t.Fatalf("kills=%d want 1", n.kills)
	}
	if n.boots != 1 {
		t.Fatalf("boots=%d want 1", n.boots)
	}
	if res.Steps == 0 {
		t.Fatal("no steps")
	}
}

type countProc struct {
	id           sim.ProcessID
	kills, boots int
}

func (p *countProc) ID() sim.ProcessID { return p.id }
func (p *countProc) StateHash() uint64 { return uint64(p.kills + p.boots) }
func (p *countProc) Step(_ sim.Context, inbox any) []any {
	switch inbox {
	case "kill":
		p.kills++
		return []any{sim.Crash{}}
	case "boot":
		p.boots++
		return nil
	default:
		return nil
	}
}

func TestErrNotMineRouting(t *testing.T) {
	r := sim.New(sim.Config{Seed: 5})
	first := &skipEngine{id: "skip"}
	second := &takeEngine{id: "take"}
	r.Register(first)
	r.Register(second)
	r.Spawn(&outProc{id: "p", out: []any{takeAct{}}})
	r.Inject("p", 0, true)
	_, err := r.Run()
	if err != nil {
		t.Fatal(err)
	}
	if !first.seen {
		t.Fatal("first engine was not tried")
	}
	if !second.hit {
		t.Fatal("second engine did not handle")
	}
}

type takeAct struct{}

type skipEngine struct {
	id   eng.ID
	seen bool
}

func (e *skipEngine) ID() eng.ID        { return e.id }
func (e *skipEngine) StateHash() uint64 { return 0 }
func (e *skipEngine) Handle(_ eng.Context, _ any) error {
	e.seen = true
	return eng.ErrNotMine
}
func (e *skipEngine) OnSelf(eng.Context, any) {}

type takeEngine struct {
	id  eng.ID
	hit bool
}

func (e *takeEngine) ID() eng.ID        { return e.id }
func (e *takeEngine) StateHash() uint64 { return 0 }
func (e *takeEngine) Handle(_ eng.Context, act any) error {
	if _, ok := act.(takeAct); ok {
		e.hit = true
		return nil
	}
	return eng.ErrNotMine
}
func (e *takeEngine) OnSelf(eng.Context, any) {}

func TestEngineSelfAndDeliver(t *testing.T) {
	r := sim.New(sim.Config{Seed: 9})
	e := &relayEngine{id: "relay", to: "p"}
	p := &inboxProc{id: "p"}
	r.Register(e)
	r.Spawn(p)
	r.Spawn(&outProc{id: "c", out: []any{relayAct{}}})
	r.Inject("c", 0, true)
	_, err := r.Run()
	if err != nil {
		t.Fatal(err)
	}
	if !e.selfHit {
		t.Fatal("OnSelf not called")
	}
	if !p.got {
		t.Fatal("Deliver did not reach process")
	}
}

type relayAct struct{}

type relayEngine struct {
	id      eng.ID
	to      eng.ActorID
	selfHit bool
}

func (e *relayEngine) ID() eng.ID        { return e.id }
func (e *relayEngine) StateHash() uint64 { return 0 }
func (e *relayEngine) Handle(cx eng.Context, act any) error {
	if _, ok := act.(relayAct); !ok {
		return eng.ErrNotMine
	}
	cx.Self(0, "wake")
	return nil
}
func (e *relayEngine) OnSelf(cx eng.Context, body any) {
	e.selfHit = body == "wake"
	cx.Deliver(e.to, 0, "hello")
}

type inboxProc struct {
	id  sim.ProcessID
	got bool
}

func (p *inboxProc) ID() sim.ProcessID { return p.id }
func (p *inboxProc) StateHash() uint64 { return 0 }
func (p *inboxProc) Step(_ sim.Context, inbox any) []any {
	if inbox == "hello" {
		p.got = true
	}
	return nil
}
