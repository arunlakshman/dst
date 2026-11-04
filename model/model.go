// Package model provides rapid-oriented lockstep helpers without importing
// a property-testing engine. Callers pass ops from their own generator
// (for example *rapid.T) or drive a fixed alphabet in a unit test.
//
// Prefer the simplest ops first when shrinking or when choosing an ordered
// alphabet. Put reads and no-ops ahead of writes, and crash/reboot ops last,
// so failures land on short prefixes. The caller supplies that order; this
// package only checks each step.
package model

import "fmt"

// Step runs a reference model and a subject in lockstep for one op.
// Equal compares model observation (want) to subject observation (got).
// Invariant is optional; nil skips it.
type Step[Op, Obs any] struct {
	ModelApply   func(Op) (Obs, error)
	SubjectApply func(Op) (Obs, error)
	Equal        func(want, got Obs) error
	Invariant    func() error
}

// Check applies op to model then subject and compares observations.
// Model errors fail the step. Subject errors fail unless Equal accepts them
// through a custom Equal; use CheckAllowFailed when a crashed subject may
// diverge while the model succeeded.
func (s Step[Op, Obs]) Check(op Op) error {
	want, err := s.ModelApply(op)
	if err != nil {
		return fmt.Errorf("model: %w", err)
	}
	got, err := s.SubjectApply(op)
	if err != nil {
		return fmt.Errorf("subject: %w", err)
	}
	if err := s.Equal(want, got); err != nil {
		return fmt.Errorf("equal: %w", err)
	}
	if s.Invariant != nil {
		if err := s.Invariant(); err != nil {
			return fmt.Errorf("invariant: %w", err)
		}
	}
	return nil
}

// CheckAllowFailed is like Check, but when failed is true a subject error is
// allowed while the model succeeded. Observations are still compared when
// both applies succeed. When failed is false, behavior matches Check.
func (s Step[Op, Obs]) CheckAllowFailed(op Op, failed bool) error {
	want, merr := s.ModelApply(op)
	got, serr := s.SubjectApply(op)
	if merr != nil {
		return fmt.Errorf("model: %w", merr)
	}
	if serr != nil {
		if failed {
			if s.Invariant != nil {
				if err := s.Invariant(); err != nil {
					return fmt.Errorf("invariant: %w", err)
				}
			}
			return nil
		}
		return fmt.Errorf("subject: %w", serr)
	}
	if err := s.Equal(want, got); err != nil {
		return fmt.Errorf("equal: %w", err)
	}
	if s.Invariant != nil {
		if err := s.Invariant(); err != nil {
			return fmt.Errorf("invariant: %w", err)
		}
	}
	return nil
}

// Write is one answered key/value pair used by CheckPersistence.
type Write[K, V any] struct {
	Key   K
	Value V
}

// CheckPersistence verifies that every answered write is still present after
// a subject crash and recover. present(key, value) reports whether durable
// state contains that write. answered is the set of writes the client
// observed as successful before the crash.
func CheckPersistence[K, V any](answered []Write[K, V], present func(K, V) bool) error {
	for i, w := range answered {
		if !present(w.Key, w.Value) {
			return fmt.Errorf("persistence: answered write %d missing", i)
		}
	}
	return nil
}

// CheckForwardProgress compares a clean model observation to the subject's
// observation after a clean restart. Both callbacks produce observations;
// equal reports mismatch.
func CheckForwardProgress[Obs any](modelClean, subjectRestart func() (Obs, error), equal func(want, got Obs) error) error {
	want, err := modelClean()
	if err != nil {
		return fmt.Errorf("model clean: %w", err)
	}
	got, err := subjectRestart()
	if err != nil {
		return fmt.Errorf("subject restart: %w", err)
	}
	if err := equal(want, got); err != nil {
		return fmt.Errorf("forward progress: %w", err)
	}
	return nil
}
