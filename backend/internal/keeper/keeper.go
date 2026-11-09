// Package keeper defines boundaries for a future off-chain monitoring worker.
// No polling, signing or execution is implemented yet.
package keeper

import (
	"context"
	"github.com/HaidarJbeily7/NEAR-Trading-Guard/backend/internal/risk"
)

type PositionSource interface {
	ReadPosition(context.Context, string) (risk.Position, error)
}

// Executor implementations must enforce freshness, idempotency, budget and
// protocol permissions, then verify settlement before reporting success.
type Executor interface {
	Execute(context.Context, string, risk.Plan) (string, error)
}
