// Package kv is a tiny Store engine + Node/Client processes on sim/eng.
package kv

import (
	"maps"
	"slices"

	"github.com/arunlakshman/dst/assert"
	"github.com/arunlakshman/dst/eng"
	"github.com/arunlakshman/dst/sched"
	"github.com/arunlakshman/dst/sim"
	"github.com/arunlakshman/dst/trace"
)

// Result is the fingerprint of one toy KV run.
type Result struct {
	Hash  uint64
	Calls uint64
	Steps int
}

// Run executes a small map store under one seeded sim.Runner.
func Run(seed uint64) Result {
	r := sim.New(sim.Config{Seed: seed})
	store := newStore("kv")
	node := newNode("n1")
	client := newClient("c1", "n1")
	r.Register(store)
	r.Spawn(node)
	r.Spawn(client)
	r.Inject("c1", 0, start{})
	res, _ := r.Run()
	return Result{Hash: res.Hash, Calls: res.Calls, Steps: res.Steps}
}

type start struct{}
type doPut2 struct{}
type doGet struct{}

type putMsg struct {
	Key, Value string
}

type getMsg struct {
	Key string
}

type getReply struct {
	Key, Value string
}

type putAct struct {
	Key, Value string
}

type getAct struct {
	Key     string
	ReplyTo eng.ActorID
}

type store struct {
	id   eng.ID
	data map[string]string
}

func newStore(id eng.ID) *store {
	return &store{id: id, data: map[string]string{}}
}

func (s *store) ID() eng.ID              { return s.id }
func (s *store) OnSelf(eng.Context, any) {}

func (s *store) StateHash() uint64 {
	h := trace.New()
	for _, k := range slices.Sorted(maps.Keys(s.data)) {
		h.String(k)
		h.String(s.data[k])
	}
	return h.Sum()
}

func (s *store) Handle(cx eng.Context, act any) error {
	switch a := act.(type) {
	case putAct:
		if cx.Faults().Fault("kv.write", 0.1) {
			return nil
		}
		s.data[a.Key] = a.Value
		cx.Assert().Always(s.data[a.Key] == a.Value, "put sticks", assert.Pair{Key: "k", Value: a.Key})
		return nil
	case getAct:
		delay := sched.TimeNs(cx.Rand().Between(1, 5) * 1_000_000)
		cx.Deliver(a.ReplyTo, delay, getReply{Key: a.Key, Value: s.data[a.Key]})
		return nil
	default:
		return eng.ErrNotMine
	}
}

type node struct {
	id sim.ProcessID
}

func newNode(id sim.ProcessID) *node { return &node{id: id} }

func (n *node) ID() sim.ProcessID { return n.id }
func (n *node) StateHash() uint64 { return 0 }

func (n *node) Step(cx sim.Context, ev sim.Event) []sim.Action {
	switch e := ev.(type) {
	case putMsg:
		return []sim.Action{putAct{Key: e.Key, Value: e.Value}}
	case getMsg:
		return []sim.Action{getAct{Key: e.Key, ReplyTo: eng.ActorID(n.id)}}
	case getReply:
		cx.Assert().Always(e.Value == "2", "read matches", assert.Pair{Key: "k", Value: e.Key})
		return nil
	default:
		return nil
	}
}

type client struct {
	id, node sim.ProcessID
}

func newClient(id, nodeID sim.ProcessID) *client {
	return &client{id: id, node: nodeID}
}

func (c *client) ID() sim.ProcessID { return c.id }
func (c *client) StateHash() uint64 { return 0 }

func (c *client) Step(cx sim.Context, ev sim.Event) []sim.Action {
	gap := sched.TimeNs(cx.Rand().Between(1, 20) * 1_000_000)
	switch ev.(type) {
	case start:
		return []sim.Action{
			sim.Send{To: c.node, Body: putMsg{Key: "a", Value: "1"}},
			sim.SetTimer{ID: 1, After: gap, Body: doPut2{}},
		}
	case doPut2:
		return []sim.Action{
			sim.Send{To: c.node, Body: putMsg{Key: "a", Value: "2"}},
			sim.SetTimer{ID: 2, After: gap, Body: doGet{}},
		}
	case doGet:
		return []sim.Action{
			sim.Send{To: c.node, Body: getMsg{Key: "a"}},
		}
	default:
		return nil
	}
}
