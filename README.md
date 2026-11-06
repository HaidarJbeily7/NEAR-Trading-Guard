# NEAR Trading Guard

A starter for budget-limited liquidation-risk protection using a Go backend and a Rust NEAR contract.

## Status

Scaffold only: the API evaluates a fixed simulated lending position. No live prices, wallet signing, swaps, collateral deposits, or background monitoring are connected. This does not guarantee protection from liquidation.

## Development

Requires Go 1.21+. Run `make run`, then visit http://localhost:8080/healthz or http://localhost:8080/api/demo.
Run `make test` and `make vet` for backend checks.

The Rust policy contract lives in `contracts/guard`. Build using Rust 1.85.1 (pinned in the contract directory), the wasm32-unknown-unknown target and `make contract`. It stores policy only; it neither holds reserves nor executes rescues.

## Layout

- `backend/cmd/api`: local HTTP API
- `backend/internal/risk`: simulated health factor and collateral top-up planning
- `backend/internal/keeper`: integration boundaries for future monitoring
- `backend/internal/intents`: dry-quote HTTP adapter
- `contracts/guard`: owner-controlled NEAR policy skeleton
- `docs`: architecture and implementation roadmap
