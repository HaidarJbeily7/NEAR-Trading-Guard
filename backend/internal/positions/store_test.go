package positions

import (
	"errors"
	"math"
	"testing"
)

func TestAutomaticRescueAndBudget(t *testing.T) {
	s := New()
	before := s.Snapshot()
	after, err := s.Apply(Action{ID: "near-1", Action: "crash", Revision: before.Revision, DropPercent: 25})
	if err != nil {
		t.Fatal(err)
	}
	p := after.Positions[0]
	if math.Abs(p.Health-1.6) > .0001 || math.Abs(p.Reserve-200) > .001 || p.RescuedUSD != 600 {
		t.Fatalf("unexpected rescue: %+v", p)
	}
	if len(p.History) != 3 || p.History[1].Label != "Price drop" || p.History[2].Label != "Rescued" {
		t.Fatal("missing before/after history")
	}
	// Replay must not spend the reserve twice.
	if _, err = s.Apply(Action{ID: "near-1", Action: "crash", Revision: before.Revision, DropPercent: 25}); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected replay conflict: %v", err)
	}
	if s.Snapshot().Positions[0].Reserve != 200 {
		t.Fatal("duplicate spending")
	}
}
func TestInsufficientReserveDoesNotSpend(t *testing.T) {
	s := New()
	result, err := s.Apply(Action{ID: "eth-2", Action: "crash", DropPercent: 25})
	if err != nil {
		t.Fatal(err)
	}
	p := result.Positions[1]
	if p.Status != "insufficient_reserve" || p.Reserve != 350 || p.RescuedUSD != 0 {
		t.Fatalf("unexpected state %+v", p)
	}
}
func TestPausedAndLiquidation(t *testing.T) {
	s := New()
	state, err := s.Apply(Action{ID: "near-1", Action: "pause"})
	if err != nil {
		t.Fatal(err)
	}
	state, err = s.Apply(Action{ID: "near-1", Action: "crash", Revision: state.Revision, DropPercent: 50})
	if err != nil {
		t.Fatal(err)
	}
	p := state.Positions[0]
	if p.Reserve != 800 || p.Health > .81 || p.Status != "paused" {
		t.Fatal("pause failed")
	}
	if _, err = s.Apply(Action{ID: p.ID, Action: "rescue", Revision: state.Revision}); err == nil {
		t.Fatal("rescued while paused")
	}
	state, err = s.Apply(Action{ID: p.ID, Action: "liquidate", Revision: state.Revision})
	if err != nil {
		t.Fatal(err)
	}
	if !state.Positions[0].Liquidated || state.Positions[0].Status != "liquidated" {
		t.Fatal("not liquidated")
	}
	if _, err = s.Apply(Action{ID: p.ID, Action: "crash", Revision: state.Revision, DropPercent: 10}); err == nil {
		t.Fatal("mutated closed position")
	}
}
func TestInvalidChangesAndSnapshotIsolation(t *testing.T) {
	s := New()
	if _, err := s.Apply(Action{ID: "near-1", Action: "crash", DropPercent: 90}); err == nil {
		t.Fatal("accepted invalid drop")
	}
	if _, err := s.Apply(Action{ID: "near-1", Action: "liquidate"}); err == nil {
		t.Fatal("liquidated healthy position")
	}
	if _, err := s.Add(Create{Name: "bad", Symbol: "X", Collateral: 100, Debt: 0, Threshold: .8}); err == nil {
		t.Fatal("accepted invalid position")
	}
	snapshot := s.Snapshot()
	snapshot.Positions[0].History[0].Label = "mutated"
	if s.Snapshot().Positions[0].History[0].Label == "mutated" {
		t.Fatal("snapshot aliases store")
	}
}
func TestAddAndReset(t *testing.T) {
	s := New()
	snapshot, err := s.Add(Create{Name: "Custom", Symbol: "NEAR", Collateral: 2400, Debt: 1200, Threshold: .8, Reserve: 800})
	if err != nil || len(snapshot.Positions) != 4 {
		t.Fatalf("%+v %v", snapshot, err)
	}
	snapshot, err = s.Apply(Action{Action: "reset", Revision: snapshot.Revision})
	if err != nil || len(snapshot.Positions) != 3 || snapshot.Revision != 2 {
		t.Fatalf("%+v %v", snapshot, err)
	}
}
