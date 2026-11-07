// Package risk contains demo-only USD calculations. Production execution must use
// integer token units, protocol-specific math, and fresh authenticated prices.
package risk

import (
	"errors"
	"math"
)

type Position struct{ CollateralUSD, DebtUSD, LiquidationThreshold float64 }
type Policy struct{ Trigger, Target, BudgetUSD float64 }
type Plan struct {
	Mode                  string  `json:"mode"`
	HealthFactor          float64 `json:"health_factor"`
	TopUpUSD              float64 `json:"top_up_usd"`
	ProjectedHealthFactor float64 `json:"projected_health_factor"`
	Status                string  `json:"status"`
}

func Evaluate(p Position, policy Policy) (Plan, error) {
	for _, v := range []float64{p.CollateralUSD, p.DebtUSD, p.LiquidationThreshold, policy.Trigger, policy.Target, policy.BudgetUSD} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return Plan{}, errors.New("values must be finite")
		}
	}
	if p.CollateralUSD < 0 || p.DebtUSD <= 0 || p.LiquidationThreshold <= 0 || p.LiquidationThreshold > 1 || policy.Trigger <= 1 || policy.Target < policy.Trigger || policy.BudgetUSD < 0 {
		return Plan{}, errors.New("invalid position or protection policy")
	}
	hf := p.CollateralUSD * p.LiquidationThreshold / p.DebtUSD
	plan := Plan{Mode: "simulation", HealthFactor: hf, ProjectedHealthFactor: hf, Status: "healthy"}
	if hf >= policy.Trigger {
		return plan, nil
	}
	if hf <= 1 {
		plan.Status = "liquidatable"
		return plan, nil
	}
	required := policy.Target*p.DebtUSD/p.LiquidationThreshold - p.CollateralUSD
	if required > policy.BudgetUSD {
		plan.Status = "insufficient_reserve"
		return plan, nil
	}
	plan.Status = "rescue_planned"
	plan.TopUpUSD = required
	plan.ProjectedHealthFactor = (p.CollateralUSD + required) * p.LiquidationThreshold / p.DebtUSD
	return plan, nil
}
