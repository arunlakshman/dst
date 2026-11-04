// Package fault is cooperative fault injection drawn from one rng stream.
package fault

import "github.com/manifesterio/dst/rng"

// Site is a stable cooperative fault point id. Do not interpolate values into it.
type Site string

// Default rates for Maybe activation and fire.
const (
	MaybeActivateRate = 0.5
	MaybeFireRate     = 0.05
)

// Injector is cooperative fault injection against one Rand stream.
// Invariant: Quiet suppresses Fault only. Maybe and MaybeP still draw.
// Invariant: Maybe and MaybeP latch per-site activation once per Injector lifetime (one run).
type Injector struct {
	rand  *rng.Rand
	quiet bool
	seen  map[string]bool
}

// New returns an injector that draws from r.
func New(r *rng.Rand) *Injector {
	return &Injector{rand: r, seen: map[string]bool{}}
}

// SetQuiet enables or disables the quiet tail.
func (i *Injector) SetQuiet(q bool) { i.quiet = q }

// Quiet reports whether Fault is suppressed.
func (i *Injector) Quiet() bool { return i.quiet }

// Maybe activates the site for the run with probability MaybeActivateRate on first
// sight, then fires with probability MaybeFireRate on each call when active.
func (i *Injector) Maybe(site Site) bool {
	return i.maybe(site, MaybeFireRate)
}

// MaybeP activates the site like Maybe, then fires with probability p when active.
func (i *Injector) MaybeP(site Site, p float64) bool {
	return i.maybe(site, p)
}

func (i *Injector) maybe(site Site, fire float64) bool {
	on, known := i.seen[string(site)]
	if !known {
		on = i.rand.Chance(MaybeActivateRate)
		i.seen[string(site)] = on
	}
	return on && i.rand.Chance(fire)
}

// Fault fires with probability p. It never fires while Quiet is set.
func (i *Injector) Fault(_ Site, p float64) bool {
	return !i.quiet && i.rand.Chance(p)
}
