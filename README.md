# NEAR Trading Guard

A Go backend and Rust NEAR contract scaffold for budget-limited liquidation-risk protection.

## Current milestone: mainnet reserve swaps

The Go integration discovers live NEAR Intents assets, previews exact-input swaps, prepares deposit addresses through a local CLI, and tracks settlement. Funding happens in the user's wallet. This first release supports NEAR-to-NEAR conversions only.

**No live swap has been funded or settlement-verified by this project yet.** The lending demo remains simulated. The Rust contract stores owner-controlled policy only; it does not hold reserves or execute rescues. This is not production-ready liquidation protection.

Follow [the mainnet swap runbook](docs/mainnet-swap.md) to configure your own route, preview it, and prepare a swap. The application never requests a private key or signs a transaction.

## Development

Requires Go 1.21+ and, on macOS, Xcode Command Line Tools for the system linker.

```sh
make test vet
make run
```

Open http://localhost:8080/healthz or http://localhost:8080/api/demo. The Intents routes are documented in the runbook. Optional `INTENTS_JWT` must be exported in the shell; no dotenv dependency is used.

```sh
make intents ARGS='tokens'
```

Rust 1.85.1 and the WebAssembly target are pinned in `contracts/guard`. Build the policy scaffold with `make contract`.

## Layout

- `backend/cmd/api`: local preview/status API and simulated lending calculation
- `backend/cmd/intents`: mainnet preview, preparation and status CLI
- `backend/internal/intents`: typed requests, route checks, amount limits and HTTP transport
- `backend/internal/risk`: demo-only USD calculations
- `backend/internal/keeper`: future monitoring interfaces
- `contracts/guard`: owner-controlled NEAR policy skeleton
- `examples/quote.json`: placeholder request template
- `docs`: architecture and mainnet runbook

Local requests and quote records belong in the gitignored `local/` directory. See [architecture](docs/architecture.md) for the remaining lending integration work.
