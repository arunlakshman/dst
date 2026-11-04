package sim_test

import (
	"errors"
	"testing"

	"github.com/arunlakshman/dst/eng"
	"github.com/arunlakshman/dst/sim"
)

type nopEngine struct{ id eng.ID }

func (e nopEngine) ID() eng.ID                    { return e.id }
func (e nopEngine) StateHash() uint64             { return 1 }
func (e nopEngine) Handle(eng.Context, any) error { return eng.ErrNotMine }
func (e nopEngine) OnSelf(eng.Context, any)       {}

type echoProc struct {
	id   sim.ProcessID
	hash uint64
}

func (p *echoProc) ID() sim.ProcessID { return p.id }
func (p *echoProc) StateHash() uint64 { return p.hash }

func (p *echoProc) Step(_ sim.Context, ev sim.Event) []sim.Action {
	switch ev.(type) {
	case string:
		p.hash++
		return nil
	default:
		return nil
	}
}

func TestSameSeedSameHash(t *testing.T) {
	run := func(seed uint64) sim.Result {
		r := sim.New(sim.Config{Seed: seed})
		r.Register(nopEngine{id: "e"})
		r.Spawn(&echoProc{id: "p"})
		r.Inject("p", 0, "hi")
		res, err := r.Run()
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	a, b := run(7), run(7)
	if a.Hash != b.Hash || a.Calls != b.Calls || a.Steps != b.Steps {
		t.Fatalf("a=%+v b=%+v", a, b)
	}
	if a.Steps == 0 {
		t.Fatal("no steps")
	}
}

func TestUnhandledAction(t *testing.T) {
	r := sim.New(sim.Config{Seed: 1})
	r.Spawn(&actionProc{id: "p", acts: []sim.Action{struct{ X int }{1}}})
	r.Inject("p", 0, "go")
	res, err := r.Run()
	if !errors.Is(err, sim.ErrProtocol) {
		t.Fatalf("err=%v", err)
	}
	if len(res.Failures) == 0 {
		t.Fatal("expected failure record")
	}
}

type actionProc struct {
	id   sim.ProcessID
	acts []sim.Action
}

func (p *actionProc) ID() sim.ProcessID                        { return p.id }
func (p *actionProc) StateHash() uint64                        { return 0 }
func (p *actionProc) Step(sim.Context, sim.Event) []sim.Action { return p.acts }
