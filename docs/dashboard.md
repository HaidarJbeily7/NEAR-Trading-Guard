# Position dashboard

Run `make run` from the repository root and open http://localhost:8080. The Go binary embeds the HTML, CSS and JavaScript: no frontend package installation, build step or separate server is required. Restart the server after changing embedded frontend files.

The page opens with no positions. It does not invent balances or provide a simulated rescue. `/api/positions` reports that the live lending adapter is unconfigured. The Intents API is contacted only after **Enable live preview** is clicked in the separate Intents card.

## Live position boundary

The positions area remains empty until a real lending adapter is configured. The adapter must read a wallet/account's current collateral, debt, oracle prices, health factor, and reserve policy. Rescue actions must require explicit authorization and be verified with on-chain receipts. There is no local fallback position or crash button.

## Intents panel

Token discovery uses the live mainnet API. Choose two NEAR-chain assets, enter a decimal input amount and a recipient/refund account, then request a preview. The browser converts amounts into exact integer token units without floating-point conversion. Editing inputs invalidates the displayed quote; an expired preview must be refreshed.

The preview cannot send funds. Use the [mainnet runbook](mainnet-swap.md) and local CLI for deposit preparation. To track a prepared swap, paste its deposit address (and memo when applicable) into **Track an existing mainnet swap**. Status refresh is manual. Provider success is not presented as a lending rescue or independent wallet-balance verification.

## API and state

- `GET /api/positions`: live adapter boundary. It currently returns `connected:false` and an empty list.
- Existing Intents preview and status routes remain unchanged.

This localhost app is not a multi-user hosted service. Add authentication, per-user durable storage, a real lending adapter and a wallet authorization flow before exposing it publicly. No lending positions or rescue transactions are connected.

## Verification

`make test vet` checks the live boundary, embedded asset serving, and Intents client validation. Browser QA checks the empty live-position state, explicit Intents opt-in, responsive layout, and quote/status rendering with fixtures. No transaction is executed by the dashboard.
