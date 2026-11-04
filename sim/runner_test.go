package sim_test

import (
	"errors"
	"testing"

	"github.com/arunlakshman/dst/eng"
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

func (p *outProc) ID() sim.ProcessID                 { return p.id }
func (p *outProc) StateHash() uint64                 { return 0 }
func (p *outProc) Step(sim.Context, any) []any       { return p.out }

var _ eng.Engine = (*noopEngine)(nil)

type noopEngine struct{}

func (noopEngine) ID() eng.ID                            { return "noop" }
func (noopEngine) StateHash() uint64                     { return 0 }
func (noopEngine) Handle(eng.Context, any) error         { return eng.ErrNotMine }
func (noopEngine) OnSelf(eng.Context, any)               {}
