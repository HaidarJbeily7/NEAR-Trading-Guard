# Position dashboard

Run `make run` from the repository root and open http://localhost:8080. The Go binary embeds the HTML, CSS and JavaScript: no frontend package installation, build step or separate server is required. Restart the server after changing embedded frontend files.

## Reproduce the rescue demo

1. Open Overview. The NEAR sample starts at $2,400 collateral, $1,200 debt, an $800 reserve and health factor 1.60.
2. Leave automatic rescue enabled and the simulated drop at 25%.
3. Click **Simulate price drop**. Collateral value falls to $1,800 and health reaches 1.20.
4. The demo guard moves $600 from reserve into collateral, restoring health to 1.60. The reserve falls to $200.
5. Inspect the chart's price-drop and rescue points and the Guard activity entries.
6. Select ETH and repeat the 25% drop. Its $350 reserve cannot fund the required top-up, so the guard blocks the rescue without spending.

Pause the guard to observe an unprotected drop, or disable automatic rescue and use **Rescue now** when eligible. A 50% drop can cross the liquidation boundary; the optional simulation-only liquidation action closes the position without modeling a real protocol's penalties. Reset restores the three initial positions and clears their event history.

**Add position** accepts collateral/debt values, a liquidation threshold and a reserve. These values create a simulated position, not an imported blockchain account. Trigger and target health are currently 1.30 and 1.60.

## Intents panel

Token discovery uses the live mainnet API. Choose two NEAR-chain assets, enter a decimal input amount and a recipient/refund account, then request a preview. The browser converts amounts into exact integer token units without floating-point conversion. Editing inputs invalidates the displayed quote; an expired preview must be refreshed.

The preview cannot send funds. Use the [mainnet runbook](mainnet-swap.md) and local CLI for deposit preparation. To track a prepared swap, paste its deposit address (and memo when applicable) into **Track an existing mainnet swap**. Status refresh is manual. Provider success is not presented as a lending rescue or independent wallet-balance verification.

## API and state

- `GET /api/positions`: simulated positions, revision and activity.
- `POST /api/positions`: create a position from user-entered simulation values.
- `POST /api/positions/action`: `crash`, `rescue`, `pause`, `auto`, `liquidate` or `reset`; include the snapshot's revision.
- Existing Intents preview and status routes remain unchanged.

The store is in memory, shared across tabs connected to this local server and reset on process restart. It retains at most 20 positions, 24 chart points per position and 100 events. Position mutations are serialized; stale revisions return HTTP 409 instead of spending a demo reserve twice. The browser refreshes positions every five seconds while visible.

This localhost app is not a multi-user hosted service. Add authentication, per-user durable storage and a real lending adapter before exposing it publicly. No actual lending positions, keeper transactions or liquidation executions are connected.

## Verification

`make test vet` checks backend behavior, including reserve conservation, stale-action rejection, paused protection, invalid inputs, simulated liquidation, reset and embedded asset serving.

Browser QA covered desktop and mobile, adding a position, HTML escaping, automatic rescue, insufficient reserves, resetting, liquidation failure scenarios, live token discovery, fixture-driven quote/status rendering, stale-preview removal and horizontal overflow. Quote and status UI fixtures did not execute any transactions.
