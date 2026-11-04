package model_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/arunlakshman/dst/model"
)

type counter struct {
	n int
}

func (c *counter) apply(op string) (int, error) {
	switch op {
	case "inc":
		c.n++
		return c.n, nil
	case "get":
		return c.n, nil
	case "boom":
		return c.n, errors.New("boom")
	default:
		return 0, fmt.Errorf("unknown op %q", op)
	}
}

func TestCheckLockstepCounter(t *testing.T) {
	m, s := &counter{}, &counter{}
	step := model.Step[string, int]{
		ModelApply:   m.apply,
		SubjectApply: s.apply,
		Equal: func(want, got int) error {
			if want != got {
				return fmt.Errorf("want %d got %d", want, got)
			}
			return nil
		},
		Invariant: func() error {
			if m.n != s.n {
				return fmt.Errorf("diverged model=%d subject=%d", m.n, s.n)
			}
			return nil
		},
	}
	for _, op := range []string{"get", "inc", "inc", "get"} {
		if err := step.Check(op); err != nil {
			t.Fatalf("%s: %v", op, err)
		}
	}
	if m.n != 2 || s.n != 2 {
		t.Fatalf("n model=%d subject=%d", m.n, s.n)
	}
}

func TestCheckDetectsMismatch(t *testing.T) {
	m, s := &counter{}, &counter{n: 1}
	step := model.Step[string, int]{
		ModelApply:   m.apply,
		SubjectApply: s.apply,
		Equal: func(want, got int) error {
			if want != got {
				return fmt.Errorf("want %d got %d", want, got)
			}
			return nil
		},
	}
	if err := step.Check("get"); err == nil {
		t.Fatal("want mismatch error")
	}
}

func TestCheckAllowFailed(t *testing.T) {
	m := &counter{}
	s := &counter{}
	step := model.Step[string, int]{
		ModelApply: m.apply,
		SubjectApply: func(op string) (int, error) {
			if op == "inc" {
				return 0, errors.New("crashed")
			}
			return s.apply(op)
		},
		Equal: func(want, got int) error {
			if want != got {
				return fmt.Errorf("want %d got %d", want, got)
			}
			return nil
		},
	}
	if err := step.CheckAllowFailed("inc", false); err == nil {
		t.Fatal("want subject error when failed=false")
	}
	if err := step.CheckAllowFailed("inc", true); err != nil {
		t.Fatalf("failed=true: %v", err)
	}
}

func TestDirtyRebootHelpers(t *testing.T) {
	store := map[string]int{"a": 1, "b": 2}
	answered := []model.Write[string, int]{
		{Key: "a", Value: 1},
		{Key: "b", Value: 2},
	}
	if err := model.CheckPersistence(answered, func(k string, v int) bool {
		return store[k] == v
	}); err != nil {
		t.Fatalf("persistence: %v", err)
	}
	delete(store, "b")
	if err := model.CheckPersistence(answered, func(k string, v int) bool {
		return store[k] == v
	}); err == nil {
		t.Fatal("want persistence error")
	}

	modelClean := func() (int, error) { return 3, nil }
	subjectOK := func() (int, error) { return 3, nil }
	subjectBad := func() (int, error) { return 1, nil }
	eq := func(want, got int) error {
		if want != got {
			return fmt.Errorf("want %d got %d", want, got)
		}
		return nil
	}
	if err := model.CheckForwardProgress(modelClean, subjectOK, eq); err != nil {
		t.Fatalf("forward: %v", err)
	}
	if err := model.CheckForwardProgress(modelClean, subjectBad, eq); err == nil {
		t.Fatal("want forward progress error")
	}
}
