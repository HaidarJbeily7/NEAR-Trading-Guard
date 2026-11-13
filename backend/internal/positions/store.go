// Package positions implements a deterministic, in-memory lending simulation.
// Its USD float calculations must never be used to authorize real transactions.
package positions

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/HaidarJbeily7/NEAR-Trading-Guard/backend/internal/risk"
)

var ErrConflict = errors.New("positions changed; refresh before retrying")

type Point struct {
	Label  string  `json:"label"`
	Health float64 `json:"health"`
}
type Position struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Symbol       string  `json:"symbol"`
	Collateral   float64 `json:"collateral"`
	Debt         float64 `json:"debt"`
	Threshold    float64 `json:"threshold"`
	Trigger      float64 `json:"trigger"`
	Target       float64 `json:"target"`
	Reserve      float64 `json:"reserve"`
	AutoProtect  bool    `json:"auto_protect"`
	Paused       bool    `json:"paused"`
	Liquidated   bool    `json:"liquidated"`
	Health       float64 `json:"health"`
	Status       string  `json:"status"`
	RescueAmount float64 `json:"rescue_amount"`
	RescuedUSD   float64 `json:"rescued_usd"`
	History      []Point `json:"history"`
}
type Event struct {
	ID         uint64    `json:"id"`
	At         time.Time `json:"at"`
	PositionID string    `json:"position_id"`
	Title      string    `json:"title"`
	Detail     string    `json:"detail"`
	Kind       string    `json:"kind"`
}
type Snapshot struct {
	Mode      string     `json:"mode"`
	Revision  uint64     `json:"revision"`
	Positions []Position `json:"positions"`
	Events    []Event    `json:"events"`
}
type Action struct {
	ID          string  `json:"id"`
	Action      string  `json:"action"`
	Revision    uint64  `json:"revision"`
	DropPercent float64 `json:"drop_percent"`
}
type Create struct {
	Name       string  `json:"name"`
	Symbol     string  `json:"symbol"`
	Collateral float64 `json:"collateral"`
	Debt       float64 `json:"debt"`
	Threshold  float64 `json:"threshold"`
	Reserve    float64 `json:"reserve"`
}
type Store struct {
	mu        sync.Mutex
	revision  uint64
	nextID    uint64
	nextEvent uint64
	positions []Position
	events    []Event
}

