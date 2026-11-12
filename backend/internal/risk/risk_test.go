package risk

import (
	"math"
	"testing"
)

func TestEvaluate(t *testing.T) {
	policy := Policy{Trigger: 1.2, Target: 1.5, BudgetUSD: 500}
	for _, tc := range []struct {
		name               string
		collateral, budget float64
		status             string
		topUp              float64
	}{
		{"healthy", 2000, 500, "healthy", 0},
		{"rescue", 1400, 500, "rescue_planned", 475},
		{"insufficient", 1400, 100, "insufficient_reserve", 0},
		{"already liquidatable", 1200, 500, "liquidatable", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := policy
			p.BudgetUSD = tc.budget
			plan, err := Evaluate(Position{tc.collateral, 1000, 0.8}, p)
			if err != nil || plan.Status != tc.status || math.Abs(plan.TopUpUSD-tc.topUp) > 0.001 {
				t.Fatalf("plan=%+v err=%v", plan, err)
			}
			if tc.topUp > 0 && math.Abs(plan.ProjectedHealthFactor-p.Target) > 0.001 {
				t.Fatal("target not restored")
			}
		})
	}
}

func TestRejectInvalidInputs(t *testing.T) {
	for _, debt := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if _, err := Evaluate(Position{1400, debt, 0.8}, Policy{1.2, 1.5, 500}); err == nil {
			t.Fatalf("accepted debt %v", debt)
		}
	}
}
