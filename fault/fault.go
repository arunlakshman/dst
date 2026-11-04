package fault

import "github.com/manifesterio/dst/rng"

// Site is a stable cooperative fault point id. Do not interpolate values into it.
type Site string

// Injector is FoundationDB-style cooperative injection against one Rand stream.
// Invariant: Quiet suppresses Fault only. Maybe and MaybeP still draw.
// Invariant: Maybe latches per-site activation once per Injector lifetime (one run).
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

// Maybe activates the site for the run with probability 1/2 on first sight,
// then fires with probability 0.05 on each call when active.
func (i *Injector) Maybe(site Site) bool {
	on, known := i.seen[string(site)]
	if !known {
		on = i.rand.Chance(0.5)
		i.seen[string(site)] = on
	}
	return on && i.rand.Chance(0.05)
}

// MaybeP fires with probability p. The site id is unused.
func (i *Injector) MaybeP(_ Site, p float64) bool { return i.rand.Chance(p) }

// Fault fires with probability p. It never fires while Quiet is set.
func (i *Injector) Fault(_ Site, p float64) bool {
	return !i.quiet && i.rand.Chance(p)
}