func New() *Store { s := &Store{}; s.reset(); return s }
func (s *Store) reset() {
	s.positions = []Position{
		{ID: "near-1", Name: "NEAR lending", Symbol: "NEAR", Collateral: 2400, Debt: 1200, Threshold: .8, Trigger: 1.3, Target: 1.6, Reserve: 800, AutoProtect: true},
		{ID: "eth-2", Name: "ETH lending", Symbol: "ETH", Collateral: 3200, Debt: 1600, Threshold: .8, Trigger: 1.3, Target: 1.6, Reserve: 350, AutoProtect: true},
		{ID: "btc-3", Name: "BTC lending", Symbol: "BTC", Collateral: 6000, Debt: 2500, Threshold: .8, Trigger: 1.3, Target: 1.6, Reserve: 500, AutoProtect: false},
	}
	s.events = nil
	for i := range s.positions {
		refresh(&s.positions[i])
		s.point(&s.positions[i], "Opened")
	}
	s.event("", "Simulation ready", "Three sample positions loaded. No wallet funds are connected.", "info")
}
func refresh(p *Position) {
	plan, _ := risk.Evaluate(risk.Position{CollateralUSD: p.Collateral, DebtUSD: p.Debt, LiquidationThreshold: p.Threshold}, risk.Policy{Trigger: p.Trigger, Target: p.Target, BudgetUSD: p.Reserve})
	p.Health = plan.HealthFactor
	p.Status = plan.Status
	p.RescueAmount = plan.TopUpUSD
	if p.Liquidated {
		p.Status = "liquidated"
	} else if p.Paused {
		p.Status = "paused"
	}
}
func (s *Store) event(id, title, detail, kind string) {
	s.nextEvent++
	s.events = append([]Event{{s.nextEvent, time.Now().UTC(), id, title, detail, kind}}, s.events...)
	if len(s.events) > 100 {
		s.events = s.events[:100]
	}
}
func (s *Store) point(p *Position, label string) {
	p.History = append(p.History, Point{label, p.Health})
	if len(p.History) > 24 {
		p.History = p.History[len(p.History)-24:]
	}
}
func (s *Store) snapshot() Snapshot {
	result := Snapshot{Mode: "simulation", Revision: s.revision, Positions: append([]Position{}, s.positions...), Events: append([]Event{}, s.events...)}
	for i := range result.Positions {
		result.Positions[i].History = append([]Point{}, result.Positions[i].History...)
	}
	return result
}
func (s *Store) Snapshot() Snapshot { s.mu.Lock(); defer s.mu.Unlock(); return s.snapshot() }
func (s *Store) Apply(a Action) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a.Revision != s.revision {
		return Snapshot{}, ErrConflict
	}
	if a.Action == "reset" {
		s.reset()
		s.revision++
		return s.snapshot(), nil
	}
	var p *Position
	for i := range s.positions {
		if s.positions[i].ID == a.ID {
			p = &s.positions[i]
			break
		}
	}
	if p == nil {
		return Snapshot{}, errors.New("position not found")
	}
	if p.Liquidated {
		return Snapshot{}, errors.New("position was liquidated; reset the simulation")
	}
	switch a.Action {
	case "crash":
		if math.IsNaN(a.DropPercent) || math.IsInf(a.DropPercent, 0) || a.DropPercent <= 0 || a.DropPercent > 50 {
			return Snapshot{}, errors.New("price drop must be between 0 and 50 percent")
		}
		p.Collateral *= 1 - a.DropPercent/100
		refresh(p)
		s.point(p, "Price drop")
		s.event(p.ID, "Price shock detected", fmt.Sprintf("Simulated collateral value fell %.0f%%. Health factor: %.2f.", a.DropPercent, p.Health), "warning")
		if p.AutoProtect && !p.Paused {
			if p.Status == "rescue_planned" {
				s.rescue(p)
			} else if p.Status == "insufficient_reserve" {
				s.event(p.ID, "Rescue blocked", "The reserve cannot restore the target health factor. No funds were deducted.", "danger")
			} else if p.Status == "liquidatable" {
				s.event(p.ID, "Liquidation threshold crossed", "Health is at or below 1.00. A rescue is not assumed to beat liquidation.", "danger")
			}
		}
	case "rescue":
		if p.Paused {
			return Snapshot{}, errors.New("resume protection before rescuing")
		}
		if p.Status != "rescue_planned" {
			return Snapshot{}, errors.New("no eligible rescue: position is healthy, liquidatable or underfunded")
		}
		s.rescue(p)
	case "pause":
		p.Paused = !p.Paused
		refresh(p)
		title := "Protection resumed"
		if p.Paused {
			title = "Protection paused"
		}
		s.event(p.ID, title, "Changed for this simulated position.", "info")
	case "auto":
		p.AutoProtect = !p.AutoProtect
		title := "Automatic rescue enabled"
		if !p.AutoProtect {
			title = "Automatic rescue disabled"
		}
		s.event(p.ID, title, "Automatic rescue is evaluated on the next simulated price change.", "info")
	case "liquidate":
		if p.Health > 1 {
			return Snapshot{}, errors.New("position is above the liquidation threshold")
		}
		p.Liquidated = true
		refresh(p)
		s.point(p, "Liquidated")
		s.event(p.ID, "Position liquidated (simulation)", "Position closed in the demo. Displayed collateral and debt are the final snapshot; protocol penalties are not modeled.", "danger")
	default:
		return Snapshot{}, errors.New("unknown action")
	}
	s.revision++
	return s.snapshot(), nil
}
func (s *Store) rescue(p *Position) {
	amount := p.RescueAmount
	p.Reserve -= amount
	p.Collateral += amount
	p.RescuedUSD += amount
	refresh(p)
	s.point(p, "Rescued")
	s.event(p.ID, "Position rescued", fmt.Sprintf("$%.2f moved from simulated reserve to collateral. Health restored to %.2f. No on-chain transaction.", amount, p.Health), "success")
}
func (s *Store) Add(c Create) (Snapshot, error) {
	if len(c.Name) == 0 || len(c.Name) > 60 || len(c.Symbol) == 0 || len(c.Symbol) > 10 {
		return Snapshot{}, errors.New("name (1–60 characters) and symbol (1–10 characters) are required")
	}
	if c.Collateral > 1e12 || c.Debt > 1e12 || c.Reserve > 1e12 {
		return Snapshot{}, errors.New("demo values must not exceed one trillion USD")
	}
	if _, err := risk.Evaluate(risk.Position{CollateralUSD: c.Collateral, DebtUSD: c.Debt, LiquidationThreshold: c.Threshold}, risk.Policy{Trigger: 1.3, Target: 1.6, BudgetUSD: c.Reserve}); err != nil {
		return Snapshot{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.positions) >= 20 {
		return Snapshot{}, errors.New("maximum 20 demo positions")
	}
	s.nextID++
	p := Position{ID: fmt.Sprintf("custom-%d", s.nextID), Name: c.Name, Symbol: c.Symbol, Collateral: c.Collateral, Debt: c.Debt, Threshold: c.Threshold, Trigger: 1.3, Target: 1.6, Reserve: c.Reserve, AutoProtect: true}
	refresh(&p)
	s.point(&p, "Imported")
	s.positions = append(s.positions, p)
	s.event(p.ID, "Position added", "User-provided values added to the simulation; no live lending data was imported.", "info")
	s.revision++
	return s.snapshot(), nil
}
