# Architecture and next steps

Implemented: Go CLI/API -> mainnet Intents token discovery and validated preview. Local CLI -> prepared quote -> manual wallet deposit -> status lookup.

Separate simulated path: embedded dashboard -> Go position store -> risk planner -> budget-limited simulated rescue. History and activity live in memory. Future on-chain keeper and position adapters remain unconnected.
Future execution: owner-approved policy -> funded swap -> confirmed settlement -> lending adapter -> verified health factor.

The Rust NEAR contract currently stores owner-controlled health thresholds and a pause flag. Its state is not connected to the Go demo. Budget enforcement, reserves and execution permissions remain unimplemented.

## Demo semantics

`GET /api/demo` evaluates $1,400 collateral, $1,000 debt and an 80% liquidation threshold. Health factor is 1.12. A $475 collateral top-up would restore 1.50, within the $500 demo budget. This is a plan, not an executed transaction. Values are USD floats exclusively for simulation; use integer token amounts and protocol-specific rounding for production.

## Implementation order

1. Add a mock lending contract and oracle with deterministic crash scenarios.
2. Wire a real read-only position source, with freshness checks.
3. Verify the implemented Intents reserve swap with a funded user wallet and confirmed settlement.
4. Add keeper scheduling, duplicate prevention and persisted operation states.
5. Implement on-chain budgets, keeper permissions and an allowlisted lending adapter.
6. Extend the implemented dashboard with authenticated live position sources and durable per-user storage.
7. Verify rescue transactions and failure/refund behavior before enabling execution.

Cross-chain settlement is asynchronous. Funds arriving in a wallet do not automatically top up a lending position. Keep emergency reserves local and treat remote replenishment as a separate operation.

## References

- https://nearhackathon.com/HACKATHON.md
- https://docs.near.org/smart-contracts/quickstart
- https://docs.near.org/api/rpc/introduction
- https://docs.near-intents.org/

Intents is mainnet-only. This scaffold never signs or sends a blockchain transaction.
