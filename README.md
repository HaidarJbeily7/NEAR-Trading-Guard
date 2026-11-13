# NEAR Trading Guard

A Go backend and Rust NEAR contract scaffold for budget-limited liquidation-risk protection.

## Dashboard

Run `make run` and open **http://localhost:8080**. The dashboard consumes only the local Go position API for its main demo, shows health and reserve balances, and simulates price drops and budget-limited rescues. Live Intents previews are hidden behind an explicit opt-in and are never needed for the rescue demo.

Try the default 25% NEAR price drop to see an automatic rescue, then the ETH position to see an insufficient-reserve failure. Add custom simulated positions, pause protection, and inspect activity and health history. See the [dashboard guide](docs/dashboard.md).

## Mainnet reserve swaps

The Go integration discovers live NEAR Intents assets, previews exact-input swaps, prepares deposit addresses through a local CLI, and tracks settlement. Funding happens in the user's wallet. This first release supports NEAR-to-NEAR conversions only.

**No live swap has been funded or settlement-verified by this project yet.** The lending demo remains simulated. The Rust contract stores owner-controlled policy only; it does not hold reserves or execute rescues. This is not production-ready liquidation protection.

Follow [the mainnet swap runbook](docs/mainnet-swap.md) to configure your own route, preview it, and prepare a swap. The application never requests a private key or signs a transaction.

## Development

Requires Go 1.21+ and, on macOS, Xcode Command Line Tools for the system linker.

```sh
make test vet
make run
```

Open http://localhost:8080 for the dashboard. Health and simulation JSON remain available at `/healthz` and `/api/demo`. The Intents routes are documented in the runbook. Optional `INTENTS_JWT` must be exported in the shell; no dotenv dependency is used.

```sh
make intents ARGS='tokens'
```

Rust 1.85.1 and the WebAssembly target are pinned in `contracts/guard`. Build the policy scaffold with `make contract`.

## Layout

- `backend/cmd/api`: local preview/status API and simulated lending calculation
- `backend/cmd/intents`: mainnet preview, preparation and status CLI
- `backend/internal/intents`: typed requests, route checks, amount limits and HTTP transport
- `backend/internal/web`: embedded responsive dashboard, with no frontend dependencies
- `backend/internal/positions`: in-memory position simulation and activity
- `backend/internal/risk`: demo-only USD calculations
- `backend/internal/keeper`: future monitoring interfaces
- `contracts/guard`: owner-controlled NEAR policy skeleton
- `examples/quote.json`: placeholder request template
- `docs`: architecture and mainnet runbook

Local requests and quote records belong in the gitignored `local/` directory. See [architecture](docs/architecture.md) for the remaining lending integration work.
